package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/apigate/config"
)

func TestLoad_ValidConfig(t *testing.T) {
	content := `
server:
  host: "127.0.0.1"
  port: 9090

upstream:
  url: "http://localhost:3000"
  timeout: 15s

auth:
  mode: "local"
  key_prefix: "test_"

database:
  driver: "sqlite"
  dsn: ":memory:"

plans:
  - id: "free"
    name: "Free Plan"
    rate_limit_per_minute: 100
    requests_per_month: 10000
`

	cfg := writeAndLoad(t, content)

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Host = %s, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Upstream.URL != "http://localhost:3000" {
		t.Errorf("Upstream.URL = %s, want http://localhost:3000", cfg.Upstream.URL)
	}
	if cfg.Upstream.Timeout != 15*time.Second {
		t.Errorf("Upstream.Timeout = %v, want 15s", cfg.Upstream.Timeout)
	}
	if cfg.Auth.KeyPrefix != "test_" {
		t.Errorf("Auth.KeyPrefix = %s, want test_", cfg.Auth.KeyPrefix)
	}
	if len(cfg.Plans) != 1 {
		t.Fatalf("len(Plans) = %d, want 1", len(cfg.Plans))
	}
	if cfg.Plans[0].ID != "free" {
		t.Errorf("Plans[0].ID = %s, want free", cfg.Plans[0].ID)
	}
}

func TestLoad_Defaults(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"
`

	cfg := writeAndLoad(t, content)

	// Check defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("default Host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Auth.Mode != "local" {
		t.Errorf("default Auth.Mode = %s, want local", cfg.Auth.Mode)
	}
	if cfg.Auth.KeyPrefix != "ak_" {
		t.Errorf("default Auth.KeyPrefix = %s, want ak_", cfg.Auth.KeyPrefix)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("default Database.Driver = %s, want sqlite", cfg.Database.Driver)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default Logging.Level = %s, want info", cfg.Logging.Level)
	}
	// Default free plan should be added
	if len(cfg.Plans) != 1 || cfg.Plans[0].ID != "free" {
		t.Errorf("default plan not added: %v", cfg.Plans)
	}
}

func TestLoad_EnvExpansion(t *testing.T) {
	os.Setenv("TEST_UPSTREAM_URL", "http://env-test:3000")
	defer os.Unsetenv("TEST_UPSTREAM_URL")

	content := `
upstream:
  url: "${TEST_UPSTREAM_URL}"
`

	cfg := writeAndLoad(t, content)

	if cfg.Upstream.URL != "http://env-test:3000" {
		t.Errorf("Upstream.URL = %s, want http://env-test:3000", cfg.Upstream.URL)
	}
}

func TestLoad_RemoteAuth(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

auth:
  mode: "remote"
  remote:
    url: "https://auth.example.com/api"
    api_key: "secret123"
    timeout: 5s
`

	cfg := writeAndLoad(t, content)

	if cfg.Auth.Mode != "remote" {
		t.Errorf("Auth.Mode = %s, want remote", cfg.Auth.Mode)
	}
	if cfg.Auth.Remote.URL != "https://auth.example.com/api" {
		t.Errorf("Auth.Remote.URL = %s, want https://auth.example.com/api", cfg.Auth.Remote.URL)
	}
	if cfg.Auth.Remote.APIKey != "secret123" {
		t.Errorf("Auth.Remote.APIKey = %s, want secret123", cfg.Auth.Remote.APIKey)
	}
}

func TestLoad_MissingUpstream(t *testing.T) {
	content := `
server:
  port: 8080
`

	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for missing upstream.url")
	}
}

func TestLoad_InvalidAuthMode(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

auth:
  mode: "invalid"
`

	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for invalid auth.mode")
	}
}

func TestLoad_RemoteAuthMissingURL(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

auth:
  mode: "remote"
`

	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for remote auth without URL")
	}
}

