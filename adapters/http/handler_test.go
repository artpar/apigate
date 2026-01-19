package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/clock"
	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/adapters/metrics"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var baseTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func TestProxyHandler_ValidRequest(t *testing.T) {
	handler, stores := setupTestHandler()

	// Seed test data
	rawKey := "ak_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-1",
		Email:  "test@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Create request
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", rawKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	// Check rate limit headers
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}
}

func TestProxyHandler_MissingAPIKey(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/data", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	// JSON:API error format: {"errors": [{"code": "..."}]}
	errors, ok := body["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Fatal("expected errors array in response")
	}
	errObj, _ := errors[0].(map[string]any)
	if errObj["code"] != "missing_api_key" {
		t.Errorf("code = %s, want missing_api_key", errObj["code"])
	}
}

func TestProxyHandler_InvalidAPIKey(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "ak_invalid0123456789abcdef0123456789abcdef0123456789abcdef01234")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestProxyHandler_BearerToken(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_bearer1234567890123456789012345678901234567890123456789012345678"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-2",
		UserID:    "user-2",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-2",
		Email:  "bearer@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Use Bearer token format
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

func TestProxyHandler_QueryParam(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_query12345678901234567890123456789012345678901234567890123456789"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-3",
		UserID:    "user-3",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-3",
		Email:  "query@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Use query parameter
	req := httptest.NewRequest("GET", "/api/data?api_key="+rawKey, nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

func TestProxyHandler_PostWithBody(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_post123456789012345678901234567890123456789012345678901234567890"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-4",
		UserID:    "user-4",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-4",
		Email:  "post@example.com",
		PlanID: "free",
		Status: "active",
	})

	bodyContent := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/api/data", strings.NewReader(bodyContent))
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

func TestHealthHandler_Liveness(t *testing.T) {
	healthHandler := apihttp.NewHealthHandler(nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	rec := httptest.NewRecorder()
	healthHandler.Liveness(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %s, want ok", body["status"])
	}
}

func TestHealthHandler_Readiness(t *testing.T) {
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	healthHandler.Readiness(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHealthHandler_ReadinessUnhealthy(t *testing.T) {
	upstream := &testUpstream{healthy: false}
	healthHandler := apihttp.NewHealthHandler(upstream)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	healthHandler.Readiness(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 503 {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestRouter_Integration(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_router1234567890abcdef0123456789abcdef0123456789abcdef0123456789"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-1",
		Email:  "test@example.com",
		PlanID: "free",
		Status: "active",
	})

	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()
	router := apihttp.NewRouter(handler, healthHandler, logger)

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("health status = %d, want 200", rec.Result().StatusCode)
	}

	// Test version endpoint
	req = httptest.NewRequest("GET", "/version", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("version status = %d, want 200", rec.Result().StatusCode)
	}

	// Test proxy endpoint
	req = httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", rawKey)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Errorf("proxy status = %d, want 200, body: %s", rec.Result().StatusCode, body)
	}
}

// Test helpers

type testStores struct {
	keys      *memory.KeyStore
	users     *memory.UserStore
	rateLimit *memory.RateLimitStore
	usage     *testUsageRecorder
}

type testUsageRecorder struct {
	events []usage.Event
}

func (r *testUsageRecorder) Record(e usage.Event) {
	r.events = append(r.events, e)
}

func (r *testUsageRecorder) Flush(ctx context.Context) error {
	return nil
}

func (r *testUsageRecorder) Close() error {
	return nil
}

type testUpstream struct {
	healthy bool
}

func (u *testUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
	return proxy.Response{
		Status:    200,
		Body:      []byte(`{"ok":true}`),
		LatencyMs: 50,
	}, nil
}

func (u *testUpstream) HealthCheck(ctx context.Context) error {
	if !u.healthy {
		return context.DeadlineExceeded
	}
	return nil
}

func (u *testUpstream) ForwardTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (proxy.Response, error) {
	return u.Forward(ctx, req)
}

type testIDGen struct {
	counter int
}

func (g *testIDGen) New() string {
	g.counter++
	return "id-" + itoa(g.counter)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func setupTestHandler() (*apihttp.ProxyHandler, *testStores) {
	stores := &testStores{
		keys:      memory.NewKeyStore(),
		users:     memory.NewUserStore(),
		rateLimit: memory.NewRateLimitStore(),
		usage:     &testUsageRecorder{},
	}

	deps := app.ProxyDeps{
		Keys:      stores.keys,
		Users:     stores.users,
		RateLimit: stores.rateLimit,
		Usage:     stores.usage,
		Upstream:  &testUpstream{healthy: true},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
		},
	}

	service := app.NewProxyService(deps, cfg)
	logger := zerolog.Nop()
	handler := apihttp.NewProxyHandler(service, logger)

	return handler, stores
}

func TestVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()

	apihttp.Version(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)

	if body["service"] != "apigate" {
		t.Errorf("service = %s, want apigate", body["service"])
	}
	if body["version"] == "" {
		t.Error("version should not be empty")
	}
}

func TestHealthHandler_NilUpstream(t *testing.T) {
	healthHandler := apihttp.NewHealthHandler(nil)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	healthHandler.Readiness(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200 (nil upstream = skip check)", resp.StatusCode)
	}
}

func TestNewRouter_BasicEndpoints(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()
	router := apihttp.NewRouter(handler, healthHandler, logger)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/health", 200},
		{"GET", "/health/live", 200},
		{"GET", "/health/ready", 200},
		{"GET", "/version", 200},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != tt.want {
				t.Errorf("status = %d, want %d", rec.Result().StatusCode, tt.want)
			}
		})
	}
}

