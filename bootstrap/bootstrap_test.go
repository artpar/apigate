package bootstrap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/apigate/bootstrap"
	"github.com/spf13/cobra"
)

func TestBootstrap_Integration(t *testing.T) {
	// Create mock upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message": "hello from upstream"}`))
	}))
	defer upstream.Close()

	// Create temp directory for database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Set environment variables for bootstrap
	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogLevel, "debug")
	os.Setenv(bootstrap.EnvLogFormat, "console")
	defer func() {
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvLogLevel)
		os.Unsetenv(bootstrap.EnvLogFormat)
	}()

	// Create app (config loaded from database)
	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Verify components initialized
	if app.DB == nil {
		t.Error("DB should not be nil")
	}
	if app.HTTPServer == nil {
		t.Error("HTTPServer should not be nil")
	}
	if app.Settings == nil {
		t.Error("Settings should not be nil")
	}
}

func TestBootstrap_DatabaseMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migrate-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Verify tables exist by querying
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err = app.DB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Errorf("query users table: %v", err)
	}

	err = app.DB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys").Scan(&count)
	if err != nil {
		t.Errorf("query api_keys table: %v", err)
	}

	err = app.DB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_events").Scan(&count)
	if err != nil {
		t.Errorf("query usage_events table: %v", err)
	}

	// Verify settings table exists
	err = app.DB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM settings").Scan(&count)
	if err != nil {
		t.Errorf("query settings table: %v", err)
	}

	// Verify plans table exists
	err = app.DB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM plans").Scan(&count)
	if err != nil {
		t.Errorf("query plans table: %v", err)
	}
}

func TestBootstrap_GracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shutdown-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Shutdown should complete without error
	err = app.Shutdown()
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Verify DB is closed (should error on query)
	_, err = app.DB.DB.Query("SELECT 1")
	if err == nil {
		t.Error("expected error querying closed database")
	}
}

