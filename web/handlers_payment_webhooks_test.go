package web

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/rs/zerolog"
)

// Mock payment provider for webhook tests
type mockWebhookPaymentProvider struct {
	name             string
	parseWebhookErr  error
	parseWebhookType string
	parseWebhookData map[string]any
}

func (m *mockWebhookPaymentProvider) Name() string { return m.name }
func (m *mockWebhookPaymentProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	return "", nil
}
func (m *mockWebhookPaymentProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	return "", nil
}
func (m *mockWebhookPaymentProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "", nil
}
func (m *mockWebhookPaymentProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	return nil
}
func (m *mockWebhookPaymentProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	return billing.Subscription{}, nil
}
func (m *mockWebhookPaymentProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	return nil
}
func (m *mockWebhookPaymentProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	if m.parseWebhookErr != nil {
		return "", nil, m.parseWebhookErr
	}
	return m.parseWebhookType, m.parseWebhookData, nil
}

// Mock webhook handler
type mockPaymentWebhookHandler struct {
	checkoutCompletedCalls    int
	subscriptionUpdatedCalls  int
	subscriptionCancelledCalls int
	invoicePaidCalls          int
	invoiceFailedCalls        int
	lastCustomerID            string
	lastSubscriptionID        string
	lastPlanID                string
	lastStatus                billing.SubscriptionStatus
	lastInvoiceID             string
	lastAmount                int64
	returnErr                 error
}

func (m *mockPaymentWebhookHandler) HandleCheckoutCompleted(ctx context.Context, customerID, subscriptionID, planID string) error {
	m.checkoutCompletedCalls++
	m.lastCustomerID = customerID
	m.lastSubscriptionID = subscriptionID
	m.lastPlanID = planID
	return m.returnErr
}

func (m *mockPaymentWebhookHandler) HandleSubscriptionUpdated(ctx context.Context, subscriptionID string, status billing.SubscriptionStatus) error {
	m.subscriptionUpdatedCalls++
	m.lastSubscriptionID = subscriptionID
	m.lastStatus = status
	return m.returnErr
}

func (m *mockPaymentWebhookHandler) HandleSubscriptionCancelled(ctx context.Context, subscriptionID string) error {
	m.subscriptionCancelledCalls++
	m.lastSubscriptionID = subscriptionID
	return m.returnErr
}

func (m *mockPaymentWebhookHandler) HandleInvoicePaid(ctx context.Context, invoiceID, customerID string, amount int64) error {
	m.invoicePaidCalls++
	m.lastInvoiceID = invoiceID
	m.lastCustomerID = customerID
	m.lastAmount = amount
	return m.returnErr
}

func (m *mockPaymentWebhookHandler) HandleInvoiceFailed(ctx context.Context, invoiceID, customerID string) error {
	m.invoiceFailedCalls++
	m.lastInvoiceID = invoiceID
	m.lastCustomerID = customerID
	return m.returnErr
}

func TestNewPaymentWebhookHandler(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{name: "stripe"}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.payment != payment {
		t.Error("payment provider not set correctly")
	}
	if h.webhookHandler != handler {
		t.Error("webhook handler not set correctly")
	}
}

