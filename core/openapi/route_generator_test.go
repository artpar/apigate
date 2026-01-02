package openapi

import (
	"testing"

	"github.com/artpar/apigate/domain/route"
)

func TestNewRouteGenerator(t *testing.T) {
	routes := []route.Route{
		{ID: "1", Name: "test", PathPattern: "/api/test", Enabled: true},
	}
	upstreams := map[string]route.Upstream{
		"upstream1": {ID: "upstream1", Name: "test-upstream", Enabled: true},
	}

	gen := NewRouteGenerator(routes, upstreams)

	if gen == nil {
		t.Fatal("NewRouteGenerator returned nil")
	}

	if len(gen.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(gen.routes))
	}

	if len(gen.upstreams) != 1 {
		t.Errorf("expected 1 upstream, got %d", len(gen.upstreams))
	}

	if gen.info.Title != "Proxied APIs" {
		t.Errorf("expected default title 'Proxied APIs', got %q", gen.info.Title)
	}
}

func TestRouteGenerator_SetInfo(t *testing.T) {
	gen := NewRouteGenerator(nil, nil)

	info := Info{
		Title:       "Custom API",
		Description: "Custom description",
		Version:     "2.0.0",
	}

	gen.SetInfo(info)

	if gen.info.Title != "Custom API" {
		t.Errorf("expected title 'Custom API', got %q", gen.info.Title)
	}
}

func TestRouteGenerator_AddServer(t *testing.T) {
	gen := NewRouteGenerator(nil, nil)

	gen.AddServer("https://api.example.com", "Production")
	gen.AddServer("https://staging.example.com", "Staging")

	if len(gen.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(gen.servers))
	}

	if gen.servers[0].URL != "https://api.example.com" {
		t.Errorf("expected first server URL 'https://api.example.com', got %q", gen.servers[0].URL)
	}
}

func TestRouteGenerator_Generate_EmptyRoutes(t *testing.T) {
	gen := NewRouteGenerator(nil, nil)
	spec := gen.Generate()

	if spec == nil {
		t.Fatal("Generate returned nil")
	}

	if spec.OpenAPI != "3.0.3" {
		t.Errorf("expected OpenAPI version '3.0.3', got %q", spec.OpenAPI)
	}

	if _, ok := spec.Components.SecuritySchemes["apiKey"]; !ok {
		t.Error("expected apiKey security scheme")
	}
}

func TestRouteGenerator_Generate_WithRoutes(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "api-users",
			PathPattern: "/api/users",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET", "POST"},
			Enabled:     true,
			UpstreamID:  "upstream1",
		},
		{
			ID:          "2",
			Name:        "api-items",
			PathPattern: "/api/items/*",
			MatchType:   route.MatchPrefix,
			Methods:     []string{"GET"},
			Enabled:     true,
			UpstreamID:  "upstream1",
		},
		{
			ID:          "3",
			Name:        "disabled-route",
			PathPattern: "/disabled",
			Enabled:     false,
		},
	}
	upstreams := map[string]route.Upstream{
		"upstream1": {ID: "upstream1", Name: "backend", BaseURL: "https://backend.example.com", Enabled: true},
	}

	gen := NewRouteGenerator(routes, upstreams)
	spec := gen.Generate()

	// Check paths are generated
	if _, ok := spec.Paths["/api/users"]; !ok {
		t.Error("expected /api/users path")
	}

	if _, ok := spec.Paths["/api/items/{path}"]; !ok {
		t.Error("expected /api/items/{path} path")
	}

	// Disabled route should not be included
	if _, ok := spec.Paths["/disabled"]; ok {
		t.Error("disabled route should not be included")
	}

	// Check operations
	usersPath := spec.Paths["/api/users"]
	if usersPath.Get == nil {
		t.Error("expected GET operation on /api/users")
	}
	if usersPath.Post == nil {
		t.Error("expected POST operation on /api/users")
	}
}

