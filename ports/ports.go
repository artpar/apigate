// Package ports defines interfaces (contracts) between layers.
// These interfaces enable dependency injection and testability.
// Implementations live in adapters/.
package ports

import (
	"errors"
	"context"
	"io"
	"time"

	"github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/entitlement"
	"github.com/artpar/apigate/domain/group"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/oauth"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/domain/tls"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/domain/webhook"
)

// -----------------------------------------------------------------------------
// Common Errors
// -----------------------------------------------------------------------------

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("not found")

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
// Note: Provider-specific customer IDs are stored in provider_mapping module,
// not in the User struct. Use ProviderMappingStore to lookup external IDs.
type User struct {
	ID           string
	Email        string
	PasswordHash []byte // bcrypt hash for web UI login (optional for API-only users)
	Name         string
	PlanID       string
	Status       string // "active", "suspended", "cancelled"
	StripeID     string // Stripe customer ID for payment integration
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

	// ClearOtherDefaults clears is_default on all plans except the specified one.
	// This ensures only one plan can be marked as default at a time.
	ClearOtherDefaults(ctx context.Context, exceptID string) error
}

// EntitlementStore persists entitlement definitions.
type EntitlementStore interface {
	// List returns all entitlements.
	List(ctx context.Context) ([]entitlement.Entitlement, error)

	// ListEnabled returns only enabled entitlements.
	ListEnabled(ctx context.Context) ([]entitlement.Entitlement, error)

	// Get retrieves an entitlement by ID.
	Get(ctx context.Context, id string) (entitlement.Entitlement, error)

	// GetByName retrieves an entitlement by name.
	GetByName(ctx context.Context, name string) (entitlement.Entitlement, error)

	// Create stores a new entitlement.
	Create(ctx context.Context, e entitlement.Entitlement) error

	// Update modifies an entitlement.
	Update(ctx context.Context, e entitlement.Entitlement) error

	// Delete removes an entitlement.
	Delete(ctx context.Context, id string) error
}

// PlanEntitlementStore persists plan-entitlement mappings.
type PlanEntitlementStore interface {
	// List returns all plan-entitlement mappings.
	List(ctx context.Context) ([]entitlement.PlanEntitlement, error)

	// ListByPlan returns all entitlements for a specific plan.
	ListByPlan(ctx context.Context, planID string) ([]entitlement.PlanEntitlement, error)

	// ListByEntitlement returns all plans that have a specific entitlement.
	ListByEntitlement(ctx context.Context, entitlementID string) ([]entitlement.PlanEntitlement, error)

	// Get retrieves a plan-entitlement mapping by ID.
	Get(ctx context.Context, id string) (entitlement.PlanEntitlement, error)

	// GetByPlanAndEntitlement retrieves a specific mapping.
	GetByPlanAndEntitlement(ctx context.Context, planID, entitlementID string) (entitlement.PlanEntitlement, error)

	// Create stores a new plan-entitlement mapping.
	Create(ctx context.Context, pe entitlement.PlanEntitlement) error

	// Update modifies a plan-entitlement mapping.
	Update(ctx context.Context, pe entitlement.PlanEntitlement) error

	// Delete removes a plan-entitlement mapping.
	Delete(ctx context.Context, id string) error
}

// WebhookStore persists webhook configurations.
type WebhookStore interface {
	// List returns all webhooks.
	List(ctx context.Context) ([]webhook.Webhook, error)

	// ListByUser returns all webhooks for a specific user.
	ListByUser(ctx context.Context, userID string) ([]webhook.Webhook, error)

	// ListEnabled returns all enabled webhooks.
	ListEnabled(ctx context.Context) ([]webhook.Webhook, error)

	// ListForEvent returns all enabled webhooks that subscribe to an event type.
	ListForEvent(ctx context.Context, eventType webhook.EventType) ([]webhook.Webhook, error)

	// Get retrieves a webhook by ID.
	Get(ctx context.Context, id string) (webhook.Webhook, error)

	// Create stores a new webhook.
	Create(ctx context.Context, w webhook.Webhook) error

	// Update modifies an existing webhook.
	Update(ctx context.Context, w webhook.Webhook) error

	// Delete removes a webhook.
	Delete(ctx context.Context, id string) error
}

