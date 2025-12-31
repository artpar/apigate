// Package bootstrap - modules.go provides integration for the declarative module system.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/artpar/apigate/core/analytics"
	cliChannel "github.com/artpar/apigate/core/channel/cli"
	httpChannel "github.com/artpar/apigate/core/channel/http"
	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/exporter"
	"github.com/artpar/apigate/core/registry"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/core/storage"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// ModuleRuntime wraps the declarative module runtime with app integration.
type ModuleRuntime struct {
	Runtime   *runtime.Runtime
	Registry  *registry.Registry
	Storage   *storage.SQLiteStore
	Analytics *analytics.SQLiteStore
	HTTP      *httpChannel.Channel
	CLI       *cliChannel.Channel
	Logger    zerolog.Logger

	modules []schema.Module
}

// ModuleConfig configures module loading.
type ModuleConfig struct {
	// ModulesDir is the directory containing module YAML files.
	// Defaults to "modules" relative to working directory.
	ModulesDir string

	// PluginsDir is the directory containing plugin modules.
	PluginsDir string

	// EmbeddedModules are modules defined in code (for core modules).
	EmbeddedModules []schema.Module
}

// NewModuleRuntime creates a new module runtime using an existing database.
func NewModuleRuntime(db *sql.DB, rootCmd *cobra.Command, logger zerolog.Logger, cfg ModuleConfig) (*ModuleRuntime, error) {
	mr := &ModuleRuntime{
		Logger: logger,
	}

	// Create storage adapter from existing DB
	mr.Storage = storage.NewSQLiteStoreFromDB(db)

	// Create analytics store
	analyticsStore, err := analytics.NewSQLiteStore(db, analytics.DefaultSQLiteConfig())
	if err != nil {
		return nil, fmt.Errorf("create analytics store: %w", err)
	}
	mr.Analytics = analyticsStore

	// Create storage adapter for runtime
	adapter := &runtimeStorageAdapter{store: mr.Storage}

	// Create runtime with analytics and logger
	mr.Runtime = runtime.New(adapter, runtime.Config{
		ModulesDir: cfg.ModulesDir,
		PluginsDir: cfg.PluginsDir,
		Analytics:  analyticsStore,
		Logger:     logger,
	})
	mr.Registry = mr.Runtime.Registry()

	// Register built-in functions and module hooks
	RegisterHooks(mr.Runtime, logger)

	// Register default Prometheus exporter
	promExporter := exporter.NewPrometheusExporter(exporter.PrometheusConfig{
		Store:  analyticsStore,
		Prefix: "apigate",
	})
	if err := mr.Runtime.RegisterExporter(promExporter); err != nil {
		logger.Warn().Err(err).Msg("failed to register prometheus exporter")
	} else {
		logger.Debug().Msg("prometheus exporter registered")
	}

	// Create HTTP channel (will be mounted later)
	mr.HTTP = httpChannel.New(mr.Runtime, "")

	// Create CLI channel
	mr.CLI = cliChannel.New(rootCmd, mr.Runtime)

	// Register channels with runtime
	mr.Runtime.RegisterChannel(mr.CLI)
	mr.Runtime.RegisterChannel(mr.HTTP)

	return mr, nil
}

// LoadModules loads modules from the configured directories and embedded modules.
func (mr *ModuleRuntime) LoadModules(ctx context.Context, cfg ModuleConfig) error {
	// Load embedded modules first
	for _, mod := range cfg.EmbeddedModules {
		if err := mr.loadModule(ctx, mod); err != nil {
			mr.Logger.Warn().Err(err).Str("module", mod.Name).Msg("failed to load embedded module")
		}
	}

	// Load modules from directory
	if cfg.ModulesDir != "" {
		if err := mr.loadModulesFromDir(ctx, cfg.ModulesDir); err != nil {
			mr.Logger.Warn().Err(err).Str("dir", cfg.ModulesDir).Msg("failed to load modules from directory")
		}
	}

	// Load plugin modules
	if cfg.PluginsDir != "" {
		if err := mr.loadModulesFromDir(ctx, cfg.PluginsDir); err != nil {
			mr.Logger.Warn().Err(err).Str("dir", cfg.PluginsDir).Msg("failed to load plugin modules")
		}
	}

	mr.Logger.Info().Int("count", len(mr.modules)).Msg("modules loaded")
	return nil
}