func TestLoad_MultipleEndpoints(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

endpoints:
  - path: "/api/expensive"
    method: "POST"
    cost_multiplier: 10.0
  - path: "/api/cheap"
    method: "GET"
    cost_multiplier: 0.5
`

	cfg := writeAndLoad(t, content)

	if len(cfg.Endpoints) != 2 {
		t.Fatalf("len(Endpoints) = %d, want 2", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0].CostMultiplier != 10.0 {
		t.Errorf("Endpoints[0].CostMultiplier = %f, want 10.0", cfg.Endpoints[0].CostMultiplier)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set env vars
	os.Setenv("APIGATE_UPSTREAM_URL", "http://env-upstream:8000")
	os.Setenv("APIGATE_SERVER_PORT", "9999")
	os.Setenv("APIGATE_DATABASE_DSN", "/tmp/env-test.db")
	os.Setenv("APIGATE_LOG_LEVEL", "debug")
	os.Setenv("APIGATE_METRICS_ENABLED", "true")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_SERVER_PORT")
		os.Unsetenv("APIGATE_DATABASE_DSN")
		os.Unsetenv("APIGATE_LOG_LEVEL")
		os.Unsetenv("APIGATE_METRICS_ENABLED")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Upstream.URL != "http://env-upstream:8000" {
		t.Errorf("Upstream.URL = %s, want http://env-upstream:8000", cfg.Upstream.URL)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Database.DSN != "/tmp/env-test.db" {
		t.Errorf("Database.DSN = %s, want /tmp/env-test.db", cfg.Database.DSN)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %s, want debug", cfg.Logging.Level)
	}
	if !cfg.Metrics.Enabled {
		t.Error("Metrics.Enabled = false, want true")
	}
}

func TestLoadFromEnv_MissingRequired(t *testing.T) {
	// Ensure APIGATE_UPSTREAM_URL is not set
	os.Unsetenv("APIGATE_UPSTREAM_URL")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for missing upstream URL")
	}
}

func TestEnvOverridesFile(t *testing.T) {
	// Set env var that should override file config
	os.Setenv("APIGATE_SERVER_PORT", "7777")
	os.Setenv("APIGATE_LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("APIGATE_SERVER_PORT")
		os.Unsetenv("APIGATE_LOG_LEVEL")
	}()

	content := `
upstream:
  url: "http://localhost:3000"
server:
  port: 8080
logging:
  level: "info"
`

	cfg := writeAndLoad(t, content)

	// Env should override file
	if cfg.Server.Port != 7777 {
		t.Errorf("Server.Port = %d, want 7777 (env override)", cfg.Server.Port)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Logging.Level = %s, want error (env override)", cfg.Logging.Level)
	}
	// File value should still be used for non-overridden
	if cfg.Upstream.URL != "http://localhost:3000" {
		t.Errorf("Upstream.URL = %s, want http://localhost:3000", cfg.Upstream.URL)
	}
}

func TestLoadWithFallback_FileExists(t *testing.T) {
	content := `
upstream:
  url: "http://file-config:3000"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadWithFallback(path)
	if err != nil {
		t.Fatalf("LoadWithFallback error: %v", err)
	}

	if cfg.Upstream.URL != "http://file-config:3000" {
		t.Errorf("Upstream.URL = %s, want http://file-config:3000", cfg.Upstream.URL)
	}
}

func TestLoadWithFallback_EnvOnly(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://env-fallback:8000")
	defer os.Unsetenv("APIGATE_UPSTREAM_URL")

	cfg, err := config.LoadWithFallback("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("LoadWithFallback error: %v", err)
	}

	if cfg.Upstream.URL != "http://env-fallback:8000" {
		t.Errorf("Upstream.URL = %s, want http://env-fallback:8000", cfg.Upstream.URL)
	}
}

func TestLoadWithFallback_NoConfig(t *testing.T) {
	os.Unsetenv("APIGATE_UPSTREAM_URL")

	_, err := config.LoadWithFallback("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error when no config available")
	}
}