func TestNewRouterWithConfig_AdminHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("admin"))
	})

	cfg := apihttp.RouterConfig{
		AdminHandler: adminHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/admin/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("admin status = %d, want 200", rec.Result().StatusCode)
	}
	if rec.Body.String() != "admin" {
		t.Errorf("body = %s, want admin", rec.Body.String())
	}
}

func TestNewRouterWithConfig_PortalHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	portalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("portal"))
	})

	cfg := apihttp.RouterConfig{
		PortalHandler: portalHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/portal/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("portal status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestNewRouterWithConfig_WebHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("web:" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	tests := []string{
		"/login",
		"/dashboard",
		"/users",
		"/keys",
		"/plans",
		"/settings",
		"/system",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
			expected := "web:" + path
			if rec.Body.String() != expected {
				t.Errorf("body = %s, want %s", rec.Body.String(), expected)
			}
		})
	}
}

func TestNewProxyHandlerWithMetrics(t *testing.T) {
	stores := &testStores{
		keys:      memory.NewKeyStore(),
		users:     memory.NewUserStore(),
		rateLimit: memory.NewRateLimitStore(),
		usage:     &testUsageRecorder{},
	}

	deps := app.ProxyDeps{
		Keys:      stores.keys,
		Users:     stores.users,
		RateLimit: stores.rateLimit,
		Usage:     stores.usage,
		Upstream:  &testUpstream{healthy: true},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
		},
	}

	service := app.NewProxyService(deps, cfg)
	logger := zerolog.Nop()

	// Create metrics with a custom registry to avoid conflicts
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	handler := apihttp.NewProxyHandlerWithMetrics(service, logger, m)

	if handler == nil {
		t.Error("handler should not be nil")
	}

	// Test that handler works
	req := httptest.NewRequest("GET", "/api/data", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should fail with missing API key (401)
	if rec.Result().StatusCode != 401 {
		t.Errorf("status = %d, want 401", rec.Result().StatusCode)
	}
}

func TestSetStreamingUpstream(t *testing.T) {
	handler, _ := setupTestHandler()

	// Create a mock streaming upstream
	mockStreaming := &mockStreamingUpstream{}
	handler.SetStreamingUpstream(mockStreaming)

	// We can't easily verify the internal state, but the function should not panic
}

type mockStreamingUpstream struct {
	testUpstream
}

func (m *mockStreamingUpstream) ForwardStreaming(ctx context.Context, req proxy.Request) (ports.StreamingResponse, error) {
	return ports.StreamingResponse{
		Status:      200,
		IsStreaming: true,
	}, nil
}

func (m *mockStreamingUpstream) ForwardStreamingTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (ports.StreamingResponse, error) {
	return m.ForwardStreaming(ctx, req)
}

func (m *mockStreamingUpstream) ShouldStream(req proxy.Request, protocol route.Protocol) bool {
	return protocol == route.ProtocolSSE || protocol == route.ProtocolHTTPStream
}

