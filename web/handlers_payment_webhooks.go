package web

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// PaymentWebhookHandler handles incoming webhooks from payment providers.
type PaymentWebhookHandler struct {
	payment        ports.PaymentProvider
	webhookHandler ports.PaymentWebhookHandler
	logger         zerolog.Logger
}

// NewPaymentWebhookHandler creates a new payment webhook handler.
func NewPaymentWebhookHandler(
	payment ports.PaymentProvider,
	webhookHandler ports.PaymentWebhookHandler,
	logger zerolog.Logger,
) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{
		payment:        payment,
		webhookHandler: webhookHandler,
		logger:         logger,
	}
}

// Routes returns the chi router for payment webhooks.
// These routes are mounted at /payment-webhooks.
func (h *PaymentWebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Payment provider webhook endpoints
	// These receive POST requests from Stripe/Paddle/LemonSqueezy
	r.Post("/stripe", h.HandleStripeWebhook)
	r.Post("/paddle", h.HandlePaddleWebhook)
	r.Post("/lemonsqueezy", h.HandleLemonSqueezyWebhook)

	return r
}

// ServeHTTP implements http.Handler for use with http.Handle.
func (h *PaymentWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Routes().ServeHTTP(w, r)
}

// HandleStripeWebhook handles Stripe webhook events.
func (h *PaymentWebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "stripe", "Stripe-Signature")
}

// HandlePaddleWebhook handles Paddle webhook events.
func (h *PaymentWebhookHandler) HandlePaddleWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "paddle", "Paddle-Signature")
}

// HandleLemonSqueezyWebhook handles LemonSqueezy webhook events.
func (h *PaymentWebhookHandler) HandleLemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "lemonsqueezy", "X-Signature")
}

// handleWebhook is the common webhook handling logic.
func (h *PaymentWebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request, provider, signatureHeader string) {
	ctx := r.Context()

	// Check if payment provider matches
	if h.payment == nil || h.payment.Name() != provider {
		h.logger.Warn().
			Str("expected_provider", provider).
			Str("configured_provider", h.paymentProviderName()).
			Msg("webhook received for wrong payment provider")
		http.Error(w, "wrong payment provider", http.StatusBadRequest)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to read webhook body")
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Get signature from header
	signature := r.Header.Get(signatureHeader)

	// Parse and validate webhook
	eventType, data, err := h.payment.ParseWebhook(body, signature)
	if err != nil {
		h.logger.Warn().Err(err).
			Str("provider", provider).
			Msg("invalid webhook signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	h.logger.Info().
		Str("provider", provider).
		Str("event_type", eventType).
		Msg("received payment webhook")

	// Handle event based on type
	if err := h.dispatchEvent(ctx, provider, eventType, data); err != nil {
		h.logger.Error().Err(err).
			Str("provider", provider).
			Str("event_type", eventType).
			Msg("failed to handle webhook event")
		// Still return 200 to prevent retries for application errors
		// Payment providers will retry on 4xx/5xx responses
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// dispatchEvent routes the webhook event to the appropriate handler.
func (h *PaymentWebhookHandler) dispatchEvent(ctx context.Context, provider, eventType string, data map[string]any) error {
	// Map common event types across providers
	switch {
	// Checkout completed events
	case h.isCheckoutCompleted(provider, eventType):
		customerID, subscriptionID, planID := h.extractCheckoutData(provider, data)
		return h.webhookHandler.HandleCheckoutCompleted(ctx, customerID, subscriptionID, planID)

	// Subscription updated events
	case h.isSubscriptionUpdated(provider, eventType):
		subscriptionID, status := h.extractSubscriptionUpdate(provider, data)
		return h.webhookHandler.HandleSubscriptionUpdated(ctx, subscriptionID, status)

	// Subscription cancelled events
	case h.isSubscriptionCancelled(provider, eventType):
		subscriptionID := h.extractSubscriptionID(provider, data)
		return h.webhookHandler.HandleSubscriptionCancelled(ctx, subscriptionID)

	// Invoice paid events
	case h.isInvoicePaid(provider, eventType):
		invoiceID, customerID, amount := h.extractInvoicePaid(provider, data)
		return h.webhookHandler.HandleInvoicePaid(ctx, invoiceID, customerID, amount)

	// Invoice failed events
	case h.isInvoiceFailed(provider, eventType):
		invoiceID, customerID, _ := h.extractInvoicePaid(provider, data)
		return h.webhookHandler.HandleInvoiceFailed(ctx, invoiceID, customerID)

	default:
		h.logger.Debug().
			Str("provider", provider).
			Str("event_type", eventType).
			Msg("ignoring unhandled webhook event type")
		return nil
	}
}

// Event type matchers

func (h *PaymentWebhookHandler) isCheckoutCompleted(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "checkout.session.completed"
	case "paddle":
		return eventType == "transaction.completed" || eventType == "subscription.created"
	case "lemonsqueezy":
		return eventType == "order_created" || eventType == "subscription_created"
	}
	return false
}

func (h *PaymentWebhookHandler) isSubscriptionUpdated(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "customer.subscription.updated"
	case "paddle":
		return eventType == "subscription.updated"
	case "lemonsqueezy":
		return eventType == "subscription_updated"
	}
	return false
}

func (h *PaymentWebhookHandler) isSubscriptionCancelled(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "customer.subscription.deleted"
	case "paddle":
		return eventType == "subscription.canceled"
	case "lemonsqueezy":
		return eventType == "subscription_cancelled"
	}
	return false
}

func (h *PaymentWebhookHandler) isInvoicePaid(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "invoice.paid"
	case "paddle":
		return eventType == "transaction.paid"
	case "lemonsqueezy":
		return eventType == "subscription_payment_success"
	}
	return false
}

func (h *PaymentWebhookHandler) isInvoiceFailed(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "invoice.payment_failed"
	case "paddle":
		return eventType == "transaction.payment_failed"
	case "lemonsqueezy":
		return eventType == "subscription_payment_failed"
	}
	return false
}

