package capability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/artpar/apigate/core/capability"
	captest "github.com/artpar/apigate/core/capability/testing"
)

func TestContainer_RegisterAndResolve(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register mock providers
	mockPayment := captest.NewMockPayment("stripe_test")
	mockCache := captest.NewMockCache("redis_test")

	if err := container.RegisterPayment("stripe_test", mockPayment, true); err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}

	if err := container.RegisterCache("redis_test", mockCache, true); err != nil {
		t.Fatalf("RegisterCache() error = %v", err)
	}

	// Resolve providers
	payment, err := container.Payment(ctx)
	if err != nil {
		t.Fatalf("Payment() error = %v", err)
	}
	if payment.Name() != "stripe_test" {
		t.Errorf("Payment().Name() = %v, want stripe_test", payment.Name())
	}

	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache.Name() != "redis_test" {
		t.Errorf("Cache().Name() = %v, want redis_test", cache.Name())
	}
}

func TestContainer_MultipleProviders(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register multiple payment providers
	stripeProd := captest.NewMockPayment("stripe_prod")
	stripeTest := captest.NewMockPayment("stripe_test")
	paddle := captest.NewMockPayment("paddle")

	container.RegisterPayment("stripe_prod", stripeProd, true)  // default
	container.RegisterPayment("stripe_test", stripeTest, false) // not default
	container.RegisterPayment("paddle_prod", paddle, false)     // not default

	// Default should return stripe_prod
	payment, _ := container.Payment(ctx)
	if payment.Name() != "stripe_prod" {
		t.Errorf("Payment() should return default, got %v", payment.Name())
	}

	// Get specific provider by name
	test, _ := container.PaymentByName(ctx, "stripe_test")
	if test.Name() != "stripe_test" {
		t.Errorf("PaymentByName() = %v, want stripe_test", test.Name())
	}

	paddleProvider, _ := container.PaymentByName(ctx, "paddle_prod")
	if paddleProvider.Name() != "paddle" {
		t.Errorf("PaymentByName() = %v, want paddle", paddleProvider.Name())
	}
}

func TestContainer_CustomCapability(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register a custom capability (e.g., reconciliation)
	type ReconciliationProvider interface {
		Name() string
		Reconcile() error
	}

	mockRecon := &mockReconciliation{name: "recon_main"}
	err := container.RegisterCustom("reconciliation", "recon_main", mockRecon, true)
	if err != nil {
		t.Fatalf("RegisterCustom() error = %v", err)
	}

	// Retrieve custom provider
	impl, err := container.Custom(ctx, "reconciliation", "recon_main")
	if err != nil {
		t.Fatalf("Custom() error = %v", err)
	}

	recon, ok := impl.(*mockReconciliation)
	if !ok {
		t.Fatal("Custom() returned wrong type")
	}
	if recon.Name() != "recon_main" {
		t.Errorf("Custom provider name = %v, want recon_main", recon.Name())
	}
}

func TestContainer_ListCapabilities(t *testing.T) {
	container := capability.NewContainer()

	container.RegisterPayment("stripe", captest.NewMockPayment("stripe"), true)
	container.RegisterCache("redis", captest.NewMockCache("redis"), true)
	container.RegisterCustom("reconciliation", "recon", &mockReconciliation{name: "recon"}, true)

	caps := container.ListCapabilities()
	if len(caps) != 3 {
		t.Errorf("ListCapabilities() = %d, want 3", len(caps))
	}

	// Verify specific capabilities exist
	found := make(map[string]bool)
	for _, c := range caps {
		found[c] = true
	}

	for _, want := range []string{"payment", "cache", "reconciliation"} {
		if !found[want] {
			t.Errorf("ListCapabilities() missing %s", want)
		}
	}
}

