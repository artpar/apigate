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

	// Create runtime with analytics
	mr.Runtime = runtime.New(adapter, runtime.Config{
		ModulesDir: cfg.ModulesDir,
		PluginsDir: cfg.PluginsDir,
		Analytics:  analyticsStore,
	})
	mr.Registry = mr.Runtime.Registry()

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

	// Load into runtime
	if err := mr.Runtime.LoadModule(mod); err != nil {
		return fmt.Errorf("load module %q: %w", mod.Name, err)
	}

	mr.modules = append(mr.modules, mod)
	mr.Logger.Debug().Str("module", mod.Name).Msg("loaded module")
	return nil
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
		Schema: map[string]schema.Field{
			"email":         {Type: schema.FieldTypeEmail, Unique: true, Lookup: true, Required: boolPtr(true)},
			"password_hash": {Type: schema.FieldTypeSecret, Internal: true},
			"name":          {Type: schema.FieldTypeString, Default: ""},
			"stripe_id":     {Type: schema.FieldTypeString, Internal: true},
			"plan_id":       {Type: schema.FieldTypeRef, To: "plan", Default: "free"},
			"status":        {Type: schema.FieldTypeEnum, Values: []string{"pending", "active", "suspended", "cancelled"}, Default: "active"},
		},
		Actions: map[string]schema.Action{
			"activate": {Set: map[string]string{"status": "active"}, Description: "Activate a user account"},
			"suspend":  {Set: map[string]string{"status": "suspended"}, Description: "Suspend a user account", Confirm: true},
			"cancel":   {Set: map[string]string{"status": "cancelled"}, Description: "Cancel a user account", Confirm: true},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/users"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "users"}},
		},
	}
}

