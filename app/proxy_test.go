package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/adapters/clock"
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

func TestProxyService_Handle_ValidRequest(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Seed test data
	// Key must be: prefix (3) + 64 hex chars = 67 chars minimum
	rawKey := "ak_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12], // "ak_012345678"
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Email:  "test@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Act
	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	// Assert
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}
	if result.Auth == nil {
		t.Fatal("expected auth context")
	}
	if result.Auth.UserID != "user-1" {
		t.Errorf("userID = %s, want user-1", result.Auth.UserID)
	}

	// Verify usage was recorded
	events := stores.usage.Drain()
	if len(events) != 1 {
		t.Fatalf("expected 1 usage event, got %d", len(events))
	}
	if events[0].KeyID != "key-1" {
		t.Errorf("event keyID = %s, want key-1", events[0].KeyID)
	}
}

func TestProxyService_Handle_InvalidKey(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	req := proxy.Request{
		APIKey: "ak_invalid0123456789abcdef0123456789abcdef0123456789abcdef01234",
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for invalid key")
	}
	if result.Error.Status != 401 {
		t.Errorf("status = %d, want 401", result.Error.Status)
	}
}

func TestProxyService_Handle_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create expired key (prefix + 64 hex chars = 67 total)
	rawKey := "ak_1111111111111111111111111111111111111111111111111111111111111111"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	expiredAt := baseTime.Add(-time.Hour)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		ExpiresAt: &expiredAt,
		CreatedAt: baseTime.Add(-24 * time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Status: "active",
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for expired key")
	}
	if result.Error.Code != key.ReasonExpired {
		t.Errorf("code = %s, want %s", result.Error.Code, key.ReasonExpired)
	}
}

func TestProxyService_Handle_RateLimited(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create key with tight rate limit (prefix + 64 hex chars = 67 total)
	rawKey := "ak_2222222222222222222222222222222222222222222222222222222222222222"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "limited",
		Status: "active",
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}

	// Make requests until rate limited
	// Plan "limited" has rate limit of 2 per minute + 2 burst = 4 total
	for i := 0; i < 10; i++ {
		result := svc.Handle(ctx, req)
		if result.Error != nil && result.Error.Code == "rate_limit_exceeded" {
			// Expected - rate limited after a few requests
			return
		}
	}

	t.Error("expected to be rate limited after exceeding limit")
}

func TestProxyService_Handle_SuspendedUser(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// prefix + 64 hex chars = 67 total
	rawKey := "ak_3333333333333333333333333333333333333333333333333333333333333333"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Status: "suspended", // Suspended user
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for suspended user")
	}
	if result.Error.Code != "user_suspended" {
		t.Errorf("code = %s, want user_suspended", result.Error.Code)
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

func (r *testUsageRecorder) Drain() []usage.Event {
	events := r.events
	r.events = nil
	return events
}

type testUpstream struct{}

func (u *testUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
	return proxy.Response{
		Status:    200,
		Body:      []byte(`{"ok":true}`),
		LatencyMs: 50,
	}, nil
}

func (u *testUpstream) HealthCheck(ctx context.Context) error {
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

func newTestProxyService() (*app.ProxyService, *testStores) {
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
		Upstream:  &testUpstream{},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
			{ID: "limited", Name: "Limited", RateLimitPerMinute: 2, RequestsPerMonth: 100},
		},
	}

	return app.NewProxyService(deps, cfg), stores
}

