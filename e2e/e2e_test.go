// Package e2e provides end-to-end tests for the complete APIGate proxy flow.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/bcrypt"
)

// TestE2E_FullProxyFlow tests the complete proxy flow:
// 1. Start upstream mock server
// 2. Start APIGate proxy
// 3. Create user and API key
// 4. Make authenticated request
// 5. Verify response and usage tracking
func TestE2E_FullProxyFlow(t *testing.T) {
	// 1. Create mock upstream
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream-Header", "test-value")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "hello from upstream",
			"path":    r.URL.Path,
			"method":  r.Method,
		})
	}))
	defer upstream.Close()

	// 2. Start APIGate
	app, apiKey, cleanup := setupTestApp(t, upstream.URL)
	defer cleanup()

	// Start server in background
	serverAddr := startServer(t, app)

	// 3. Make authenticated request
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://"+serverAddr+"/api/data", nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// 4. Verify response
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	// Check response body
	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if respBody["message"] != "hello from upstream" {
		t.Errorf("message = %v, want 'hello from upstream'", respBody["message"])
	}

	// Check rate limit headers added by proxy
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}

	// Check upstream was called
	if upstreamCalls != 1 {
		t.Errorf("upstream calls = %d, want 1", upstreamCalls)
	}
}

// TestE2E_InvalidAPIKey tests rejection of invalid API keys.
func TestE2E_InvalidAPIKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called for invalid key")
	}))
	defer upstream.Close()

	app, _, cleanup := setupTestApp(t, upstream.URL)
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name   string
		apiKey string
		code   string
	}{
		{"missing key", "", "missing_api_key"},
		{"wrong prefix", "sk_wrong1234567890123456789012345678901234567890123456789012345678", "invalid_api_key"},
		{"too short", "ak_short", "invalid_api_key"},
		{"nonexistent", "ak_nonexistent123456789012345678901234567890123456789012345678901", "invalid_api_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+serverAddr+"/api/data", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 401 {
				t.Errorf("status = %d, want 401", resp.StatusCode)
			}

			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)
			errObj, ok := body["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected error object")
			}
			if errObj["code"] != tt.code {
				t.Errorf("code = %v, want %s", errObj["code"], tt.code)
			}
		})
	}
}

// TestE2E_RateLimiting tests rate limit enforcement.
func TestE2E_RateLimiting(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	app, apiKey, cleanup := setupTestAppWithPlan(t, upstream.URL, 2) // 2 requests/minute
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}

	// Make requests until rate limited
	var rateLimited bool
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "http://"+serverAddr+"/api/data", nil)
		req.Header.Set("X-API-Key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode == 429 {
			rateLimited = true
			// Check Retry-After header
			if resp.Header.Get("Retry-After") == "" {
				t.Error("missing Retry-After header on 429")
			}
			break
		}
	}

	if !rateLimited {
		t.Error("expected to be rate limited after exceeding limit")
	}
}

// TestE2E_ExpiredKey tests rejection of expired API keys.
func TestE2E_ExpiredKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called for expired key")
	}))
	defer upstream.Close()

	app, apiKey, cleanup := setupTestAppWithExpiredKey(t, upstream.URL)
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://"+serverAddr+"/api/data", nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 401, body: %s", resp.StatusCode, body)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	errObj := body["error"].(map[string]interface{})
	if errObj["code"] != "key_expired" {
		t.Errorf("code = %v, want key_expired", errObj["code"])
	}
}

// TestE2E_HealthEndpoints tests health check endpoints.
func TestE2E_HealthEndpoints(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	app, _, cleanup := setupTestApp(t, upstream.URL)
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		path   string
		status int
	}{
		{"/health", 200},
		{"/health/live", 200},
		{"/health/ready", 200},
		{"/version", 200},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := client.Get("http://" + serverAddr + tt.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.status)
			}
		})
	}
}

// TestE2E_POSTWithBody tests proxying POST requests with body.
func TestE2E_POSTWithBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(201)
		w.Write([]byte(`{"created":true}`))
	}))
	defer upstream.Close()

	app, apiKey, cleanup := setupTestApp(t, upstream.URL)
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}

	body := `{"name":"test","value":123}`
	req, _ := http.NewRequest("POST", "http://"+serverAddr+"/api/items", bytes.NewBufferString(body))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}

	if receivedContentType != "application/json" {
		t.Errorf("upstream content-type = %s, want application/json", receivedContentType)
	}

	if string(receivedBody) != body {
		t.Errorf("upstream body = %s, want %s", receivedBody, body)
	}
}

