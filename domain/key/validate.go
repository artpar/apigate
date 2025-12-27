package key

import (
	"strings"
	"time"
)

// Validate checks if a key is valid at the given time.
// This is a PURE function - no side effects, deterministic.
func Validate(k Key, now time.Time) ValidationResult {
	// Check if revoked
	if k.RevokedAt != nil {
		return ValidationResult{
			Valid:  false,
			Reason: ReasonRevoked,
		}
	}

	// Check if expired
	if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
		return ValidationResult{
			Valid:  false,
			Reason: ReasonExpired,
		}
	}

	return ValidationResult{
		Valid: true,
		Key:   k,
	}
}

// ValidateFormat checks if a raw API key has valid format.
// Returns (prefix, valid). Prefix is used for database lookup.
// This is a PURE function.
func ValidateFormat(rawKey string, expectedPrefix string) (prefix string, valid bool) {
	// Must start with expected prefix
	if !strings.HasPrefix(rawKey, expectedPrefix) {
		return "", false
	}

	// Must be at least prefix + 32 chars (prefix + 64 hex chars)
	minLen := len(expectedPrefix) + 64
	if len(rawKey) < minLen {
		return "", false
	}

	// Extract prefix for lookup (first 12 chars)
	if len(rawKey) >= 12 {
		prefix = rawKey[:12]
	} else {
		prefix = rawKey
	}

	return prefix, true
}

// HasScope checks if the key has access to a given scope.
// Empty scopes means access to everything.
// This is a PURE function.
func HasScope(k Key, requiredScope string) bool {
	// No scopes = full access
	if len(k.Scopes) == 0 {
		return true
	}

	// Check if required scope is in allowed scopes
	for _, s := range k.Scopes {
		if s == requiredScope || s == "*" {
			return true
		}
	}

	return false
}

// MatchPath checks if a path matches a scope pattern.
// Supports simple wildcards: "/api/*" matches "/api/users", "/api/items/123"
// This is a PURE function.
func MatchPath(pattern, path string) bool {
	// Exact match
	if pattern == path {
		return true
	}

	// Wildcard match
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	return false
}