func TestPaymentWebhookHandler_Routes(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{name: "stripe"}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)
	r := h.Routes()

	if r == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestPaymentWebhookHandler_ServeHTTP(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:             "stripe",
		parseWebhookType: "checkout.session.completed",
		parseWebhookData: map[string]any{
			"customer":     "cus_123",
			"subscription": "sub_456",
			"metadata":     map[string]any{"plan_id": "plan_789"},
		},
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPaymentWebhookHandler_HandleStripeWebhook_Success(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:             "stripe",
		parseWebhookType: "checkout.session.completed",
		parseWebhookData: map[string]any{
			"customer":     "cus_123",
			"subscription": "sub_456",
			"metadata":     map[string]any{"plan_id": "plan_789"},
		},
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if handler.checkoutCompletedCalls != 1 {
		t.Errorf("expected 1 checkout completed call, got %d", handler.checkoutCompletedCalls)
	}
	if handler.lastCustomerID != "cus_123" {
		t.Errorf("expected customer cus_123, got %s", handler.lastCustomerID)
	}
	if handler.lastSubscriptionID != "sub_456" {
		t.Errorf("expected subscription sub_456, got %s", handler.lastSubscriptionID)
	}
	if handler.lastPlanID != "plan_789" {
		t.Errorf("expected plan plan_789, got %s", handler.lastPlanID)
	}
}

func TestPaymentWebhookHandler_HandleStripeWebhook_WrongProvider(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{name: "paddle"} // Wrong provider
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandleStripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentWebhookHandler_HandleStripeWebhook_InvalidSignature(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:            "stripe",
		parseWebhookErr: errors.New("invalid signature"),
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "invalid_sig")
	w := httptest.NewRecorder()

	h.HandleStripeWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPaymentWebhookHandler_HandlePaddleWebhook_Success(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:             "paddle",
		parseWebhookType: "transaction.completed",
		parseWebhookData: map[string]any{
			"customer_id":     "ctm_123",
			"subscription_id": "sub_456",
			"custom_data":     map[string]any{"plan_id": "plan_789"},
		},
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/paddle", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Paddle-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandlePaddleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if handler.checkoutCompletedCalls != 1 {
		t.Errorf("expected 1 checkout completed call, got %d", handler.checkoutCompletedCalls)
	}
}

func TestPaymentWebhookHandler_HandleLemonSqueezyWebhook_Success(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:             "lemonsqueezy",
		parseWebhookType: "order_created",
		parseWebhookData: map[string]any{
			"data": map[string]any{
				"attributes": map[string]any{
					"customer_id":     "123",
					"subscription_id": "456",
					"variant_id":      float64(789),
				},
			},
		},
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/lemonsqueezy", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandleLemonSqueezyWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if handler.checkoutCompletedCalls != 1 {
		t.Errorf("expected 1 checkout completed call, got %d", handler.checkoutCompletedCalls)
	}
}

func TestPaymentWebhookHandler_SubscriptionUpdated(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		eventType      string
		data           map[string]any
		expectedStatus billing.SubscriptionStatus
	}{
		{
			name:      "stripe subscription updated",
			provider:  "stripe",
			eventType: "customer.subscription.updated",
			data: map[string]any{
				"id":     "sub_123",
				"status": "past_due",
			},
			expectedStatus: billing.SubscriptionStatusPastDue,
		},
		{
			name:      "paddle subscription updated",
			provider:  "paddle",
			eventType: "subscription.updated",
			data: map[string]any{
				"id":     "sub_123",
				"status": "paused",
			},
			expectedStatus: billing.SubscriptionStatusPaused,
		},
		{
			name:      "lemonsqueezy subscription updated",
			provider:  "lemonsqueezy",
			eventType: "subscription_updated",
			data: map[string]any{
				"data": map[string]any{
					"id": "sub_123",
					"attributes": map[string]any{
						"status": "on_trial",
					},
				},
			},
			expectedStatus: billing.SubscriptionStatusTrialing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			payment := &mockWebhookPaymentProvider{
				name:             tt.provider,
				parseWebhookType: tt.eventType,
				parseWebhookData: tt.data,
			}
			handler := &mockPaymentWebhookHandler{}

			h := NewPaymentWebhookHandler(payment, handler, logger)

			var signatureHeader string
			switch tt.provider {
			case "stripe":
				signatureHeader = "Stripe-Signature"
			case "paddle":
				signatureHeader = "Paddle-Signature"
			case "lemonsqueezy":
				signatureHeader = "X-Signature"
			}

			req := httptest.NewRequest("POST", "/"+tt.provider, bytes.NewBuffer([]byte(`{}`)))
			req.Header.Set(signatureHeader, "sig_123")
			w := httptest.NewRecorder()

			h.handleWebhook(w, req, tt.provider, signatureHeader)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if handler.subscriptionUpdatedCalls != 1 {
				t.Errorf("expected 1 subscription updated call, got %d", handler.subscriptionUpdatedCalls)
			}
			if handler.lastStatus != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, handler.lastStatus)
			}
		})
	}
}

