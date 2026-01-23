package route_test

import (
	"testing"

	"github.com/artpar/apigate/domain/route"
)

func TestMatcher_ExactMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "exact-users",
			PathPattern: "/api/users",
			MatchType:   route.MatchExact,
			UpstreamID:  "up1",
			Enabled:     true,
		},
		{
			ID:          "r2",
			Name:        "exact-posts",
			PathPattern: "/api/posts",
			MatchType:   route.MatchExact,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		method   string
		wantID   string
		wantNil  bool
	}{
		{"exact match /api/users", "/api/users", "GET", "r1", false},
		{"exact match /api/posts", "/api/posts", "GET", "r2", false},
		{"no match /api/users/123", "/api/users/123", "GET", "", true},
		{"no match /api", "/api", "GET", "", true},
		{"no match /api/other", "/api/other", "GET", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.method, tt.path, nil)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_PrefixMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "api-prefix",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up1",
			Enabled:     true,
		},
		{
			ID:          "r2",
			Name:        "v2-prefix",
			PathPattern: "/v2/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up2",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantID   string
		wantNil  bool
	}{
		{"prefix /api/users", "/api/users", "r1", false},
		{"prefix /api/users/123", "/api/users/123", "r1", false},
		{"prefix /api/", "/api/", "r1", false},
		{"prefix /v2/data", "/v2/data", "r2", false},
		{"no match /other", "/other", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match("GET", tt.path, nil)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_RegexMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "users-id",
			PathPattern: "/users/{id}",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up1",
			Enabled:     true,
		},
		{
			ID:          "r2",
			Name:        "posts-id-comments",
			PathPattern: "/posts/{postId}/comments/{commentId}",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantID     string
		wantNil    bool
		wantParams map[string]string
	}{
		{
			"users with id",
			"/users/123",
			"r1",
			false,
			map[string]string{"id": "123"},
		},
		{
			"users with uuid",
			"/users/abc-def-123",
			"r1",
			false,
			map[string]string{"id": "abc-def-123"},
		},
		{
			"posts with nested params",
			"/posts/456/comments/789",
			"r2",
			false,
			map[string]string{"postId": "456", "commentId": "789"},
		},
		{"no match /users", "/users", "", true, nil},
		{"no match /users/", "/users/", "", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match("GET", tt.path, nil)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
			for k, v := range tt.wantParams {
				if result.PathParams[k] != v {
					t.Errorf("param %s = %s, want %s", k, result.PathParams[k], v)
				}
			}
		})
	}
}

func TestMatcher_MethodFiltering(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "get-only",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET"},
			UpstreamID:  "up1",
			Enabled:     true,
		},
		{
			ID:          "r2",
			Name:        "post-put",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Methods:     []string{"POST", "PUT"},
			UpstreamID:  "up1",
			Enabled:     true,
			Priority:    -1, // Lower priority
		},
		{
			ID:          "r3",
			Name:        "all-methods",
			PathPattern: "/api/all",
			MatchType:   route.MatchExact,
			Methods:     []string{}, // Empty = all methods
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		method  string
		path    string
		wantID  string
		wantNil bool
	}{
		{"GET /api/data matches r1", "GET", "/api/data", "r1", false},
		{"POST /api/data matches r2", "POST", "/api/data", "r2", false},
		{"PUT /api/data matches r2", "PUT", "/api/data", "r2", false},
		{"DELETE /api/data no match", "DELETE", "/api/data", "", true},
		{"GET /api/all matches r3", "GET", "/api/all", "r3", false},
		{"POST /api/all matches r3", "POST", "/api/all", "r3", false},
		{"DELETE /api/all matches r3", "DELETE", "/api/all", "r3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.method, tt.path, nil)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_HeaderMatching(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "json-only",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Headers: []route.HeaderMatch{
				{Name: "Content-Type", Value: "application/json", Required: true},
			},
			UpstreamID: "up1",
			Enabled:    true,
			Priority:   10,
		},
		{
			ID:          "r2",
			Name:        "xml-only",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Headers: []route.HeaderMatch{
				{Name: "Content-Type", Value: "application/xml", Required: true},
			},
			UpstreamID: "up1",
			Enabled:    true,
			Priority:   10,
		},
		{
			ID:          "r3",
			Name:        "fallback",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			UpstreamID:  "up1",
			Enabled:     true,
			Priority:    0,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		headers map[string]string
		wantID  string
	}{
		{
			"json header matches r1",
			map[string]string{"Content-Type": "application/json"},
			"r1",
		},
		{
			"xml header matches r2",
			map[string]string{"Content-Type": "application/xml"},
			"r2",
		},
		{
			"no header matches fallback",
			nil,
			"r3",
		},
		{
			"other content-type matches fallback",
			map[string]string{"Content-Type": "text/plain"},
			"r3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match("GET", "/api/data", tt.headers)
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_HeaderRegex(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "version-header",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Headers: []route.HeaderMatch{
				{Name: "X-API-Version", Value: `^v[0-9]+$`, IsRegex: true, Required: true},
			},
			UpstreamID: "up1",
			Enabled:    true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		headers map[string]string
		wantNil bool
	}{
		{"v1 matches", map[string]string{"X-API-Version": "v1"}, false},
		{"v123 matches", map[string]string{"X-API-Version": "v123"}, false},
		{"invalid version no match", map[string]string{"X-API-Version": "version1"}, true},
		{"missing header no match", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match("GET", "/api/data", tt.headers)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
		})
	}
}

