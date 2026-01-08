package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/google/uuid"
)

// DummyProvider is a test/demo payment provider that simulates successful payments.
// Use this for development and demos when real payment credentials aren't available.
type DummyProvider struct {
	baseURL string
}

// NewDummyProvider creates a new dummy payment provider.
func NewDummyProvider(baseURL string) *DummyProvider {
	return &DummyProvider{baseURL: baseURL}
}

// Name returns the provider name.
func (p *DummyProvider) Name() string {
	return "dummy"
}

// CreateCustomer simulates creating a customer and returns a fake customer ID.
func (p *DummyProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	// Generate a fake customer ID
	return fmt.Sprintf("cus_dummy_%s", userID[:8]), nil
}

// CreateCheckoutSession simulates checkout by redirecting directly to success URL.
// This allows testing the full upgrade flow without real payment.
func (p *DummyProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	// In dummy mode, skip actual checkout and redirect to success
	return successURL, nil
}

// CreatePortalSession returns a redirect to the plans page (no external portal).
func (p *DummyProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return returnURL, nil
}

// CancelSubscription simulates successful cancellation.
func (p *DummyProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return nil
}

// GetSubscription returns a dummy subscription.
func (p *DummyProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	return billing.Subscription{
		ID:                 subscriptionID,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().UTC(),
		CurrentPeriodEnd:   time.Now().UTC().AddDate(0, 1, 0),
	}, nil
}

// ReportUsage simulates successful usage reporting.
func (p *DummyProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	return nil
}

// ParseWebhook simulates webhook parsing.
func (p *DummyProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return "dummy.event", map[string]any{
		"id": uuid.New().String(),
	}, nil
}