func (mr *ModuleRuntime) loadModulesFromDir(ctx context.Context, dir string) error {
	modules, err := schema.ParseDir(dir)
	if err != nil {
		return fmt.Errorf("parse modules from %q: %w", dir, err)
	}

	for _, mod := range modules {
		if err := mr.loadModule(ctx, mod); err != nil {
			mr.Logger.Warn().Err(err).Str("module", mod.Name).Msg("failed to load module")
			continue
		}
	}

	return nil
}

func (mr *ModuleRuntime) loadModule(ctx context.Context, mod schema.Module) error {
	// Validate module
	if err := schema.Validate(mod); err != nil {
		return fmt.Errorf("validate module %q: %w", mod.Name, err)
	}

	// Load into runtime (may fail if already registered)
	if err := mr.Runtime.LoadModule(mod); err != nil {
		// If module already registered, still register its hooks
		// This handles the case where embedded modules load first,
		// then YAML versions with hooks are loaded after
		if len(mod.Hooks) > 0 {
			mr.Runtime.RegisterModuleHooks(mod)
			mr.Logger.Info().
				Str("module", mod.Name).
				Int("hooks", countHooks(mod.Hooks)).
				Msg("registered YAML hooks for module")
		}
		return fmt.Errorf("load module %q: %w", mod.Name, err)
	}

	// Register YAML-declared hooks for this module
	mr.Runtime.RegisterModuleHooks(mod)

	mr.modules = append(mr.modules, mod)
	mr.Logger.Debug().Str("module", mod.Name).Msg("loaded module")
	return nil
}

// countHooks counts total hooks across all phases
func countHooks(hooks map[string][]schema.Hook) int {
	count := 0
	for _, h := range hooks {
		count += len(h)
	}
	return count
}

// Handler returns an HTTP handler for all module endpoints.
// This should be mounted at a base path like /api/v2 or /modules.
func (mr *ModuleRuntime) Handler() http.Handler {
	return mr.HTTP.Handler()
}

// MetricsHandler returns an HTTP handler for the /metrics endpoint.
// Returns nil if no pull exporters are registered.
func (mr *ModuleRuntime) MetricsHandler() http.Handler {
	exporters := mr.Runtime.Exporters().PullExporters()
	if len(exporters) == 0 {
		return nil
	}
	// Return the first pull exporter's handler (typically Prometheus)
	return exporters[0].Handler()
}

// Start starts the module runtime (channels, hooks, etc.).
func (mr *ModuleRuntime) Start(ctx context.Context) error {
	return mr.Runtime.Start(ctx)
}

// Stop stops the module runtime.
func (mr *ModuleRuntime) Stop(ctx context.Context) error {
	// Stop runtime first
	if err := mr.Runtime.Stop(ctx); err != nil {
		return err
	}
	// Close analytics (flushes pending events)
	if mr.Analytics != nil {
		return mr.Analytics.Close()
	}
	return nil
}

// Modules returns the list of loaded modules.
func (mr *ModuleRuntime) Modules() []schema.Module {
	return mr.modules
}

// GetModule returns a specific module by name.
func (mr *ModuleRuntime) GetModule(name string) (convention.Derived, bool) {
	return mr.Registry.Get(name)
}

// Execute executes an action on a module.
func (mr *ModuleRuntime) Execute(ctx context.Context, module, action string, input runtime.ActionInput) (runtime.ActionResult, error) {
	return mr.Runtime.Execute(ctx, module, action, input)
}

// GetHTTPPaths returns all HTTP paths registered by modules.
func (mr *ModuleRuntime) GetHTTPPaths() []schema.PathClaim {
	return mr.Registry.GetHTTPPaths()
}

// GetCLIPaths returns all CLI paths registered by modules.
func (mr *ModuleRuntime) GetCLIPaths() []schema.PathClaim {
	return mr.Registry.GetCLIPaths()
}

// runtimeStorageAdapter adapts storage.SQLiteStore to runtime.Storage.
type runtimeStorageAdapter struct {
	store *storage.SQLiteStore
}

func (a *runtimeStorageAdapter) CreateTable(ctx context.Context, mod convention.Derived) error {
	return a.store.CreateTable(ctx, mod)
}

