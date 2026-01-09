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
		name        string
		config      PaddleConfig
		expectedURL string
	}{
		{
			name: "production config",
			config: PaddleConfig{
				VendorID:      "vendor123",
				APIKey:        "pdl_live_api_key_123", // Live key prefix
				PublicKey:     "public_key_123",
				WebhookSecret: "webhook_secret",
				Sandbox:       false,
			},
			expectedURL: "https://api.paddle.com",
		},
		{
			name: "sandbox config",
			config: PaddleConfig{
				VendorID:      "vendor123",
				APIKey:        "pdl_sdbx_api_key_123", // Sandbox key auto-detects
				PublicKey:     "public_key_123",
				WebhookSecret: "webhook_secret",
				Sandbox:       false, // Auto-detected from key prefix
			},
			expectedURL: "https://sandbox-api.paddle.com",
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
	// Create mock server for Paddle Billing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customers" {
			t.Errorf("unexpected path: %s, want /customers", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s, want POST", r.Method)
		}

		// Paddle Billing API response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":    "ctm_123abc",
				"email": "test@example.com",
				"name":  "Test User",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewPaddleProvider(PaddleConfig{APIKey: "api_key_123"})
	provider.baseURL = server.URL
	ctx := context.Background()

	customerID, err := provider.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")

	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	if customerID != "ctm_123abc" {
		t.Errorf("customerID = %s, want ctm_123abc", customerID)
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

	// Paddle Billing API uses event_type instead of alert_name
	payload := []byte(`{"event_type":"subscription.created","data":{"id":"sub_123"}}`)
	ts := "1234567890"

	// Paddle Billing signature format: ts=timestamp;h1=HMAC-SHA256(ts + ":" + payload, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":"))
	mac.Write(payload)
	h1 := hex.EncodeToString(mac.Sum(nil))
	signature := "ts=" + ts + ";h1=" + h1

	eventType, data, err := provider.ParseWebhook(payload, signature)

	if err != nil {
		t.Fatalf("ParseWebhook failed: %v", err)
	}
	if eventType != "subscription.created" {
		t.Errorf("eventType = %s, want subscription.created", eventType)
	}
	if data == nil {
		t.Error("expected non-nil data")
	}
}

func TestPaddleProvider_ParseWebhook_InvalidSignature(t *testing.T) {
	config := PaddleConfig{
		WebhookSecret: "correct_secret",
	}
	provider := NewPaddleProvider(config)

	payload := []byte(`{"event_type":"subscription.created"}`)
	// Properly formatted but wrong signature
	signature := "ts=1234567890;h1=invalid_signature_hash"

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
	ts := "1234567890"

	// Generate valid signature for invalid JSON
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":"))
	mac.Write(payload)
	h1 := hex.EncodeToString(mac.Sum(nil))
	signature := "ts=" + ts + ";h1=" + h1

	_, _, err := provider.ParseWebhook(payload, signature)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPaddleProvider_ParseWebhook_MissingEventType(t *testing.T) {
	secret := "test_secret"
	config := PaddleConfig{
		WebhookSecret: secret,
	}
	provider := NewPaddleProvider(config)

	// Payload without event_type field
	payload := []byte(`{"data":{"id":"sub_123"}}`)
	ts := "1234567890"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":"))
	mac.Write(payload)
	h1 := hex.EncodeToString(mac.Sum(nil))
	signature := "ts=" + ts + ";h1=" + h1

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

