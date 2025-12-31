package capability_test

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/capability"
	captest "github.com/artpar/apigate/core/capability/testing"
)

// =============================================================================
// Resolver Tests - Additional Coverage
// =============================================================================

func TestResolver_Registry(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	// Registry() should return the underlying registry
	if resolver.Registry() != reg {
		t.Error("Registry() should return the underlying registry")
	}
}

func TestResolver_RegisterAndUnregisterImplementation(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	mockPayment := &MockPaymentProvider{name: "test"}

	// Register implementation
	resolver.RegisterImplementation("test", mockPayment)

	// Get implementation
	impl, ok := resolver.GetImplementation("test")
	if !ok {
		t.Fatal("GetImplementation() should find registered implementation")
	}
	if impl != mockPayment {
		t.Error("GetImplementation() returned wrong implementation")
	}

	// Unregister implementation
	resolver.UnregisterImplementation("test")

	// Should not find after unregister
	_, ok = resolver.GetImplementation("test")
	if ok {
		t.Error("GetImplementation() should not find after unregister")
	}
}

func TestResolver_PaymentByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.PaymentByName(ctx, "nonexistent")
	if err == nil {
		t.Error("PaymentByName() should return error for nonexistent provider")
	}
}

func TestResolver_PaymentByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register an email provider
	info := capability.ProviderInfo{
		Name:       "smtp_main",
		Module:     "email_smtp",
		Capability: capability.Email,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("smtp_main", captest.NewMockEmail("smtp_main"))

	// Try to get it as a payment provider
	_, err := resolver.PaymentByName(ctx, "smtp_main")
	if err == nil {
		t.Error("PaymentByName() should return error for wrong capability type")
	}
}

func TestResolver_PaymentByName_NoImplementation(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register provider in registry but no implementation
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.PaymentByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("PaymentByName() should return error when implementation not registered")
	}
}

func TestResolver_PaymentByName_WrongImplementationType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register provider in registry
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	// Register wrong type of implementation (string instead of PaymentProvider)
	resolver.RegisterImplementation("stripe_prod", "not a payment provider")

	_, err := resolver.PaymentByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("PaymentByName() should return error when implementation doesn't implement interface")
	}
}

func TestResolver_AllPayments(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register multiple payment providers
	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "stripe_test", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: false},
		{Name: "paddle_prod", Module: "payment_paddle", Capability: capability.Payment, Enabled: false, IsDefault: false}, // disabled
	}

	for _, p := range providers {
		reg.Register(p)
	}

	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))
	resolver.RegisterImplementation("stripe_test", captest.NewMockPayment("stripe_test"))
	resolver.RegisterImplementation("paddle_prod", captest.NewMockPayment("paddle_prod"))

	payments := resolver.AllPayments(ctx)
	if len(payments) != 2 {
		t.Errorf("AllPayments() returned %d providers, want 2 (only enabled)", len(payments))
	}
}

func TestResolver_AllPayments_WithMissingImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register providers but only one has implementation
	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "stripe_test", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: false},
	}

	for _, p := range providers {
		reg.Register(p)
	}

	// Only register one implementation
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	payments := resolver.AllPayments(ctx)
	if len(payments) != 1 {
		t.Errorf("AllPayments() returned %d providers, want 1 (only those with impl)", len(payments))
	}
}

// =============================================================================
// Email Resolver Tests
// =============================================================================

func TestResolver_Email(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "smtp_main",
		Module:     "email_smtp",
		Capability: capability.Email,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("smtp_main", captest.NewMockEmail("smtp_main"))

	email, err := resolver.Email(ctx)
	if err != nil {
		t.Fatalf("Email() error = %v", err)
	}
	if email.Name() != "smtp_main" {
		t.Errorf("Email().Name() = %s, want smtp_main", email.Name())
	}
}

func TestResolver_Email_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Email(ctx)
	if err == nil {
		t.Error("Email() should return error when no provider registered")
	}
}

func TestResolver_EmailByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "smtp_main",
		Module:     "email_smtp",
		Capability: capability.Email,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("smtp_main", captest.NewMockEmail("smtp_main"))

	email, err := resolver.EmailByName(ctx, "smtp_main")
	if err != nil {
		t.Fatalf("EmailByName() error = %v", err)
	}
	if email.Name() != "smtp_main" {
		t.Errorf("EmailByName().Name() = %s, want smtp_main", email.Name())
	}
}

