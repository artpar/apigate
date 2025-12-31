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
	"time"

	"github.com/artpar/apigate/domain/billing"
)

// PaddleConfig holds Paddle configuration.
type PaddleConfig struct {
	VendorID      string
	APIKey        string
	PublicKey     string
	WebhookSecret string
	Sandbox       bool
}

// PaddleProvider implements ports.PaymentProvider for Paddle.
type PaddleProvider struct {
	config     PaddleConfig
	httpClient *http.Client
	baseURL    string
}

// NewPaddleProvider creates a new Paddle payment provider.
func NewPaddleProvider(config PaddleConfig) *PaddleProvider {
	baseURL := "https://vendors.paddle.com/api/2.0"
	if config.Sandbox {
		baseURL = "https://sandbox-vendors.paddle.com/api/2.0"
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

// CreateCustomer creates a customer in Paddle (not directly supported, handled via subscriptions).
func (p *PaddleProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	// Paddle doesn't have a separate customer creation API
	// Customers are created implicitly when they subscribe
	// Return the userID as the "customer ID" for our tracking
	return userID, nil
}

// CreateCheckoutSession creates a Paddle checkout link.
func (p *PaddleProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string, trialDays int) (string, error) {
	// Paddle uses client-side checkout with Paddle.js
	// This would generate a Pay Link via their API
	payload := map[string]interface{}{
		"vendor_id":       p.config.VendorID,
		"vendor_auth_code": p.config.APIKey,
		"product_id":      priceID,
		"customer_email":  customerID, // In practice, this would be the email
		"passthrough":     fmt.Sprintf(`{"user_id": "%s"}`, customerID),
		"return_url":      successURL,
	}

	// Add trial period if specified
	if trialDays > 0 {
		payload["trial_days"] = trialDays
	}

	resp, err := p.doRequest(ctx, "/product/generate_pay_link", payload)
	if err != nil {
		return "", err
	}

	if url, ok := resp["url"].(string); ok {
		return url, nil
	}
	return "", errors.New("failed to generate Paddle pay link")
}

// CreatePortalSession creates a customer portal link (Paddle subscription management).
func (p *PaddleProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	// Paddle uses update_url from webhook data for subscription management
	// This is typically stored when subscription is created
	return "", errors.New("Paddle uses subscription-specific update URLs, not a general portal")
}

// CancelSubscription cancels a Paddle subscription.
func (p *PaddleProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	payload := map[string]interface{}{
		"vendor_id":       p.config.VendorID,
		"vendor_auth_code": p.config.APIKey,
		"subscription_id": subscriptionID,
	}

	_, err := p.doRequest(ctx, "/subscription/users_cancel", payload)
	return err
}

// GetSubscription retrieves subscription details from Paddle.
func (p *PaddleProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	payload := map[string]interface{}{
		"vendor_id":       p.config.VendorID,
		"vendor_auth_code": p.config.APIKey,
		"subscription_id": subscriptionID,
	}

	resp, err := p.doRequest(ctx, "/subscription/users", payload)
	if err != nil {
		return billing.Subscription{}, err
	}

	// Parse response - Paddle returns an array
	users, ok := resp["response"].([]interface{})
	if !ok || len(users) == 0 {
		return billing.Subscription{}, errors.New("subscription not found")
	}

	user := users[0].(map[string]interface{})
	return billing.Subscription{
		ID:     subscriptionID,
		Status: mapPaddleStatus(user["state"].(string)),
	}, nil
}

// ReportUsage reports metered usage (Paddle handles this differently via modifiers).
func (p *PaddleProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	// Paddle uses subscription modifiers for usage-based billing
	// This would add a one-time charge modifier
	return errors.New("Paddle usage reporting requires subscription modifiers")
}

// ParseWebhook parses and validates a Paddle webhook.
func (p *PaddleProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	// Verify signature
	mac := hmac.New(sha256.New, []byte(p.config.WebhookSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", nil, errors.New("invalid webhook signature")
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", nil, err
	}

	eventType, _ := data["alert_name"].(string)
	return eventType, data, nil
}

func (p *PaddleProvider) doRequest(ctx context.Context, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return nil, fmt.Errorf("Paddle API error: %v", result["error"])
	}

	return result, nil
}

func mapPaddleStatus(state string) billing.SubscriptionStatus {
	switch state {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "deleted", "cancelled":
		return billing.SubscriptionStatusCancelled
	case "paused":
		return billing.SubscriptionStatusPaused
	case "trialing":
		return billing.SubscriptionStatusTrialing
	default:
		return billing.SubscriptionStatusActive
	}
}