func (a *runtimeStorageAdapter) Create(ctx context.Context, module string, data map[string]any) (string, error) {
	return a.store.Create(ctx, module, data)
}

func (a *runtimeStorageAdapter) Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error) {
	return a.store.Get(ctx, module, lookup, value)
}

func (a *runtimeStorageAdapter) List(ctx context.Context, module string, opts runtime.ListOptions) ([]map[string]any, int64, error) {
	return a.store.List(ctx, module, storage.ListOptions{
		Limit:     opts.Limit,
		Offset:    opts.Offset,
		Filters:   opts.Filters,
		OrderBy:   opts.OrderBy,
		OrderDesc: opts.OrderDesc,
	})
}

func (a *runtimeStorageAdapter) Update(ctx context.Context, module string, id string, data map[string]any) error {
	return a.store.Update(ctx, module, id, data)
}

func (a *runtimeStorageAdapter) Delete(ctx context.Context, module string, id string) error {
	return a.store.Delete(ctx, module, id)
}

// CoreModules returns the core module definitions that are embedded in the application.
// These define the standard user, plan, api_key, route, upstream, and setting modules.
// Note: Analytics is a runtime capability, not a data module.
func CoreModules() []schema.Module {
	return []schema.Module{
		coreUserModule(),
		corePlanModule(),
		coreAPIKeyModule(),
		coreRouteModule(),
		coreUpstreamModule(),
		coreSettingModule(),
	}
}

// CoreModulesDir returns the path to the core modules directory.
func CoreModulesDir() string {
	// This returns a path relative to the binary or working directory
	// In production, modules would be embedded or in a known location
	return filepath.Join("core", "modules")
}

// coreUserModule returns the user module definition.
func coreUserModule() schema.Module {
	return schema.Module{
		Name: "user",
		Meta: schema.ModuleMeta{
			Description: "User accounts for API access and billing",
		},
		Schema: map[string]schema.Field{
			"email":         {Type: schema.FieldTypeEmail, Unique: true, Lookup: true, Required: boolPtr(true), Description: "Primary email address for login and notifications"},
			"password_hash": {Type: schema.FieldTypeSecret, Internal: true, Description: "Hashed password for authentication"},
			"name":          {Type: schema.FieldTypeString, Default: "", Description: "Display name for the user"},
			"stripe_id":     {Type: schema.FieldTypeString, Internal: true, Description: "Stripe customer ID for payment processing"},
			"plan_id":       {Type: schema.FieldTypeRef, To: "plan", Default: "free", Description: "Reference to the user's pricing plan"},
			"status":        {Type: schema.FieldTypeEnum, Values: []string{"pending", "active", "suspended", "cancelled"}, Default: "active", Description: "Current account status controlling access"},
		},
		Actions: map[string]schema.Action{
			"activate": {Set: map[string]string{"status": "active"}, Description: "Activate a user account"},
			"suspend":  {Set: map[string]string{"status": "suspended"}, Description: "Suspend a user account", Confirm: true},
			"cancel":   {Set: map[string]string{"status": "cancelled"}, Description: "Cancel a user account", Confirm: true},
			"set_password": {
				Input: []schema.ActionInput{
					{Name: "password", Type: "secret", Required: true, Prompt: true, PromptText: "Enter new password"},
				},
				Description: "Set user password",
				Auth:        "admin",
			},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "users"}},
		},
	}
}

