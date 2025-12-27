// Package app provides application services that orchestrate domain logic.
package app

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/ratelimit"
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

	// 9. Forward to upstream (I/O)
	resp, err := s.upstream.Forward(ctx, req)
	if err != nil {
		return HandleResult{Error: &proxy.ErrUpstreamError, Auth: &auth}
	}

	// 10. Calculate cost multiplier (PURE) - uses dynamic config
	costMult := plan.GetCostMultiplier(dynCfg.Endpoints, req.Method, req.Path)

	// 11. Record usage event (async I/O)
	event := usage.Event{
		ID:             s.idGen.New(),
		KeyID:          matchedKey.ID,
		UserID:         matchedKey.UserID,
		Method:         req.Method,
		Path:           req.Path,
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

	// 12. Update last used (async I/O)
	go s.keys.UpdateLastUsed(ctx, matchedKey.ID, now)

	// 13. Add rate limit headers to response (PURE)
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