func TestHasEnvConfig(t *testing.T) {
	os.Unsetenv("APIGATE_UPSTREAM_URL")
	if config.HasEnvConfig() {
		t.Error("HasEnvConfig() = true, want false")
	}

	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	defer os.Unsetenv("APIGATE_UPSTREAM_URL")
	if !config.HasEnvConfig() {
		t.Error("HasEnvConfig() = false, want true")
	}
}

func TestParseBoolValues(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
		os.Setenv("APIGATE_METRICS_ENABLED", tt.value)

		cfg, err := config.LoadFromEnv()
		if err != nil {
			t.Fatalf("LoadFromEnv error: %v", err)
		}

		if cfg.Metrics.Enabled != tt.expected {
			t.Errorf("value=%q: Metrics.Enabled = %v, want %v", tt.value, cfg.Metrics.Enabled, tt.expected)
		}

		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_METRICS_ENABLED")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"
  this is not valid yaml: [
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_InvalidUsageMode(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

usage:
  mode: "invalid"
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for invalid usage.mode")
	}
}

func TestLoad_RemoteUsageMissingURL(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

usage:
  mode: "remote"
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for remote usage without URL")
	}
}

func TestLoad_InvalidBillingMode(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

billing:
  mode: "invalid"
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for invalid billing.mode")
	}
}

func TestLoad_RemoteBillingMissingURL(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

billing:
  mode: "remote"
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for remote billing without URL")
	}
}

func TestLoad_PlanMissingID(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

plans:
  - name: "No ID Plan"
    rate_limit_per_minute: 100
`
	_, err := writeAndLoadErr(t, content)
	if err == nil {
		t.Fatal("expected error for plan without ID")
	}
}

func TestLoad_ValidBillingModes(t *testing.T) {
	modes := []string{"none", "stripe", "paddle", "lemonsqueezy"}
	for _, mode := range modes {
		content := `
upstream:
  url: "http://localhost:3000"

billing:
  mode: "` + mode + `"
`
		cfg, err := writeAndLoadErr(t, content)
		if err != nil {
			t.Errorf("billing mode %q should be valid, got error: %v", mode, err)
			continue
		}
		if cfg.Billing.Mode != mode {
			t.Errorf("Billing.Mode = %s, want %s", cfg.Billing.Mode, mode)
		}
	}
}

func TestLoad_RemoteBillingWithURL(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

billing:
  mode: "remote"
  remote:
    url: "https://billing.example.com/api"
`
	cfg, err := writeAndLoadErr(t, content)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Billing.Mode != "remote" {
		t.Errorf("Billing.Mode = %s, want remote", cfg.Billing.Mode)
	}
	if cfg.Billing.Remote.URL != "https://billing.example.com/api" {
		t.Errorf("Billing.Remote.URL = %s, want https://billing.example.com/api", cfg.Billing.Remote.URL)
	}
}

func TestLoad_RemoteUsageWithURL(t *testing.T) {
	content := `
upstream:
  url: "http://localhost:3000"

usage:
  mode: "remote"
  remote:
    url: "https://usage.example.com/api"
`
	cfg, err := writeAndLoadErr(t, content)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Usage.Mode != "remote" {
		t.Errorf("Usage.Mode = %s, want remote", cfg.Usage.Mode)
	}
	if cfg.Usage.Remote.URL != "https://usage.example.com/api" {
		t.Errorf("Usage.Remote.URL = %s, want https://usage.example.com/api", cfg.Usage.Remote.URL)
	}
}

