// Package plan provides plan value types and pure functions.
package plan

// QuotaEnforceMode determines how quota limits are enforced.
type QuotaEnforceMode string

const (
	QuotaEnforceHard QuotaEnforceMode = "hard" // Reject requests when quota exceeded
	QuotaEnforceWarn QuotaEnforceMode = "warn" // Allow but add warning headers
	QuotaEnforceSoft QuotaEnforceMode = "soft" // Allow and bill overage
)

// Plan represents a pricing tier (immutable value type).
type Plan struct {
	ID                 string
	Name               string
	RequestsPerMonth   int64 // -1 = unlimited
	RateLimitPerMinute int
	PriceMonthly       int64 // cents
	OveragePrice       int64 // cents per request
	StripePriceID      string
	QuotaEnforceMode   QuotaEnforceMode // "hard", "warn", "soft" - defaults to "hard"
	QuotaGracePct      float64          // Grace percentage before hard block (e.g., 0.05 = 5%)
}

// Endpoint represents endpoint-specific pricing (value type).
type Endpoint struct {
	Path           string
	Method         string // GET, POST, etc. Empty = all methods
	CostMultiplier float64
}

// GetCostMultiplier returns the cost multiplier for a given endpoint.
// This is a PURE function.
func GetCostMultiplier(endpoints []Endpoint, method, path string) float64 {
	for _, e := range endpoints {
		if matchEndpoint(e, method, path) {
			return e.CostMultiplier
		}
	}
	return 1.0 // Default: 1 request = 1 unit
}

// matchEndpoint checks if an endpoint rule matches the request.
func matchEndpoint(e Endpoint, method, path string) bool {
	// Check method (empty = match all)
	if e.Method != "" && e.Method != method {
		return false
	}

	// Exact path match
	if e.Path == path {
		return true
	}

	// Prefix match (e.g., "/api/v1/*" matches "/api/v1/users")
	if len(e.Path) > 0 && e.Path[len(e.Path)-1] == '*' {
		prefix := e.Path[:len(e.Path)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	return false
}

// FindPlan finds a plan by ID in a list.
// This is a PURE function.
func FindPlan(plans []Plan, id string) (Plan, bool) {
	for _, p := range plans {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// IsUnlimited checks if a plan has unlimited requests.
// This is a PURE function.
func IsUnlimited(p Plan) bool {
	return p.RequestsPerMonth < 0
}
