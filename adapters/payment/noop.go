package payment

import (
	"context"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/billing"
)

var (
	// ErrPaymentsDisabled is returned when payments are not configured.
	ErrPaymentsDisabled = errors.New("payments are not configured")
)

// NoopProvider is a no-op payment provider for when payments are disabled.
type NoopProvider struct{}

// NewNoopProvider creates a new no-op payment provider.
func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

// Name returns the provider name.
func (p *NoopProvider) Name() string {
	return "none"
}

// CreateCustomer returns an error as payments are disabled.
func (p *NoopProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	return "", ErrPaymentsDisabled
}

// CreateCheckoutSession returns an error as payments are disabled.
func (p *NoopProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	return "", ErrPaymentsDisabled
}

// CreatePortalSession returns an error as payments are disabled.
func (p *NoopProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "", ErrPaymentsDisabled
}

// CancelSubscription returns an error as payments are disabled.
func (p *NoopProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return ErrPaymentsDisabled
}

// GetSubscription returns an error as payments are disabled.
func (p *NoopProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	return billing.Subscription{}, ErrPaymentsDisabled
}

// ReportUsage returns an error as payments are disabled.
func (p *NoopProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	return ErrPaymentsDisabled
}

// ParseWebhook returns an error as payments are disabled.
func (p *NoopProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return "", nil, ErrPaymentsDisabled
}
