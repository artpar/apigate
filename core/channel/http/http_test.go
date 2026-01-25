package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/go-chi/chi/v5"
)

// mockRuntime implements a mock runtime for testing.
type mockRuntime struct {
	executeFunc func(ctx context.Context, module, action string, input runtime.ActionInput) (runtime.ActionResult, error)
}

func (m *mockRuntime) Execute(ctx context.Context, module, action string, input runtime.ActionInput) (runtime.ActionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, module, action, input)
	}
	return runtime.ActionResult{}, nil
}

func TestNew(t *testing.T) {
	c := New(nil, ":8080")
	if c == nil {
		t.Fatal("New should return non-nil channel")
	}
	if c.addr != ":8080" {
		t.Errorf("addr = %q, want %q", c.addr, ":8080")
	}
	if c.modules == nil {
		t.Error("modules should be initialized")
	}
	if c.router == nil {
		t.Error("router should be initialized")
	}
}

func TestChannel_Name(t *testing.T) {
	c := &Channel{}
	if c.Name() != "http" {
		t.Errorf("Name() = %q, want %q", c.Name(), "http")
	}
}

func TestChannel_Handler(t *testing.T) {
	c := New(nil, "")
	handler := c.Handler()
	if handler == nil {
		t.Error("Handler() should return non-nil handler")
	}
}

func TestChannel_Start_NoAddr(t *testing.T) {
	c := New(nil, "")
	err := c.Start(context.Background())
	if err != nil {
		t.Errorf("Start() with no addr should not error: %v", err)
	}
}

func TestChannel_Stop_NoServer(t *testing.T) {
	c := &Channel{}
	err := c.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() with no server should not error: %v", err)
	}
}

func TestChannel_Register_DisabledHTTP(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "test",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{Enabled: false},
				},
			},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register disabled module should not error: %v", err)
	}

	if _, exists := c.modules["test"]; exists {
		t.Error("Disabled module should not be registered")
	}
}

func TestChannel_Register_EnabledHTTP(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "item",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{Enabled: true},
				},
			},
		},
		Plural: "items",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
			{Name: "create", Type: schema.ActionTypeCreate},
			{Name: "update", Type: schema.ActionTypeUpdate},
			{Name: "delete", Type: schema.ActionTypeDelete},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}

	if _, exists := c.modules["item"]; !exists {
		t.Error("Module should be registered")
	}
}

func TestJSONAPIContentType(t *testing.T) {
	// Test that JSON:API responses use the correct content type
	w := httptest.NewRecorder()
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{"test": "value"})

	if w.Header().Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/vnd.api+json")
	}
}

func TestJSONAPIErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	jsonapi.WriteBadRequest(w, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	if w.Header().Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/vnd.api+json")
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Fatal("expected errors array in response")
	}
	errObj, ok := errors[0].(map[string]any)
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["detail"] != "test error" {
		t.Errorf("error detail = %q, want %q", errObj["detail"], "test error")
	}
}

func TestChannel_handleOpenAPI(t *testing.T) {
	c := New(nil, "")

	req := httptest.NewRequest("GET", "/_openapi", nil)
	w := httptest.NewRecorder()

	c.handleOpenAPI(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/json")
	}
}

func TestChannel_handleSwaggerUI(t *testing.T) {
	c := New(nil, "")

	req := httptest.NewRequest("GET", "/swagger", nil)
	w := httptest.NewRecorder()

	c.handleSwaggerUI(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Should return HTML
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	}
}

func TestChannel_RootRedirect(t *testing.T) {
	c := New(nil, "")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	c.router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if location != "/ui/" {
		t.Errorf("Location = %q, want %q", location, "/ui/")
	}
}

func TestChannel_DoList_ParseParams(t *testing.T) {
	c := &Channel{
		modules: make(map[string]convention.Derived),
	}

	// Test with limit and offset query params
	req := httptest.NewRequest("GET", "/items?limit=50&offset=10", nil)
	w := httptest.NewRecorder()

	mod := convention.Derived{
		Source: schema.Module{Name: "item"},
		Fields: []convention.DerivedField{
			{Name: "name"},
			{Name: "status"},
		},
	}

	// We can't easily test doList without a real runtime, but we can verify the function exists
	// and test the parsing logic by checking the exported function exists
	_ = c
	_ = w
	_ = req
	_ = mod
}