func TestRouteGenerator_Generate_AllMethods(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "all-methods",
			PathPattern: "/api/resource",
			MatchType:   route.MatchExact,
			Methods:     []string{}, // Empty means all methods
			Enabled:     true,
		},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	path := spec.Paths["/api/resource"]
	if path.Get == nil {
		t.Error("expected GET operation")
	}
	if path.Post == nil {
		t.Error("expected POST operation")
	}
	if path.Put == nil {
		t.Error("expected PUT operation")
	}
	if path.Patch == nil {
		t.Error("expected PATCH operation")
	}
	if path.Delete == nil {
		t.Error("expected DELETE operation")
	}
}

func TestRouteGenerator_Generate_WithProtocol(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "sse-route",
			PathPattern: "/api/stream",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET"},
			Protocol:    route.ProtocolSSE,
			Enabled:     true,
		},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	path := spec.Paths["/api/stream"]
	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	// Description should include protocol info
	if path.Get.Description == "" {
		t.Error("expected description")
	}
}

func TestRouteGenerator_Generate_WithHeaders(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "header-route",
			PathPattern: "/api/versioned",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET"},
			Headers: []route.HeaderMatch{
				{Name: "X-API-Version", Required: true},
				{Name: "X-Custom-Header", Required: false},
			},
			Enabled: true,
		},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	path := spec.Paths["/api/versioned"]
	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	// Check header parameters
	foundVersion := false
	foundCustom := false
	for _, param := range path.Get.Parameters {
		if param.Name == "X-API-Version" {
			foundVersion = true
			if !param.Required {
				t.Error("X-API-Version should be required")
			}
		}
		if param.Name == "X-Custom-Header" {
			foundCustom = true
			if param.Required {
				t.Error("X-Custom-Header should not be required")
			}
		}
	}

	if !foundVersion {
		t.Error("expected X-API-Version parameter")
	}
	if !foundCustom {
		t.Error("expected X-Custom-Header parameter")
	}
}

func TestRouteGenerator_SecuritySchemes(t *testing.T) {
	upstreams := map[string]route.Upstream{
		"bearer-auth": {
			ID:       "bearer-auth",
			Name:     "bearer-api",
			AuthType: route.AuthBearer,
			Enabled:  true,
		},
		"basic-auth": {
			ID:       "basic-auth",
			Name:     "basic-api",
			AuthType: route.AuthBasic,
			Enabled:  true,
		},
		"header-auth": {
			ID:         "header-auth",
			Name:       "header-api",
			AuthType:   route.AuthHeader,
			AuthHeader: "X-API-Key",
			Enabled:    true,
		},
		"no-auth": {
			ID:       "no-auth",
			Name:     "no-auth-api",
			AuthType: route.AuthNone,
			Enabled:  true,
		},
		"disabled": {
			ID:       "disabled",
			Name:     "disabled-api",
			AuthType: route.AuthBearer,
			Enabled:  false, // Should not be included
		},
	}

	gen := NewRouteGenerator(nil, upstreams)
	spec := gen.Generate()

	// Check bearer auth scheme
	if _, ok := spec.Components.SecuritySchemes["bearer_api_bearer"]; !ok {
		t.Error("expected bearer_api_bearer security scheme")
	}

	// Check basic auth scheme
	if _, ok := spec.Components.SecuritySchemes["basic_api_basic"]; !ok {
		t.Error("expected basic_api_basic security scheme")
	}

	// Check header auth scheme
	if _, ok := spec.Components.SecuritySchemes["header_api_header"]; !ok {
		t.Error("expected header_api_header security scheme")
	}

	// Disabled upstream should not have security scheme (besides apiKey)
	if _, ok := spec.Components.SecuritySchemes["disabled_api_bearer"]; ok {
		t.Error("disabled upstream should not have security scheme")
	}
}

