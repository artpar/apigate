package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/rs/zerolog"
)

// mockWebhookStore implements ports.WebhookStore for testing.
type mockWebhookStore struct {
	mu       sync.RWMutex
	webhooks map[string]webhook.Webhook
}

func newMockWebhookStore() *mockWebhookStore {
	return &mockWebhookStore{
		webhooks: make(map[string]webhook.Webhook),
	}
}

func (m *mockWebhookStore) List(ctx context.Context) ([]webhook.Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]webhook.Webhook, 0, len(m.webhooks))
	for _, w := range m.webhooks {
		result = append(result, w)
	}
	return result, nil
}

func (m *mockWebhookStore) ListByUser(ctx context.Context, userID string) ([]webhook.Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []webhook.Webhook
	for _, w := range m.webhooks {
		if w.UserID == userID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookStore) ListEnabled(ctx context.Context) ([]webhook.Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []webhook.Webhook
	for _, w := range m.webhooks {
		if w.Enabled {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookStore) ListForEvent(ctx context.Context, eventType webhook.EventType) ([]webhook.Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []webhook.Webhook
	for _, w := range m.webhooks {
		if w.Enabled && webhook.SubscribesToEvent(w, eventType) {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookStore) Get(ctx context.Context, id string) (webhook.Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.webhooks[id], nil
}

func (m *mockWebhookStore) Create(ctx context.Context, w webhook.Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[w.ID] = w
	return nil
}

func (m *mockWebhookStore) Update(ctx context.Context, w webhook.Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[w.ID] = w
	return nil
}

func (m *mockWebhookStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.webhooks, id)
	return nil
}

// mockDeliveryStore implements ports.DeliveryStore for testing.
type mockDeliveryStore struct {
	mu         sync.RWMutex
	deliveries map[string]webhook.Delivery
}

func newMockDeliveryStore() *mockDeliveryStore {
	return &mockDeliveryStore{
		deliveries: make(map[string]webhook.Delivery),
	}
}

func (m *mockDeliveryStore) List(ctx context.Context, webhookID string, limit int) ([]webhook.Delivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []webhook.Delivery
	for _, d := range m.deliveries {
		if d.WebhookID == webhookID {
			result = append(result, d)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockDeliveryStore) ListPending(ctx context.Context, before time.Time, limit int) ([]webhook.Delivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []webhook.Delivery
	for _, d := range m.deliveries {
		if d.Status == webhook.DeliveryPending ||
			(d.Status == webhook.DeliveryRetrying && d.NextRetry != nil && d.NextRetry.Before(before)) {
			result = append(result, d)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockDeliveryStore) Get(ctx context.Context, id string) (webhook.Delivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deliveries[id], nil
}

func (m *mockDeliveryStore) Create(ctx context.Context, d webhook.Delivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries[d.ID] = d
	return nil
}

func (m *mockDeliveryStore) Update(ctx context.Context, d webhook.Delivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries[d.ID] = d
	return nil
}

func (m *mockDeliveryStore) getDeliveries() []webhook.Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]webhook.Delivery, 0, len(m.deliveries))
	for _, d := range m.deliveries {
		result = append(result, d)
	}
	return result
}

func TestWebhookService_Dispatch(t *testing.T) {
	// Create a test server that accepts webhooks
	received := make(chan bool, 1)
	var receivedPayload webhook.Payload
	var receivedSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
		if r.Header.Get("X-Event-Type") != "usage.threshold" {
			t.Errorf("Expected X-Event-Type: usage.threshold, got %s", r.Header.Get("X-Event-Type"))
		}

		receivedSignature = r.Header.Get("X-Webhook-Signature")

		// Decode payload
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
		received <- true
	}))
	defer server.Close()

	// Set up mocks
	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	// Create a webhook
	wh := webhook.Webhook{
		ID:         "wh_test",
		UserID:     "usr_123",
		Name:       "Test Webhook",
		URL:        server.URL,
		Secret:     "whsec_testsecret",
		Events:     []webhook.EventType{webhook.EventUsageThreshold},
		RetryCount: 3,
		TimeoutMS:  5000,
		Enabled:    true,
	}
	webhookStore.Create(context.Background(), wh)

	// Create service
	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	// Dispatch an event
	event := webhook.Event{
		ID:        "evt_123",
		Type:      webhook.EventUsageThreshold,
		UserID:    "usr_123",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"usage":     8000,
			"threshold": 80,
		},
	}

	err := svc.Dispatch(context.Background(), event)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	// Wait for webhook to be received
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}

	// Verify payload
	if receivedPayload.ID != "evt_123" {
		t.Errorf("Expected event ID evt_123, got %s", receivedPayload.ID)
	}
	if receivedPayload.Type != "usage.threshold" {
		t.Errorf("Expected type usage.threshold, got %s", receivedPayload.Type)
	}

	// Verify signature
	payloadBytes, _ := json.Marshal(receivedPayload)
	if !webhook.VerifySignature(payloadBytes, receivedSignature, "whsec_testsecret") {
		t.Error("Signature verification failed")
	}

	// Give time for delivery status update
	time.Sleep(100 * time.Millisecond)

	// Verify delivery was recorded
	deliveries := deliveryStore.getDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}

	d := deliveries[0]
	if d.WebhookID != "wh_test" {
		t.Errorf("Expected webhook_id wh_test, got %s", d.WebhookID)
	}
	if d.Status != webhook.DeliverySuccess {
		t.Errorf("Expected status success, got %s", d.Status)
	}
}

