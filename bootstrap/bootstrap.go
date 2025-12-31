// Package bootstrap wires all dependencies and starts the application.
// Configuration is loaded from the database, with minimal environment variables
// only for bootstrap (database connection and server port).
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/artpar/apigate/adapters/clock"
	"github.com/artpar/apigate/adapters/email"
	"github.com/artpar/apigate/adapters/hasher"
	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/adapters/idgen"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/adapters/metrics"
	"github.com/artpar/apigate/adapters/payment"
	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/core/capability"
	capAdapters "github.com/artpar/apigate/core/capability/adapters"
	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/artpar/apigate/web"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Environment variable names for bootstrap configuration.
// These are the ONLY config values that come from environment.
const (
	EnvDatabaseDSN = "APIGATE_DATABASE_DSN"
	EnvServerPort  = "APIGATE_SERVER_PORT"
	EnvServerHost  = "APIGATE_SERVER_HOST"
	EnvLogLevel    = "APIGATE_LOG_LEVEL"
	EnvLogFormat   = "APIGATE_LOG_FORMAT"
)

// App represents the running application.
type App struct {
	Logger     zerolog.Logger
	DB         *sqlite.DB
	HTTPServer *http.Server
	Metrics    *metrics.Collector
	Settings   *app.SettingsService

	// Capability container (DI for pluggable providers)
	Capabilities *capability.Container

	// Services
	proxyService     *app.ProxyService
	routeService     *app.RouteService
	transformService *app.TransformService

	// Module runtime (declarative modules)
	ModuleRuntime *ModuleRuntime

	// Adapters (for cleanup)
	usageRecorder   ports.UsageRecorder
	upstream        *apihttp.UpstreamClient
	paymentProvider ports.PaymentProvider
	emailSender     ports.EmailSender
}

// Config provides optional configuration for application initialization.
type Config struct {
	// RootCmd is the cobra root command for CLI module integration.
	// If provided, module CLI commands will be registered.
	RootCmd *cobra.Command
}

// New creates and initializes the application.
// Configuration is loaded from the database after connection.
func New() (*App, error) {
	return NewWithConfig(Config{})
}

// NewWithConfig creates and initializes the application with custom configuration.
func NewWithConfig(cfg Config) (*App, error) {
	// Setup logger from env (only bootstrap config from env)
	logger := setupLoggerFromEnv()

	logger.Info().Msg("initializing apigate")

	a := &App{
		Logger: logger,
	}

	// Initialize database (DSN from env with default)
	if err := a.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// Load settings from database
	settingsStore := sqlite.NewSettingsStore(a.DB)
	a.Settings = app.NewSettingsService(settingsStore, logger)
	if err := a.Settings.Load(context.Background()); err != nil {
		logger.Warn().Err(err).Msg("failed to load settings, using defaults")
	}

	// Initialize metrics if enabled
	s := a.Settings.Get()
	if s.GetBool("metrics.enabled") {
		a.Metrics = metrics.New()
		logger.Info().Msg("prometheus metrics enabled")
	}

	// Initialize capability container (DI for pluggable providers)
	capContainer, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize capability container")
	} else {
		a.Capabilities = capContainer
	}

	// Initialize module runtime if root command provided
	if cfg.RootCmd != nil {
		if err := a.InitModuleRuntime(cfg.RootCmd); err != nil {
			logger.Warn().Err(err).Msg("failed to initialize module runtime")
		}
	}

	// Initialize HTTP server (after module runtime so handlers are available)
	if err := a.initHTTPServer(); err != nil {
		return nil, fmt.Errorf("init http server: %w", err)
	}

	return a, nil
}

// InitModuleRuntime initializes the declarative module runtime.
// This must be called after New() and before Run() if you want module support.
// The rootCmd is the cobra root command for CLI integration.
func (a *App) InitModuleRuntime(rootCmd interface{}) error {
	// Type assert to cobra command if provided
	var cobraCmd *cobra.Command
	if rootCmd != nil {
		if cmd, ok := rootCmd.(*cobra.Command); ok {
			cobraCmd = cmd
		}
	}

	// Create module runtime
	mr, err := NewModuleRuntime(a.DB.DB, cobraCmd, a.Logger, ModuleConfig{})
	if err != nil {
		return fmt.Errorf("create module runtime: %w", err)
	}

	// Load core modules (embedded definitions)
	ctx := context.Background()
	if err := mr.LoadModules(ctx, ModuleConfig{
		EmbeddedModules: CoreModules(),
		ModulesDir:      CoreModulesDir(),
	}); err != nil {
		a.Logger.Warn().Err(err).Msg("failed to load some modules")
	}

	a.ModuleRuntime = mr

	// Log loaded modules
	for _, mod := range mr.Modules() {
		a.Logger.Info().Str("module", mod.Name).Msg("loaded declarative module")
	}

	// Log HTTP paths
	paths := mr.GetHTTPPaths()
	a.Logger.Info().Int("count", len(paths)).Msg("module HTTP endpoints registered")

	return nil
}

