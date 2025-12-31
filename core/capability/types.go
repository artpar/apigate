// Package capability defines the capability system for pluggable providers.
//
// A capability is a contract that defines what operations are available.
// A provider is an implementation of a capability (e.g., Stripe implements Payment).
// An instance is a configured provider (e.g., stripe_prod, stripe_test).
//
// Built-in capabilities: payment, email, cache, storage, auth, queue, notification, hasher
// Custom capabilities can be added by external modules (e.g., reconciliation, analytics)
package capability

import (
	"errors"
	"fmt"
	"strings"
)

// Type represents a capability type.
// Built-in types are defined as constants.
// Custom types can be registered at runtime.
type Type string

// Built-in capability types
const (
	Unknown      Type = ""
	Payment      Type = "payment"      // Billing, subscriptions, invoicing
	Email        Type = "email"        // Transactional emails
	Cache        Type = "cache"        // Key-value caching with TTL
	Storage      Type = "storage"      // Blob/file storage (S3, disk, etc.)
	Auth         Type = "auth"         // Token generation, session management
	Queue        Type = "queue"        // Async job/message processing
	Notification Type = "notification" // Alerts (Slack, Discord, webhook)
	Hasher       Type = "hasher"       // Password/key hashing

	// Custom indicates a user-defined capability type
	Custom Type = "custom"
)

// String returns the string representation of the capability type.
func (t Type) String() string {
	return string(t)
}

// IsBuiltin returns true if this is a built-in capability type.
func (t Type) IsBuiltin() bool {
	switch t {
	case Payment, Email, Cache, Storage, Auth, Queue, Notification, Hasher:
		return true
	default:
		return false
	}
}

// IsValid returns true if the capability type is valid (not empty).
func (t Type) IsValid() bool {
	return t != Unknown && t != ""
}

// ParseType parses a string into a capability Type.
// Returns Custom type for unrecognized but valid strings (user-defined capabilities).
// Returns error for empty strings and reserved words like "unknown".
func ParseType(s string) (Type, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return Unknown, errors.New("capability type cannot be empty")
	}

	switch s {
	case "payment":
		return Payment, nil
	case "email":
		return Email, nil
	case "cache":
		return Cache, nil
	case "storage":
		return Storage, nil
	case "auth":
		return Auth, nil
	case "queue":
		return Queue, nil
	case "notification":
		return Notification, nil
	case "hasher":
		return Hasher, nil
	case "unknown", "custom":
		// Reserved words that shouldn't be used as custom capability names
		return Unknown, errors.New("capability type 'unknown' and 'custom' are reserved")
	default:
		// Allow custom capability types - this enables extensibility
		// Someone can define "reconciliation", "analytics", "cdn", etc.
		return Custom, nil
	}
}

// ProviderInfo contains metadata about a registered provider instance.
type ProviderInfo struct {
	// Name is the unique instance name (e.g., "stripe_prod", "redis_cache")
	Name string

	// Module is the module that provides this capability (e.g., "payment_stripe")
	Module string

	// Capability is the type of capability this provider implements
	Capability Type

	// CustomCapability is the string name if Capability == Custom
	// e.g., "reconciliation", "analytics"
	CustomCapability string

	// Enabled indicates if this provider instance is active
	Enabled bool

	// IsDefault indicates if this is the default provider for its capability
	IsDefault bool

	// Description is a human-readable description
	Description string

	// Config holds provider-specific configuration
	Config map[string]any
}

// Validate checks if the provider info is valid.
func (p ProviderInfo) Validate() error {
	var errs []string

	if p.Name == "" {
		errs = append(errs, "name is required")
	}

	if p.Module == "" {
		errs = append(errs, "module is required")
	}

	if !p.Capability.IsValid() {
		errs = append(errs, "capability type is required")
	}

	if p.Capability == Custom && p.CustomCapability == "" {
		errs = append(errs, "custom capability name is required when type is custom")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid provider info: %s", strings.Join(errs, ", "))
	}

	return nil
}

// CapabilityKey returns the key used to look up this provider's capability.
// For built-in types, returns the type string.
// For custom types, returns the custom capability name.
func (p ProviderInfo) CapabilityKey() string {
	if p.Capability == Custom {
		return p.CustomCapability
	}
	return p.Capability.String()
}
