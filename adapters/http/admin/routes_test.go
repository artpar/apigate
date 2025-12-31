package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/domain/route"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// -----------------------------------------------------------------------------
// Mock Stores
// -----------------------------------------------------------------------------

type mockRouteStore struct {
	routes map[string]route.Route
}

func newMockRouteStore() *mockRouteStore {
	return &mockRouteStore{routes: make(map[string]route.Route)}
}

func (m *mockRouteStore) Get(ctx context.Context, id string) (route.Route, error) {
	r, ok := m.routes[id]
	if !ok {
		return route.Route{}, errNotFound
	}
	return r, nil
}

func (m *mockRouteStore) List(ctx context.Context) ([]route.Route, error) {
	routes := make([]route.Route, 0, len(m.routes))
	for _, r := range m.routes {
		routes = append(routes, r)
	}
	return routes, nil
}

func (m *mockRouteStore) ListEnabled(ctx context.Context) ([]route.Route, error) {
	routes := make([]route.Route, 0)
	for _, r := range m.routes {
		if r.Enabled {
			routes = append(routes, r)
		}
	}
	return routes, nil
}

func (m *mockRouteStore) Create(ctx context.Context, r route.Route) error {
	if _, exists := m.routes[r.ID]; exists {
		return errDuplicate
	}
	m.routes[r.ID] = r
	return nil
}

func (m *mockRouteStore) Update(ctx context.Context, r route.Route) error {
	if _, exists := m.routes[r.ID]; !exists {
		return errNotFound
	}
	m.routes[r.ID] = r
	return nil
}

func (m *mockRouteStore) Delete(ctx context.Context, id string) error {
	if _, exists := m.routes[id]; !exists {
		return errNotFound
	}
	delete(m.routes, id)
	return nil
}

// mockUpstreamStoreRoutes is a simple mock for routes tests
// (separate from the concurrent-safe version in admin_test.go)
type mockUpstreamStoreRoutes struct {
	upstreams map[string]route.Upstream
}

func newMockUpstreamStoreRoutes() *mockUpstreamStoreRoutes {
	return &mockUpstreamStoreRoutes{upstreams: make(map[string]route.Upstream)}
}

func (m *mockUpstreamStoreRoutes) Get(ctx context.Context, id string) (route.Upstream, error) {
	u, ok := m.upstreams[id]
	if !ok {
		return route.Upstream{}, errNotFound
	}
	return u, nil
}

func (m *mockUpstreamStoreRoutes) List(ctx context.Context) ([]route.Upstream, error) {
	upstreams := make([]route.Upstream, 0, len(m.upstreams))
	for _, u := range m.upstreams {
		upstreams = append(upstreams, u)
	}
	return upstreams, nil
}

func (m *mockUpstreamStoreRoutes) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	upstreams := make([]route.Upstream, 0)
	for _, u := range m.upstreams {
		if u.Enabled {
			upstreams = append(upstreams, u)
		}
	}
	return upstreams, nil
}

func (m *mockUpstreamStoreRoutes) Create(ctx context.Context, u route.Upstream) error {
	if _, exists := m.upstreams[u.ID]; exists {
		return errDuplicate
	}
	m.upstreams[u.ID] = u
	return nil
}

func (m *mockUpstreamStoreRoutes) Update(ctx context.Context, u route.Upstream) error {
	if _, exists := m.upstreams[u.ID]; !exists {
		return errNotFound
	}
	m.upstreams[u.ID] = u
	return nil
}

func (m *mockUpstreamStoreRoutes) Delete(ctx context.Context, id string) error {
	if _, exists := m.upstreams[id]; !exists {
		return errNotFound
	}
	delete(m.upstreams, id)
	return nil
}

type mockError struct{}

func (e mockError) Error() string { return "not found" }

var errNotFound = mockError{}
var errDuplicate = mockError{}

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

func setupRoutesHandler() (*admin.RoutesHandler, *mockRouteStore, *mockUpstreamStoreRoutes) {
	routeStore := newMockRouteStore()
	upstreamStore := newMockUpstreamStoreRoutes()
	logger := zerolog.Nop()
	handler := admin.NewRoutesHandler(routeStore, upstreamStore, logger)
	return handler, routeStore, upstreamStore
}

func createRouter(handler *admin.RoutesHandler) http.Handler {
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	return r
}

// -----------------------------------------------------------------------------
// Route API Tests
// -----------------------------------------------------------------------------

