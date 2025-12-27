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
	"github.com/artpar/apigate/config"
)

func TestBootstrap_Integration(t *testing.T) {
	// Create mock upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message": "hello from upstream"}`))
	}))
	defer upstream.Close()

	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "test.db")

	configContent := `
upstream:
  url: "` + upstream.URL + `"
  timeout: 5s

database:
  driver: sqlite
  dsn: "` + dbPath + `"

server:
  host: "127.0.0.1"
  port: 0

auth:
  mode: local
  key_prefix: "ak_"

logging:
  level: debug
  format: console
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// Create app
	app, err := bootstrap.New(cfg)
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
	if app.Config == nil {
		t.Error("Config should not be nil")
	}
}

func TestBootstrap_DatabaseMigration(t *testing.T) {
	// Create temp config with in-memory DB
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "migrate-test.db")

	configContent := `
upstream:
  url: "http://localhost:9999"

database:
  driver: sqlite
  dsn: "` + dbPath + `"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()

	// Verify tables exist by querying
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err = app.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Errorf("query users table: %v", err)
	}

	err = app.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys").Scan(&count)
	if err != nil {
		t.Errorf("query api_keys table: %v", err)
	}

	err = app.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_events").Scan(&count)
	if err != nil {
		t.Errorf("query usage_events table: %v", err)
	}
}

func TestBootstrap_GracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "shutdown-test.db")

	configContent := `
upstream:
  url: "http://localhost:9999"

database:
  driver: sqlite
  dsn: "` + dbPath + `"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Shutdown should complete without error
	err = app.Shutdown()
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Verify DB is closed (should error on query)
	_, err = app.DB.Query("SELECT 1")
	if err == nil {
		t.Error("expected error querying closed database")
	}
}
