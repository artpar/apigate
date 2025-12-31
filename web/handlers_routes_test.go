package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/settings"
	"github.com/go-chi/chi/v5"
)

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single value", "GET", []string{"GET"}},
		{"multiple values", "GET,POST,PUT", []string{"GET", "POST", "PUT"}},
		{"with spaces", "GET, POST, PUT", []string{"GET", "POST", "PUT"}},
		{"trailing comma", "GET,POST,", []string{"GET", "POST"}},
		{"empty values", "GET,,POST", []string{"GET", "POST"}},
		{"only spaces", " , , ", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSV(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseCSV(%q) = %v (len=%d), want %v (len=%d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCSV(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"10", 10},
		{"123", 123},
		{"abc", 0},
		{"12abc34", 1234},
		{"-5", 5}, // ignores minus sign
		{"3.14", 314},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single pair",
			input:    "key=value",
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "multiple pairs",
			input:    "key1=value1\nkey2=value2",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "with spaces",
			input:    "key1 = value1\n key2=value2 ",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "value with equals",
			input:    "key=value=with=equals",
			expected: map[string]string{"key": "value=with=equals"},
		},
		{
			name:     "empty lines",
			input:    "key1=value1\n\nkey2=value2\n",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "no equals",
			input:    "keyonly",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyValue(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseKeyValue result len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseKeyValue[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestParseTransform(t *testing.T) {
	tests := []struct {
		name     string
		formData url.Values
		prefix   string
		isNil    bool
	}{
		{
			name:     "empty form",
			formData: url.Values{},
			prefix:   "request_",
			isNil:    true,
		},
		{
			name: "with set headers",
			formData: url.Values{
				"request_set_headers": {"X-Custom=value"},
			},
			prefix: "request_",
			isNil:  false,
		},
		{
			name: "with delete headers",
			formData: url.Values{
				"request_delete_headers": {"X-Remove,X-Delete"},
			},
			prefix: "request_",
			isNil:  false,
		},
		{
			name: "with body expr",
			formData: url.Values{
				"response_body_expr": {"json_encode(body)"},
			},
			prefix: "response_",
			isNil:  false,
		},
		{
			name: "with query params",
			formData: url.Values{
				"request_set_query":    {"key=value"},
				"request_delete_query": {"remove"},
			},
			prefix: "request_",
			isNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.ParseForm()

			result := parseTransform(req, tt.prefix)
			if tt.isNil {
				if result != nil {
					t.Errorf("parseTransform() should be nil for empty form")
				}
			} else {
				if result == nil {
					t.Errorf("parseTransform() should not be nil")
				}
			}
		})
	}
}

func TestHandler_RouteCreate(t *testing.T) {
	h, _, _, _ := newTestHandler()

	form := url.Values{
		"name":          {"Test Route"},
		"path_pattern":  {"/api/test"},
		"match_type":    {"prefix"},
		"methods":       {"GET,POST"},
		"upstream_id":   {"upstream1"},
		"metering_expr": {"1"},
		"metering_mode": {"request"},
		"protocol":      {"http"},
		"priority":      {"10"},
		"enabled":       {"on"},
	}

	req := httptest.NewRequest("POST", "/routes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.RouteCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestHandler_RouteCreate_InvalidForm(t *testing.T) {
	h, _, _, _ := newTestHandler()

	// Send invalid form data (not properly encoded)
	req := httptest.NewRequest("POST", "/routes", strings.NewReader("%invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.RouteCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_RouteUpdate(t *testing.T) {
	h, _, _, _ := newTestHandler()

	// Create a route first
	routes := h.routes.(*mockRoutes)
	routes.routes["route1"] = route.Route{
		ID:        "route1",
		Name:      "Original",
		CreatedAt: time.Now(),
	}

	r := chi.NewRouter()
	r.Post("/routes/{id}", h.RouteUpdate)

	form := url.Values{
		"name":          {"Updated Route"},
		"path_pattern":  {"/api/updated"},
		"match_type":    {"exact"},
		"methods":       {"GET"},
		"upstream_id":   {"upstream1"},
		"metering_expr": {"2"},
		"metering_mode": {"tokens"},
		"protocol":      {"http"},
		"priority":      {"20"},
		"enabled":       {"on"},
	}

	req := httptest.NewRequest("POST", "/routes/route1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if routes.routes["route1"].Name != "Updated Route" {
		t.Errorf("Route name = %s, want Updated Route", routes.routes["route1"].Name)
	}
}

func TestHandler_RouteUpdate_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Post("/routes/{id}", h.RouteUpdate)

	form := url.Values{"name": {"Test"}}
	req := httptest.NewRequest("POST", "/routes/nonexistent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_RouteDelete_Regular(t *testing.T) {
	h, _, _, _ := newTestHandler()

	routes := h.routes.(*mockRoutes)
	routes.routes["route1"] = route.Route{ID: "route1", Name: "Test"}

	r := chi.NewRouter()
	r.Delete("/routes/{id}", h.RouteDelete)

	req := httptest.NewRequest("DELETE", "/routes/route1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if _, ok := routes.routes["route1"]; ok {
		t.Error("Route should be deleted")
	}
}

func TestHandler_RouteEditPage_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Get("/routes/{id}", h.RouteEditPage)

	req := httptest.NewRequest("GET", "/routes/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_UpstreamCreate(t *testing.T) {
	h, _, _, _ := newTestHandler()

	form := url.Values{
		"name":                 {"Test Upstream"},
		"base_url":             {"http://api.example.com"},
		"timeout_ms":           {"30000"},
		"auth_type":            {"none"},
		"max_idle_conns":       {"100"},
		"idle_conn_timeout_ms": {"90000"},
		"enabled":              {"on"},
	}

	req := httptest.NewRequest("POST", "/upstreams", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.UpstreamCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestHandler_UpstreamCreate_DefaultValues(t *testing.T) {
	h, _, _, _ := newTestHandler()

	// Test with empty timeout values (should use defaults)
	form := url.Values{
		"name":     {"Test Upstream"},
		"base_url": {"http://api.example.com"},
		"enabled":  {"on"},
	}

	req := httptest.NewRequest("POST", "/upstreams", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.UpstreamCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	// Verify defaults were applied
	upstreams := h.upstreams.(*mockUpstreams)
	for _, u := range upstreams.upstreams {
		if u.MaxIdleConns != 100 {
			t.Errorf("MaxIdleConns = %d, want 100", u.MaxIdleConns)
		}
	}
}

func TestHandler_UpstreamUpdate(t *testing.T) {
	h, _, _, _ := newTestHandler()

	upstreams := h.upstreams.(*mockUpstreams)
	upstreams.upstreams["upstream1"] = route.Upstream{
		ID:        "upstream1",
		Name:      "Original",
		CreatedAt: time.Now(),
	}

	r := chi.NewRouter()
	r.Post("/upstreams/{id}", h.UpstreamUpdate)

	form := url.Values{
		"name":                 {"Updated Upstream"},
		"base_url":             {"http://updated.example.com"},
		"timeout_ms":           {"60000"},
		"auth_type":            {"bearer"},
		"auth_header":          {"Authorization"},
		"auth_value":           {"Bearer token"},
		"max_idle_conns":       {"200"},
		"idle_conn_timeout_ms": {"120000"},
		"enabled":              {"on"},
	}

	req := httptest.NewRequest("POST", "/upstreams/upstream1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if upstreams.upstreams["upstream1"].Name != "Updated Upstream" {
		t.Error("Upstream should be updated")
	}
}

func TestHandler_UpstreamUpdate_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Post("/upstreams/{id}", h.UpstreamUpdate)

	form := url.Values{"name": {"Test"}}
	req := httptest.NewRequest("POST", "/upstreams/nonexistent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_UpstreamDelete_Regular(t *testing.T) {
	h, _, _, _ := newTestHandler()

	upstreams := h.upstreams.(*mockUpstreams)
	upstreams.upstreams["upstream1"] = route.Upstream{ID: "upstream1", Name: "Test"}

	r := chi.NewRouter()
	r.Delete("/upstreams/{id}", h.UpstreamDelete)

	req := httptest.NewRequest("DELETE", "/upstreams/upstream1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if _, ok := upstreams.upstreams["upstream1"]; ok {
		t.Error("Upstream should be deleted")
	}
}

func TestHandler_UpstreamEditPage_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Get("/upstreams/{id}", h.UpstreamEditPage)

	req := httptest.NewRequest("GET", "/upstreams/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ValidateExpr(t *testing.T) {
	h, _, _, _ := newTestHandler()

	tests := []struct {
		name       string
		reqBody    map[string]string
		wantStatus int
	}{
		{
			name:       "valid expression",
			reqBody:    map[string]string{"expression": "1 + 1", "context": "request"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty context defaults to request",
			reqBody:    map[string]string{"expression": "body.field"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			reqBody:    nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			if tt.reqBody != nil {
				jsonBody, _ := json.Marshal(tt.reqBody)
				body = string(jsonBody)
			} else {
				body = "invalid json"
			}

			req := httptest.NewRequest("POST", "/api/expr/validate", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.ValidateExpr(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_ValidateExpr_NoValidator(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.exprValidator = nil

	req := httptest.NewRequest("POST", "/api/expr/validate", strings.NewReader(`{"expression": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateExpr(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandler_TestRoute(t *testing.T) {
	h, _, _, _ := newTestHandler()

	tests := []struct {
		name       string
		reqBody    interface{}
		wantStatus int
	}{
		{
			name: "valid request",
			reqBody: app.RouteTestRequest{
				Method: "GET",
				Path:   "/test",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "empty method defaults to GET",
			reqBody: app.RouteTestRequest{
				Path: "/test",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			reqBody:    nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			if tt.reqBody != nil {
				jsonBody, _ := json.Marshal(tt.reqBody)
				body = string(jsonBody)
			} else {
				body = "invalid json"
			}

			req := httptest.NewRequest("POST", "/api/routes/test", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.TestRoute(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_TestRoute_NoTester(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.routeTester = nil

	req := httptest.NewRequest("POST", "/api/routes/test", strings.NewReader(`{"path": "/test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.TestRoute(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var result app.RouteTestResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Error == "" {
		t.Error("Expected error message in response")
	}
}

// mockSettingsStore implements ports.SettingsStore for testing
type mockSettingsStore struct {
	settings map[string]string
}

func newMockSettingsStore() *mockSettingsStore {
	return &mockSettingsStore{settings: make(map[string]string)}
}

func (m *mockSettingsStore) Get(ctx context.Context, key string) (settings.Setting, error) {
	if v, ok := m.settings[key]; ok {
		return settings.Setting{Key: key, Value: v}, nil
	}
	return settings.Setting{Key: key}, nil
}

func (m *mockSettingsStore) GetAll(ctx context.Context) (settings.Settings, error) {
	result := make(settings.Settings)
	for k, v := range m.settings {
		result[k] = v
	}
	return result, nil
}

func (m *mockSettingsStore) GetByPrefix(ctx context.Context, prefix string) (settings.Settings, error) {
	result := make(settings.Settings)
	for k, v := range m.settings {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v
		}
	}
	return result, nil
}

func (m *mockSettingsStore) Set(ctx context.Context, key, value string, encrypted bool) error {
	m.settings[key] = value
	return nil
}

func (m *mockSettingsStore) SetBatch(ctx context.Context, s settings.Settings) error {
	for k, v := range s {
		m.settings[k] = v
	}
	return nil
}

func (m *mockSettingsStore) Delete(ctx context.Context, key string) error {
	delete(m.settings, key)
	return nil
}
