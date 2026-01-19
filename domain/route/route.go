// Package route provides route configuration value types and pure matching functions.
// Routes define how incoming requests map to upstream services with optional transformations.
package route

import (
	"time"
)

// MatchType defines how a route pattern matches paths.
type MatchType string

const (
	MatchExact  MatchType = "exact"  // /api/users matches only /api/users
	MatchPrefix MatchType = "prefix" // /api/* matches /api/users, /api/posts
	MatchRegex  MatchType = "regex"  // /api/users/[0-9]+ matches /api/users/123
)

// HostMatchType defines how a route pattern matches hostnames.
type HostMatchType string

const (
	HostMatchNone     HostMatchType = ""         // No host matching (matches any host, default)
	HostMatchExact    HostMatchType = "exact"    // api.example.com matches only api.example.com
	HostMatchWildcard HostMatchType = "wildcard" // *.example.com matches api.example.com, www.example.com
	HostMatchRegex    HostMatchType = "regex"    // ^[a-z]+\.api\.example\.com$ for complex patterns
)

// Protocol defines the communication protocol for the route.
type Protocol string

const (
	ProtocolHTTP       Protocol = "http"        // Buffered HTTP (default)
	ProtocolHTTPStream Protocol = "http_stream" // Chunked transfer, real-time forwarding
	ProtocolSSE        Protocol = "sse"         // Server-Sent Events passthrough
	ProtocolWebSocket  Protocol = "websocket"   // Bidirectional WebSocket (future)
)

// AuthType defines how to authenticate with an upstream.
type AuthType string

const (
	AuthNone   AuthType = "none"   // No authentication
	AuthHeader AuthType = "header" // Custom header (X-API-Key, etc.)
	AuthBearer AuthType = "bearer" // Authorization: Bearer <token>
	AuthBasic  AuthType = "basic"  // Authorization: Basic <base64>
)

