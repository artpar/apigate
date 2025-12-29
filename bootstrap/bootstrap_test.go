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
