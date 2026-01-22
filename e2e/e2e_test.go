// Package e2e provides end-to-end tests for the complete APIGate proxy flow.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
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
			// JSON:API error format: {"errors": [{"code": "..."}]}
			errors, ok := body["errors"].([]interface{})
			if !ok || len(errors) == 0 {
				t.Fatal("expected errors array in response")
			}
			errObj, _ := errors[0].(map[string]interface{})
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
	// JSON:API error format: {"errors": [{"code": "..."}]}
	errors, ok := body["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("expected errors array in response")
	}
	errObj, _ := errors[0].(map[string]interface{})
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

// TestE2E_PriorityRouting tests that database routes with priority > 0 override built-in routes.
// This addresses issue #39: Custom routes with path /* should override built-in admin routes.
func TestE2E_PriorityRouting(t *testing.T) {
	// 1. Create custom upstream that serves a simple response
	customUpstreamCalled := false
	customUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customUpstreamCalled = true
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("<html><body>Custom Frontend</body></html>"))
	}))
	defer customUpstream.Close()

	// 2. Setup test app
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()

	// Insert basic settings
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.burst_tokens", "2")
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.window_secs", "60")

	// 3. Create custom upstream in database
	_, err = db.DB.ExecContext(ctx,
		"INSERT INTO upstreams (id, name, base_url, timeout_ms, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))",
		"upstream-custom", "Custom Frontend", customUpstream.URL, 30000, 1)
	if err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	// 4. Create route with priority > 0 for root path
	// This route should override the built-in admin route at /
	_, err = db.DB.ExecContext(ctx,
		"INSERT INTO routes (id, name, path_pattern, match_type, upstream_id, priority, auth_required, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))",
		"route-custom-root", "Custom Root", "/", "exact", "upstream-custom", 10, 0, 1)
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	db.Close()

	// 5. Bootstrap app
	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogLevel, "debug")
	os.Setenv(bootstrap.EnvLogFormat, "json")

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	defer app.Shutdown()
	defer os.Unsetenv(bootstrap.EnvDatabaseDSN)
	defer os.Unsetenv(bootstrap.EnvLogLevel)
	defer os.Unsetenv(bootstrap.EnvLogFormat)

	// 6. Start server
	serverAddr := startServer(t, app)

	// 7. Test that / routes to custom upstream (not admin interface)
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects - if we get redirected to /login, the test should fail
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("http://" + serverAddr + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// 8. Verify response
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		t.Fatalf("got redirect to %s, expected custom route to be served", location)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !contains(bodyStr, "Custom Frontend") {
		t.Errorf("body = %s, want to contain 'Custom Frontend'", bodyStr)
	}

	if !customUpstreamCalled {
		t.Error("custom upstream was not called - priority route did not override built-in route")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper functions

func setupTestApp(t *testing.T, upstreamURL string) (*bootstrap.App, string, func()) {
	return setupTestAppWithPlan(t, upstreamURL, 60) // 60 requests/minute default
}

func setupTestAppWithPlan(t *testing.T, upstreamURL string, rateLimit int) (*bootstrap.App, string, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Pre-create database and insert settings BEFORE bootstrap
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	// Insert settings for upstream and rate limit
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "upstream.url", upstreamURL)
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.burst_tokens", "2")
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.window_secs", "60")

	// Create test plan in database
	_, err = db.DB.ExecContext(ctx,
		"INSERT OR REPLACE INTO plans (id, name, rate_limit_per_minute, requests_per_month, enabled) VALUES (?, ?, ?, ?, ?)",
		"test", "Test Plan", rateLimit, 10000, 1)
	if err != nil {
		t.Fatalf("create test plan: %v", err)
	}

	db.Close()

	// Set environment variables for bootstrap
	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogLevel, "error")
	os.Setenv(bootstrap.EnvLogFormat, "json")

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Create test user and API key
	apiKey := createTestUserAndKey(t, app.DB, "test")

	cleanup := func() {
		app.Shutdown()
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvLogLevel)
		os.Unsetenv(bootstrap.EnvLogFormat)
	}

	return app, apiKey, cleanup
}

func setupTestAppWithExpiredKey(t *testing.T, upstreamURL string) (*bootstrap.App, string, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Pre-create database and insert settings BEFORE bootstrap
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "upstream.url", upstreamURL)
	db.Close()

	// Set environment variables for bootstrap
	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogLevel, "error")
	os.Setenv(bootstrap.EnvLogFormat, "json")

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Create expired key
	apiKey := createExpiredKey(t, app.DB)

	cleanup := func() {
		app.Shutdown()
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvLogLevel)
		os.Unsetenv(bootstrap.EnvLogFormat)
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
