// Package app provides application services that orchestrate domain logic.
package app

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/bcrypt"
)

// ProxyService handles incoming proxy requests.
type ProxyService struct {
	keys      ports.KeyStore
	users     ports.UserStore
	rateLimit ports.RateLimitStore
	usage     ports.UsageRecorder
	upstream  ports.Upstream
	clock     ports.Clock
	idGen     ports.IDGenerator

	// Route and transform services (optional - nil for simple proxy mode)
	routeService     *RouteService
	transformService *TransformService

	// Static configuration (requires restart)
	keyPrefix string

	// Dynamic configuration (hot-reloadable)
	dynamicCfg atomic.Pointer[DynamicConfig]
}

// DynamicConfig contains hot-reloadable configuration.
type DynamicConfig struct {
	Plans       []plan.Plan
	Endpoints   []plan.Endpoint
	RateBurst   int
	RateWindow  int // seconds
}

// ProxyDeps contains dependencies for ProxyService.
type ProxyDeps struct {
	Keys      ports.KeyStore
	Users     ports.UserStore
	RateLimit ports.RateLimitStore
	Usage     ports.UsageRecorder
	Upstream  ports.Upstream
	Clock     ports.Clock
	IDGen     ports.IDGenerator
}

// ProxyConfig contains configuration for ProxyService.
type ProxyConfig struct {
	KeyPrefix   string
	Plans       []plan.Plan
	Endpoints   []plan.Endpoint
	RateBurst   int
	RateWindow  int // seconds
}

// NewProxyService creates a new proxy service.
func NewProxyService(deps ProxyDeps, cfg ProxyConfig) *ProxyService {
	s := &ProxyService{
		keys:      deps.Keys,
		users:     deps.Users,
		rateLimit: deps.RateLimit,
		usage:     deps.Usage,
		upstream:  deps.Upstream,
		clock:     deps.Clock,
		idGen:     deps.IDGen,
		keyPrefix: cfg.KeyPrefix,
	}

	// Set initial dynamic config
	s.UpdateConfig(cfg.Plans, cfg.Endpoints, cfg.RateBurst, cfg.RateWindow)

	return s
}

// SetRouteService sets the route service for advanced routing.
// This enables route matching, path rewriting, and upstream selection.
func (s *ProxyService) SetRouteService(routeService *RouteService) {
	s.routeService = routeService
}

// SetTransformService sets the transform service for request/response transforms.
func (s *ProxyService) SetTransformService(transformService *TransformService) {
	s.transformService = transformService
}

// UpdateConfig updates the hot-reloadable configuration.
// This is thread-safe and can be called while handling requests.
func (s *ProxyService) UpdateConfig(plans []plan.Plan, endpoints []plan.Endpoint, rateBurst, rateWindow int) {
	cfg := &DynamicConfig{
		Plans:      plans,
		Endpoints:  endpoints,
		RateBurst:  rateBurst,
		RateWindow: rateWindow,
	}
	s.dynamicCfg.Store(cfg)
}

// getDynamicConfig returns the current dynamic configuration.
func (s *ProxyService) getDynamicConfig() *DynamicConfig {
	return s.dynamicCfg.Load()
}

// HandleResult represents the outcome of handling a request.
type HandleResult struct {
	Response proxy.Response
	Error    *proxy.ErrorResponse
	Auth     *proxy.AuthContext
}