// DeliveryStore persists webhook delivery attempts.
type DeliveryStore interface {
	// List returns deliveries with optional filters.
	List(ctx context.Context, webhookID string, limit int) ([]webhook.Delivery, error)

	// ListPending returns deliveries ready for retry.
	ListPending(ctx context.Context, before time.Time, limit int) ([]webhook.Delivery, error)

	// Get retrieves a delivery by ID.
	Get(ctx context.Context, id string) (webhook.Delivery, error)

	// Create stores a new delivery.
	Create(ctx context.Context, d webhook.Delivery) error

	// Update modifies an existing delivery.
	Update(ctx context.Context, d webhook.Delivery) error
}

// QuotaEnforceMode determines how quota limits are enforced.
type QuotaEnforceMode string

const (
	QuotaEnforceHard QuotaEnforceMode = "hard" // Reject requests when quota exceeded
	QuotaEnforceWarn QuotaEnforceMode = "warn" // Allow but add warning headers
	QuotaEnforceSoft QuotaEnforceMode = "soft" // Allow and bill overage
)

// MeterType determines which metric to use for quota enforcement and billing.
type MeterType string

const (
	MeterTypeRequests     MeterType = "requests"      // Count raw API requests (default)
	MeterTypeComputeUnits MeterType = "compute_units" // Count weighted units (tokens, etc.)
)

