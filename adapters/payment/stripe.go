// Package payment provides payment provider adapters.
package payment

import (
	"context"
	"encoding/json"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/usagerecord"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeConfig holds Stripe configuration.
type StripeConfig struct {
	SecretKey     string
	PublicKey     string
	WebhookSecret string
}

// StripeProvider implements ports.PaymentProvider for Stripe.
type StripeProvider struct {
	config StripeConfig
}

// NewStripeProvider creates a new Stripe payment provider.
func NewStripeProvider(config StripeConfig) *StripeProvider {
	stripe.Key = config.SecretKey
	return &StripeProvider{config: config}
}

// Name returns the provider name.
func (p *StripeProvider) Name() string {
	return "stripe"
}

// CreateCustomer creates a customer in Stripe.
func (p *StripeProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	params.AddMetadata("user_id", userID)

	c, err := customer.New(params)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout session.
func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

// CreatePortalSession creates a customer portal session.
func (p *StripeProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

// CancelSubscription cancels a subscription.
func (p *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	if immediately {
		_, err := subscription.Cancel(subscriptionID, nil)
		return err
	}

	// Cancel at period end
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := subscription.Update(subscriptionID, params)
	return err
}

// GetSubscription retrieves subscription details.
func (p *StripeProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	s, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return billing.Subscription{}, err
	}

	return billing.Subscription{
		ID:               s.ID,
		UserID:           s.Metadata["user_id"],
		PlanID:           s.Items.Data[0].Price.ID,
		Status:           mapStripeStatus(s.Status),
		CurrentPeriodEnd: time.Unix(s.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd: s.CancelAtPeriodEnd,
	}, nil
}

// ReportUsage reports metered usage.
func (p *StripeProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	params := &stripe.UsageRecordParams{
		SubscriptionItem: stripe.String(subscriptionItemID),
		Quantity:         stripe.Int64(quantity),
		Timestamp:        stripe.Int64(timestamp.Unix()),
		Action:           stripe.String(string(stripe.UsageRecordActionIncrement)),
	}

	_, err := usagerecord.New(params)
	return err
}

// ParseWebhook parses and validates a Stripe webhook.
func (p *StripeProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	event, err := webhook.ConstructEvent(payload, signature, p.config.WebhookSecret)
	if err != nil {
		return "", nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(event.Data.Raw, &data); err != nil {
		return "", nil, err
	}

	return string(event.Type), data, nil
}

func mapStripeStatus(status stripe.SubscriptionStatus) billing.SubscriptionStatus {
	switch status {
	case stripe.SubscriptionStatusActive:
		return billing.SubscriptionStatusActive
	case stripe.SubscriptionStatusPastDue:
		return billing.SubscriptionStatusPastDue
	case stripe.SubscriptionStatusCanceled:
		return billing.SubscriptionStatusCancelled
	case stripe.SubscriptionStatusUnpaid:
		return billing.SubscriptionStatusUnpaid
	case stripe.SubscriptionStatusTrialing:
		return billing.SubscriptionStatusTrialing
	default:
		return billing.SubscriptionStatusActive
	}
}
