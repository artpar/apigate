// Package bootstrap wires all dependencies and starts the application.
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artpar/apigate/adapters/clock"
	"github.com/artpar/apigate/adapters/hasher"
	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/adapters/idgen"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/adapters/metrics"
	"github.com/artpar/apigate/adapters/remote"
	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/ports"
	"github.com/artpar/apigate/web"
	"github.com/rs/zerolog"
)

// App represents the running application.
type App struct {
	Config       *config.Config
	ConfigHolder *config.Holder
	Logger       zerolog.Logger
	DB           *sqlite.DB
	HTTPServer   *http.Server
	Metrics      *metrics.Collector

	// Services (for hot reload)
	proxyService     *app.ProxyService
	routeService     *app.RouteService
	transformService *app.TransformService

	// Adapters (for cleanup)
	usageRecorder ports.UsageRecorder
	upstream      *apihttp.UpstreamClient
}

// New creates and initializes the application from a config file path.
func New(cfg *config.Config) (*App, error) {
	// Setup logger
	logger := setupLogger(cfg.Logging)

	logger.Info().
		Str("upstream", cfg.Upstream.URL).
		Str("auth_mode", cfg.Auth.Mode).
		Str("usage_mode", cfg.Usage.Mode).
		Str("billing_mode", cfg.Billing.Mode).
		Msg("initializing apigate")

	app := &App{
		Config: cfg,
		Logger: logger,
	}

	// Initialize database
	if err := app.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// Initialize HTTP server
	if err := app.initHTTPServer(); err != nil {
		return nil, fmt.Errorf("init http server: %w", err)
	}

	return app, nil
}

// NewWithHotReload creates the application with hot reload support.
func NewWithHotReload(configPath string) (*App, error) {
	// Create initial logger for bootstrap
	initLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create config holder
	holder, err := config.NewHolder(configPath, initLogger)
	if err != nil {
		return nil, fmt.Errorf("create config holder: %w", err)
	}

	cfg := holder.Get()

	// Setup proper logger
	logger := setupLogger(cfg.Logging)
	holder = replaceHolderLogger(holder, configPath, logger)

	logger.Info().
		Str("upstream", cfg.Upstream.URL).
		Str("auth_mode", cfg.Auth.Mode).
		Str("usage_mode", cfg.Usage.Mode).
		Str("billing_mode", cfg.Billing.Mode).
		Bool("hot_reload", true).
		Msg("initializing apigate")

	a := &App{
		Config:       cfg,
		ConfigHolder: holder,
		Logger:       logger,
	}

	// Initialize database
	if err := a.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// Initialize HTTP server
	if err := a.initHTTPServer(); err != nil {
		return nil, fmt.Errorf("init http server: %w", err)
	}

	// Wire up hot reload callback
	holder.OnChange(a.onConfigChange)

	// Start watching for changes
	if err := holder.WatchFile(); err != nil {
		logger.Warn().Err(err).Msg("failed to watch config file, hot reload disabled")
	}

	// Start listening for SIGHUP
	holder.WatchSignals()

	return a, nil
}

// replaceHolderLogger creates a new holder with the proper logger
func replaceHolderLogger(old *config.Holder, path string, logger zerolog.Logger) *config.Holder {
	old.Stop()
	holder, _ := config.NewHolder(path, logger)
	return holder
}

// onConfigChange is called when configuration changes.
func (a *App) onConfigChange(cfg *config.Config) {
	a.Logger.Info().Msg("applying configuration changes")

	// Update proxy service with new dynamic config
	if a.proxyService != nil {
		a.proxyService.UpdateConfig(
			convertPlans(cfg.Plans),
			convertEndpoints(cfg.Endpoints),
			cfg.RateLimit.BurstTokens,
			cfg.RateLimit.WindowSecs,
		)
	}

	// Update log level if changed
	if cfg.Logging.Level != a.Config.Logging.Level {
		level, err := zerolog.ParseLevel(cfg.Logging.Level)
		if err == nil {
			zerolog.SetGlobalLevel(level)
			a.Logger.Info().Str("level", cfg.Logging.Level).Msg("log level updated")
		}
	}

	// Record config reload in metrics
	if a.Metrics != nil {
		a.Metrics.ConfigReloads.Inc()
		a.Metrics.ConfigLastReload.SetToCurrentTime()
	}

	// Store new config
	a.Config = cfg
}

// Reload manually triggers a configuration reload.
func (a *App) Reload() error {
	if a.ConfigHolder == nil {
		return fmt.Errorf("hot reload not enabled")
	}
	return a.ConfigHolder.Reload()
}

func (a *App) initDatabase() error {
	if a.Config.Database.Driver != "sqlite" {
		return fmt.Errorf("unsupported database driver: %s", a.Config.Database.Driver)
	}

	db, err := sqlite.Open(a.Config.Database.DSN)
	if err != nil {
		return err
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		return fmt.Errorf("migrate: %w", err)
	}

	a.DB = db
	a.Logger.Info().Str("dsn", a.Config.Database.DSN).Msg("database initialized")
	return nil
}

