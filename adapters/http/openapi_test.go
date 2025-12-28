package http_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/rs/zerolog"
)

func TestOpenAPI_WellKnownEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()
	healthHandler := apihttp.NewHealthHandler(nil)
	logger := zerolog.Nop()

	// Enable OpenAPI
	routerCfg := apihttp.RouterConfig{
		EnableOpenAPI: true,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, routerCfg)

	// Test /.well-known/openapi.json endpoint
	req := httptest.NewRequest("GET", "/.well-known/openapi.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	resp := rec.Result()

	// The endpoint should return 200 (it serves a file)
	// In tests, the file might not exist so it could return 404
	// We mainly want to verify the route is registered
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Errorf("status = %d, want 200 or 404", resp.StatusCode)
	}

	// Verify content type if success
	if resp.StatusCode == 200 {
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", contentType)
		}
	}
}

func TestOpenAPI_SwaggerUIEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()
	healthHandler := apihttp.NewHealthHandler(nil)
	logger := zerolog.Nop()

	// Enable OpenAPI
	routerCfg := apihttp.RouterConfig{
		EnableOpenAPI: true,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, routerCfg)

	// Test /swagger/index.html endpoint
	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	resp := rec.Result()

	// Swagger UI should return 200
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestOpenAPI_Disabled(t *testing.T) {
	handler, _ := setupTestHandler()
	healthHandler := apihttp.NewHealthHandler(nil)
	logger := zerolog.Nop()

	// OpenAPI disabled (default)
	routerCfg := apihttp.RouterConfig{
		EnableOpenAPI: false,
	}
	router := apihttp.NewRouterWithConfig(handler, healthHandler, logger, routerCfg)

	// Test /.well-known/openapi.json - should hit the proxy handler (require auth)
	req := httptest.NewRequest("GET", "/.well-known/openapi.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	resp := rec.Result()
	// When disabled, should hit proxy handler which requires auth
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401 (proxy handler auth failure)", resp.StatusCode)
	}
}

func TestVersion_Response(t *testing.T) {
	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()

	apihttp.Version(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body apihttp.VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Service != "apigate" {
		t.Errorf("service = %s, want apigate", body.Service)
	}

	if body.Version == "" {
		t.Error("version should not be empty")
	}
}
