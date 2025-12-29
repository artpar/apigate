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
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
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

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
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