func TestProxyService_UpdateConfig(t *testing.T) {
	svc, _ := newTestProxyService()

	// Update config
	newPlans := []plan.Plan{
		{ID: "premium", Name: "Premium", RateLimitPerMinute: 1000, RequestsPerMonth: 100000},
	}
	newEndpoints := []plan.Endpoint{
		{Method: "POST", Path: "/expensive/*", CostMultiplier: 10.0},
	}

	svc.UpdateConfig(newPlans, newEndpoints, 10, 120, nil, nil)

	// Service should still function
	ctx := context.Background()
	req := proxy.Request{
		APIKey: "ak_invalid0123456789abcdef0123456789abcdef0123456789abcdef01234",
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	// Should still validate keys (expect invalid key error)
	if result.Error == nil {
		t.Fatal("expected error")
	}
}

func TestProxyService_SetRouteService(t *testing.T) {
	svc, _ := newTestProxyService()

	// Create a route service
	routes := []route.Route{
		{ID: "r1", Name: "API", PathPattern: "/api/*", MatchType: route.MatchPrefix, Enabled: true},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	svc.SetRouteService(routeService)

	// SetTransformService
	transformService := app.NewTransformService()
	svc.SetTransformService(transformService)
}

func TestProxyService_Handle_WithRouteMatching(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create route and transform services
	routes := []route.Route{
		{
			ID:             "r1",
			Name:           "API Route",
			PathPattern:    "/api/*",
			MatchType:      route.MatchPrefix,
			MethodOverride: "POST",
			PathRewrite:    `"/v2" + path`,
			UpstreamID:     "upstream-1",
			Enabled:        true,
			Priority:       10,
		},
	}
	upstreams := []route.Upstream{
		{
			ID:       "upstream-1",
			Name:     "Backend",
			BaseURL:  "https://backend.example.com",
			AuthType: route.AuthBearer,
			AuthValue: "test-token",
			Enabled:  true,
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{upstreams: upstreams}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	// Create valid key
	rawKey := "ak_4444444444444444444444444444444444444444444444444444444444444444"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "free",
		Status: "active",
	})

	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
}

func TestProxyService_Handle_WithRequestTransform(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create route with request transform
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "API Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			Enabled:     true,
			Priority:    10,
			RequestTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-Custom": `"value"`,
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	// Create valid key
	rawKey := "ak_5555555555555555555555555555555555555555555555555555555555555555"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "free",
		Status: "active",
	})

	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		Headers:   map[string]string{"Existing": "keep"},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
}

func TestProxyService_Handle_WithResponseTransform(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create route with response transform
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "API Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			Enabled:     true,
			Priority:    10,
			ResponseTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-Response": `"added"`,
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	// Create valid key
	rawKey := "ak_6666666666666666666666666666666666666666666666666666666666666666"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "free",
		Status: "active",
	})

	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
}

func TestProxyService_Handle_WithMeteringExpr(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create route with metering expression
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "API Route",
			PathPattern:  "/api/*",
			MatchType:    route.MatchPrefix,
			Enabled:      true,
			Priority:     10,
			MeteringExpr: "responseBytes / 1000",
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	// Create valid key
	rawKey := "ak_7777777777777777777777777777777777777777777777777777777777777777"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "free",
		Status: "active",
	})

	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
}