func TestContainer_Close(t *testing.T) {
	container := capability.NewContainer()

	cache := captest.NewMockCache("test")
	container.RegisterCache("test", cache, true)

	// Close should not error
	if err := container.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Mock reconciliation provider for testing custom capabilities
type mockReconciliation struct {
	name string
}

func (m *mockReconciliation) Name() string {
	return m.name
}

func (m *mockReconciliation) Reconcile() error {
	return nil
}

// =============================================================================
// Container Additional Tests for Coverage
// =============================================================================

func TestContainer_RegistryAndResolver(t *testing.T) {
	container := capability.NewContainer()

	// Test Registry() accessor
	if container.Registry() == nil {
		t.Error("Registry() should not return nil")
	}

	// Test Resolver() accessor
	if container.Resolver() == nil {
		t.Error("Resolver() should not return nil")
	}
}

func TestContainer_RegisterEmail(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockEmail := captest.NewMockEmail("smtp_test")
	if err := container.RegisterEmail("smtp_test", mockEmail, true); err != nil {
		t.Fatalf("RegisterEmail() error = %v", err)
	}

	email, err := container.Email(ctx)
	if err != nil {
		t.Fatalf("Email() error = %v", err)
	}
	if email.Name() != "smtp_test" {
		t.Errorf("Email().Name() = %v, want smtp_test", email.Name())
	}
}

func TestContainer_EmailByName(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockEmail := captest.NewMockEmail("smtp_prod")
	container.RegisterEmail("smtp_prod", mockEmail, true)

	email, err := container.EmailByName(ctx, "smtp_prod")
	if err != nil {
		t.Fatalf("EmailByName() error = %v", err)
	}
	if email.Name() != "smtp_prod" {
		t.Errorf("EmailByName().Name() = %v, want smtp_prod", email.Name())
	}
}

func TestContainer_RegisterStorage(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockStorage := captest.NewMockStorage("s3_test")
	if err := container.RegisterStorage("s3_test", mockStorage, true); err != nil {
		t.Fatalf("RegisterStorage() error = %v", err)
	}

	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}
	if storage.Name() != "s3_test" {
		t.Errorf("Storage().Name() = %v, want s3_test", storage.Name())
	}
}

func TestContainer_StorageByName(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockStorage := captest.NewMockStorage("s3_prod")
	container.RegisterStorage("s3_prod", mockStorage, true)

	storage, err := container.StorageByName(ctx, "s3_prod")
	if err != nil {
		t.Fatalf("StorageByName() error = %v", err)
	}
	if storage.Name() != "s3_prod" {
		t.Errorf("StorageByName().Name() = %v, want s3_prod", storage.Name())
	}
}

func TestContainer_RegisterQueue(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockQueue := captest.NewMockQueue("redis_queue")
	if err := container.RegisterQueue("redis_queue", mockQueue, true); err != nil {
		t.Fatalf("RegisterQueue() error = %v", err)
	}

	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if queue.Name() != "redis_queue" {
		t.Errorf("Queue().Name() = %v, want redis_queue", queue.Name())
	}

	// Queue should be tracked for cleanup
	if err := container.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestContainer_QueueByName(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockQueue := captest.NewMockQueue("redis_queue_prod")
	container.RegisterQueue("redis_queue_prod", mockQueue, true)

	queue, err := container.QueueByName(ctx, "redis_queue_prod")
	if err != nil {
		t.Fatalf("QueueByName() error = %v", err)
	}
	if queue.Name() != "redis_queue_prod" {
		t.Errorf("QueueByName().Name() = %v, want redis_queue_prod", queue.Name())
	}
}

func TestContainer_RegisterNotification(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockNotification := captest.NewMockNotification("slack_test")
	if err := container.RegisterNotification("slack_test", mockNotification, true); err != nil {
		t.Fatalf("RegisterNotification() error = %v", err)
	}

	notification, err := container.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}
	if notification.Name() != "slack_test" {
		t.Errorf("Notification().Name() = %v, want slack_test", notification.Name())
	}
}

func TestContainer_NotificationByName(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockNotification := captest.NewMockNotification("slack_prod")
	container.RegisterNotification("slack_prod", mockNotification, true)

	notification, err := container.NotificationByName(ctx, "slack_prod")
	if err != nil {
		t.Fatalf("NotificationByName() error = %v", err)
	}
	if notification.Name() != "slack_prod" {
		t.Errorf("NotificationByName().Name() = %v, want slack_prod", notification.Name())
	}
}