func (a *App) initHTTPServer() error {
	// Build dependencies
	deps, err := a.buildDependencies()
	if err != nil {
		return err
	}

	// Build proxy config
	proxyCfg := app.ProxyConfig{
		KeyPrefix:  a.Config.Auth.KeyPrefix,
		Plans:      convertPlans(a.Config.Plans),
		Endpoints:  convertEndpoints(a.Config.Endpoints),
		RateBurst:  a.Config.RateLimit.BurstTokens,
		RateWindow: a.Config.RateLimit.WindowSecs,
	}

	// Create proxy service
	a.proxyService = app.NewProxyService(deps, proxyCfg)

	// Create and wire route service for dynamic routing
	routeStore := sqlite.NewRouteStore(a.DB)
	upstreamStore := sqlite.NewUpstreamStore(a.DB)
	a.routeService = app.NewRouteService(
		routeStore,
		upstreamStore,
		deps.Clock,
		a.Logger,
		app.RouteServiceConfig{
			RefreshInterval: 30 * time.Second,
		},
	)
	a.proxyService.SetRouteService(a.routeService)

	// Start route service to load initial routes and begin refresh loop
	if err := a.routeService.Start(context.Background()); err != nil {
		a.Logger.Warn().Err(err).Msg("failed to start route service, continuing with empty routes")
	}

	// Create and wire transform service for request/response transformations
	a.transformService = app.NewTransformService()
	a.proxyService.SetTransformService(a.transformService)

	a.Logger.Info().Msg("route and transform services initialized")

	// Initialize metrics if enabled
	if a.Config.Metrics.Enabled {
		a.Metrics = metrics.New()
		a.Logger.Info().Msg("prometheus metrics enabled")
	}

	// Create HTTP handlers
	var proxyHandler *apihttp.ProxyHandler
	if a.Metrics != nil {
		proxyHandler = apihttp.NewProxyHandlerWithMetrics(a.proxyService, a.Logger, a.Metrics)
	} else {
		proxyHandler = apihttp.NewProxyHandler(a.proxyService, a.Logger)
	}
	// Enable streaming support by setting the streaming upstream
	proxyHandler.SetStreamingUpstream(a.upstream)
	healthHandler := apihttp.NewHealthHandler(a.upstream)

	// Create shared stores for admin and web handlers
	usageStore := sqlite.NewUsageStore(a.DB)
	bcryptHasher := hasher.NewBcrypt(0) // 0 = default cost

	// Create admin handler with route/upstream stores
	adminHandler := admin.NewHandler(admin.Deps{
		Users:     deps.Users,
		Keys:      deps.Keys,
		Usage:     usageStore,
		Routes:    routeStore,
		Upstreams: upstreamStore,
		Config:    a.Config,
		Logger:    a.Logger,
		Hasher:    bcryptHasher,
	})

	// Create web UI handler
	webHandler, err := web.NewHandler(web.Deps{
		Users:         deps.Users,
		Keys:          deps.Keys,
		Usage:         usageStore,
		Routes:        routeStore,
		Upstreams:     upstreamStore,
		Config:        a.Config,
		Logger:        a.Logger,
		Hasher:        bcryptHasher,
		JWTSecret:     a.Config.Auth.JWTSecret,
		ExprValidator: a.transformService,
		RouteTester:   a.routeService,
		IsSetup: func() bool {
			// Check if admin user exists
			users, err := deps.Users.List(context.Background(), 1, 0)
			return err == nil && len(users) > 0
		},
	})
	if err != nil {
		return fmt.Errorf("create web handler: %w", err)
	}

	// Create router with metrics and OpenAPI if enabled
	routerCfg := apihttp.RouterConfig{
		Metrics:       a.Metrics,
		EnableOpenAPI: a.Config.OpenAPI.Enabled,
		AdminHandler:  adminHandler.Router(),
		WebHandler:    webHandler.Router(),
	}
	router := apihttp.NewRouterWithConfig(proxyHandler, healthHandler, a.Logger, routerCfg)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", a.Config.Server.Host, a.Config.Server.Port)
	a.HTTPServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  a.Config.Server.ReadTimeout,
		WriteTimeout: a.Config.Server.WriteTimeout,
	}

	a.Logger.Info().Str("addr", addr).Msg("http server configured")
	return nil
}

func (a *App) buildDependencies() (app.ProxyDeps, error) {
	var deps app.ProxyDeps

	// Clock and ID generator (always local)
	deps.Clock = clock.Real{}
	deps.IDGen = idgen.UUID{}

	// Key store
	keyStore, err := a.buildKeyStore()
	if err != nil {
		return deps, fmt.Errorf("build key store: %w", err)
	}
	deps.Keys = keyStore

	// User store (always local for now)
	deps.Users = sqlite.NewUserStore(a.DB)

	// Rate limit store (always local - ephemeral)
	deps.RateLimit = memory.NewRateLimitStore()

	// Usage recorder
	usageRecorder, err := a.buildUsageRecorder()
	if err != nil {
		return deps, fmt.Errorf("build usage recorder: %w", err)
	}
	deps.Usage = usageRecorder
	a.usageRecorder = usageRecorder

	// Upstream client
	upstream, err := a.buildUpstream()
	if err != nil {
		return deps, fmt.Errorf("build upstream: %w", err)
	}
	deps.Upstream = upstream
	a.upstream = upstream

	return deps, nil
}

