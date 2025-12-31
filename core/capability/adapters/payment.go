// Package adapters provides wrappers that adapt existing implementations
// to the capability interfaces.
package adapters

import (
	"context"

	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/ports"
)

// PaymentAdapter wraps a ports.PaymentProvider to implement capability.PaymentProvider.
// This allows existing payment implementations (Stripe, Paddle, LemonSqueezy)
// to be used with the capability system.
type PaymentAdapter struct {
	inner ports.PaymentProvider
}

// WrapPayment creates a capability.PaymentProvider from a ports.PaymentProvider.
func WrapPayment(inner ports.PaymentProvider) *PaymentAdapter {
	return &PaymentAdapter{inner: inner}
}

func (a *PaymentAdapter) Name() string {
	return a.inner.Name()
}

func (a *PaymentAdapter) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	return a.inner.CreateCustomer(ctx, email, name, userID)
}

func (a *PaymentAdapter) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	return a.inner.CreateCheckoutSession(ctx, customerID, priceID, successURL, cancelURL, trialDays)
}

func (a *PaymentAdapter) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return a.inner.CreatePortalSession(ctx, customerID, returnURL)
}

func (a *PaymentAdapter) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return a.inner.CancelSubscription(ctx, subscriptionID, immediately)
}

func (a *PaymentAdapter) GetSubscription(ctx context.Context, subscriptionID string) (capability.Subscription, error) {
	sub, err := a.inner.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return capability.Subscription{}, err
	}
	return capability.Subscription{
		ID:         sub.ID,
		CustomerID: sub.UserID, // mapping UserID to CustomerID
		PriceID:    sub.PlanID,
		Status:     string(sub.Status),
		CurrentPeriod: capability.Period{
			Start: sub.CurrentPeriodEnd.Add(-30 * 24 * 3600 * 1e9).Unix(), // Approximate
			End:   sub.CurrentPeriodEnd.Unix(),
		},
	}, nil
}

func (a *PaymentAdapter) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp int64) error {
	// Convert int64 timestamp to time.Time
	ts := timeFromUnix(timestamp)
	return a.inner.ReportUsage(ctx, subscriptionItemID, quantity, ts)
}

func (a *PaymentAdapter) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return a.inner.ParseWebhook(payload, signature)
}

func (a *PaymentAdapter) CreatePrice(ctx context.Context, name string, amountCents int64, interval string) (string, error) {
	// Note: The existing ports.PaymentProvider doesn't have CreatePrice.
	// For now, return an error. Implementers should override this or update the inner provider.
	return "", ErrNotImplemented
}

// Ensure PaymentAdapter implements capability.PaymentProvider
var _ capability.PaymentProvider = (*PaymentAdapter)(nil)