func TestMatcher_Priority(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "low-priority",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up1",
			Enabled:     true,
			Priority:    0,
		},
		{
			ID:          "r2",
			Name:        "high-priority",
			PathPattern: "/api/special",
			MatchType:   route.MatchExact,
			UpstreamID:  "up2",
			Enabled:     true,
			Priority:    100,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// /api/special should match high-priority route
	result := matcher.Match("GET", "/api/special", nil)
	if result == nil {
		t.Fatal("expected match")
	}
	if result.Route.ID != "r2" {
		t.Errorf("expected r2 (high priority), got %s", result.Route.ID)
	}

	// /api/other should match low-priority route
	result = matcher.Match("GET", "/api/other", nil)
	if result == nil {
		t.Fatal("expected match")
	}
	if result.Route.ID != "r1" {
		t.Errorf("expected r1 (low priority), got %s", result.Route.ID)
	}
}

func TestMatcher_DisabledRoutes(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "disabled",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			UpstreamID:  "up1",
			Enabled:     false, // Disabled
		},
		{
			ID:          "r2",
			Name:        "enabled",
			PathPattern: "/api/other",
			MatchType:   route.MatchExact,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Disabled route should not match
	result := matcher.Match("GET", "/api/data", nil)
	if result != nil {
		t.Errorf("expected nil for disabled route, got %s", result.Route.ID)
	}

	// Enabled route should match
	result = matcher.Match("GET", "/api/other", nil)
	if result == nil {
		t.Fatal("expected match for enabled route")
	}
}

func TestMatcher_EmptyRoutes(t *testing.T) {
	matcher, err := route.NewMatcher(nil)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	result := matcher.Match("GET", "/anything", nil)
	if result != nil {
		t.Error("expected nil for empty routes")
	}
}

func TestMatcher_MatchTypePriority(t *testing.T) {
	// When routes have same priority, exact should win over prefix over regex
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "prefix",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up1",
			Enabled:     true,
			Priority:    0,
		},
		{
			ID:          "r2",
			Name:        "exact",
			PathPattern: "/api/users",
			MatchType:   route.MatchExact,
			UpstreamID:  "up2",
			Enabled:     true,
			Priority:    0,
		},
		{
			ID:          "r3",
			Name:        "regex",
			PathPattern: "/api/users",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up3",
			Enabled:     true,
			Priority:    0,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	result := matcher.Match("GET", "/api/users", nil)
	if result == nil {
		t.Fatal("expected match")
	}
	// Exact match should win
	if result.Route.ID != "r2" {
		t.Errorf("expected r2 (exact), got %s", result.Route.ID)
	}
}

