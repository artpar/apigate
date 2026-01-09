package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/billing"
)

// PaddleConfig holds Paddle Billing configuration.
type PaddleConfig struct {
	VendorID      string // Not used in Billing API, kept for compatibility
	APIKey        string // Bearer token for Paddle Billing API
	PublicKey     string // Not used in Billing API
	WebhookSecret string // For webhook signature verification
	Sandbox       bool   // Auto-detected from API key prefix
}

// PaddleProvider implements ports.PaymentProvider for Paddle Billing API.
type PaddleProvider struct {
	config     PaddleConfig
	httpClient *http.Client
	baseURL    string
}

// NewPaddleProvider creates a new Paddle Billing payment provider.
func NewPaddleProvider(config PaddleConfig) *PaddleProvider {
	// Auto-detect sandbox mode from API key prefix
	isSandbox := strings.HasPrefix(config.APIKey, "pdl_sdbx_") || config.Sandbox

	baseURL := "https://api.paddle.com"
	if isSandbox {
		baseURL = "https://sandbox-api.paddle.com"
	}

	return &PaddleProvider{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// Name returns the provider name.
func (p *PaddleProvider) Name() string {
	return "paddle"
}

// CreateCustomer creates a customer in Paddle Billing.
func (p *PaddleProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	payload := map[string]interface{}{
		"email": email,
		"name":  name,
		"custom_data": map[string]string{
			"user_id": userID,
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/customers", payload)
	if err != nil {
		return "", fmt.Errorf("create customer: %w", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response from Paddle")
	}

	customerID, ok := data["id"].(string)
	if !ok {
		return "", errors.New("customer ID not found in response")
	}

	return customerID, nil
}

// CreateCheckoutSession creates a Paddle Billing checkout transaction.
func (p *PaddleProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	// First, get customer email if we have a customer ID
	var customerEmail string
	if strings.HasPrefix(customerID, "ctm_") {
		// It's a Paddle customer ID, fetch the customer
		customer, err := p.getCustomer(ctx, customerID)
		if err == nil {
			customerEmail = customer["email"].(string)
		}
	} else {
		// Assume it's an email address
		customerEmail = customerID
	}

	// Create a transaction (checkout) with Paddle Billing API
	payload := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"price_id": priceID,
				"quantity": 1,
			},
		},
		"checkout": map[string]interface{}{
			"url": successURL,
		},
		"custom_data": map[string]string{
			"user_id": customerID,
		},
	}

	// Add customer if we have their ID
	if strings.HasPrefix(customerID, "ctm_") {
		payload["customer_id"] = customerID
	} else if customerEmail != "" {
		// For new customers, we can pass email
		payload["customer"] = map[string]string{
			"email": customerEmail,
		}
	}

	resp, err := p.doRequest(ctx, "POST", "/transactions", payload)
	if err != nil {
		return "", fmt.Errorf("create checkout: %w", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response from Paddle")
	}

	// Get the checkout URL
	checkout, ok := data["checkout"].(map[string]interface{})
	if !ok {
		return "", errors.New("checkout data not found in response")
	}

	checkoutURL, ok := checkout["url"].(string)
	if !ok {
		return "", errors.New("checkout URL not found in response")
	}

	return checkoutURL, nil
}

// CreatePortalSession creates a customer portal link for subscription management.
func (p *PaddleProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	// Paddle Billing uses customer portal sessions
	payload := map[string]interface{}{
		"customer_id": customerID,
	}

	resp, err := p.doRequest(ctx, "POST", "/customer-portal-sessions", payload)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response from Paddle")
	}

	urls, ok := data["urls"].(map[string]interface{})
	if !ok {
		return "", errors.New("portal URLs not found in response")
	}

	// Get the general portal URL
	generalURL, ok := urls["general"].(map[string]interface{})
	if ok {
		if overview, ok := generalURL["overview"].(string); ok {
			return overview, nil
		}
	}

	return "", errors.New("portal URL not found in response")
}

// CancelSubscription cancels a Paddle subscription.
func (p *PaddleProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	endpoint := fmt.Sprintf("/subscriptions/%s/cancel", subscriptionID)

	payload := map[string]interface{}{}
	if immediately {
		payload["effective_from"] = "immediately"
	} else {
		payload["effective_from"] = "next_billing_period"
	}

	_, err := p.doRequest(ctx, "POST", endpoint, payload)
	if err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}

	return nil
}

// GetSubscription retrieves subscription details from Paddle.
func (p *PaddleProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	endpoint := fmt.Sprintf("/subscriptions/%s", subscriptionID)

	resp, err := p.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return billing.Subscription{}, fmt.Errorf("get subscription: %w", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return billing.Subscription{}, errors.New("invalid response from Paddle")
	}

	status := mapPaddleBillingStatus(data["status"].(string))

	sub := billing.Subscription{
		ID:     subscriptionID,
		Status: status,
	}

	// Parse dates if available
	if currentPeriodEnd, ok := data["current_billing_period"].(map[string]interface{}); ok {
		if endStr, ok := currentPeriodEnd["ends_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				sub.CurrentPeriodEnd = t
			}
		}
	}

	return sub, nil
}

// ReportUsage reports metered usage to Paddle.
func (p *PaddleProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	// Paddle Billing doesn't support metered usage in the same way as Stripe
	// Usage-based billing is handled differently
	return errors.New("Paddle Billing usage reporting not implemented")
}

// ParseWebhook parses and validates a Paddle Billing webhook.
func (p *PaddleProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	// Paddle Billing uses a different webhook signature format
	// The signature header contains: ts=timestamp;h1=hash

	if p.config.WebhookSecret != "" {
		// Parse the signature header
		parts := strings.Split(signature, ";")
		var ts, h1 string
		for _, part := range parts {
			if strings.HasPrefix(part, "ts=") {
				ts = strings.TrimPrefix(part, "ts=")
			} else if strings.HasPrefix(part, "h1=") {
				h1 = strings.TrimPrefix(part, "h1=")
			}
		}

		if ts != "" && h1 != "" {
			// Compute expected signature: HMAC-SHA256(ts + ":" + payload, secret)
			mac := hmac.New(sha256.New, []byte(p.config.WebhookSecret))
			mac.Write([]byte(ts + ":"))
			mac.Write(payload)
			expectedSig := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(h1), []byte(expectedSig)) {
				return "", nil, errors.New("invalid webhook signature")
			}
		}
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", nil, err
	}

	// Paddle Billing uses "event_type" instead of "alert_name"
	eventType, _ := data["event_type"].(string)
	return eventType, data, nil
}

func (p *PaddleProvider) getCustomer(ctx context.Context, customerID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/customers/%s", customerID)

	resp, err := p.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid response from Paddle")
	}

	return data, nil
}

func (p *PaddleProvider) doRequest(ctx context.Context, method, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, string(respBody))
	}

	// Paddle Billing API returns error in "error" field
	if errData, ok := result["error"].(map[string]interface{}); ok {
		errType, _ := errData["type"].(string)
		errDetail, _ := errData["detail"].(string)
		return nil, fmt.Errorf("Paddle API error: %s - %s", errType, errDetail)
	}

	// Check HTTP status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Paddle API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return result, nil
}

func mapPaddleBillingStatus(status string) billing.SubscriptionStatus {
	switch status {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "canceled":
		return billing.SubscriptionStatusCancelled
	case "paused":
		return billing.SubscriptionStatusPaused
	case "trialing":
		return billing.SubscriptionStatusTrialing
	default:
		return billing.SubscriptionStatusActive
	}
}