// Handle processes an incoming proxy request.
// This method orchestrates pure domain functions with I/O operations.
func (s *ProxyService) Handle(ctx context.Context, req proxy.Request) HandleResult {
	now := s.clock.Now()

	// Get current dynamic config (hot-reloadable)
	dynCfg := s.getDynamicConfig()

	// 1. Validate API key format (PURE)
	prefix, valid := key.ValidateFormat(req.APIKey, s.keyPrefix)
	if !valid {
		return HandleResult{Error: &proxy.ErrInvalidKey}
	}

	// 2. Lookup key (I/O)
	keys, err := s.keys.Get(ctx, prefix)
	if err != nil || len(keys) == 0 {
		return HandleResult{Error: &proxy.ErrInvalidKey}
	}

	// 3. Find matching key by comparing hash (PURE comparison, I/O lookup)
	var matchedKey key.Key
	found := false
	for _, k := range keys {
		if bcrypt.CompareHashAndPassword(k.Hash, []byte(req.APIKey)) == nil {
			matchedKey = k
			found = true
			break
		}
	}
	if !found {
		return HandleResult{Error: &proxy.ErrInvalidKey}
	}

	// 4. Validate key (PURE)
	validation := key.Validate(matchedKey, now)
	if !validation.Valid {
		return HandleResult{Error: &proxy.ErrorResponse{
			Status:  401,
			Code:    validation.Reason,
			Message: reasonToMessage(validation.Reason),
		}}
	}

	// 5. Get user and check status (I/O)
	user, err := s.users.Get(ctx, matchedKey.UserID)
	if err != nil {
		return HandleResult{Error: &proxy.ErrInvalidKey}
	}
	if user.Status != "active" {
		return HandleResult{Error: &proxy.ErrorResponse{
			Status:  403,
			Code:    "user_suspended",
			Message: "Account is suspended",
		}}
	}

	// 6. Get plan and rate limit config (PURE) - uses dynamic config
	userPlan, _ := plan.FindPlan(dynCfg.Plans, user.PlanID)
	rlConfig := ratelimit.Config{
		Limit:       userPlan.RateLimitPerMinute,
		Window:      time.Duration(dynCfg.RateWindow) * time.Second,
		BurstTokens: dynCfg.RateBurst,
	}
	if rlConfig.Limit == 0 {
		rlConfig.Limit = 60 // default
	}

	// 7. Check rate limit (PURE + I/O for state)
	rlState, _ := s.rateLimit.Get(ctx, matchedKey.ID)
	rlResult, newRLState := ratelimit.Check(rlState, rlConfig, now)
	s.rateLimit.Set(ctx, matchedKey.ID, newRLState)

	if !rlResult.Allowed {
		return HandleResult{
			Error: &proxy.ErrRateLimited,
			Response: proxy.Response{
				Headers: map[string]string{
					"X-RateLimit-Remaining": "0",
					"X-RateLimit-Reset":     rlResult.ResetAt.Format("2006-01-02T15:04:05Z"),
					"Retry-After":           itoa(int(rlResult.ResetAt.Sub(now).Seconds())),
				},
			},
		}
	}

	// 8. Build auth context (PURE)
	auth := proxy.AuthContext{
		KeyID:     matchedKey.ID,
		UserID:    matchedKey.UserID,
		PlanID:    user.PlanID,
		RateLimit: rlConfig.Limit,
		Scopes:    matchedKey.Scopes,
	}

	// 9. Route matching (PURE) - optional, falls back to default upstream
	var matchedRoute *route.Route
	var pathParams map[string]string
	originalPath := req.Path

	if s.routeService != nil {
		if match := s.routeService.Match(req.Method, req.Path, req.Headers); match != nil {
			matchedRoute = match.Route
			pathParams = match.PathParams
		}
	}

	// 10. Apply request transform (PURE + Expr eval)
	if matchedRoute != nil && matchedRoute.RequestTransform != nil && s.transformService != nil {
		var err error
		req, err = s.transformService.TransformRequest(ctx, req, matchedRoute.RequestTransform, &auth)
		if err != nil {
			return HandleResult{Error: &proxy.ErrorResponse{
				Status:  500,
				Code:    "transform_error",
				Message: "Request transformation failed",
			}, Auth: &auth}
		}
	}

	// 11. Path rewriting (PURE + Expr eval)
	if matchedRoute != nil && matchedRoute.PathRewrite != "" && s.transformService != nil {
		// Build context with path params
		rewriteCtx := map[string]any{
			"path":       req.Path,
			"pathParams": pathParams,
			"method":     req.Method,
		}
		newPath, err := s.transformService.EvalString(ctx, matchedRoute.PathRewrite, rewriteCtx)
		if err == nil && newPath != "" {
			req.Path = newPath
		}
	}

	// 12. Method override (PURE)
	if matchedRoute != nil && matchedRoute.MethodOverride != "" {
		req.Method = matchedRoute.MethodOverride
	}

	// 13. Forward to upstream (I/O)
	// If route matched and has an upstream, use that upstream instead of default
	var resp proxy.Response
	var routeUpstream *route.Upstream
	if matchedRoute != nil && matchedRoute.UpstreamID != "" && s.routeService != nil {
		routeUpstream = s.routeService.GetUpstream(matchedRoute.UpstreamID)
		if routeUpstream != nil {
			// Apply upstream authentication headers
			req.Headers = s.routeService.ApplyUpstreamAuth(routeUpstream, req.Headers)
		}
	}

	// Forward to route's upstream if available, otherwise use default
	if routeUpstream != nil {
		resp, err = s.upstream.ForwardTo(ctx, req, routeUpstream)
	} else {
		resp, err = s.upstream.Forward(ctx, req)
	}
	if err != nil {
		return HandleResult{Error: &proxy.ErrUpstreamError, Auth: &auth}
	}

	// 14. Apply response transform (PURE + Expr eval)
	if matchedRoute != nil && matchedRoute.ResponseTransform != nil && s.transformService != nil {
		resp, err = s.transformService.TransformResponse(ctx, resp, matchedRoute.ResponseTransform, &auth)
		if err != nil {
			// Log error but continue with original response
		}
	}

	// 15. Calculate cost/metering value (PURE + Expr eval)
	var costMult float64 = 1.0

	if matchedRoute != nil && matchedRoute.MeteringExpr != "" && s.transformService != nil {
		// Build metering context with response data
		meteringCtx := map[string]any{
			"status":        resp.Status,
			"responseBytes": int64(len(resp.Body)),
			"requestBytes":  int64(len(req.Body)),
			"path":          originalPath,
			"method":        req.Method,
		}
		// Try to parse response body as JSON for metering expressions
		if len(resp.Body) > 0 {
			var respBody any
			if jsonErr := json.Unmarshal(resp.Body, &respBody); jsonErr == nil {
				meteringCtx["respBody"] = respBody
			}
		}

		if val, err := s.transformService.EvalFloat(ctx, matchedRoute.MeteringExpr, meteringCtx); err == nil {
			costMult = val
		}
	} else {
		// Fallback to static endpoint cost multiplier
		costMult = plan.GetCostMultiplier(dynCfg.Endpoints, req.Method, originalPath)
	}

	// 16. Record usage event (async I/O)
	event := usage.Event{
		ID:             s.idGen.New(),
		KeyID:          matchedKey.ID,
		UserID:         matchedKey.UserID,
		Method:         req.Method,
		Path:           originalPath, // Use original path for tracking
		StatusCode:     resp.Status,
		LatencyMs:      resp.LatencyMs,
		RequestBytes:   int64(len(req.Body)),
		ResponseBytes:  int64(len(resp.Body)),
		CostMultiplier: costMult,
		IPAddress:      req.RemoteIP,
		UserAgent:      req.UserAgent,
		Timestamp:      now,
	}
	s.usage.Record(event)

	// 17. Update last used (async I/O)
	go s.keys.UpdateLastUsed(ctx, matchedKey.ID, now)

	// 18. Add rate limit headers to response (PURE)
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	resp.Headers["X-RateLimit-Remaining"] = itoa(rlResult.Remaining)
	resp.Headers["X-RateLimit-Reset"] = rlResult.ResetAt.Format("2006-01-02T15:04:05Z")

	return HandleResult{
		Response: resp,
		Auth:     &auth,
	}
}

