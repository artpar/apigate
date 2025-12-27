// Package ports defines interfaces (contracts) between layers.
// These interfaces enable dependency injection and testability.
// Implementations live in adapters/.
package ports

import (
	"context"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/usage"
)

// -----------------------------------------------------------------------------
// Infrastructure Ports
// -----------------------------------------------------------------------------

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

// Random abstracts randomness for testability.
type Random interface {
	// Bytes generates n random bytes.
	Bytes(n int) ([]byte, error)
	// String generates a random string of n characters.
	String(n int) (string, error)
}

// IDGenerator generates unique identifiers.
type IDGenerator interface {
	New() string
}

// -----------------------------------------------------------------------------
// Data Store Ports
// -----------------------------------------------------------------------------

// KeyStore persists API keys.
type KeyStore interface {
	// Get retrieves keys matching a prefix (for validation).
	Get(ctx context.Context, prefix string) ([]key.Key, error)

	// Create stores a new key.
	Create(ctx context.Context, k key.Key) error

	// Revoke marks a key as revoked.
	Revoke(ctx context.Context, id string, at time.Time) error

	// ListByUser returns all keys for a user.
	ListByUser(ctx context.Context, userID string) ([]key.Key, error)

	// UpdateLastUsed updates the last used timestamp.
	UpdateLastUsed(ctx context.Context, id string, at time.Time) error
}

// User represents a user account.
type User struct {
	ID        string
	Email     string
	Name      string
	StripeID  string
	PlanID    string
	Status    string // "active", "suspended", "cancelled"
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserStore persists user accounts.
type UserStore interface {
	// Get retrieves a user by ID.
	Get(ctx context.Context, id string) (User, error)

	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (User, error)

	// Create stores a new user.
	Create(ctx context.Context, u User) error

	// Update modifies an existing user.
	Update(ctx context.Context, u User) error

	// List returns users with pagination.
	List(ctx context.Context, limit, offset int) ([]User, error)

	// Count returns total user count.
	Count(ctx context.Context) (int, error)
}

// UsageStore persists usage events and summaries.
type UsageStore interface {
	// RecordBatch stores multiple usage events.
	RecordBatch(ctx context.Context, events []usage.Event) error

	// GetSummary returns aggregated usage for a period.
	GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error)

	// GetHistory returns usage summaries for past periods.
	GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error)

	// GetRecentRequests returns recent request logs.
	GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error)
}

// RateLimitStore persists rate limit state.
type RateLimitStore interface {
	// Get retrieves current rate limit state for a key.
	Get(ctx context.Context, keyID string) (ratelimit.WindowState, error)

	// Set updates rate limit state for a key.
	Set(ctx context.Context, keyID string, state ratelimit.WindowState) error
}

// SubscriptionStore persists billing subscriptions.
type SubscriptionStore interface {
	// Get retrieves a subscription by ID.
	Get(ctx context.Context, id string) (billing.Subscription, error)

	// GetByUser retrieves active subscription for a user.
	GetByUser(ctx context.Context, userID string) (billing.Subscription, error)

	// Create stores a new subscription.
	Create(ctx context.Context, sub billing.Subscription) error

	// Update modifies a subscription.
	Update(ctx context.Context, sub billing.Subscription) error
}

// InvoiceStore persists invoices.
type InvoiceStore interface {
	// Create stores a new invoice.
	Create(ctx context.Context, inv billing.Invoice) error

	// ListByUser returns invoices for a user.
	ListByUser(ctx context.Context, userID string, limit int) ([]billing.Invoice, error)

	// UpdateStatus updates invoice status.
	UpdateStatus(ctx context.Context, id string, status billing.InvoiceStatus, paidAt *time.Time) error
}

// -----------------------------------------------------------------------------
// External Service Ports
// -----------------------------------------------------------------------------

// Upstream represents the backend API being proxied.
type Upstream interface {
	// Forward sends a request to the upstream and returns the response.
	Forward(ctx context.Context, req proxy.Request) (proxy.Response, error)

	// HealthCheck verifies upstream is reachable.
	HealthCheck(ctx context.Context) error
}

// BillingProvider interfaces with payment processor (Stripe).
type BillingProvider interface {
	// CreateCustomer creates a customer in the billing system.
	CreateCustomer(ctx context.Context, email, name, userID string) (customerID string, err error)

	// CreateSubscription creates a subscription for a customer.
	CreateSubscription(ctx context.Context, customerID, priceID string) (billing.Subscription, error)

	// CancelSubscription cancels a subscription.
	CancelSubscription(ctx context.Context, subscriptionID string) error

	// ReportUsage reports metered usage.
	ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error

	// CreateInvoice creates an invoice for a customer.
	CreateInvoice(ctx context.Context, customerID string, items []billing.InvoiceItem) (billing.Invoice, error)
}

// -----------------------------------------------------------------------------
// Event Ports
// -----------------------------------------------------------------------------

// UsageRecorder accepts usage events for async processing.
type UsageRecorder interface {
	// Record queues a usage event for processing.
	// This should be non-blocking.
	Record(event usage.Event)

	// Flush forces immediate processing of queued events.
	Flush(ctx context.Context) error

	// Close stops the recorder and flushes remaining events.
	Close() error
}

// WebhookSender sends webhook notifications.
type WebhookSender interface {
	// Send delivers a webhook to the configured URL.
	Send(ctx context.Context, eventType string, payload interface{}) error
}

// -----------------------------------------------------------------------------
// Hasher Port
// -----------------------------------------------------------------------------

// Hasher provides password/key hashing.
type Hasher interface {
	// Hash generates a hash from a plaintext value.
	Hash(plaintext string) ([]byte, error)

	// Compare checks if plaintext matches hash.
	Compare(hash []byte, plaintext string) bool
}