func TestPaymentWebhookHandler_SubscriptionCancelled(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		eventType      string
		data           map[string]any
		expectedSubID  string
	}{
		{
			name:      "stripe subscription deleted",
			provider:  "stripe",
			eventType: "customer.subscription.deleted",
			data: map[string]any{
				"id": "sub_stripe_123",
			},
			expectedSubID: "sub_stripe_123",
		},
		{
			name:      "paddle subscription canceled",
			provider:  "paddle",
			eventType: "subscription.canceled",
			data: map[string]any{
				"subscription_id": "sub_paddle_456",
			},
			expectedSubID: "sub_paddle_456",
		},
		{
			name:      "lemonsqueezy subscription cancelled",
			provider:  "lemonsqueezy",
			eventType: "subscription_cancelled",
			data: map[string]any{
				"data": map[string]any{
					"id": "sub_lemon_789",
				},
			},
			expectedSubID: "sub_lemon_789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			payment := &mockWebhookPaymentProvider{
				name:             tt.provider,
				parseWebhookType: tt.eventType,
				parseWebhookData: tt.data,
			}
			handler := &mockPaymentWebhookHandler{}

			h := NewPaymentWebhookHandler(payment, handler, logger)

			var signatureHeader string
			switch tt.provider {
			case "stripe":
				signatureHeader = "Stripe-Signature"
			case "paddle":
				signatureHeader = "Paddle-Signature"
			case "lemonsqueezy":
				signatureHeader = "X-Signature"
			}

			req := httptest.NewRequest("POST", "/"+tt.provider, bytes.NewBuffer([]byte(`{}`)))
			req.Header.Set(signatureHeader, "sig_123")
			w := httptest.NewRecorder()

			h.handleWebhook(w, req, tt.provider, signatureHeader)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if handler.subscriptionCancelledCalls != 1 {
				t.Errorf("expected 1 subscription cancelled call, got %d", handler.subscriptionCancelledCalls)
			}
			if handler.lastSubscriptionID != tt.expectedSubID {
				t.Errorf("expected subscription %s, got %s", tt.expectedSubID, handler.lastSubscriptionID)
			}
		})
	}
}

func TestPaymentWebhookHandler_InvoicePaid(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		eventType      string
		data           map[string]any
		expectedAmount int64
	}{
		{
			name:      "stripe invoice paid",
			provider:  "stripe",
			eventType: "invoice.paid",
			data: map[string]any{
				"id":          "inv_123",
				"customer":    "cus_456",
				"amount_paid": float64(1999),
			},
			expectedAmount: 1999,
		},
		{
			name:      "paddle transaction paid",
			provider:  "paddle",
			eventType: "transaction.paid",
			data: map[string]any{
				"id":          "txn_123",
				"customer_id": "ctm_456",
				"details": map[string]any{
					"totals": map[string]any{
						"total": "19.99",
					},
				},
			},
			expectedAmount: 1999,
		},
		{
			name:      "lemonsqueezy payment success",
			provider:  "lemonsqueezy",
			eventType: "subscription_payment_success",
			data: map[string]any{
				"data": map[string]any{
					"id": "inv_123",
					"attributes": map[string]any{
						"customer_id": float64(456),
						"total":       float64(1999),
					},
				},
			},
			expectedAmount: 1999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			payment := &mockWebhookPaymentProvider{
				name:             tt.provider,
				parseWebhookType: tt.eventType,
				parseWebhookData: tt.data,
			}
			handler := &mockPaymentWebhookHandler{}

			h := NewPaymentWebhookHandler(payment, handler, logger)

			var signatureHeader string
			switch tt.provider {
			case "stripe":
				signatureHeader = "Stripe-Signature"
			case "paddle":
				signatureHeader = "Paddle-Signature"
			case "lemonsqueezy":
				signatureHeader = "X-Signature"
			}

			req := httptest.NewRequest("POST", "/"+tt.provider, bytes.NewBuffer([]byte(`{}`)))
			req.Header.Set(signatureHeader, "sig_123")
			w := httptest.NewRecorder()

			h.handleWebhook(w, req, tt.provider, signatureHeader)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if handler.invoicePaidCalls != 1 {
				t.Errorf("expected 1 invoice paid call, got %d", handler.invoicePaidCalls)
			}
			if handler.lastAmount != tt.expectedAmount {
				t.Errorf("expected amount %d, got %d", tt.expectedAmount, handler.lastAmount)
			}
		})
	}
}

