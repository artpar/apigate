// Package proxy provides request/response value types for the proxy layer.
package proxy

import "time"

// Request represents an incoming proxy request (value type).
// This is extracted from HTTP and passed to pure functions.
type Request struct {
	// Authentication
	APIKey string

	// HTTP request details
	Method  string
	Path    string
	Query   string
	Headers map[string]string
	Body    []byte

	// Metadata
	RemoteIP  string
	UserAgent string
	TraceID   string
}

// Response represents a proxy response (value type).
type Response struct {
	// HTTP response
	Status  int
	Headers map[string]string
	Body    []byte

	// Metadata (for logging)
	LatencyMs    int64
	UpstreamAddr string
}

// AuthContext contains authenticated user information (value type).
type AuthContext struct {
	KeyID     string
	UserID    string
	PlanID    string
	RateLimit int
	Scopes    []string
}

// RequestContext combines request with auth context (value type).
type RequestContext struct {
	Request   Request
	Auth      AuthContext
	Timestamp time.Time
}

// ErrorResponse represents an error to return to client (value type).
type ErrorResponse struct {
	Status  int
	Code    string
	Message string
}

// Common error responses
var (
	ErrMissingKey = ErrorResponse{
		Status:  401,
		Code:    "missing_api_key",
		Message: "API key is required",
	}
	ErrInvalidKey = ErrorResponse{
		Status:  401,
		Code:    "invalid_api_key",
		Message: "Invalid or expired API key",
	}
	ErrRateLimited = ErrorResponse{
		Status:  429,
		Code:    "rate_limit_exceeded",
		Message: "Rate limit exceeded",
	}
	ErrQuotaExceeded = ErrorResponse{
		Status:  402,
		Code:    "quota_exceeded",
		Message: "Monthly request quota exceeded",
	}
	ErrUpstreamError = ErrorResponse{
		Status:  502,
		Code:    "upstream_error",
		Message: "Upstream service unavailable",
	}
	ErrTimeout = ErrorResponse{
		Status:  504,
		Code:    "upstream_timeout",
		Message: "Upstream service timeout",
	}
)