func TestWebhookService_DispatchEvent(t *testing.T) {
	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	// Dispatch event with no webhooks subscribed (should not error)
	err := svc.DispatchEvent(context.Background(), webhook.EventKeyCreated, "usr_123", map[string]interface{}{
		"key_id": "key_abc",
	})
	if err != nil {
		t.Errorf("DispatchEvent should not error with no webhooks: %v", err)
	}
}

func TestWebhookService_RetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	wh := webhook.Webhook{
		ID:         "wh_retry",
		UserID:     "usr_123",
		URL:        server.URL,
		Secret:     "whsec_test",
		Events:     []webhook.EventType{webhook.EventPaymentFailed},
		RetryCount: 3,
		Enabled:    true,
	}
	webhookStore.Create(context.Background(), wh)

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	event := webhook.Event{
		ID:        "evt_456",
		Type:      webhook.EventPaymentFailed,
		UserID:    "usr_123",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	}

	svc.Dispatch(context.Background(), event)

	// Wait for delivery
	time.Sleep(200 * time.Millisecond)

	// Verify delivery is marked for retry
	deliveries := deliveryStore.getDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}

	d := deliveries[0]
	if d.Status != webhook.DeliveryRetrying {
		t.Errorf("Expected status retrying, got %s", d.Status)
	}
	if d.NextRetry == nil {
		t.Error("Expected NextRetry to be set")
	}
	if d.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", d.StatusCode)
	}
}

func TestWebhookService_NoRetryOnClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	wh := webhook.Webhook{
		ID:         "wh_noretry",
		UserID:     "usr_123",
		URL:        server.URL,
		Secret:     "whsec_test",
		Events:     []webhook.EventType{webhook.EventTest},
		RetryCount: 3,
		Enabled:    true,
	}
	webhookStore.Create(context.Background(), wh)

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	event := webhook.Event{
		ID:        "evt_789",
		Type:      webhook.EventTest,
		UserID:    "usr_123",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	}

	svc.Dispatch(context.Background(), event)

	// Wait for delivery
	time.Sleep(200 * time.Millisecond)

	// Verify delivery is marked as failed (not retrying)
	deliveries := deliveryStore.getDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}

	d := deliveries[0]
	if d.Status != webhook.DeliveryFailed {
		t.Errorf("Expected status failed, got %s", d.Status)
	}
	if d.NextRetry != nil {
		t.Error("Expected NextRetry to be nil for client error")
	}
}

func TestWebhookService_StartStopRetryWorker(t *testing.T) {
	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start retry worker
	svc.StartRetryWorker(ctx, 100*time.Millisecond)

	// Wait briefly
	time.Sleep(50 * time.Millisecond)

	// Try to start again (should be no-op since already running)
	svc.StartRetryWorker(ctx, 100*time.Millisecond)

	// Stop retry worker
	svc.StopRetryWorker()

	// Try to stop again (should be no-op since already stopped)
	svc.StopRetryWorker()
}

