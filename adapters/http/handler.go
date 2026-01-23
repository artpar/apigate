// Package http provides HTTP handlers for the proxy service.
package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/apigate/adapters/metrics"
	"github.com/artpar/apigate/app"
	_ "github.com/artpar/apigate/docs/swagger" // swagger docs
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/streaming"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
)

// ErrorResponseBody represents an error response body for swagger docs.
type ErrorResponseBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details for swagger docs.
type ErrorDetail struct {
	Code    string `json:"code" example:"invalid_api_key"`
	Message string `json:"message" example:"The provided API key is invalid"`
}

// VersionResponse represents the version endpoint response.
type VersionResponse struct {
	Version string `json:"version" example:"1.0.0"`
	Service string `json:"service" example:"apigate"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// ProxyHandler wraps the proxy service for HTTP handling.
type ProxyHandler struct {
	service           *app.ProxyService
	streamingUpstream ports.StreamingUpstream
	logger            zerolog.Logger
	metrics           *metrics.Collector
}

// NewProxyHandler creates a new HTTP proxy handler.
func NewProxyHandler(service *app.ProxyService, logger zerolog.Logger) *ProxyHandler {
	return &ProxyHandler{
		service: service,
		logger:  logger,
	}
}

// NewProxyHandlerWithMetrics creates a new HTTP proxy handler with metrics.
func NewProxyHandlerWithMetrics(service *app.ProxyService, logger zerolog.Logger, m *metrics.Collector) *ProxyHandler {
	return &ProxyHandler{
		service: service,
		logger:  logger,
		metrics: m,
	}
}

// SetStreamingUpstream sets the streaming upstream for SSE/streaming support.
func (h *ProxyHandler) SetStreamingUpstream(upstream ports.StreamingUpstream) {
	h.streamingUpstream = upstream
}

// ServeHTTP handles incoming proxy requests.
//
//	@Summary		Proxy request to upstream
//	@Description	Authenticates the request, applies rate limiting, forwards to upstream, and records usage
//	@Tags			Proxy
//	@Accept			json
//	@Produce		json
//	@Param			X-API-Key		header	string	false	"API Key"
//	@Param			Authorization	header	string	false	"Bearer token (format: Bearer {api_key})"
//	@Success		200				"Upstream response"
//	@Failure		401				{object}	ErrorResponseBody	"Invalid or missing API key"
//	@Failure		429				{object}	ErrorResponseBody	"Rate limit exceeded"
//	@Failure		502				{object}	ErrorResponseBody	"Upstream error"
//	@Security		ApiKeyAuth
//	@Security		BearerAuth
//	@Router			/{path} [get]
//	@Router			/{path} [post]
//	@Router			/{path} [put]
//	@Router			/{path} [delete]
//	@Router			/{path} [patch]
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract API key from header or query
	// Note: Empty API key is allowed for public routes (AuthRequired=false)
	// The proxy service will validate based on route configuration
	apiKey := extractAPIKey(r)

	// Extract session token from cookie (for session-based auth)
	sessionToken := extractSessionToken(r)

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
		APIKey:       apiKey,
		SessionToken: sessionToken,
		Method:       r.Method,
		Path:         r.URL.Path,
		Query:        r.URL.RawQuery,
		Headers:      extractHeaders(r),
		Body:         body,
		RemoteIP:     extractIP(r),
		UserAgent:    r.UserAgent(),
		TraceID:      middleware.GetReqID(ctx),
	}

	// Check if this should be a streaming request
	if h.streamingUpstream != nil && h.service.ShouldStream(req) {
		h.handleStreamingRequest(w, r, ctx, req)
		return
	}

	// Handle request (buffered)
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
		if _, err := w.Write(result.Response.Body); err != nil {
			h.logger.Error().Err(err).Msg("failed to write response body")
		}
	}
}

// handleStreamingRequest handles SSE/streaming requests.
func (h *ProxyHandler) handleStreamingRequest(w http.ResponseWriter, r *http.Request, ctx context.Context, req proxy.Request) {
	start := time.Now()

	// Auth and rate limiting (reuse the streaming handle method)
	result := h.service.HandleStreaming(ctx, req, nil)

	if result.Error != nil {
		// Add rate limit headers even on error
		for k, v := range result.Headers {
			w.Header().Set(k, v)
		}
		writeError(w, result.Error)
		return
	}

	// Use modified request (with path rewrites, transforms applied)
	streamingReq := req
	if result.ModifiedRequest != nil {
		streamingReq = *result.ModifiedRequest
	}

	// Forward to upstream with streaming (use route's upstream if available)
	var streamResp ports.StreamingResponse
	var err error
	if result.RouteUpstream != nil {
		streamResp, err = h.streamingUpstream.ForwardStreamingTo(ctx, streamingReq, result.RouteUpstream)
	} else {
		streamResp, err = h.streamingUpstream.ForwardStreaming(ctx, streamingReq)
	}
	if err != nil {
		upstreamURL := ""
		if result.RouteUpstream != nil {
			upstreamURL = result.RouteUpstream.BaseURL
		}
		h.logger.Error().
			Err(err).
			Str("path", streamingReq.Path).
			Bool("has_route_upstream", result.RouteUpstream != nil).
			Str("upstream_url", upstreamURL).
			Msg("streaming upstream error")
		writeError(w, &proxy.ErrUpstreamError)
		return
	}

	// Determine if we need to accumulate data for metering
	// Only accumulate if there's a metering expression that might need the data
	needsAccumulation := false
	meteringExpr := ""
	if result.StreamingResponse != nil && result.StreamingResponse.MatchedRoute != nil {
		meteringExpr = result.StreamingResponse.MatchedRoute.MeteringExpr
		// Accumulate if expression references allData, sseEvents, sseLastData, etc.
		if meteringExpr != "" && meteringExpr != "1" && meteringExpr != "responseBytes" {
			needsAccumulation = true
		}
	}

	// Wrap the body to track bytes (accumulate if metering needs it)
	streamReader := streaming.NewStreamReader(streamResp.Body, needsAccumulation)
	defer func() {
		if closeErr := streamReader.Close(); closeErr != nil {
			h.logger.Error().Err(closeErr).Msg("failed to close stream reader")
		}
	}()

	// Copy response headers
	for k, v := range streamResp.Headers {
		w.Header().Set(k, v)
	}

	// Add rate limit headers
	for k, v := range result.Headers {
		w.Header().Set(k, v)
	}

	// Set streaming headers if not already set
	if w.Header().Get("Content-Type") == "" && streamResp.ContentType != "" {
		w.Header().Set("Content-Type", streamResp.ContentType)
	}

	// Disable buffering for streaming
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(streamResp.Status)

	// Stream the response
	flusher, canFlush := w.(http.Flusher)

	buf := make([]byte, 4096)
	for {
		n, readErr := streamReader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				h.logger.Error().Err(writeErr).Msg("failed to write streaming response")
				break
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				h.logger.Error().Err(readErr).Msg("error reading stream")
			}
			break
		}
	}

	latencyMs := time.Since(start).Milliseconds()

	// Record usage with streaming metrics
	streamMetrics := streamReader.GetMetrics()
	meteringValue := 1.0 // Default metering

	// Evaluate metering expression if configured
	if meteringExpr != "" {
		meteringValue = h.service.EvalStreamingMetering(
			ctx,
			meteringExpr,
			streamResp.Status,
			streamMetrics.TotalBytes,
			streamMetrics.LastChunk,
			streamMetrics.AllData,
			result.Auth,
		)
	}

	h.service.RecordStreamingUsage(
		result.StreamingResponse,
		streamResp.Status,
		int64(len(req.Body)),
		streamMetrics.TotalBytes,
		latencyMs,
		meteringValue,
		req.RemoteIP,
		req.UserAgent,
	)

	// Log streaming request
	h.logger.Info().
		Str("method", req.Method).
		Str("path", req.Path).
		Str("type", "streaming").
		Int("status", streamResp.Status).
		Int64("bytes", streamMetrics.TotalBytes).
		Int64("latency_ms", latencyMs).
		Float64("metering_value", meteringValue).
		Msg("streaming request completed")
}

func (h *ProxyHandler) logRequest(ctx context.Context, req proxy.Request, result app.HandleResult) {
	event := h.logger.Info()

	planID := ""
	userID := ""
	if result.Auth != nil {
		planID = result.Auth.PlanID
		userID = result.Auth.UserID
	}

	if result.Error != nil {
		event = h.logger.Warn()
		event.Int("error_status", result.Error.Status)
		event.Str("error_code", result.Error.Code)

		// Record error metrics
		if h.metrics != nil {
			status := "4xx"
			if result.Error.Status >= 500 {
				status = "5xx"
			}
			path := metrics.NormalizePath(req.Path)
			h.metrics.RequestsTotal.WithLabelValues(req.Method, path, status, planID).Inc()

			// Record specific error types
			switch result.Error.Code {
			case "invalid_api_key", "missing_api_key":
				h.metrics.AuthFailures.WithLabelValues(result.Error.Code).Inc()
			case "rate_limit_exceeded":
				h.metrics.RateLimitHits.WithLabelValues(planID, userID).Inc()
			}
		}
	} else {
		event.Int("status", result.Response.Status)
		event.Int64("latency_ms", result.Response.LatencyMs)

		// Record success metrics
		if h.metrics != nil {
			path := metrics.NormalizePath(req.Path)
			h.metrics.RequestsTotal.WithLabelValues(req.Method, path, "2xx", planID).Inc()
			h.metrics.UsageRequests.WithLabelValues(userID, planID).Inc()

			// Record bytes transferred
			if len(req.Body) > 0 {
				h.metrics.UsageBytes.WithLabelValues(userID, planID, "request").Add(float64(len(req.Body)))
			}
			if len(result.Response.Body) > 0 {
				h.metrics.UsageBytes.WithLabelValues(userID, planID, "response").Add(float64(len(result.Response.Body)))
			}
		}
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

// extractSessionToken extracts the JWT session token from cookies.
// This enables session-based authentication for users logged in via the portal.
func extractSessionToken(r *http.Request) string {
	// Try "token" cookie (JWT from portal login)
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}
	return ""
}

// extractHeaders extracts relevant headers from the request.
// Note: Go stores the Host header in r.Host, not r.Header["Host"], so we extract it explicitly.
func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	// Add Host header from r.Host (Go stores it separately from r.Header)
	// This is critical for host-based routing (e.g., *.apps.localhost patterns)
	if r.Host != "" {
		headers["Host"] = r.Host
	}

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

// writeError writes a JSON:API error response.
func writeError(w http.ResponseWriter, err *proxy.ErrorResponse) {
	jsonapi.WriteError(w, jsonapi.Error{
		Status: strconv.Itoa(err.Status),
		Code:   err.Code,
		Title:  err.Code,
		Detail: err.Message,
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
//
//	@Summary		Liveness check
//	@Description	Returns OK if the service is running
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	map[string]string	"status: ok"
//	@Router			/health [get]
//	@Router			/health/live [get]
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness checks if the service is ready to handle traffic.
//
//	@Summary		Readiness check
//	@Description	Checks if the service and upstream are ready to handle traffic
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	map[string]string		"status: ok"
//	@Failure		503	{object}	map[string]interface{}	"status: unhealthy, error: message"
//	@Router			/health/ready [get]
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

// Version returns the service version.
//
//	@Summary		Get service version
//	@Description	Returns the version information for the APIGate service
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	VersionResponse	"Version information"
//	@Router			/version [get]
func Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VersionResponse{
		Version: "dev",
		Service: "apigate",
	})
}

// RouterConfig holds optional configuration for the router.
type RouterConfig struct {
	Metrics               *metrics.Collector
	MetricsHandler        http.Handler  // Optional metrics exporter handler (for /metrics endpoint)
	EnableOpenAPI         bool
	AdminHandler          http.Handler  // Optional admin API handler
	WebHandler            http.Handler  // Optional web UI handler (enabled by default if provided, unless WebUIEnabled is set to false)
	WebUIEnabled          *bool         // Whether to enable web UI (default: true if WebHandler provided, false otherwise). Use pointer to distinguish between "not set" and "explicitly false"
	WebUIBasePath         string        // Base path to mount web UI (default: "" = root)
	PortalHandler         http.Handler  // Optional user portal handler
	PortalAuthHandler     http.Handler  // Optional JSON API auth handler (mounted at /api/portal/auth for SPA frontends)
	DocsHandler           http.Handler  // Optional developer documentation portal handler
	ModuleHandler         http.Handler  // Optional declarative module handler (mounted at /api/v2)
	PaymentWebhookHandler http.Handler  // Optional payment webhook handler for Stripe/Paddle/LemonSqueezy
	MeterHandler          http.Handler  // Optional metering API handler (mounted at /api/v1/meter)
	RouteService          interface{}   // Optional route service for priority-based routing (uses reflection to avoid circular dependency)
}

// NewRouter creates the main HTTP router.
func NewRouter(proxyHandler *ProxyHandler, healthHandler *HealthHandler, logger zerolog.Logger) chi.Router {
	return NewRouterWithConfig(proxyHandler, healthHandler, logger, RouterConfig{})
}

// NewRouterWithConfig creates the main HTTP router with optional config.
func NewRouterWithConfig(proxyHandler *ProxyHandler, healthHandler *HealthHandler, logger zerolog.Logger, cfg RouterConfig) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewLoggingMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Metrics middleware (if enabled)
	if cfg.Metrics != nil {
		r.Use(NewMetricsMiddleware(cfg.Metrics))
	}

	// Priority route middleware (if enabled)
	// This allows database routes with priority > 0 to override built-in routes
	if cfg.RouteService != nil {
		r.Use(NewPriorityRouteMiddleware(proxyHandler, cfg.RouteService, logger))
	}

	// Health endpoints (no auth required)
	r.Get("/health", healthHandler.Liveness)
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// Metrics endpoint (prefer new exporter handler, fall back to promhttp)
	if cfg.MetricsHandler != nil {
		r.Handle("/metrics", cfg.MetricsHandler)
	} else if cfg.Metrics != nil {
		r.Handle("/metrics", promhttp.Handler())
	}

	// OpenAPI/Swagger endpoints (if enabled)
	if cfg.EnableOpenAPI {
		// Serve OpenAPI spec at well-known location
		r.Get("/.well-known/openapi.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.ServeFile(w, r, "docs/swagger/swagger.json")
		})

		// Swagger UI
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/.well-known/openapi.json"),
		))
	}

	// Version endpoint
	r.Get("/version", Version)

	// Admin API (if enabled)
	if cfg.AdminHandler != nil {
		r.Mount("/admin", cfg.AdminHandler)
	}

	// User portal (if enabled)
	if cfg.PortalHandler != nil {
		r.Mount("/portal", cfg.PortalHandler)
	}

	// Portal JSON API auth endpoints (for SPA frontends)
	// Enables login/register/logout/me at /api/portal/auth/* without API key
	if cfg.PortalAuthHandler != nil {
		r.Mount("/api/portal/auth", cfg.PortalAuthHandler)
	}

	// Developer documentation portal (if enabled)
	if cfg.DocsHandler != nil {
		r.Mount("/docs", cfg.DocsHandler)
	}

	// Module API (declarative modules, if enabled)
	// Mounted at root since modules define their own base paths (e.g., /api/users)
	if cfg.ModuleHandler != nil {
		r.Mount("/mod", cfg.ModuleHandler)
	}

	// Payment provider webhooks (Stripe, Paddle, LemonSqueezy)
	// These endpoints receive POST requests from payment providers
	// They are NOT authenticated - signature verification happens inside the handler
	if cfg.PaymentWebhookHandler != nil {
		r.Mount("/payment-webhooks", cfg.PaymentWebhookHandler)
	}

	// Metering API (for external usage event submission)
	// Mounted at /api/v1/meter for service account access
	if cfg.MeterHandler != nil {
		r.Mount("/api/v1/meter", cfg.MeterHandler)
	}

	// Web UI (if enabled) - pass through specific paths to the web handler
	// Default behavior: if WebHandler is provided, it's enabled (backward compatible)
	// Explicit disable: set WebUIEnabled to false pointer
	webUIEnabled := cfg.WebUIEnabled == nil || *cfg.WebUIEnabled // nil means "not set" -> default to true
	if cfg.WebHandler != nil && webUIEnabled {
		// Normalize base path
		basePath := strings.TrimSuffix(cfg.WebUIBasePath, "/")
		if basePath != "" && !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}

		if basePath == "" {
			// Mount at root (backward compatible)
			mountWebUIAtRoot(r, cfg.WebHandler, cfg.PortalHandler, logger)
		} else {
			// Mount at custom base path
			logger.Info().
				Str("base_path", basePath).
				Msg("mounting web UI at custom base path")
			r.Mount(basePath, cfg.WebHandler)
		}
	} else if cfg.WebHandler != nil && !webUIEnabled {
		logger.Info().Msg("web UI disabled (API-only mode)")
	}

	// Proxy handles /api/* and catch-all for unmatched routes
	r.HandleFunc("/api/*", proxyHandler.ServeHTTP)

	// Catch-all for proxy: routes not matched by web UI or other handlers
	// This allows dynamic routes (from database) to work as a fallback
	r.NotFound(proxyHandler.ServeHTTP)

	return r
}

// mountWebUIAtRoot mounts the web UI at root path (backward compatible behavior).
func mountWebUIAtRoot(r chi.Router, webHandler http.Handler, portalHandler http.Handler, logger zerolog.Logger) {
	// Root URL: redirect to portal for unauthenticated users (self-onboarding)
	// If portal is enabled, new users should land there to sign up
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		// Check if user has admin session cookie
		if _, err := req.Cookie("admin_session"); err != nil && portalHandler != nil {
			// No admin session and portal is enabled - redirect to portal
			http.Redirect(w, req, "/portal", http.StatusFound)
			return
		}
		// Has session or no portal - go to admin UI
		webHandler.ServeHTTP(w, req)
	})

		// Signup/register redirects to portal (UX: common URLs users might try)
		r.Get("/signup", func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/portal/signup", http.StatusFound)
		})
		r.Get("/register", func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/portal/signup", http.StatusFound)
		})
		// Admin setup redirect (users might try /admin/setup)
		r.Get("/admin/setup", func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/setup", http.StatusFound)
		})
		// Admin registration from invite link
		r.Get("/admin/register/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/admin/register/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/login", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/login", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/logout", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/forgot-password", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/forgot-password", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/reset-password", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/reset-password", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Legal pages
		r.Get("/terms", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/privacy", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/setup", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/setup", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/setup/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/setup/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/dashboard", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/users", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/users/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/users", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/users/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/users/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/keys", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/keys", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/keys/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Plans management
		r.Get("/plans", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/plans/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/plans", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/plans/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/plans/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/usage", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/settings", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/settings", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Payment providers
		r.Get("/payments", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/payments", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Email provider
		r.Get("/email", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/email", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Webhooks management
		r.Get("/webhooks", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/webhooks/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/webhooks", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/webhooks/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/webhooks/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/system", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Admin invites management
		r.Get("/invites", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/invites", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/invites/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Entitlements management
		r.Get("/entitlements", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/entitlements/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/entitlements", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/entitlements/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/entitlements/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Routes management
		r.Get("/routes", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/routes/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/routes", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/routes/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/routes/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// Upstreams management
		r.Get("/upstreams", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/upstreams/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/upstreams", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/upstreams/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Delete("/upstreams/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		// API endpoints for UI features
		r.Post("/api/expr/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Post("/api/routes/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Get("/partials/*", func(w http.ResponseWriter, req *http.Request) { webHandler.ServeHTTP(w, req) })
		r.Handle("/static/*", webHandler)
	}


// NewMetricsMiddleware creates middleware that records request metrics.
func NewMetricsMiddleware(m *metrics.Collector) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics for internal endpoints
			if strings.HasPrefix(r.URL.Path, "/health") || r.URL.Path == "/metrics" ||
				strings.HasPrefix(r.URL.Path, "/swagger") || strings.HasPrefix(r.URL.Path, "/.well-known") {
				next.ServeHTTP(w, r)
				return
			}

			m.RequestsInFlight.Inc()
			defer m.RequestsInFlight.Dec()

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := statusLabel(ww.Status())
			path := metrics.NormalizePath(r.URL.Path)

			m.RequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
		})
	}
}

// statusLabel returns a string label for the status code.
func statusLabel(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "other"
	}
}

// MatchResult represents a route match result (to avoid circular imports).
type MatchResult interface {
	GetRoute() RouteInfo
}

// RouteInfo provides access to route information needed for priority checking.
type RouteInfo interface {
	GetPriority() int
}

// RouteService interface for middleware to access route matching.
type RouteService interface {
	Match(method, path string, headers map[string]string) *routeMatchResult
}

// routeMatchResult wraps the actual domain/route.MatchResult for the middleware.
// This is needed because we can't directly import domain/route here without circular dependency.
type routeMatchResult struct {
	Route      *routeInfo
	PathParams map[string]string
}

type routeInfo struct {
	Priority int
}

func (r *routeInfo) GetPriority() int {
	return r.Priority
}

func (m *routeMatchResult) GetRoute() RouteInfo {
	return m.Route
}

// NewPriorityRouteMiddleware creates middleware that checks database routes before chi routing.
// This allows database routes with priority > 0 to override built-in routes.
func NewPriorityRouteMiddleware(proxyHandler *ProxyHandler, routeService interface{}, logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health and metrics endpoints - these should always use built-in handlers
			if strings.HasPrefix(r.URL.Path, "/health") || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip admin API endpoints - these should always use built-in handlers
			if strings.HasPrefix(r.URL.Path, "/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Check if there's a matching database route
			if routeService != nil {
				headers := extractHeaders(r)

				// Use reflection to call Match method on route service
				// This avoids circular dependency on domain/route package
				rsVal := reflect.ValueOf(routeService)
				matchMethod := rsVal.MethodByName("Match")

				if matchMethod.IsValid() {
					args := []reflect.Value{
						reflect.ValueOf(r.Method),
						reflect.ValueOf(r.URL.Path),
						reflect.ValueOf(headers),
					}

					results := matchMethod.Call(args)
					if len(results) > 0 && !results[0].IsNil() {
						matchResult := results[0]

						// Extract Route field
						routeField := matchResult.Elem().FieldByName("Route")
						if routeField.IsValid() && !routeField.IsNil() {
							route := routeField.Elem()

							// Extract Priority field
							priorityField := route.FieldByName("Priority")
							if priorityField.IsValid() && priorityField.Kind() == reflect.Int {
								priority := int(priorityField.Int())

								// If priority > 0, this route should override built-in routes
								if priority > 0 {
									logger.Debug().
										Str("path", r.URL.Path).
										Int("priority", priority).
										Msg("routing to high-priority database route")

									// Handle via proxy (which will respect AuthRequired)
									proxyHandler.ServeHTTP(w, r)
									return
								}
							}
						}
					}
				}
			}

			// No high-priority route found, continue to chi router (built-in routes)
			next.ServeHTTP(w, r)
		})
	}
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

			// Skip logging for health checks and metrics
			if strings.HasPrefix(r.URL.Path, "/health") || r.URL.Path == "/metrics" {
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