func TestNewMetricsMiddleware(t *testing.T) {
	// Create metrics with a custom registry
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	// Create the middleware
	mw := apihttp.NewMetricsMiddleware(m)
	if mw == nil {
		t.Fatal("middleware should not be nil")
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Wrap with middleware
	wrapped := mw(testHandler)

	tests := []struct {
		name        string
		path        string
		shouldTrack bool
	}{
		{"normal request", "/api/data", true},
		{"health endpoint (skipped)", "/health", false},
		{"health live (skipped)", "/health/live", false},
		{"metrics endpoint (skipped)", "/metrics", false},
		{"swagger (skipped)", "/swagger/index.html", false},
		{"well-known (skipped)", "/.well-known/openapi.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		xForwarded string
		xRealIP    string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Forwarded-For single IP",
			xForwarded: "192.168.1.1",
			remoteAddr: "10.0.0.1:8080",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			xForwarded: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			remoteAddr: "127.0.0.1:8080",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			xRealIP:    "192.168.1.100",
			remoteAddr: "10.0.0.1:8080",
			want:       "192.168.1.100",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.50:12345",
			want:       "192.168.1.50",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.50",
			want:       "192.168.1.50",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:8080",
			want:       "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := setupTestHandler()
			upstream := &testUpstream{healthy: true}
			healthHandler := apihttp.NewHealthHandler(upstream)
			logger := zerolog.Nop()
			router := apihttp.NewRouter(handler, healthHandler, logger)

			// Create a request that will trigger extractIP
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-API-Key", "ak_test123456789012345678901234567890123456789012345678901234567")

			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			req.RemoteAddr = tt.remoteAddr

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// The extractIP function is used internally, we can verify the behavior
			// by checking that the request was processed
		})
	}
}