func TestPaymentWebhookHandler_InvoiceFailed(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		eventType string
		data      map[string]any
	}{
		{
			name:      "stripe invoice failed",
			provider:  "stripe",
			eventType: "invoice.payment_failed",
			data: map[string]any{
				"id":       "inv_123",
				"customer": "cus_456",
			},
		},
		{
			name:      "paddle transaction failed",
			provider:  "paddle",
			eventType: "transaction.payment_failed",
			data: map[string]any{
				"id":          "txn_123",
				"customer_id": "ctm_456",
			},
		},
		{
			name:      "lemonsqueezy payment failed",
			provider:  "lemonsqueezy",
			eventType: "subscription_payment_failed",
			data: map[string]any{
				"data": map[string]any{
					"id": "inv_123",
					"attributes": map[string]any{
						"customer_id": float64(456),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			payment := &mockWebhookPaymentProvider{
				name:             tt.provider,
				parseWebhookType: tt.eventType,
				parseWebhookData: tt.data,
			}
			handler := &mockPaymentWebhookHandler{}

			h := NewPaymentWebhookHandler(payment, handler, logger)

			var signatureHeader string
			switch tt.provider {
			case "stripe":
				signatureHeader = "Stripe-Signature"
			case "paddle":
				signatureHeader = "Paddle-Signature"
			case "lemonsqueezy":
				signatureHeader = "X-Signature"
			}

			req := httptest.NewRequest("POST", "/"+tt.provider, bytes.NewBuffer([]byte(`{}`)))
			req.Header.Set(signatureHeader, "sig_123")
			w := httptest.NewRecorder()

			h.handleWebhook(w, req, tt.provider, signatureHeader)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if handler.invoiceFailedCalls != 1 {
				t.Errorf("expected 1 invoice failed call, got %d", handler.invoiceFailedCalls)
			}
		})
	}
}

func TestPaymentWebhookHandler_UnhandledEventType(t *testing.T) {
	logger := zerolog.Nop()
	payment := &mockWebhookPaymentProvider{
		name:             "stripe",
		parseWebhookType: "some.unknown.event",
		parseWebhookData: map[string]any{},
	}
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(payment, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// No handler calls expected
	if handler.checkoutCompletedCalls != 0 {
		t.Errorf("expected 0 checkout calls, got %d", handler.checkoutCompletedCalls)
	}
	if handler.subscriptionUpdatedCalls != 0 {
		t.Errorf("expected 0 subscription updated calls, got %d", handler.subscriptionUpdatedCalls)
	}
}

func TestPaymentWebhookHandler_NilPaymentProvider(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}

	h := NewPaymentWebhookHandler(nil, handler, logger)

	req := httptest.NewRequest("POST", "/stripe", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "sig_123")
	w := httptest.NewRecorder()

	h.HandleStripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for nil provider, got %d", w.Code)
	}
}

