// Package testing provides in-memory test doubles for all capability interfaces.
// These can be used in unit tests to avoid external dependencies.
//
// Usage:
//
//	payment := testing.NewMockPayment("test_payment")
//	resolver.RegisterImplementation("test_payment", payment)
//
//	// Run tests...
//
//	// Verify expectations
//	assert.Equal(t, 1, payment.CreateCustomerCalls())
//	assert.Contains(t, payment.Customers(), "cus_123")
package testing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// Mock Payment Provider
// =============================================================================

// MockPayment is an in-memory payment provider for testing.
type MockPayment struct {
	name string
	mu   sync.RWMutex

	// State
	customers     map[string]mockCustomer
	subscriptions map[string]capability.Subscription
	prices        map[string]mockPrice

	// Call tracking
	createCustomerCalls int
	createPriceCalls    int
	webhookEvents       []mockWebhookEvent

	// Error injection
	createCustomerErr error
	createPriceErr    error
}

type mockCustomer struct {
	ID     string
	Email  string
	Name   string
	UserID string
}

type mockPrice struct {
	ID          string
	Name        string
	AmountCents int64
	Interval    string
}

type mockWebhookEvent struct {
	EventType string
	Data      map[string]any
}

// NewMockPayment creates a new mock payment provider.
func NewMockPayment(name string) *MockPayment {
	return &MockPayment{
		name:          name,
		customers:     make(map[string]mockCustomer),
		subscriptions: make(map[string]capability.Subscription),
		prices:        make(map[string]mockPrice),
	}
}

func (m *MockPayment) Name() string { return m.name }

func (m *MockPayment) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCustomerCalls++

	if m.createCustomerErr != nil {
		return "", m.createCustomerErr
	}

	id := fmt.Sprintf("cus_mock_%d", len(m.customers)+1)
	m.customers[id] = mockCustomer{
		ID:     id,
		Email:  email,
		Name:   name,
		UserID: userID,
	}
	return id, nil
}

func (m *MockPayment) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	if trialDays > 0 {
		return fmt.Sprintf("https://checkout.mock.com/session/%s?trial=%d", customerID, trialDays), nil
	}
	return fmt.Sprintf("https://checkout.mock.com/session/%s", customerID), nil
}

func (m *MockPayment) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return fmt.Sprintf("https://portal.mock.com/session/%s", customerID), nil
}

func (m *MockPayment) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub, ok := m.subscriptions[subscriptionID]; ok {
		sub.Status = "cancelled"
		m.subscriptions[subscriptionID] = sub
	}
	return nil
}

func (m *MockPayment) GetSubscription(ctx context.Context, subscriptionID string) (capability.Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sub, ok := m.subscriptions[subscriptionID]; ok {
		return sub, nil
	}
	return capability.Subscription{}, fmt.Errorf("subscription %s not found", subscriptionID)
}

func (m *MockPayment) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp int64) error {
	return nil
}

func (m *MockPayment) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return "test.event", map[string]any{"test": true}, nil
}

func (m *MockPayment) CreatePrice(ctx context.Context, name string, amountCents int64, interval string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createPriceCalls++

	if m.createPriceErr != nil {
		return "", m.createPriceErr
	}

	id := fmt.Sprintf("price_mock_%d", len(m.prices)+1)
	m.prices[id] = mockPrice{
		ID:          id,
		Name:        name,
		AmountCents: amountCents,
		Interval:    interval,
	}
	return id, nil
}

// Test helpers

func (m *MockPayment) CreateCustomerCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.createCustomerCalls
}

func (m *MockPayment) Customers() map[string]mockCustomer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.customers
}

func (m *MockPayment) SetCreateCustomerError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCustomerErr = err
}

func (m *MockPayment) AddSubscription(sub capability.Subscription) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[sub.ID] = sub
}

// =============================================================================
// Mock Email Provider
// =============================================================================

// MockEmail is an in-memory email provider for testing.
type MockEmail struct {
	name string
	mu   sync.RWMutex

	// State
	sent []capability.EmailMessage

	// Error injection
	sendErr error
}

// NewMockEmail creates a new mock email provider.
func NewMockEmail(name string) *MockEmail {
	return &MockEmail{
		name: name,
		sent: make([]capability.EmailMessage, 0),
	}
}

func (m *MockEmail) Name() string { return m.name }

func (m *MockEmail) Send(ctx context.Context, msg capability.EmailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendErr != nil {
		return m.sendErr
	}

	m.sent = append(m.sent, msg)
	return nil
}

func (m *MockEmail) SendTemplate(ctx context.Context, to, templateID string, vars map[string]string) error {
	return m.Send(ctx, capability.EmailMessage{
		To:      to,
		Subject: fmt.Sprintf("Template: %s", templateID),
	})
}

func (m *MockEmail) TestConnection(ctx context.Context) error {
	return nil
}

// Test helpers

func (m *MockEmail) SentMessages() []capability.EmailMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sent
}

func (m *MockEmail) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