func TestConvertPathPattern_Exact(t *testing.T) {
	tests := []struct {
		pattern        string
		expectedPath   string
		expectedParams []string
	}{
		{"/api/users", "/api/users", nil},
		{"/api/users/{id}", "/api/users/{id}", []string{"id"}},
		{"/api/{org}/users/{id}", "/api/{org}/users/{id}", []string{"org", "id"}},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			path, params := convertPathPattern(tt.pattern, route.MatchExact)

			if path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, path)
			}

			if len(params) != len(tt.expectedParams) {
				t.Errorf("expected %d params, got %d", len(tt.expectedParams), len(params))
			}
		})
	}
}

func TestConvertPathPattern_Prefix(t *testing.T) {
	tests := []struct {
		pattern        string
		expectedPath   string
		expectedParams []string
	}{
		{"/api/*", "/api/{path}", []string{"path"}},
		{"/api/v1/*", "/api/v1/{path}", []string{"path"}},
		{"/api*", "/api{path}", []string{"path"}},
		{"/api/users", "/api/users", nil},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			path, params := convertPathPattern(tt.pattern, route.MatchPrefix)

			if path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, path)
			}

			if len(params) != len(tt.expectedParams) {
				t.Errorf("expected %d params, got %d", len(tt.expectedParams), len(params))
			}
		})
	}
}

func TestConvertPathPattern_Regex(t *testing.T) {
	tests := []struct {
		pattern      string
		expectedPath string
	}{
		{"/api/users/[0-9]+", "/api/users/{id}"},
		{"^/api/.+$", "/api/{path}"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			path, _ := convertPathPattern(tt.pattern, route.MatchRegex)

			if path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, path)
			}
		})
	}
}

func TestConvertPathPattern_Default(t *testing.T) {
	path, params := convertPathPattern("/api/{id}", "unknown")

	if path != "/api/{id}" {
		t.Errorf("expected path '/api/{id}', got %q", path)
	}

	if len(params) != 1 || params[0] != "id" {
		t.Error("expected id param")
	}
}

func TestExtractBraceParams(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/api/users", nil},
		{"/api/users/{id}", []string{"id"}},
		{"/api/{org}/users/{id}", []string{"org", "id"}},
		{"/api/{a}/{b}/{c}", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := extractBraceParams(tt.path)

			if len(params) != len(tt.expected) {
				t.Errorf("expected %d params, got %d", len(tt.expected), len(params))
				return
			}

			for i, p := range params {
				if p != tt.expected[i] {
					t.Errorf("expected param %q, got %q", tt.expected[i], p)
				}
			}
		})
	}
}

