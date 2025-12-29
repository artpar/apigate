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

// LemonSqueezyConfig holds LemonSqueezy configuration.
type LemonSqueezyConfig struct {
	APIKey        string
	StoreID       string
	WebhookSecret string
}

// LemonSqueezyProvider implements ports.PaymentProvider for LemonSqueezy.
type LemonSqueezyProvider struct {
	config     LemonSqueezyConfig
	httpClient *http.Client
	baseURL    string
}

// NewLemonSqueezyProvider creates a new LemonSqueezy payment provider.
func NewLemonSqueezyProvider(config LemonSqueezyConfig) *LemonSqueezyProvider {
	return &LemonSqueezyProvider{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.lemonsqueezy.com/v1",
	}
}

// Name returns the provider name.
func (p *LemonSqueezyProvider) Name() string {
	return "lemonsqueezy"
}

// CreateCustomer creates a customer in LemonSqueezy.
func (p *LemonSqueezyProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "customers",
			"attributes": map[string]interface{}{
				"name":     name,
				"email":    email,
				"store_id": p.config.StoreID,
			},
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/customers", payload)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response format")
	}

	if id, ok := data["id"].(string); ok {
		return id, nil
	}
	return "", errors.New("failed to create customer")
}

// CreateCheckoutSession creates a LemonSqueezy checkout.
func (p *LemonSqueezyProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"checkout_data": map[string]interface{}{
					"custom": map[string]string{
						"user_id": customerID,
					},
				},
				"product_options": map[string]interface{}{
					"redirect_url": successURL,
				},
			},
			"relationships": map[string]interface{}{
				"store": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "stores",
						"id":   p.config.StoreID,
					},
				},
				"variant": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "variants",
						"id":   priceID,
					},
				},
			},
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/checkouts", payload)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response format")
	}

	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid response format")
	}

	if url, ok := attrs["url"].(string); ok {
		return url, nil
	}
	return "", errors.New("failed to create checkout")
}

// CreatePortalSession creates a customer portal session.
func (p *LemonSqueezyProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	// LemonSqueezy provides customer portal URLs in subscription data
	// Fetch subscription and return the update URL
	return "", errors.New("LemonSqueezy uses subscription-specific portal URLs")
}

// CancelSubscription cancels a subscription.
func (p *LemonSqueezyProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	_, err := p.doRequest(ctx, "DELETE", "/subscriptions/"+subscriptionID, nil)
	return err
}

// GetSubscription retrieves subscription details.
func (p *LemonSqueezyProvider) GetSubscription(ctx context.Context, subscriptionID string) (billing.Subscription, error) {
	resp, err := p.doRequest(ctx, "GET", "/subscriptions/"+subscriptionID, nil)
	if err != nil {
		return billing.Subscription{}, err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return billing.Subscription{}, errors.New("invalid response format")
	}

	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		return billing.Subscription{}, errors.New("invalid response format")
	}

	sub := billing.Subscription{
		ID:     subscriptionID,
		Status: mapLemonStatus(attrs["status"].(string)),
	}

	if endsAt, ok := attrs["ends_at"].(string); ok && endsAt != "" {
		if t, err := time.Parse(time.RFC3339, endsAt); err == nil {
			sub.CurrentPeriodEnd = t
		}
	}

	return sub, nil
}

// ReportUsage reports metered usage.
func (p *LemonSqueezyProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "usage-records",
			"attributes": map[string]interface{}{
				"quantity": quantity,
			},
			"relationships": map[string]interface{}{
				"subscription-item": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "subscription-items",
						"id":   subscriptionItemID,
					},
				},
			},
		},
	}

	_, err := p.doRequest(ctx, "POST", "/usage-records", payload)
	return err
}

// ParseWebhook parses and validates a LemonSqueezy webhook.
func (p *LemonSqueezyProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
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

	meta, ok := data["meta"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("invalid webhook format")
	}

	eventType, _ := meta["event_name"].(string)
	return eventType, data, nil
}

func (p *LemonSqueezyProvider) doRequest(ctx context.Context, method, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LemonSqueezy API error: %s", string(respBody))
	}

	if resp.StatusCode == 204 {
		return nil, nil // No content
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func mapLemonStatus(status string) billing.SubscriptionStatus {
	switch status {
	case "active":
		return billing.SubscriptionStatusActive
	case "past_due":
		return billing.SubscriptionStatusPastDue
	case "cancelled", "expired":
		return billing.SubscriptionStatusCancelled
	case "paused":
		return billing.SubscriptionStatusPaused
	case "on_trial":
		return billing.SubscriptionStatusTrialing
	case "unpaid":
		return billing.SubscriptionStatusUnpaid
	default:
		return billing.SubscriptionStatusActive
	}
}