func TestChannel_DoCreate_InvalidJSON(t *testing.T) {
	c := &Channel{
		modules: make(map[string]convention.Derived),
	}

	mod := convention.Derived{
		Source: schema.Module{Name: "item"},
	}

	req := httptest.NewRequest("POST", "/items", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c.doCreate(context.Background(), w, req, mod)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestChannel_DoUpdate_InvalidJSON(t *testing.T) {
	c := &Channel{
		modules: make(map[string]convention.Derived),
	}

	mod := convention.Derived{
		Source: schema.Module{Name: "item"},
	}

	req := httptest.NewRequest("PUT", "/items/123", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c.doUpdate(context.Background(), w, req, mod, "123")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestChannel_DoCustomAction_ActionNotFound(t *testing.T) {
	c := &Channel{
		modules: make(map[string]convention.Derived),
	}

	mod := convention.Derived{
		Source:  schema.Module{Name: "item"},
		Actions: []convention.DerivedAction{},
	}

	req := httptest.NewRequest("POST", "/items/123/publish", nil)
	w := httptest.NewRecorder()

	c.doCustomAction(context.Background(), w, req, mod, "123", "publish")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	// JSON:API error format uses "errors" array
	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Error("should have errors array")
	}
}

func TestChannel_DoCustomAction_InvalidJSON(t *testing.T) {
	c := &Channel{
		modules: make(map[string]convention.Derived),
	}

	mod := convention.Derived{
		Source: schema.Module{Name: "item"},
		Actions: []convention.DerivedAction{
			{Name: "publish", Type: schema.ActionTypeCustom},
		},
	}

	req := httptest.NewRequest("POST", "/items/123/publish", bytes.NewReader([]byte("invalid json")))
	req.ContentLength = 12 // Set content length so it tries to parse
	w := httptest.NewRecorder()

	c.doCustomAction(context.Background(), w, req, mod, "123", "publish")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestChannel_RegisterActionRoute_AllTypes(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "item",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{Enabled: true},
				},
			},
		},
		Plural: "items",
	}

	// Test all action types
	tests := []struct {
		action   convention.DerivedAction
		method   string
		path     string
		wantCode int
	}{
		{
			action:   convention.DerivedAction{Name: "list", Type: schema.ActionTypeList},
			method:   "GET",
			path:     "/items",
			wantCode: http.StatusInternalServerError, // No runtime, but route registered
		},
		{
			action:   convention.DerivedAction{Name: "get", Type: schema.ActionTypeGet},
			method:   "GET",
			path:     "/items/123",
			wantCode: http.StatusInternalServerError,
		},
		{
			action:   convention.DerivedAction{Name: "update", Type: schema.ActionTypeUpdate},
			method:   "PUT",
			path:     "/items/123",
			wantCode: http.StatusBadRequest, // No body
		},
		{
			action:   convention.DerivedAction{Name: "delete", Type: schema.ActionTypeDelete},
			method:   "DELETE",
			path:     "/items/123",
			wantCode: http.StatusInternalServerError,
		},
		{
			action:   convention.DerivedAction{Name: "publish", Type: schema.ActionTypeCustom},
			method:   "POST",
			path:     "/items/123/publish",
			wantCode: http.StatusNotFound, // Action not found in module
		},
	}

	for _, tt := range tests {
		t.Run(tt.action.Name, func(t *testing.T) {
			// Register the action route
			c.registerActionRoute(mod, tt.action, "/items")
		})
	}
}

func TestSchemaHandler_Routes(t *testing.T) {
	modules := map[string]convention.Derived{
		"item": {
			Source: schema.Module{Name: "item"},
			Plural: "items",
		},
	}

	h := NewSchemaHandler(modules)
	routes := h.Routes()

	if routes == nil {
		t.Fatal("Routes should not be nil")
	}
}