func TestEnvOverrides_AllServerSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_SERVER_HOST", "192.168.1.1")
	os.Setenv("APIGATE_SERVER_PORT", "3000")
	os.Setenv("APIGATE_SERVER_READ_TIMEOUT", "45s")
	os.Setenv("APIGATE_SERVER_WRITE_TIMEOUT", "90s")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_SERVER_HOST")
		os.Unsetenv("APIGATE_SERVER_PORT")
		os.Unsetenv("APIGATE_SERVER_READ_TIMEOUT")
		os.Unsetenv("APIGATE_SERVER_WRITE_TIMEOUT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Server.Host = %s, want 192.168.1.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 45s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 90*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want 90s", cfg.Server.WriteTimeout)
	}
}

func TestEnvOverrides_UpstreamTimeout(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_UPSTREAM_TIMEOUT", "120s")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_UPSTREAM_TIMEOUT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Upstream.Timeout != 120*time.Second {
		t.Errorf("Upstream.Timeout = %v, want 120s", cfg.Upstream.Timeout)
	}
}

func TestEnvOverrides_AuthSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_AUTH_MODE", "remote")
	os.Setenv("APIGATE_AUTH_KEY_PREFIX", "custom_")
	os.Setenv("APIGATE_AUTH_REMOTE_URL", "https://auth.example.com/validate")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_AUTH_MODE")
		os.Unsetenv("APIGATE_AUTH_KEY_PREFIX")
		os.Unsetenv("APIGATE_AUTH_REMOTE_URL")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Auth.Mode != "remote" {
		t.Errorf("Auth.Mode = %s, want remote", cfg.Auth.Mode)
	}
	if cfg.Auth.KeyPrefix != "custom_" {
		t.Errorf("Auth.KeyPrefix = %s, want custom_", cfg.Auth.KeyPrefix)
	}
	if cfg.Auth.Remote.URL != "https://auth.example.com/validate" {
		t.Errorf("Auth.Remote.URL = %s, want https://auth.example.com/validate", cfg.Auth.Remote.URL)
	}
}

func TestEnvOverrides_RateLimitSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_RATELIMIT_ENABLED", "false")
	os.Setenv("APIGATE_RATELIMIT_BURST", "100")
	os.Setenv("APIGATE_RATELIMIT_WINDOW", "120")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_RATELIMIT_ENABLED")
		os.Unsetenv("APIGATE_RATELIMIT_BURST")
		os.Unsetenv("APIGATE_RATELIMIT_WINDOW")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.RateLimit.Enabled != false {
		t.Errorf("RateLimit.Enabled = %v, want false", cfg.RateLimit.Enabled)
	}
	if cfg.RateLimit.BurstTokens != 100 {
		t.Errorf("RateLimit.BurstTokens = %d, want 100", cfg.RateLimit.BurstTokens)
	}
	if cfg.RateLimit.WindowSecs != 120 {
		t.Errorf("RateLimit.WindowSecs = %d, want 120", cfg.RateLimit.WindowSecs)
	}
}

func TestEnvOverrides_UsageSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_USAGE_MODE", "remote")
	os.Setenv("APIGATE_USAGE_REMOTE_URL", "https://usage.example.com/api")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_USAGE_MODE")
		os.Unsetenv("APIGATE_USAGE_REMOTE_URL")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Usage.Mode != "remote" {
		t.Errorf("Usage.Mode = %s, want remote", cfg.Usage.Mode)
	}
	if cfg.Usage.Remote.URL != "https://usage.example.com/api" {
		t.Errorf("Usage.Remote.URL = %s, want https://usage.example.com/api", cfg.Usage.Remote.URL)
	}
}

func TestEnvOverrides_BillingSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_BILLING_MODE", "stripe")
	os.Setenv("APIGATE_BILLING_STRIPE_KEY", "sk_test_12345")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_BILLING_MODE")
		os.Unsetenv("APIGATE_BILLING_STRIPE_KEY")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Billing.Mode != "stripe" {
		t.Errorf("Billing.Mode = %s, want stripe", cfg.Billing.Mode)
	}
	if cfg.Billing.StripeKey != "sk_test_12345" {
		t.Errorf("Billing.StripeKey = %s, want sk_test_12345", cfg.Billing.StripeKey)
	}
}

