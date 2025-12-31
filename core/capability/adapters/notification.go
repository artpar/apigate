package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// Console Notification Provider (for development/testing)
// =============================================================================

// ConsoleNotification logs notifications to console.
// Suitable for development and testing.
type ConsoleNotification struct {
	name     string
	mu       sync.Mutex
	messages []capability.NotificationMessage
}

// NewConsoleNotification creates a console notification provider.
func NewConsoleNotification(name string) *ConsoleNotification {
	return &ConsoleNotification{
		name:     name,
		messages: make([]capability.NotificationMessage, 0),
	}
}

func (n *ConsoleNotification) Name() string {
	return n.name
}

func (n *ConsoleNotification) Send(ctx context.Context, msg capability.NotificationMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.messages = append(n.messages, msg)
	// In production, this would log or print
	return nil
}

func (n *ConsoleNotification) SendBatch(ctx context.Context, msgs []capability.NotificationMessage) error {
	for _, msg := range msgs {
		if err := n.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (n *ConsoleNotification) TestConnection(ctx context.Context) error {
	return nil // Console always works
}

// GetMessages returns all captured messages (for testing).
func (n *ConsoleNotification) GetMessages() []capability.NotificationMessage {
	n.mu.Lock()
	defer n.mu.Unlock()

	result := make([]capability.NotificationMessage, len(n.messages))
	copy(result, n.messages)
	return result
}

// ClearMessages clears all captured messages (for testing).
func (n *ConsoleNotification) ClearMessages() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messages = make([]capability.NotificationMessage, 0)
}

// Ensure ConsoleNotification implements capability.NotificationProvider
var _ capability.NotificationProvider = (*ConsoleNotification)(nil)

// =============================================================================
// Webhook Notification Provider
// =============================================================================

// WebhookNotification sends notifications via HTTP webhook.
type WebhookNotification struct {
	name    string
	url     string
	headers map[string]string
	client  *http.Client
}

// WebhookConfig configures a webhook notification provider.
type WebhookConfig struct {
	Name    string
	URL     string
	Headers map[string]string
}

// NewWebhookNotification creates a webhook notification provider.
func NewWebhookNotification(cfg WebhookConfig) *WebhookNotification {
	return &WebhookNotification{
		name:    cfg.Name,
		url:     cfg.URL,
		headers: cfg.Headers,
		client:  &http.Client{},
	}
}

func (n *WebhookNotification) Name() string {
	return n.name
}

func (n *WebhookNotification) Send(ctx context.Context, msg capability.NotificationMessage) error {
	payload := map[string]any{
		"channel":  msg.Channel,
		"title":    msg.Title,
		"message":  msg.Message,
		"severity": msg.Severity,
		"fields":   msg.Fields,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotification) SendBatch(ctx context.Context, msgs []capability.NotificationMessage) error {
	for _, msg := range msgs {
		if err := n.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (n *WebhookNotification) TestConnection(ctx context.Context) error {
	testMsg := capability.NotificationMessage{
		Title:    "Test Connection",
		Message:  "This is a test notification",
		Severity: "info",
	}
	return n.Send(ctx, testMsg)
}

// Ensure WebhookNotification implements capability.NotificationProvider
var _ capability.NotificationProvider = (*WebhookNotification)(nil)