func TestResolver_EmailByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.EmailByName(ctx, "nonexistent")
	if err == nil {
		t.Error("EmailByName() should return error for nonexistent provider")
	}
}

func TestResolver_EmailByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.EmailByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("EmailByName() should return error for wrong capability type")
	}
}

func TestResolver_EmailByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "smtp_main",
		Module:     "email_smtp",
		Capability: capability.Email,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.EmailByName(ctx, "smtp_main")
	if err == nil {
		t.Error("EmailByName() should return error when implementation not registered")
	}
}

func TestResolver_EmailByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "smtp_main",
		Module:     "email_smtp",
		Capability: capability.Email,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("smtp_main", "not an email provider")

	_, err := resolver.EmailByName(ctx, "smtp_main")
	if err == nil {
		t.Error("EmailByName() should return error when implementation doesn't implement interface")
	}
}

// =============================================================================
// Cache Resolver Tests
// =============================================================================

func TestResolver_Cache(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_main",
		Module:     "cache_redis",
		Capability: capability.Cache,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_main", captest.NewMockCache("redis_main"))

	cache, err := resolver.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache.Name() != "redis_main" {
		t.Errorf("Cache().Name() = %s, want redis_main", cache.Name())
	}
}

func TestResolver_Cache_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Cache(ctx)
	if err == nil {
		t.Error("Cache() should return error when no provider registered")
	}
}

func TestResolver_CacheByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_main",
		Module:     "cache_redis",
		Capability: capability.Cache,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_main", captest.NewMockCache("redis_main"))

	cache, err := resolver.CacheByName(ctx, "redis_main")
	if err != nil {
		t.Fatalf("CacheByName() error = %v", err)
	}
	if cache.Name() != "redis_main" {
		t.Errorf("CacheByName().Name() = %s, want redis_main", cache.Name())
	}
}

func TestResolver_CacheByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.CacheByName(ctx, "nonexistent")
	if err == nil {
		t.Error("CacheByName() should return error for nonexistent provider")
	}
}

func TestResolver_CacheByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.CacheByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("CacheByName() should return error for wrong capability type")
	}
}

func TestResolver_CacheByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_main",
		Module:     "cache_redis",
		Capability: capability.Cache,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.CacheByName(ctx, "redis_main")
	if err == nil {
		t.Error("CacheByName() should return error when implementation not registered")
	}
}

func TestResolver_CacheByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_main",
		Module:     "cache_redis",
		Capability: capability.Cache,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_main", "not a cache provider")

	_, err := resolver.CacheByName(ctx, "redis_main")
	if err == nil {
		t.Error("CacheByName() should return error when implementation doesn't implement interface")
	}
}

// =============================================================================
// Storage Resolver Tests
// =============================================================================

func TestResolver_Storage(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "s3_main",
		Module:     "storage_s3",
		Capability: capability.Storage,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("s3_main", captest.NewMockStorage("s3_main"))

	storage, err := resolver.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}
	if storage.Name() != "s3_main" {
		t.Errorf("Storage().Name() = %s, want s3_main", storage.Name())
	}
}

func TestResolver_Storage_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Storage(ctx)
	if err == nil {
		t.Error("Storage() should return error when no provider registered")
	}
}

func TestResolver_StorageByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "s3_main",
		Module:     "storage_s3",
		Capability: capability.Storage,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("s3_main", captest.NewMockStorage("s3_main"))

	storage, err := resolver.StorageByName(ctx, "s3_main")
	if err != nil {
		t.Fatalf("StorageByName() error = %v", err)
	}
	if storage.Name() != "s3_main" {
		t.Errorf("StorageByName().Name() = %s, want s3_main", storage.Name())
	}
}

func TestResolver_StorageByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.StorageByName(ctx, "nonexistent")
	if err == nil {
		t.Error("StorageByName() should return error for nonexistent provider")
	}
}

func TestResolver_StorageByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.StorageByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("StorageByName() should return error for wrong capability type")
	}
}

