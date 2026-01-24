// Package app contains the PaymentWebhookService for handling payment provider webhooks.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// PaymentWebhookService handles incoming webhooks from payment providers.
// It implements ports.PaymentWebhookHandler interface.
// All business logic is pure - I/O happens at the edges via injected stores.
type PaymentWebhookService struct {
	users         ports.UserStore
	subscriptions ports.SubscriptionStore
	plans         ports.PlanStore
	idGen         ports.IDGenerator
	logger        zerolog.Logger
}

// NewPaymentWebhookService creates a new payment webhook service.
func NewPaymentWebhookService(
	users ports.UserStore,
	subscriptions ports.SubscriptionStore,
	plans ports.PlanStore,
	idGen ports.IDGenerator,
	logger zerolog.Logger,
) *PaymentWebhookService {
	return &PaymentWebhookService{
		users:         users,
		subscriptions: subscriptions,
		plans:         plans,
		idGen:         idGen,
		logger:        logger,
	}
}

// HandleCheckoutCompleted handles successful checkout events from payment providers.
// Creates a subscription record and updates the user's plan.
func (s *PaymentWebhookService) HandleCheckoutCompleted(
	ctx context.Context,
	customerID, subscriptionID, planID string,
) error {
	s.logger.Info().
		Str("customer_id", customerID).
		Str("subscription_id", subscriptionID).
		Str("plan_id", planID).
		Msg("handling checkout completed webhook")

	// Look up user by StripeID (customer_id from webhook)
	user, err := s.findUserByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("customer_id", customerID).
			Msg("failed to find user for customer")
		return err
	}

	// Verify plan exists
	plan, err := s.plans.Get(ctx, planID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("plan_id", planID).
			Msg("failed to get plan")
		return err
	}

	now := time.Now().UTC()

	// Create subscription record
	sub := billing.Subscription{
		ID:                 s.idGen.New(),
		UserID:             user.ID,
		PlanID:             plan.ID,
		ProviderID:         subscriptionID,
		Provider:           "stripe", // Will be set by caller based on endpoint
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0), // Default to 1 month
		CancelAtPeriodEnd:  false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.subscriptions.Create(ctx, sub); err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", subscriptionID).
			Msg("failed to create subscription")
		return err
	}

	// Update user's plan (with compensation if this fails)
	user.PlanID = plan.ID
	user.UpdatedAt = now
	if err := s.users.Update(ctx, user); err != nil {
		s.logger.Error().Err(err).
			Str("user_id", user.ID).
			Str("plan_id", plan.ID).
			Msg("failed to update user plan, attempting to rollback subscription")

		// Compensation: Mark subscription as cancelled to avoid orphan
		sub.Status = billing.SubscriptionStatusCancelled
		sub.CancelledAt = &now
		if rollbackErr := s.subscriptions.Update(ctx, sub); rollbackErr != nil {
			s.logger.Error().Err(rollbackErr).
				Str("subscription_id", sub.ID).
				Msg("failed to rollback subscription after user update failure")
		}
		return err
	}

	s.logger.Info().
		Str("user_id", user.ID).
		Str("plan_id", plan.ID).
		Str("subscription_id", sub.ID).
		Msg("checkout completed: subscription created and user plan updated")

	return nil
}

// HandleSubscriptionUpdated handles subscription update events.
// Updates subscription status in the database.
func (s *PaymentWebhookService) HandleSubscriptionUpdated(
	ctx context.Context,
	subscriptionID string,
	status billing.SubscriptionStatus,
) error {
	s.logger.Info().
		Str("subscription_id", subscriptionID).
		Str("status", string(status)).
		Msg("handling subscription updated webhook")

	// Look up subscription by provider ID
	sub, err := s.subscriptions.GetByProviderID(ctx, subscriptionID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", subscriptionID).
			Msg("failed to find subscription")
		return err
	}

	// Update status
	sub.Status = status
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subscriptions.Update(ctx, sub); err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", sub.ID).
			Msg("failed to update subscription")
		return err
	}

	s.logger.Info().
		Str("subscription_id", sub.ID).
		Str("user_id", sub.UserID).
		Str("status", string(status)).
		Msg("subscription status updated")

	return nil
}

