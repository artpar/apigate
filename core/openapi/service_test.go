package openapi

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/domain/route"
	"github.com/rs/zerolog"
)

// mockRouteStore implements ports.RouteStore for testing
type mockRouteStore struct {
	routes []route.Route
	err    error
}

func (m *mockRouteStore) List(ctx context.Context) ([]route.Route, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.routes, nil
}

func (m *mockRouteStore) ListEnabled(ctx context.Context) ([]route.Route, error) {
	if m.err != nil {
		return nil, m.err
	}
	var enabled []route.Route
	for _, r := range m.routes {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled, nil
}

func (m *mockRouteStore) Get(ctx context.Context, id string) (route.Route, error) {
	for _, r := range m.routes {
		if r.ID == id {
			return r, nil
		}
	}
	return route.Route{}, nil
}

func (m *mockRouteStore) Create(ctx context.Context, r route.Route) error {
	m.routes = append(m.routes, r)
	return nil
}

func (m *mockRouteStore) Update(ctx context.Context, r route.Route) error {
	for i, existing := range m.routes {
		if existing.ID == r.ID {
			m.routes[i] = r
			return nil
		}
	}
	return nil
}

func (m *mockRouteStore) Delete(ctx context.Context, id string) error {
	for i, r := range m.routes {
		if r.ID == id {
			m.routes = append(m.routes[:i], m.routes[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockUpstreamStore implements ports.UpstreamStore for testing
type mockUpstreamStore struct {
	upstreams []route.Upstream
	err       error
}

func (m *mockUpstreamStore) List(ctx context.Context) ([]route.Upstream, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.upstreams, nil
}

func (m *mockUpstreamStore) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	if m.err != nil {
		return nil, m.err
	}
	var enabled []route.Upstream
	for _, u := range m.upstreams {
		if u.Enabled {
			enabled = append(enabled, u)
		}
	}
	return enabled, nil
}

func (m *mockUpstreamStore) Get(ctx context.Context, id string) (route.Upstream, error) {
	for _, u := range m.upstreams {
		if u.ID == id {
			return u, nil
		}
	}
	return route.Upstream{}, nil
}

func (m *mockUpstreamStore) Create(ctx context.Context, u route.Upstream) error {
	m.upstreams = append(m.upstreams, u)
	return nil
}

func (m *mockUpstreamStore) Update(ctx context.Context, u route.Upstream) error {
	for i, existing := range m.upstreams {
		if existing.ID == u.ID {
			m.upstreams[i] = u
			return nil
		}
	}
	return nil
}

func (m *mockUpstreamStore) Delete(ctx context.Context, id string) error {
	for i, u := range m.upstreams {
		if u.ID == id {
			m.upstreams = append(m.upstreams[:i], m.upstreams[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestNewService(t *testing.T) {
	routeStore := &mockRouteStore{}
	upstreamStore := &mockUpstreamStore{}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: upstreamStore,
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.appName != "TestAPI" {
		t.Errorf("appName = %s, want TestAPI", svc.appName)
	}
}

func TestNewService_DefaultAppName(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "",
		Logger:        zerolog.Nop(),
	})

	if svc.appName != "APIGate" {
		t.Errorf("appName = %s, want APIGate (default)", svc.appName)
	}
}

func TestGetUnifiedSpec_EmptyStores(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	spec := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	if spec == nil {
		t.Fatal("GetUnifiedSpec returned nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI version = %s, want 3.0.3", spec.OpenAPI)
	}
	if spec.Info.Title != "TestAPI API" {
		t.Errorf("Title = %s, want TestAPI API", spec.Info.Title)
	}
	if len(spec.Servers) == 0 || spec.Servers[0].URL != "http://example.com" {
		t.Error("Server URL not set correctly")
	}
}

func TestGetUnifiedSpec_WithRoutes(t *testing.T) {
	routeStore := &mockRouteStore{
		routes: []route.Route{
			{
				ID:          "r1",
				PathPattern: "/api/users",
				MatchType:   route.MatchExact,
				Methods:     []string{"GET"},
				UpstreamID:  "u1",
				Enabled:     true,
			},
		},
	}
	upstreamStore := &mockUpstreamStore{
		upstreams: []route.Upstream{
			{
				ID:      "u1",
				Name:    "Users API",
				BaseURL: "http://users.internal",
				Enabled: true,
			},
		},
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: upstreamStore,
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	spec := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	if spec == nil {
		t.Fatal("GetUnifiedSpec returned nil")
	}
	if len(spec.Paths) == 0 {
		t.Error("Expected paths in spec")
	}
	if _, ok := spec.Paths["/api/users"]; !ok {
		t.Error("Expected /api/users path in spec")
	}
}

func TestGetUnifiedSpec_WithModules(t *testing.T) {
	// Test that module getter is called and doesn't cause errors
	// Even with minimal data, the service should still work
	moduleGetterCalled := false
	moduleGetter := func() map[string]convention.Derived {
		moduleGetterCalled = true
		return map[string]convention.Derived{
			"test": {
				Plural: "tests",
				Table:  "tests",
				Fields: []convention.DerivedField{
					{Name: "id", SQLType: "TEXT"},
					{Name: "name", SQLType: "TEXT"},
				},
			},
		}
	}

	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		ModuleGetter:  moduleGetter,
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	spec := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	if spec == nil {
		t.Fatal("GetUnifiedSpec returned nil")
	}
	if !moduleGetterCalled {
		t.Error("Expected module getter to be called")
	}
	// Spec should be valid even if module data is minimal
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI = %s, want 3.0.3", spec.OpenAPI)
	}
}

func TestGetUnifiedSpec_Caching(t *testing.T) {
	routeStore := &mockRouteStore{
		routes: []route.Route{
			{
				ID:          "r1",
				PathPattern: "/api/test",
				MatchType:   route.MatchExact,
				Methods:     []string{"GET"},
				Enabled:     true,
			},
		},
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	// First call
	spec1 := svc.GetUnifiedSpec(context.Background(), "http://example.com")
	if spec1 == nil {
		t.Fatal("First GetUnifiedSpec returned nil")
	}

	// Second call should use cache (we verify this by checking cache is populated)
	spec2 := svc.GetUnifiedSpec(context.Background(), "http://example.com")
	if spec2 == nil {
		t.Fatal("Second GetUnifiedSpec returned nil")
	}

	// Verify cache is populated
	if svc.cache.Load() == nil {
		t.Error("Cache should be populated")
	}
}

func TestGetUnifiedSpec_DifferentServerURLs(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	spec1 := svc.GetUnifiedSpec(context.Background(), "http://server1.com")
	spec2 := svc.GetUnifiedSpec(context.Background(), "http://server2.com")

	if spec1.Servers[0].URL != "http://server1.com" {
		t.Errorf("spec1 server URL = %s, want http://server1.com", spec1.Servers[0].URL)
	}
	if spec2.Servers[0].URL != "http://server2.com" {
		t.Errorf("spec2 server URL = %s, want http://server2.com", spec2.Servers[0].URL)
	}
}

func TestInvalidateCache(t *testing.T) {
	routeStore := &mockRouteStore{
		routes: []route.Route{
			{ID: "r1", PathPattern: "/api/test", MatchType: route.MatchExact, Methods: []string{"GET"}, Enabled: true},
		},
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	// Generate and cache spec
	svc.GetUnifiedSpec(context.Background(), "http://example.com")

	// Modify routes
	routeStore.routes = append(routeStore.routes, route.Route{
		ID:          "r2",
		PathPattern: "/api/new",
		MatchType:   route.MatchExact,
		Methods:     []string{"POST"},
		Enabled:     true,
	})

	// Invalidate cache
	svc.InvalidateCache()

	// Get new spec
	spec := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	// Should contain new route
	if _, ok := spec.Paths["/api/new"]; !ok {
		t.Error("Expected /api/new path after cache invalidation")
	}
}

func TestComputeDataHash_Deterministic(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{
		{ID: "r1", PathPattern: "/api/users", Enabled: true},
		{ID: "r2", PathPattern: "/api/posts", Enabled: false},
	}
	upstreams := map[string]route.Upstream{
		"u1": {ID: "u1", Name: "API 1", AuthType: route.AuthBearer, Enabled: true},
		"u2": {ID: "u2", Name: "API 2", AuthType: route.AuthNone, Enabled: true},
	}

	hash1 := svc.computeDataHash(routes, upstreams)
	hash2 := svc.computeDataHash(routes, upstreams)

	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %s != %s", hash1, hash2)
	}
}

func TestComputeDataHash_DifferentForDifferentData(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	routes1 := []route.Route{{ID: "r1", PathPattern: "/api/users", Enabled: true}}
	routes2 := []route.Route{{ID: "r1", PathPattern: "/api/posts", Enabled: true}}
	upstreams := map[string]route.Upstream{}

	hash1 := svc.computeDataHash(routes1, upstreams)
	hash2 := svc.computeDataHash(routes2, upstreams)

	if hash1 == hash2 {
		t.Error("Hash should be different for different data")
	}
}

func TestComputeDataHash_EnabledFlag(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	routes1 := []route.Route{{ID: "r1", PathPattern: "/api/users", Enabled: true}}
	routes2 := []route.Route{{ID: "r1", PathPattern: "/api/users", Enabled: false}}
	upstreams := map[string]route.Upstream{}

	hash1 := svc.computeDataHash(routes1, upstreams)
	hash2 := svc.computeDataHash(routes2, upstreams)

	if hash1 == hash2 {
		t.Error("Hash should differ based on Enabled flag")
	}
}

func TestComputeDataHash_UpstreamAuthType(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{}
	upstreams1 := map[string]route.Upstream{"u1": {ID: "u1", Name: "API", AuthType: route.AuthBearer, Enabled: true}}
	upstreams2 := map[string]route.Upstream{"u1": {ID: "u1", Name: "API", AuthType: route.AuthBasic, Enabled: true}}

	hash1 := svc.computeDataHash(routes, upstreams1)
	hash2 := svc.computeDataHash(routes, upstreams2)

	if hash1 == hash2 {
		t.Error("Hash should differ based on AuthType")
	}
}

func TestGenerateSpec_DefaultSecuritySchemes(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{}
	upstreams := map[string]route.Upstream{}

	spec := svc.generateSpec(routes, upstreams, "http://example.com")

	// Check default security schemes
	if _, ok := spec.Components.SecuritySchemes["apiKey"]; !ok {
		t.Error("Missing default apiKey security scheme")
	}
	if _, ok := spec.Components.SecuritySchemes["bearerAuth"]; !ok {
		t.Error("Missing default bearerAuth security scheme")
	}
}

func TestGenerateSpec_MergesRouteSecuritySchemes(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{
		{
			ID:          "r1",
			PathPattern: "/api/test",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET"},
			UpstreamID:  "u1",
			Enabled:     true,
		},
	}
	upstreams := map[string]route.Upstream{
		"u1": {ID: "u1", Name: "Custom API", BaseURL: "http://api.internal", AuthType: route.AuthHeader, AuthHeader: "X-Custom-Key", Enabled: true},
	}

	spec := svc.generateSpec(routes, upstreams, "http://example.com")

	// Should have route's security scheme (format: sanitized_name + "_header")
	// "Custom API" sanitizes to "custom_api" + "_header" = "custom_api_header"
	found := false
	for name := range spec.Components.SecuritySchemes {
		if name == "custom_api_header" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected route-specific security scheme custom_api_header, got schemes: %v", spec.Components.SecuritySchemes)
	}
}

func TestGenerateSpec_TagsFromUpstreams(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{
		{ID: "r1", PathPattern: "/api/test", MatchType: route.MatchExact, Methods: []string{"GET"}, UpstreamID: "u1", Enabled: true},
	}
	upstreams := map[string]route.Upstream{
		"u1": {ID: "u1", Name: "Users Service", Description: "User management API", Enabled: true},
	}

	spec := svc.generateSpec(routes, upstreams, "http://example.com")

	// Should have tag from upstream
	found := false
	for _, tag := range spec.Tags {
		if tag.Name == "Users Service" {
			found = true
			if tag.Description != "User management API" {
				t.Errorf("Tag description = %s, want User management API", tag.Description)
			}
			break
		}
	}
	if !found {
		t.Error("Expected tag from upstream")
	}
}

func TestGenerateSpec_PathMerging(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	// Two routes with same path but different methods
	routes := []route.Route{
		{ID: "r1", PathPattern: "/api/items", MatchType: route.MatchExact, Methods: []string{"GET"}, UpstreamID: "u1", Enabled: true},
		{ID: "r2", PathPattern: "/api/items", MatchType: route.MatchExact, Methods: []string{"POST"}, UpstreamID: "u1", Enabled: true},
	}
	upstreams := map[string]route.Upstream{
		"u1": {ID: "u1", Name: "Items API", Enabled: true},
	}

	spec := svc.generateSpec(routes, upstreams, "http://example.com")

	pathItem, ok := spec.Paths["/api/items"]
	if !ok {
		t.Fatal("Expected /api/items path")
	}

	if pathItem.Get == nil {
		t.Error("Expected GET operation")
	}
	if pathItem.Post == nil {
		t.Error("Expected POST operation")
	}
}

func TestGenerateSpec_TagsSorted(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	routes := []route.Route{
		{ID: "r1", PathPattern: "/z/api", MatchType: route.MatchExact, Methods: []string{"GET"}, UpstreamID: "u1", Enabled: true},
		{ID: "r2", PathPattern: "/a/api", MatchType: route.MatchExact, Methods: []string{"GET"}, UpstreamID: "u2", Enabled: true},
		{ID: "r3", PathPattern: "/m/api", MatchType: route.MatchExact, Methods: []string{"GET"}, UpstreamID: "u3", Enabled: true},
	}
	upstreams := map[string]route.Upstream{
		"u1": {ID: "u1", Name: "Zebra API", Enabled: true},
		"u2": {ID: "u2", Name: "Apple API", Enabled: true},
		"u3": {ID: "u3", Name: "Mango API", Enabled: true},
	}

	spec := svc.generateSpec(routes, upstreams, "http://example.com")

	// Tags should be sorted alphabetically
	for i := 1; i < len(spec.Tags); i++ {
		if spec.Tags[i-1].Name > spec.Tags[i].Name {
			t.Errorf("Tags not sorted: %s > %s", spec.Tags[i-1].Name, spec.Tags[i].Name)
		}
	}
}

func TestCloneSpecWithServer(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	original := &Spec{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Test API"},
		Servers: []Server{{URL: "http://original.com", Description: "Original"}},
		Paths: map[string]PathItem{
			"/api/test": {
				Get: &Operation{Summary: "Test endpoint"},
			},
		},
	}

	cloned := svc.cloneSpecWithServer(original, "http://new-server.com")

	// Server URL should be updated
	if cloned.Servers[0].URL != "http://new-server.com" {
		t.Errorf("Server URL = %s, want http://new-server.com", cloned.Servers[0].URL)
	}

	// Original should be unchanged
	if original.Servers[0].URL != "http://original.com" {
		t.Error("Original spec was modified")
	}

	// Other fields should be preserved
	if cloned.Info.Title != "Test API" {
		t.Errorf("Title = %s, want Test API", cloned.Info.Title)
	}
	if _, ok := cloned.Paths["/api/test"]; !ok {
		t.Error("Paths not preserved in clone")
	}
}

func TestCloneSpecWithServer_EmptyServers(t *testing.T) {
	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{},
		UpstreamStore: &mockUpstreamStore{},
		Logger:        zerolog.Nop(),
	})

	original := &Spec{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Test API"},
		Servers: []Server{}, // Empty servers
	}

	cloned := svc.cloneSpecWithServer(original, "http://new-server.com")

	// Should add server
	if len(cloned.Servers) == 0 {
		t.Fatal("Expected servers to be added")
	}
	if cloned.Servers[0].URL != "http://new-server.com" {
		t.Errorf("Server URL = %s, want http://new-server.com", cloned.Servers[0].URL)
	}
}

func TestLoadData_RouteError(t *testing.T) {
	routeStore := &mockRouteStore{
		err: context.DeadlineExceeded,
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: &mockUpstreamStore{upstreams: []route.Upstream{{ID: "u1"}}},
		Logger:        zerolog.Nop(),
	})

	routes, upstreams := svc.loadData(context.Background())

	// Should return nil routes on error
	if routes != nil {
		t.Error("Expected nil routes on error")
	}
	// Upstreams should still be loaded
	if len(upstreams) != 1 {
		t.Errorf("Expected 1 upstream, got %d", len(upstreams))
	}
}

func TestLoadData_UpstreamError(t *testing.T) {
	upstreamStore := &mockUpstreamStore{
		err: context.DeadlineExceeded,
	}

	svc := NewService(ServiceConfig{
		RouteStore:    &mockRouteStore{routes: []route.Route{{ID: "r1"}}},
		UpstreamStore: upstreamStore,
		Logger:        zerolog.Nop(),
	})

	routes, upstreams := svc.loadData(context.Background())

	// Routes should still be loaded
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}
	// Should return empty upstreams map on error
	if len(upstreams) != 0 {
		t.Error("Expected empty upstreams on error")
	}
}

func TestGetUnifiedSpec_CacheRefreshWithSameData(t *testing.T) {
	routeStore := &mockRouteStore{
		routes: []route.Route{
			{ID: "r1", PathPattern: "/api/test", Enabled: true},
		},
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: &mockUpstreamStore{},
		AppName:       "TestAPI",
		Logger:        zerolog.Nop(),
	})

	// First call caches
	spec1 := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	// Force cache expiration by setting timestamp in the past
	cached := svc.cache.Load()
	if cached != nil {
		svc.cache.Store(&cachedSpec{
			spec:        cached.spec,
			generatedAt: time.Now().Add(-35 * time.Second), // Expired
			dataHash:    cached.dataHash,
		})
	}

	// Second call should refresh but reuse spec (same hash)
	spec2 := svc.GetUnifiedSpec(context.Background(), "http://example.com")

	if spec1 == nil || spec2 == nil {
		t.Fatal("Specs should not be nil")
	}
}

func TestService_Integration(t *testing.T) {
	// Full integration test with routes, upstreams, and modules
	routeStore := &mockRouteStore{
		routes: []route.Route{
			{
				ID:          "r1",
				PathPattern: "/v1/users",
				MatchType:   route.MatchPrefix,
				Methods:     []string{"GET", "POST"},
				UpstreamID:  "u1",
				Protocol:    route.ProtocolHTTP,
				Description: "User management endpoints",
				Enabled:     true,
				Priority:    100,
			},
			{
				ID:          "r2",
				PathPattern: "/v1/products/*",
				MatchType:   route.MatchPrefix,
				Methods:     []string{"GET"},
				UpstreamID:  "u2",
				Protocol:    route.ProtocolHTTPStream,
				Description: "Product catalog",
				Enabled:     true,
				Priority:    90,
			},
		},
	}
	upstreamStore := &mockUpstreamStore{
		upstreams: []route.Upstream{
			{
				ID:          "u1",
				Name:        "User Service",
				Description: "Internal user management service",
				BaseURL:     "http://users.internal:8080",
				AuthType:    route.AuthBearer,
				AuthHeader:  "Authorization",
				Enabled:     true,
			},
			{
				ID:          "u2",
				Name:        "Product Service",
				Description: "Product catalog service",
				BaseURL:     "https://products.internal:443",
				AuthType:    route.AuthHeader,
				AuthHeader:  "X-Product-Key",
				Enabled:     true,
			},
		},
	}

	svc := NewService(ServiceConfig{
		RouteStore:    routeStore,
		UpstreamStore: upstreamStore,
		AppName:       "MyApp",
		Logger:        zerolog.Nop(),
	})

	spec := svc.GetUnifiedSpec(context.Background(), "https://api.myapp.com")

	// Verify basic spec properties
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI = %s, want 3.0.3", spec.OpenAPI)
	}
	if spec.Info.Title != "MyApp API" {
		t.Errorf("Title = %s, want MyApp API", spec.Info.Title)
	}

	// Verify server
	if len(spec.Servers) == 0 || spec.Servers[0].URL != "https://api.myapp.com" {
		t.Error("Server URL not set correctly")
	}

	// Verify paths exist
	if len(spec.Paths) < 2 {
		t.Errorf("Expected at least 2 paths, got %d", len(spec.Paths))
	}

	// Verify security schemes
	if len(spec.Components.SecuritySchemes) < 2 {
		t.Errorf("Expected at least 2 security schemes, got %d", len(spec.Components.SecuritySchemes))
	}

	// Verify tags
	if len(spec.Tags) < 2 {
		t.Errorf("Expected at least 2 tags, got %d", len(spec.Tags))
	}

	// Verify cache is populated
	if svc.cache.Load() == nil {
		t.Error("Cache should be populated")
	}
}