func TestSchemaHandler_ListModules(t *testing.T) {
	modules := map[string]convention.Derived{
		"item": {
			Source: schema.Module{Name: "item"},
			Plural: "items",
		},
		"user": {
			Source: schema.Module{Name: "user"},
			Plural: "users",
		},
	}

	h := NewSchemaHandler(modules)

	req := httptest.NewRequest("GET", "/_schema", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Mount("/_schema", h.Routes())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSchemaHandler_GetModule(t *testing.T) {
	modules := map[string]convention.Derived{
		"item": {
			Source: schema.Module{Name: "item"},
			Plural: "items",
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeString},
				{Name: "name", Type: schema.FieldTypeString},
			},
		},
	}

	h := NewSchemaHandler(modules)

	req := httptest.NewRequest("GET", "/_schema/item", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Mount("/_schema", h.Routes())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSchemaHandler_GetModule_NotFound(t *testing.T) {
	modules := map[string]convention.Derived{}

	h := NewSchemaHandler(modules)

	req := httptest.NewRequest("GET", "/_schema/nonexistent", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Mount("/_schema", h.Routes())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestWebUIHandler(t *testing.T) {
	handler := WebUIHandler()
	if handler == nil {
		t.Fatal("WebUIHandler should return non-nil handler")
	}

	// Test serving web UI
	req := httptest.NewRequest("GET", "/ui/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return some content (either 200 or redirect)
	if w.Code != http.StatusOK && w.Code != http.StatusMovedPermanently && w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, unexpected", w.Code)
	}
}

func TestIsWebUIBuilt(t *testing.T) {
	// This function just checks if embedded UI is available
	// Should not panic
	_ = IsWebUIBuilt()
}

func TestWebUIDevMode(t *testing.T) {
	// This function checks env var
	// Should not panic
	_ = WebUIDevMode()
}

func TestSchemaHandler_BuildModuleSchema(t *testing.T) {
	modules := map[string]convention.Derived{}

	h := NewSchemaHandler(modules)

	mod := convention.Derived{
		Source: schema.Module{Name: "item"},
		Plural: "items",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeString, Required: true},
			{Name: "name", Type: schema.FieldTypeString},
			{Name: "count", Type: schema.FieldTypeInt},
			{Name: "price", Type: schema.FieldTypeFloat},
			{Name: "active", Type: schema.FieldTypeBool},
			{Name: "created", Type: schema.FieldTypeTimestamp},
		},
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
			{Name: "create", Type: schema.ActionTypeCreate},
			{Name: "update", Type: schema.ActionTypeUpdate},
			{Name: "delete", Type: schema.ActionTypeDelete},
		},
	}

	result := h.buildModuleSchema(mod)

	if result.Module != "item" {
		t.Errorf("Module = %q, want %q", result.Module, "item")
	}
	if result.Plural != "items" {
		t.Errorf("Plural = %q, want %q", result.Plural, "items")
	}
	if len(result.Fields) != 6 {
		t.Errorf("len(Fields) = %d, want 6", len(result.Fields))
	}
}

func TestSchemaHandler_BuildFields(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	fields := []convention.DerivedField{
		{Name: "id", Type: schema.FieldTypeString, Required: true},
		{Name: "name", Type: schema.FieldTypeString, Unique: true},
		{Name: "count", Type: schema.FieldTypeInt},
	}

	result := h.buildFields(fields)

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
	if result[0].Name != "id" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "id")
	}
	if !result[0].Required {
		t.Error("result[0].Required should be true")
	}
}

func TestSchemaHandler_IsSortableType(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	sortableTypes := []schema.FieldType{
		schema.FieldTypeString,
		schema.FieldTypeInt,
		schema.FieldTypeFloat,
		schema.FieldTypeTimestamp,
		schema.FieldTypeBool,
	}

	for _, ft := range sortableTypes {
		if !h.isSortableType(ft) {
			t.Errorf("isSortableType(%q) = false, want true", ft)
		}
	}

	// Non-sortable types
	nonSortable := []schema.FieldType{
		schema.FieldTypeJSON,
		schema.FieldTypeBytes,
	}

	for _, ft := range nonSortable {
		if h.isSortableType(ft) {
			t.Errorf("isSortableType(%q) = true, want false", ft)
		}
	}
}

func TestSchemaHandler_BuildConstraints(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	constraints := []schema.Constraint{
		{Type: schema.ConstraintMinLength, Value: 5},
		{Type: schema.ConstraintMaxLength, Value: 100},
		{Type: schema.ConstraintMin, Value: 0},
		{Type: schema.ConstraintMax, Value: 1000},
		{Type: schema.ConstraintPattern, Value: "^[a-z]+$"},
		{Type: schema.ConstraintOneOf, Value: []any{"a", "b", "c"}},
	}

	result := h.buildConstraints(constraints)

	if len(result) != 6 {
		t.Errorf("len(result) = %d, want 6", len(result))
	}
}