func TestContainer_RegisterHasher(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockHasher := captest.NewMockHasher("bcrypt_test")
	if err := container.RegisterHasher("bcrypt_test", mockHasher, true); err != nil {
		t.Fatalf("RegisterHasher() error = %v", err)
	}

	hasher, err := container.Hasher(ctx)
	if err != nil {
		t.Fatalf("Hasher() error = %v", err)
	}
	if hasher.Name() != "bcrypt_test" {
		t.Errorf("Hasher().Name() = %v, want bcrypt_test", hasher.Name())
	}
}

func TestContainer_HasherByName(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	mockHasher := captest.NewMockHasher("bcrypt_prod")
	container.RegisterHasher("bcrypt_prod", mockHasher, true)

	hasher, err := container.HasherByName(ctx, "bcrypt_prod")
	if err != nil {
		t.Fatalf("HasherByName() error = %v", err)
	}
	if hasher.Name() != "bcrypt_prod" {
		t.Errorf("HasherByName().Name() = %v, want bcrypt_prod", hasher.Name())
	}
}

func TestContainer_RegisterPayment_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockPayment := captest.NewMockPayment("stripe_prod")
	err := container.RegisterPayment("stripe_prod", mockPayment, true)
	if err != nil {
		t.Fatalf("First RegisterPayment() error = %v", err)
	}

	// Second registration with same name should fail
	err = container.RegisterPayment("stripe_prod", mockPayment, true)
	if err == nil {
		t.Error("Second RegisterPayment() should fail with duplicate name")
	}
}

func TestContainer_RegisterEmail_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockEmail := captest.NewMockEmail("smtp_prod")
	err := container.RegisterEmail("smtp_prod", mockEmail, true)
	if err != nil {
		t.Fatalf("First RegisterEmail() error = %v", err)
	}

	err = container.RegisterEmail("smtp_prod", mockEmail, true)
	if err == nil {
		t.Error("Second RegisterEmail() should fail with duplicate name")
	}
}

func TestContainer_RegisterCache_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockCache := captest.NewMockCache("redis_main")
	err := container.RegisterCache("redis_main", mockCache, true)
	if err != nil {
		t.Fatalf("First RegisterCache() error = %v", err)
	}

	err = container.RegisterCache("redis_main", mockCache, true)
	if err == nil {
		t.Error("Second RegisterCache() should fail with duplicate name")
	}
}

func TestContainer_RegisterStorage_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockStorage := captest.NewMockStorage("s3_main")
	err := container.RegisterStorage("s3_main", mockStorage, true)
	if err != nil {
		t.Fatalf("First RegisterStorage() error = %v", err)
	}

	err = container.RegisterStorage("s3_main", mockStorage, true)
	if err == nil {
		t.Error("Second RegisterStorage() should fail with duplicate name")
	}
}

func TestContainer_RegisterQueue_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockQueue := captest.NewMockQueue("redis_queue")
	err := container.RegisterQueue("redis_queue", mockQueue, true)
	if err != nil {
		t.Fatalf("First RegisterQueue() error = %v", err)
	}

	err = container.RegisterQueue("redis_queue", mockQueue, true)
	if err == nil {
		t.Error("Second RegisterQueue() should fail with duplicate name")
	}
}

func TestContainer_RegisterNotification_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockNotification := captest.NewMockNotification("slack_main")
	err := container.RegisterNotification("slack_main", mockNotification, true)
	if err != nil {
		t.Fatalf("First RegisterNotification() error = %v", err)
	}

	err = container.RegisterNotification("slack_main", mockNotification, true)
	if err == nil {
		t.Error("Second RegisterNotification() should fail with duplicate name")
	}
}

func TestContainer_RegisterHasher_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockHasher := captest.NewMockHasher("bcrypt_main")
	err := container.RegisterHasher("bcrypt_main", mockHasher, true)
	if err != nil {
		t.Fatalf("First RegisterHasher() error = %v", err)
	}

	err = container.RegisterHasher("bcrypt_main", mockHasher, true)
	if err == nil {
		t.Error("Second RegisterHasher() should fail with duplicate name")
	}
}

func TestContainer_RegisterCustom_Duplicate(t *testing.T) {
	container := capability.NewContainer()

	mockRecon := &mockReconciliation{name: "recon_main"}
	err := container.RegisterCustom("reconciliation", "recon_main", mockRecon, true)
	if err != nil {
		t.Fatalf("First RegisterCustom() error = %v", err)
	}

	err = container.RegisterCustom("reconciliation", "recon_main", mockRecon, true)
	if err == nil {
		t.Error("Second RegisterCustom() should fail with duplicate name")
	}
}