func TestConvertRegexToOpenAPI(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"/api/users/[0-9]+", "/api/users/{id}"},
		{"/api/.+", "/api/{path}"},
		{"^/api/test$", "/api/test"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := convertRegexToOpenAPI(tt.pattern)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeSchemeName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"simple", "simple"},
		{"With Spaces", "with_spaces"},
		{"special-chars!", "special_chars"},
		{"  leading-trailing  ", "leading_trailing"},
		{"UPPERCASE", "uppercase"},
		{"", "upstream"},
		{"!!!@@@", "upstream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSchemeName(tt.name)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		routeName string
		method    string
		expected  string
	}{
		{"get-users", "GET", "get_get_users"},
		{"create-user", "POST", "post_create_user"},
		{"User Profile", "GET", "get_user_profile"},
		{"", "GET", "get_route"},
		{"special!@#chars", "DELETE", "delete_special_chars"},
	}

	for _, tt := range tests {
		t.Run(tt.routeName+"_"+tt.method, func(t *testing.T) {
			result := generateOperationID(tt.routeName, tt.method)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRouteGenerator_RequestBody(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "crud-route",
			PathPattern: "/api/resource",
			MatchType:   route.MatchExact,
			Methods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Enabled:     true,
		},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	path := spec.Paths["/api/resource"]

	// GET and DELETE should not have request body
	if path.Get != nil && path.Get.RequestBody != nil {
		t.Error("GET should not have request body")
	}
	if path.Delete != nil && path.Delete.RequestBody != nil {
		t.Error("DELETE should not have request body")
	}

	// POST, PUT, PATCH should have request body
	if path.Post != nil && path.Post.RequestBody == nil {
		t.Error("POST should have request body")
	}
	if path.Put != nil && path.Put.RequestBody == nil {
		t.Error("PUT should have request body")
	}
	if path.Patch != nil && path.Patch.RequestBody == nil {
		t.Error("PATCH should have request body")
	}
}

func TestRouteGenerator_RoutePrioritySorting(t *testing.T) {
	routes := []route.Route{
		{ID: "1", Name: "low-priority", PathPattern: "/api/a", Priority: 1, Enabled: true, Methods: []string{"GET"}},
		{ID: "2", Name: "high-priority", PathPattern: "/api/b", Priority: 100, Enabled: true, Methods: []string{"GET"}},
		{ID: "3", Name: "medium-priority", PathPattern: "/api/c", Priority: 50, Enabled: true, Methods: []string{"GET"}},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	// All routes should be present
	if len(spec.Paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(spec.Paths))
	}
}

func TestRouteGenerator_UpstreamTags(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "route1",
			PathPattern: "/api/test",
			UpstreamID:  "upstream1",
			Methods:     []string{"GET"},
			Enabled:     true,
		},
	}
	upstreams := map[string]route.Upstream{
		"upstream1": {ID: "upstream1", Name: "my-backend", Enabled: true},
	}

	gen := NewRouteGenerator(routes, upstreams)
	spec := gen.Generate()

	path := spec.Paths["/api/test"]
	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	// Tags should include upstream name
	found := false
	for _, tag := range path.Get.Tags {
		if tag == "my-backend" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected upstream name in tags")
	}
}

func TestRouteGenerator_SecurityRequirements(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "secured-route",
			PathPattern: "/api/secure",
			UpstreamID:  "bearer-upstream",
			Methods:     []string{"GET"},
			Enabled:     true,
		},
	}
	upstreams := map[string]route.Upstream{
		"bearer-upstream": {
			ID:       "bearer-upstream",
			Name:     "secure-api",
			AuthType: route.AuthBearer,
			Enabled:  true,
		},
	}

	gen := NewRouteGenerator(routes, upstreams)
	spec := gen.Generate()

	path := spec.Paths["/api/secure"]
	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	// Should have security requirements
	if len(path.Get.Security) == 0 {
		t.Error("expected security requirements")
	}

	// Should include apiKey
	foundAPIKey := false
	for _, sec := range path.Get.Security {
		if _, ok := sec["apiKey"]; ok {
			foundAPIKey = true
			break
		}
	}

	if !foundAPIKey {
		t.Error("expected apiKey in security requirements")
	}
}

func TestRouteGenerator_HeaderAuthWithEmptyHeader(t *testing.T) {
	upstreams := map[string]route.Upstream{
		"empty-header": {
			ID:         "empty-header",
			Name:       "empty-header-api",
			AuthType:   route.AuthHeader,
			AuthHeader: "", // Empty header
			Enabled:    true,
		},
	}

	gen := NewRouteGenerator(nil, upstreams)
	spec := gen.Generate()

	// Should not create security scheme for empty header
	if _, ok := spec.Components.SecuritySchemes["empty_header_api_header"]; ok {
		t.Error("should not create security scheme for empty auth header")
	}
}

func TestRouteGenerator_RouteWithDescription(t *testing.T) {
	routes := []route.Route{
		{
			ID:          "1",
			Name:        "described-route",
			PathPattern: "/api/described",
			Description: "This is a custom description",
			Methods:     []string{"GET"},
			Enabled:     true,
		},
	}

	gen := NewRouteGenerator(routes, nil)
	spec := gen.Generate()

	path := spec.Paths["/api/described"]
	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	if path.Get.Description != "This is a custom description" {
		t.Errorf("expected custom description, got %q", path.Get.Description)
	}
}