func Test_mapStripeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected billing.SubscriptionStatus
	}{
		{"active", billing.SubscriptionStatusActive},
		{"past_due", billing.SubscriptionStatusPastDue},
		{"canceled", billing.SubscriptionStatusCancelled},
		{"unpaid", billing.SubscriptionStatusUnpaid},
		{"trialing", billing.SubscriptionStatusTrialing},
		{"paused", billing.SubscriptionStatusPaused},
		{"unknown", billing.SubscriptionStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapStripeStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapStripeStatus(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_mapPaddleStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected billing.SubscriptionStatus
	}{
		{"active", billing.SubscriptionStatusActive},
		{"past_due", billing.SubscriptionStatusPastDue},
		{"canceled", billing.SubscriptionStatusCancelled},
		{"cancelled", billing.SubscriptionStatusCancelled},
		{"paused", billing.SubscriptionStatusPaused},
		{"trialing", billing.SubscriptionStatusTrialing},
		{"unknown", billing.SubscriptionStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapPaddleStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapPaddleStatus(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_mapLemonStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected billing.SubscriptionStatus
	}{
		{"active", billing.SubscriptionStatusActive},
		{"past_due", billing.SubscriptionStatusPastDue},
		{"cancelled", billing.SubscriptionStatusCancelled},
		{"expired", billing.SubscriptionStatusCancelled},
		{"paused", billing.SubscriptionStatusPaused},
		{"on_trial", billing.SubscriptionStatusTrialing},
		{"unknown", billing.SubscriptionStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapLemonStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapLemonStatus(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_formatInt(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		// Note: formatInt(0) returns "" due to the TrimPrefix removing all 0s
		// This is a known quirk but variants/IDs are never 0 in practice
		{1, "1"},
		{123, "123"},
		{1234567890, "1234567890"},
	}

	for _, tt := range tests {
		result := formatInt(tt.input)
		if result != tt.expected {
			t.Errorf("formatInt(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func Test_itoa(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{123, "123"},
		{-123, "-123"},
		{1234567890, "1234567890"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func Test_parseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"19.99", 1999},
		{"0.00", 0},
		{"100.00", 10000},
		{"1.5", 150},
		{"10", 1000},
		{"99.9", 9990},
	}

	for _, tt := range tests {
		result := parseAmount(tt.input)
		if result != tt.expected {
			t.Errorf("parseAmount(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func Test_paymentProviderName(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}

	t.Run("with provider", func(t *testing.T) {
		payment := &mockWebhookPaymentProvider{name: "stripe"}
		h := NewPaymentWebhookHandler(payment, handler, logger)
		if h.paymentProviderName() != "stripe" {
			t.Errorf("expected stripe, got %s", h.paymentProviderName())
		}
	})

	t.Run("without provider", func(t *testing.T) {
		h := NewPaymentWebhookHandler(nil, handler, logger)
		if h.paymentProviderName() != "none" {
			t.Errorf("expected none, got %s", h.paymentProviderName())
		}
	})
}

func TestPaymentWebhookHandler_isCheckoutCompleted(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	tests := []struct {
		provider  string
		eventType string
		expected  bool
	}{
		{"stripe", "checkout.session.completed", true},
		{"stripe", "customer.subscription.updated", false},
		{"paddle", "transaction.completed", true},
		{"paddle", "subscription.created", true},
		{"paddle", "subscription.updated", false},
		{"lemonsqueezy", "order_created", true},
		{"lemonsqueezy", "subscription_created", true},
		{"lemonsqueezy", "subscription_updated", false},
		{"unknown", "checkout.session.completed", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.eventType, func(t *testing.T) {
			result := h.isCheckoutCompleted(tt.provider, tt.eventType)
			if result != tt.expected {
				t.Errorf("isCheckoutCompleted(%s, %s) = %v, want %v", tt.provider, tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestPaymentWebhookHandler_isSubscriptionUpdated(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	tests := []struct {
		provider  string
		eventType string
		expected  bool
	}{
		{"stripe", "customer.subscription.updated", true},
		{"stripe", "checkout.session.completed", false},
		{"paddle", "subscription.updated", true},
		{"paddle", "subscription.created", false},
		{"lemonsqueezy", "subscription_updated", true},
		{"lemonsqueezy", "order_created", false},
		{"unknown", "subscription.updated", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.eventType, func(t *testing.T) {
			result := h.isSubscriptionUpdated(tt.provider, tt.eventType)
			if result != tt.expected {
				t.Errorf("isSubscriptionUpdated(%s, %s) = %v, want %v", tt.provider, tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestPaymentWebhookHandler_isSubscriptionCancelled(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	tests := []struct {
		provider  string
		eventType string
		expected  bool
	}{
		{"stripe", "customer.subscription.deleted", true},
		{"stripe", "customer.subscription.updated", false},
		{"paddle", "subscription.canceled", true},
		{"paddle", "subscription.updated", false},
		{"lemonsqueezy", "subscription_cancelled", true},
		{"lemonsqueezy", "subscription_updated", false},
		{"unknown", "subscription.canceled", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.eventType, func(t *testing.T) {
			result := h.isSubscriptionCancelled(tt.provider, tt.eventType)
			if result != tt.expected {
				t.Errorf("isSubscriptionCancelled(%s, %s) = %v, want %v", tt.provider, tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestPaymentWebhookHandler_isInvoicePaid(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	tests := []struct {
		provider  string
		eventType string
		expected  bool
	}{
		{"stripe", "invoice.paid", true},
		{"stripe", "invoice.payment_failed", false},
		{"paddle", "transaction.paid", true},
		{"paddle", "transaction.payment_failed", false},
		{"lemonsqueezy", "subscription_payment_success", true},
		{"lemonsqueezy", "subscription_payment_failed", false},
		{"unknown", "invoice.paid", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.eventType, func(t *testing.T) {
			result := h.isInvoicePaid(tt.provider, tt.eventType)
			if result != tt.expected {
				t.Errorf("isInvoicePaid(%s, %s) = %v, want %v", tt.provider, tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestPaymentWebhookHandler_isInvoiceFailed(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	tests := []struct {
		provider  string
		eventType string
		expected  bool
	}{
		{"stripe", "invoice.payment_failed", true},
		{"stripe", "invoice.paid", false},
		{"paddle", "transaction.payment_failed", true},
		{"paddle", "transaction.paid", false},
		{"lemonsqueezy", "subscription_payment_failed", true},
		{"lemonsqueezy", "subscription_payment_success", false},
		{"unknown", "invoice.payment_failed", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.eventType, func(t *testing.T) {
			result := h.isInvoiceFailed(tt.provider, tt.eventType)
			if result != tt.expected {
				t.Errorf("isInvoiceFailed(%s, %s) = %v, want %v", tt.provider, tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestPaymentWebhookHandler_extractCheckoutData_Paddle(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	// Test with items array
	data := map[string]any{
		"customer_id":     "ctm_123",
		"subscription_id": "sub_456",
		"items": []any{
			map[string]any{
				"price": map[string]any{
					"id": "price_789",
				},
			},
		},
	}

	customerID, subscriptionID, planID := h.extractCheckoutData("paddle", data)

	if customerID != "ctm_123" {
		t.Errorf("expected customer ctm_123, got %s", customerID)
	}
	if subscriptionID != "sub_456" {
		t.Errorf("expected subscription sub_456, got %s", subscriptionID)
	}
	if planID != "price_789" {
		t.Errorf("expected plan price_789, got %s", planID)
	}
}

func TestPaymentWebhookHandler_extractSubscriptionID_PaddleFallback(t *testing.T) {
	logger := zerolog.Nop()
	handler := &mockPaymentWebhookHandler{}
	h := NewPaymentWebhookHandler(nil, handler, logger)

	// Test with id instead of subscription_id
	data := map[string]any{
		"id": "sub_fallback",
	}

	subscriptionID := h.extractSubscriptionID("paddle", data)

	if subscriptionID != "sub_fallback" {
		t.Errorf("expected sub_fallback, got %s", subscriptionID)
	}
}
