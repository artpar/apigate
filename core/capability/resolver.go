package capability

import (
	"context"
	"fmt"
	"sync"
)

// Resolver provides type-safe access to capability provider implementations.
// It bridges the registry (metadata) with actual provider instances.
//
// Usage:
//
//	resolver := capability.NewResolver(registry)
//	resolver.RegisterImplementation("stripe_prod", stripeAdapter)
//	payment, _ := resolver.Payment(ctx) // Returns typed PaymentProvider
type Resolver struct {
	registry        *Registry
	implementations sync.Map // name -> any (actual implementation)
}

// NewResolver creates a new resolver for the given registry.
func NewResolver(registry *Registry) *Resolver {
	return &Resolver{
		registry: registry,
	}
}

// RegisterImplementation associates an implementation with a provider name.
// The implementation should implement the appropriate capability interface.
func (r *Resolver) RegisterImplementation(name string, impl any) {
	r.implementations.Store(name, impl)
}

// UnregisterImplementation removes an implementation.
func (r *Resolver) UnregisterImplementation(name string) {
	r.implementations.Delete(name)
}

// GetImplementation returns the raw implementation for a provider name.
func (r *Resolver) GetImplementation(name string) (any, bool) {
	return r.implementations.Load(name)
}

// =============================================================================
// Payment Capability
// =============================================================================

// PaymentProvider is the interface that payment providers must implement.
type PaymentProvider interface {
	// Name returns the provider instance name.
	Name() string

	// CreateCustomer creates a customer in the payment system.
	CreateCustomer(ctx context.Context, email, name, userID string) (customerID string, err error)

	// CreateCheckoutSession creates a checkout session for subscription.
	// trialDays specifies the number of trial days (0 = no trial).
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (sessionURL string, err error)

	// CreatePortalSession creates a customer portal session.
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (portalURL string, err error)

	// CancelSubscription cancels a subscription.
	CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

	// GetSubscription retrieves subscription details.
	GetSubscription(ctx context.Context, subscriptionID string) (Subscription, error)

	// ReportUsage reports metered usage for billing.
	ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp int64) error

	// ParseWebhook parses and validates an incoming webhook.
	ParseWebhook(payload []byte, signature string) (eventType string, data map[string]any, err error)

	// CreatePrice creates a price/product for a plan.
	CreatePrice(ctx context.Context, name string, amountCents int64, interval string) (priceID string, err error)
}

// Subscription represents a billing subscription.
type Subscription struct {
	ID            string
	CustomerID    string
	PriceID       string
	Status        string // "active", "past_due", "cancelled", etc.
	CurrentPeriod Period
}

// Period represents a billing period.
type Period struct {
	Start int64
	End   int64
}

// Payment returns the default enabled payment provider.
func (r *Resolver) Payment(ctx context.Context) (PaymentProvider, error) {
	info, ok := r.registry.GetDefault(Payment)
	if !ok {
		return nil, fmt.Errorf("no enabled payment provider")
	}
	return r.getPaymentImpl(info.Name)
}

// PaymentByName returns a specific payment provider instance.
func (r *Resolver) PaymentByName(ctx context.Context, name string) (PaymentProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("payment provider %q not found", name)
	}
	if info.Capability != Payment {
		return nil, fmt.Errorf("provider %q is not a payment provider", name)
	}
	return r.getPaymentImpl(name)
}

// AllPayments returns all enabled payment providers.
func (r *Resolver) AllPayments(ctx context.Context) []PaymentProvider {
	providers := r.registry.GetEnabled(Payment)
	result := make([]PaymentProvider, 0, len(providers))
	for _, info := range providers {
		if impl, err := r.getPaymentImpl(info.Name); err == nil {
			result = append(result, impl)
		}
	}
	return result
}