func (m *MockEmail) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = make([]capability.EmailMessage, 0)
}

// =============================================================================
// Mock Cache Provider
// =============================================================================

// MockCache is an in-memory cache provider for testing.
type MockCache struct {
	name string
	mu   sync.RWMutex

	// State
	data    map[string]cacheEntry
	closed  bool
	flushed int

	// Error injection
	getErr error
	setErr error
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMockCache creates a new mock cache provider.
func NewMockCache(name string) *MockCache {
	return &MockCache{
		name: name,
		data: make(map[string]cacheEntry),
	}
}

func (m *MockCache) Name() string { return m.name }

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getErr != nil {
		return nil, m.getErr
	}

	entry, ok := m.data[key]
	if !ok {
		return nil, nil // Not found, but not an error
	}

	// Check expiry
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	return entry.value, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setErr != nil {
		return m.setErr
	}

	entry := cacheEntry{value: value}
	if ttlSeconds > 0 {
		entry.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	m.data[key] = entry
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.data[key]
	if !ok {
		return false, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (m *MockCache) Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var current int64
	if entry, ok := m.data[key]; ok {
		fmt.Sscanf(string(entry.value), "%d", &current)
	}

	newVal := current + delta
	entry := cacheEntry{value: []byte(fmt.Sprintf("%d", newVal))}
	if ttlSeconds > 0 {
		entry.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	m.data[key] = entry

	return newVal, nil
}

func (m *MockCache) Flush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]cacheEntry)
	m.flushed++
	return nil
}

func (m *MockCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Test helpers

func (m *MockCache) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockCache) FlushCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.flushed
}

// =============================================================================
// Mock Storage Provider
// =============================================================================

// MockStorage is an in-memory storage provider for testing.
type MockStorage struct {
	name string
	mu   sync.RWMutex

	// State
	objects map[string]storageObject
}

type storageObject struct {
	data        []byte
	contentType string
	modTime     time.Time
}

// NewMockStorage creates a new mock storage provider.
func NewMockStorage(name string) *MockStorage {
	return &MockStorage{
		name:    name,
		objects: make(map[string]storageObject),
	}
}

func (m *MockStorage) Name() string { return m.name }

func (m *MockStorage) Put(ctx context.Context, key string, data []byte, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.objects[key] = storageObject{
		data:        data,
		contentType: contentType,
		modTime:     time.Now(),
	}
	return nil
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, ok := m.objects[key]
	if !ok {
		return nil, "", fmt.Errorf("object %s not found", key)
	}
	return obj.data, obj.contentType, nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.objects[key]
	return ok, nil
}

func (m *MockStorage) List(ctx context.Context, prefix string, limit int) ([]capability.StorageObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []capability.StorageObject
	for key, obj := range m.objects {
		if len(prefix) == 0 || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, capability.StorageObject{
				Key:          key,
				Size:         int64(len(obj.data)),
				ContentType:  obj.contentType,
				LastModified: obj.modTime.Unix(),
			})
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockStorage) GetURL(ctx context.Context, key string, expiresIn int) (string, error) {
	return fmt.Sprintf("https://storage.mock.com/%s?expires=%d", key, expiresIn), nil
}

func (m *MockStorage) PutStream(ctx context.Context, key string, reader capability.Reader, contentType string) error {
	// Read all data from reader
	var data []byte
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return m.Put(ctx, key, data, contentType)
}

// Test helpers

func (m *MockStorage) ObjectCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}

// =============================================================================
// Mock Queue Provider
// =============================================================================

// MockQueue is an in-memory queue provider for testing.
type MockQueue struct {
	name string
	mu   sync.RWMutex

	// State
	queues   map[string][]capability.Job
	acked    map[string]bool
	nacked   map[string]int
	jobIndex int
}

// NewMockQueue creates a new mock queue provider.
func NewMockQueue(name string) *MockQueue {
	return &MockQueue{
		name:   name,
		queues: make(map[string][]capability.Job),
		acked:  make(map[string]bool),
		nacked: make(map[string]int),
	}
}

func (m *MockQueue) Name() string { return m.name }

func (m *MockQueue) Enqueue(ctx context.Context, queue string, job capability.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job.ID == "" {
		m.jobIndex++
		job.ID = fmt.Sprintf("job_%d", m.jobIndex)
	}

	m.queues[queue] = append(m.queues[queue], job)
	return nil
}

func (m *MockQueue) EnqueueDelayed(ctx context.Context, queue string, job capability.Job, delaySeconds int) error {
	// For testing, just enqueue immediately
	return m.Enqueue(ctx, queue, job)
}

func (m *MockQueue) Dequeue(ctx context.Context, queue string, timeoutSeconds int) (*capability.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobs := m.queues[queue]
	if len(jobs) == 0 {
		return nil, nil
	}

	job := jobs[0]
	m.queues[queue] = jobs[1:]
	return &job, nil
}

func (m *MockQueue) Ack(ctx context.Context, queue string, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acked[jobID] = true
	return nil
}

