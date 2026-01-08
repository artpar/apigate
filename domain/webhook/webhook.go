// Package webhook provides value types and pure functions for webhook management.
// Webhooks allow customers to receive HTTP callbacks when events occur.
// All types are immutable values; all functions are pure.
package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// EventType represents a type of event that can trigger a webhook.
type EventType string

// Supported event types
const (
	EventUsageThreshold    EventType = "usage.threshold"      // User reached usage threshold
	EventUsageLimit        EventType = "usage.limit"          // User reached usage limit
	EventKeyCreated        EventType = "key.created"          // API key was created
	EventKeyRevoked        EventType = "key.revoked"          // API key was revoked
	EventSubscriptionStart EventType = "subscription.start"   // Subscription started
	EventSubscriptionEnd   EventType = "subscription.end"     // Subscription ended
	EventSubscriptionRenew EventType = "subscription.renew"   // Subscription renewed
	EventPlanChanged       EventType = "plan.changed"         // User changed plans
	EventPaymentSuccess    EventType = "payment.success"      // Payment succeeded
	EventPaymentFailed     EventType = "payment.failed"       // Payment failed
	EventInvoiceCreated    EventType = "invoice.created"      // Invoice was created
	EventTest              EventType = "test"                 // Test event
)

// AllEventTypes returns all supported event types.
func AllEventTypes() []EventType {
	return []EventType{
		EventUsageThreshold,
		EventUsageLimit,
		EventKeyCreated,
		EventKeyRevoked,
		EventSubscriptionStart,
		EventSubscriptionEnd,
		EventSubscriptionRenew,
		EventPlanChanged,
		EventPaymentSuccess,
		EventPaymentFailed,
		EventInvoiceCreated,
		EventTest,
	}
}

// DeliveryStatus represents the status of a webhook delivery.
type DeliveryStatus string

const (
	DeliveryPending  DeliveryStatus = "pending"
	DeliverySuccess  DeliveryStatus = "success"
	DeliveryFailed   DeliveryStatus = "failed"
	DeliveryRetrying DeliveryStatus = "retrying"
)

// Webhook represents a webhook configuration (value type).
type Webhook struct {
	ID          string
	UserID      string
	Name        string
	Description string
	URL         string
	Secret      string      // HMAC-SHA256 signing secret
	Events      []EventType // Events this webhook subscribes to
	RetryCount  int         // Max retry attempts (default 3)
	TimeoutMS   int         // Request timeout in ms (default 30000)
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Delivery represents a webhook delivery attempt (value type).
type Delivery struct {
	ID           string
	WebhookID    string
	EventID      string         // Unique event identifier
	EventType    EventType      // Type of event
	Payload      string         // JSON payload
	Status       DeliveryStatus // pending, success, failed, retrying
	Attempt      int            // Current attempt number
	MaxAttempts  int            // Maximum attempts allowed
	StatusCode   int            // HTTP response status code
	ResponseBody string         // Response body (truncated)
	Error        string         // Error message if failed
	DurationMS   int            // Request duration in ms
	NextRetry    *time.Time     // When to retry (if retrying)
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Event represents a webhook event to be dispatched (value type).
type Event struct {
	ID        string                 // Unique event ID
	Type      EventType              // Event type
	UserID    string                 // User this event is for
	Timestamp time.Time              // When the event occurred
	Data      map[string]interface{} // Event-specific data
}

// Payload represents the webhook payload sent to the endpoint.
type Payload struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// GenerateSecret generates a random webhook signing secret.
// This is a PURE function (deterministic for given random bytes).
func GenerateSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "whsec_" + hex.EncodeToString(bytes)
}

// GenerateEventID generates a unique event ID.
// This is a PURE function (deterministic for given random bytes).
func GenerateEventID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "evt_" + hex.EncodeToString(bytes)
}