func (a *App) initDatabase() error {
	dsn := os.Getenv(EnvDatabaseDSN)
	if dsn == "" {
		dsn = "apigate.db"
	}

	db, err := sqlite.Open(dsn)
	if err != nil {
		return err
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		return fmt.Errorf("migrate: %w", err)
	}

	a.DB = db
	a.Logger.Info().Str("dsn", dsn).Msg("database initialized")
	return nil
}

func (a *App) initHTTPServer() error {
	s := a.Settings.Get()
	ctx := context.Background()

	// Build dependencies
	deps, err := a.buildDependencies(s)
	if err != nil {
		return err
	}

	// Build proxy config from settings and database plans
	plans := a.loadPlans(ctx)
	proxyCfg := app.ProxyConfig{
		KeyPrefix:  s.GetOrDefault(settings.KeyAuthKeyPrefix, "ak_"),
		Plans:      plans,
		Endpoints:  nil, // Load from database if needed
		RateBurst:  s.GetInt(settings.KeyRateLimitBurstTokens, 5),
		RateWindow: s.GetInt(settings.KeyRateLimitWindowSecs, 60),
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

	// Start route service to load initial routes
	if err := a.routeService.Start(ctx); err != nil {
		a.Logger.Warn().Err(err).Msg("failed to start route service, continuing with empty routes")
	}

	// Wire router reloader for hook-triggered reloads
	SetRouterReloader(a.routeService)

	// Wire plan reloader for hook-triggered plan reloads
	SetPlanReloader(a)

	// Create and wire transform service
	a.transformService = app.NewTransformService()
	a.proxyService.SetTransformService(a.transformService)

	a.Logger.Info().Msg("route and transform services initialized")

	// Create HTTP handlers
	var proxyHandler *apihttp.ProxyHandler
	if a.Metrics != nil {
		proxyHandler = apihttp.NewProxyHandlerWithMetrics(a.proxyService, a.Logger, a.Metrics)
	} else {
		proxyHandler = apihttp.NewProxyHandler(a.proxyService, a.Logger)
	}
	proxyHandler.SetStreamingUpstream(a.upstream)
	healthHandler := apihttp.NewHealthHandler(a.upstream)

	// Create shared stores for admin and web handlers
	usageStore := sqlite.NewUsageStore(a.DB)
	planStore := sqlite.NewPlanStore(a.DB)
	bcryptHasher := hasher.NewBcrypt(0)
	sessionStore := sqlite.NewSessionStore(a.DB)
	tokenStore := sqlite.NewTokenStore(a.DB)

	// Create email sender (used by both admin and portal)
	emailSender, err := email.NewSender(s)
	if err != nil {
		a.Logger.Warn().Err(err).Msg("failed to create email sender, email features disabled")
		emailSender = email.NewNoopSender()
	}
	a.emailSender = emailSender

	// Register email provider with capability container
	if a.Capabilities != nil {
		emailAdapter := capAdapters.WrapEmail("default", emailSender)
		if err := a.Capabilities.RegisterEmail("default", emailAdapter, true); err != nil {
			a.Logger.Debug().Err(err).Msg("email already registered in capability container")
		}
	}

	// Create admin handler
	adminHandler := admin.NewHandler(admin.Deps{
		Users:     deps.Users,
		Keys:      deps.Keys,
		Usage:     usageStore,
		Routes:    routeStore,
		Upstreams: upstreamStore,
		Plans:     planStore,
		Logger:    a.Logger,
		Hasher:    bcryptHasher,
	})

	// Create web UI handler
	webHandler, err := web.NewHandler(web.Deps{
		Users:       deps.Users,
		Keys:        deps.Keys,
		Usage:       usageStore,
		Routes:      routeStore,
		Upstreams:   upstreamStore,
		Plans:       planStore,
		Settings:    a.Settings.Store(),
		AuthTokens:  tokenStore,
		EmailSender: emailSender,
		AppSettings: web.AppSettings{
			UpstreamURL:     s.Get(settings.KeyUpstreamURL),
			UpstreamTimeout: s.GetOrDefault(settings.KeyUpstreamTimeout, "30s"),
			AuthMode:        s.GetOrDefault(settings.KeyAuthMode, "local"),
			AuthHeader:      s.GetOrDefault(settings.KeyAuthHeader, "X-API-Key"),
			DatabaseDSN:     os.Getenv(EnvDatabaseDSN),
		},
		Logger:        a.Logger,
		Hasher:        bcryptHasher,
		JWTSecret:     s.Get(settings.KeyAuthJWTSecret),
		ExprValidator: a.transformService,
		RouteTester:   a.routeService,
		IsSetup: func() bool {
			users, err := deps.Users.List(context.Background(), 1, 0)
			return err == nil && len(users) > 0
		},
		OnPlanChange: func(ctx context.Context) error {
			return a.ReloadPlans(ctx)
		},
		OnRouteChange: func(ctx context.Context) error {
			if a.routeService != nil {
				return a.routeService.Reload(ctx)
			}
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("create web handler: %w", err)
	}

	// Create payment provider (if configured)
	paymentProvider, err := payment.NewProvider(s)
	if err != nil {
		a.Logger.Warn().Err(err).Msg("failed to create payment provider")
		paymentProvider = payment.NewNoopProvider()
	}
	a.paymentProvider = paymentProvider

	// Create user portal handler (if enabled)
	var portalRouter http.Handler
	if s.GetBool(settings.KeyPortalEnabled) {
		portalHandler, err := web.NewPortalHandler(web.PortalDeps{
			Users:       deps.Users,
			Keys:        deps.Keys,
			Usage:       usageStore,
			Plans:       planStore,
			Sessions:    sessionStore,
			AuthTokens:  tokenStore,
			EmailSender: emailSender,
			Settings:    a.Settings.Store(),
			Logger:      a.Logger,
			Hasher:      bcryptHasher,
			IDGen:       deps.IDGen,
			Payment:     paymentProvider,
			JWTSecret:   s.Get(settings.KeyAuthJWTSecret),
			BaseURL:     s.Get(settings.KeyPortalBaseURL),
			AppName:     s.GetOrDefault(settings.KeyPortalAppName, "APIGate"),
		})
		if err != nil {
			return fmt.Errorf("create portal handler: %w", err)
		}
		portalRouter = portalHandler.Router()
		a.Logger.Info().Msg("user portal enabled at /portal")
	}

	// Register payment provider with capability container
	if a.Capabilities != nil {
		paymentAdapter := capAdapters.WrapPayment(paymentProvider)
		if err := a.Capabilities.RegisterPayment("default", paymentAdapter, true); err != nil {
			a.Logger.Debug().Err(err).Msg("payment already registered in capability container")
		}
	}

	// Register hasher with capability container
	if a.Capabilities != nil {
		hasherAdapter := capAdapters.WrapHasher("bcrypt", bcryptHasher)
		if err := a.Capabilities.RegisterHasher("bcrypt", hasherAdapter, true); err != nil {
			a.Logger.Debug().Err(err).Msg("hasher already registered in capability container")
		}
	}

	// Create developer documentation portal handler (always enabled for self-service)
	docsHandler := web.NewDocsHandler(web.DocsDeps{
		Modules: func() map[string]convention.Derived {
			result := make(map[string]convention.Derived)
			if a.ModuleRuntime != nil {
				for _, mod := range a.ModuleRuntime.Modules() {
					result[mod.Name] = convention.Derive(mod)
				}
			}
			return result
		},
		Routes:   routeStore,
		Settings: a.Settings.Store(),
		Logger:   a.Logger,
		AppName:  s.GetOrDefault(settings.KeyPortalAppName, "APIGate"),
	})
	docsRouter := docsHandler.Router()
	a.Logger.Info().Msg("developer documentation portal enabled at /docs")

	// Create router
	routerCfg := apihttp.RouterConfig{
		Metrics:       a.Metrics,
		EnableOpenAPI: s.GetBool("openapi.enabled"),
		AdminHandler:  adminHandler.Router(),
		WebHandler:    webHandler.Router(),
		PortalHandler: portalRouter,
		DocsHandler:   docsRouter,
	}

	// Add module handler if runtime is initialized
	if a.ModuleRuntime != nil {
		routerCfg.ModuleHandler = a.ModuleRuntime.Handler()
		a.Logger.Info().Msg("module handler mounted at /mod")

		// Add metrics handler from exporter
		if metricsHandler := a.ModuleRuntime.MetricsHandler(); metricsHandler != nil {
			routerCfg.MetricsHandler = metricsHandler
			a.Logger.Info().Msg("prometheus metrics handler mounted at /metrics")
		}
	}

	router := apihttp.NewRouterWithConfig(proxyHandler, healthHandler, a.Logger, routerCfg)

	// Get server config from env (bootstrap) or settings
	host := os.Getenv(EnvServerHost)
	if host == "" {
		host = s.GetOrDefault(settings.KeyServerHost, "0.0.0.0")
	}
	port := os.Getenv(EnvServerPort)
	if port == "" {
		port = s.GetOrDefault(settings.KeyServerPort, "8080")
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	readTimeout := s.GetDuration(settings.KeyServerReadTimeout, 30*time.Second)
	writeTimeout := s.GetDuration(settings.KeyServerWriteTimeout, 60*time.Second)

	a.HTTPServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	a.Logger.Info().Str("addr", addr).Msg("http server configured")
	return nil
}

func (a *App) buildDependencies(s settings.Settings) (app.ProxyDeps, error) {
	var deps app.ProxyDeps

	// Clock and ID generator (always local)
	deps.Clock = clock.Real{}
	deps.IDGen = idgen.UUID{}

	// Key store (always local for now)
	deps.Keys = sqlite.NewKeyStore(a.DB)

	// User store
	deps.Users = sqlite.NewUserStore(a.DB)

	// Rate limit store (sharded for high throughput)
	deps.RateLimit = memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards:       32,
		CleanupInterval: 5 * time.Minute,
	})

	// Usage recorder
	usageStore := sqlite.NewUsageStore(a.DB)

	// Quota store (sharded, syncs with usage store)
	deps.Quota = memory.NewQuotaStore(memory.QuotaStoreConfig{
		NumShards:       32,
		CleanupInterval: time.Hour,
		UsageStore:      usageStore,
	})
	deps.Usage = NewLocalUsageRecorder(usageStore, 100, 10*time.Second)
	a.usageRecorder = deps.Usage

	// Upstream client
	upstreamURL := s.Get(settings.KeyUpstreamURL)
	if upstreamURL == "" {
		upstreamURL = "http://localhost:8081" // Default fallback
	}

	upstream, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL:         upstreamURL,
		Timeout:         s.GetDuration(settings.KeyUpstreamTimeout, 30*time.Second),
		MaxIdleConns:    s.GetInt(settings.KeyUpstreamMaxIdleConns, 100),
		IdleConnTimeout: s.GetDuration(settings.KeyUpstreamIdleConnTimeout, 90*time.Second),
	})
	if err != nil {
		return deps, fmt.Errorf("build upstream: %w", err)
	}
	deps.Upstream = upstream
	a.upstream = upstream

	return deps, nil
}

