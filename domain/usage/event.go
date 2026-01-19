// Package usage provides usage event types and aggregation functions.
// All functions are pure - no side effects.
package usage

import "time"

// EventSource identifies the origin of a usage event.
type EventSource string

const (
	SourceProxy    EventSource = "proxy"    // Event from APIGate proxy
	SourceExternal EventSource = "external" // Event from external service via metering API
)

// Event represents a single usage event (immutable value type).
// Events can originate from the proxy (API requests) or external services (metering API).
type Event struct {
	ID             string
	KeyID          string
	UserID         string
	Method         string
	Path           string
	StatusCode     int
	LatencyMs      int64
	RequestBytes   int64
	ResponseBytes  int64
	CostMultiplier float64 // For endpoint-specific pricing
	IPAddress      string
	UserAgent      string
	Timestamp      time.Time

	// External event fields (for events submitted via metering API)
	EventType    string            // Event category: "deployment.started", "compute.minutes", etc.
	ResourceID   string            // Identifier of the resource used
	ResourceType string            // Type of resource: "deployment", "storage", etc.
	Quantity     float64           // Units consumed (default 1.0)
	Source       EventSource       // Origin: "proxy" or "external"
	SourceName   string            // Service name for external events (e.g., "hoster-service")
	Metadata     map[string]string // Arbitrary context for external events
}

// IsExternal returns true if this event was submitted via the metering API.
func (e Event) IsExternal() bool {
	return e.Source == SourceExternal
}

// EffectiveQuantity returns the quantity for billing calculation.
// For proxy events, returns 1.0. For external events, returns the Quantity field.
func (e Event) EffectiveQuantity() float64 {
	if e.IsExternal() && e.Quantity > 0 {
		return e.Quantity
	}
	return 1.0
}

// EffectiveCost returns the cost for this event, considering multipliers.
func (e Event) EffectiveCost() float64 {
	quantity := e.EffectiveQuantity()
	if e.CostMultiplier > 0 {
		return quantity * e.CostMultiplier
	}
	return quantity
}

// NewProxyEvent creates an event from a proxy request.
func NewProxyEvent(id, keyID, userID, method, path string, statusCode int, latencyMs, requestBytes, responseBytes int64, costMultiplier float64, ipAddress, userAgent string, timestamp time.Time) Event {
	return Event{
		ID:             id,
		KeyID:          keyID,
		UserID:         userID,
		Method:         method,
		Path:           path,
		StatusCode:     statusCode,
		LatencyMs:      latencyMs,
		RequestBytes:   requestBytes,
		ResponseBytes:  responseBytes,
		CostMultiplier: costMultiplier,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		Timestamp:      timestamp,
		Source:         SourceProxy,
		Quantity:       1.0,
	}
}

// NewExternalEvent creates an event from the metering API.
func NewExternalEvent(id, userID, eventType, resourceID, resourceType, sourceName string, quantity float64, metadata map[string]string, timestamp time.Time) Event {
	if quantity <= 0 {
		quantity = 1.0
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	return Event{
		ID:           id,
		UserID:       userID,
		EventType:    eventType,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Quantity:     quantity,
		Source:       SourceExternal,
		SourceName:   sourceName,
		Metadata:     metadata,
		Timestamp:    timestamp,
	}
}

// ValidEventTypes defines the allowed event types for external events.
var ValidEventTypes = map[string]bool{
	"api.request":          true,
	"deployment.created":   true,
	"deployment.started":   true,
	"deployment.stopped":   true,
	"deployment.deleted":   true,
	"compute.minutes":      true,
	"storage.gb_hours":     true,
	"bandwidth.gb":         true,
}

// IsValidEventType checks if the event type is valid.
// Custom event types (prefixed with "custom.") are always valid.
func IsValidEventType(eventType string) bool {
	if ValidEventTypes[eventType] {
		return true
	}
	// Allow custom.* event types
	if len(eventType) > 7 && eventType[:7] == "custom." {
		return true
	}
	return false
}

// Summary represents aggregated usage for a period (value type).
type Summary struct {
	UserID        string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	RequestCount  int64
	ComputeUnits  float64 // Weighted by cost multipliers
	BytesIn       int64
	BytesOut      int64
	ErrorCount    int64 // 4xx + 5xx responses
	AvgLatencyMs  int64
}

// Quota represents usage limits for a plan (value type).
type Quota struct {
	RequestsPerMonth int64
	BytesPerMonth    int64 // 0 = unlimited
}

// QuotaStatus represents current quota usage (value type).
type QuotaStatus struct {
	RequestsUsed     int64
	RequestsLimit    int64
	RequestsPercent  float64
	BytesUsed        int64
	BytesLimit       int64
	BytesPercent     float64
	IsOverQuota      bool
	OverageRequests  int64
}
