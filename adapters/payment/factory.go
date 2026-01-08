package payment

import (
	"fmt"

	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
)

// NewProvider creates a payment provider based on settings.
func NewProvider(s settings.Settings) (ports.PaymentProvider, error) {
	provider := s.Get(settings.KeyPaymentProvider)

	switch provider {
	case "stripe":
		config := StripeConfig{
			SecretKey:     s.Get(settings.KeyPaymentStripeSecretKey),
			PublicKey:     s.Get(settings.KeyPaymentStripePublicKey),
			WebhookSecret: s.Get(settings.KeyPaymentStripeWebhookSecret),
		}
		if config.SecretKey == "" {
			return nil, fmt.Errorf("stripe secret key is required")
		}
		return NewStripeProvider(config), nil

	case "paddle":
		config := PaddleConfig{
			VendorID:      s.Get(settings.KeyPaymentPaddleVendorID),
			APIKey:        s.Get(settings.KeyPaymentPaddleAPIKey),
			PublicKey:     s.Get(settings.KeyPaymentPaddlePublicKey),
			WebhookSecret: s.Get(settings.KeyPaymentPaddleWebhookSecret),
		}
		if config.VendorID == "" || config.APIKey == "" {
			return nil, fmt.Errorf("paddle vendor ID and API key are required")
		}
		return NewPaddleProvider(config), nil

	case "lemonsqueezy":
		config := LemonSqueezyConfig{
			APIKey:        s.Get(settings.KeyPaymentLemonAPIKey),
			StoreID:       s.Get(settings.KeyPaymentLemonStoreID),
			WebhookSecret: s.Get(settings.KeyPaymentLemonWebhookSecret),
		}
		if config.APIKey == "" || config.StoreID == "" {
			return nil, fmt.Errorf("lemonsqueezy API key and store ID are required")
		}
		return NewLemonSqueezyProvider(config), nil

	case "dummy", "test":
		// Dummy provider for development/testing - simulates successful payments
		baseURL := s.Get(settings.KeyPortalBaseURL)
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
		return NewDummyProvider(baseURL), nil

	case "none", "":
		return NewNoopProvider(), nil

	default:
		return nil, fmt.Errorf("unknown payment provider: %s", provider)
	}
}