// TestE2E_BearerAuth tests Bearer token authentication.
func TestE2E_BearerAuth(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	app, apiKey, cleanup := setupTestApp(t, upstream.URL)
	defer cleanup()

	serverAddr := startServer(t, app)

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://"+serverAddr+"/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

// Helper functions

func setupTestApp(t *testing.T, upstreamURL string) (*bootstrap.App, string, func()) {
	return setupTestAppWithPlan(t, upstreamURL, 60) // 60 requests/minute default
}

func setupTestAppWithPlan(t *testing.T, upstreamURL string, rateLimit int) (*bootstrap.App, string, func()) {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "test.db")

	configContent := fmt.Sprintf(`
upstream:
  url: "%s"
  timeout: 5s

database:
  driver: sqlite
  dsn: "%s"

server:
  host: "127.0.0.1"
  port: 0

auth:
  mode: local
  key_prefix: "ak_"

rate_limit:
  enabled: true
  burst_tokens: 2
  window_secs: 60

plans:
  - id: "test"
    name: "Test Plan"
    rate_limit_per_minute: %d
    requests_per_month: 10000

logging:
  level: error
  format: json
`, upstreamURL, dbPath, rateLimit)

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

	// Create test user and API key
	apiKey := createTestUserAndKey(t, app.DB, "test")

	cleanup := func() {
		app.Shutdown()
	}

	return app, apiKey, cleanup
}

func setupTestAppWithExpiredKey(t *testing.T, upstreamURL string) (*bootstrap.App, string, func()) {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "test.db")

	configContent := fmt.Sprintf(`
upstream:
  url: "%s"
  timeout: 5s

database:
  driver: sqlite
  dsn: "%s"

server:
  host: "127.0.0.1"
  port: 0

auth:
  mode: local
  key_prefix: "ak_"

logging:
  level: error
  format: json
`, upstreamURL, dbPath)

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

	// Create expired key
	apiKey := createExpiredKey(t, app.DB)

	cleanup := func() {
		app.Shutdown()
	}

	return app, apiKey, cleanup
}

func createTestUserAndKey(t *testing.T, db *sqlite.DB, planID string) string {
	t.Helper()
	ctx := context.Background()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)

	// Create user
	user := ports.User{
		ID:     "test-user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		PlanID: planID,
		Status: "active",
	}
	if err := userStore.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create API key - must be 67 chars: 3 char prefix + 64 hex chars
	rawKey := "ak_e2etest123456789012345678901234567890123456789012345678901234567"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	k := key.Key{
		ID:        "test-key-1",
		UserID:    user.ID,
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		Name:      "Test Key",
		CreatedAt: time.Now().UTC(),
	}
	if err := keyStore.Create(ctx, k); err != nil {
		t.Fatalf("create key: %v", err)
	}

	return rawKey
}

func createExpiredKey(t *testing.T, db *sqlite.DB) string {
	t.Helper()
	ctx := context.Background()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)

	// Create user
	user := ports.User{
		ID:     "expired-user-1",
		Email:  "expired@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	// Create expired key - must be 67 chars: 3 char prefix + 64 hex chars
	rawKey := "ak_expired123456789012345678901234567890123456789012345678901234567"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	expiredAt := time.Now().Add(-24 * time.Hour)

	k := key.Key{
		ID:        "expired-key-1",
		UserID:    user.ID,
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		ExpiresAt: &expiredAt,
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	keyStore.Create(ctx, k)

	return rawKey
}

func startServer(t *testing.T, app *bootstrap.App) string {
	t.Helper()

	// Find free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	addr := listener.Addr().String()

	// Update server address
	app.HTTPServer.Addr = addr

	// Close the listener so server can use the port
	listener.Close()

	// Start server in goroutine
	go func() {
		if err := app.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log but don't fail - server might be shutting down
		}
	}()

	// Wait for server to be ready
	waitForServer(t, addr)

	return addr
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}

	for i := 0; i < 50; i++ {
		resp, err := client.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("server at %s did not become ready", addr)
}