func (r *Resolver) getPaymentImpl(name string) (PaymentProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("payment provider %q implementation not registered", name)
	}
	provider, ok := impl.(PaymentProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement PaymentProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Email Capability
// =============================================================================

// EmailProvider is the interface that email providers must implement.
type EmailProvider interface {
	// Name returns the provider instance name.
	Name() string

	// Send sends an email.
	Send(ctx context.Context, msg EmailMessage) error

	// SendTemplate sends an email using a template.
	SendTemplate(ctx context.Context, to, templateID string, vars map[string]string) error

	// TestConnection verifies the email configuration.
	TestConnection(ctx context.Context) error
}

// EmailMessage represents an email to be sent.
type EmailMessage struct {
	To       string
	From     string
	FromName string
	Subject  string
	HTMLBody string
	TextBody string
	ReplyTo  string
}

// Email returns the default enabled email provider.
func (r *Resolver) Email(ctx context.Context) (EmailProvider, error) {
	info, ok := r.registry.GetDefault(Email)
	if !ok {
		return nil, fmt.Errorf("no enabled email provider")
	}
	return r.getEmailImpl(info.Name)
}

// EmailByName returns a specific email provider instance.
func (r *Resolver) EmailByName(ctx context.Context, name string) (EmailProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("email provider %q not found", name)
	}
	if info.Capability != Email {
		return nil, fmt.Errorf("provider %q is not an email provider", name)
	}
	return r.getEmailImpl(name)
}

func (r *Resolver) getEmailImpl(name string) (EmailProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("email provider %q implementation not registered", name)
	}
	provider, ok := impl.(EmailProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement EmailProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Cache Capability
// =============================================================================

// CacheProvider is the interface that cache providers must implement.
type CacheProvider interface {
	// Name returns the provider instance name.
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

	// Flush clears all keys.
	Flush(ctx context.Context) error

	// Close releases resources.
	Close() error
}

// Cache returns the default enabled cache provider.
func (r *Resolver) Cache(ctx context.Context) (CacheProvider, error) {
	info, ok := r.registry.GetDefault(Cache)
	if !ok {
		return nil, fmt.Errorf("no enabled cache provider")
	}
	return r.getCacheImpl(info.Name)
}

// CacheByName returns a specific cache provider instance.
func (r *Resolver) CacheByName(ctx context.Context, name string) (CacheProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("cache provider %q not found", name)
	}
	if info.Capability != Cache {
		return nil, fmt.Errorf("provider %q is not a cache provider", name)
	}
	return r.getCacheImpl(name)
}

func (r *Resolver) getCacheImpl(name string) (CacheProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("cache provider %q implementation not registered", name)
	}
	provider, ok := impl.(CacheProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement CacheProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Storage Capability (Blob/File Storage)
// =============================================================================

// StorageProvider is the interface that blob/file storage providers must implement.
// Implementations: disk, s3, gcs, azure-blob, minio
type StorageProvider interface {
	// Name returns the provider instance name.
	Name() string

	// Put stores a file/blob.
	Put(ctx context.Context, key string, data []byte, contentType string) error

	// Get retrieves a file/blob.
	Get(ctx context.Context, key string) ([]byte, string, error) // data, contentType, error

	// Delete removes a file/blob.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// List lists keys with a prefix.
	List(ctx context.Context, prefix string, limit int) ([]StorageObject, error)

	// GetURL returns a URL to access the object (signed URL for private buckets).
	GetURL(ctx context.Context, key string, expiresIn int) (string, error)

	// PutStream stores a file from a reader (for large files).
	PutStream(ctx context.Context, key string, reader Reader, contentType string) error
}

// Reader is a minimal interface for streaming reads.
type Reader interface {
	Read(p []byte) (n int, err error)
}

// StorageObject represents a stored object's metadata.
type StorageObject struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified int64
}

// Storage returns the default enabled storage provider.
func (r *Resolver) Storage(ctx context.Context) (StorageProvider, error) {
	info, ok := r.registry.GetDefault(Storage)
	if !ok {
		return nil, fmt.Errorf("no enabled storage provider")
	}
	return r.getStorageImpl(info.Name)
}