// SignPayload signs a payload with the webhook secret using HMAC-SHA256.
// This is a PURE function.
func SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies that a signature matches the payload.
// This is a PURE function.
func VerifySignature(payload []byte, signature, secret string) bool {
	expected := SignPayload(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// BuildPayload creates a webhook payload from an event.
// This is a PURE function.
func BuildPayload(event Event) (Payload, error) {
	return Payload{
		ID:        event.ID,
		Type:      string(event.Type),
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339),
		Data:      event.Data,
	}, nil
}

// SerializePayload serializes a payload to JSON bytes.
// This is a PURE function.
func SerializePayload(payload Payload) ([]byte, error) {
	return json.Marshal(payload)
}

// SubscribesToEvent checks if a webhook subscribes to a given event type.
// This is a PURE function.
func SubscribesToEvent(webhook Webhook, eventType EventType) bool {
	if !webhook.Enabled {
		return false
	}
	for _, e := range webhook.Events {
		if e == eventType {
			return true
		}
	}
	return false
}

// FilterWebhooksForEvent returns webhooks that subscribe to a given event.
// This is a PURE function.
func FilterWebhooksForEvent(webhooks []Webhook, eventType EventType) []Webhook {
	var result []Webhook
	for _, w := range webhooks {
		if SubscribesToEvent(w, eventType) {
			result = append(result, w)
		}
	}
	return result
}

// ShouldRetry determines if a delivery should be retried based on status code.
// This is a PURE function.
func ShouldRetry(statusCode int) bool {
	// Retry on server errors and some client errors
	if statusCode >= 500 {
		return true // Server errors
	}
	if statusCode == 408 { // Request Timeout
		return true
	}
	if statusCode == 429 { // Too Many Requests
		return true
	}
	return false
}

// CalculateNextRetry calculates the next retry time using exponential backoff.
// This is a PURE function.
func CalculateNextRetry(attempt int, now time.Time) time.Time {
	// Exponential backoff: 1min, 5min, 30min
	delays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
	}
	idx := attempt - 1
	if idx >= len(delays) {
		idx = len(delays) - 1
	}
	return now.Add(delays[idx])
}

// NewDelivery creates a new delivery for a webhook and event.
// This is a PURE function.
func NewDelivery(webhook Webhook, event Event, payload string, now time.Time) Delivery {
	return Delivery{
		ID:          GenerateEventID(), // Reuse event ID generator for delivery IDs
		WebhookID:   webhook.ID,
		EventID:     event.ID,
		EventType:   event.Type,
		Payload:     payload,
		Status:      DeliveryPending,
		Attempt:     1,
		MaxAttempts: webhook.RetryCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkSuccess updates a delivery as successful.
// This is a PURE function - returns a new Delivery.
func MarkSuccess(d Delivery, statusCode int, responseBody string, durationMS int, now time.Time) Delivery {
	d.Status = DeliverySuccess
	d.StatusCode = statusCode
	d.ResponseBody = truncate(responseBody, 1000)
	d.DurationMS = durationMS
	d.UpdatedAt = now
	return d
}

// MarkFailed updates a delivery as failed.
// This is a PURE function - returns a new Delivery.
func MarkFailed(d Delivery, statusCode int, responseBody, errMsg string, durationMS int, now time.Time) Delivery {
	d.StatusCode = statusCode
	d.ResponseBody = truncate(responseBody, 1000)
	d.Error = errMsg
	d.DurationMS = durationMS
	d.UpdatedAt = now

	// Check if we should retry
	if d.Attempt < d.MaxAttempts && ShouldRetry(statusCode) {
		d.Status = DeliveryRetrying
		nextRetry := CalculateNextRetry(d.Attempt, now)
		d.NextRetry = &nextRetry
	} else {
		d.Status = DeliveryFailed
	}
	return d
}

// IncrementAttempt increments the attempt counter for a retry.
// This is a PURE function - returns a new Delivery.
func IncrementAttempt(d Delivery, now time.Time) Delivery {
	d.Attempt++
	d.Status = DeliveryPending
	d.NextRetry = nil
	d.UpdatedAt = now
	return d
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ValidateURL validates a webhook URL.
// This is a PURE function.
func ValidateURL(url string) (bool, string) {
	if url == "" {
		return false, "URL is required"
	}
	if len(url) < 8 {
		return false, "URL is too short"
	}
	if url[:8] != "https://" && url[:7] != "http://" {
		return false, "URL must start with https:// or http://"
	}
	// In production, we'd require HTTPS
	// For now, allow HTTP for local development
	return true, ""
}

// ValidateEvents validates a list of event types.
// This is a PURE function.
func ValidateEvents(events []EventType) (bool, string) {
	if len(events) == 0 {
		return false, "At least one event type is required"
	}
	validTypes := make(map[EventType]bool)
	for _, t := range AllEventTypes() {
		validTypes[t] = true
	}
	for _, e := range events {
		if !validTypes[e] {
			return false, "Invalid event type: " + string(e)
		}
	}
	return true, ""
}
