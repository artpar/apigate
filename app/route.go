// Package app provides application services that orchestrate domain logic.
package app

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// RouteService manages route configuration and request matching.
type RouteService struct {
	routes    ports.RouteStore
	upstreams ports.UpstreamStore
	clock     ports.Clock
	logger    zerolog.Logger

	// Cached route data for fast matching
	cache atomic.Pointer[RouteCache]

	// Refresh interval
	refreshInterval time.Duration
	stopRefresh     chan struct{}
}

// RouteCache holds in-memory route matching state.
type RouteCache struct {
	Matcher     *route.Matcher
	Routes      []route.Route
	Upstreams   map[string]route.Upstream
	RefreshedAt time.Time
}

// RouteServiceConfig contains configuration for RouteService.
type RouteServiceConfig struct {
	RefreshInterval time.Duration // How often to reload routes from DB
}

// NewRouteService creates a new route service.
func NewRouteService(
	routes ports.RouteStore,
	upstreams ports.UpstreamStore,
	clock ports.Clock,
	logger zerolog.Logger,
	cfg RouteServiceConfig,
) *RouteService {
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 30 * time.Second
	}

	s := &RouteService{
		routes:          routes,
		upstreams:       upstreams,
		clock:           clock,
		logger:          logger.With().Str("service", "route").Logger(),
		refreshInterval: cfg.RefreshInterval,
		stopRefresh:     make(chan struct{}),
	}

	return s
}

// Start begins the background route refresh goroutine.
func (s *RouteService) Start(ctx context.Context) error {
	// Initial load
	if err := s.Reload(ctx); err != nil {
		return err
	}

	// Start background refresh
	go s.refreshLoop()

	return nil
}

// Stop stops the background refresh goroutine.
func (s *RouteService) Stop() {
	close(s.stopRefresh)
}

// refreshLoop periodically reloads routes from the database.
func (s *RouteService) refreshLoop() {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopRefresh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.Reload(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to refresh routes")
			}
			cancel()
		}
	}
}

// Reload refreshes routes from storage.
func (s *RouteService) Reload(ctx context.Context) error {
	// Load enabled routes
	routes, err := s.routes.ListEnabled(ctx)
	if err != nil {
		return err
	}

	// Load enabled upstreams
	upstreamsList, err := s.upstreams.ListEnabled(ctx)
	if err != nil {
		return err
	}

	// Build upstream map
	upstreamMap := make(map[string]route.Upstream, len(upstreamsList))
	for _, u := range upstreamsList {
		upstreamMap[u.ID] = u
	}

	// Build matcher
	matcher, err := route.NewMatcher(routes)
	if err != nil {
		return err
	}

	// Atomic swap
	cache := &RouteCache{
		Matcher:     matcher,
		Routes:      routes,
		Upstreams:   upstreamMap,
		RefreshedAt: s.clock.Now(),
	}
	s.cache.Store(cache)

	s.logger.Debug().
		Int("routes", len(routes)).
		Int("upstreams", len(upstreamsList)).
		Msg("routes reloaded")

	return nil
}

// Match finds the best matching route for a request.
// Returns nil if no route matches.
func (s *RouteService) Match(method, path string, headers map[string]string) *route.MatchResult {
	cache := s.cache.Load()
	if cache == nil || cache.Matcher == nil {
		return nil
	}
	return cache.Matcher.Match(method, path, headers)
}

// GetUpstream returns an upstream by ID.
func (s *RouteService) GetUpstream(id string) *route.Upstream {
	cache := s.cache.Load()
	if cache == nil {
		return nil
	}
	if u, ok := cache.Upstreams[id]; ok {
		return &u
	}
	return nil
}

// GetRoutes returns all cached routes.
func (s *RouteService) GetRoutes() []route.Route {
	cache := s.cache.Load()
	if cache == nil {
		return nil
	}
	return cache.Routes
}

// GetUpstreams returns all cached upstreams.
func (s *RouteService) GetUpstreams() map[string]route.Upstream {
	cache := s.cache.Load()
	if cache == nil {
		return nil
	}
	return cache.Upstreams
}

// BuildUpstreamClient creates an HTTP client configured for the given upstream.
func (s *RouteService) BuildUpstreamClient(u *route.Upstream) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        u.MaxIdleConns,
		IdleConnTimeout:     u.IdleConnTimeout,
		DisableCompression:  false,
		MaxIdleConnsPerHost: u.MaxIdleConns,
	}

	return &http.Client{
		Timeout:   u.Timeout,
		Transport: transport,
	}
}

// ResolveUpstreamURL builds the full upstream URL for a request.
func (s *RouteService) ResolveUpstreamURL(upstream *route.Upstream, path, query string) (*url.URL, error) {
	baseURL, err := url.Parse(upstream.BaseURL)
	if err != nil {
		return nil, err
	}

	return baseURL.ResolveReference(&url.URL{
		Path:     path,
		RawQuery: query,
	}), nil
}

// ApplyUpstreamAuth adds authentication headers based on upstream configuration.
func (s *RouteService) ApplyUpstreamAuth(upstream *route.Upstream, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}

	switch upstream.AuthType {
	case route.AuthNone:
		// No auth to add

	case route.AuthHeader:
		// Custom header authentication
		if upstream.AuthHeader != "" && upstream.AuthValue != "" {
			headers[upstream.AuthHeader] = expandEnvVars(upstream.AuthValue)
		}

	case route.AuthBearer:
		// Bearer token authentication
		if upstream.AuthValue != "" {
			headers["Authorization"] = "Bearer " + expandEnvVars(upstream.AuthValue)
		}

	case route.AuthBasic:
		// Basic authentication
		if upstream.AuthValue != "" {
			headers["Authorization"] = "Basic " + expandEnvVars(upstream.AuthValue)
		}
	}

	return headers
}