// StorageByName returns a specific storage provider instance.
func (r *Resolver) StorageByName(ctx context.Context, name string) (StorageProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("storage provider %q not found", name)
	}
	if info.Capability != Storage {
		return nil, fmt.Errorf("provider %q is not a storage provider", name)
	}
	return r.getStorageImpl(name)
}

func (r *Resolver) getStorageImpl(name string) (StorageProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("storage provider %q implementation not registered", name)
	}
	provider, ok := impl.(StorageProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement StorageProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Queue Capability (Async Job Processing)
// =============================================================================

// QueueProvider is the interface that queue providers must implement.
// Implementations: memory, redis, sqs, rabbitmq
type QueueProvider interface {
	// Name returns the provider instance name.
	Name() string

	// Enqueue adds a job to a queue.
	Enqueue(ctx context.Context, queue string, job Job) error

	// EnqueueDelayed adds a job to be processed after a delay.
	EnqueueDelayed(ctx context.Context, queue string, job Job, delaySeconds int) error

	// Dequeue retrieves the next job from a queue (blocking up to timeout).
	Dequeue(ctx context.Context, queue string, timeoutSeconds int) (*Job, error)

	// Ack acknowledges successful job processing.
	Ack(ctx context.Context, queue string, jobID string) error

	// Nack returns a job to the queue for retry.
	Nack(ctx context.Context, queue string, jobID string) error

	// QueueLength returns the number of pending jobs.
	QueueLength(ctx context.Context, queue string) (int64, error)

	// Close releases resources.
	Close() error
}

// Job represents a queued job.
type Job struct {
	ID      string
	Type    string         // Job type for routing to handlers
	Payload map[string]any // Job data
	Retries int            // Number of retry attempts
	MaxAge  int            // Max seconds before job expires
}

// Queue returns the default enabled queue provider.
func (r *Resolver) Queue(ctx context.Context) (QueueProvider, error) {
	info, ok := r.registry.GetDefault(Queue)
	if !ok {
		return nil, fmt.Errorf("no enabled queue provider")
	}
	return r.getQueueImpl(info.Name)
}

// QueueByName returns a specific queue provider instance.
func (r *Resolver) QueueByName(ctx context.Context, name string) (QueueProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("queue provider %q not found", name)
	}
	if info.Capability != Queue {
		return nil, fmt.Errorf("provider %q is not a queue provider", name)
	}
	return r.getQueueImpl(name)
}

func (r *Resolver) getQueueImpl(name string) (QueueProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("queue provider %q implementation not registered", name)
	}
	provider, ok := impl.(QueueProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement QueueProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Notification Capability
// =============================================================================

// NotificationProvider is the interface that notification providers must implement.
// Implementations: slack, discord, webhook, telegram, email
type NotificationProvider interface {
	// Name returns the provider instance name.
	Name() string

	// Send sends a notification.
	Send(ctx context.Context, msg NotificationMessage) error

	// SendBatch sends multiple notifications.
	SendBatch(ctx context.Context, msgs []NotificationMessage) error

	// TestConnection verifies the notification configuration.
	TestConnection(ctx context.Context) error
}

// NotificationMessage represents a notification to be sent.
type NotificationMessage struct {
	Channel  string         // Channel-specific routing (e.g., Slack channel, Discord webhook)
	Title    string         // Notification title/subject
	Message  string         // Notification body
	Severity string         // "info", "warning", "error", "critical"
	Fields   map[string]any // Additional structured data
}

// Notification returns the default enabled notification provider.
func (r *Resolver) Notification(ctx context.Context) (NotificationProvider, error) {
	info, ok := r.registry.GetDefault(Notification)
	if !ok {
		return nil, fmt.Errorf("no enabled notification provider")
	}
	return r.getNotificationImpl(info.Name)
}

// NotificationByName returns a specific notification provider instance.
func (r *Resolver) NotificationByName(ctx context.Context, name string) (NotificationProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("notification provider %q not found", name)
	}
	if info.Capability != Notification {
		return nil, fmt.Errorf("provider %q is not a notification provider", name)
	}
	return r.getNotificationImpl(name)
}

