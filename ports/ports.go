// Package ports defines interfaces (contracts) between layers.
// These interfaces enable dependency injection and testability.
// Implementations live in adapters/.
package ports

import (
	"context"
	"io"
	"time"

	"github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/settings"
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
	ID           string
	Email        string
	PasswordHash []byte // bcrypt hash for web UI login (optional for API-only users)
	Name         string
	StripeID     string
	PlanID       string
	Status       string // "active", "suspended", "cancelled"
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

	// Delete removes a user.
	Delete(ctx context.Context, id string) error

	// List returns users with pagination.
	List(ctx context.Context, limit, offset int) ([]User, error)

	// Count returns total user count.
	Count(ctx context.Context) (int, error)
}

// PlanStore persists pricing plans.
type PlanStore interface {
	// List returns all enabled plans.
	List(ctx context.Context) ([]Plan, error)

	// Get retrieves a plan by ID.
	Get(ctx context.Context, id string) (Plan, error)

	// Create stores a new plan.
	Create(ctx context.Context, p Plan) error

	// Update modifies a plan.
	Update(ctx context.Context, p Plan) error

	// Delete removes a plan.
	Delete(ctx context.Context, id string) error
}

// Plan represents a pricing tier.
type Plan struct {
	ID                 string
	Name               string
	Description        string
	RateLimitPerMinute int
	RequestsPerMonth   int64
	PriceMonthly       int64 // cents
	OveragePrice       int64 // cents per request
	StripePriceID      string
	PaddlePriceID      string
	LemonVariantID     string
	IsDefault          bool
	Enabled            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
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

	// ForwardTo sends a request to a specific upstream (not the default).
	// Used when a route specifies a different upstream.
	ForwardTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (proxy.Response, error)

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

// -----------------------------------------------------------------------------
// Route Ports
// -----------------------------------------------------------------------------

// RouteStore persists route configurations.
type RouteStore interface {
	// Get retrieves a route by ID.
	Get(ctx context.Context, id string) (route.Route, error)

	// List returns all routes ordered by priority.
	List(ctx context.Context) ([]route.Route, error)

	// ListEnabled returns only enabled routes ordered by priority.
	ListEnabled(ctx context.Context) ([]route.Route, error)

	// Create stores a new route.
	Create(ctx context.Context, r route.Route) error

	// Update modifies an existing route.
	Update(ctx context.Context, r route.Route) error

	// Delete removes a route.
	Delete(ctx context.Context, id string) error
}

// UpstreamStore persists upstream configurations.
type UpstreamStore interface {
	// Get retrieves an upstream by ID.
	Get(ctx context.Context, id string) (route.Upstream, error)

	// List returns all upstreams.
	List(ctx context.Context) ([]route.Upstream, error)

	// ListEnabled returns only enabled upstreams.
	ListEnabled(ctx context.Context) ([]route.Upstream, error)

	// Create stores a new upstream.
	Create(ctx context.Context, u route.Upstream) error

	// Update modifies an existing upstream.
	Update(ctx context.Context, u route.Upstream) error

	// Delete removes an upstream.
	Delete(ctx context.Context, id string) error
}

// -----------------------------------------------------------------------------
// Router Ports
// -----------------------------------------------------------------------------

// Router matches incoming requests to routes.
type Router interface {
	// Match finds the best matching route for a request.
	// Returns nil if no route matches.
	Match(method, path string, headers map[string]string) *route.MatchResult

	// Reload refreshes routes from storage.
	Reload(ctx context.Context) error
}

// -----------------------------------------------------------------------------
// Transformer Ports
// -----------------------------------------------------------------------------

// Transformer applies transformations to requests and responses.
type Transformer interface {
	// TransformRequest applies request transformations.
	TransformRequest(ctx context.Context, req proxy.Request, transform *route.Transform, auth *proxy.AuthContext) (proxy.Request, error)

	// TransformResponse applies response transformations.
	TransformResponse(ctx context.Context, resp proxy.Response, transform *route.Transform, auth *proxy.AuthContext) (proxy.Response, error)

	// EvalString evaluates an Expr expression and returns a string.
	EvalString(ctx context.Context, expr string, data map[string]any) (string, error)

	// EvalFloat evaluates an Expr expression and returns a float64.
	EvalFloat(ctx context.Context, expr string, data map[string]any) (float64, error)
}

// -----------------------------------------------------------------------------
// Streaming Ports
// -----------------------------------------------------------------------------

// StreamingResponse represents a response that may be streamed.
type StreamingResponse struct {
	Status       int
	Headers      map[string]string
	Body         io.ReadCloser // For streaming (nil if buffered)
	BodyBytes    []byte        // For buffered (nil if streaming)
	IsStreaming  bool
	ContentType  string
	LatencyMs    int64
	UpstreamAddr string
}

// StreamingUpstream extends Upstream with streaming capabilities.
type StreamingUpstream interface {
	Upstream // Embed existing interface for backward compatibility

	// ForwardStreaming returns a streaming response.
	// The caller is responsible for closing the response body.
	ForwardStreaming(ctx context.Context, req proxy.Request) (StreamingResponse, error)

	// ForwardStreamingTo sends a streaming request to a specific upstream (not the default).
	ForwardStreamingTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (StreamingResponse, error)

	// ShouldStream determines if a request should use streaming.
	ShouldStream(req proxy.Request, protocol route.Protocol) bool
}

// -----------------------------------------------------------------------------
// Authentication Ports
// -----------------------------------------------------------------------------

// TokenStore persists authentication tokens (email verification, password reset).
type TokenStore interface {
	// Create stores a new token.
	Create(ctx context.Context, token auth.Token) error

	// GetByHash retrieves a token by its hash (for validation).
	GetByHash(ctx context.Context, hash []byte) (auth.Token, error)

	// GetByUserAndType retrieves the latest token for a user of a specific type.
	GetByUserAndType(ctx context.Context, userID string, tokenType auth.TokenType) (auth.Token, error)

	// MarkUsed marks a token as used.
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error

	// DeleteExpired removes all expired tokens.
	DeleteExpired(ctx context.Context) (int64, error)

	// DeleteByUser removes all tokens for a user.
	DeleteByUser(ctx context.Context, userID string) error
}

// SessionStore persists user portal sessions.
type SessionStore interface {
	// Create stores a new session.
	Create(ctx context.Context, session auth.Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (auth.Session, error)

	// Delete removes a session (logout).
	Delete(ctx context.Context, id string) error

	// DeleteByUser removes all sessions for a user (logout everywhere).
	DeleteByUser(ctx context.Context, userID string) error

	// DeleteExpired removes all expired sessions.
	DeleteExpired(ctx context.Context) (int64, error)
}

// EmailMessage represents an email to be sent.
type EmailMessage struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

// EmailSender sends emails.
type EmailSender interface {
	// Send sends an email.
	Send(ctx context.Context, msg EmailMessage) error

	// SendVerification sends an email verification link.
	SendVerification(ctx context.Context, to, name, token string) error

	// SendPasswordReset sends a password reset link.
	SendPasswordReset(ctx context.Context, to, name, token string) error

	// SendWelcome sends a welcome email after verification.
	SendWelcome(ctx context.Context, to, name string) error
}

// -----------------------------------------------------------------------------
// Settings Ports
// -----------------------------------------------------------------------------

// SettingsStore persists application settings.
type SettingsStore interface {
	// Get retrieves a single setting by key.
	Get(ctx context.Context, key string) (settings.Setting, error)

	// GetAll retrieves all settings as a map.
	GetAll(ctx context.Context) (settings.Settings, error)

	// GetByPrefix retrieves all settings with a given prefix.
	GetByPrefix(ctx context.Context, prefix string) (settings.Settings, error)

	// Set stores or updates a setting.
	Set(ctx context.Context, key, value string, encrypted bool) error

	// SetBatch stores or updates multiple settings.
	SetBatch(ctx context.Context, s settings.Settings) error

	// Delete removes a setting.
	Delete(ctx context.Context, key string) error
}

// -----------------------------------------------------------------------------
// Payment Provider Ports
// -----------------------------------------------------------------------------

// PaymentProvider interfaces with payment processor (Stripe, Paddle, LemonSqueezy).
type PaymentProvider interface {
	// Name returns the provider name (e.g., "stripe", "paddle").
	Name() string

	// CreateCustomer creates a customer in the payment system.
	CreateCustomer(ctx context.Context, email, name, userID string) (customerID string, err error)

	// CreateCheckoutSession creates a checkout session for subscription.
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (sessionURL string, err error)

	// CreatePortalSession creates a customer portal session for managing subscription.
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (portalURL string, err error)

	// CancelSubscription cancels a subscription.
	CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

	// GetSubscription retrieves subscription details.
	GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error)

	// ReportUsage reports metered usage for billing.
	ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error

	// ParseWebhook parses and validates an incoming webhook.
	// Returns the event type and payload.
	ParseWebhook(payload []byte, signature string) (eventType string, data map[string]any, err error)
}

// PaymentWebhookHandler handles payment provider webhooks.
type PaymentWebhookHandler interface {
	// HandleCheckoutCompleted handles successful checkout.
	HandleCheckoutCompleted(ctx context.Context, customerID, subscriptionID, planID string) error

	// HandleSubscriptionUpdated handles subscription changes.
	HandleSubscriptionUpdated(ctx context.Context, subscriptionID string, status billing.SubscriptionStatus) error

	// HandleSubscriptionCancelled handles subscription cancellation.
	HandleSubscriptionCancelled(ctx context.Context, subscriptionID string) error

	// HandleInvoicePaid handles successful invoice payment.
	HandleInvoicePaid(ctx context.Context, invoiceID, customerID string, amountPaid int64) error

	// HandleInvoiceFailed handles failed invoice payment.
	HandleInvoiceFailed(ctx context.Context, invoiceID, customerID string) error
}