func TestBootstrap_SettingsLoad(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "settings-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Get settings
	s := app.Settings.Get()

	// Check defaults were loaded
	if s.Get("auth.key_prefix") != "ak_" {
		t.Errorf("expected key_prefix 'ak_', got '%s'", s.Get("auth.key_prefix"))
	}

	if s.Get("portal.app_name") != "APIGate" {
		t.Errorf("expected app_name 'APIGate', got '%s'", s.Get("portal.app_name"))
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		defaultVal int
		expected   int
	}{
		{"empty env", "TEST_GET_ENV_INT_EMPTY", "", 42, 42},
		{"valid int", "TEST_GET_ENV_INT_VALID", "100", 42, 100},
		{"invalid int", "TEST_GET_ENV_INT_INVALID", "abc", 42, 42},
		{"negative int", "TEST_GET_ENV_INT_NEG", "-5", 42, -5},
		{"zero", "TEST_GET_ENV_INT_ZERO", "0", 42, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			result := bootstrap.GetEnvInt(tt.envKey, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetEnvInt(%q, %d) = %d, expected %d", tt.envKey, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestBootstrap_Reload(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "reload-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Reload settings
	err = app.Reload()
	if err != nil {
		t.Errorf("Reload should not error: %v", err)
	}

	// Verify settings are still accessible
	s := app.Settings.Get()
	if s.Get("auth.key_prefix") != "ak_" {
		t.Errorf("expected key_prefix 'ak_' after reload, got '%s'", s.Get("auth.key_prefix"))
	}
}

func TestBootstrap_ReloadPlans(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "reload-plans-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	ctx := context.Background()

	// Insert a test plan
	_, err = app.DB.DB.ExecContext(ctx, `
		INSERT INTO plans (id, name, rate_limit_per_minute, requests_per_month, price_monthly, overage_price, enabled)
		VALUES ('test_plan', 'Test Plan', 100, 10000, 999, 1, 1)
	`)
	if err != nil {
		t.Fatalf("insert test plan: %v", err)
	}

	// Reload plans
	err = app.ReloadPlans(ctx)
	if err != nil {
		t.Errorf("ReloadPlans should not error: %v", err)
	}
}

func TestBootstrap_Capabilities(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capabilities-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Verify capability container is initialized
	if app.Capabilities == nil {
		t.Error("Capabilities should not be nil")
	}

	// Verify capabilities are registered
	ctx := context.Background()
	caps := app.Capabilities.ListCapabilities()
	if len(caps) == 0 {
		t.Error("should have at least one capability registered")
	}

	// Verify cache provider is available
	cache, err := app.Capabilities.Cache(ctx)
	if err != nil {
		t.Errorf("Cache() error: %v", err)
	}
	if cache == nil {
		t.Error("cache provider should not be nil")
	}

	// Verify storage provider is available
	storage, err := app.Capabilities.Storage(ctx)
	if err != nil {
		t.Errorf("Storage() error: %v", err)
	}
	if storage == nil {
		t.Error("storage provider should not be nil")
	}

	// Verify queue provider is available
	queue, err := app.Capabilities.Queue(ctx)
	if err != nil {
		t.Errorf("Queue() error: %v", err)
	}
	if queue == nil {
		t.Error("queue provider should not be nil")
	}
}

func TestBootstrap_WithConfig(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "config-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	rootCmd := &cobra.Command{Use: "test"}

	app, err := bootstrap.NewWithConfig(bootstrap.Config{
		RootCmd: rootCmd,
	})
	if err != nil {
		t.Fatalf("create app with config: %v", err)
	}
	defer app.Shutdown()

	// Verify module runtime is initialized when RootCmd is provided
	if app.ModuleRuntime == nil {
		t.Error("ModuleRuntime should be initialized when RootCmd is provided")
	}
}

func TestBootstrap_InitModuleRuntime(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "module-runtime-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Module runtime should be nil when not initialized with RootCmd
	if app.ModuleRuntime != nil {
		t.Error("ModuleRuntime should be nil when no RootCmd provided")
	}

	// Initialize module runtime
	rootCmd := &cobra.Command{Use: "test"}
	err = app.InitModuleRuntime(rootCmd)
	if err != nil {
		t.Errorf("InitModuleRuntime should not error: %v", err)
	}

	// Now module runtime should be available
	if app.ModuleRuntime == nil {
		t.Error("ModuleRuntime should be initialized after InitModuleRuntime")
	}
}

func TestBootstrap_InitModuleRuntime_NilCmd(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "module-nil-cmd-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Initialize with nil should still work
	err = app.InitModuleRuntime(nil)
	if err != nil {
		t.Errorf("InitModuleRuntime(nil) should not error: %v", err)
	}
}

func TestBootstrap_LoadPlans_DefaultPlan(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "default-plan-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// When no plans exist in DB, default plan should be used
	// We verify this by checking ReloadPlans doesn't error
	ctx := context.Background()
	err = app.ReloadPlans(ctx)
	if err != nil {
		t.Errorf("ReloadPlans with default plan should not error: %v", err)
	}
}

func TestBootstrap_LoadPlans_WithQuotaModes(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "quota-modes-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	ctx := context.Background()

	// Insert plans with different quota modes
	plans := []struct {
		id   string
		mode string
	}{
		{"hard_plan", "hard"},
		{"warn_plan", "warn"},
		{"soft_plan", "soft"},
	}

	for _, p := range plans {
		_, err = app.DB.DB.ExecContext(ctx, `
			INSERT INTO plans (id, name, rate_limit_per_minute, requests_per_month, price_monthly, overage_price, enabled, quota_enforce_mode, quota_grace_pct)
			VALUES (?, ?, 100, 10000, 999, 1, 1, ?, 0.05)
		`, p.id, p.id, p.mode)
		if err != nil {
			t.Fatalf("insert plan %s: %v", p.id, err)
		}
	}

	// Reload plans
	err = app.ReloadPlans(ctx)
	if err != nil {
		t.Errorf("ReloadPlans should not error: %v", err)
	}
}

func TestBootstrap_ServerConfig_FromEnv(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "server-config-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvServerHost, "127.0.0.1")
	os.Setenv(bootstrap.EnvServerPort, "9999")
	defer func() {
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvServerHost)
		os.Unsetenv(bootstrap.EnvServerPort)
	}()

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Verify server is configured with env values
	expectedAddr := "127.0.0.1:9999"
	if app.HTTPServer.Addr != expectedAddr {
		t.Errorf("HTTPServer.Addr = %q, expected %q", app.HTTPServer.Addr, expectedAddr)
	}
}

func TestBootstrap_LogLevel_FromEnv(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "log-level-test.db")

	// Test different log levels
	levels := []string{"debug", "info", "warn", "error", "invalid"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
			os.Setenv(bootstrap.EnvLogLevel, level)
			defer func() {
				os.Unsetenv(bootstrap.EnvDatabaseDSN)
				os.Unsetenv(bootstrap.EnvLogLevel)
			}()

			app, err := bootstrap.New()
			if err != nil {
				t.Fatalf("create app with log level %s: %v", level, err)
			}
			app.Shutdown()
		})
	}
}

func TestBootstrap_LogFormat_JSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "log-format-json-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogFormat, "json") // default format
	defer func() {
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvLogFormat)
	}()

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()
}

func TestBootstrap_Metrics_Enabled(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metrics-enabled-test.db")

	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Enable metrics through settings
	ctx := context.Background()
	_, err = app.DB.DB.ExecContext(ctx, `
		INSERT OR REPLACE INTO settings (key, value, encrypted)
		VALUES ('metrics.enabled', 'true', 0)
	`)
	if err != nil {
		t.Fatalf("insert metrics setting: %v", err)
	}
}

func TestBootstrap_DefaultDatabase(t *testing.T) {
	// Clear any existing DSN env var
	os.Unsetenv(bootstrap.EnvDatabaseDSN)

	// Clean up any existing default database file
	os.Remove("apigate.db")
	defer os.Remove("apigate.db")

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app with default database: %v", err)
	}
	defer app.Shutdown()

	// Verify database was created at default location
	if _, err := os.Stat("apigate.db"); os.IsNotExist(err) {
		t.Error("default database file should be created at apigate.db")
	}
}