func TestFindByID(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Name: "Route 1"},
		{ID: "r2", Name: "Route 2"},
		{ID: "r3", Name: "Route 3"},
	}

	r := route.FindByID(routes, "r2")
	if r == nil {
		t.Fatal("expected to find route r2")
	}
	if r.Name != "Route 2" {
		t.Errorf("name = %s, want Route 2", r.Name)
	}

	r = route.FindByID(routes, "nonexistent")
	if r != nil {
		t.Error("expected nil for nonexistent route")
	}
}

func TestFindUpstreamByID(t *testing.T) {
	upstreams := []route.Upstream{
		{ID: "up1", Name: "Upstream 1"},
		{ID: "up2", Name: "Upstream 2"},
	}

	u := route.FindUpstreamByID(upstreams, "up2")
	if u == nil {
		t.Fatal("expected to find upstream up2")
	}
	if u.Name != "Upstream 2" {
		t.Errorf("name = %s, want Upstream 2", u.Name)
	}

	u = route.FindUpstreamByID(upstreams, "nonexistent")
	if u != nil {
		t.Error("expected nil for nonexistent upstream")
	}
}

func TestFilterEnabled(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Enabled: true},
		{ID: "r2", Enabled: false},
		{ID: "r3", Enabled: true},
		{ID: "r4", Enabled: false},
	}

	enabled := route.FilterEnabled(routes)
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled routes, got %d", len(enabled))
	}

	ids := make(map[string]bool)
	for _, r := range enabled {
		ids[r.ID] = true
	}
	if !ids["r1"] || !ids["r3"] {
		t.Errorf("expected r1 and r3, got %v", ids)
	}
}

func TestSortByPriority(t *testing.T) {
	routes := []route.Route{
		{ID: "r1", Priority: 5},
		{ID: "r2", Priority: 10},
		{ID: "r3", Priority: 1},
	}

	route.SortByPriority(routes)

	if routes[0].ID != "r2" || routes[1].ID != "r1" || routes[2].ID != "r3" {
		t.Errorf("unexpected order: %s, %s, %s", routes[0].ID, routes[1].ID, routes[2].ID)
	}
}

// Tests for route.go functions

func TestNewRoute(t *testing.T) {
	r := route.NewRoute("id1", "test-route", "/api/*", "upstream1")

	if r.ID != "id1" {
		t.Errorf("ID = %s, want id1", r.ID)
	}
	if r.Name != "test-route" {
		t.Errorf("Name = %s, want test-route", r.Name)
	}
	if r.PathPattern != "/api/*" {
		t.Errorf("PathPattern = %s, want /api/*", r.PathPattern)
	}
	if r.UpstreamID != "upstream1" {
		t.Errorf("UpstreamID = %s, want upstream1", r.UpstreamID)
	}
	if r.MatchType != route.MatchPrefix {
		t.Errorf("MatchType = %s, want prefix", r.MatchType)
	}
	if r.MeteringExpr != "1" {
		t.Errorf("MeteringExpr = %s, want 1", r.MeteringExpr)
	}
	if r.MeteringMode != "request" {
		t.Errorf("MeteringMode = %s, want request", r.MeteringMode)
	}
	if r.Protocol != route.ProtocolHTTP {
		t.Errorf("Protocol = %s, want http", r.Protocol)
	}
	if r.Priority != 0 {
		t.Errorf("Priority = %d, want 0", r.Priority)
	}
	if !r.Enabled {
		t.Error("Enabled should be true")
	}
	if r.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if r.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestNewUpstream(t *testing.T) {
	u := route.NewUpstream("up1", "test-upstream", "https://api.example.com")

	if u.ID != "up1" {
		t.Errorf("ID = %s, want up1", u.ID)
	}
	if u.Name != "test-upstream" {
		t.Errorf("Name = %s, want test-upstream", u.Name)
	}
	if u.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %s, want https://api.example.com", u.BaseURL)
	}
	if u.Timeout.Seconds() != 30 {
		t.Errorf("Timeout = %v, want 30s", u.Timeout)
	}
	if u.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", u.MaxIdleConns)
	}
	if u.IdleConnTimeout.Seconds() != 90 {
		t.Errorf("IdleConnTimeout = %v, want 90s", u.IdleConnTimeout)
	}
	if u.AuthType != route.AuthNone {
		t.Errorf("AuthType = %s, want none", u.AuthType)
	}
	if !u.Enabled {
		t.Error("Enabled should be true")
	}
	if u.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if u.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestRoute_WithRequestTransform(t *testing.T) {
	r := route.NewRoute("id1", "test", "/api/*", "up1")
	originalUpdatedAt := r.UpdatedAt

	transform := &route.Transform{
		SetHeaders:    map[string]string{"X-Custom": "value"},
		DeleteHeaders: []string{"X-Remove"},
	}

	r2 := r.WithRequestTransform(transform)

	if r2.RequestTransform != transform {
		t.Error("RequestTransform not set correctly")
	}
	if r2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if r.RequestTransform != nil {
		t.Error("Original route should not be modified")
	}
}

