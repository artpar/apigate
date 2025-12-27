// Package key provides API key value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package key

import "time"

// Key represents an API key (immutable value type).
type Key struct {
	ID        string
	UserID    string
	Hash      []byte     // bcrypt hash of the full key
	Prefix    string     // First 12 chars for lookup
	Name      string
	Scopes    []string   // Optional: restrict to specific endpoints
	ExpiresAt *time.Time // nil = never expires
	RevokedAt *time.Time // nil = not revoked
	CreatedAt time.Time
	LastUsed  *time.Time
}

// ValidationResult represents the outcome of key validation (value type).
type ValidationResult struct {
	Valid  bool
	Key    Key    // Populated only if Valid=true
	Reason string // Populated only if Valid=false
}

// UserContext contains user info extracted from a valid key.
type UserContext struct {
	KeyID     string
	UserID    string
	PlanID    string
	RateLimit int      // requests per minute
	Scopes    []string
}

// CreateParams contains parameters for creating a new key.
type CreateParams struct {
	UserID    string
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

// Reasons for validation failure.
const (
	ReasonValid       = ""
	ReasonNotFound    = "key_not_found"
	ReasonExpired     = "key_expired"
	ReasonRevoked     = "key_revoked"
	ReasonBadFormat   = "invalid_format"
	ReasonUserSuspend = "user_suspended"
)