func TestResolver_StorageByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "s3_main",
		Module:     "storage_s3",
		Capability: capability.Storage,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.StorageByName(ctx, "s3_main")
	if err == nil {
		t.Error("StorageByName() should return error when implementation not registered")
	}
}

func TestResolver_StorageByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "s3_main",
		Module:     "storage_s3",
		Capability: capability.Storage,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("s3_main", "not a storage provider")

	_, err := resolver.StorageByName(ctx, "s3_main")
	if err == nil {
		t.Error("StorageByName() should return error when implementation doesn't implement interface")
	}
}

// =============================================================================
// Queue Resolver Tests
// =============================================================================

func TestResolver_Queue(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_queue",
		Module:     "queue_redis",
		Capability: capability.Queue,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_queue", captest.NewMockQueue("redis_queue"))

	queue, err := resolver.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if queue.Name() != "redis_queue" {
		t.Errorf("Queue().Name() = %s, want redis_queue", queue.Name())
	}
}

func TestResolver_Queue_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Queue(ctx)
	if err == nil {
		t.Error("Queue() should return error when no provider registered")
	}
}

func TestResolver_QueueByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_queue",
		Module:     "queue_redis",
		Capability: capability.Queue,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_queue", captest.NewMockQueue("redis_queue"))

	queue, err := resolver.QueueByName(ctx, "redis_queue")
	if err != nil {
		t.Fatalf("QueueByName() error = %v", err)
	}
	if queue.Name() != "redis_queue" {
		t.Errorf("QueueByName().Name() = %s, want redis_queue", queue.Name())
	}
}

func TestResolver_QueueByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.QueueByName(ctx, "nonexistent")
	if err == nil {
		t.Error("QueueByName() should return error for nonexistent provider")
	}
}

func TestResolver_QueueByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.QueueByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("QueueByName() should return error for wrong capability type")
	}
}

func TestResolver_QueueByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_queue",
		Module:     "queue_redis",
		Capability: capability.Queue,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.QueueByName(ctx, "redis_queue")
	if err == nil {
		t.Error("QueueByName() should return error when implementation not registered")
	}
}

func TestResolver_QueueByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "redis_queue",
		Module:     "queue_redis",
		Capability: capability.Queue,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("redis_queue", "not a queue provider")

	_, err := resolver.QueueByName(ctx, "redis_queue")
	if err == nil {
		t.Error("QueueByName() should return error when implementation doesn't implement interface")
	}
}

// =============================================================================
// Notification Resolver Tests
// =============================================================================

func TestResolver_Notification(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "slack_main",
		Module:     "notification_slack",
		Capability: capability.Notification,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("slack_main", captest.NewMockNotification("slack_main"))

	notification, err := resolver.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}
	if notification.Name() != "slack_main" {
		t.Errorf("Notification().Name() = %s, want slack_main", notification.Name())
	}
}

func TestResolver_Notification_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Notification(ctx)
	if err == nil {
		t.Error("Notification() should return error when no provider registered")
	}
}

func TestResolver_NotificationByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "slack_main",
		Module:     "notification_slack",
		Capability: capability.Notification,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("slack_main", captest.NewMockNotification("slack_main"))

	notification, err := resolver.NotificationByName(ctx, "slack_main")
	if err != nil {
		t.Fatalf("NotificationByName() error = %v", err)
	}
	if notification.Name() != "slack_main" {
		t.Errorf("NotificationByName().Name() = %s, want slack_main", notification.Name())
	}
}

func TestResolver_NotificationByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.NotificationByName(ctx, "nonexistent")
	if err == nil {
		t.Error("NotificationByName() should return error for nonexistent provider")
	}
}

func TestResolver_NotificationByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.NotificationByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("NotificationByName() should return error for wrong capability type")
	}
}

func TestResolver_NotificationByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "slack_main",
		Module:     "notification_slack",
		Capability: capability.Notification,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.NotificationByName(ctx, "slack_main")
	if err == nil {
		t.Error("NotificationByName() should return error when implementation not registered")
	}
}

func TestResolver_NotificationByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "slack_main",
		Module:     "notification_slack",
		Capability: capability.Notification,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("slack_main", "not a notification provider")

	_, err := resolver.NotificationByName(ctx, "slack_main")
	if err == nil {
		t.Error("NotificationByName() should return error when implementation doesn't implement interface")
	}
}

