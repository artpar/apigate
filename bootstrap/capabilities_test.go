package bootstrap

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/domain/settings"
	"github.com/rs/zerolog"
)

func TestNewCapabilityContainer(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	// Verify capabilities are registered
	caps := container.ListCapabilities()
	if len(caps) == 0 {
		t.Error("expected at least one capability registered")
	}

	// Test cache provider is available
	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache.Name() != "default" {
		t.Errorf("Cache().Name() = %v, want default", cache.Name())
	}

	// Test storage provider is available
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}
	if storage.Name() != "default" {
		t.Errorf("Storage().Name() = %v, want default", storage.Name())
	}

	// Test queue provider is available
	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if queue.Name() != "default" {
		t.Errorf("Queue().Name() = %v, want default", queue.Name())
	}

	// Test notification provider is available
	notification, err := container.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}
	if notification.Name() != "default" {
		t.Errorf("Notification().Name() = %v, want default", notification.Name())
	}
}

func TestCapabilityContainer_CacheOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}

	// Test Set and Get
	key := "test_key"
	value := []byte("test_value")

	if err := cache.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", string(got), string(value))
	}
}

func TestCapabilityContainer_StorageOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}

	// Test Put and Get
	key := "test/file.txt"
	data := []byte("file content")
	contentType := "text/plain"

	if err := storage.Put(ctx, key, data, contentType); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	gotData, gotContentType, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(gotData) != string(data) {
		t.Errorf("Get() data = %v, want %v", string(gotData), string(data))
	}
	if gotContentType != contentType {
		t.Errorf("Get() contentType = %v, want %v", gotContentType, contentType)
	}
}

func TestCapabilityContainer_QueueOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	// Verify queue operations work
	length, err := queue.QueueLength(ctx, "test_queue")
	if err != nil {
		t.Fatalf("QueueLength() error = %v", err)
	}
	if length != 0 {
		t.Errorf("QueueLength() = %d, want 0", length)
	}
}

// mockCacheProvider implements ports.CacheProvider for testing.
type mockCacheProvider struct {
	name string
}

func (m *mockCacheProvider) Name() string { return m.name }
func (m *mockCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
func (m *mockCacheProvider) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	return nil
}
func (m *mockCacheProvider) Delete(ctx context.Context, key string) error { return nil }
func (m *mockCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockCacheProvider) Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error) {
	return 0, nil
}
func (m *mockCacheProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	return nil, nil
}
func (m *mockCacheProvider) SetMulti(ctx context.Context, entries map[string][]byte, ttlSeconds int) error {
	return nil
}
func (m *mockCacheProvider) Flush(ctx context.Context) error { return nil }
func (m *mockCacheProvider) Close() error                    { return nil }

// mockPaymentProvider implements ports.PaymentProvider for testing.
type mockPaymentProvider struct {
	name string
}

func (m *mockPaymentProvider) Name() string { return m.name }
func (m *mockPaymentProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	return "cust_123", nil
}
func (m *mockPaymentProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	return "https://checkout.example.com", nil
}
func (m *mockPaymentProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "https://portal.example.com", nil
}
func (m *mockPaymentProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return nil
}
func (m *mockPaymentProvider) GetSubscription(ctx context.Context, subscriptionID string) (interface{}, error) {
	return nil, nil
}
func (m *mockPaymentProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp interface{}) error {
	return nil
}
func (m *mockPaymentProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return "", nil, nil
}

// mockEmailSender implements ports.EmailSender for testing.
type mockEmailSender struct{}

func (m *mockEmailSender) Send(ctx context.Context, msg interface{}) error { return nil }
func (m *mockEmailSender) SendVerification(ctx context.Context, to, name, token string) error {
	return nil
}
func (m *mockEmailSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	return nil
}
func (m *mockEmailSender) SendWelcome(ctx context.Context, to, name string) error { return nil }

// mockHasher implements ports.Hasher for testing.
type mockHasher struct{}

func (m *mockHasher) Hash(plaintext string) ([]byte, error) { return []byte("hashed"), nil }
func (m *mockHasher) Compare(hash []byte, plaintext string) bool {
	return string(hash) == "hashed"
}

func TestCapabilityContainer_WithPreConfiguredCache(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}
	mockCache := &mockCacheProvider{name: "mock_cache"}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
		Cache:    mockCache,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}

	// When a cache is provided, it should be wrapped and used
	if cache == nil {
		t.Error("cache provider should not be nil")
	}
}

func TestCapabilityContainer_WithPreConfiguredHasher(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}
	mockHasher := &mockHasher{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
		Hasher:   mockHasher,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()

	// Verify hasher is registered
	hasher, err := container.Hasher(ctx)
	if err != nil {
		t.Fatalf("Hasher() error = %v", err)
	}
	if hasher == nil {
		t.Error("hasher provider should not be nil")
	}
}

func TestCapabilityContainer_NotificationOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	notification, err := container.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}

	// Test notification operations
	err = notification.Send(ctx, capability.NotificationMessage{
		Channel:  "test@example.com",
		Title:    "Test Subject",
		Message:  "Test body",
		Severity: "info",
		Fields:   map[string]any{},
	})
	if err != nil {
		t.Errorf("Notification.Send() error = %v", err)
	}
}

func TestCapabilityContainer_CacheDelete(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}

	// Set a value
	key := "delete_test_key"
	value := []byte("test_value")
	if err := cache.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete it
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %v", got)
	}
}

func TestCapabilityContainer_CacheExists(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}

	key := "exists_test_key"

	// Check before setting
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("key should not exist before setting")
	}

	// Set the value
	if err := cache.Set(ctx, key, []byte("value"), 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Check after setting
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("key should exist after setting")
	}
}

func TestCapabilityContainer_StorageDelete(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}

	// Put a file
	key := "test/delete_file.txt"
	data := []byte("delete me")
	if err := storage.Put(ctx, key, data, "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Delete it
	if err := storage.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, _, err = storage.Get(ctx, key)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestCapabilityContainer_StorageExists(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}

	key := "test/exists_file.txt"

	// Check before putting
	exists, err := storage.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("file should not exist before putting")
	}

	// Put the file
	if err := storage.Put(ctx, key, []byte("content"), "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Check after putting
	exists, err = storage.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("file should exist after putting")
	}
}

func TestCapabilityContainer_QueueEnqueueDequeue(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	queueName := "test_enqueue_dequeue"
	job := capability.Job{
		ID:      "test-job-1",
		Type:    "test",
		Payload: map[string]any{"test": "message"},
	}

	// Enqueue
	if err := queue.Enqueue(ctx, queueName, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Check length
	length, err := queue.QueueLength(ctx, queueName)
	if err != nil {
		t.Fatalf("QueueLength() error = %v", err)
	}
	if length != 1 {
		t.Errorf("QueueLength() = %d, want 1", length)
	}

	// Dequeue
	got, err := queue.Dequeue(ctx, queueName, 1)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if got == nil || got.ID != job.ID {
		t.Errorf("Dequeue() job ID = %v, want %v", got, job.ID)
	}

	// Check length after dequeue
	length, err = queue.QueueLength(ctx, queueName)
	if err != nil {
		t.Fatalf("QueueLength() error = %v", err)
	}
	if length != 0 {
		t.Errorf("QueueLength() after dequeue = %d, want 0", length)
	}
}