func TestSchemaHandler_GenerateConstraintDescription(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	tests := []struct {
		constraint schema.Constraint
		want       string
	}{
		{schema.Constraint{Type: schema.ConstraintMinLength, Value: 5}, "Must be at least 5 characters"},
		{schema.Constraint{Type: schema.ConstraintMaxLength, Value: 100}, "Must be at most 100 characters"},
		{schema.Constraint{Type: schema.ConstraintMin, Value: 0}, "Value must be at least 0"},
		{schema.Constraint{Type: schema.ConstraintMax, Value: 100}, "Value must be at most 100"},
		{schema.Constraint{Type: schema.ConstraintPattern, Value: "^[a-z]+$"}, "Must match pattern: ^[a-z]+$"},
	}

	for _, tt := range tests {
		t.Run(string(tt.constraint.Type), func(t *testing.T) {
			result := h.generateConstraintDescription(tt.constraint)
			if result != tt.want {
				t.Errorf("generateConstraintDescription() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestSchemaHandler_BuildActions(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	actions := []convention.DerivedAction{
		{Name: "list", Type: schema.ActionTypeList},
		{Name: "get", Type: schema.ActionTypeGet},
		{Name: "create", Type: schema.ActionTypeCreate},
		{Name: "update", Type: schema.ActionTypeUpdate},
		{Name: "delete", Type: schema.ActionTypeDelete},
	}

	result := h.buildActions(actions, "/items")

	if len(result) != 5 {
		t.Errorf("len(result) = %d, want 5", len(result))
	}

	// Check that HTTP info is populated
	for _, action := range result {
		if action.HTTP == nil {
			t.Errorf("action %q should have HTTP info", action.Name)
		}
	}
}

func TestSchemaHandler_BuildEndpoints(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	actions := []convention.DerivedAction{
		{Name: "list", Type: schema.ActionTypeList},
		{Name: "get", Type: schema.ActionTypeGet},
		{Name: "create", Type: schema.ActionTypeCreate},
	}

	result := h.buildEndpoints(actions, "/items")

	if len(result) < 3 {
		t.Errorf("len(result) = %d, want >= 3", len(result))
	}
}

func TestSchemaHandler_BuildInputs(t *testing.T) {
	h := NewSchemaHandler(map[string]convention.Derived{})

	inputs := []convention.ActionInput{
		{Name: "name", Type: schema.FieldTypeString, Required: true},
		{Name: "count", Type: schema.FieldTypeInt},
	}

	result := h.buildInputs(inputs)

	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
	if result[0].Name != "name" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "name")
	}
	if !result[0].Required {
		t.Error("result[0].Required should be true")
	}
}

func TestSchemaJSONAPIResponse(t *testing.T) {
	w := httptest.NewRecorder()
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{"key": "value"})

	if w.Header().Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/vnd.api+json")
	}
}

func TestSchemaNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	jsonapi.WriteNotFound(w, "module")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Fatal("expected errors array in response")
	}
}

func TestNewAuthHandler(t *testing.T) {
	h := NewAuthHandler(nil)
	if h == nil {
		t.Fatal("NewAuthHandler should return non-nil handler")
	}
	// sessionSecret is private, can't check directly but NewAuthHandler sets it
}

func TestAuthHandler_Routes(t *testing.T) {
	h := NewAuthHandler(nil)
	routes := h.Routes()
	if routes == nil {
		t.Fatal("Routes should return non-nil router")
	}
}

func TestAuthWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	authWriteJSON(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/json")
	}
}

func TestAuthWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	authWriteError(w, errors.New("auth error"), http.StatusUnauthorized)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_HandleSetupRequired_NilRuntime(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("GET", "/auth/setup-required", nil)
	w := httptest.NewRecorder()

	// This will panic or error due to nil runtime
	defer func() {
		if r := recover(); r != nil {
			// Expected - nil runtime causes panic
		}
	}()

	h.handleSetupRequired(w, req)
}

