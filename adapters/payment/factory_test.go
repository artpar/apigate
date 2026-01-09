package payment

import (
	"testing"

	"github.com/artpar/apigate/domain/settings"
)

func TestNewProvider_StripeWithAllFields(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:              "stripe",
		settings.KeyPaymentStripeSecretKey:       "sk_test_123",
		settings.KeyPaymentStripePublicKey:       "pk_test_123",
		settings.KeyPaymentStripeWebhookSecret:   "whsec_123",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "stripe" {
		t.Errorf("Name() = %s, want stripe", p.Name())
	}
}

func TestNewProvider_StripeMinimalConfig(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:        "stripe",
		settings.KeyPaymentStripeSecretKey: "sk_test_minimal",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "stripe" {
		t.Errorf("Name() = %s, want stripe", p.Name())
	}
}

func TestNewProvider_StripeMissingSecretKey(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:        "stripe",
		settings.KeyPaymentStripePublicKey: "pk_test_123",
		// Missing secret key
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing stripe secret key")
	}
}

func TestNewProvider_StripeEmptySecretKey(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:        "stripe",
		settings.KeyPaymentStripeSecretKey: "",
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for empty stripe secret key")
	}
}

func TestNewProvider_PaddleWithAllFields(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:            "paddle",
		settings.KeyPaymentPaddleVendorID:      "vendor_123",
		settings.KeyPaymentPaddleAPIKey:        "api_key_123",
		settings.KeyPaymentPaddlePublicKey:     "public_key_123",
		settings.KeyPaymentPaddleWebhookSecret: "webhook_secret",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "paddle" {
		t.Errorf("Name() = %s, want paddle", p.Name())
	}
}

func TestNewProvider_PaddleMinimalConfig(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:       "paddle",
		settings.KeyPaymentPaddleVendorID: "vendor_123",
		settings.KeyPaymentPaddleAPIKey:   "api_key_123",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "paddle" {
		t.Errorf("Name() = %s, want paddle", p.Name())
	}
}

// TestNewProvider_PaddleMissingVendorID removed - VendorID no longer required for Paddle Billing API

func TestNewProvider_PaddleMissingAPIKey(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:       "paddle",
		settings.KeyPaymentPaddleVendorID: "vendor_123",
		// Missing API key
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing paddle API key")
	}
}

func TestNewProvider_PaddleBothMissing(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "paddle",
		// Missing both vendor ID and API key
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing paddle configuration")
	}
}

func TestNewProvider_LemonSqueezyWithAllFields(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:           "lemonsqueezy",
		settings.KeyPaymentLemonAPIKey:        "api_key_123",
		settings.KeyPaymentLemonStoreID:       "store_123",
		settings.KeyPaymentLemonWebhookSecret: "webhook_secret",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "lemonsqueezy" {
		t.Errorf("Name() = %s, want lemonsqueezy", p.Name())
	}
}

func TestNewProvider_LemonSqueezyMinimalConfig(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:     "lemonsqueezy",
		settings.KeyPaymentLemonAPIKey:  "api_key_123",
		settings.KeyPaymentLemonStoreID: "store_123",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "lemonsqueezy" {
		t.Errorf("Name() = %s, want lemonsqueezy", p.Name())
	}
}

func TestNewProvider_LemonSqueezyMissingAPIKey(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:     "lemonsqueezy",
		settings.KeyPaymentLemonStoreID: "store_123",
		// Missing API key
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing lemonsqueezy API key")
	}
}

func TestNewProvider_LemonSqueezyMissingStoreID(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:    "lemonsqueezy",
		settings.KeyPaymentLemonAPIKey: "api_key_123",
		// Missing store ID
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing lemonsqueezy store ID")
	}
}

func TestNewProvider_LemonSqueezyBothMissing(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "lemonsqueezy",
		// Missing both API key and store ID
	}

	_, err := NewProvider(s)
	if err == nil {
		t.Error("expected error for missing lemonsqueezy configuration")
	}
}

func TestNewProvider_NoneExplicit(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "none",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("Name() = %s, want none", p.Name())
	}
}

func TestNewProvider_EmptyProvider(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("Name() = %s, want none", p.Name())
	}
}

func TestNewProvider_EmptySettings(t *testing.T) {
	s := settings.Settings{}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("Name() = %s, want none", p.Name())
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{"unknown provider", "unknown"},
		{"typo in stripe", "strype"},
		{"typo in paddle", "padle"},
		{"typo in lemonsqueezy", "lemonsquezy"},
		{"random string", "random_provider_123"},
		{"similar name", "stripe_v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := settings.Settings{
				settings.KeyPaymentProvider: tt.provider,
			}

			_, err := NewProvider(s)
			if err == nil {
				t.Errorf("expected error for unknown provider %q", tt.provider)
			}
		})
	}
}

func TestNewProvider_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		shouldSucceed bool
	}{
		{"uppercase STRIPE", "STRIPE", false},
		{"uppercase PADDLE", "PADDLE", false},
		{"uppercase LEMONSQUEEZY", "LEMONSQUEEZY", false},
		{"uppercase NONE", "NONE", false},
		{"mixed case Stripe", "Stripe", false},
		{"mixed case Paddle", "Paddle", false},
		{"mixed case LemonSqueezy", "LemonSqueezy", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := settings.Settings{
				settings.KeyPaymentProvider: tt.provider,
			}

			_, err := NewProvider(s)
			if tt.shouldSucceed && err != nil {
				t.Errorf("expected success for provider %q, got error: %v", tt.provider, err)
			}
			if !tt.shouldSucceed && err == nil {
				t.Errorf("expected error for provider %q", tt.provider)
			}
		})
	}
}

func TestNewProvider_NilSettings(t *testing.T) {
	// Test with nil settings (which becomes empty map)
	var s settings.Settings

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("Name() = %s, want none", p.Name())
	}
}

func TestNewProvider_ExtraFieldsIgnored(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:        "stripe",
		settings.KeyPaymentStripeSecretKey: "sk_test_123",
		// Extra fields that should be ignored
		settings.KeyPaymentPaddleVendorID: "vendor_123",
		settings.KeyPaymentLemonAPIKey:    "api_key_123",
	}

	p, err := NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "stripe" {
		t.Errorf("Name() = %s, want stripe", p.Name())
	}
}