// corePlanModule returns the plan module definition.
func corePlanModule() schema.Module {
	return schema.Module{
		Name: "plan",
		Schema: map[string]schema.Field{
			"name":                 {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true},
			"description":         {Type: schema.FieldTypeString, Default: ""},
			"rate_limit_per_minute": {Type: schema.FieldTypeInt, Default: 60},
			"requests_per_month":   {Type: schema.FieldTypeInt, Default: 1000},
			"price_monthly":        {Type: schema.FieldTypeInt, Default: 0},
			"overage_price":        {Type: schema.FieldTypeInt, Default: 0},
			"stripe_price_id":      {Type: schema.FieldTypeString},
			"paddle_price_id":      {Type: schema.FieldTypeString},
			"lemon_variant_id":     {Type: schema.FieldTypeString},
			"is_default":           {Type: schema.FieldTypeBool, Default: false},
			"enabled":              {Type: schema.FieldTypeBool, Default: true},
		},
		Actions: map[string]schema.Action{
			"enable":      {Set: map[string]string{"enabled": "true"}, Description: "Enable a pricing plan"},
			"disable":     {Set: map[string]string{"enabled": "false"}, Description: "Disable a pricing plan", Confirm: true},
			"set_default": {Set: map[string]string{"is_default": "true"}, Description: "Set as the default plan"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/plans"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "plans"}},
		},
	}
}

// coreAPIKeyModule returns the api_key module definition.
func coreAPIKeyModule() schema.Module {
	return schema.Module{
		Name: "api_key",
		Schema: map[string]schema.Field{
			"user_id":    {Type: schema.FieldTypeRef, To: "user", Required: boolPtr(true)},
			"hash":       {Type: schema.FieldTypeSecret, Internal: true},
			"prefix":     {Type: schema.FieldTypeString, Lookup: true},
			"name":       {Type: schema.FieldTypeString, Default: ""},
			"scopes":     {Type: schema.FieldTypeJSON},
			"expires_at": {Type: schema.FieldTypeTimestamp},
			"revoked_at": {Type: schema.FieldTypeTimestamp},
			"last_used":  {Type: schema.FieldTypeTimestamp},
		},
		Actions: map[string]schema.Action{
			"revoke": {Set: map[string]string{"revoked_at": "${NOW}"}, Description: "Revoke an API key", Confirm: true},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/keys"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "keys"}},
		},
	}
}

// coreRouteModule returns the route module definition.
func coreRouteModule() schema.Module {
	return schema.Module{
		Name: "route",
		Schema: map[string]schema.Field{
			"name":               {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true},
			"description":        {Type: schema.FieldTypeString, Default: ""},
			"path_pattern":       {Type: schema.FieldTypeString, Required: boolPtr(true)},
			"match_type":         {Type: schema.FieldTypeEnum, Values: []string{"exact", "prefix", "regex"}, Default: "prefix"},
			"methods":            {Type: schema.FieldTypeJSON},
			"headers":            {Type: schema.FieldTypeJSON},
			"upstream_id":        {Type: schema.FieldTypeRef, To: "upstream", Required: boolPtr(true)},
			"path_rewrite":       {Type: schema.FieldTypeString},
			"method_override":    {Type: schema.FieldTypeString},
			"request_transform":  {Type: schema.FieldTypeJSON},
			"response_transform": {Type: schema.FieldTypeJSON},
			"metering_expr":      {Type: schema.FieldTypeString, Default: "1"},
			"metering_mode":      {Type: schema.FieldTypeEnum, Values: []string{"request", "response_field", "bytes", "custom"}, Default: "request"},
			"protocol":           {Type: schema.FieldTypeEnum, Values: []string{"http", "http_stream", "sse", "websocket"}, Default: "http"},
			"priority":           {Type: schema.FieldTypeInt, Default: 0},
			"enabled":            {Type: schema.FieldTypeBool, Default: true},
		},
		Actions: map[string]schema.Action{
			"enable":  {Set: map[string]string{"enabled": "true"}, Description: "Enable a route"},
			"disable": {Set: map[string]string{"enabled": "false"}, Description: "Disable a route"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/routes"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "routes"}},
		},
	}
}

// coreUpstreamModule returns the upstream module definition.
func coreUpstreamModule() schema.Module {
	return schema.Module{
		Name: "upstream",
		Schema: map[string]schema.Field{
			"name":                 {Type: schema.FieldTypeString, Required: boolPtr(true), Lookup: true},
			"description":          {Type: schema.FieldTypeString, Default: ""},
			"base_url":             {Type: schema.FieldTypeString, Required: boolPtr(true)},
			"timeout_ms":           {Type: schema.FieldTypeInt, Default: 30000},
			"max_idle_conns":       {Type: schema.FieldTypeInt, Default: 100},
			"idle_conn_timeout_ms": {Type: schema.FieldTypeInt, Default: 90000},
			"auth_type":            {Type: schema.FieldTypeEnum, Values: []string{"none", "header", "bearer", "basic"}, Default: "none"},
			"auth_header":          {Type: schema.FieldTypeString, Required: boolPtr(false)},
			"auth_value_encrypted": {Type: schema.FieldTypeBytes, Required: boolPtr(false)}, // Encrypted auth value (BLOB in DB)
			"enabled":              {Type: schema.FieldTypeBool, Default: true},
		},
		Actions: map[string]schema.Action{
			"enable":  {Set: map[string]string{"enabled": "true"}, Description: "Enable an upstream"},
			"disable": {Set: map[string]string{"enabled": "false"}, Description: "Disable an upstream"},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/upstreams"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "upstreams"}},
		},
	}
}

// coreSettingModule returns the setting module definition.
func coreSettingModule() schema.Module {
	return schema.Module{
		Name: "setting",
		Schema: map[string]schema.Field{
			"key":       {Type: schema.FieldTypeString, Unique: true, Lookup: true, Required: boolPtr(true)},
			"value":     {Type: schema.FieldTypeString, Required: boolPtr(true)},
			"encrypted": {Type: schema.FieldTypeInt, Default: 0}, // SQLite stores as INTEGER
		},
		Actions: map[string]schema.Action{},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{Serve: schema.HTTPServe{Enabled: true, BasePath: "/settings"}},
			CLI:  schema.CLIChannel{Serve: schema.CLIServe{Enabled: true, Command: "settings"}},
		},
	}
}

