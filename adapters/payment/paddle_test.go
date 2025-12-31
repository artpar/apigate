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

func TestNewPaddleProvider(t *testing.T) {
	tests := []struct {
		name           string
		config         PaddleConfig
		expectedURL    string
	}{
		{
			name: "production config",
			config: PaddleConfig{
				VendorID:      "vendor123",
				APIKey:        "api_key_123",
				PublicKey:     "public_key_123",
				WebhookSecret: "webhook_secret",
				Sandbox:       false,
			},
			expectedURL: "https://vendors.paddle.com/api/2.0",
		},
		{
			name: "sandbox config",
			config: PaddleConfig{
				VendorID:      "vendor123",
				APIKey:        "api_key_123",
				PublicKey:     "public_key_123",
				WebhookSecret: "webhook_secret",
				Sandbox:       true,
			},
			expectedURL: "https://sandbox-vendors.paddle.com/api/2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewPaddleProvider(tt.config)

			if provider == nil {
				t.Fatal("expected non-nil provider")
			}
			if provider.baseURL != tt.expectedURL {
				t.Errorf("baseURL = %s, want %s", provider.baseURL, tt.expectedURL)
			}
			if provider.httpClient == nil {
				t.Error("expected non-nil httpClient")
			}
			if provider.config.VendorID != tt.config.VendorID {
				t.Errorf("VendorID = %s, want %s", provider.config.VendorID, tt.config.VendorID)
			}
		})
	}
}

func TestPaddleProvider_Name(t *testing.T) {
	provider := &PaddleProvider{}

	name := provider.Name()

	if name != "paddle" {
		t.Errorf("Name() = %s, want paddle", name)
	}
}

func TestPaddleProvider_CreateCustomer(t *testing.T) {
	provider := NewPaddleProvider(PaddleConfig{})
	ctx := context.Background()

	// Paddle doesn't have a separate customer API, so it returns the userID
	customerID, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	if customerID != "user_123" {
		t.Errorf("customerID = %s, want user_123", customerID)
	}
}

func TestPaddleProvider_CreatePortalSession(t *testing.T) {
	provider := NewPaddleProvider(PaddleConfig{})
	ctx := context.Background()

	// Paddle doesn't support general portal sessions
	_, err := provider.CreatePortalSession(ctx, "customer_123", "https://return.example.com")

	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}

func TestPaddleProvider_ReportUsage(t *testing.T) {
	provider := NewPaddleProvider(PaddleConfig{})
	ctx := context.Background()

	// Paddle doesn't support usage reporting in the same way
	err := provider.ReportUsage(ctx, "si_123", 1000, time.Now())

	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}

func TestPaddleProvider_ParseWebhook_ValidSignature(t *testing.T) {
	secret := "test_webhook_secret"
	config := PaddleConfig{
		WebhookSecret: secret,
	}
	provider := NewPaddleProvider(config)

	payload := []byte(`{"alert_name":"subscription_created","subscription_id":"123"}`)

	// Generate valid signature
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
	if data["subscription_id"] != "123" {
		t.Errorf("subscription_id = %v, want 123", data["subscription_id"])
	}
}

