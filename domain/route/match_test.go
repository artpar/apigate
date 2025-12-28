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