// Route represents a routing rule (immutable value type).
// Routes are stored in the database and loaded into memory for fast matching.
type Route struct {
	ID          string
	Name        string
	Description string

	// API Documentation (for customer-facing docs)
	ExampleRequest  string // Sample request body (JSON) shown in docs
	ExampleResponse string // Sample response body (JSON) shown in docs

	// Host matching (for multi-tenant/subdomain routing)
	HostPattern   string        // Pattern: "api.example.com", "*.example.com", regex
	HostMatchType HostMatchType // How to interpret host pattern; empty = match any host

	// Path matching criteria
	PathPattern string    // Pattern to match: "/api/v1/*", "/users/{id}", regex
	MatchType   MatchType // How to interpret pattern
	Methods     []string  // HTTP methods to match; empty = all methods
	Headers     []HeaderMatch // Optional header-based matching conditions

	// Target configuration
	UpstreamID     string // Reference to Upstream entity
	PathRewrite    string // Expr expression for path rewriting
	MethodOverride string // Override request method (e.g., GET -> POST)

	// Transformations (stored as JSON, parsed into Transform structs)
	RequestTransform  *Transform // Applied before forwarding
	ResponseTransform *Transform // Applied after receiving response

	// Metering configuration
	MeteringExpr string // Expr to extract usage value from response
	MeteringMode string // "request", "response_field", "bytes", "custom"
	MeteringUnit string // Display unit: "requests", "tokens", "data_points", "bytes" (for UI labels)

	// Protocol behavior
	Protocol Protocol // http, http_stream, sse, websocket

	// Metadata
	Priority  int  // Higher = evaluated first (for overlapping patterns)
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HeaderMatch defines header-based routing criteria.
type HeaderMatch struct {
	Name     string // Header name
	Value    string // Expected value (exact or regex if IsRegex=true)
	IsRegex  bool   // If true, Value is treated as regex
	Required bool   // If true, header must be present
}

// Transform defines request or response transformation operations.
// All string values can be Expr expressions.
type Transform struct {
	// Header operations
	SetHeaders    map[string]string `json:"set_headers,omitempty"`    // Header name -> value (can be Expr)
	DeleteHeaders []string          `json:"delete_headers,omitempty"` // Headers to remove

	// Body transformation
	BodyExpr string `json:"body_expr,omitempty"` // Expr expression that returns new body (JSON)

	// Query parameter operations
	SetQuery    map[string]string `json:"set_query,omitempty"`    // Query param name -> value (can be Expr)
	DeleteQuery []string          `json:"delete_query,omitempty"` // Query params to remove
}

// Upstream represents a backend service configuration (immutable value type).
type Upstream struct {
	ID          string
	Name        string
	Description string

	// Target
	BaseURL string        // e.g., https://api.example.com
	Timeout time.Duration // Request timeout

	// Connection pooling
	MaxIdleConns    int           // Max idle connections to keep
	IdleConnTimeout time.Duration // How long to keep idle connections

	// Authentication injection (added to every request)
	AuthType   AuthType // none, header, bearer, basic
	AuthHeader string   // Header name for AuthType=header
	AuthValue  string   // Value (encrypted at rest), supports ${ENV_VAR}

	// Metadata
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewRoute creates a new Route with sensible defaults.
func NewRoute(id, name, pathPattern string, upstreamID string) Route {
	return Route{
		ID:            id,
		Name:          name,
		HostPattern:   "",            // Default: match any host
		HostMatchType: HostMatchNone, // Default: no host matching
		PathPattern:   pathPattern,
		MatchType:     MatchPrefix,
		UpstreamID:    upstreamID,
		MeteringExpr:  "1", // Default: count requests
		MeteringMode:  "request",
		MeteringUnit:  "requests", // Display unit for UI
		Protocol:      ProtocolHTTP,
		Priority:      0,
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// NewUpstream creates a new Upstream with sensible defaults.
func NewUpstream(id, name, baseURL string) Upstream {
	return Upstream{
		ID:              id,
		Name:            name,
		BaseURL:         baseURL,
		Timeout:         30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
		AuthType:        AuthNone,
		Enabled:         true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// WithHost returns a copy of the route with host-based matching configured.
func (r Route) WithHost(pattern string, matchType HostMatchType) Route {
	r.HostPattern = pattern
	r.HostMatchType = matchType
	r.UpdatedAt = time.Now()
	return r
}

// WithRequestTransform returns a copy of the route with the given request transform.
func (r Route) WithRequestTransform(t *Transform) Route {
	r.RequestTransform = t
	r.UpdatedAt = time.Now()
	return r
}

// WithResponseTransform returns a copy of the route with the given response transform.
func (r Route) WithResponseTransform(t *Transform) Route {
	r.ResponseTransform = t
	r.UpdatedAt = time.Now()
	return r
}

// WithMeteringExpr returns a copy of the route with the given metering expression.
func (r Route) WithMeteringExpr(expr string) Route {
	r.MeteringExpr = expr
	r.MeteringMode = "custom"
	r.UpdatedAt = time.Now()
	return r
}

// WithProtocol returns a copy of the route with the given protocol.
func (r Route) WithProtocol(p Protocol) Route {
	r.Protocol = p
	r.UpdatedAt = time.Now()
	return r
}

// WithAuth returns a copy of the upstream with authentication configured.
func (u Upstream) WithAuth(authType AuthType, header, value string) Upstream {
	u.AuthType = authType
	u.AuthHeader = header
	u.AuthValue = value
	u.UpdatedAt = time.Now()
	return u
}

// IsValid returns true if the route has minimum required fields.
func (r Route) IsValid() bool {
	return r.ID != "" && r.Name != "" && r.PathPattern != "" && r.UpstreamID != ""
}

// IsValid returns true if the upstream has minimum required fields.
func (u Upstream) IsValid() bool {
	return u.ID != "" && u.Name != "" && u.BaseURL != ""
}