// Data extractors

func (h *PaymentWebhookHandler) extractCheckoutData(provider string, data map[string]any) (customerID, subscriptionID, planID string) {
	switch provider {
	case "stripe":
		customerID, _ = data["customer"].(string)
		subscriptionID, _ = data["subscription"].(string)
		// Plan ID from metadata
		if meta, ok := data["metadata"].(map[string]any); ok {
			planID, _ = meta["plan_id"].(string)
		}
	case "paddle":
		customerID, _ = data["customer_id"].(string)
		subscriptionID, _ = data["subscription_id"].(string)
		// Extract from items
		if items, ok := data["items"].([]any); ok && len(items) > 0 {
			if item, ok := items[0].(map[string]any); ok {
				if price, ok := item["price"].(map[string]any); ok {
					planID, _ = price["id"].(string)
				}
			}
		}
		// Check custom_data for plan_id
		if customData, ok := data["custom_data"].(map[string]any); ok {
			if pid, ok := customData["plan_id"].(string); ok {
				planID = pid
			}
		}
	case "lemonsqueezy":
		if attrs, ok := data["data"].(map[string]any); ok {
			if a, ok := attrs["attributes"].(map[string]any); ok {
				customerID, _ = a["customer_id"].(string)
				subscriptionID, _ = a["subscription_id"].(string)
				if variant, _ := a["variant_id"].(float64); variant > 0 {
					planID = formatInt(int64(variant))
				}
			}
		}
	}
	return
}

func (h *PaymentWebhookHandler) extractSubscriptionUpdate(provider string, data map[string]any) (subscriptionID string, status billing.SubscriptionStatus) {
	subscriptionID = h.extractSubscriptionID(provider, data)
	status = billing.SubscriptionStatusActive // Default

	switch provider {
	case "stripe":
		if s, ok := data["status"].(string); ok {
			status = mapStripeStatus(s)
		}
	case "paddle":
		if s, ok := data["status"].(string); ok {
			status = mapPaddleStatus(s)
		}
	case "lemonsqueezy":
		if attrs, ok := data["data"].(map[string]any); ok {
			if a, ok := attrs["attributes"].(map[string]any); ok {
				if s, ok := a["status"].(string); ok {
					status = mapLemonStatus(s)
				}
			}
		}
	}
	return
}