// HandleSubscriptionCancelled handles subscription cancellation events.
// Marks the subscription as cancelled and optionally updates user's plan.
func (s *PaymentWebhookService) HandleSubscriptionCancelled(
	ctx context.Context,
	subscriptionID string,
) error {
	s.logger.Info().
		Str("subscription_id", subscriptionID).
		Msg("handling subscription cancelled webhook")

	// Look up subscription by provider ID
	sub, err := s.subscriptions.GetByProviderID(ctx, subscriptionID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", subscriptionID).
			Msg("failed to find subscription")
		return err
	}

	now := time.Now().UTC()

	// Update subscription status
	sub.Status = billing.SubscriptionStatusCancelled
	sub.CancelledAt = &now
	sub.UpdatedAt = now

	if err := s.subscriptions.Update(ctx, sub); err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", sub.ID).
			Msg("failed to update subscription")
		return err
	}

	// Get user and revert to default plan
	user, err := s.users.Get(ctx, sub.UserID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("user_id", sub.UserID).
			Msg("failed to get user for plan reversion")
		return err
	}

	// Find default plan
	defaultPlan, err := s.findDefaultPlan(ctx)
	if err != nil {
		s.logger.Warn().Err(err).
			Str("user_id", user.ID).
			Msg("no default plan found, user plan unchanged")
	} else {
		// Revert user to default plan
		user.PlanID = defaultPlan.ID
		user.UpdatedAt = now
		if err := s.users.Update(ctx, user); err != nil {
			s.logger.Error().Err(err).
				Str("user_id", user.ID).
				Msg("failed to revert user to default plan")
			return err
		}

		s.logger.Info().
			Str("user_id", user.ID).
			Str("old_plan", sub.PlanID).
			Str("new_plan", defaultPlan.ID).
			Msg("user reverted to default plan after subscription cancellation")
	}

	s.logger.Info().
		Str("subscription_id", sub.ID).
		Str("user_id", sub.UserID).
		Msg("subscription cancelled")

	return nil
}

// HandleInvoicePaid handles successful invoice payment events.
func (s *PaymentWebhookService) HandleInvoicePaid(
	ctx context.Context,
	invoiceID, customerID string,
	amountPaid int64,
) error {
	s.logger.Info().
		Str("invoice_id", invoiceID).
		Str("customer_id", customerID).
		Int64("amount_paid", amountPaid).
		Msg("handling invoice paid webhook")

	// Find user by customer ID for logging
	user, err := s.findUserByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Warn().Err(err).
			Str("customer_id", customerID).
			Msg("could not find user for invoice paid event")
		// Don't return error - invoice might be for a user we don't track
		return nil
	}

	s.logger.Info().
		Str("user_id", user.ID).
		Str("invoice_id", invoiceID).
		Int64("amount_cents", amountPaid).
		Msg("invoice payment recorded")

	return nil
}

// HandleInvoiceFailed handles failed invoice payment events.
func (s *PaymentWebhookService) HandleInvoiceFailed(
	ctx context.Context,
	invoiceID, customerID string,
) error {
	s.logger.Warn().
		Str("invoice_id", invoiceID).
		Str("customer_id", customerID).
		Msg("handling invoice failed webhook")

	// Find user by customer ID
	user, err := s.findUserByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Warn().Err(err).
			Str("customer_id", customerID).
			Msg("could not find user for invoice failed event")
		return nil
	}

	// Update user's subscription to past_due if exists
	sub, err := s.subscriptions.GetByUser(ctx, user.ID)
	if err != nil {
		s.logger.Debug().
			Str("user_id", user.ID).
			Msg("no active subscription found for user with failed invoice")
		return nil
	}

	sub.Status = billing.SubscriptionStatusPastDue
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subscriptions.Update(ctx, sub); err != nil {
		s.logger.Error().Err(err).
			Str("subscription_id", sub.ID).
			Msg("failed to mark subscription as past due")
		return err
	}

	s.logger.Warn().
		Str("user_id", user.ID).
		Str("subscription_id", sub.ID).
		Str("invoice_id", invoiceID).
		Msg("subscription marked as past due due to failed invoice")

	return nil
}

// findUserByCustomerID finds a user by their payment provider customer ID.
// Uses indexed lookup for efficiency (O(1) instead of O(n)).
func (s *PaymentWebhookService) findUserByCustomerID(ctx context.Context, customerID string) (ports.User, error) {
	user, err := s.users.GetByStripeID(ctx, customerID)
	if err != nil {
		return ports.User{}, errors.New("user not found for customer ID")
	}
	return user, nil
}

// findDefaultPlan finds the default plan.
func (s *PaymentWebhookService) findDefaultPlan(ctx context.Context) (ports.Plan, error) {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return ports.Plan{}, err
	}

	for _, p := range plans {
		if p.IsDefault && p.Enabled {
			return p, nil
		}
	}

	return ports.Plan{}, errors.New("no default plan found")
}

// Ensure interface compliance.
var _ ports.PaymentWebhookHandler = (*PaymentWebhookService)(nil)