func (a *App) loadPlans(ctx context.Context) []plan.Plan {
	// Load plans from database (with quota fields using COALESCE for backwards compatibility)
	rows, err := a.DB.DB.QueryContext(ctx, `
		SELECT id, name, rate_limit_per_minute, requests_per_month, price_monthly, overage_price,
		       COALESCE(quota_enforce_mode, 'hard') as quota_enforce_mode,
		       COALESCE(quota_grace_pct, 0.05) as quota_grace_pct
		FROM plans WHERE enabled = 1
	`)
	if err != nil {
		a.Logger.Warn().Err(err).Msg("failed to load plans, using default")
		return []plan.Plan{{
			ID:                 "free",
			Name:               "Free",
			RateLimitPerMinute: 60,
			RequestsPerMonth:   1000,
			QuotaEnforceMode:   plan.QuotaEnforceHard,
			QuotaGracePct:      0.05,
		}}
	}
	defer rows.Close()

	var plans []plan.Plan
	for rows.Next() {
		var p plan.Plan
		var enforceMode string
		if err := rows.Scan(&p.ID, &p.Name, &p.RateLimitPerMinute, &p.RequestsPerMonth, &p.PriceMonthly, &p.OveragePrice, &enforceMode, &p.QuotaGracePct); err != nil {
			continue
		}
		// Convert enforce mode string to type
		switch enforceMode {
		case "warn":
			p.QuotaEnforceMode = plan.QuotaEnforceWarn
		case "soft":
			p.QuotaEnforceMode = plan.QuotaEnforceSoft
		default:
			p.QuotaEnforceMode = plan.QuotaEnforceHard
		}
		plans = append(plans, p)
	}

	if len(plans) == 0 {
		return []plan.Plan{{
			ID:                 "free",
			Name:               "Free",
			RateLimitPerMinute: 60,
			RequestsPerMonth:   1000,
			QuotaEnforceMode:   plan.QuotaEnforceHard,
			QuotaGracePct:      0.05,
		}}
	}
	return plans
}

