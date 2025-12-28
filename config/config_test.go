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
