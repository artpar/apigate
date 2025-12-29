package payment_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/payment"
	"github.com/artpar/apigate/domain/settings"
)

func TestNoopProvider_Name(t *testing.T) {
	p := payment.NewNoopProvider()
	if p.Name() != "none" {
		t.Errorf("Name() = %s, want none", p.Name())
	}
}

func TestNoopProvider_CreateCustomer(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	_, err := p.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_CreateCheckoutSession(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	_, err := p.CreateCheckoutSession(ctx, "cus_123", "price_abc", "https://success.com", "https://cancel.com")
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_CreatePortalSession(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	_, err := p.CreatePortalSession(ctx, "cus_123", "https://return.com")
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_CancelSubscription(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	err := p.CancelSubscription(ctx, "sub_123", false)
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}

	err = p.CancelSubscription(ctx, "sub_123", true)
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_GetSubscription(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	_, err := p.GetSubscription(ctx, "sub_123")
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_ReportUsage(t *testing.T) {
	p := payment.NewNoopProvider()
	ctx := context.Background()

	err := p.ReportUsage(ctx, "si_123", 1000, time.Now())
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_ParseWebhook(t *testing.T) {
	p := payment.NewNoopProvider()

	_, _, err := p.ParseWebhook([]byte("payload"), "signature")
	if err != payment.ErrPaymentsDisabled {
		t.Errorf("expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNewProvider_None(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "none",
	}

	p, err := payment.NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("expected noop provider")
	}
}

func TestNewProvider_Empty(t *testing.T) {
	s := settings.Settings{}

	p, err := payment.NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "none" {
		t.Errorf("expected noop provider for empty setting")
	}
}

func TestNewProvider_Unknown(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "unknown-provider",
	}

	_, err := payment.NewProvider(s)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewProvider_Stripe_MissingKey(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider: "stripe",
		// Missing secret key
	}

	_, err := payment.NewProvider(s)
	if err == nil {
		t.Error("expected error for missing stripe secret key")
	}
}

func TestNewProvider_Stripe_Valid(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:        "stripe",
		settings.KeyPaymentStripeSecretKey: "sk_test_123",
		settings.KeyPaymentStripePublicKey: "pk_test_123",
	}

	p, err := payment.NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "stripe" {
		t.Errorf("Name() = %s, want stripe", p.Name())
	}
}

func TestNewProvider_Paddle_MissingConfig(t *testing.T) {
	tests := []struct {
		name string
		s    settings.Settings
	}{
		{
			"missing vendor ID",
			settings.Settings{
				settings.KeyPaymentProvider:     "paddle",
				settings.KeyPaymentPaddleAPIKey: "key123",
			},
		},
		{
			"missing API key",
			settings.Settings{
				settings.KeyPaymentProvider:       "paddle",
				settings.KeyPaymentPaddleVendorID: "vendor123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := payment.NewProvider(tt.s)
			if err == nil {
				t.Error("expected error for missing paddle config")
			}
		})
	}
}

func TestNewProvider_Paddle_Valid(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:       "paddle",
		settings.KeyPaymentPaddleVendorID: "vendor123",
		settings.KeyPaymentPaddleAPIKey:   "key123",
	}

	p, err := payment.NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "paddle" {
		t.Errorf("Name() = %s, want paddle", p.Name())
	}
}

func TestNewProvider_LemonSqueezy_MissingConfig(t *testing.T) {
	tests := []struct {
		name string
		s    settings.Settings
	}{
		{
			"missing API key",
			settings.Settings{
				settings.KeyPaymentProvider:      "lemonsqueezy",
				settings.KeyPaymentLemonStoreID:  "store123",
			},
		},
		{
			"missing store ID",
			settings.Settings{
				settings.KeyPaymentProvider:    "lemonsqueezy",
				settings.KeyPaymentLemonAPIKey: "key123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := payment.NewProvider(tt.s)
			if err == nil {
				t.Error("expected error for missing lemonsqueezy config")
			}
		})
	}
}

func TestNewProvider_LemonSqueezy_Valid(t *testing.T) {
	s := settings.Settings{
		settings.KeyPaymentProvider:     "lemonsqueezy",
		settings.KeyPaymentLemonAPIKey:  "key123",
		settings.KeyPaymentLemonStoreID: "store123",
	}

	p, err := payment.NewProvider(s)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if p.Name() != "lemonsqueezy" {
		t.Errorf("Name() = %s, want lemonsqueezy", p.Name())
	}
}

func TestErrPaymentsDisabled(t *testing.T) {
	if payment.ErrPaymentsDisabled.Error() != "payments are not configured" {
		t.Error("unexpected error message")
	}
}