// Run starts the HTTP server and blocks until shutdown.
func (a *App) Run() error {
	ctx := context.Background()

	// Start module runtime if initialized
	if a.ModuleRuntime != nil {
		if err := a.ModuleRuntime.Start(ctx); err != nil {
			a.Logger.Warn().Err(err).Msg("failed to start module runtime")
		}
	}

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

	// Stop module runtime
	if a.ModuleRuntime != nil {
		if err := a.ModuleRuntime.Stop(ctx); err != nil {
			a.Logger.Error().Err(err).Msg("module runtime stop error")
		}
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

	// Close capability container (releases provider resources)
	if a.Capabilities != nil {
		if err := a.Capabilities.Close(); err != nil {
			a.Logger.Error().Err(err).Msg("capability container close error")
		}
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

// Reload reloads settings from the database.
func (a *App) Reload() error {
	ctx := context.Background()
	if err := a.Settings.Load(ctx); err != nil {
		return err
	}

	s := a.Settings.Get()

	// Update proxy service with new config
	if a.proxyService != nil {
		plans := a.loadPlans(ctx)
		a.proxyService.UpdateConfig(
			plans,
			nil, // endpoints
			s.GetInt(settings.KeyRateLimitBurstTokens, 5),
			s.GetInt(settings.KeyRateLimitWindowSecs, 60),
		)
	}

	a.Logger.Info().Msg("settings reloaded from database")
	return nil
}

// ReloadPlans reloads only the plans from the database into the proxy service.
// This is called by the reload_plans hook after plan create/update/delete.
func (a *App) ReloadPlans(ctx context.Context) error {
	a.Logger.Info().Msg("ReloadPlans called")

	if a.proxyService == nil {
		a.Logger.Warn().Msg("ReloadPlans: proxyService is nil")
		return nil
	}

	s := a.Settings.Get()
	plans := a.loadPlans(ctx)

	// Log loaded plans for debugging
	for _, p := range plans {
		a.Logger.Info().
			Str("id", p.ID).
			Str("name", p.Name).
			Int64("requests_per_month", p.RequestsPerMonth).
			Msg("loaded plan")
	}

	a.proxyService.UpdateConfig(
		plans,
		nil, // endpoints - keep existing
		s.GetInt(settings.KeyRateLimitBurstTokens, 5),
		s.GetInt(settings.KeyRateLimitWindowSecs, 60),
	)

	a.Logger.Info().Int("count", len(plans)).Msg("plans reloaded from database")
	return nil
}

func setupLoggerFromEnv() zerolog.Logger {
	levelStr := os.Getenv(EnvLogLevel)
	if levelStr == "" {
		levelStr = "info"
	}

	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	format := os.Getenv(EnvLogFormat)
	if format == "console" {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		return zerolog.New(output).With().Timestamp().Logger()
	}

	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// GetEnvInt returns an integer from env or default.
func GetEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}

// noopEmailSender is a no-op email sender used when email is disabled.
type noopEmailSender struct{}

func (n *noopEmailSender) Send(ctx context.Context, msg ports.EmailMessage) error {
	return nil
}

func (n *noopEmailSender) SendVerification(ctx context.Context, to, name, token string) error {
	return nil
}

func (n *noopEmailSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	return nil
}

func (n *noopEmailSender) SendWelcome(ctx context.Context, to, name string) error {
	return nil
}
