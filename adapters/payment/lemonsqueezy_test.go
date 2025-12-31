package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
)

func TestNewLemonSqueezyProvider(t *testing.T) {
	config := LemonSqueezyConfig{
		APIKey:        "api_key_123",
		StoreID:       "store_456",
		WebhookSecret: "webhook_secret",
	}

	provider := NewLemonSqueezyProvider(config)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.baseURL != "https://api.lemonsqueezy.com/v1" {
		t.Errorf("baseURL = %s, want https://api.lemonsqueezy.com/v1", provider.baseURL)
	}
	if provider.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
	if provider.config.APIKey != config.APIKey {
		t.Errorf("APIKey = %s, want %s", provider.config.APIKey, config.APIKey)
	}
	if provider.config.StoreID != config.StoreID {
		t.Errorf("StoreID = %s, want %s", provider.config.StoreID, config.StoreID)
	}
}

func TestLemonSqueezyProvider_Name(t *testing.T) {
	provider := &LemonSqueezyProvider{}

	name := provider.Name()

	if name != "lemonsqueezy" {
		t.Errorf("Name() = %s, want lemonsqueezy", name)
	}
}

func TestLemonSqueezyProvider_CreateCustomer_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Check headers
		if r.Header.Get("Authorization") != "Bearer api_key_123" {
			t.Error("missing or incorrect Authorization header")
		}
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Error("missing or incorrect Accept header")
		}

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		data := reqBody["data"].(map[string]interface{})
		if data["type"] != "customers" {
			t.Errorf("type = %v, want customers", data["type"])
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "customer_789",
				"type": "customers",
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	customerID, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	if customerID != "customer_789" {
		t.Errorf("customerID = %s, want customer_789", customerID)
	}
}

func TestLemonSqueezyProvider_CreateCustomer_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"not_data": "invalid",
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err == nil {
		t.Error("expected error for invalid response")
	}
}

func TestLemonSqueezyProvider_CreateCustomer_MissingID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "customers",
				// Missing "id"
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err == nil {
		t.Error("expected error for missing customer ID")
	}
}

func TestLemonSqueezyProvider_CreateCheckoutSession_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkouts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		data := reqBody["data"].(map[string]interface{})
		if data["type"] != "checkouts" {
			t.Errorf("type = %v, want checkouts", data["type"])
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "checkout_123",
				"type": "checkouts",
				"attributes": map[string]interface{}{
					"url": "https://checkout.lemonsqueezy.com/checkout/123",
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	url, err := provider.CreateCheckoutSession(ctx, "customer_123", "variant_456", "https://success.com", "https://cancel.com", 0)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.lemonsqueezy.com/checkout/123" {
		t.Errorf("url = %s, want https://checkout.lemonsqueezy.com/checkout/123", url)
	}
}

func TestLemonSqueezyProvider_CreateCheckoutSession_WithTrialDays(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		data := reqBody["data"].(map[string]interface{})
		attrs := data["attributes"].(map[string]interface{})

		// Check that checkout_options includes subscription_preview
		checkoutOptions, ok := attrs["checkout_options"].(map[string]interface{})
		if !ok {
			t.Error("expected checkout_options in request")
		}
		if checkoutOptions["subscription_preview"] != true {
			t.Error("expected subscription_preview to be true")
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "checkout_trial",
				"type": "checkouts",
				"attributes": map[string]interface{}{
					"url": "https://checkout.lemonsqueezy.com/checkout/trial",
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	url, err := provider.CreateCheckoutSession(ctx, "customer_123", "variant_456", "https://success.com", "https://cancel.com", 14)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.lemonsqueezy.com/checkout/trial" {
		t.Errorf("url = %s, want https://checkout.lemonsqueezy.com/checkout/trial", url)
	}
}

func TestLemonSqueezyProvider_CreateCheckoutSession_InvalidDataResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"not_data": "invalid",
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCheckoutSession(ctx, "customer_123", "variant_456", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error for invalid response")
	}
}

func TestLemonSqueezyProvider_CreateCheckoutSession_InvalidAttributesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "checkout_123",
				"type": "checkouts",
				// Missing attributes
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCheckoutSession(ctx, "customer_123", "variant_456", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error for missing attributes")
	}
}

func TestLemonSqueezyProvider_CreateCheckoutSession_MissingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "checkout_123",
				"type": "checkouts",
				"attributes": map[string]interface{}{
					// Missing url
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCheckoutSession(ctx, "customer_123", "variant_456", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestLemonSqueezyProvider_CreatePortalSession(t *testing.T) {
	provider := NewLemonSqueezyProvider(LemonSqueezyConfig{})
	ctx := context.Background()

	_, err := provider.CreatePortalSession(ctx, "customer_123", "https://return.com")

	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}

func TestLemonSqueezyProvider_CancelSubscription_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscriptions/sub_123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	err := provider.CancelSubscription(ctx, "sub_123", true)

	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}

func TestLemonSqueezyProvider_GetSubscription_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscriptions/sub_123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "sub_123",
				"type": "subscriptions",
				"attributes": map[string]interface{}{
					"status":  "active",
					"ends_at": "2025-12-31T23:59:59Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	sub, err := provider.GetSubscription(ctx, "sub_123")

	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}
	if sub.ID != "sub_123" {
		t.Errorf("ID = %s, want sub_123", sub.ID)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("Status = %s, want active", sub.Status)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2025-12-31T23:59:59Z")
	if !sub.CurrentPeriodEnd.Equal(expectedTime) {
		t.Errorf("CurrentPeriodEnd = %v, want %v", sub.CurrentPeriodEnd, expectedTime)
	}
}

func TestLemonSqueezyProvider_GetSubscription_NoEndsAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "sub_123",
				"type": "subscriptions",
				"attributes": map[string]interface{}{
					"status": "active",
					// No ends_at
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	sub, err := provider.GetSubscription(ctx, "sub_123")

	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}
	if sub.ID != "sub_123" {
		t.Errorf("ID = %s, want sub_123", sub.ID)
	}
}

