// Package entitlement provides entitlement value types and pure functions.
// Entitlements are feature flags granted by subscription plans.
package entitlement

import "time"

// ValueType determines how entitlement values are interpreted.
type ValueType string

const (
	ValueTypeBoolean ValueType = "boolean" // true/false access
	ValueTypeNumber  ValueType = "number"  // numeric limit (e.g., requests per second)
	ValueTypeString  ValueType = "string"  // arbitrary string value
)

// Category groups entitlements for organization and display.
type Category string

const (
	CategoryFeature     Category = "feature"     // Product features
	CategoryAPI         Category = "api"         // API capabilities
	CategorySupport     Category = "support"     // Support levels
	CategoryIntegration Category = "integration" // Third-party integrations
)

// Entitlement represents a feature flag or capability (immutable value type).
type Entitlement struct {
	ID           string
	Name         string // Unique identifier (e.g., "api.streaming")
	DisplayName  string // Human-readable name
	Description  string
	Category     Category
	ValueType    ValueType
	DefaultValue string // Default value when granted
	HeaderName   string // Optional HTTP header to send upstream
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PlanEntitlement represents the grant of an entitlement to a plan (immutable value type).
type PlanEntitlement struct {
	ID            string
	PlanID        string
	EntitlementID string
	Value         string // Override value, empty uses Entitlement.DefaultValue
	Notes         string
	Enabled       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserEntitlement represents a resolved entitlement for a user (computed value type).
// This is what gets cached and used at request time.
type UserEntitlement struct {
	Name        string    // Entitlement name (e.g., "api.streaming")
	DisplayName string    // Human-readable name (e.g., "Streaming Responses")
	Value       string    // Resolved value
	ValueType   ValueType // How to interpret the value
	HeaderName  string    // Optional header to send upstream
}

// GetValue returns the effective value for a plan entitlement.
// Uses the override value if set, otherwise the default value.
// This is a PURE function.
func GetValue(pe PlanEntitlement, e Entitlement) string {
	if pe.Value != "" {
		return pe.Value
	}
	return e.DefaultValue
}

// FindEntitlement finds an entitlement by ID in a list.
// This is a PURE function.
func FindEntitlement(entitlements []Entitlement, id string) (Entitlement, bool) {
	for _, e := range entitlements {
		if e.ID == id {
			return e, true
		}
	}
	return Entitlement{}, false
}

// FindByName finds an entitlement by name in a list.
// This is a PURE function.
func FindByName(entitlements []Entitlement, name string) (Entitlement, bool) {
	for _, e := range entitlements {
		if e.Name == name {
			return e, true
		}
	}
	return Entitlement{}, false
}

// ResolveForPlan resolves all entitlements for a plan.
// Returns a list of UserEntitlement with resolved values.
// This is a PURE function.
func ResolveForPlan(
	planID string,
	entitlements []Entitlement,
	planEntitlements []PlanEntitlement,
) []UserEntitlement {
	var result []UserEntitlement

	// Build entitlement lookup map
	entMap := make(map[string]Entitlement)
	for _, e := range entitlements {
		if e.Enabled {
			entMap[e.ID] = e
		}
	}

	// Resolve each plan entitlement
	for _, pe := range planEntitlements {
		if pe.PlanID != planID || !pe.Enabled {
			continue
		}

		ent, ok := entMap[pe.EntitlementID]
		if !ok {
			continue
		}

		result = append(result, UserEntitlement{
			Name:        ent.Name,
			DisplayName: ent.DisplayName,
			Value:       GetValue(pe, ent),
			ValueType:   ent.ValueType,
			HeaderName:  ent.HeaderName,
		})
	}

	return result
}

// HasEntitlement checks if a user has a specific entitlement.
// This is a PURE function.
func HasEntitlement(userEntitlements []UserEntitlement, name string) bool {
	for _, ue := range userEntitlements {
		if ue.Name == name {
			return true
		}
	}
	return false
}

// GetEntitlementValue gets the value of a specific entitlement.
// Returns empty string and false if not found.
// This is a PURE function.
func GetEntitlementValue(userEntitlements []UserEntitlement, name string) (string, bool) {
	for _, ue := range userEntitlements {
		if ue.Name == name {
			return ue.Value, true
		}
	}
	return "", false
}

// ToHeaders converts user entitlements to HTTP headers for upstream.
// Only includes entitlements that have a HeaderName set.
// This is a PURE function.
func ToHeaders(userEntitlements []UserEntitlement) map[string]string {
	headers := make(map[string]string)
	for _, ue := range userEntitlements {
		if ue.HeaderName != "" {
			headers[ue.HeaderName] = ue.Value
		}
	}
	return headers
}

// FilterByCategory returns entitlements matching a category.
// This is a PURE function.
func FilterByCategory(entitlements []Entitlement, category Category) []Entitlement {
	var result []Entitlement
	for _, e := range entitlements {
		if e.Category == category {
			result = append(result, e)
		}
	}
	return result
}