func TestRoute_WithResponseTransform(t *testing.T) {
	r := route.NewRoute("id1", "test", "/api/*", "up1")
	originalUpdatedAt := r.UpdatedAt

	transform := &route.Transform{
		SetHeaders: map[string]string{"X-Response": "value"},
		BodyExpr:   `{"processed": true}`,
	}

	r2 := r.WithResponseTransform(transform)

	if r2.ResponseTransform != transform {
		t.Error("ResponseTransform not set correctly")
	}
	if r2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if r.ResponseTransform != nil {
		t.Error("Original route should not be modified")
	}
}

func TestRoute_WithMeteringExpr(t *testing.T) {
	r := route.NewRoute("id1", "test", "/api/*", "up1")
	originalUpdatedAt := r.UpdatedAt

	r2 := r.WithMeteringExpr("response.usage.tokens")

	if r2.MeteringExpr != "response.usage.tokens" {
		t.Errorf("MeteringExpr = %s, want response.usage.tokens", r2.MeteringExpr)
	}
	if r2.MeteringMode != "custom" {
		t.Errorf("MeteringMode = %s, want custom", r2.MeteringMode)
	}
	if r2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if r.MeteringExpr != "1" {
		t.Error("Original route should not be modified")
	}
}

func TestRoute_WithProtocol(t *testing.T) {
	r := route.NewRoute("id1", "test", "/api/*", "up1")
	originalUpdatedAt := r.UpdatedAt

	r2 := r.WithProtocol(route.ProtocolSSE)

	if r2.Protocol != route.ProtocolSSE {
		t.Errorf("Protocol = %s, want sse", r2.Protocol)
	}
	if r2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if r.Protocol != route.ProtocolHTTP {
		t.Error("Original route should not be modified")
	}

	// Test other protocols
	r3 := r.WithProtocol(route.ProtocolHTTPStream)
	if r3.Protocol != route.ProtocolHTTPStream {
		t.Errorf("Protocol = %s, want http_stream", r3.Protocol)
	}

	r4 := r.WithProtocol(route.ProtocolWebSocket)
	if r4.Protocol != route.ProtocolWebSocket {
		t.Errorf("Protocol = %s, want websocket", r4.Protocol)
	}
}

func TestUpstream_WithAuth(t *testing.T) {
	u := route.NewUpstream("up1", "test", "https://api.example.com")
	originalUpdatedAt := u.UpdatedAt

	u2 := u.WithAuth(route.AuthBearer, "", "my-token")

	if u2.AuthType != route.AuthBearer {
		t.Errorf("AuthType = %s, want bearer", u2.AuthType)
	}
	if u2.AuthValue != "my-token" {
		t.Errorf("AuthValue = %s, want my-token", u2.AuthValue)
	}
	if u2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if u.AuthType != route.AuthNone {
		t.Error("Original upstream should not be modified")
	}

	// Test other auth types
	u3 := u.WithAuth(route.AuthHeader, "X-API-Key", "secret-key")
	if u3.AuthType != route.AuthHeader {
		t.Errorf("AuthType = %s, want header", u3.AuthType)
	}
	if u3.AuthHeader != "X-API-Key" {
		t.Errorf("AuthHeader = %s, want X-API-Key", u3.AuthHeader)
	}

	u4 := u.WithAuth(route.AuthBasic, "", "user:pass")
	if u4.AuthType != route.AuthBasic {
		t.Errorf("AuthType = %s, want basic", u4.AuthType)
	}
}