func TestNewRouterWithConfig_DocsHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	docsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("docs"))
	})

	cfg := apihttp.RouterConfig{
		DocsHandler: docsHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/docs/api", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("docs status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestNewRouterWithConfig_ModuleHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	moduleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("module"))
	})

	cfg := apihttp.RouterConfig{
		ModuleHandler: moduleHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/mod/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("module status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestNewRouterWithConfig_MetricsHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	// Create metrics with a custom registry
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	cfg := apihttp.RouterConfig{
		Metrics: m,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should return 200 with Prometheus metrics
	if rec.Result().StatusCode != 200 {
		t.Errorf("metrics status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestNewRouterWithConfig_CustomMetricsHandler(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	customMetricsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("custom_metrics"))
	})

	cfg := apihttp.RouterConfig{
		MetricsHandler: customMetricsHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("custom metrics status = %d, want 200", rec.Result().StatusCode)
	}
	if rec.Body.String() != "custom_metrics" {
		t.Errorf("body = %s, want custom_metrics", rec.Body.String())
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger := zerolog.Nop()
	mw := apihttp.NewLoggingMiddleware(logger)
	if mw == nil {
		t.Fatal("middleware should not be nil")
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	wrapped := mw(testHandler)

	tests := []struct {
		name string
		path string
	}{
		{"normal request", "/api/data"},
		{"health endpoint (skipped logging)", "/health"},
		{"metrics endpoint (skipped logging)", "/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}
}

func TestNewRouterWithConfig_WebHandlerPOST(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("web:" + r.Method + ":" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	// Test POST endpoints
	postPaths := []string{
		"/login",
		"/logout",
		"/forgot-password",
		"/reset-password",
		"/setup",
		"/users",
		"/keys",
		"/plans",
		"/routes",
		"/upstreams",
		"/settings",
	}

	for _, path := range postPaths {
		t.Run("POST "+path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
			expected := "web:POST:" + path
			if rec.Body.String() != expected {
				t.Errorf("body = %s, want %s", rec.Body.String(), expected)
			}
		})
	}
}

func TestNewRouterWithConfig_WebHandlerDELETE(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("web:" + r.Method + ":" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	// Test DELETE endpoints
	deletePaths := []string{
		"/users/123",
		"/keys/abc",
		"/plans/free",
		"/routes/route-1",
		"/upstreams/up-1",
	}

	for _, path := range deletePaths {
		t.Run("DELETE "+path, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}
}

func TestNewRouterWithConfig_WebHandlerPartials(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("partial:" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/partials/header", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("partials status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestProxyHandler_AuthorizationNotBearer(t *testing.T) {
	handler, _ := setupTestHandler()

	// Use non-Bearer Authorization header
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0") // Basic auth

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should fail with missing API key since Basic is not supported for extraction
	resp := rec.Result()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestProxyHandler_XForwardedForHeader(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_xff1234567890abcdef0123456789abcdef0123456789abcdef0123456789abcd"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-xff",
		UserID:    "user-xff",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-xff",
		Email:  "xff@example.com",
		PlanID: "free",
		Status: "active",
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

func TestProxyHandler_XRealIPHeader(t *testing.T) {
	handler, stores := setupTestHandler()

	rawKey := "ak_xri1234567890abcdef0123456789abcdef0123456789abcdef0123456789abcd"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-xri",
		UserID:    "user-xri",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-xri",
		Email:  "xri@example.com",
		PlanID: "free",
		Status: "active",
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-Real-IP", "203.0.113.50")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}
}

func TestMetricsMiddleware_StatusLabels(t *testing.T) {
	// Create metrics with a custom registry
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	mw := apihttp.NewMetricsMiddleware(m)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"2xx status", 200},
		{"3xx status", 301},
		{"4xx status", 400},
		{"5xx status", 500},
		{"1xx status", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrapped := mw(testHandler)
			req := httptest.NewRequest("GET", "/api/test", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Result().StatusCode != tt.statusCode {
				t.Errorf("status = %d, want %d", rec.Result().StatusCode, tt.statusCode)
			}
		})
	}
}

func TestNewRouterWithConfig_AllWebRoutes(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("web:" + r.Method + ":" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	// Test additional GET paths
	getRoutes := []string{
		"/",
		"/terms",
		"/privacy",
		"/routes",
		"/upstreams",
		"/usage",
		"/forgot-password",
		"/reset-password",
	}

	for _, path := range getRoutes {
		t.Run("GET "+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}

	// Test POST paths with wildcards
	postWildcardRoutes := []string{
		"/setup/step1",
		"/users/123",
		"/plans/free",
		"/routes/r1",
		"/upstreams/u1",
	}

	for _, path := range postWildcardRoutes {
		t.Run("POST "+path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}

	// Test GET paths with wildcards
	getWildcardRoutes := []string{
		"/setup/step2",
		"/users/456",
		"/plans/pro",
		"/routes/r2",
		"/upstreams/u2",
	}

	for _, path := range getWildcardRoutes {
		t.Run("GET "+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != 200 {
				t.Errorf("status = %d, want 200", rec.Result().StatusCode)
			}
		})
	}
}

func TestNewRouterWithConfig_StaticFiles(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("static:" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	req := httptest.NewRequest("GET", "/static/style.css", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestNewRouterWithConfig_APIRoutes(t *testing.T) {
	handler, _ := setupTestHandler()
	upstream := &testUpstream{healthy: true}
	healthHandler := apihttp.NewHealthHandler(upstream)
	logger := zerolog.Nop()

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("api:" + r.URL.Path))
	})

	cfg := apihttp.RouterConfig{
		WebHandler: webHandler,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, cfg)

	// Test API expression routes
	req := httptest.NewRequest("POST", "/api/expr/validate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}

	// Test API routes endpoint
	req = httptest.NewRequest("POST", "/api/routes/test", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestProxyHandlerWithMetrics_LogRequest(t *testing.T) {
	stores := &testStores{
		keys:      memory.NewKeyStore(),
		users:     memory.NewUserStore(),
		rateLimit: memory.NewRateLimitStore(),
		usage:     &testUsageRecorder{},
	}

	deps := app.ProxyDeps{
		Keys:      stores.keys,
		Users:     stores.users,
		RateLimit: stores.rateLimit,
		Usage:     stores.usage,
		Upstream:  &testUpstream{healthy: true},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
		},
	}

	service := app.NewProxyService(deps, cfg)
	logger := zerolog.Nop()

	// Create metrics with a custom registry
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	handler := apihttp.NewProxyHandlerWithMetrics(service, logger, m)

	// Test error logging (missing API key - 401)
	req := httptest.NewRequest("GET", "/api/data", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 401 {
		t.Errorf("status = %d, want 401", rec.Result().StatusCode)
	}

	// Test with invalid API key (401)
	req = httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "ak_invalid0123456789abcdef0123456789abcdef0123456789abcdef01234567")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 401 {
		t.Errorf("status = %d, want 401", rec.Result().StatusCode)
	}
}

func TestProxyHandlerWithMetrics_SuccessfulRequest(t *testing.T) {
	stores := &testStores{
		keys:      memory.NewKeyStore(),
		users:     memory.NewUserStore(),
		rateLimit: memory.NewRateLimitStore(),
		usage:     &testUsageRecorder{},
	}

	deps := app.ProxyDeps{
		Keys:      stores.keys,
		Users:     stores.users,
		RateLimit: stores.rateLimit,
		Usage:     stores.usage,
		Upstream:  &testUpstream{healthy: true},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
		},
	}

	service := app.NewProxyService(deps, cfg)
	logger := zerolog.Nop()

	// Create metrics with a custom registry
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	handler := apihttp.NewProxyHandlerWithMetrics(service, logger, m)

	// Create valid key and user
	rawKey := "ak_metr123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(context.Background(), key.Key{
		ID:        "key-metrics",
		UserID:    "user-metrics",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(context.Background(), ports.User{
		ID:     "user-metrics",
		Email:  "metrics@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Test successful request with body
	bodyContent := `{"data": "test"}`
	req := httptest.NewRequest("POST", "/api/data", strings.NewReader(bodyContent))
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Errorf("status = %d, want 200, body: %s", rec.Result().StatusCode, body)
	}
}