func (a *App) buildKeyStore() (ports.KeyStore, error) {
	switch a.Config.Auth.Mode {
	case "local":
		return sqlite.NewKeyStore(a.DB), nil
	case "remote":
		client := remote.NewClient(remote.ClientConfig{
			BaseURL: a.Config.Auth.Remote.URL,
			APIKey:  a.Config.Auth.Remote.APIKey,
			Timeout: a.Config.Auth.Remote.Timeout,
			Headers: a.Config.Auth.Remote.Headers,
		})
		return remote.NewKeyStore(client), nil
	default:
		return nil, fmt.Errorf("unknown auth mode: %s", a.Config.Auth.Mode)
	}
}

func (a *App) buildUsageRecorder() (ports.UsageRecorder, error) {
	switch a.Config.Usage.Mode {
	case "local":
		usageStore := sqlite.NewUsageStore(a.DB)
		return NewLocalUsageRecorder(usageStore, a.Config.Usage.BatchSize, a.Config.Usage.FlushInterval), nil
	case "remote":
		client := remote.NewClient(remote.ClientConfig{
			BaseURL: a.Config.Usage.Remote.URL,
			APIKey:  a.Config.Usage.Remote.APIKey,
			Timeout: a.Config.Usage.Remote.Timeout,
			Headers: a.Config.Usage.Remote.Headers,
		})
		return remote.NewUsageRecorder(client, remote.UsageRecorderConfig{
			BatchSize:     a.Config.Usage.BatchSize,
			FlushInterval: a.Config.Usage.FlushInterval,
		}), nil
	default:
		return nil, fmt.Errorf("unknown usage mode: %s", a.Config.Usage.Mode)
	}
}

func (a *App) buildUpstream() (*apihttp.UpstreamClient, error) {
	return apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL:         a.Config.Upstream.URL,
		Timeout:         a.Config.Upstream.Timeout,
		MaxIdleConns:    a.Config.Upstream.MaxIdleConns,
		IdleConnTimeout: a.Config.Upstream.IdleConnTimeout,
	})
}

func convertPlans(cfgPlans []config.PlanConfig) []plan.Plan {
	plans := make([]plan.Plan, len(cfgPlans))
	for i, p := range cfgPlans {
		plans[i] = plan.Plan{
			ID:                 p.ID,
			Name:               p.Name,
			RateLimitPerMinute: p.RateLimitPerMinute,
			RequestsPerMonth:   p.RequestsPerMonth,
			PriceMonthly:       p.PriceMonthly,
			OveragePrice:       p.OveragePrice,
		}
	}
	return plans
}

func convertEndpoints(cfgEndpoints []config.EndpointConfig) []plan.Endpoint {
	endpoints := make([]plan.Endpoint, len(cfgEndpoints))
	for i, e := range cfgEndpoints {
		endpoints[i] = plan.Endpoint{
			Method:         e.Method,
			Path:           e.Path,
			CostMultiplier: e.CostMultiplier,
		}
	}
	return endpoints
}

// Run starts the HTTP server and blocks until shutdown.
func (a *App) Run() error {
	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		a.Logger.Info().
			Str("addr", a.HTTPServer.Addr).
			Msg("starting http server")
		if err := a.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		a.Logger.Info().Str("signal", sig.String()).Msg("shutting down")
	}

	return a.Shutdown()
}

// Shutdown gracefully stops the application.
func (a *App) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop config watcher
	if a.ConfigHolder != nil {
		a.ConfigHolder.Stop()
	}

	// Stop route service refresh loop
	if a.routeService != nil {
		a.routeService.Stop()
	}

	// Shutdown HTTP server
	if a.HTTPServer != nil {
		if err := a.HTTPServer.Shutdown(ctx); err != nil {
			a.Logger.Error().Err(err).Msg("http server shutdown error")
		}
	}

	// Flush usage recorder
	if a.usageRecorder != nil {
		if err := a.usageRecorder.Close(); err != nil {
			a.Logger.Error().Err(err).Msg("usage recorder close error")
		}
	}

	// Close upstream
	if a.upstream != nil {
		a.upstream.Close()
	}

	// Close database
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			a.Logger.Error().Err(err).Msg("database close error")
		}
	}

	a.Logger.Info().Msg("shutdown complete")
	return nil
}

func setupLogger(cfg config.LoggingConfig) zerolog.Logger {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set output format
	var output zerolog.ConsoleWriter
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		return zerolog.New(output).With().Timestamp().Logger()
	}

	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