func TestResolver_AllNotifications(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	providers := []capability.ProviderInfo{
		{Name: "slack_main", Module: "notification_slack", Capability: capability.Notification, Enabled: true, IsDefault: true},
		{Name: "discord_main", Module: "notification_discord", Capability: capability.Notification, Enabled: true, IsDefault: false},
		{Name: "webhook_main", Module: "notification_webhook", Capability: capability.Notification, Enabled: false, IsDefault: false}, // disabled
	}

	for _, p := range providers {
		reg.Register(p)
	}

	resolver.RegisterImplementation("slack_main", captest.NewMockNotification("slack_main"))
	resolver.RegisterImplementation("discord_main", captest.NewMockNotification("discord_main"))
	resolver.RegisterImplementation("webhook_main", captest.NewMockNotification("webhook_main"))

	notifications := resolver.AllNotifications(ctx)
	if len(notifications) != 2 {
		t.Errorf("AllNotifications() returned %d providers, want 2 (only enabled)", len(notifications))
	}
}

func TestResolver_AllNotifications_WithMissingImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	providers := []capability.ProviderInfo{
		{Name: "slack_main", Module: "notification_slack", Capability: capability.Notification, Enabled: true, IsDefault: true},
		{Name: "discord_main", Module: "notification_discord", Capability: capability.Notification, Enabled: true, IsDefault: false},
	}

	for _, p := range providers {
		reg.Register(p)
	}

	// Only register one implementation
	resolver.RegisterImplementation("slack_main", captest.NewMockNotification("slack_main"))

	notifications := resolver.AllNotifications(ctx)
	if len(notifications) != 1 {
		t.Errorf("AllNotifications() returned %d providers, want 1 (only those with impl)", len(notifications))
	}
}

// =============================================================================
// Hasher Resolver Tests
// =============================================================================

func TestResolver_Hasher(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "bcrypt_main",
		Module:     "hasher_bcrypt",
		Capability: capability.Hasher,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("bcrypt_main", captest.NewMockHasher("bcrypt_main"))

	hasher, err := resolver.Hasher(ctx)
	if err != nil {
		t.Fatalf("Hasher() error = %v", err)
	}
	if hasher.Name() != "bcrypt_main" {
		t.Errorf("Hasher().Name() = %s, want bcrypt_main", hasher.Name())
	}
}

func TestResolver_Hasher_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.Hasher(ctx)
	if err == nil {
		t.Error("Hasher() should return error when no provider registered")
	}
}

func TestResolver_HasherByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "bcrypt_main",
		Module:     "hasher_bcrypt",
		Capability: capability.Hasher,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("bcrypt_main", captest.NewMockHasher("bcrypt_main"))

	hasher, err := resolver.HasherByName(ctx, "bcrypt_main")
	if err != nil {
		t.Fatalf("HasherByName() error = %v", err)
	}
	if hasher.Name() != "bcrypt_main" {
		t.Errorf("HasherByName().Name() = %s, want bcrypt_main", hasher.Name())
	}
}

func TestResolver_HasherByName_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.HasherByName(ctx, "nonexistent")
	if err == nil {
		t.Error("HasherByName() should return error for nonexistent provider")
	}
}

func TestResolver_HasherByName_WrongType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", captest.NewMockPayment("stripe_prod"))

	_, err := resolver.HasherByName(ctx, "stripe_prod")
	if err == nil {
		t.Error("HasherByName() should return error for wrong capability type")
	}
}

func TestResolver_HasherByName_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "bcrypt_main",
		Module:     "hasher_bcrypt",
		Capability: capability.Hasher,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	_, err := resolver.HasherByName(ctx, "bcrypt_main")
	if err == nil {
		t.Error("HasherByName() should return error when implementation not registered")
	}
}

func TestResolver_HasherByName_WrongImplType(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:       "bcrypt_main",
		Module:     "hasher_bcrypt",
		Capability: capability.Hasher,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("bcrypt_main", "not a hasher provider")

	_, err := resolver.HasherByName(ctx, "bcrypt_main")
	if err == nil {
		t.Error("HasherByName() should return error when implementation doesn't implement interface")
	}
}

// =============================================================================
// Custom Provider Resolver Tests
// =============================================================================