func TestRoute_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		route    route.Route
		expected bool
	}{
		{
			"valid route",
			route.Route{ID: "id1", Name: "name", PathPattern: "/api/*", UpstreamID: "up1"},
			true,
		},
		{
			"missing ID",
			route.Route{ID: "", Name: "name", PathPattern: "/api/*", UpstreamID: "up1"},
			false,
		},
		{
			"missing Name",
			route.Route{ID: "id1", Name: "", PathPattern: "/api/*", UpstreamID: "up1"},
			false,
		},
		{
			"missing PathPattern",
			route.Route{ID: "id1", Name: "name", PathPattern: "", UpstreamID: "up1"},
			false,
		},
		{
			"missing UpstreamID",
			route.Route{ID: "id1", Name: "name", PathPattern: "/api/*", UpstreamID: ""},
			false,
		},
		{
			"all fields empty",
			route.Route{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.route.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUpstream_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		upstream route.Upstream
		expected bool
	}{
		{
			"valid upstream",
			route.Upstream{ID: "up1", Name: "name", BaseURL: "https://api.example.com"},
			true,
		},
		{
			"missing ID",
			route.Upstream{ID: "", Name: "name", BaseURL: "https://api.example.com"},
			false,
		},
		{
			"missing Name",
			route.Upstream{ID: "up1", Name: "", BaseURL: "https://api.example.com"},
			false,
		},
		{
			"missing BaseURL",
			route.Upstream{ID: "up1", Name: "name", BaseURL: ""},
			false,
		},
		{
			"all fields empty",
			route.Upstream{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.upstream.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Additional tests for edge cases in match.go

func TestMatcher_InvalidRegexPattern(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "invalid-regex",
			PathPattern: "[invalid(regex",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	_, err := route.NewMatcher(routes)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestMatcher_UnknownMatchType(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "unknown-type",
			PathPattern: "/api/*",
			MatchType:   route.MatchType("unknown"),
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Unknown match type should not match anything
	result := matcher.Match("GET", "/api/test", nil)
	if result != nil {
		t.Error("expected no match for unknown match type")
	}
}

func TestMatcher_HeaderMatchInvalidRegex(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "invalid-header-regex",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Headers: []route.HeaderMatch{
				{Name: "X-Custom", Value: "[invalid(regex", IsRegex: true, Required: true},
			},
			UpstreamID: "up1",
			Enabled:    true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Invalid regex in header should not match
	result := matcher.Match("GET", "/api/data", map[string]string{"X-Custom": "value"})
	if result != nil {
		t.Error("expected no match for invalid header regex")
	}
}

func TestMatcher_HeaderMatchNotRequired(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "optional-header",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Headers: []route.HeaderMatch{
				{Name: "X-Optional", Value: "specific", IsRegex: false, Required: false},
			},
			UpstreamID: "up1",
			Enabled:    true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Without header, should still match (not required)
	result := matcher.Match("GET", "/api/data", nil)
	if result == nil {
		t.Fatal("expected match when optional header is missing")
	}

	// With correct header value, should match
	result = matcher.Match("GET", "/api/data", map[string]string{"X-Optional": "specific"})
	if result == nil {
		t.Fatal("expected match when optional header has correct value")
	}

	// With incorrect header value, should not match
	result = matcher.Match("GET", "/api/data", map[string]string{"X-Optional": "wrong"})
	if result != nil {
		t.Error("expected no match when optional header has wrong value")
	}
}

func TestMatcher_PrefixWithoutWildcard(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "prefix-no-wildcard",
			PathPattern: "/api/v1",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Without wildcard, prefix pattern "/api/v1" becomes regex "^/api/v1$"
	// which only matches exact path
	tests := []struct {
		name    string
		path    string
		wantNil bool
	}{
		{"exact path", "/api/v1", false},
		{"with suffix does not match without wildcard", "/api/v1/users", true},
		{"partial match", "/api/v", true},
		{"different path", "/api/v2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match("GET", tt.path, nil)
			if tt.wantNil && result != nil {
				t.Errorf("expected nil, got route %s", result.Route.ID)
			}
			if !tt.wantNil && result == nil {
				t.Error("expected match, got nil")
			}
		})
	}
}

func TestMatcher_RegexWithAnchors(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "regex-with-anchors",
			PathPattern: "^/api/users$",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up1",
			Enabled:     true,
		},
		{
			ID:          "r2",
			Name:        "regex-without-anchors",
			PathPattern: "/api/posts",
			MatchType:   route.MatchRegex,
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Pre-anchored pattern should work
	result := matcher.Match("GET", "/api/users", nil)
	if result == nil || result.Route.ID != "r1" {
		t.Error("expected r1 to match /api/users")
	}

	// Auto-anchored pattern should also work
	result = matcher.Match("GET", "/api/posts", nil)
	if result == nil || result.Route.ID != "r2" {
		t.Error("expected r2 to match /api/posts")
	}
}

func TestMatcher_LowerCaseMethod(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "get-only",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET"},
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Lower case method should match
	result := matcher.Match("get", "/api/data", nil)
	if result == nil {
		t.Error("expected lowercase 'get' to match")
	}

	// Mixed case method should match
	result = matcher.Match("Get", "/api/data", nil)
	if result == nil {
		t.Error("expected mixed case 'Get' to match")
	}
}

func TestMatcher_LowerCaseMethodInRoute(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "lowercase-methods",
			PathPattern: "/api/data",
			MatchType:   route.MatchExact,
			Methods:     []string{"get", "post"},
			UpstreamID:  "up1",
			Enabled:     true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Uppercase request method should match lowercase route method
	result := matcher.Match("GET", "/api/data", nil)
	if result == nil {
		t.Error("expected uppercase 'GET' to match lowercase 'get' in route")
	}

	result = matcher.Match("POST", "/api/data", nil)
	if result == nil {
		t.Error("expected uppercase 'POST' to match lowercase 'post' in route")
	}
}

// ====================
// Host Matching Tests
// ====================

func TestMatcher_HostExactMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "api-host",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
		{
			ID:            "r2",
			Name:          "www-host",
			HostPattern:   "www.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up2",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		host    string
		path    string
		wantID  string
		wantNil bool
	}{
		{"exact match api.example.com", "api.example.com", "/v1/users", "r1", false},
		{"exact match www.example.com", "www.example.com", "/v1/users", "r2", false},
		{"case insensitive API.EXAMPLE.COM", "API.EXAMPLE.COM", "/v1/users", "r1", false},
		{"case insensitive mixed", "Api.Example.Com", "/v1/users", "r1", false},
		{"different host no match", "other.example.com", "/v1/users", "", true},
		{"subdomain no match", "sub.api.example.com", "/v1/users", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", tt.path, headers)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_HostWildcardMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "wildcard-api",
			HostPattern:   "*.example.com",
			HostMatchType: route.HostMatchWildcard,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		host    string
		wantNil bool
	}{
		{"single subdomain api", "api.example.com", false},
		{"single subdomain www", "www.example.com", false},
		{"single subdomain tenant1", "tenant1.example.com", false},
		{"case insensitive", "API.EXAMPLE.COM", false},
		{"multiple subdomains rejected", "a.b.example.com", true},
		{"deep subdomain rejected", "deep.sub.example.com", true},
		{"bare domain rejected", "example.com", true},
		{"different domain", "api.other.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", "/v1/users", headers)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
		})
	}
}

func TestMatcher_HostRegexMatch(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "versioned-api",
			HostPattern:   `^v[0-9]+\.api\.example\.com$`,
			HostMatchType: route.HostMatchRegex,
			PathPattern:   "/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		host    string
		wantNil bool
	}{
		{"v1 matches", "v1.api.example.com", false},
		{"v2 matches", "v2.api.example.com", false},
		{"v123 matches", "v123.api.example.com", false},
		{"case insensitive", "V1.API.EXAMPLE.COM", false},
		{"no version number", "vx.api.example.com", true},
		{"missing v prefix", "1.api.example.com", true},
		{"different domain", "v1.api.other.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", "/users", headers)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
		})
	}
}

