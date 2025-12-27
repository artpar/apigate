package config_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/config"
	"github.com/rs/zerolog"
)

func TestHolder_Get(t *testing.T) {
	cfg := writeConfig(t, validConfig())

	h, err := config.NewHolder(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	got := h.Get()
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Upstream.URL != "http://localhost:3000" {
		t.Errorf("Upstream.URL = %s, want http://localhost:3000", got.Upstream.URL)
	}
}

func TestHolder_Reload(t *testing.T) {
	path := writeConfig(t, validConfig())

	h, err := config.NewHolder(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	// Verify initial config
	cfg := h.Get()
	if cfg.Plans[0].RateLimitPerMinute != 100 {
		t.Errorf("initial RateLimitPerMinute = %d, want 100", cfg.Plans[0].RateLimitPerMinute)
	}

	// Write new config
	newContent := `
upstream:
  url: "http://localhost:3000"

plans:
  - id: "free"
    name: "Free Plan"
    rate_limit_per_minute: 200
    requests_per_month: 20000
`
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("write new config: %v", err)
	}

	// Reload
	if err := h.Reload(); err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify new config
	cfg = h.Get()
	if cfg.Plans[0].RateLimitPerMinute != 200 {
		t.Errorf("reloaded RateLimitPerMinute = %d, want 200", cfg.Plans[0].RateLimitPerMinute)
	}
}

func TestHolder_OnChange(t *testing.T) {
	path := writeConfig(t, validConfig())

	h, err := config.NewHolder(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	var mu sync.Mutex
	var called bool
	var receivedCfg *config.Config

	h.OnChange(func(cfg *config.Config) {
		mu.Lock()
		called = true
		receivedCfg = cfg
		mu.Unlock()
	})

	// Write new config and reload
	newContent := `
upstream:
  url: "http://localhost:4000"
`
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("write new config: %v", err)
	}

	if err := h.Reload(); err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	mu.Lock()
	if !called {
		t.Error("OnChange callback was not called")
	}
	if receivedCfg == nil {
		t.Error("received nil config in callback")
	} else if receivedCfg.Upstream.URL != "http://localhost:4000" {
		t.Errorf("callback received URL = %s, want http://localhost:4000", receivedCfg.Upstream.URL)
	}
	mu.Unlock()
}

func TestHolder_ReloadInvalidConfig(t *testing.T) {
	path := writeConfig(t, validConfig())

	h, err := config.NewHolder(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	// Write invalid config
	invalidContent := `
server:
  port: 8080
# Missing required upstream.url
`
	if err := os.WriteFile(path, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	// Reload should fail
	err = h.Reload()
	if err == nil {
		t.Error("Reload should fail for invalid config")
	}

	// Old config should still be valid
	cfg := h.Get()
	if cfg.Upstream.URL != "http://localhost:3000" {
		t.Errorf("should keep old config, got Upstream.URL = %s", cfg.Upstream.URL)
	}
}

func TestHolder_WatchFile(t *testing.T) {
	path := writeConfig(t, validConfig())

	h, err := config.NewHolder(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	var mu sync.Mutex
	var callCount int

	h.OnChange(func(cfg *config.Config) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	if err := h.WatchFile(); err != nil {
		t.Fatalf("WatchFile error: %v", err)
	}

	// Write new config
	newContent := `
upstream:
  url: "http://localhost:5000"
`
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("write new config: %v", err)
	}

	// Wait for file watcher to trigger
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount == 0 {
		t.Error("file watcher did not trigger reload")
	}
	mu.Unlock()

	// Verify config was updated
	cfg := h.Get()
	if cfg.Upstream.URL != "http://localhost:5000" {
		t.Errorf("after file watch, Upstream.URL = %s, want http://localhost:5000", cfg.Upstream.URL)
	}
}

func TestHolder_ConcurrentAccess(t *testing.T) {
	path := writeConfig(t, validConfig())

	h, err := config.NewHolder(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewHolder error: %v", err)
	}
	defer h.Stop()

	// Start many readers
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cfg := h.Get()
				if cfg == nil {
					t.Error("concurrent Get returned nil")
				}
			}
		}()
	}

	// Concurrent reloads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.Reload()
		}()
	}

	wg.Wait()
}

func TestReloadableFields(t *testing.T) {
	fields := config.ReloadableFields()
	if len(fields) == 0 {
		t.Error("ReloadableFields returned empty")
	}

	// Check expected fields
	expected := []string{"plans", "endpoints", "rate_limit.burst_tokens"}
	for _, e := range expected {
		found := false
		for _, f := range fields {
			if f == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not in ReloadableFields", e)
		}
	}
}

func TestNonReloadableFields(t *testing.T) {
	fields := config.NonReloadableFields()
	if len(fields) == 0 {
		t.Error("NonReloadableFields returned empty")
	}

	// Check expected fields
	expected := []string{"server.host", "server.port", "upstream.url", "database.dsn"}
	for _, e := range expected {
		found := false
		for _, f := range fields {
			if f == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not in NonReloadableFields", e)
		}
	}
}

// Helpers

func validConfig() string {
	return `
upstream:
  url: "http://localhost:3000"

plans:
  - id: "free"
    name: "Free Plan"
    rate_limit_per_minute: 100
    requests_per_month: 10000
`
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