func TestResolver_CustomProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          true,
		IsDefault:        true,
	}
	reg.Register(info)

	mockRecon := &mockReconciliation{name: "recon_main"}
	resolver.RegisterImplementation("recon_main", mockRecon)

	impl, err := resolver.CustomProvider(ctx, "reconciliation", "recon_main")
	if err != nil {
		t.Fatalf("CustomProvider() error = %v", err)
	}

	recon, ok := impl.(*mockReconciliation)
	if !ok {
		t.Fatal("CustomProvider() returned wrong type")
	}
	if recon.Name() != "recon_main" {
		t.Errorf("CustomProvider().Name() = %s, want recon_main", recon.Name())
	}
}

func TestResolver_CustomProvider_NotFound(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.CustomProvider(ctx, "reconciliation", "nonexistent")
	if err == nil {
		t.Error("CustomProvider() should return error for nonexistent provider")
	}
}

func TestResolver_CustomProvider_WrongCapability(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          true,
		IsDefault:        true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("recon_main", &mockReconciliation{name: "recon_main"})

	// Try to get as different capability
	_, err := resolver.CustomProvider(ctx, "analytics", "recon_main")
	if err == nil {
		t.Error("CustomProvider() should return error for wrong capability")
	}
}

func TestResolver_CustomProvider_NoImpl(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          true,
		IsDefault:        true,
	}
	reg.Register(info)

	_, err := resolver.CustomProvider(ctx, "reconciliation", "recon_main")
	if err == nil {
		t.Error("CustomProvider() should return error when implementation not registered")
	}
}

func TestResolver_CustomProviderDefault(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	providers := []capability.ProviderInfo{
		{Name: "recon_main", Module: "reconciliation_default", Capability: capability.Custom, CustomCapability: "reconciliation", Enabled: true, IsDefault: true},
		{Name: "recon_backup", Module: "reconciliation_default", Capability: capability.Custom, CustomCapability: "reconciliation", Enabled: true, IsDefault: false},
	}

	for _, p := range providers {
		reg.Register(p)
	}

	resolver.RegisterImplementation("recon_main", &mockReconciliation{name: "recon_main"})
	resolver.RegisterImplementation("recon_backup", &mockReconciliation{name: "recon_backup"})

	impl, err := resolver.CustomProviderDefault(ctx, "reconciliation")
	if err != nil {
		t.Fatalf("CustomProviderDefault() error = %v", err)
	}

	recon, ok := impl.(*mockReconciliation)
	if !ok {
		t.Fatal("CustomProviderDefault() returned wrong type")
	}
	if recon.Name() != "recon_main" {
		t.Errorf("CustomProviderDefault().Name() = %s, want recon_main (default)", recon.Name())
	}
}

func TestResolver_CustomProviderDefault_FallbackToEnabled(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	// Register without setting default
	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          true,
		IsDefault:        false,
	}
	reg.Register(info)
	resolver.RegisterImplementation("recon_main", &mockReconciliation{name: "recon_main"})

	impl, err := resolver.CustomProviderDefault(ctx, "reconciliation")
	if err != nil {
		t.Fatalf("CustomProviderDefault() error = %v", err)
	}

	recon, ok := impl.(*mockReconciliation)
	if !ok {
		t.Fatal("CustomProviderDefault() returned wrong type")
	}
	if recon.Name() != "recon_main" {
		t.Errorf("CustomProviderDefault().Name() = %s, want recon_main", recon.Name())
	}
}

func TestResolver_CustomProviderDefault_NoProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	_, err := resolver.CustomProviderDefault(ctx, "reconciliation")
	if err == nil {
		t.Error("CustomProviderDefault() should return error when no provider registered")
	}
}

func TestResolver_CustomProviderDefault_AllDisabled(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)
	ctx := context.Background()

	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          false, // disabled
		IsDefault:        true,
	}
	reg.Register(info)
	resolver.RegisterImplementation("recon_main", &mockReconciliation{name: "recon_main"})

	_, err := resolver.CustomProviderDefault(ctx, "reconciliation")
	if err == nil {
		t.Error("CustomProviderDefault() should return error when all providers disabled")
	}
}

// =============================================================================
// Mock Implementations for Testing (reused from container_test.go)
// mockReconciliation is already defined in container_test.go
// =============================================================================