func (h *PaymentWebhookHandler) extractSubscriptionID(provider string, data map[string]any) string {
	switch provider {
	case "stripe":
		if id, ok := data["id"].(string); ok {
			return id
		}
	case "paddle":
		if id, ok := data["subscription_id"].(string); ok {
			return id
		}
		if id, ok := data["id"].(string); ok {
			return id
		}
	case "lemonsqueezy":
		if attrs, ok := data["data"].(map[string]any); ok {
			if id, ok := attrs["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

func (h *PaymentWebhookHandler) extractInvoicePaid(provider string, data map[string]any) (invoiceID, customerID string, amount int64) {
	switch provider {
	case "stripe":
		invoiceID, _ = data["id"].(string)
		customerID, _ = data["customer"].(string)
		if total, ok := data["amount_paid"].(float64); ok {
			amount = int64(total)
		}
	case "paddle":
		invoiceID, _ = data["id"].(string)
		customerID, _ = data["customer_id"].(string)
		if details, ok := data["details"].(map[string]any); ok {
			if totals, ok := details["totals"].(map[string]any); ok {
				if total, ok := totals["total"].(string); ok {
					// Parse string amount
					amount = parseAmount(total)
				}
			}
		}
	case "lemonsqueezy":
		if attrs, ok := data["data"].(map[string]any); ok {
			invoiceID, _ = attrs["id"].(string)
			if a, ok := attrs["attributes"].(map[string]any); ok {
				if custID, ok := a["customer_id"].(float64); ok {
					customerID = formatInt(int64(custID))
				}
				if total, ok := a["total"].(float64); ok {
					amount = int64(total)
				}
			}
		}
	}
	return
}

func (h *PaymentWebhookHandler) paymentProviderName() string {
	if h.payment != nil {
		return h.payment.Name()
	}
	return "none"
}

// Status mappers

func mapStripeStatus(s string) billing.SubscriptionStatus {
	switch s {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "canceled":
		return billing.SubscriptionStatusCancelled
	case "unpaid":
		return billing.SubscriptionStatusUnpaid
	case "trialing":
		return billing.SubscriptionStatusTrialing
	case "paused":
		return billing.SubscriptionStatusPaused
	default:
		return billing.SubscriptionStatusActive
	}
}

func mapPaddleStatus(s string) billing.SubscriptionStatus {
	switch s {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "canceled", "cancelled":
		return billing.SubscriptionStatusCancelled
	case "paused":
		return billing.SubscriptionStatusPaused
	case "trialing":
		return billing.SubscriptionStatusTrialing
	default:
		return billing.SubscriptionStatusActive
	}
}

func mapLemonStatus(s string) billing.SubscriptionStatus {
	switch s {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "cancelled", "expired":
		return billing.SubscriptionStatusCancelled
	case "paused", "on_trial":
		if s == "on_trial" {
			return billing.SubscriptionStatusTrialing
		}
		return billing.SubscriptionStatusPaused
	default:
		return billing.SubscriptionStatusActive
	}
}

// Helper functions

func formatInt(n int64) string {
	return strings.TrimPrefix(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					"0000000000000000000"+itoa(n), "0000000000000000000", ""),
				"000000000000000000", ""),
			"00000000000000000", ""),
		"0000000000000000", ""), "0")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func parseAmount(s string) int64 {
	// Parse decimal string to cents (e.g., "19.99" -> 1999)
	var result int64
	var decimal bool
	var decimalPlaces int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
			if decimal {
				decimalPlaces++
			}
		} else if c == '.' {
			decimal = true
		}
	}
	// Adjust for decimal places (assuming 2 decimal places for cents)
	for decimalPlaces < 2 {
		result *= 10
		decimalPlaces++
	}
	return result
}