// expandEnvVars replaces ${VAR} patterns with environment variable values.
func expandEnvVars(s string) string {
	// Simple implementation - replace ${VAR} with env value
	result := s
	for {
		start := -1
		for i := 0; i < len(result)-1; i++ {
			if result[i] == '$' && result[i+1] == '{' {
				start = i
				break
			}
		}
		if start == -1 {
			break
		}

		end := -1
		for i := start + 2; i < len(result); i++ {
			if result[i] == '}' {
				end = i
				break
			}
		}
		if end == -1 {
			break
		}

		varName := result[start+2 : end]
		varValue := os.Getenv(varName)
		result = result[:start] + varValue + result[end+1:]
	}
	return result
}

// RouteTestRequest contains the input for testing a route.
type RouteTestRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	// Optional: test a specific route by ID (bypasses matching)
	RouteID string `json:"route_id,omitempty"`
}

// RouteTestResult contains the result of testing a route.
type RouteTestResult struct {
	// Match result
	Matched     bool              `json:"matched"`
	RouteName   string            `json:"route_name,omitempty"`
	RouteID     string            `json:"route_id,omitempty"`
	PathParams  map[string]string `json:"path_params,omitempty"`
	MatchReason string            `json:"match_reason,omitempty"`

	// Upstream info
	UpstreamName string `json:"upstream_name,omitempty"`
	UpstreamURL  string `json:"upstream_url,omitempty"`

	// Transformed request
	TransformedMethod  string            `json:"transformed_method,omitempty"`
	TransformedPath    string            `json:"transformed_path,omitempty"`
	TransformedHeaders map[string]string `json:"transformed_headers,omitempty"`
	TransformedBody    string            `json:"transformed_body,omitempty"`

	// Metering preview
	MeteringExpr   string  `json:"metering_expr,omitempty"`
	MeteringSample float64 `json:"metering_sample,omitempty"`

	// Errors
	Error string `json:"error,omitempty"`
}

// TestRoute tests route matching and transformation without making actual requests.
func (s *RouteService) TestRoute(req RouteTestRequest) RouteTestResult {
	result := RouteTestResult{
		Matched: false,
	}

	// Get current cache
	cache := s.cache.Load()
	if cache == nil || cache.Matcher == nil {
		result.Error = "Route service not initialized"
		return result
	}

	var matchedRoute *route.Route
	var pathParams map[string]string

	// If a specific route ID is provided, use that instead of matching
	if req.RouteID != "" {
		for i := range cache.Routes {
			if cache.Routes[i].ID == req.RouteID {
				matchedRoute = &cache.Routes[i]
				break
			}
		}
		if matchedRoute == nil {
			result.Error = "Route not found: " + req.RouteID
			return result
		}
		result.MatchReason = "Tested directly by route ID"
	} else {
		// Match against all routes
		matchResult := cache.Matcher.Match(req.Method, req.Path, req.Headers)
		if matchResult == nil {
			result.MatchReason = "No route matched the request"
			return result
		}
		matchedRoute = matchResult.Route
		pathParams = matchResult.PathParams
		result.MatchReason = "Matched by pattern: " + string(matchedRoute.MatchType) + " " + matchedRoute.PathPattern
	}

	result.Matched = true
	result.RouteName = matchedRoute.Name
	result.RouteID = matchedRoute.ID
	result.PathParams = pathParams

	// Get upstream
	upstream, ok := cache.Upstreams[matchedRoute.UpstreamID]
	if ok {
		result.UpstreamName = upstream.Name
		upstreamURL, err := s.ResolveUpstreamURL(&upstream, req.Path, "")
		if err == nil {
			result.UpstreamURL = upstreamURL.String()
		}
	}

	// Build transformed request
	result.TransformedMethod = req.Method
	if matchedRoute.MethodOverride != "" {
		result.TransformedMethod = matchedRoute.MethodOverride
	}

	result.TransformedPath = req.Path
	// Note: Path rewrite would need TransformService evaluation
	// For now, just show the raw path_rewrite expression if set
	if matchedRoute.PathRewrite != "" {
		result.TransformedPath = "[expr: " + matchedRoute.PathRewrite + "]"
	}

	// Apply upstream auth to show what headers would be added
	result.TransformedHeaders = make(map[string]string)
	for k, v := range req.Headers {
		result.TransformedHeaders[k] = v
	}
	if ok {
		result.TransformedHeaders = s.ApplyUpstreamAuth(&upstream, result.TransformedHeaders)
	}

	// Show request transform headers if any
	if matchedRoute.RequestTransform != nil {
		for k := range matchedRoute.RequestTransform.SetHeaders {
			result.TransformedHeaders[k] = "[expr: " + matchedRoute.RequestTransform.SetHeaders[k] + "]"
		}
		for _, k := range matchedRoute.RequestTransform.DeleteHeaders {
			delete(result.TransformedHeaders, k)
		}
	}

	// Body transformation
	result.TransformedBody = req.Body
	if matchedRoute.RequestTransform != nil && matchedRoute.RequestTransform.BodyExpr != "" {
		result.TransformedBody = "[expr: " + matchedRoute.RequestTransform.BodyExpr + "]"
	}

	// Metering info
	result.MeteringExpr = matchedRoute.MeteringExpr
	if result.MeteringExpr == "" {
		result.MeteringExpr = "1"
	}
	result.MeteringSample = 1.0 // Default sample value

	return result
}