func (m *MockQueue) Nack(ctx context.Context, queue string, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nacked[jobID]++
	return nil
}

func (m *MockQueue) QueueLength(ctx context.Context, queue string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.queues[queue])), nil
}

func (m *MockQueue) Close() error {
	return nil
}

// Test helpers

func (m *MockQueue) AckedJobs() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acked
}

func (m *MockQueue) NackedJobs() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nacked
}

// =============================================================================
// Mock Notification Provider
// =============================================================================

// MockNotification is an in-memory notification provider for testing.
type MockNotification struct {
	name string
	mu   sync.RWMutex

	// State
	sent []capability.NotificationMessage

	// Error injection
	sendErr error
}

// NewMockNotification creates a new mock notification provider.
func NewMockNotification(name string) *MockNotification {
	return &MockNotification{
		name: name,
		sent: make([]capability.NotificationMessage, 0),
	}
}

func (m *MockNotification) Name() string { return m.name }

func (m *MockNotification) Send(ctx context.Context, msg capability.NotificationMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendErr != nil {
		return m.sendErr
	}

	m.sent = append(m.sent, msg)
	return nil
}

func (m *MockNotification) SendBatch(ctx context.Context, msgs []capability.NotificationMessage) error {
	for _, msg := range msgs {
		if err := m.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockNotification) TestConnection(ctx context.Context) error {
	return nil
}

// Test helpers

func (m *MockNotification) SentNotifications() []capability.NotificationMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sent
}

func (m *MockNotification) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

// =============================================================================
// Mock Hasher Provider
// =============================================================================

// MockHasher is an in-memory hasher for testing (NOT secure, DO NOT use in production).
type MockHasher struct {
	name string
}

// NewMockHasher creates a new mock hasher.
func NewMockHasher(name string) *MockHasher {
	return &MockHasher{name: name}
}

func (m *MockHasher) Name() string { return m.name }

func (m *MockHasher) Hash(plaintext string) ([]byte, error) {
	// Simple reversible "hash" for testing - just prefix with "hash:"
	return []byte("hash:" + plaintext), nil
}

func (m *MockHasher) Compare(hash []byte, plaintext string) bool {
	return string(hash) == "hash:"+plaintext
}

// =============================================================================
// Test Resolver Helper
// =============================================================================

// TestResolver creates a resolver pre-configured with mock providers.
// Useful for setting up tests quickly.
//
// Usage:
//
//	tr := testing.NewTestResolver()
//	payment, _ := tr.Payment(ctx)      // Get the PaymentProvider interface via Resolver
//	payment.CreateCustomer(...)        // Use the interface
//	tr.MockPayment.CreateCustomerCalls() // Access mock helpers for verification
type TestResolver struct {
	*capability.Resolver
	Reg *capability.Registry

	MockPayment      *MockPayment
	MockEmail        *MockEmail
	MockCache        *MockCache
	MockStorage      *MockStorage
	MockQueue        *MockQueue
	MockNotification *MockNotification
	MockHasher       *MockHasher
}

// NewTestResolver creates a fully configured test resolver with all mock providers.
func NewTestResolver() *TestResolver {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	tr := &TestResolver{
		Resolver:         resolver,
		Reg:              reg,
		MockPayment:      NewMockPayment("test_payment"),
		MockEmail:        NewMockEmail("test_email"),
		MockCache:        NewMockCache("test_cache"),
		MockStorage:      NewMockStorage("test_storage"),
		MockQueue:        NewMockQueue("test_queue"),
		MockNotification: NewMockNotification("test_notification"),
		MockHasher:       NewMockHasher("test_hasher"),
	}

	// Register all mock providers
	providers := []capability.ProviderInfo{
		{Name: "test_payment", Module: "mock_payment", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "test_email", Module: "mock_email", Capability: capability.Email, Enabled: true, IsDefault: true},
		{Name: "test_cache", Module: "mock_cache", Capability: capability.Cache, Enabled: true, IsDefault: true},
		{Name: "test_storage", Module: "mock_storage", Capability: capability.Storage, Enabled: true, IsDefault: true},
		{Name: "test_queue", Module: "mock_queue", Capability: capability.Queue, Enabled: true, IsDefault: true},
		{Name: "test_notification", Module: "mock_notification", Capability: capability.Notification, Enabled: true, IsDefault: true},
		{Name: "test_hasher", Module: "mock_hasher", Capability: capability.Hasher, Enabled: true, IsDefault: true},
	}

	for _, p := range providers {
		reg.Register(p)
	}

	resolver.RegisterImplementation("test_payment", tr.MockPayment)
	resolver.RegisterImplementation("test_email", tr.MockEmail)
	resolver.RegisterImplementation("test_cache", tr.MockCache)
	resolver.RegisterImplementation("test_storage", tr.MockStorage)
	resolver.RegisterImplementation("test_queue", tr.MockQueue)
	resolver.RegisterImplementation("test_notification", tr.MockNotification)
	resolver.RegisterImplementation("test_hasher", tr.MockHasher)

	return tr
}