func reasonToMessage(reason string) string {
	switch reason {
	case key.ReasonExpired:
		return "API key has expired"
	case key.ReasonRevoked:
		return "API key has been revoked"
	case key.ReasonNotFound:
		return "API key not found"
	default:
		return "Invalid API key"
	}
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// StreamingHandleResult represents the outcome of handling a streaming request.
type StreamingHandleResult struct {
	StreamingResponse *StreamingResponseContext
	ModifiedRequest   *proxy.Request     // Request after transforms/rewrites
	RouteUpstream     *route.Upstream    // Route's upstream (if different from default)
	Error             *proxy.ErrorResponse
	Auth              *proxy.AuthContext
	Headers           map[string]string // Rate limit headers to add
}

// StreamingResponseContext contains everything needed to stream a response.
type StreamingResponseContext struct {
	Status       int
	Headers      map[string]string
	Body         interface{} // io.ReadCloser for streaming
	ContentType  string
	UpstreamAddr string

	// For metering after stream completes
	MatchedRoute *route.Route
	OriginalPath string
	KeyID        string
	UserID       string
}

// ShouldStream determines if a request should use streaming.
func (s *ProxyService) ShouldStream(req proxy.Request) bool {
	// Check if route service exists and can determine streaming
	if s.routeService != nil {
		if match := s.routeService.Match(req.Method, req.Path, req.Headers); match != nil {
			// Check route protocol
			switch match.Route.Protocol {
			case route.ProtocolSSE, route.ProtocolHTTPStream, route.ProtocolWebSocket:
				return true
			}
		}
	}

	// Check Accept header for SSE
	if accept, ok := req.Headers["Accept"]; ok {
		if containsIgnoreCase(accept, "text/event-stream") {
			return true
		}
	}

	return false
}

// HandleStreaming processes an incoming streaming proxy request.
// This performs auth and rate limiting, then returns the streaming response context.
// The caller is responsible for streaming the response body and closing it.
func (s *ProxyService) HandleStreaming(ctx context.Context, req proxy.Request, streamingUpstream interface {
	ForwardStreaming(ctx context.Context, req proxy.Request) (interface{ Status() int }, error)
}) StreamingHandleResult {
	now := s.clock.Now()

	// Get current dynamic config (hot-reloadable)
	dynCfg := s.getDynamicConfig()

	// 1-7: Same auth and rate limiting as Handle()
	// Validate API key format
	prefix, valid := key.ValidateFormat(req.APIKey, s.keyPrefix)
	if !valid {
		return StreamingHandleResult{Error: &proxy.ErrInvalidKey}
	}

	// Lookup key
	keys, err := s.keys.Get(ctx, prefix)
	if err != nil || len(keys) == 0 {
		return StreamingHandleResult{Error: &proxy.ErrInvalidKey}
	}

	// Find matching key
	var matchedKey key.Key
	found := false
	for _, k := range keys {
		if bcrypt.CompareHashAndPassword(k.Hash, []byte(req.APIKey)) == nil {
			matchedKey = k
			found = true
			break
		}
	}
	if !found {
		return StreamingHandleResult{Error: &proxy.ErrInvalidKey}
	}

	// Validate key
	validation := key.Validate(matchedKey, now)
	if !validation.Valid {
		return StreamingHandleResult{Error: &proxy.ErrorResponse{
			Status:  401,
			Code:    validation.Reason,
			Message: reasonToMessage(validation.Reason),
		}}
	}

	// Get user and check status
	user, err := s.users.Get(ctx, matchedKey.UserID)
	if err != nil {
		return StreamingHandleResult{Error: &proxy.ErrInvalidKey}
	}
	if user.Status != "active" {
		return StreamingHandleResult{Error: &proxy.ErrorResponse{
			Status:  403,
			Code:    "user_suspended",
			Message: "Account is suspended",
		}}
	}

	// Get plan and rate limit config
	userPlan, _ := plan.FindPlan(dynCfg.Plans, user.PlanID)
	rlConfig := ratelimit.Config{
		Limit:       userPlan.RateLimitPerMinute,
		Window:      time.Duration(dynCfg.RateWindow) * time.Second,
		BurstTokens: dynCfg.RateBurst,
	}
	if rlConfig.Limit == 0 {
		rlConfig.Limit = 60
	}

	// Check rate limit
	rlState, _ := s.rateLimit.Get(ctx, matchedKey.ID)
	rlResult, newRLState := ratelimit.Check(rlState, rlConfig, now)
	if setErr := s.rateLimit.Set(ctx, matchedKey.ID, newRLState); setErr != nil {
		// Log but don't fail
	}

	if !rlResult.Allowed {
		return StreamingHandleResult{
			Error: &proxy.ErrRateLimited,
			Headers: map[string]string{
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     rlResult.ResetAt.Format("2006-01-02T15:04:05Z"),
				"Retry-After":           itoa(int(rlResult.ResetAt.Sub(now).Seconds())),
			},
		}
	}

	// Build auth context
	auth := proxy.AuthContext{
		KeyID:     matchedKey.ID,
		UserID:    matchedKey.UserID,
		PlanID:    user.PlanID,
		RateLimit: rlConfig.Limit,
		Scopes:    matchedKey.Scopes,
	}

	// Route matching
	var matchedRoute *route.Route
	var routeUpstream *route.Upstream
	originalPath := req.Path

	if s.routeService != nil {
		if match := s.routeService.Match(req.Method, req.Path, req.Headers); match != nil {
			matchedRoute = match.Route

			// Apply request transform
			if matchedRoute.RequestTransform != nil && s.transformService != nil {
				var transformErr error
				req, transformErr = s.transformService.TransformRequest(ctx, req, matchedRoute.RequestTransform, &auth)
				if transformErr != nil {
					return StreamingHandleResult{Error: &proxy.ErrorResponse{
						Status:  500,
						Code:    "transform_error",
						Message: "Request transformation failed: " + transformErr.Error(),
					}, Auth: &auth}
				}
			}

			// Path rewriting
			if matchedRoute.PathRewrite != "" && s.transformService != nil {
				rewriteCtx := map[string]any{
					"path":       req.Path,
					"pathParams": match.PathParams,
					"method":     req.Method,
				}
				if newPath, evalErr := s.transformService.EvalString(ctx, matchedRoute.PathRewrite, rewriteCtx); evalErr == nil && newPath != "" {
					req.Path = newPath
				}
			}

			// Method override
			if matchedRoute.MethodOverride != "" {
				req.Method = matchedRoute.MethodOverride
			}

			// Get and apply upstream auth
			if matchedRoute.UpstreamID != "" {
				routeUpstream = s.routeService.GetUpstream(matchedRoute.UpstreamID)
				if routeUpstream != nil {
					req.Headers = s.routeService.ApplyUpstreamAuth(routeUpstream, req.Headers)
				}
			}
		}
	}

	// Update last used
	go s.keys.UpdateLastUsed(ctx, matchedKey.ID, now)

	// Return streaming context with modified request and upstream
	return StreamingHandleResult{
		StreamingResponse: &StreamingResponseContext{
			Headers:      make(map[string]string),
			MatchedRoute: matchedRoute,
			OriginalPath: originalPath,
			KeyID:        matchedKey.ID,
			UserID:       matchedKey.UserID,
		},
		ModifiedRequest: &req,
		RouteUpstream:   routeUpstream,
		Auth:            &auth,
		Headers: map[string]string{
			"X-RateLimit-Remaining": itoa(rlResult.Remaining),
			"X-RateLimit-Reset":     rlResult.ResetAt.Format("2006-01-02T15:04:05Z"),
		},
	}
}

// RecordStreamingUsage records usage for a completed streaming request.
func (s *ProxyService) RecordStreamingUsage(
	streamCtx *StreamingResponseContext,
	statusCode int,
	requestBytes int64,
	responseBytes int64,
	latencyMs int64,
	meteringValue float64,
	remoteIP string,
	userAgent string,
) {
	now := s.clock.Now()

	event := usage.Event{
		ID:             s.idGen.New(),
		KeyID:          streamCtx.KeyID,
		UserID:         streamCtx.UserID,
		Method:         "STREAM", // Mark as streaming
		Path:           streamCtx.OriginalPath,
		StatusCode:     statusCode,
		LatencyMs:      latencyMs,
		RequestBytes:   requestBytes,
		ResponseBytes:  responseBytes,
		CostMultiplier: meteringValue,
		IPAddress:      remoteIP,
		UserAgent:      userAgent,
		Timestamp:      now,
	}
	s.usage.Record(event)
}

// EvalStreamingMetering evaluates a metering expression for streaming responses.
// It takes the raw streaming data and builds a context with:
// - lastChunk: the last chunk of data received
// - allData: all accumulated data (if available)
// - responseBytes: total bytes streamed
// - status: HTTP status code
// - userID, planID, keyID: auth context
//
// The expression can use Expr functions like json(), sseLastData(), etc.
// Example expressions:
//   - "1" (count requests)
//   - "responseBytes / 1000" (KB-based)
//   - "json(sseLastData(allData)).usage.tokens ?? 1" (extract from SSE)
func (s *ProxyService) EvalStreamingMetering(
	ctx context.Context,
	meteringExpr string,
	status int,
	responseBytes int64,
	lastChunk []byte,
	allData []byte,
	auth *proxy.AuthContext,
) float64 {
	if s.transformService == nil || meteringExpr == "" {
		return 1.0
	}

	// Build metering context with streaming data
	meteringCtx := map[string]any{
		"status":        status,
		"responseBytes": responseBytes,
		"lastChunk":     lastChunk,
		"allData":       allData,
		"userID":        "",
		"planID":        "",
		"keyID":         "",
	}

	if auth != nil {
		meteringCtx["userID"] = auth.UserID
		meteringCtx["planID"] = auth.PlanID
		meteringCtx["keyID"] = auth.KeyID
	}

	val, err := s.transformService.EvalFloat(ctx, meteringExpr, meteringCtx)
	if err != nil {
		// Log but don't fail - return default value
		return 1.0
	}

	// Ensure non-negative
	if val < 0 {
		return 0
	}

	return val
}

func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			sLower[i] = s[i] + 32
		} else {
			sLower[i] = s[i]
		}
	}
	for i := 0; i < len(substr); i++ {
		if substr[i] >= 'A' && substr[i] <= 'Z' {
			substrLower[i] = substr[i] + 32
		} else {
			substrLower[i] = substr[i]
		}
	}
	return bytesContains(sLower, substrLower)
}

func bytesContains(s, substr []byte) bool {
	if len(substr) > len(s) {
		return false
	}
outer:
	for i := 0; i <= len(s)-len(substr); i++ {
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				continue outer
			}
		}
		return true
	}
	return false
}