func TestProxyService_ShouldStream(t *testing.T) {
	svc, _ := newTestProxyService()

	// Without route service, should check Accept header
	tests := []struct {
		name    string
		req     proxy.Request
		want    bool
	}{
		{
			"no streaming headers",
			proxy.Request{Method: "GET", Path: "/api/data", Headers: map[string]string{}},
			false,
		},
		{
			"SSE accept header",
			proxy.Request{Method: "GET", Path: "/api/events", Headers: map[string]string{"Accept": "text/event-stream"}},
			true,
		},
		{
			"mixed accept header with SSE",
			proxy.Request{Method: "GET", Path: "/api/events", Headers: map[string]string{"Accept": "application/json, text/event-stream"}},
			true,
		},
		{
			"case insensitive SSE",
			proxy.Request{Method: "GET", Path: "/api/events", Headers: map[string]string{"Accept": "TEXT/EVENT-STREAM"}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ShouldStream(tt.req)
			if got != tt.want {
				t.Errorf("ShouldStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxyService_ShouldStream_WithRouteService(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	// Create route with SSE protocol
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "SSE Route",
			PathPattern: "/api/stream/*",
			MatchType:   route.MatchPrefix,
			Protocol:    route.ProtocolSSE,
			Enabled:     true,
			Priority:    10,
		},
		{
			ID:          "r2",
			Name:        "HTTP Stream Route",
			PathPattern: "/api/httpstream/*",
			MatchType:   route.MatchPrefix,
			Protocol:    route.ProtocolHTTPStream,
			Enabled:     true,
			Priority:    10,
		},
		{
			ID:          "r3",
			Name:        "WebSocket Route",
			PathPattern: "/api/ws/*",
			MatchType:   route.MatchPrefix,
			Protocol:    route.ProtocolWebSocket,
			Enabled:     true,
			Priority:    10,
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	svc.SetRouteService(routeService)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"SSE route", "/api/stream/events", true},
		{"HTTP stream route", "/api/httpstream/data", true},
		{"WebSocket route", "/api/ws/connect", true},
		{"regular route", "/api/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := proxy.Request{Method: "GET", Path: tt.path}
			got := svc.ShouldStream(req)
			if got != tt.want {
				t.Errorf("ShouldStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxyService_RecordStreamingUsage(t *testing.T) {
	svc, stores := newTestProxyService()

	streamCtx := &app.StreamingResponseContext{
		OriginalPath: "/api/stream",
		KeyID:        "key-1",
		UserID:       "user-1",
	}

	svc.RecordStreamingUsage(
		streamCtx,
		200,       // statusCode
		100,       // requestBytes
		5000,      // responseBytes
		500,       // latencyMs
		2.5,       // meteringValue
		"1.2.3.4", // remoteIP
		"test-agent",
	)

	events := stores.usage.Drain()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Method != "STREAM" {
		t.Errorf("Method = %s, want STREAM", e.Method)
	}
	if e.Path != "/api/stream" {
		t.Errorf("Path = %s, want /api/stream", e.Path)
	}
	if e.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", e.StatusCode)
	}
	if e.ResponseBytes != 5000 {
		t.Errorf("ResponseBytes = %d, want 5000", e.ResponseBytes)
	}
	if e.CostMultiplier != 2.5 {
		t.Errorf("CostMultiplier = %f, want 2.5", e.CostMultiplier)
	}
}

func TestProxyService_EvalStreamingMetering(t *testing.T) {
	svc, _ := newTestProxyService()
	transformService := app.NewTransformService()
	svc.SetTransformService(transformService)

	ctx := context.Background()
	auth := &proxy.AuthContext{UserID: "user-1", PlanID: "free", KeyID: "key-1"}

	tests := []struct {
		name         string
		meteringExpr string
		responseBytes int64
		want         float64
	}{
		{
			"simple count",
			"1",
			1000,
			1.0,
		},
		{
			"KB based",
			"responseBytes / 1000",
			5000,
			5.0,
		},
		{
			"negative result returns 0",
			"-10",
			1000,
			0.0,
		},
		{
			"empty expr returns default",
			"",
			1000,
			1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.EvalStreamingMetering(ctx, tt.meteringExpr, 200, tt.responseBytes, nil, nil, auth)
			if got != tt.want {
				t.Errorf("EvalStreamingMetering() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestProxyService_EvalStreamingMetering_NoTransformService(t *testing.T) {
	svc, _ := newTestProxyService()
	// No transform service set

	ctx := context.Background()
	got := svc.EvalStreamingMetering(ctx, "responseBytes / 1000", 200, 5000, nil, nil, nil)

	// Should return default 1.0 when no transform service
	if got != 1.0 {
		t.Errorf("EvalStreamingMetering() = %f, want 1.0", got)
	}
}

func TestProxyService_Handle_RevokedKey(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	rawKey := "ak_8888888888888888888888888888888888888888888888888888888888888888"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	revokedAt := baseTime.Add(-time.Hour)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		RevokedAt: &revokedAt,
		CreatedAt: baseTime.Add(-24 * time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Status: "active",
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for revoked key")
	}
	if result.Error.Code != key.ReasonRevoked {
		t.Errorf("code = %s, want %s", result.Error.Code, key.ReasonRevoked)
	}
}

func TestProxyService_Handle_ShortAPIKey(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	req := proxy.Request{
		APIKey: "ak_short",
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for short key")
	}
	if result.Error.Status != 401 {
		t.Errorf("status = %d, want 401", result.Error.Status)
	}
}

func TestProxyService_Handle_WrongPrefix(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	req := proxy.Request{
		APIKey: "sk_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for wrong prefix")
	}
}

func TestProxyService_Handle_KeyNotFound(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	// Valid format but key doesn't exist
	rawKey := "ak_9999999999999999999999999999999999999999999999999999999999999999"

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for non-existent key")
	}
	if result.Error.Status != 401 {
		t.Errorf("status = %d, want 401", result.Error.Status)
	}
}

func TestProxyService_Handle_HashMismatch(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create key with different hash
	rawKey := "ak_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	wrongHash, _ := bcrypt.GenerateFromPassword([]byte("different_key"), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      wrongHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for hash mismatch")
	}
}

// capturingUpstream records the last forwarded request for header inspection.
type capturingUpstream struct {
	mu      sync.Mutex
	lastReq proxy.Request
}

func (u *capturingUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
	u.mu.Lock()
	u.lastReq = req
	u.mu.Unlock()
	return proxy.Response{
		Status:    200,
		Body:      []byte(`{"ok":true}`),
		LatencyMs: 10,
	}, nil
}

func (u *capturingUpstream) HealthCheck(ctx context.Context) error { return nil }

func (u *capturingUpstream) ForwardTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (proxy.Response, error) {
	return u.Forward(ctx, req)
}

func (u *capturingUpstream) LastRequest() proxy.Request {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.lastReq
}

func newTestProxyServiceWithUpstream(up ports.Upstream) (*app.ProxyService, *testStores) {
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
		Upstream:  up,
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

	return app.NewProxyService(deps, cfg), stores
}

func TestProxyService_Handle_PublicRouteWithJWT(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, stores := newTestProxyServiceWithUpstream(upstream)

	// Create a public route with set_headers transform referencing auth variables
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "Public API",
			PathPattern:  "/public/*",
			MatchType:    route.MatchPrefix,
			AuthRequired: false,
			Enabled:      true,
			Priority:     10,
			RequestTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-User-ID": "userID",
					"X-Email":   "email",
					"X-Role":    "role",
					"X-Plan-ID": "planID",
					"X-Key-ID":  "keyID",
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	// Create token service and generate a valid JWT
	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	jwt, _, err := tokenService.GenerateToken("user-42", "alice@example.com", "admin", "pro")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// No user in store needed — public routes skip user lookup
	_ = stores

	req := proxy.Request{
		APIKey:    jwt,
		Method:    "GET",
		Path:      "/public/data",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}

	// Verify auth context is populated
	if result.Auth == nil {
		t.Fatal("expected auth context on public route with valid JWT")
	}
	if result.Auth.UserID != "user-42" {
		t.Errorf("auth.UserID = %q, want %q", result.Auth.UserID, "user-42")
	}
	if result.Auth.Email != "alice@example.com" {
		t.Errorf("auth.Email = %q, want %q", result.Auth.Email, "alice@example.com")
	}

	// Verify transform resolved auth variables in forwarded headers
	forwarded := upstream.LastRequest()
	if forwarded.Headers["X-User-ID"] != "user-42" {
		t.Errorf("X-User-ID = %q, want %q", forwarded.Headers["X-User-ID"], "user-42")
	}
	if forwarded.Headers["X-Email"] != "alice@example.com" {
		t.Errorf("X-Email = %q, want %q", forwarded.Headers["X-Email"], "alice@example.com")
	}
	if forwarded.Headers["X-Role"] != "admin" {
		t.Errorf("X-Role = %q, want %q", forwarded.Headers["X-Role"], "admin")
	}
	if forwarded.Headers["X-Plan-ID"] != "pro" {
		t.Errorf("X-Plan-ID = %q, want %q", forwarded.Headers["X-Plan-ID"], "pro")
	}
	if forwarded.Headers["X-Key-ID"] != "session:user-42" {
		t.Errorf("X-Key-ID = %q, want %q", forwarded.Headers["X-Key-ID"], "session:user-42")
	}
}

func TestProxyService_Handle_PublicRouteWithoutToken(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, _ := newTestProxyServiceWithUpstream(upstream)

	// Public route with set_headers transform
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "Public API",
			PathPattern:  "/public/*",
			MatchType:    route.MatchPrefix,
			AuthRequired: false,
			Enabled:      true,
			Priority:     10,
			RequestTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-User-ID": "userID",
					"X-Email":   "email",
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	// No APIKey at all
	req := proxy.Request{
		Method:    "GET",
		Path:      "/public/data",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error for public route without token, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}

	// Auth should be nil (anonymous)
	if result.Auth != nil {
		t.Error("expected nil auth for public route without token")
	}

	// Transform should resolve auth variables to empty strings
	forwarded := upstream.LastRequest()
	if forwarded.Headers["X-User-ID"] != "" {
		t.Errorf("X-User-ID = %q, want empty", forwarded.Headers["X-User-ID"])
	}
	if forwarded.Headers["X-Email"] != "" {
		t.Errorf("X-Email = %q, want empty", forwarded.Headers["X-Email"])
	}
}

func TestProxyService_Handle_PublicRouteWithInvalidJWT(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, _ := newTestProxyServiceWithUpstream(upstream)

	// Public route with set_headers transform
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "Public API",
			PathPattern:  "/public/*",
			MatchType:    route.MatchPrefix,
			AuthRequired: false,
			Enabled:      true,
			Priority:     10,
			RequestTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-User-ID": "userID",
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	// Send an invalid JWT (not API key format, but not valid JWT)
	req := proxy.Request{
		APIKey:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		Method:    "GET",
		Path:      "/public/data",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	// Should succeed (no 401) — public route ignores invalid JWT
	if result.Error != nil {
		t.Fatalf("expected no error for public route with invalid JWT, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}

	// Auth should be nil (invalid JWT silently ignored)
	if result.Auth != nil {
		t.Error("expected nil auth for public route with invalid JWT")
	}

	// Auth variables should be empty
	forwarded := upstream.LastRequest()
	if forwarded.Headers["X-User-ID"] != "" {
		t.Errorf("X-User-ID = %q, want empty", forwarded.Headers["X-User-ID"])
	}
}

func TestProxyService_Handle_PublicRouteWithAPIKey(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, _ := newTestProxyServiceWithUpstream(upstream)

	// Public route with set_headers transform
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "Public API",
			PathPattern:  "/public/*",
			MatchType:    route.MatchPrefix,
			AuthRequired: false,
			Enabled:      true,
			Priority:     10,
			RequestTransform: &route.Transform{
				SetHeaders: map[string]string{
					"X-User-ID": "userID",
				},
			},
		},
	}
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{}
	clk := clock.NewFake(baseTime)
	logger := zerolog.Nop()

	routeService := app.NewRouteService(routeStore, upstreamStore, clk, logger, app.RouteServiceConfig{})
	_ = routeService.Start(ctx)
	defer routeService.Stop()

	transformService := app.NewTransformService()

	svc.SetRouteService(routeService)
	svc.SetTransformService(transformService)

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	// Send API key format token — should be ignored (no DB lookup for public routes)
	rawKey := "ak_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/public/data",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	// Should succeed — API keys are silently ignored on public routes
	if result.Error != nil {
		t.Fatalf("expected no error for public route with API key, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}

	// Auth should be nil (API key not validated on public routes)
	if result.Auth != nil {
		t.Error("expected nil auth for public route with API key format token")
	}

	// Auth variables should be empty
	forwarded := upstream.LastRequest()
	if forwarded.Headers["X-User-ID"] != "" {
		t.Errorf("X-User-ID = %q, want empty", forwarded.Headers["X-User-ID"])
	}
}

func TestProxyService_Handle_JWTBillingRoute_UserNotInDB(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, _ := newTestProxyServiceWithUpstream(upstream)

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	jwt, _, err := tokenService.GenerateToken("ext-user-1", "ext@example.com", "user", "pro")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// No user created in store — simulates external auth (user in Hoster, not in APIGate DB)

	req := proxy.Request{
		APIKey:    jwt,
		Method:    "GET",
		Path:      "/api/billing",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got status=%d code=%s msg=%s",
			result.Error.Status, result.Error.Code, result.Error.Message)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}
	if result.Auth == nil {
		t.Fatal("expected auth context")
	}
	if result.Auth.UserID != "ext-user-1" {
		t.Errorf("auth.UserID = %q, want %q", result.Auth.UserID, "ext-user-1")
	}
	if result.Auth.Email != "ext@example.com" {
		t.Errorf("auth.Email = %q, want %q", result.Auth.Email, "ext@example.com")
	}
	if result.Auth.PlanID != "pro" {
		t.Errorf("auth.PlanID = %q, want %q", result.Auth.PlanID, "pro")
	}
	if result.Auth.KeyID != "session:ext-user-1" {
		t.Errorf("auth.KeyID = %q, want %q", result.Auth.KeyID, "session:ext-user-1")
	}
}

func TestProxyService_Handle_JWTBillingRoute_UserInDB(t *testing.T) {
	ctx := context.Background()
	upstream := &capturingUpstream{}
	svc, stores := newTestProxyServiceWithUpstream(upstream)

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	jwt, _, err := tokenService.GenerateToken("user-1", "db@example.com", "user", "free")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// User exists in DB with different plan — DB values should be used
	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Email:  "db@example.com",
		PlanID: "enterprise",
		Status: "active",
	})

	req := proxy.Request{
		APIKey:    jwt,
		Method:    "GET",
		Path:      "/api/billing",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error != nil {
		t.Fatalf("expected no error, got status=%d code=%s msg=%s",
			result.Error.Status, result.Error.Code, result.Error.Message)
	}
	if result.Auth == nil {
		t.Fatal("expected auth context")
	}
	if result.Auth.UserID != "user-1" {
		t.Errorf("auth.UserID = %q, want %q", result.Auth.UserID, "user-1")
	}
	// PlanID should come from DB user, not JWT claims
	if result.Auth.PlanID != "enterprise" {
		t.Errorf("auth.PlanID = %q, want %q (from DB)", result.Auth.PlanID, "enterprise")
	}
}

func TestProxyService_Handle_JWTBillingRoute_UserSuspended(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyServiceWithUpstream(&capturingUpstream{})

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	jwt, _, err := tokenService.GenerateToken("user-1", "suspended@example.com", "user", "free")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Email:  "suspended@example.com",
		PlanID: "free",
		Status: "suspended",
	})

	req := proxy.Request{
		APIKey:    jwt,
		Method:    "GET",
		Path:      "/api/billing",
		Headers:   map[string]string{},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for suspended user")
	}
	if result.Error.Status != 403 {
		t.Errorf("status = %d, want 403", result.Error.Status)
	}
	if result.Error.Code != "user_suspended" {
		t.Errorf("code = %q, want %q", result.Error.Code, "user_suspended")
	}
}

func TestProxyService_HandleStreaming_JWTBillingRoute_UserNotInDB(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyServiceWithUpstream(&capturingUpstream{})

	tokenService := auth.NewTokenService("test-secret-key", time.Hour)
	svc.SetTokenService(tokenService)

	jwt, _, err := tokenService.GenerateToken("ext-user-2", "stream@example.com", "user", "pro")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// No user in DB — should use JWT claims

	req := proxy.Request{
		APIKey:    jwt,
		Method:    "GET",
		Path:      "/api/stream",
		Headers:   map[string]string{"Accept": "text/event-stream"},
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}

	streamUpstream := &testStreamingUpstream{}
	result := svc.HandleStreaming(ctx, req, streamUpstream)

	if result.Error != nil {
		t.Fatalf("expected no error, got status=%d code=%s msg=%s",
			result.Error.Status, result.Error.Code, result.Error.Message)
	}
	if result.Auth == nil {
		t.Fatal("expected auth context")
	}
	if result.Auth.UserID != "ext-user-2" {
		t.Errorf("auth.UserID = %q, want %q", result.Auth.UserID, "ext-user-2")
	}
	if result.Auth.Email != "stream@example.com" {
		t.Errorf("auth.Email = %q, want %q", result.Auth.Email, "stream@example.com")
	}
	if result.Auth.PlanID != "pro" {
		t.Errorf("auth.PlanID = %q, want %q", result.Auth.PlanID, "pro")
	}
}

type testStreamingUpstream struct{}

func (u *testStreamingUpstream) ForwardStreaming(ctx context.Context, req proxy.Request) (interface{ Status() int }, error) {
	return &testStreamResponse{status: 200}, nil
}

type testStreamResponse struct{ status int }

func (r *testStreamResponse) Status() int { return r.status }
