package payment

import (
	"testing"

	"github.com/artpar/apigate/domain/billing"
	"github.com/stripe/stripe-go/v76"
)

func TestNewStripeProvider(t *testing.T) {
	config := StripeConfig{
		SecretKey:     "sk_test_123",
		PublicKey:     "pk_test_123",
		WebhookSecret: "whsec_123",
	}

	provider := NewStripeProvider(config)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.config.SecretKey != config.SecretKey {
		t.Errorf("SecretKey = %s, want %s", provider.config.SecretKey, config.SecretKey)
	}
	if provider.config.PublicKey != config.PublicKey {
		t.Errorf("PublicKey = %s, want %s", provider.config.PublicKey, config.PublicKey)
	}
	if provider.config.WebhookSecret != config.WebhookSecret {
		t.Errorf("WebhookSecret = %s, want %s", provider.config.WebhookSecret, config.WebhookSecret)
	}
}

func TestStripeProvider_Name(t *testing.T) {
	provider := &StripeProvider{}

	name := provider.Name()

	if name != "stripe" {
		t.Errorf("Name() = %s, want stripe", name)
	}
}

func TestMapStripeStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   stripe.SubscriptionStatus
		expected billing.SubscriptionStatus
	}{
		{
			name:     "active status",
			status:   stripe.SubscriptionStatusActive,
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "past due status",
			status:   stripe.SubscriptionStatusPastDue,
			expected: billing.SubscriptionStatusPastDue,
		},
		{
			name:     "canceled status",
			status:   stripe.SubscriptionStatusCanceled,
			expected: billing.SubscriptionStatusCancelled,
		},
		{
			name:     "unpaid status",
			status:   stripe.SubscriptionStatusUnpaid,
			expected: billing.SubscriptionStatusUnpaid,
		},
		{
			name:     "trialing status",
			status:   stripe.SubscriptionStatusTrialing,
			expected: billing.SubscriptionStatusTrialing,
		},
		{
			name:     "incomplete status maps to active",
			status:   stripe.SubscriptionStatusIncomplete,
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "incomplete_expired status maps to active",
			status:   stripe.SubscriptionStatusIncompleteExpired,
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "paused status maps to active",
			status:   stripe.SubscriptionStatusPaused,
			expected: billing.SubscriptionStatusActive,
		},
		{
			name:     "unknown status maps to active",
			status:   stripe.SubscriptionStatus("unknown"),
			expected: billing.SubscriptionStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStripeStatus(tt.status)
			if result != tt.expected {
				t.Errorf("mapStripeStatus(%s) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

func TestStripeProvider_ParseWebhook_InvalidSignature(t *testing.T) {
	config := StripeConfig{
		SecretKey:     "sk_test_123",
		PublicKey:     "pk_test_123",
		WebhookSecret: "whsec_test_secret",
	}
	provider := NewStripeProvider(config)

	// Invalid signature should fail
	payload := []byte(`{"type":"test.event","data":{}}`)
	signature := "invalid_signature"

	_, _, err := provider.ParseWebhook(payload, signature)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestStripeProvider_ParseWebhook_EmptyPayload(t *testing.T) {
	config := StripeConfig{
		SecretKey:     "sk_test_123",
		PublicKey:     "pk_test_123",
		WebhookSecret: "whsec_test_secret",
	}
	provider := NewStripeProvider(config)

	// Empty payload
	_, _, err := provider.ParseWebhook([]byte{}, "signature")
	if err == nil {
		t.Error("expected error for empty payload")
	}
}

func TestStripeProvider_ParseWebhook_MalformedPayload(t *testing.T) {
	config := StripeConfig{
		SecretKey:     "sk_test_123",
		PublicKey:     "pk_test_123",
		WebhookSecret: "whsec_test_secret",
	}
	provider := NewStripeProvider(config)

	// Malformed JSON payload
	_, _, err := provider.ParseWebhook([]byte(`not json`), "signature")
	if err == nil {
		t.Error("expected error for malformed payload")
	}
}

func TestStripeConfig_Empty(t *testing.T) {
	config := StripeConfig{}

	if config.SecretKey != "" {
		t.Error("expected empty SecretKey")
	}
	if config.PublicKey != "" {
		t.Error("expected empty PublicKey")
	}
	if config.WebhookSecret != "" {
		t.Error("expected empty WebhookSecret")
	}
}

func TestStripeConfig_AllFields(t *testing.T) {
	config := StripeConfig{
		SecretKey:     "sk_test_abc123",
		PublicKey:     "pk_test_xyz789",
		WebhookSecret: "whsec_secret456",
	}

	if config.SecretKey != "sk_test_abc123" {
		t.Errorf("SecretKey = %s, want sk_test_abc123", config.SecretKey)
	}
	if config.PublicKey != "pk_test_xyz789" {
		t.Errorf("PublicKey = %s, want pk_test_xyz789", config.PublicKey)
	}
	if config.WebhookSecret != "whsec_secret456" {
		t.Errorf("WebhookSecret = %s, want whsec_secret456", config.WebhookSecret)
	}
}