// Plan represents a pricing tier.
type Plan struct {
	ID                 string
	Name               string
	Description        string
	RateLimitPerMinute int
	RequestsPerMonth   int64
	PriceMonthly       int64 // cents
	OveragePrice       int64 // hundredths of cents per request (10000 = $1)
	IsDefault          bool
	Enabled            bool
	QuotaEnforceMode   QuotaEnforceMode // "hard", "warn", "soft" - defaults to "hard"
	QuotaGracePct      float64          // Grace percentage before hard block (e.g., 0.05 = 5%)
	TrialDays          int              // Number of trial days (0 = no trial)
	MeterType          MeterType        // Which metric to enforce: "requests" or "compute_units"
	EstimatedCostPerReq float64         // Estimated cost per request for pre-check (default 1.0)
	CreatedAt          time.Time
	UpdatedAt          time.Time

	// Provider-specific price IDs for payment integration
	StripePriceID  string // Stripe price ID (e.g., price_xxx)
	PaddlePriceID  string // Paddle price ID
	LemonVariantID string // LemonSqueezy variant ID
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

// QuotaState represents current period usage for fast quota checks.
type QuotaState struct {
	UserID       string
	PeriodStart  time.Time
	RequestCount int64
	ComputeUnits float64
	BytesUsed    int64
	LastUpdated  time.Time
}

// QuotaStore persists quota state for synchronous enforcement.
type QuotaStore interface {
	// Get retrieves current quota state for a user's billing period.
	Get(ctx context.Context, userID string, periodStart time.Time) (QuotaState, error)

	// Increment atomically adds to quota counters, returns new state.
	Increment(ctx context.Context, userID string, periodStart time.Time,
		requests int64, computeUnits float64, bytes int64) (QuotaState, error)

	// Sync reconciles quota state from usage store (background job).
	Sync(ctx context.Context, userID string, periodStart time.Time, summary usage.Summary) error
}

// SubscriptionStore persists billing subscriptions.
type SubscriptionStore interface {
	// Get retrieves a subscription by ID.
	Get(ctx context.Context, id string) (billing.Subscription, error)

	// GetByUser retrieves active subscription for a user.
	GetByUser(ctx context.Context, userID string) (billing.Subscription, error)

	// GetByProviderID retrieves subscription by external provider subscription ID.
	// Used by webhook handlers to look up subscriptions from Stripe/Paddle/LemonSqueezy events.
	GetByProviderID(ctx context.Context, providerID string) (billing.Subscription, error)

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
	// trialDays specifies the number of trial days (0 = no trial).
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (sessionURL string, err error)

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

// -----------------------------------------------------------------------------
// Provider Mapping Ports
// -----------------------------------------------------------------------------

// ProviderMapping represents a mapping between local entities and external provider IDs.
type ProviderMapping struct {
	Provider   string            // e.g., "stripe", "paddle"
	EntityType string            // e.g., "user", "plan", "subscription"
	EntityID   string            // Local entity ID
	ExternalID string            // Provider's ID for this entity
	Metadata   map[string]string // Additional provider-specific data
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ProviderMappingStore persists mappings between local entities and external provider IDs.
// This replaces hardcoded fields like stripe_id in User or stripe_price_id in Plan.
type ProviderMappingStore interface {
	// Get retrieves an external ID for a local entity.
	Get(ctx context.Context, provider, entityType, entityID string) (ProviderMapping, error)

	// Set creates or updates a mapping.
	Set(ctx context.Context, m ProviderMapping) error

	// Delete removes a mapping.
	Delete(ctx context.Context, provider, entityType, entityID string) error

	// ListByEntity returns all mappings for an entity across all providers.
	ListByEntity(ctx context.Context, entityType, entityID string) ([]ProviderMapping, error)

	// ListByProvider returns all mappings for a provider.
	ListByProvider(ctx context.Context, provider, entityType string) ([]ProviderMapping, error)
}

// -----------------------------------------------------------------------------
// Cache Capability Ports
// -----------------------------------------------------------------------------

// CacheProvider provides key-value caching with TTL support.
// Implementations: memory, redis
type CacheProvider interface {
	// Name returns the provider instance name (e.g., "redis_prod").
	Name() string

	// Get retrieves a value by key. Returns nil, nil if not found.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with optional TTL (0 = no expiry).
	Set(ctx context.Context, key string, value []byte, ttlSeconds int) error

	// Delete removes a key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment atomically increments a counter, returns new value.
	Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error)

	// GetMulti retrieves multiple values by keys.
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)

	// SetMulti stores multiple key-value pairs.
	SetMulti(ctx context.Context, entries map[string][]byte, ttlSeconds int) error

	// Flush clears all keys (use with caution).
	Flush(ctx context.Context) error

	// Close releases any resources.
	Close() error
}

// -----------------------------------------------------------------------------
// Auth Capability Ports
// -----------------------------------------------------------------------------

// AuthProvider handles token generation and validation.
// Implementations: jwt
type AuthProvider interface {
	// Name returns the provider name.
	Name() string

	// GenerateToken creates an authentication token.
	GenerateToken(ctx context.Context, userID string, claims map[string]any, ttlSeconds int) (token string, expiresAt time.Time, err error)

	// ValidateToken validates and decodes a token.
	ValidateToken(ctx context.Context, token string) (userID string, claims map[string]any, err error)

	// RevokeToken invalidates a token (if supported).
	RevokeToken(ctx context.Context, token string) error
}

// -----------------------------------------------------------------------------
// Capability Registry
// -----------------------------------------------------------------------------

// CapabilityRegistry provides access to capability providers.
// Injected into services that need to use capabilities.
type CapabilityRegistry interface {
	// Payment returns the enabled payment provider, or nil if none.
	Payment(ctx context.Context) PaymentProvider

	// PaymentByName returns a specific payment provider instance.
	PaymentByName(ctx context.Context, name string) PaymentProvider

	// Email returns the enabled email provider, or nil if none.
	Email(ctx context.Context) EmailSender

	// EmailByName returns a specific email provider instance.
	EmailByName(ctx context.Context, name string) EmailSender

	// Cache returns the enabled cache provider.
	Cache(ctx context.Context) CacheProvider

	// CacheByName returns a specific cache provider instance.
	CacheByName(ctx context.Context, name string) CacheProvider

	// ProviderMapping returns the provider mapping store.
	ProviderMapping() ProviderMappingStore
}

// -----------------------------------------------------------------------------
// Admin Invite Ports
// -----------------------------------------------------------------------------

// AdminInvite represents an invitation for a new admin user.
type AdminInvite struct {
	ID        string
	Email     string
	TokenHash []byte
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// InviteStore persists admin invitations.
type InviteStore interface {
	// Create stores a new invite.
	Create(ctx context.Context, invite AdminInvite) error

	// GetByTokenHash retrieves an invite by token hash.
	GetByTokenHash(ctx context.Context, hash []byte) (AdminInvite, error)

	// List returns all invites with pagination.
	List(ctx context.Context, limit, offset int) ([]AdminInvite, error)

	// MarkUsed marks an invite as used.
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error

	// Delete removes an invite.
	Delete(ctx context.Context, id string) error

	// DeleteExpired removes all expired unused invites.
	DeleteExpired(ctx context.Context) (int64, error)

	// Count returns total invite count.
	Count(ctx context.Context) (int, error)
}

// -----------------------------------------------------------------------------
// Group Ports
// -----------------------------------------------------------------------------

// GroupStore persists user groups.
type GroupStore interface {
	// Get retrieves a group by ID.
	Get(ctx context.Context, id string) (group.Group, error)

	// GetBySlug retrieves a group by slug.
	GetBySlug(ctx context.Context, slug string) (group.Group, error)

	// Create stores a new group.
	Create(ctx context.Context, g group.Group) error

	// Update modifies an existing group.
	Update(ctx context.Context, g group.Group) error

	// Delete removes a group.
	Delete(ctx context.Context, id string) error

	// ListByUser returns all groups a user is a member of.
	ListByUser(ctx context.Context, userID string) ([]group.Group, error)

	// ListOwned returns all groups owned by a user.
	ListOwned(ctx context.Context, ownerID string) ([]group.Group, error)
}

// GroupMemberStore persists group memberships.
type GroupMemberStore interface {
	// Get retrieves a membership by ID.
	Get(ctx context.Context, id string) (group.Member, error)

	// GetByGroupAndUser retrieves a membership by group and user.
	GetByGroupAndUser(ctx context.Context, groupID, userID string) (group.Member, error)

	// Create stores a new membership.
	Create(ctx context.Context, m group.Member) error

	// Update modifies a membership (e.g., change role).
	Update(ctx context.Context, m group.Member) error

	// Delete removes a membership.
	Delete(ctx context.Context, id string) error

	// ListByGroup returns all members of a group.
	ListByGroup(ctx context.Context, groupID string) ([]group.Member, error)

	// ListByUser returns all group memberships for a user.
	ListByUser(ctx context.Context, userID string) ([]group.Member, error)
}

// GroupInviteStore persists group invitations.
type GroupInviteStore interface {
	// Get retrieves an invite by ID.
	Get(ctx context.Context, id string) (group.Invite, error)

	// GetByToken retrieves an invite by token.
	GetByToken(ctx context.Context, token string) (group.Invite, error)

	// Create stores a new invite.
	Create(ctx context.Context, inv group.Invite) error

	// Delete removes an invite.
	Delete(ctx context.Context, id string) error

	// ListByGroup returns all pending invites for a group.
	ListByGroup(ctx context.Context, groupID string) ([]group.Invite, error)

	// ListByEmail returns all pending invites for an email address.
	ListByEmail(ctx context.Context, email string) ([]group.Invite, error)

	// DeleteExpired removes all expired invites.
	DeleteExpired(ctx context.Context) (int64, error)
}

// -----------------------------------------------------------------------------
// OAuth Ports
// -----------------------------------------------------------------------------

// OAuthIdentityStore persists OAuth identities linked to users.
type OAuthIdentityStore interface {
	// Get retrieves an identity by ID.
	Get(ctx context.Context, id string) (oauth.Identity, error)

	// GetByProviderUser retrieves an identity by provider and provider user ID.
	GetByProviderUser(ctx context.Context, provider oauth.Provider, providerUserID string) (oauth.Identity, error)

	// Create stores a new identity.
	Create(ctx context.Context, identity oauth.Identity) error

	// Update modifies an identity (e.g., refresh tokens).
	Update(ctx context.Context, identity oauth.Identity) error

	// Delete removes an identity.
	Delete(ctx context.Context, id string) error

	// ListByUser returns all identities for a user.
	ListByUser(ctx context.Context, userID string) ([]oauth.Identity, error)

	// GetByUserAndProvider retrieves identity for a user from a specific provider.
	GetByUserAndProvider(ctx context.Context, userID string, provider oauth.Provider) (oauth.Identity, error)
}

// OAuthStateStore persists OAuth state tokens for CSRF protection.
// Database-backed for horizontal scaling (stateless servers).
type OAuthStateStore interface {
	// Create stores a new state.
	Create(ctx context.Context, state oauth.State) error

	// Get retrieves a state by state string.
	Get(ctx context.Context, state string) (oauth.State, error)

	// Delete removes a state (after use).
	Delete(ctx context.Context, state string) error

	// DeleteExpired removes all expired states.
	DeleteExpired(ctx context.Context) (int64, error)
}

// OAuthProvider handles OAuth authentication flow.
// Implementations: google, github, oidc
type OAuthProvider interface {
	// Name returns the provider name (e.g., "google", "github").
	Name() string

	// GetAuthURL returns the authorization URL to redirect the user.
	GetAuthURL(ctx context.Context, state, codeChallenge, nonce, redirectURI string) (string, error)

	// ExchangeCode exchanges an authorization code for tokens.
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (oauth.TokenResponse, error)

	// GetUserProfile fetches the user profile using the access token.
	GetUserProfile(ctx context.Context, accessToken string) (oauth.UserProfile, error)

	// RefreshToken refreshes an access token using a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (oauth.TokenResponse, error)
}

// -----------------------------------------------------------------------------
// TLS/Certificate Ports
// -----------------------------------------------------------------------------

// CertificateStore persists TLS certificates.
// Database-backed for horizontal scaling (stateless servers).
type CertificateStore interface {
	// Get retrieves a certificate by ID.
	Get(ctx context.Context, id string) (tls.Certificate, error)

	// GetByDomain retrieves a certificate by domain.
	GetByDomain(ctx context.Context, domain string) (tls.Certificate, error)

	// Create stores a new certificate.
	Create(ctx context.Context, cert tls.Certificate) error

	// Update modifies a certificate (e.g., renewal).
	Update(ctx context.Context, cert tls.Certificate) error

	// Delete removes a certificate.
	Delete(ctx context.Context, id string) error

	// List returns all certificates.
	List(ctx context.Context) ([]tls.Certificate, error)

	// ListExpiring returns certificates expiring within N days.
	ListExpiring(ctx context.Context, days int) ([]tls.Certificate, error)

	// ListExpired returns expired certificates.
	ListExpired(ctx context.Context) ([]tls.Certificate, error)
}

// TLSProvider handles TLS certificate provisioning.
// Implementations: acme (Let's Encrypt), manual
type TLSProvider interface {
	// Name returns the provider name (e.g., "acme", "manual").
	Name() string

	// GetCertificate retrieves or obtains a certificate for a domain.
	GetCertificate(ctx context.Context, domain string) (tls.Certificate, error)

	// ObtainCertificate obtains a new certificate for a domain.
	ObtainCertificate(ctx context.Context, domain string) (tls.Certificate, error)

	// RenewCertificate renews an existing certificate.
	RenewCertificate(ctx context.Context, domain string) (tls.Certificate, error)

	// RevokeCertificate revokes a certificate.
	RevokeCertificate(ctx context.Context, domain string, reason string) error

	// CheckRenewal checks if a certificate needs renewal.
	CheckRenewal(ctx context.Context, domain string, renewalDays int) (bool, error)
}