func TestWebhookService_ProcessRetries(t *testing.T) {
	// Create a test server that succeeds
	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
		received <- true
	}))
	defer server.Close()

	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	// Create a webhook
	wh := webhook.Webhook{
		ID:         "wh_retry_test",
		UserID:     "usr_123",
		URL:        server.URL,
		Secret:     "whsec_test",
		Events:     []webhook.EventType{webhook.EventUsageThreshold},
		RetryCount: 3,
		TimeoutMS:  5000,
		Enabled:    true,
	}
	webhookStore.Create(context.Background(), wh)

	// Create a pending delivery
	now := time.Now()
	nextRetry := now.Add(-time.Second) // Already due
	d := webhook.Delivery{
		ID:         "del_retry_test",
		WebhookID:  "wh_retry_test",
		EventID:    "evt_123",
		Payload:    `{"id":"evt_123","type":"usage.threshold","timestamp":"2024-01-01T00:00:00Z","data":{}}`,
		Status:     webhook.DeliveryRetrying,
		Attempt:    1,
		NextRetry:  &nextRetry,
		CreatedAt:  now.Add(-time.Minute),
	}
	deliveryStore.Create(context.Background(), d)

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	// Process retries manually
	ctx := context.Background()
	svc.processRetries(ctx)

	// Wait for delivery to be processed
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Retry webhook not received within timeout")
	}
}

func TestWebhookService_ProcessRetries_DisabledWebhook(t *testing.T) {
	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	// Create a disabled webhook
	wh := webhook.Webhook{
		ID:      "wh_disabled",
		UserID:  "usr_123",
		URL:     "http://example.com",
		Enabled: false,
	}
	webhookStore.Create(context.Background(), wh)

	// Create a pending delivery for the disabled webhook
	now := time.Now()
	nextRetry := now.Add(-time.Second)
	d := webhook.Delivery{
		ID:        "del_disabled",
		WebhookID: "wh_disabled",
		EventID:   "evt_123",
		Status:    webhook.DeliveryRetrying,
		Attempt:   1,
		NextRetry: &nextRetry,
		CreatedAt: now.Add(-time.Minute),
	}
	deliveryStore.Create(context.Background(), d)

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	// Process retries - should skip disabled webhook
	ctx := context.Background()
	svc.processRetries(ctx)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Delivery should still be in retrying state (skipped)
	deliveries := deliveryStore.getDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}
}

func TestWebhookService_RetryWorkerContextCancel(t *testing.T) {
	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start retry worker
	svc.StartRetryWorker(ctx, 100*time.Millisecond)

	// Wait briefly then cancel context
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Give time for worker to stop
	time.Sleep(100 * time.Millisecond)
}

func TestWebhookService_TestWebhook(t *testing.T) {
	// Create a test server that accepts webhooks
	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a test event
		if r.Header.Get("X-Event-Type") != "test" {
			t.Errorf("Expected X-Event-Type: test, got %s", r.Header.Get("X-Event-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
		received <- true
	}))
	defer server.Close()

	webhookStore := newMockWebhookStore()
	deliveryStore := newMockDeliveryStore()
	logger := zerolog.Nop()

	// Create a webhook
	wh := webhook.Webhook{
		ID:         "wh_test_webhook",
		UserID:     "usr_123",
		Name:       "Test Webhook",
		URL:        server.URL,
		Secret:     "whsec_testsecret",
		Events:     []webhook.EventType{webhook.EventTest},
		RetryCount: 3,
		TimeoutMS:  5000,
		Enabled:    true,
	}
	webhookStore.Create(context.Background(), wh)

	svc := NewWebhookService(webhookStore, deliveryStore, logger)

	// Send a test webhook
	err := svc.TestWebhook(context.Background(), "wh_test_webhook")
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	// Wait for webhook to be received
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Test webhook not received within timeout")
	}

	// Verify delivery was created
	time.Sleep(100 * time.Millisecond)
	deliveries := deliveryStore.getDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}

	d := deliveries[0]
	if d.WebhookID != "wh_test_webhook" {
		t.Errorf("Expected webhook_id wh_test_webhook, got %s", d.WebhookID)
	}
}