func TestPaddleProvider_ParseWebhook_InvalidSignature(t *testing.T) {
	config := PaddleConfig{
		WebhookSecret: "correct_secret",
	}
	provider := NewPaddleProvider(config)

	payload := []byte(`{"alert_name":"subscription_created"}`)
	signature := "invalid_signature"

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestPaddleProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	secret := "test_secret"
	config := PaddleConfig{
		WebhookSecret: secret,
	}
	provider := NewPaddleProvider(config)

	payload := []byte(`not valid json`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPaddleProvider_ParseWebhook_MissingAlertName(t *testing.T) {
	secret := "test_secret"
	config := PaddleConfig{
		WebhookSecret: secret,
	}
	provider := NewPaddleProvider(config)

	payload := []byte(`{"subscription_id":"123"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	eventType, data, err := provider.ParseWebhook(payload, signature)

	if err != nil {
		t.Fatalf("ParseWebhook failed: %v", err)
	}
	if eventType != "" {
		t.Errorf("eventType = %s, want empty string", eventType)
	}
	if data == nil {
		t.Error("expected non-nil data")
	}
}

func TestMapPaddleStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected billing.SubscriptionStatus
	}{
		{
			name:     "active state",
			state:    "active",
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "past_due state",
			state:    "past_due",
			expected: billing.SubscriptionStatusPastDue,
		},
		{
			name:     "deleted state",
			state:    "deleted",
			expected: billing.SubscriptionStatusCancelled,
		},
		{
			name:     "cancelled state",
			state:    "cancelled",
			expected: billing.SubscriptionStatusCancelled,
		},
		{
			name:     "paused state",
			state:    "paused",
			expected: billing.SubscriptionStatusPaused,
		},
		{
			name:     "trialing state",
			state:    "trialing",
			expected: billing.SubscriptionStatusTrialing,
		},
		{
			name:     "unknown state defaults to active",
			state:    "unknown",
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "empty state defaults to active",
			state:    "",
			expected: billing.SubscriptionStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapPaddleStatus(tt.state)
			if result != tt.expected {
				t.Errorf("mapPaddleStatus(%s) = %s, want %s", tt.state, result, tt.expected)
			}
		})
	}
}

func TestPaddleConfig_Empty(t *testing.T) {
	config := PaddleConfig{}

	if config.VendorID != "" {
		t.Error("expected empty VendorID")
	}
	if config.APIKey != "" {
		t.Error("expected empty APIKey")
	}
	if config.Sandbox != false {
		t.Error("expected Sandbox to be false")
	}
}

func TestPaddleProvider_CreateCheckoutSession_WithMockServer(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/product/generate_pay_link" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Return success response
		resp := map[string]interface{}{
			"success": true,
			"url":     "https://checkout.paddle.com/checkout/123",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	url, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "product_123", "https://success.com", "https://cancel.com", 0)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.paddle.com/checkout/123" {
		t.Errorf("url = %s, want https://checkout.paddle.com/checkout/123", url)
	}
}

func TestPaddleProvider_CreateCheckoutSession_WithTrialDays(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Check trial_days is set
		if reqBody["trial_days"] == nil {
			t.Error("expected trial_days in request")
		}
		if reqBody["trial_days"].(float64) != 14 {
			t.Errorf("trial_days = %v, want 14", reqBody["trial_days"])
		}

		resp := map[string]interface{}{
			"success": true,
			"url":     "https://checkout.paddle.com/checkout/trial",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	url, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "product_123", "https://success.com", "https://cancel.com", 14)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.paddle.com/checkout/trial" {
		t.Errorf("url = %s, want https://checkout.paddle.com/checkout/trial", url)
	}
}

func TestPaddleProvider_CreateCheckoutSession_NoURL(t *testing.T) {
	// Create mock server that returns success but no URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			// Missing "url" field
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "product_123", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error when URL is missing")
	}
}

func TestPaddleProvider_CreateCheckoutSession_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": false,
			"error":   "Invalid product ID",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "product_123", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestPaddleProvider_CancelSubscription_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscription/users_cancel" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["subscription_id"] != "sub_123" {
			t.Errorf("subscription_id = %v, want sub_123", reqBody["subscription_id"])
		}

		resp := map[string]interface{}{
			"success": true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	err := provider.CancelSubscription(ctx, "sub_123", true)

	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}

func TestPaddleProvider_GetSubscription_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscription/users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"success": true,
			"response": []interface{}{
				map[string]interface{}{
					"subscription_id": "sub_123",
					"state":           "active",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
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
}

func TestPaddleProvider_GetSubscription_NotFound(t *testing.T) {
	// Create mock server returning empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success":  true,
			"response": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.GetSubscription(ctx, "sub_not_found")

	if err == nil {
		t.Error("expected error for subscription not found")
	}
}

func TestPaddleProvider_DoRequest_InvalidJSON(t *testing.T) {
	// Create mock server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	err := provider.CancelSubscription(ctx, "sub_123", true)

	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestPaddleProvider_DoRequest_NetworkError(t *testing.T) {
	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = "http://invalid.local:99999"

	ctx := context.Background()
	err := provider.CancelSubscription(ctx, "sub_123", true)

	if err == nil {
		t.Error("expected error for network error")
	}
}

func TestPaddleProvider_GetSubscription_InvalidResponse(t *testing.T) {
	// Create mock server returning response with wrong type
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success":  true,
			"response": "not an array",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	_, err := provider.GetSubscription(ctx, "sub_123")

	if err == nil {
		t.Error("expected error for invalid response format")
	}
}