// AllNotifications returns all enabled notification providers.
// Useful when you want to send to multiple channels (Slack AND email).
func (r *Resolver) AllNotifications(ctx context.Context) []NotificationProvider {
	providers := r.registry.GetEnabled(Notification)
	result := make([]NotificationProvider, 0, len(providers))
	for _, info := range providers {
		if impl, err := r.getNotificationImpl(info.Name); err == nil {
			result = append(result, impl)
		}
	}
	return result
}

func (r *Resolver) getNotificationImpl(name string) (NotificationProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("notification provider %q implementation not registered", name)
	}
	provider, ok := impl.(NotificationProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement NotificationProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Hasher Capability
// =============================================================================

// HasherProvider is the interface that password/key hashing providers must implement.
// Implementations: bcrypt, argon2, scrypt
type HasherProvider interface {
	// Name returns the provider instance name.
	Name() string

	// Hash generates a hash from plaintext.
	Hash(plaintext string) ([]byte, error)

	// Compare checks if plaintext matches hash.
	Compare(hash []byte, plaintext string) bool
}

// Hasher returns the default enabled hasher provider.
func (r *Resolver) Hasher(ctx context.Context) (HasherProvider, error) {
	info, ok := r.registry.GetDefault(Hasher)
	if !ok {
		return nil, fmt.Errorf("no enabled hasher provider")
	}
	return r.getHasherImpl(info.Name)
}

// HasherByName returns a specific hasher provider instance.
func (r *Resolver) HasherByName(ctx context.Context, name string) (HasherProvider, error) {
	info, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("hasher provider %q not found", name)
	}
	if info.Capability != Hasher {
		return nil, fmt.Errorf("provider %q is not a hasher provider", name)
	}
	return r.getHasherImpl(name)
}

func (r *Resolver) getHasherImpl(name string) (HasherProvider, error) {
	impl, ok := r.implementations.Load(name)
	if !ok {
		return nil, fmt.Errorf("hasher provider %q implementation not registered", name)
	}
	provider, ok := impl.(HasherProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement HasherProvider", name)
	}
	return provider, nil
}

// =============================================================================
// Custom Capability (for user-defined capabilities like "reconciliation")
// =============================================================================

// CustomProvider returns a custom capability provider by capability name and instance name.
// Returns the raw implementation - caller must type assert to the expected interface.
func (r *Resolver) CustomProvider(ctx context.Context, capabilityName, instanceName string) (any, error) {
	info, ok := r.registry.Get(instanceName)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", instanceName)
	}
	if info.CapabilityKey() != capabilityName {
		return nil, fmt.Errorf("provider %q does not implement capability %q", instanceName, capabilityName)
	}
	impl, ok := r.implementations.Load(instanceName)
	if !ok {
		return nil, fmt.Errorf("provider %q implementation not registered", instanceName)
	}
	return impl, nil
}

// CustomProviderDefault returns the default provider for a custom capability.
func (r *Resolver) CustomProviderDefault(ctx context.Context, capabilityName string) (any, error) {
	providers := r.registry.GetByCustomCapability(capabilityName)
	for _, p := range providers {
		if p.IsDefault && p.Enabled {
			return r.CustomProvider(ctx, capabilityName, p.Name)
		}
	}
	// Fall back to first enabled
	for _, p := range providers {
		if p.Enabled {
			return r.CustomProvider(ctx, capabilityName, p.Name)
		}
	}
	return nil, fmt.Errorf("no enabled provider for capability %q", capabilityName)
}

// Registry returns the underlying registry.
func (r *Resolver) Registry() *Registry {
	return r.registry
}