func TestEnvOverrides_DatabaseSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_DATABASE_DRIVER", "postgres")
	os.Setenv("APIGATE_DATABASE_DSN", "postgres://user:pass@localhost/db")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_DATABASE_DRIVER")
		os.Unsetenv("APIGATE_DATABASE_DSN")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver = %s, want postgres", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "postgres://user:pass@localhost/db" {
		t.Errorf("Database.DSN = %s, want postgres://user:pass@localhost/db", cfg.Database.DSN)
	}
}

func TestEnvOverrides_LoggingSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_LOG_LEVEL", "warn")
	os.Setenv("APIGATE_LOG_FORMAT", "console")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_LOG_LEVEL")
		os.Unsetenv("APIGATE_LOG_FORMAT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %s, want warn", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "console" {
		t.Errorf("Logging.Format = %s, want console", cfg.Logging.Format)
	}
}

func TestEnvOverrides_MetricsPath(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_METRICS_ENABLED", "true")
	os.Setenv("APIGATE_METRICS_PATH", "/custom-metrics")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_METRICS_ENABLED")
		os.Unsetenv("APIGATE_METRICS_PATH")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if !cfg.Metrics.Enabled {
		t.Error("Metrics.Enabled = false, want true")
	}
	if cfg.Metrics.Path != "/custom-metrics" {
		t.Errorf("Metrics.Path = %s, want /custom-metrics", cfg.Metrics.Path)
	}
}

func TestEnvOverrides_OpenAPISettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_OPENAPI_ENABLED", "true")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_OPENAPI_ENABLED")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if !cfg.OpenAPI.Enabled {
		t.Error("OpenAPI.Enabled = false, want true")
	}
}

func TestEnvOverrides_PortalSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_PORTAL_ENABLED", "true")
	os.Setenv("APIGATE_PORTAL_BASE_URL", "https://api.example.com")
	os.Setenv("APIGATE_PORTAL_APP_NAME", "My API Gateway")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_PORTAL_ENABLED")
		os.Unsetenv("APIGATE_PORTAL_BASE_URL")
		os.Unsetenv("APIGATE_PORTAL_APP_NAME")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if !cfg.Portal.Enabled {
		t.Error("Portal.Enabled = false, want true")
	}
	if cfg.Portal.BaseURL != "https://api.example.com" {
		t.Errorf("Portal.BaseURL = %s, want https://api.example.com", cfg.Portal.BaseURL)
	}
	if cfg.Portal.AppName != "My API Gateway" {
		t.Errorf("Portal.AppName = %s, want My API Gateway", cfg.Portal.AppName)
	}
}

func TestEnvOverrides_SMTPSettings(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_EMAIL_PROVIDER", "smtp")
	os.Setenv("APIGATE_SMTP_HOST", "smtp.example.com")
	os.Setenv("APIGATE_SMTP_PORT", "465")
	os.Setenv("APIGATE_SMTP_USERNAME", "user@example.com")
	os.Setenv("APIGATE_SMTP_PASSWORD", "secret123")
	os.Setenv("APIGATE_SMTP_FROM", "noreply@example.com")
	os.Setenv("APIGATE_SMTP_FROM_NAME", "API Gateway")
	os.Setenv("APIGATE_SMTP_USE_TLS", "true")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_EMAIL_PROVIDER")
		os.Unsetenv("APIGATE_SMTP_HOST")
		os.Unsetenv("APIGATE_SMTP_PORT")
		os.Unsetenv("APIGATE_SMTP_USERNAME")
		os.Unsetenv("APIGATE_SMTP_PASSWORD")
		os.Unsetenv("APIGATE_SMTP_FROM")
		os.Unsetenv("APIGATE_SMTP_FROM_NAME")
		os.Unsetenv("APIGATE_SMTP_USE_TLS")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.Email.Provider != "smtp" {
		t.Errorf("Email.Provider = %s, want smtp", cfg.Email.Provider)
	}
	if cfg.Email.SMTP.Host != "smtp.example.com" {
		t.Errorf("Email.SMTP.Host = %s, want smtp.example.com", cfg.Email.SMTP.Host)
	}
	if cfg.Email.SMTP.Port != 465 {
		t.Errorf("Email.SMTP.Port = %d, want 465", cfg.Email.SMTP.Port)
	}
	if cfg.Email.SMTP.Username != "user@example.com" {
		t.Errorf("Email.SMTP.Username = %s, want user@example.com", cfg.Email.SMTP.Username)
	}
	if cfg.Email.SMTP.Password != "secret123" {
		t.Errorf("Email.SMTP.Password = %s, want secret123", cfg.Email.SMTP.Password)
	}
	if cfg.Email.SMTP.From != "noreply@example.com" {
		t.Errorf("Email.SMTP.From = %s, want noreply@example.com", cfg.Email.SMTP.From)
	}
	if cfg.Email.SMTP.FromName != "API Gateway" {
		t.Errorf("Email.SMTP.FromName = %s, want API Gateway", cfg.Email.SMTP.FromName)
	}
	if !cfg.Email.SMTP.UseTLS {
		t.Error("Email.SMTP.UseTLS = false, want true")
	}
}

