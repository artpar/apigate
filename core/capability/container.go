// Package capability provides the dependency injection container for the capability system.
package capability

import (
	"context"
	"fmt"
	"sync"
)

// Container is a compile-time DI container for capability providers.
// It holds all registered providers and provides type-safe access.
//
// Usage:
//
//	container := capability.NewContainer()
//	container.RegisterPayment("stripe_prod", stripeProvider, true)
//	container.RegisterCache("redis_main", redisCache, true)
//
//	// Get providers
//	payment, _ := container.Payment(ctx)
//	cache, _ := container.Cache(ctx)
type Container struct {
	mu       sync.RWMutex
	registry *Registry
	resolver *Resolver

	// Lifecycle management
	closers []func() error
}

// NewContainer creates a new capability container.
func NewContainer() *Container {
	registry := NewRegistry()
	resolver := NewResolver(registry)

	return &Container{
		registry: registry,
		resolver: resolver,
		closers:  make([]func() error, 0),
	}
}

// Registry returns the underlying registry.
func (c *Container) Registry() *Registry {
	return c.registry
}

// Resolver returns the underlying resolver.
func (c *Container) Resolver() *Resolver {
	return c.resolver
}

// =============================================================================
// Provider Registration
// =============================================================================

// qualifiedName creates a unique registry name by combining capability type and provider name.
func qualifiedName(cap Type, name string) string {
	return fmt.Sprintf("%s:%s", cap.String(), name)
}

// RegisterPayment registers a payment provider.
func (c *Container) RegisterPayment(name string, provider PaymentProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Payment, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("payment_%s", provider.Name()),
		Capability: Payment,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// RegisterEmail registers an email provider.
func (c *Container) RegisterEmail(name string, provider EmailProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Email, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("email_%s", provider.Name()),
		Capability: Email,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// RegisterCache registers a cache provider.
func (c *Container) RegisterCache(name string, provider CacheProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Cache, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("cache_%s", provider.Name()),
		Capability: Cache,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)

	// Track for cleanup
	c.closers = append(c.closers, provider.Close)

	return nil
}

// RegisterStorage registers a storage provider.
func (c *Container) RegisterStorage(name string, provider StorageProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Storage, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("storage_%s", provider.Name()),
		Capability: Storage,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// RegisterQueue registers a queue provider.
func (c *Container) RegisterQueue(name string, provider QueueProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Queue, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("queue_%s", provider.Name()),
		Capability: Queue,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)

	// Track for cleanup
	c.closers = append(c.closers, provider.Close)

	return nil
}

// RegisterNotification registers a notification provider.
func (c *Container) RegisterNotification(name string, provider NotificationProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Notification, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("notification_%s", provider.Name()),
		Capability: Notification,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// RegisterHasher registers a hasher provider.
func (c *Container) RegisterHasher(name string, provider HasherProvider, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	qname := qualifiedName(Hasher, name)
	info := ProviderInfo{
		Name:       qname,
		Module:     fmt.Sprintf("hasher_%s", provider.Name()),
		Capability: Hasher,
		Enabled:    true,
		IsDefault:  isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// RegisterCustom registers a custom capability provider.
func (c *Container) RegisterCustom(capabilityName, providerName string, provider any, isDefault bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For custom capabilities, use capabilityName:providerName as the qualified name
	qname := fmt.Sprintf("%s:%s", capabilityName, providerName)
	info := ProviderInfo{
		Name:             qname,
		Module:           fmt.Sprintf("%s_custom", capabilityName),
		Capability:       Custom,
		CustomCapability: capabilityName,
		Enabled:          true,
		IsDefault:        isDefault,
	}

	if err := c.registry.Register(info); err != nil {
		return err
	}

	c.resolver.RegisterImplementation(qname, provider)
	return nil
}

// =============================================================================
// Provider Access (delegated to Resolver)
// =============================================================================

// Payment returns the default payment provider.
func (c *Container) Payment(ctx context.Context) (PaymentProvider, error) {
	return c.resolver.Payment(ctx)
}

// PaymentByName returns a specific payment provider.
func (c *Container) PaymentByName(ctx context.Context, name string) (PaymentProvider, error) {
	qname := qualifiedName(Payment, name)
	return c.resolver.PaymentByName(ctx, qname)
}

// Email returns the default email provider.
func (c *Container) Email(ctx context.Context) (EmailProvider, error) {
	return c.resolver.Email(ctx)
}

// EmailByName returns a specific email provider.
func (c *Container) EmailByName(ctx context.Context, name string) (EmailProvider, error) {
	qname := qualifiedName(Email, name)
	return c.resolver.EmailByName(ctx, qname)
}

// Cache returns the default cache provider.
func (c *Container) Cache(ctx context.Context) (CacheProvider, error) {
	return c.resolver.Cache(ctx)
}

// CacheByName returns a specific cache provider.
func (c *Container) CacheByName(ctx context.Context, name string) (CacheProvider, error) {
	qname := qualifiedName(Cache, name)
	return c.resolver.CacheByName(ctx, qname)
}

// Storage returns the default storage provider.
func (c *Container) Storage(ctx context.Context) (StorageProvider, error) {
	return c.resolver.Storage(ctx)
}

// StorageByName returns a specific storage provider.
func (c *Container) StorageByName(ctx context.Context, name string) (StorageProvider, error) {
	qname := qualifiedName(Storage, name)
	return c.resolver.StorageByName(ctx, qname)
}

// Queue returns the default queue provider.
func (c *Container) Queue(ctx context.Context) (QueueProvider, error) {
	return c.resolver.Queue(ctx)
}

// QueueByName returns a specific queue provider.
func (c *Container) QueueByName(ctx context.Context, name string) (QueueProvider, error) {
	qname := qualifiedName(Queue, name)
	return c.resolver.QueueByName(ctx, qname)
}

// Notification returns the default notification provider.
func (c *Container) Notification(ctx context.Context) (NotificationProvider, error) {
	return c.resolver.Notification(ctx)
}

// NotificationByName returns a specific notification provider.
func (c *Container) NotificationByName(ctx context.Context, name string) (NotificationProvider, error) {
	qname := qualifiedName(Notification, name)
	return c.resolver.NotificationByName(ctx, qname)
}

// Hasher returns the default hasher provider.
func (c *Container) Hasher(ctx context.Context) (HasherProvider, error) {
	return c.resolver.Hasher(ctx)
}

// HasherByName returns a specific hasher provider.
func (c *Container) HasherByName(ctx context.Context, name string) (HasherProvider, error) {
	qname := qualifiedName(Hasher, name)
	return c.resolver.HasherByName(ctx, qname)
}

// Custom returns a custom capability provider.
func (c *Container) Custom(ctx context.Context, capabilityName, providerName string) (any, error) {
	qname := fmt.Sprintf("%s:%s", capabilityName, providerName)
	return c.resolver.CustomProvider(ctx, capabilityName, qname)
}

// =============================================================================
// Lifecycle
// =============================================================================

// Close releases all resources held by providers.
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for _, closer := range c.closers {
		if err := closer(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}
	return nil
}

// =============================================================================
// Capability Info
// =============================================================================

// ListCapabilities returns all registered capability types.
func (c *Container) ListCapabilities() []string {
	return c.registry.ListCapabilities()
}

// ListProviders returns all registered providers.
func (c *Container) ListProviders() []ProviderInfo {
	return c.registry.All()
}

// HasCapability checks if a capability has any providers registered.
func (c *Container) HasCapability(cap Type) bool {
	return c.registry.HasCapability(cap)
}
