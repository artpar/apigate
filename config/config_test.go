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