// corePlanModule returns the plan module definition.
func corePlanModule() schema.Module {
	return schema.Module{
		Name: "plan",
		Meta: schema.ModuleMeta{
			Description: "Pricing plans with rate limits and billing",
		},
		Schema: map[string]schema.Field{
			"name":                  {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true, Description: "Unique name identifying this pricing plan"},
			"description":           {Type: schema.FieldTypeString, Default: "", Description: "Human-readable description of plan features"},
			"rate_limit_per_minute": {Type: schema.FieldTypeInt, Default: 60, Description: "Maximum API requests allowed per minute"},
			"requests_per_month":    {Type: schema.FieldTypeInt, Default: 1000, Description: "Total API requests included per billing cycle"},
			"price_monthly":         {Type: schema.FieldTypeInt, Default: 0, Description: "Monthly subscription price in cents"},
			"overage_price":         {Type: schema.FieldTypeInt, Default: 0, Description: "Price per additional request beyond quota in cents"},
			"stripe_price_id":       {Type: schema.FieldTypeString, Description: "Stripe Price ID for subscription billing"},
			"paddle_price_id":       {Type: schema.FieldTypeString, Description: "Paddle Price ID for subscription billing"},
			"lemon_variant_id":      {Type: schema.FieldTypeString, Description: "LemonSqueezy variant ID for subscription billing"},
			"is_default":            {Type: schema.FieldTypeBool, Default: false, Description: "Whether this plan is assigned to new users"},
			"enabled":               {Type: schema.FieldTypeBool, Default: true, Description: "Whether this plan is available for selection"},
		},
		Actions: map[string]schema.Action{
			"enable":      {Set: map[string]string{"enabled": "true"}, Description: "Enable a pricing plan"},
			"disable":     {Set: map[string]string{"enabled": "false"}, Description: "Disable a pricing plan", Confirm: true},
			"set_default": {Set: map[string]string{"is_default": "true"}, Description: "Set as the default plan"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "plans"}},
		},
	}
}

// coreAPIKeyModule returns the api_key module definition.
func coreAPIKeyModule() schema.Module {
	return schema.Module{
		Name: "api_key",
		Meta: schema.ModuleMeta{
			Description: "API keys for authenticating API requests",
		},
		Schema: map[string]schema.Field{
			"user_id":    {Type: schema.FieldTypeRef, To: "user", Required: boolPtr(true), Description: "The user who owns this API key"},
			"hash":       {Type: schema.FieldTypeSecret, Internal: true, Description: "Cryptographic hash of the API key"},
			"prefix":     {Type: schema.FieldTypeString, Lookup: true, Description: "Visible key prefix for identification (e.g., ak_xxxxx)"},
			"name":       {Type: schema.FieldTypeString, Default: "", Description: "Human-readable label for this key"},
			"scopes":     {Type: schema.FieldTypeJSON, Description: "Array of permission scopes granted to this key"},
			"expires_at": {Type: schema.FieldTypeTimestamp, Description: "When this key expires and becomes invalid"},
			"revoked_at": {Type: schema.FieldTypeTimestamp, Internal: true, Description: "When this key was manually revoked"},
			"last_used":  {Type: schema.FieldTypeTimestamp, Internal: true, Description: "Timestamp of most recent API call with this key"},
		},
		Actions: map[string]schema.Action{
			"revoke": {Set: map[string]string{"revoked_at": "${NOW}"}, Description: "Revoke an API key", Confirm: true},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "keys"}},
		},
	}
}

// coreRouteModule returns the route module definition.
func coreRouteModule() schema.Module {
	return schema.Module{
		Name: "route",
		Meta: schema.ModuleMeta{
			Description: "API routing rules mapping paths to upstreams",
		},
		Schema: map[string]schema.Field{
			"name":               {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true, Description: "Unique name identifying this route"},
			"description":        {Type: schema.FieldTypeString, Default: "", Description: "Human-readable description of this route's purpose"},
			"path_pattern":       {Type: schema.FieldTypeString, Required: boolPtr(true), Description: "URL path pattern to match incoming requests"},
			"match_type":         {Type: schema.FieldTypeEnum, Values: []string{"exact", "prefix", "regex"}, Default: "prefix", Description: "How path_pattern is matched: exact, prefix, or regex"},
			"methods":            {Type: schema.FieldTypeJSON, Description: "HTTP methods to match (empty array matches all methods)"},
			"headers":            {Type: schema.FieldTypeJSON, Description: "Header conditions that must match for this route"},
			"upstream_id":        {Type: schema.FieldTypeRef, To: "upstream", Required: boolPtr(true), Description: "Backend service to forward matching requests to"},
			"path_rewrite":       {Type: schema.FieldTypeString, Description: "Expression to transform the request path before forwarding"},
			"method_override":    {Type: schema.FieldTypeString, Description: "Override the HTTP method when forwarding to upstream"},
			"request_transform":  {Type: schema.FieldTypeJSON, Description: "Rules to transform request headers and body"},
			"response_transform": {Type: schema.FieldTypeJSON, Description: "Rules to transform response headers and body"},
			"metering_expr":      {Type: schema.FieldTypeString, Default: "1", Description: "Expression to calculate request cost for rate limiting"},
			"metering_mode":      {Type: schema.FieldTypeEnum, Values: []string{"request", "response_field", "bytes", "custom"}, Default: "request", Description: "How API usage is measured for billing"},
			"protocol":           {Type: schema.FieldTypeEnum, Values: []string{"http", "http_stream", "sse", "websocket"}, Default: "http", Description: "Protocol handling mode for this route"},
			"priority":           {Type: schema.FieldTypeInt, Default: 0, Description: "Route matching priority (higher values match first)"},
			"enabled":            {Type: schema.FieldTypeBool, Default: true, Description: "Whether this route is active and processing requests"},
		},
		Actions: map[string]schema.Action{
			"enable":  {Set: map[string]string{"enabled": "true"}, Description: "Enable a route"},
			"disable": {Set: map[string]string{"enabled": "false"}, Description: "Disable a route"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "routes"}},
		},
		Hooks: map[string][]schema.Hook{
			"after_create": {{Call: "reload_router"}},
			"after_update": {{Call: "reload_router"}},
			"after_delete": {{Call: "reload_router"}},
		},
	}
}