func TestEnvOverrides_InvalidPort(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_SERVER_PORT", "not-a-number")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_SERVER_PORT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	// Should use default port when env var is invalid
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080 (default)", cfg.Server.Port)
	}
}

func TestEnvOverrides_InvalidDuration(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_SERVER_READ_TIMEOUT", "not-a-duration")
	os.Setenv("APIGATE_UPSTREAM_TIMEOUT", "bad-value")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_SERVER_READ_TIMEOUT")
		os.Unsetenv("APIGATE_UPSTREAM_TIMEOUT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	// Should use default when env var is invalid
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 30s (default)", cfg.Server.ReadTimeout)
	}
}

func TestEnvOverrides_InvalidIntegers(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://test:8000")
	os.Setenv("APIGATE_RATELIMIT_BURST", "not-a-number")
	os.Setenv("APIGATE_RATELIMIT_WINDOW", "invalid")
	os.Setenv("APIGATE_SMTP_PORT", "bad-port")
	defer func() {
		os.Unsetenv("APIGATE_UPSTREAM_URL")
		os.Unsetenv("APIGATE_RATELIMIT_BURST")
		os.Unsetenv("APIGATE_RATELIMIT_WINDOW")
		os.Unsetenv("APIGATE_SMTP_PORT")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	// Should use defaults when env vars are invalid
	if cfg.RateLimit.BurstTokens != 5 {
		t.Errorf("RateLimit.BurstTokens = %d, want 5 (default)", cfg.RateLimit.BurstTokens)
	}
	if cfg.RateLimit.WindowSecs != 60 {
		t.Errorf("RateLimit.WindowSecs = %d, want 60 (default)", cfg.RateLimit.WindowSecs)
	}
	if cfg.Email.SMTP.Port != 587 {
		t.Errorf("Email.SMTP.Port = %d, want 587 (default)", cfg.Email.SMTP.Port)
	}
}

func TestLoadWithFallback_EmptyPath(t *testing.T) {
	os.Setenv("APIGATE_UPSTREAM_URL", "http://env-fallback:8000")
	defer os.Unsetenv("APIGATE_UPSTREAM_URL")

	cfg, err := config.LoadWithFallback("")
	if err != nil {
		t.Fatalf("LoadWithFallback error: %v", err)
	}

	if cfg.Upstream.URL != "http://env-fallback:8000" {
		t.Errorf("Upstream.URL = %s, want http://env-fallback:8000", cfg.Upstream.URL)
	}
}

func TestLoad_AllConfigFields(t *testing.T) {
	content := `
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 60s

upstream:
  url: "http://localhost:3000"
  timeout: 30s
  max_idle_conns: 100
  idle_conn_timeout: 90s

auth:
  mode: "local"
  key_prefix: "ak_"
  header: "X-API-Key"
  jwt_secret: "super-secret"

rate_limit:
  enabled: true
  burst_tokens: 10
  window_secs: 60

usage:
  mode: "local"
  batch_size: 50
  flush_interval: 5s

billing:
  mode: "stripe"
  stripe_key: "sk_test_xxx"

database:
  driver: "sqlite"
  dsn: ":memory:"

plans:
  - id: "free"
    name: "Free"
    rate_limit_per_minute: 60
    requests_per_month: 1000
    price_monthly: 0
    overage_price: 0
  - id: "pro"
    name: "Pro"
    rate_limit_per_minute: 600
    requests_per_month: 100000
    price_monthly: 2999
    overage_price: 1

endpoints:
  - path: "/api/heavy"
    method: "POST"
    cost_multiplier: 5.0

logging:
  level: "debug"
  format: "console"

metrics:
  enabled: true
  path: "/metrics"

openapi:
  enabled: true

portal:
  enabled: true
  base_url: "https://api.example.com"
  app_name: "Test Gateway"

email:
  provider: "smtp"
  smtp:
    host: "smtp.example.com"
    port: 587
    username: "user"
    password: "pass"
    from: "noreply@example.com"
    from_name: "Test"
    use_tls: true
    use_implicit: false
    skip_verify: false
    timeout: 30s
`

	cfg := writeAndLoad(t, content)

	// Verify all fields are properly loaded
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Upstream.MaxIdleConns != 100 {
		t.Errorf("Upstream.MaxIdleConns = %d, want 100", cfg.Upstream.MaxIdleConns)
	}
	if cfg.Upstream.IdleConnTimeout != 90*time.Second {
		t.Errorf("Upstream.IdleConnTimeout = %v, want 90s", cfg.Upstream.IdleConnTimeout)
	}
	if cfg.Auth.Header != "X-API-Key" {
		t.Errorf("Auth.Header = %s, want X-API-Key", cfg.Auth.Header)
	}
	if cfg.Auth.JWTSecret != "super-secret" {
		t.Errorf("Auth.JWTSecret = %s, want super-secret", cfg.Auth.JWTSecret)
	}
	if cfg.RateLimit.Enabled != true {
		t.Error("RateLimit.Enabled = false, want true")
	}
	if cfg.Usage.BatchSize != 50 {
		t.Errorf("Usage.BatchSize = %d, want 50", cfg.Usage.BatchSize)
	}
	if cfg.Usage.FlushInterval != 5*time.Second {
		t.Errorf("Usage.FlushInterval = %v, want 5s", cfg.Usage.FlushInterval)
	}
	if len(cfg.Plans) != 2 {
		t.Errorf("len(Plans) = %d, want 2", len(cfg.Plans))
	}
	if cfg.Plans[1].PriceMonthly != 2999 {
		t.Errorf("Plans[1].PriceMonthly = %d, want 2999", cfg.Plans[1].PriceMonthly)
	}
	if cfg.Plans[1].OveragePrice != 1 {
		t.Errorf("Plans[1].OveragePrice = %d, want 1", cfg.Plans[1].OveragePrice)
	}
	if cfg.Metrics.Path != "/metrics" {
		t.Errorf("Metrics.Path = %s, want /metrics", cfg.Metrics.Path)
	}
	if cfg.Email.SMTP.Timeout != 30*time.Second {
		t.Errorf("Email.SMTP.Timeout = %v, want 30s", cfg.Email.SMTP.Timeout)
	}
}

// Helpers

func writeAndLoad(t *testing.T, content string) *config.Config {
	t.Helper()
	cfg, err := writeAndLoadErr(t, content)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	return cfg
}

func writeAndLoadErr(t *testing.T, content string) (*config.Config, error) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return config.Load(path)
}