func TestLemonSqueezyProvider_GetSubscription_InvalidDataResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"not_data": "invalid",
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.GetSubscription(ctx, "sub_123")

	if err == nil {
		t.Error("expected error for invalid response")
	}
}

func TestLemonSqueezyProvider_GetSubscription_InvalidAttributesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "sub_123",
				"type": "subscriptions",
				// Missing attributes
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.GetSubscription(ctx, "sub_123")

	if err == nil {
		t.Error("expected error for missing attributes")
	}
}

func TestLemonSqueezyProvider_ReportUsage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage-records" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		data := reqBody["data"].(map[string]interface{})
		if data["type"] != "usage-records" {
			t.Errorf("type = %v, want usage-records", data["type"])
		}

		attrs := data["attributes"].(map[string]interface{})
		if attrs["quantity"].(float64) != 100 {
			t.Errorf("quantity = %v, want 100", attrs["quantity"])
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "usage_123",
				"type": "usage-records",
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	err := provider.ReportUsage(ctx, "si_123", 100, time.Now())

	if err != nil {
		t.Fatalf("ReportUsage failed: %v", err)
	}
}

func TestLemonSqueezyProvider_ParseWebhook_ValidSignature(t *testing.T) {
	secret := "test_webhook_secret"
	config := LemonSqueezyConfig{
		WebhookSecret: secret,
	}
	provider := NewLemonSqueezyProvider(config)

	payload := []byte(`{"meta":{"event_name":"subscription_created"},"data":{"id":"123"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	eventType, data, err := provider.ParseWebhook(payload, signature)

	if err != nil {
		t.Fatalf("ParseWebhook failed: %v", err)
	}
	if eventType != "subscription_created" {
		t.Errorf("eventType = %s, want subscription_created", eventType)
	}
	if data == nil {
		t.Error("expected non-nil data")
	}
}

func TestLemonSqueezyProvider_ParseWebhook_InvalidSignature(t *testing.T) {
	config := LemonSqueezyConfig{
		WebhookSecret: "correct_secret",
	}
	provider := NewLemonSqueezyProvider(config)

	payload := []byte(`{"meta":{"event_name":"subscription_created"}}`)
	signature := "invalid_signature"

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestLemonSqueezyProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	secret := "test_secret"
	config := LemonSqueezyConfig{
		WebhookSecret: secret,
	}
	provider := NewLemonSqueezyProvider(config)

	payload := []byte(`not valid json`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLemonSqueezyProvider_ParseWebhook_MissingMeta(t *testing.T) {
	secret := "test_secret"
	config := LemonSqueezyConfig{
		WebhookSecret: secret,
	}
	provider := NewLemonSqueezyProvider(config)

	payload := []byte(`{"data":{"id":"123"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for missing meta")
	}
}

func TestMapLemonStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected billing.SubscriptionStatus
	}{
		{
			name:     "active status",
			status:   "active",
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "past_due status",
			status:   "past_due",
			expected: billing.SubscriptionStatusPastDue,
		},
		{
			name:     "cancelled status",
			status:   "cancelled",
			expected: billing.SubscriptionStatusCancelled,
		},
		{
			name:     "expired status",
			status:   "expired",
			expected: billing.SubscriptionStatusCancelled,
		},
		{
			name:     "paused status",
			status:   "paused",
			expected: billing.SubscriptionStatusPaused,
		},
		{
			name:     "on_trial status",
			status:   "on_trial",
			expected: billing.SubscriptionStatusTrialing,
		},
		{
			name:     "unpaid status",
			status:   "unpaid",
			expected: billing.SubscriptionStatusUnpaid,
		},
		{
			name:     "unknown status defaults to active",
			status:   "unknown",
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "empty status defaults to active",
			status:   "",
			expected: billing.SubscriptionStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapLemonStatus(tt.status)
			if result != tt.expected {
				t.Errorf("mapLemonStatus(%s) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

func TestLemonSqueezyConfig_Empty(t *testing.T) {
	config := LemonSqueezyConfig{}

	if config.APIKey != "" {
		t.Error("expected empty APIKey")
	}
	if config.StoreID != "" {
		t.Error("expected empty StoreID")
	}
	if config.WebhookSecret != "" {
		t.Error("expected empty WebhookSecret")
	}
}

func TestLemonSqueezyProvider_DoRequest_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"detail":"Invalid request"}]}`))
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestLemonSqueezyProvider_DoRequest_NetworkError(t *testing.T) {
	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = "http://invalid.local:99999"

	ctx := context.Background()
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err == nil {
		t.Error("expected error for network error")
	}
}

func TestLemonSqueezyProvider_DoRequest_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	config := LemonSqueezyConfig{
		APIKey:  "api_key_123",
		StoreID: "store_456",
	}
	provider := NewLemonSqueezyProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}