// coreUpstreamModule returns the upstream module definition.
func coreUpstreamModule() schema.Module {
	return schema.Module{
		Name: "upstream",
		Meta: schema.ModuleMeta{
			Description: "Backend services that routes forward requests to",
		},
		Schema: map[string]schema.Field{
			"name":                 {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true, Description: "Unique name identifying this upstream service"},
			"description":          {Type: schema.FieldTypeString, Default: "", Description: "Human-readable description of this backend service"},
			"base_url":             {Type: schema.FieldTypeString, Required: boolPtr(true), Description: "Base URL of the backend service (e.g., https://api.example.com)"},
			"timeout_ms":           {Type: schema.FieldTypeInt, Default: 30000, Description: "Request timeout in milliseconds"},
			"max_idle_conns":       {Type: schema.FieldTypeInt, Default: 100, Description: "Maximum number of idle connections to maintain"},
			"idle_conn_timeout_ms": {Type: schema.FieldTypeInt, Default: 90000, Description: "How long idle connections are kept alive in milliseconds"},
			"auth_type":            {Type: schema.FieldTypeEnum, Values: []string{"none", "header", "bearer", "basic"}, Default: "none", Description: "Type of authentication to inject into upstream requests"},
			"auth_header":          {Type: schema.FieldTypeString, Required: boolPtr(false), Description: "Custom header name for authentication (when auth_type is header)"},
			"auth_value_encrypted": {Type: schema.FieldTypeBytes, Required: boolPtr(false), Description: "Encrypted authentication credentials"},
			"enabled":              {Type: schema.FieldTypeBool, Default: true, Description: "Whether this upstream is available for routing"},
		},
		Actions: map[string]schema.Action{
			"enable":  {Set: map[string]string{"enabled": "true"}, Description: "Enable an upstream"},
			"disable": {Set: map[string]string{"enabled": "false"}, Description: "Disable an upstream"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "upstreams"}},
		},
		Hooks: map[string][]schema.Hook{
			"after_create": {{Call: "reload_router"}},
			"after_update": {{Call: "reload_router"}},
			"after_delete": {{Call: "reload_router"}},
		},
	}
}

// coreSettingModule returns the setting module definition.
func coreSettingModule() schema.Module {
	return schema.Module{
		Name: "setting",
		Meta: schema.ModuleMeta{
			Description: "Application configuration settings",
		},
		Schema: map[string]schema.Field{
			"key":       {Type: schema.FieldTypeString, Unique: true, Lookup: true, Required: boolPtr(true), Description: "Unique configuration key (e.g., smtp.host, auth.jwt_secret)"},
			"value":     {Type: schema.FieldTypeString, Required: boolPtr(true), Description: "Configuration value (may be encrypted if sensitive)"},
			"encrypted": {Type: schema.FieldTypeInt, Default: 0, Description: "Whether the value is stored encrypted (0=plaintext, 1=encrypted)"},
		},
		Actions: map[string]schema.Action{},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "settings"}},
		},
	}
}