func TestMatcher_HostMatchNone_BackwardCompatible(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "r1",
			Name:        "no-host-pattern",
			PathPattern: "/api/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "up1",
			Enabled:     true,
			// No HostPattern/HostMatchType = matches any host
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name string
		host string
	}{
		{"any host", "api.example.com"},
		{"different host", "www.other.com"},
		{"localhost", "localhost"},
		{"empty host", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var headers map[string]string
			if tt.host != "" {
				headers = map[string]string{"Host": tt.host}
			}
			result := matcher.Match("GET", "/api/users", headers)
			if result == nil {
				t.Fatal("expected match with any host")
			}
			if result.Route.ID != "r1" {
				t.Errorf("route ID = %s, want r1", result.Route.ID)
			}
		})
	}
}

// TestMatcher_HostPatternWithoutMatchType tests that host patterns are respected
// even when host_match_type is not explicitly set (inferred from pattern).
func TestMatcher_HostPatternWithoutMatchType(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "wildcard-apps",
			HostPattern:   "*.apps.example.com", // Wildcard pattern, no match type
			HostMatchType: "",                   // Empty = should infer wildcard
			PathPattern:   "/*",
			MatchType:     route.MatchPrefix,
			Priority:      100,
			UpstreamID:    "up1",
			Enabled:       true,
		},
		{
			ID:            "r2",
			Name:          "exact-host",
			HostPattern:   "api.example.com", // Exact pattern, no match type
			HostMatchType: "",                // Empty = should infer exact
			PathPattern:   "/*",
			MatchType:     route.MatchPrefix,
			Priority:      50,
			UpstreamID:    "up2",
			Enabled:       true,
		},
		{
			ID:          "r3",
			Name:        "fallback",
			PathPattern: "/*",
			MatchType:   route.MatchPrefix,
			Priority:    10,
			UpstreamID:  "up3",
			Enabled:     true,
			// No host pattern = matches any host
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name       string
		host       string
		wantRoute  string
		wantReason string
	}{
		{
			name:       "wildcard matches subdomain",
			host:       "tenant1.apps.example.com",
			wantRoute:  "r1",
			wantReason: "wildcard pattern should match single subdomain",
		},
		{
			name:       "wildcard does not match bare domain",
			host:       "example.com",
			wantRoute:  "r3",
			wantReason: "bare domain should fall through to fallback",
		},
		{
			name:       "wildcard does not match different domain",
			host:       "other.com",
			wantRoute:  "r3",
			wantReason: "different domain should fall through to fallback",
		},
		{
			name:       "exact matches specified host",
			host:       "api.example.com",
			wantRoute:  "r2",
			wantReason: "exact pattern should match specified host",
		},
		{
			name:       "exact does not match subdomain",
			host:       "www.api.example.com",
			wantRoute:  "r3",
			wantReason: "subdomain should not match exact pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", "/test", headers)
			if result == nil {
				t.Fatalf("expected a match for host %s", tt.host)
			}
			if result.Route.ID != tt.wantRoute {
				t.Errorf("route = %s, want %s (%s)", result.Route.ID, tt.wantRoute, tt.wantReason)
			}
		})
	}
}

