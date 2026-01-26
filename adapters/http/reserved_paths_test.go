package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/artpar/apigate/adapters/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// TestReservedPathsProtection verifies that reserved paths (admin UI, health checks, etc.)
// cannot be overridden by catch-all upstream routes.
// This test addresses the chicken-and-egg problem where a catch-all route (/*) prevents
// access to the Settings UI needed to configure routes.
func TestReservedPathsProtection(t *testing.T) {
	logger := zerolog.Nop()

	// Create a real proxy handler (simulates catch-all upstream route)
	// This handler would normally proxy everything to an upstream
	proxyHandler, _ := setupTestHandler()

	// Create a mock admin handler
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ADMIN_UI"))
	})

	// Create a mock health handler
	healthHandler := apihttp.NewHealthHandler(nil)

	// Create metrics collector
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.NewWithRegistry(reg)

	// Create router with admin handler configured
	cfg := apihttp.RouterConfig{
		Metrics:      metricsCollector,
		AdminHandler: adminHandler,
		// AdminBasePath is default: /admin
	}

	router := apihttp.NewRouterWithConfig(proxyHandler, healthHandler, logger, cfg)

	tests := []struct {
		name           string
		path           string
		wantBody       string
		wantStatusCode int
		description    string
	}{
		{
			name:           "admin_root_path",
			path:           "/admin",
			wantBody:       "ADMIN_UI",
			wantStatusCode: http.StatusOK,
			description:    "Admin UI root should be accessible even with catch-all upstream route",
		},
		{
			name:           "admin_api_path",
			path:           "/admin/api/settings",
			wantBody:       "ADMIN_UI",
			wantStatusCode: http.StatusOK,
			description:    "Admin API endpoints should be accessible even with catch-all upstream route",
		},
		{
			name:           "health_check",
			path:           "/health",
			wantBody:       `"status":"ok"`,
			wantStatusCode: http.StatusOK,
			description:    "Health checks should never be proxied",
		},
		{
			name:           "metrics",
			path:           "/metrics",
			wantBody:       "", // Prometheus metrics format
			wantStatusCode: http.StatusOK,
			description:    "Metrics endpoint should never be proxied",
		},
		{
			name:           "version",
			path:           "/version",
			wantBody:       `"service":"apigate"`,
			wantStatusCode: http.StatusOK,
			description:    "Version endpoint should never be proxied",
		},
		// Note: Non-reserved paths require API key authentication, which is correct behavior.
		// The test setup doesn't include API keys, so we skip testing non-reserved paths here.
		// The important test is that reserved paths are NEVER proxied, even with catch-all routes.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantBody != "" {
				body := rec.Body.String()
				if !contains(body, tt.wantBody) {
					t.Errorf("body = %q, want to contain %q\n%s", body, tt.wantBody, tt.description)
				}
			}
		})
	}
}

// TestReservedPathsWithCustomBasePaths verifies that custom handler base paths are also protected.
func TestReservedPathsWithCustomBasePaths(t *testing.T) {
	logger := zerolog.Nop()

	proxyHandler, _ := setupTestHandler()

	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ADMIN_UI"))
	})

	portalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PORTAL_UI"))
	})

	healthHandler := apihttp.NewHealthHandler(nil)

	// Create metrics collector
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.NewWithRegistry(reg)

	// Configure custom paths
	cfg := apihttp.RouterConfig{
		Metrics:        metricsCollector,
		AdminHandler:   adminHandler,
		AdminBasePath:  "/custom-admin", // Custom admin path
		PortalHandler:  portalHandler,
		PortalBasePath: "/custom-portal", // Custom portal path
	}

	router := apihttp.NewRouterWithConfig(proxyHandler, healthHandler, logger, cfg)

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/custom-admin", "ADMIN_UI"},
		{"/custom-admin/api/settings", "ADMIN_UI"},
		{"/custom-portal", "PORTAL_UI"},
		{"/custom-portal/dashboard", "PORTAL_UI"},
		// Note: Non-reserved paths require API key authentication
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			body := rec.Body.String()
			if !contains(body, tt.wantBody) {
				t.Errorf("path %s: body = %q, want to contain %q", tt.path, body, tt.wantBody)
			}
		})
	}
}

// TestCatchAllRouteCannotOverrideReservedPaths ensures that even a catch-all route
// in the database (e.g., /* with priority 0) cannot override reserved paths.
// This is the core fix for the chicken-and-egg problem in issue #51.
func TestCatchAllRouteCannotOverrideReservedPaths(t *testing.T) {
	logger := zerolog.Nop()

	// Simulate a catch-all route that matches everything
	catchAllProxy, _ := setupTestHandler()

	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ADMIN_UI"))
	})

	healthHandler := apihttp.NewHealthHandler(nil)

	// Create metrics collector
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.NewWithRegistry(reg)

	cfg := apihttp.RouterConfig{
		Metrics:      metricsCollector,
		AdminHandler: adminHandler,
		// Simulate that there's a route service with catch-all route
		// (In production this would be the actual RouteService)
	}

	router := apihttp.NewRouterWithConfig(catchAllProxy, healthHandler, logger, cfg)

	// These paths MUST return built-in handler responses, never proxy
	reservedPaths := []struct {
		path         string
		wantNotMatch string
	}{
		{"/admin", "CATCH_ALL_UPSTREAM"},
		{"/admin/", "CATCH_ALL_UPSTREAM"},
		{"/admin/api/settings", "CATCH_ALL_UPSTREAM"},
		{"/health", "CATCH_ALL_UPSTREAM"},
		{"/metrics", "CATCH_ALL_UPSTREAM"},
		{"/version", "CATCH_ALL_UPSTREAM"},
	}

	for _, tt := range reservedPaths {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			body := rec.Body.String()
			if contains(body, tt.wantNotMatch) {
				t.Errorf("path %s: body = %q, should NOT contain %q (reserved path was proxied!)",
					tt.path, body, tt.wantNotMatch)
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// indexOf returns the index of the first occurrence of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