func TestMapPaddleBillingStatus(t *testing.T) {
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
			name:     "canceled state",
			state:    "canceled",
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
			result := mapPaddleBillingStatus(tt.state)
			if result != tt.expected {
				t.Errorf("mapPaddleBillingStatus(%s) = %s, want %s", tt.state, result, tt.expected)
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
	// Create mock server for Paddle Billing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Paddle Billing API uses /transactions
		if r.URL.Path != "/transactions" {
			t.Errorf("unexpected path: %s, want /transactions", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s, want POST", r.Method)
		}

		// Paddle Billing API response format
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id": "txn_123",
				"checkout": map[string]interface{}{
					"url": "https://checkout.paddle.com/checkout/123",
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
	url, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "pri_123", "https://success.com", "https://cancel.com", 0)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.paddle.com/checkout/123" {
		t.Errorf("url = %s, want https://checkout.paddle.com/checkout/123", url)
	}
}

func TestPaddleProvider_CreateCheckoutSession_WithCustomerID(t *testing.T) {
	// Create mock server for Paddle Billing API with existing customer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle both customer fetch and transaction creation
		if r.URL.Path == "/customers/ctm_123" && r.Method == "GET" {
			// Return customer data
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"id":    "ctm_123",
					"email": "customer@example.com",
					"name":  "Test Customer",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/transactions" && r.Method == "POST" {
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			// Check customer_id is set when it's a Paddle customer ID
			if reqBody["customer_id"] != "ctm_123" {
				t.Errorf("customer_id = %v, want ctm_123", reqBody["customer_id"])
			}

			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"id": "txn_456",
					"checkout": map[string]interface{}{
						"url": "https://checkout.paddle.com/checkout/existing",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	config := PaddleConfig{
		VendorID: "vendor_123",
		APIKey:   "api_key_123",
	}
	provider := NewPaddleProvider(config)
	provider.baseURL = server.URL

	ctx := context.Background()
	// Use a Paddle customer ID (ctm_ prefix)
	url, err := provider.CreateCheckoutSession(ctx, "ctm_123", "pri_123", "https://success.com", "https://cancel.com", 0)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url != "https://checkout.paddle.com/checkout/existing" {
		t.Errorf("url = %s, want https://checkout.paddle.com/checkout/existing", url)
	}
}

func TestPaddleProvider_CreateCheckoutSession_NoURL(t *testing.T) {
	// Create mock server that returns data but no checkout URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Paddle Billing response with missing checkout data
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id": "txn_123",
				// Missing "checkout" field
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
	_, err := provider.CreateCheckoutSession(ctx, "customer@example.com", "pri_123", "https://success.com", "https://cancel.com", 0)

	if err == nil {
		t.Error("expected error when checkout URL is missing")
	}
}

func TestPaddleProvider_CreateCheckoutSession_APIError(t *testing.T) {
	// Create mock server that returns a Paddle Billing API error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := map[string]interface{}{
			"error": map[string]interface{}{
				"type":   "request_error",
				"code":   "invalid_field",
				"detail": "Invalid price ID",
			},
		}
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
	// Create mock server for Paddle Billing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Paddle Billing API uses /subscriptions/{id}/cancel
		if r.URL.Path != "/subscriptions/sub_123/cancel" {
			t.Errorf("unexpected path: %s, want /subscriptions/sub_123/cancel", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("unexpected method: %s, want POST", r.Method)
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer api_key_123" {
			t.Errorf("Authorization = %s, want Bearer api_key_123", auth)
		}

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["effective_from"] != "immediately" {
			t.Errorf("effective_from = %v, want immediately", reqBody["effective_from"])
		}

		// Paddle Billing API response format
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":     "sub_123",
				"status": "canceled",
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
	err := provider.CancelSubscription(ctx, "sub_123", true)

	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}

func TestPaddleProvider_GetSubscription_Success(t *testing.T) {
	// Create mock server for Paddle Billing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Paddle Billing API uses /subscriptions/{id}
		if r.URL.Path != "/subscriptions/sub_123" {
			t.Errorf("unexpected path: %s, want /subscriptions/sub_123", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("unexpected method: %s, want GET", r.Method)
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer api_key_123" {
			t.Errorf("Authorization = %s, want Bearer api_key_123", auth)
		}

		// Paddle Billing API response format
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":     "sub_123",
				"status": "active",
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
	// Create mock server returning 404 error for Paddle Billing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		resp := map[string]interface{}{
			"error": map[string]interface{}{
				"type":   "request_error",
				"code":   "not_found",
				"detail": "Subscription not found",
			},
		}
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