func TestRoutesHandler_ListRoutes_Empty(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("GET", "/routes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	routes := resp["routes"].([]interface{})
	if len(routes) != 0 {
		t.Errorf("routes length = %d, want 0", len(routes))
	}
}

func TestRoutesHandler_ListRoutes_WithData(t *testing.T) {
	handler, routeStore, _ := setupRoutesHandler()
	router := createRouter(handler)

	// Seed data
	routeStore.Create(context.Background(), route.Route{
		ID:          "rt1",
		Name:        "Route 1",
		PathPattern: "/api/*",
		MatchType:   route.MatchPrefix,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	req := httptest.NewRequest("GET", "/routes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	routes := resp["routes"].([]interface{})
	if len(routes) != 1 {
		t.Errorf("routes length = %d, want 1", len(routes))
	}
}

func TestRoutesHandler_CreateRoute(t *testing.T) {
	handler, routeStore, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{
		"name": "Test Route",
		"path_pattern": "/api/v1/*",
		"match_type": "prefix",
		"methods": ["GET", "POST"],
		"priority": 10
	}`

	req := httptest.NewRequest("POST", "/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify route was created
	if len(routeStore.routes) != 1 {
		t.Errorf("routes count = %d, want 1", len(routeStore.routes))
	}

	var resp admin.RouteResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Name != "Test Route" {
		t.Errorf("name = %s, want Test Route", resp.Name)
	}
	if resp.PathPattern != "/api/v1/*" {
		t.Errorf("path_pattern = %s, want /api/v1/*", resp.PathPattern)
	}
}

func TestRoutesHandler_CreateRoute_WithTransform(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{
		"name": "Transform Route",
		"path_pattern": "/api/*",
		"request_transform": {
			"set_headers": {"X-Custom": "\"value\""},
			"delete_headers": ["X-Remove"]
		},
		"response_transform": {
			"body_expr": "{\"wrapped\": respBody}"
		}
	}`

	req := httptest.NewRequest("POST", "/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp admin.RouteResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.RequestTransform == nil {
		t.Fatal("request_transform should not be nil")
	}
	if resp.RequestTransform.SetHeaders["X-Custom"] != `"value"` {
		t.Errorf("set_headers = %v", resp.RequestTransform.SetHeaders)
	}
}

func TestRoutesHandler_CreateRoute_MissingName(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"path_pattern": "/api/*"}`

	req := httptest.NewRequest("POST", "/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoutesHandler_CreateRoute_MissingPathPattern(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"name": "Test Route"}`

	req := httptest.NewRequest("POST", "/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoutesHandler_GetRoute(t *testing.T) {
	handler, routeStore, _ := setupRoutesHandler()
	router := createRouter(handler)

	routeStore.Create(context.Background(), route.Route{
		ID:          "rt1",
		Name:        "Test Route",
		PathPattern: "/api/*",
		MatchType:   route.MatchPrefix,
		Priority:    5,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	req := httptest.NewRequest("GET", "/routes/rt1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp admin.RouteResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.ID != "rt1" {
		t.Errorf("id = %s, want rt1", resp.ID)
	}
	if resp.Name != "Test Route" {
		t.Errorf("name = %s, want Test Route", resp.Name)
	}
}

func TestRoutesHandler_GetRoute_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("GET", "/routes/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRoutesHandler_UpdateRoute(t *testing.T) {
	handler, routeStore, _ := setupRoutesHandler()
	router := createRouter(handler)

	routeStore.Create(context.Background(), route.Route{
		ID:          "rt1",
		Name:        "Original Name",
		PathPattern: "/api/*",
		MatchType:   route.MatchPrefix,
		Priority:    5,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	body := `{"name": "Updated Name", "priority": 100}`

	req := httptest.NewRequest("PUT", "/routes/rt1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp admin.RouteResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Name != "Updated Name" {
		t.Errorf("name = %s, want Updated Name", resp.Name)
	}
	if resp.Priority != 100 {
		t.Errorf("priority = %d, want 100", resp.Priority)
	}
}

func TestRoutesHandler_UpdateRoute_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"name": "Updated Name"}`

	req := httptest.NewRequest("PUT", "/routes/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRoutesHandler_DeleteRoute(t *testing.T) {
	handler, routeStore, _ := setupRoutesHandler()
	router := createRouter(handler)

	routeStore.Create(context.Background(), route.Route{
		ID:          "rt1",
		Name:        "To Delete",
		PathPattern: "/api/*",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	req := httptest.NewRequest("DELETE", "/routes/rt1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify deletion
	if len(routeStore.routes) != 0 {
		t.Errorf("routes count = %d, want 0", len(routeStore.routes))
	}
}

func TestRoutesHandler_DeleteRoute_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("DELETE", "/routes/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -----------------------------------------------------------------------------
// Upstream API Tests
// -----------------------------------------------------------------------------

func TestRoutesHandler_ListUpstreams_Empty(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("GET", "/upstreams", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	upstreams := resp["upstreams"].([]interface{})
	if len(upstreams) != 0 {
		t.Errorf("upstreams length = %d, want 0", len(upstreams))
	}
}

func TestRoutesHandler_CreateUpstream(t *testing.T) {
	handler, _, upstreamStore := setupRoutesHandler()
	router := createRouter(handler)

	body := `{
		"name": "Backend API",
		"base_url": "https://api.example.com",
		"timeout_ms": 30000
	}`

	req := httptest.NewRequest("POST", "/upstreams", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify upstream was created
	if len(upstreamStore.upstreams) != 1 {
		t.Errorf("upstreams count = %d, want 1", len(upstreamStore.upstreams))
	}

	var resp admin.UpstreamResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Name != "Backend API" {
		t.Errorf("name = %s, want Backend API", resp.Name)
	}
	if resp.BaseURL != "https://api.example.com" {
		t.Errorf("base_url = %s, want https://api.example.com", resp.BaseURL)
	}
}

func TestRoutesHandler_CreateUpstream_WithAuth(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{
		"name": "Auth Backend",
		"base_url": "https://api.example.com",
		"auth_type": "bearer",
		"auth_value": "secret-token"
	}`

	req := httptest.NewRequest("POST", "/upstreams", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp admin.UpstreamResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.AuthType != "bearer" {
		t.Errorf("auth_type = %s, want bearer", resp.AuthType)
	}
}

func TestRoutesHandler_CreateUpstream_MissingName(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"base_url": "https://api.example.com"}`

	req := httptest.NewRequest("POST", "/upstreams", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoutesHandler_CreateUpstream_MissingBaseURL(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"name": "Test"}`

	req := httptest.NewRequest("POST", "/upstreams", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoutesHandler_GetUpstream(t *testing.T) {
	handler, _, upstreamStore := setupRoutesHandler()
	router := createRouter(handler)

	upstreamStore.Create(context.Background(), route.Upstream{
		ID:        "up1",
		Name:      "Test Upstream",
		BaseURL:   "https://api.example.com",
		Timeout:   30 * time.Second,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	req := httptest.NewRequest("GET", "/upstreams/up1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp admin.UpstreamResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.ID != "up1" {
		t.Errorf("id = %s, want up1", resp.ID)
	}
	if resp.Name != "Test Upstream" {
		t.Errorf("name = %s, want Test Upstream", resp.Name)
	}
}

func TestRoutesHandler_GetUpstream_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("GET", "/upstreams/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRoutesHandler_UpdateUpstream(t *testing.T) {
	handler, _, upstreamStore := setupRoutesHandler()
	router := createRouter(handler)

	upstreamStore.Create(context.Background(), route.Upstream{
		ID:        "up1",
		Name:      "Original",
		BaseURL:   "https://old.example.com",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	body := `{"name": "Updated", "base_url": "https://new.example.com"}`

	req := httptest.NewRequest("PUT", "/upstreams/up1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp admin.UpstreamResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Name != "Updated" {
		t.Errorf("name = %s, want Updated", resp.Name)
	}
	if resp.BaseURL != "https://new.example.com" {
		t.Errorf("base_url = %s, want https://new.example.com", resp.BaseURL)
	}
}

func TestRoutesHandler_UpdateUpstream_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{"name": "Updated"}`

	req := httptest.NewRequest("PUT", "/upstreams/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRoutesHandler_DeleteUpstream(t *testing.T) {
	handler, _, upstreamStore := setupRoutesHandler()
	router := createRouter(handler)

	upstreamStore.Create(context.Background(), route.Upstream{
		ID:        "up1",
		Name:      "To Delete",
		BaseURL:   "https://api.example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	req := httptest.NewRequest("DELETE", "/upstreams/up1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify deletion
	if len(upstreamStore.upstreams) != 0 {
		t.Errorf("upstreams count = %d, want 0", len(upstreamStore.upstreams))
	}
}

func TestRoutesHandler_DeleteUpstream_NotFound(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	req := httptest.NewRequest("DELETE", "/upstreams/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRoutesHandler_InvalidJSON(t *testing.T) {
	handler, _, _ := setupRoutesHandler()
	router := createRouter(handler)

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
