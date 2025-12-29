package app_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/clock"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/route"
	"github.com/rs/zerolog"
)

// mockRouteStore implements ports.RouteStore for testing.
type mockRouteStore struct {
	routes []route.Route
	err    error
}

func (m *mockRouteStore) Create(ctx context.Context, r route.Route) error { return nil }
func (m *mockRouteStore) Update(ctx context.Context, r route.Route) error { return nil }
func (m *mockRouteStore) Delete(ctx context.Context, id string) error     { return nil }
func (m *mockRouteStore) Get(ctx context.Context, id string) (route.Route, error) {
	for _, r := range m.routes {
		if r.ID == id {
			return r, nil
		}
	}
	return route.Route{}, nil
}
func (m *mockRouteStore) List(ctx context.Context) ([]route.Route, error) { return m.routes, m.err }
func (m *mockRouteStore) ListEnabled(ctx context.Context) ([]route.Route, error) {
	var enabled []route.Route
	for _, r := range m.routes {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled, m.err
}

// mockUpstreamStore implements ports.UpstreamStore for testing.
type mockUpstreamStore struct {
	upstreams []route.Upstream
	err       error
}

func (m *mockUpstreamStore) Create(ctx context.Context, u route.Upstream) error { return nil }
func (m *mockUpstreamStore) Update(ctx context.Context, u route.Upstream) error { return nil }
func (m *mockUpstreamStore) Delete(ctx context.Context, id string) error        { return nil }
func (m *mockUpstreamStore) Get(ctx context.Context, id string) (route.Upstream, error) {
	for _, u := range m.upstreams {
		if u.ID == id {
			return u, nil
		}
	}
	return route.Upstream{}, nil
}
func (m *mockUpstreamStore) List(ctx context.Context) ([]route.Upstream, error) {
	return m.upstreams, m.err
}
func (m *mockUpstreamStore) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	var enabled []route.Upstream
	for _, u := range m.upstreams {
		if u.Enabled {
			enabled = append(enabled, u)
		}
	}
	return enabled, m.err
}

func newTestRouteService(routes []route.Route, upstreams []route.Upstream) *app.RouteService {
	routeStore := &mockRouteStore{routes: routes}
	upstreamStore := &mockUpstreamStore{upstreams: upstreams}
	logger := zerolog.Nop()
	clk := clock.NewFake(time.Now())

	return app.NewRouteService(
		routeStore,
		upstreamStore,
		clk,
		logger,
		app.RouteServiceConfig{RefreshInterval: time.Hour},
	)
}

func TestRouteService_Match_NoCache(t *testing.T) {
	svc := newTestRouteService(nil, nil)

	// Without Start(), cache is nil
	result := svc.Match("GET", "/api/users", nil)
	if result != nil {
		t.Error("expected nil result when cache not initialized")
	}
}

func TestRouteService_Match_WithRoutes(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "route-1",
			Name:        "API V1",
			PathPattern: "/api/v1/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "upstream-1",
			Enabled:     true,
			Priority:    10,
		},
		{
			ID:          "route-2",
			Name:        "API V2",
			PathPattern: "/api/v2/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "upstream-1",
			Enabled:     true,
			Priority:    10,
		},
	}

	upstreams := []route.Upstream{
		{
			ID:      "upstream-1",
			Name:    "Backend",
			BaseURL: "https://api.example.com",
			Enabled: true,
		},
	}

	svc := newTestRouteService(routes, upstreams)
	ctx := context.Background()

	err := svc.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer svc.Stop()

	tests := []struct {
		name      string
		method    string
		path      string
		wantMatch bool
		wantRoute string
	}{
		{"v1 match", "GET", "/api/v1/users", true, "route-1"},
		{"v2 match", "POST", "/api/v2/data", true, "route-2"},
		{"no match", "GET", "/other/path", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.Match(tt.method, tt.path, nil)
			if tt.wantMatch {
				if result == nil {
					t.Fatal("expected match, got nil")
				}
				if result.Route.ID != tt.wantRoute {
					t.Errorf("got route %s, want %s", result.Route.ID, tt.wantRoute)
				}
			} else {
				if result != nil {
					t.Errorf("expected no match, got %s", result.Route.ID)
				}
			}
		})
	}
}