func TestContainer_ListProviders(t *testing.T) {
	container := capability.NewContainer()

	container.RegisterPayment("stripe", captest.NewMockPayment("stripe"), true)
	container.RegisterCache("redis", captest.NewMockCache("redis"), true)
	container.RegisterEmail("smtp", captest.NewMockEmail("smtp"), true)

	providers := container.ListProviders()
	if len(providers) != 3 {
		t.Errorf("ListProviders() = %d, want 3", len(providers))
	}
}

func TestContainer_HasCapability(t *testing.T) {
	container := capability.NewContainer()

	// No capabilities registered yet
	if container.HasCapability(capability.Payment) {
		t.Error("HasCapability(Payment) should return false when no providers registered")
	}

	container.RegisterPayment("stripe", captest.NewMockPayment("stripe"), true)

	if !container.HasCapability(capability.Payment) {
		t.Error("HasCapability(Payment) should return true after registration")
	}

	if container.HasCapability(capability.Email) {
		t.Error("HasCapability(Email) should return false when not registered")
	}
}

func TestContainer_CloseWithError(t *testing.T) {
	container := capability.NewContainer()

	// Register a cache that returns error on close
	errorCache := &errorCacheProvider{name: "error_cache", closeErr: errors.New("close error")}
	container.RegisterCache("error_cache", errorCache, true)

	// Close should return error
	err := container.Close()
	if err == nil {
		t.Error("Close() should return error when closer fails")
	}
}

// errorCacheProvider is a cache provider that returns an error on Close
type errorCacheProvider struct {
	name     string
	closeErr error
}

func (e *errorCacheProvider) Name() string { return e.name }
func (e *errorCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
func (e *errorCacheProvider) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	return nil
}
func (e *errorCacheProvider) Delete(ctx context.Context, key string) error { return nil }
func (e *errorCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (e *errorCacheProvider) Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error) {
	return 0, nil
}
func (e *errorCacheProvider) Flush(ctx context.Context) error { return nil }
func (e *errorCacheProvider) Close() error                    { return e.closeErr }

// errorQueueProvider is a queue provider that returns an error on Close
type errorQueueProvider struct {
	name     string
	closeErr error
}

func (e *errorQueueProvider) Name() string { return e.name }
func (e *errorQueueProvider) Enqueue(ctx context.Context, queue string, job capability.Job) error {
	return nil
}
func (e *errorQueueProvider) EnqueueDelayed(ctx context.Context, queue string, job capability.Job, delaySeconds int) error {
	return nil
}
func (e *errorQueueProvider) Dequeue(ctx context.Context, queue string, timeoutSeconds int) (*capability.Job, error) {
	return nil, nil
}
func (e *errorQueueProvider) Ack(ctx context.Context, queue string, jobID string) error  { return nil }
func (e *errorQueueProvider) Nack(ctx context.Context, queue string, jobID string) error { return nil }
func (e *errorQueueProvider) QueueLength(ctx context.Context, queue string) (int64, error) {
	return 0, nil
}
func (e *errorQueueProvider) Close() error { return e.closeErr }

func TestContainer_CloseWithQueueError(t *testing.T) {
	container := capability.NewContainer()

	// Register a queue that returns error on close
	errorQueue := &errorQueueProvider{name: "error_queue", closeErr: errors.New("queue close error")}
	container.RegisterQueue("error_queue", errorQueue, true)

	// Close should return error
	err := container.Close()
	if err == nil {
		t.Error("Close() should return error when queue closer fails")
	}
}

func TestContainer_CloseMultipleErrors(t *testing.T) {
	container := capability.NewContainer()

	// Register multiple providers that return errors on close
	errorCache := &errorCacheProvider{name: "error_cache", closeErr: errors.New("cache close error")}
	errorQueue := &errorQueueProvider{name: "error_queue", closeErr: errors.New("queue close error")}

	container.RegisterCache("error_cache", errorCache, true)
	container.RegisterQueue("error_queue", errorQueue, true)

	// Close should return error containing all errors
	err := container.Close()
	if err == nil {
		t.Error("Close() should return error when closers fail")
	}
}