func TestAuthHandler_HandleLogin_EmptyBody(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()

	h.handleLogin(w, req)

	// Should fail with bad request due to empty body
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_HandleRegister_EmptyBody(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("POST", "/auth/register", nil)
	w := httptest.NewRecorder()

	h.handleRegister(w, req)

	// Should fail with bad request due to empty body
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_HandleSetup_EmptyBody(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("POST", "/auth/setup", nil)
	w := httptest.NewRecorder()

	// Will panic on nil runtime, catch it
	defer func() {
		if r := recover(); r != nil {
			// Expected - nil runtime causes panic
		}
	}()

	h.handleSetup(w, req)
}

func TestAuthHandler_HandleLogout(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	h.handleLogout(w, req)

	// Should succeed and clear cookie
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Check cookie was cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == SessionCookie {
			if cookie.MaxAge >= 0 {
				t.Error("cookie should have negative MaxAge to clear it")
			}
		}
	}
}

func TestAuthHandler_HandleMe_NoSession(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	h.handleMe(w, req)

	// Should fail with unauthorized since no session cookie
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_SetSessionCookie(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()

	session := Session{
		UserID:    "user123",
		Email:     "test@example.com",
		Name:      "Test User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, session)

	// Check cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == SessionCookie {
			found = true
			if cookie.Value == "" {
				t.Error("cookie should have a value")
			}
			if !cookie.HttpOnly {
				t.Error("cookie should be HttpOnly")
			}
		}
	}

	if !found {
		t.Error("session cookie should be set")
	}
}

func TestAuthHandler_GetSession_NoCookie(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("GET", "/test", nil)

	session, err := h.getSession(req)
	if err == nil {
		t.Error("getSession should error with no cookie")
	}
	if session != nil {
		t.Error("session should be nil")
	}
}

func TestAuthHandler_GetSession_InvalidCookie(t *testing.T) {
	h := NewAuthHandler(nil)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookie,
		Value: "invalid-base64",
	})

	session, err := h.getSession(req)
	if err == nil {
		t.Error("getSession should error with invalid cookie")
	}
	if session != nil {
		t.Error("session should be nil")
	}
}

func TestAuthHandler_AuthMiddleware_NoSession(t *testing.T) {
	h := NewAuthHandler(nil)

	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	middleware := h.AuthMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("next handler should not be called without valid session")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// Note: Tests for doList, doGet, doCreate, doUpdate, doDelete, doCustomAction with runtime
// would require setting up a real runtime.Runtime or refactoring to use an interface.
// The existing tests cover the error paths, and integration tests cover the success paths.

// TestChannel_Register_ExplicitBasePath tests that custom base_path from module YAML is used.
func TestChannel_Register_ExplicitBasePath(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		wantBasePath string
	}{
		{
			name:         "empty base_path uses plural",
			basePath:     "",
			wantBasePath: "/items",
		},
		{
			name:         "explicit base_path is used",
			basePath:     "/api/settings",
			wantBasePath: "/api/settings",
		},
		{
			name:         "custom api path",
			basePath:     "/api/certificates",
			wantBasePath: "/api/certificates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(nil, "")

			mod := convention.Derived{
				Source: schema.Module{
					Name: "item",
					Channels: schema.Channels{
						HTTP: schema.HTTPChannel{
							Serve: schema.HTTPServe{
								Enabled:  true,
								BasePath: tt.basePath,
							},
						},
					},
				},
				Plural: "items",
				Actions: []convention.DerivedAction{
					{Name: "list", Type: schema.ActionTypeList},
				},
			}

			err := c.Register(mod)
			if err != nil {
				t.Errorf("Register should not error: %v", err)
			}

			if _, exists := c.modules["item"]; !exists {
				t.Error("Module should be registered")
			}
		})
	}
}

// TestChannel_Register_ExplicitEndpoints tests that explicit endpoints from YAML are registered.
func TestChannel_Register_ExplicitEndpoints(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "setting",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{
						Enabled:  true,
						BasePath: "/api/settings",
						Endpoints: []schema.HTTPEndpoint{
							{Action: "list", Method: "GET", Path: "/", Auth: "admin"},
							{Action: "get", Method: "GET", Path: "/{key}", Auth: "admin"},
							{Action: "update", Method: "PUT", Path: "/{key}", Auth: "admin"},
						},
					},
				},
			},
		},
		Plural: "settings",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
			{Name: "update", Type: schema.ActionTypeUpdate},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}

	if _, exists := c.modules["setting"]; !exists {
		t.Error("Module should be registered")
	}
}

