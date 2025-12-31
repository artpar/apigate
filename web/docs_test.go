package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func newTestDocsHandler() *DocsHandler {
	return NewDocsHandler(DocsDeps{
		Modules: func() map[string]convention.Derived {
			return map[string]convention.Derived{}
		},
		Routes:   newMockRoutes(),
		Settings: newMockSettingsStore(),
		Logger:   zerolog.Nop(),
		AppName:  "TestAPI",
	})
}

func TestNewDocsHandler(t *testing.T) {
	h := newTestDocsHandler()

	if h.appName != "TestAPI" {
		t.Errorf("AppName = %s, want TestAPI", h.appName)
	}
}

func TestNewDocsHandler_DefaultAppName(t *testing.T) {
	h := NewDocsHandler(DocsDeps{
		Modules: func() map[string]convention.Derived { return nil },
		Logger:  zerolog.Nop(),
		AppName: "",
	})

	if h.appName != "APIGate" {
		t.Errorf("AppName = %s, want APIGate", h.appName)
	}
}

func TestDocsHandler_Router(t *testing.T) {
	h := newTestDocsHandler()
	router := h.Router()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/"},
		{"GET", "/quickstart"},
		{"GET", "/authentication"},
		{"GET", "/api-reference"},
		{"GET", "/examples"},
		{"GET", "/try-it"},
		{"GET", "/openapi.json"},
		{"GET", "/openapi.yaml"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			r := chi.NewRouter()
			r.Mount("/docs", router)

			req := httptest.NewRequest(rt.method, "/docs"+rt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s should exist", rt.method, rt.path)
			}
		})
	}
}

func TestDocsHandler_DocsHome(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()

	h.DocsHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/html; charset=utf-8", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "TestAPI") {
		t.Error("Body should contain app name")
	}
	if !strings.Contains(body, "API Documentation") {
		t.Error("Body should contain 'API Documentation'")
	}
}

func TestDocsHandler_QuickstartPage(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/quickstart", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.QuickstartPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Quickstart") {
		t.Error("Body should contain 'Quickstart'")
	}
	if !strings.Contains(body, "http://example.com") {
		t.Error("Body should contain base URL")
	}
}

func TestDocsHandler_AuthenticationPage(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/authentication", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.AuthenticationPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Authentication") {
		t.Error("Body should contain 'Authentication'")
	}
	if !strings.Contains(body, "X-API-Key") {
		t.Error("Body should contain 'X-API-Key'")
	}
}

func TestDocsHandler_APIReferencePage(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/api-reference", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.APIReferencePage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "API Reference") {
		t.Error("Body should contain 'API Reference'")
	}
}

func TestDocsHandler_ExamplesPage(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/examples", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.ExamplesPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Code Examples") {
		t.Error("Body should contain 'Code Examples'")
	}
	if !strings.Contains(body, "curl") {
		t.Error("Body should contain curl examples")
	}
}

func TestDocsHandler_TryItPage(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/try-it", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.TryItPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Try It") {
		t.Error("Body should contain 'Try It'")
	}
	if !strings.Contains(body, "API Key") {
		t.Error("Body should contain 'API Key'")
	}
}

func TestDocsHandler_OpenAPISpec(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/openapi.json", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.OpenAPISpec(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi") {
		t.Error("Body should contain 'openapi'")
	}
}

func TestDocsHandler_OpenAPISpecYAML(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/docs/openapi.yaml", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.OpenAPISpecYAML(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/x-yaml" {
		t.Errorf("Content-Type = %s, want application/x-yaml", contentType)
	}
}

func TestDocsHandler_GetBaseURL_HTTP(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"

	url := h.getBaseURL(req)
	if url != "http://example.com" {
		t.Errorf("getBaseURL() = %s, want http://example.com", url)
	}
}

func TestDocsHandler_GetBaseURL_HTTPS_Header(t *testing.T) {
	h := newTestDocsHandler()

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	url := h.getBaseURL(req)
	if url != "https://example.com" {
		t.Errorf("getBaseURL() = %s, want https://example.com", url)
	}
}

func TestDocsHandler_RenderDocsNav(t *testing.T) {
	h := newTestDocsHandler()

	tests := []string{"home", "quickstart", "authentication", "api-reference", "examples", "try-it"}

	for _, active := range tests {
		t.Run(active, func(t *testing.T) {
			nav := h.renderDocsNav(active)

			if !strings.Contains(nav, "TestAPI Docs") {
				t.Error("Nav should contain app name")
			}
			if !strings.Contains(nav, "active") {
				t.Error("Nav should have an active item")
			}
		})
	}
}

func TestDocsHandler_RenderEndpoint(t *testing.T) {
	h := newTestDocsHandler()

	// This tests the renderEndpoint method indirectly
	req := httptest.NewRequest("GET", "/docs/api-reference", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	h.APIReferencePage(w, req)

	// Should contain endpoint classes for styling
	body := w.Body.String()
	if !strings.Contains(body, "docs-section") {
		t.Error("Body should contain docs-section class")
	}
}