func TestMatcher_HostPriority(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "exact-host",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
			Priority:      0,
		},
		{
			ID:            "r2",
			Name:          "wildcard-host",
			HostPattern:   "*.example.com",
			HostMatchType: route.HostMatchWildcard,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up2",
			Enabled:       true,
			Priority:      0,
		},
		{
			ID:            "r3",
			Name:          "no-host",
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up3",
			Enabled:       true,
			Priority:      0,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name   string
		host   string
		wantID string
	}{
		// Exact host should match r1 (highest host specificity)
		{"exact host matches r1", "api.example.com", "r1"},
		// Other subdomains should match r2 (wildcard)
		{"wildcard matches r2", "www.example.com", "r2"},
		{"wildcard matches r2 tenant", "tenant1.example.com", "r2"},
		// Different domain should match r3 (no host pattern = any host)
		{"no pattern matches r3", "other.domain.com", "r3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", "/v1/users", headers)
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_HostWithPort(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "api-host",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name string
		host string
	}{
		{"without port", "api.example.com"},
		{"with port 80", "api.example.com:80"},
		{"with port 8080", "api.example.com:8080"},
		{"with port 443", "api.example.com:443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", "/v1/users", headers)
			if result == nil {
				t.Fatalf("expected match for host %s", tt.host)
			}
			if result.Route.ID != "r1" {
				t.Errorf("route ID = %s, want r1", result.Route.ID)
			}
		})
	}
}

