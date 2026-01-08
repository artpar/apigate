package webhook

import (
	"testing"
	"time"
)

func TestSignPayload(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"test"}`)
	secret := "whsec_testsecret"

	signature := SignPayload(payload, secret)

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	// Verify the signature
	if !VerifySignature(payload, signature, secret) {
		t.Error("Signature verification failed")
	}

	// Wrong secret should fail
	if VerifySignature(payload, signature, "wrong_secret") {
		t.Error("Signature should not verify with wrong secret")
	}

	// Modified payload should fail
	modifiedPayload := []byte(`{"id":"evt_123","type":"modified"}`)
	if VerifySignature(modifiedPayload, signature, secret) {
		t.Error("Signature should not verify with modified payload")
	}
}

func TestSubscribesToEvent(t *testing.T) {
	webhook := Webhook{
		ID:      "wh_1",
		Events:  []EventType{EventUsageThreshold, EventKeyCreated},
		Enabled: true,
	}

	// Should match subscribed events
	if !SubscribesToEvent(webhook, EventUsageThreshold) {
		t.Error("Should subscribe to usage.threshold")
	}
	if !SubscribesToEvent(webhook, EventKeyCreated) {
		t.Error("Should subscribe to key.created")
	}

	// Should not match unsubscribed events
	if SubscribesToEvent(webhook, EventPaymentFailed) {
		t.Error("Should not subscribe to payment.failed")
	}

	// Disabled webhook should not match
	webhook.Enabled = false
	if SubscribesToEvent(webhook, EventUsageThreshold) {
		t.Error("Disabled webhook should not subscribe to any events")
	}
}

func TestFilterWebhooksForEvent(t *testing.T) {
	webhooks := []Webhook{
		{ID: "wh_1", Events: []EventType{EventUsageThreshold, EventKeyCreated}, Enabled: true},
		{ID: "wh_2", Events: []EventType{EventPaymentSuccess}, Enabled: true},
		{ID: "wh_3", Events: []EventType{EventUsageThreshold}, Enabled: false}, // Disabled
		{ID: "wh_4", Events: []EventType{EventUsageThreshold, EventPlanChanged}, Enabled: true},
	}

	// Filter for usage.threshold
	filtered := FilterWebhooksForEvent(webhooks, EventUsageThreshold)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(filtered))
	}

	// Filter for payment.success
	filtered = FilterWebhooksForEvent(webhooks, EventPaymentSuccess)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(filtered))
	}

	// Filter for event with no subscribers
	filtered = FilterWebhooksForEvent(webhooks, EventInvoiceCreated)
	if len(filtered) != 0 {
		t.Errorf("Expected 0 webhooks, got %d", len(filtered))
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		statusCode int
		shouldRetry bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{404, false},
		{408, true},  // Request Timeout
		{429, true},  // Too Many Requests
		{500, true},  // Internal Server Error
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
	}

	for _, tc := range tests {
		result := ShouldRetry(tc.statusCode)
		if result != tc.shouldRetry {
			t.Errorf("ShouldRetry(%d) = %v, expected %v", tc.statusCode, result, tc.shouldRetry)
		}
	}
}

func TestCalculateNextRetry(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// First attempt: 1 minute
	next := CalculateNextRetry(1, now)
	expected := now.Add(1 * time.Minute)
	if !next.Equal(expected) {
		t.Errorf("Attempt 1: expected %v, got %v", expected, next)
	}

	// Second attempt: 5 minutes
	next = CalculateNextRetry(2, now)
	expected = now.Add(5 * time.Minute)
	if !next.Equal(expected) {
		t.Errorf("Attempt 2: expected %v, got %v", expected, next)
	}

	// Third attempt: 30 minutes
	next = CalculateNextRetry(3, now)
	expected = now.Add(30 * time.Minute)
	if !next.Equal(expected) {
		t.Errorf("Attempt 3: expected %v, got %v", expected, next)
	}

	// Fourth+ attempt: still 30 minutes (max)
	next = CalculateNextRetry(5, now)
	expected = now.Add(30 * time.Minute)
	if !next.Equal(expected) {
		t.Errorf("Attempt 5: expected %v, got %v", expected, next)
	}
}

func TestNewDelivery(t *testing.T) {
	webhook := Webhook{
		ID:         "wh_1",
		RetryCount: 3,
	}
	event := Event{
		ID:   "evt_123",
		Type: EventUsageThreshold,
	}
	now := time.Now()

	delivery := NewDelivery(webhook, event, `{"test":true}`, now)

	if delivery.WebhookID != "wh_1" {
		t.Errorf("Expected WebhookID 'wh_1', got '%s'", delivery.WebhookID)
	}
	if delivery.EventID != "evt_123" {
		t.Errorf("Expected EventID 'evt_123', got '%s'", delivery.EventID)
	}
	if delivery.Status != DeliveryPending {
		t.Errorf("Expected status 'pending', got '%s'", delivery.Status)
	}
	if delivery.Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", delivery.Attempt)
	}
	if delivery.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", delivery.MaxAttempts)
	}
}

func TestMarkSuccess(t *testing.T) {
	delivery := Delivery{
		ID:          "del_1",
		Status:      DeliveryPending,
		Attempt:     1,
		MaxAttempts: 3,
	}
	now := time.Now()

	updated := MarkSuccess(delivery, 200, "OK", 150, now)

	if updated.Status != DeliverySuccess {
		t.Errorf("Expected status 'success', got '%s'", updated.Status)
	}
	if updated.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", updated.StatusCode)
	}
	if updated.DurationMS != 150 {
		t.Errorf("Expected duration 150ms, got %d", updated.DurationMS)
	}
}

func TestMarkFailed_WithRetry(t *testing.T) {
	delivery := Delivery{
		ID:          "del_1",
		Status:      DeliveryPending,
		Attempt:     1,
		MaxAttempts: 3,
	}
	now := time.Now()

	// 500 error should trigger retry
	updated := MarkFailed(delivery, 500, "Internal Server Error", "server error", 100, now)

	if updated.Status != DeliveryRetrying {
		t.Errorf("Expected status 'retrying', got '%s'", updated.Status)
	}
	if updated.NextRetry == nil {
		t.Error("Expected NextRetry to be set")
	}
}

func TestMarkFailed_NoRetry(t *testing.T) {
	delivery := Delivery{
		ID:          "del_1",
		Status:      DeliveryPending,
		Attempt:     1,
		MaxAttempts: 3,
	}
	now := time.Now()

	// 400 error should not trigger retry
	updated := MarkFailed(delivery, 400, "Bad Request", "bad request", 100, now)

	if updated.Status != DeliveryFailed {
		t.Errorf("Expected status 'failed', got '%s'", updated.Status)
	}
	if updated.NextRetry != nil {
		t.Error("Expected NextRetry to be nil")
	}
}

func TestMarkFailed_MaxAttemptsReached(t *testing.T) {
	delivery := Delivery{
		ID:          "del_1",
		Status:      DeliveryPending,
		Attempt:     3, // Already at max
		MaxAttempts: 3,
	}
	now := time.Now()

	// Even with 500, should not retry if max attempts reached
	updated := MarkFailed(delivery, 500, "Internal Server Error", "server error", 100, now)

	if updated.Status != DeliveryFailed {
		t.Errorf("Expected status 'failed', got '%s'", updated.Status)
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url     string
		valid   bool
	}{
		{"", false},
		{"http", false},
		{"https://example.com/webhook", true},
		{"http://localhost:8080/webhook", true},
		{"ftp://example.com", false},
	}

	for _, tc := range tests {
		valid, _ := ValidateURL(tc.url)
		if valid != tc.valid {
			t.Errorf("ValidateURL(%q) = %v, expected %v", tc.url, valid, tc.valid)
		}
	}
}

func TestValidateEvents(t *testing.T) {
	// Valid events
	valid, _ := ValidateEvents([]EventType{EventUsageThreshold, EventKeyCreated})
	if !valid {
		t.Error("Expected valid events to pass validation")
	}

	// Empty events
	valid, _ = ValidateEvents([]EventType{})
	if valid {
		t.Error("Expected empty events to fail validation")
	}

	// Invalid event type
	valid, msg := ValidateEvents([]EventType{"invalid.event"})
	if valid {
		t.Error("Expected invalid event type to fail validation")
	}
	if msg == "" {
		t.Error("Expected error message for invalid event")
	}
}

func TestBuildPayload(t *testing.T) {
	event := Event{
		ID:        "evt_123",
		Type:      EventUsageThreshold,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"user_id":   "usr_456",
			"threshold": 80,
		},
	}

	payload, err := BuildPayload(event)
	if err != nil {
		t.Fatalf("BuildPayload failed: %v", err)
	}

	if payload.ID != "evt_123" {
		t.Errorf("Expected ID 'evt_123', got '%s'", payload.ID)
	}
	if payload.Type != "usage.threshold" {
		t.Errorf("Expected type 'usage.threshold', got '%s'", payload.Type)
	}
	if payload.Data["user_id"] != "usr_456" {
		t.Errorf("Expected user_id 'usr_456', got '%v'", payload.Data["user_id"])
	}
}

func TestGenerateSecret(t *testing.T) {
	secret := GenerateSecret()

	if len(secret) < 10 {
		t.Error("Secret is too short")
	}
	if secret[:6] != "whsec_" {
		t.Errorf("Secret should start with 'whsec_', got '%s'", secret[:6])
	}

	// Generate another and verify they're different
	secret2 := GenerateSecret()
	if secret == secret2 {
		t.Error("Generated secrets should be unique")
	}
}