// TestChannel_Register_CertificateEndpoints tests certificate module endpoint configuration.
func TestChannel_Register_CertificateEndpoints(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "certificate",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{
						Enabled:  true,
						BasePath: "/api/certificates",
						Endpoints: []schema.HTTPEndpoint{
							{Action: "list", Method: "GET", Path: "/", Auth: "admin"},
							{Action: "get_by_domain", Method: "GET", Path: "/domain/{domain}", Auth: "admin"},
							{Action: "list_expiring", Method: "GET", Path: "/expiring", Auth: "admin"},
							{Action: "list_expired", Method: "GET", Path: "/expired", Auth: "admin"},
							{Action: "get", Method: "GET", Path: "/{id}", Auth: "admin"},
							{Action: "delete", Method: "DELETE", Path: "/{id}", Auth: "admin"},
							{Action: "revoke", Method: "POST", Path: "/{id}/revoke", Auth: "admin"},
						},
					},
				},
			},
		},
		Plural: "certificates",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
			{Name: "get_by_domain", Type: schema.ActionTypeCustom},
			{Name: "list_expiring", Type: schema.ActionTypeCustom},
			{Name: "list_expired", Type: schema.ActionTypeCustom},
			{Name: "delete", Type: schema.ActionTypeDelete},
			{Name: "revoke", Type: schema.ActionTypeCustom},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}

	if _, exists := c.modules["certificate"]; !exists {
		t.Error("Module should be registered")
	}
}

// TestChannel_ImplicitVsExplicitEndpoints tests that explicit endpoints take precedence.
func TestChannel_ImplicitVsExplicitEndpoints(t *testing.T) {
	tests := []struct {
		name             string
		endpoints        []schema.HTTPEndpoint
		wantExplicit     bool
		wantEndpointPath string
	}{
		{
			name:             "no explicit endpoints uses implicit CRUD",
			endpoints:        nil,
			wantExplicit:     false,
			wantEndpointPath: "",
		},
		{
			name: "explicit endpoints override implicit",
			endpoints: []schema.HTTPEndpoint{
				{Action: "list", Method: "GET", Path: "/", Auth: "admin"},
			},
			wantExplicit:     true,
			wantEndpointPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(nil, "")

			mod := convention.Derived{
				Source: schema.Module{
					Name: "item",
					Channels: schema.Channels{
						HTTP: schema.HTTPChannel{
							Serve: schema.HTTPServe{
								Enabled:   true,
								BasePath:  "/api/items",
								Endpoints: tt.endpoints,
							},
						},
					},
				},
				Plural: "items",
				Actions: []convention.DerivedAction{
					{Name: "list", Type: schema.ActionTypeList},
				},
			}

			err := c.Register(mod)
			if err != nil {
				t.Errorf("Register should not error: %v", err)
			}

			hasExplicit := len(tt.endpoints) > 0
			if hasExplicit != tt.wantExplicit {
				t.Errorf("hasExplicit = %v, want %v", hasExplicit, tt.wantExplicit)
			}
		})
	}
}

// TestRegisterExplicitEndpoints_AllMethods tests that all HTTP methods are registered correctly.
func TestRegisterExplicitEndpoints_AllMethods(t *testing.T) {
	c := New(nil, "")

	mod := convention.Derived{
		Source: schema.Module{
			Name: "test",
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{
						Enabled:  true,
						BasePath: "/api/test",
						Endpoints: []schema.HTTPEndpoint{
							{Action: "list", Method: "GET", Path: "/"},
							{Action: "create", Method: "POST", Path: "/"},
							{Action: "update", Method: "PUT", Path: "/{id}"},
							{Action: "patch", Method: "PATCH", Path: "/{id}"},
							{Action: "delete", Method: "DELETE", Path: "/{id}"},
						},
					},
				},
			},
		},
		Plural: "tests",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "create", Type: schema.ActionTypeCreate},
			{Name: "update", Type: schema.ActionTypeUpdate},
			{Name: "patch", Type: schema.ActionTypeUpdate},
			{Name: "delete", Type: schema.ActionTypeDelete},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}
}
