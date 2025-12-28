// Package key provides API key value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package key

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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

// Generate creates a new API key with the given prefix.
// Returns the raw key (to give to user) and the Key struct (to store).
// The raw key is: prefix + 64 hex chars (total 67 chars for "ak_" prefix).
func Generate(prefix string) (rawKey string, k Key) {
	// Generate 32 random bytes = 64 hex chars
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}

	randomHex := hex.EncodeToString(randomBytes)
	rawKey = prefix + randomHex

	// Hash the raw key
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("bcrypt failed: %v", err))
	}

	// Generate key ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	keyID := "key_" + hex.EncodeToString(idBytes)

	k = Key{
		ID:        keyID,
		Hash:      hash,
		Prefix:    rawKey[:12], // First 12 chars for lookup
		CreatedAt: time.Now().UTC(),
	}

	return rawKey, k
}

// WithUserID returns a copy of the key with the UserID set.
func (k Key) WithUserID(userID string) Key {
	k.UserID = userID
	return k
}

// WithName returns a copy of the key with the Name set.
func (k Key) WithName(name string) Key {
	k.Name = name
	return k
}