func TestRouteService_GetUpstream(t *testing.T) {
	upstreams := []route.Upstream{
		{
			ID:      "upstream-1",
			Name:    "Backend",
			BaseURL: "https://api.example.com",
			Enabled: true,
		},
	}

	svc := newTestRouteService(nil, upstreams)
	ctx := context.Background()

	// Before Start
	u := svc.GetUpstream("upstream-1")
	if u != nil {
		t.Error("expected nil before Start")
	}

	// After Start
	_ = svc.Start(ctx)
	defer svc.Stop()

	u = svc.GetUpstream("upstream-1")
	if u == nil {
		t.Fatal("expected upstream after Start")
	}
	if u.Name != "Backend" {
		t.Errorf("got %s, want Backend", u.Name)
	}

	// Non-existent upstream
	u = svc.GetUpstream("nonexistent")
	if u != nil {
		t.Error("expected nil for nonexistent upstream")
	}
}

func TestRouteService_GetRoutes(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route 1", PathPattern: "/a/*", MatchType: route.MatchPrefix, Enabled: true},
		{ID: "r2", Name: "Route 2", PathPattern: "/b/*", MatchType: route.MatchPrefix, Enabled: true},
	}

	svc := newTestRouteService(routes, nil)
	ctx := context.Background()

	// Before Start
	r := svc.GetRoutes()
	if r != nil {
		t.Error("expected nil before Start")
	}

	// After Start
	_ = svc.Start(ctx)
	defer svc.Stop()

	r = svc.GetRoutes()
	if len(r) != 2 {
		t.Errorf("got %d routes, want 2", len(r))
	}
}

func TestRouteService_GetUpstreams(t *testing.T) {
	upstreams := []route.Upstream{
		{ID: "u1", Name: "Upstream 1", BaseURL: "http://a.com", Enabled: true},
		{ID: "u2", Name: "Upstream 2", BaseURL: "http://b.com", Enabled: true},
	}

	svc := newTestRouteService(nil, upstreams)
	ctx := context.Background()

	// Before Start
	u := svc.GetUpstreams()
	if u != nil {
		t.Error("expected nil before Start")
	}

	// After Start
	_ = svc.Start(ctx)
	defer svc.Stop()

	u = svc.GetUpstreams()
	if len(u) != 2 {
		t.Errorf("got %d upstreams, want 2", len(u))
	}
}

func TestRouteService_ResolveUpstreamURL(t *testing.T) {
	svc := newTestRouteService(nil, nil)

	upstream := &route.Upstream{
		BaseURL: "https://api.example.com/v1",
	}

	tests := []struct {
		path  string
		query string
		want  string
	}{
		{"/users", "", "https://api.example.com/users"},
		{"/users", "page=1", "https://api.example.com/users?page=1"},
		{"/data/123", "format=json", "https://api.example.com/data/123?format=json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			url, err := svc.ResolveUpstreamURL(upstream, tt.path, tt.query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if url.String() != tt.want {
				t.Errorf("got %s, want %s", url.String(), tt.want)
			}
		})
	}
}

func TestRouteService_ApplyUpstreamAuth_None(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{AuthType: route.AuthNone}

	headers := svc.ApplyUpstreamAuth(upstream, nil)
	if len(headers) != 0 {
		t.Errorf("expected empty headers, got %v", headers)
	}
}

func TestRouteService_ApplyUpstreamAuth_Header(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		AuthType:   route.AuthHeader,
		AuthHeader: "X-API-Key",
		AuthValue:  "secret123",
	}

	headers := svc.ApplyUpstreamAuth(upstream, nil)
	if headers["X-API-Key"] != "secret123" {
		t.Errorf("got %s, want secret123", headers["X-API-Key"])
	}
}

func TestRouteService_ApplyUpstreamAuth_Bearer(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		AuthType:  route.AuthBearer,
		AuthValue: "token123",
	}

	headers := svc.ApplyUpstreamAuth(upstream, nil)
	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("got %s, want Bearer token123", headers["Authorization"])
	}
}

func TestRouteService_ApplyUpstreamAuth_Basic(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		AuthType:  route.AuthBasic,
		AuthValue: "dXNlcjpwYXNz", // base64 of user:pass
	}

	headers := svc.ApplyUpstreamAuth(upstream, nil)
	if headers["Authorization"] != "Basic dXNlcjpwYXNz" {
		t.Errorf("got %s, want Basic dXNlcjpwYXNz", headers["Authorization"])
	}
}

func TestRouteService_ApplyUpstreamAuth_EnvVar(t *testing.T) {
	os.Setenv("TEST_API_TOKEN", "env-secret")
	defer os.Unsetenv("TEST_API_TOKEN")

	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		AuthType:   route.AuthHeader,
		AuthHeader: "X-Token",
		AuthValue:  "${TEST_API_TOKEN}",
	}

	headers := svc.ApplyUpstreamAuth(upstream, nil)
	if headers["X-Token"] != "env-secret" {
		t.Errorf("got %s, want env-secret", headers["X-Token"])
	}
}

func TestRouteService_ApplyUpstreamAuth_PreserveExisting(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		AuthType:  route.AuthBearer,
		AuthValue: "token",
	}

	existing := map[string]string{"X-Custom": "value"}
	headers := svc.ApplyUpstreamAuth(upstream, existing)

	if headers["X-Custom"] != "value" {
		t.Error("existing header should be preserved")
	}
	if headers["Authorization"] != "Bearer token" {
		t.Error("auth header should be added")
	}
}

func TestRouteService_BuildUpstreamClient(t *testing.T) {
	svc := newTestRouteService(nil, nil)
	upstream := &route.Upstream{
		Timeout:         30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}

	client := svc.BuildUpstreamClient(upstream)
	if client == nil {
		t.Fatal("expected client")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", client.Timeout)
	}
}

func TestRouteService_TestRoute_NoCache(t *testing.T) {
	svc := newTestRouteService(nil, nil)

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/test",
	})

	if result.Error != "Route service not initialized" {
		t.Errorf("got error %q", result.Error)
	}
}

func TestRouteService_TestRoute_NoMatch(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route", PathPattern: "/api/*", MatchType: route.MatchPrefix, Enabled: true},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/other/path",
	})

	if result.Matched {
		t.Error("expected no match")
	}
	if result.MatchReason != "No route matched the request" {
		t.Errorf("got reason %q", result.MatchReason)
	}
}

func TestRouteService_TestRoute_Match(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "API Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "u1",
			Enabled:     true,
		},
	}
	upstreams := []route.Upstream{
		{ID: "u1", Name: "Backend", BaseURL: "https://backend.com", Enabled: true},
	}
	svc := newTestRouteService(routes, upstreams)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/users",
	})

	if !result.Matched {
		t.Error("expected match")
	}
	if result.RouteID != "r1" {
		t.Errorf("got route %s, want r1", result.RouteID)
	}
	if result.UpstreamName != "Backend" {
		t.Errorf("got upstream %s, want Backend", result.UpstreamName)
	}
}

func TestRouteService_TestRoute_ByRouteID(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route 1", PathPattern: "/a/*", MatchType: route.MatchPrefix, Enabled: true},
		{ID: "r2", Name: "Route 2", PathPattern: "/b/*", MatchType: route.MatchPrefix, Enabled: true},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	// Test specific route by ID
	result := svc.TestRoute(app.RouteTestRequest{
		Method:  "GET",
		Path:    "/c/something", // Doesn't match either pattern
		RouteID: "r2",
	})

	if !result.Matched {
		t.Error("expected match by route ID")
	}
	if result.RouteID != "r2" {
		t.Errorf("got route %s, want r2", result.RouteID)
	}
	if result.MatchReason != "Tested directly by route ID" {
		t.Errorf("got reason %q", result.MatchReason)
	}
}

func TestRouteService_TestRoute_NotFoundRouteID(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route 1", PathPattern: "/a/*", MatchType: route.MatchPrefix, Enabled: true},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method:  "GET",
		Path:    "/test",
		RouteID: "nonexistent",
	})

	if result.Matched {
		t.Error("expected no match for nonexistent route")
	}
	if result.Error != "Route not found: nonexistent" {
		t.Errorf("got error %q", result.Error)
	}
}

func TestRouteService_TestRoute_MethodOverride(t *testing.T) {
	routes := []route.Route{
		{
			ID:             "r1",
			Name:           "Route",
			PathPattern:    "/api/*",
			MatchType:      route.MatchPrefix,
			MethodOverride: "POST",
			Enabled:        true,
		},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/data",
	})

	if result.TransformedMethod != "POST" {
		t.Errorf("got method %s, want POST", result.TransformedMethod)
	}
}

func TestRouteService_TestRoute_PathRewrite(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			PathRewrite: `"/v2" + trimPrefix(path, "/api")`,
			Enabled:     true,
		},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/users",
	})

	if result.TransformedPath != `[expr: "/v2" + trimPrefix(path, "/api")]` {
		t.Errorf("got path %s", result.TransformedPath)
	}
}

func TestRouteService_TestRoute_RequestTransform(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			RequestTransform: &route.Transform{
				SetHeaders:    map[string]string{"X-Added": `"value"`},
				DeleteHeaders: []string{"X-Remove"},
			},
			Enabled: true,
		},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/data",
		Headers: map[string]string{
			"X-Keep":   "kept",
			"X-Remove": "removed",
		},
	})

	if result.TransformedHeaders["X-Keep"] != "kept" {
		t.Error("X-Keep should be preserved")
	}
	if _, ok := result.TransformedHeaders["X-Remove"]; ok {
		t.Error("X-Remove should be deleted")
	}
	if result.TransformedHeaders["X-Added"] != `[expr: "value"]` {
		t.Errorf("X-Added = %s", result.TransformedHeaders["X-Added"])
	}
}

func TestRouteService_TestRoute_MeteringExpr(t *testing.T) {
	routes := []route.Route{
		{
			ID:           "r1",
			Name:         "Route",
			PathPattern:  "/api/*",
			MatchType:    route.MatchPrefix,
			MeteringExpr: "respBody.usage.tokens",
			Enabled:      true,
		},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/chat",
	})

	if result.MeteringExpr != "respBody.usage.tokens" {
		t.Errorf("got metering expr %s", result.MeteringExpr)
	}
}

func TestRouteService_TestRoute_DefaultMeteringExpr(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "Route",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			Enabled:     true,
		},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	result := svc.TestRoute(app.RouteTestRequest{
		Method: "GET",
		Path:   "/api/data",
	})

	if result.MeteringExpr != "1" {
		t.Errorf("got metering expr %s, want 1", result.MeteringExpr)
	}
}

func TestRouteService_Reload(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route", PathPattern: "/api/*", MatchType: route.MatchPrefix, Enabled: true},
	}
	svc := newTestRouteService(routes, nil)
	ctx := context.Background()

	err := svc.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// Verify cache was populated
	cached := svc.GetRoutes()
	if len(cached) != 1 {
		t.Errorf("got %d routes, want 1", len(cached))
	}
}

func TestRouteService_OnlyEnabledRoutes(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Enabled", PathPattern: "/a/*", MatchType: route.MatchPrefix, Enabled: true},
		{ID: "r2", Name: "Disabled", PathPattern: "/b/*", MatchType: route.MatchPrefix, Enabled: false},
	}
	svc := newTestRouteService(routes, nil)
	_ = svc.Start(context.Background())
	defer svc.Stop()

	cached := svc.GetRoutes()
	if len(cached) != 1 {
		t.Errorf("got %d routes, want 1 (only enabled)", len(cached))
	}
	if cached[0].Name != "Enabled" {
		t.Errorf("got %s, want Enabled", cached[0].Name)
	}
}