func TestMatcher_HostWithTrailingDot(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "api-host",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	// Trailing dot should be normalized
	headers := map[string]string{"Host": "api.example.com."}
	result := matcher.Match("GET", "/v1/users", headers)
	if result == nil {
		t.Fatal("expected match with trailing dot")
	}
	if result.Route.ID != "r1" {
		t.Errorf("route ID = %s, want r1", result.Route.ID)
	}
}

func TestMatcher_HostAndPathCombined(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "api-users",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/users/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
		{
			ID:            "r2",
			Name:          "api-orders",
			HostPattern:   "api.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/orders/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up2",
			Enabled:       true,
		},
		{
			ID:            "r3",
			Name:          "admin-users",
			HostPattern:   "admin.example.com",
			HostMatchType: route.HostMatchExact,
			PathPattern:   "/users/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up3",
			Enabled:       true,
		},
	}

	matcher, err := route.NewMatcher(routes)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	tests := []struct {
		name    string
		host    string
		path    string
		wantID  string
		wantNil bool
	}{
		{"api users", "api.example.com", "/users/123", "r1", false},
		{"api orders", "api.example.com", "/orders/456", "r2", false},
		{"admin users", "admin.example.com", "/users/789", "r3", false},
		{"api wrong path", "api.example.com", "/other/path", "", true},
		{"admin wrong path", "admin.example.com", "/orders/456", "", true},
		{"wrong host", "other.example.com", "/users/123", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Host": tt.host}
			result := matcher.Match("GET", tt.path, headers)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got route %s", result.Route.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected match, got nil")
			}
			if result.Route.ID != tt.wantID {
				t.Errorf("route ID = %s, want %s", result.Route.ID, tt.wantID)
			}
		})
	}
}

func TestMatcher_InvalidWildcardPattern(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "invalid-wildcard",
			HostPattern:   "api.*.example.com", // Invalid: * must be at start
			HostMatchType: route.HostMatchWildcard,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	_, err := route.NewMatcher(routes)
	if err == nil {
		t.Error("expected error for invalid wildcard pattern")
	}
}

func TestMatcher_InvalidHostRegex(t *testing.T) {
	routes := []route.Route{
		{
			ID:            "r1",
			Name:          "invalid-regex",
			HostPattern:   "[invalid(regex",
			HostMatchType: route.HostMatchRegex,
			PathPattern:   "/v1/*",
			MatchType:     route.MatchPrefix,
			UpstreamID:    "up1",
			Enabled:       true,
		},
	}

	_, err := route.NewMatcher(routes)
	if err == nil {
		t.Error("expected error for invalid host regex pattern")
	}
}

func TestRoute_WithHost(t *testing.T) {
	r := route.NewRoute("id1", "test-route", "/api/*", "upstream1")
	originalUpdatedAt := r.UpdatedAt

	r2 := r.WithHost("api.example.com", route.HostMatchExact)

	if r2.HostPattern != "api.example.com" {
		t.Errorf("HostPattern = %s, want api.example.com", r2.HostPattern)
	}
	if r2.HostMatchType != route.HostMatchExact {
		t.Errorf("HostMatchType = %s, want exact", r2.HostMatchType)
	}
	if r2.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
	// Original should be unchanged
	if r.HostPattern != "" {
		t.Error("Original route should not be modified")
	}

	// Test other host match types
	r3 := r.WithHost("*.example.com", route.HostMatchWildcard)
	if r3.HostMatchType != route.HostMatchWildcard {
		t.Errorf("HostMatchType = %s, want wildcard", r3.HostMatchType)
	}

	r4 := r.WithHost(`^v[0-9]+\.api\.example\.com$`, route.HostMatchRegex)
	if r4.HostMatchType != route.HostMatchRegex {
		t.Errorf("HostMatchType = %s, want regex", r4.HostMatchType)
	}
}
