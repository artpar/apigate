// Package http provides HTTP handlers for the proxy service.
package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// ProxyHandler wraps the proxy service for HTTP handling.
type ProxyHandler struct {
	service *app.ProxyService
	logger  zerolog.Logger
}

// NewProxyHandler creates a new HTTP proxy handler.
func NewProxyHandler(service *app.ProxyService, logger zerolog.Logger) *ProxyHandler {
	return &ProxyHandler{
		service: service,
		logger:  logger,
	}
}

// ServeHTTP handles incoming proxy requests.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract API key from header or query
	apiKey := extractAPIKey(r)
	if apiKey == "" {
		writeError(w, &proxy.ErrMissingKey)
		return
	}

	// Read request body
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to read request body")
			writeError(w, &proxy.ErrorResponse{
				Status:  400,
				Code:    "bad_request",
				Message: "Failed to read request body",
			})
			return
		}
	}

	// Build proxy request
	req := proxy.Request{
		APIKey:    apiKey,
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		Headers:   extractHeaders(r),
		Body:      body,
		RemoteIP:  extractIP(r),
		UserAgent: r.UserAgent(),
		TraceID:   middleware.GetReqID(ctx),
	}

	// Handle request
	result := h.service.Handle(ctx, req)

	// Log request
	h.logRequest(ctx, req, result)

	// Write response
	if result.Error != nil {
		// Add rate limit headers even on error
		for k, v := range result.Response.Headers {
			w.Header().Set(k, v)
		}
		writeError(w, result.Error)
		return
	}

	// Copy response headers
	for k, v := range result.Response.Headers {
		w.Header().Set(k, v)
	}

	// Write response
	w.WriteHeader(result.Response.Status)
	if len(result.Response.Body) > 0 {
		w.Write(result.Response.Body)
	}
}

func (h *ProxyHandler) logRequest(ctx context.Context, req proxy.Request, result app.HandleResult) {
	event := h.logger.Info()

	if result.Error != nil {
		event = h.logger.Warn()
		event.Int("error_status", result.Error.Status)
		event.Str("error_code", result.Error.Code)
	} else {
		event.Int("status", result.Response.Status)
		event.Int64("latency_ms", result.Response.LatencyMs)
	}

	event.
		Str("method", req.Method).
		Str("path", req.Path).
		Str("remote_ip", req.RemoteIP).
		Str("trace_id", req.TraceID)

	if result.Auth != nil {
		event.
			Str("user_id", result.Auth.UserID).
			Str("key_id", result.Auth.KeyID).
			Str("plan_id", result.Auth.PlanID)
	}

	event.Msg("proxy request")
}

// extractAPIKey extracts the API key from the request.
// Supports: Authorization header (Bearer token), X-API-Key header, api_key query param.
func extractAPIKey(r *http.Request) string {
	// Try Authorization header first (Bearer token)
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Try X-API-Key header
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	// Try query parameter
	if key := r.URL.Query().Get("api_key"); key != "" {
		return key
	}

	return ""
}

// extractHeaders extracts relevant headers from the request.
func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for k, v := range r.Header {
		// Skip sensitive and hop-by-hop headers
		lower := strings.ToLower(k)
		if lower == "authorization" || lower == "x-api-key" ||
			lower == "connection" || lower == "keep-alive" ||
			lower == "proxy-authenticate" || lower == "proxy-authorization" ||
			lower == "te" || lower == "trailers" || lower == "transfer-encoding" ||
			lower == "upgrade" {
			continue
		}
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}

// extractIP extracts the client IP from the request.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, err *proxy.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	upstream HealthChecker
}

// HealthChecker interface for checking upstream health.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(upstream HealthChecker) *HealthHandler {
	return &HealthHandler{upstream: upstream}
}

// Liveness returns a simple liveness check.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness checks if the service is ready to handle traffic.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if h.upstream != nil {
		if err := h.upstream.HealthCheck(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// NewRouter creates the main HTTP router.
func NewRouter(proxyHandler *ProxyHandler, healthHandler *HealthHandler, logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewLoggingMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health endpoints (no auth required)
	r.Get("/health", healthHandler.Liveness)
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// Version endpoint
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version": "dev",
			"service": "apigate",
		})
	})

	// Proxy all other requests
	r.HandleFunc("/*", proxyHandler.ServeHTTP)

	return r
}

// LoggingMiddleware logs HTTP requests.
type LoggingMiddleware struct {
	logger zerolog.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			// Skip logging for health checks
			if strings.HasPrefix(r.URL.Path, "/health") {
				return
			}

			logger.Debug().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).
				Str("request_id", middleware.GetReqID(r.Context())).
				Msg("http request")
		})
	}
}
