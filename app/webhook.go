// Package app contains the WebhookService for dispatching webhook events.
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// WebhookService dispatches webhook events to registered endpoints.
type WebhookService struct {
	webhooks    ports.WebhookStore
	deliveries  ports.DeliveryStore
	logger      zerolog.Logger
	client      *http.Client
	mu          sync.Mutex
	stopCh      chan struct{}
	running     bool
	shutdownCtx context.Context    // For graceful shutdown of spawned goroutines
	shutdownFn  context.CancelFunc // Cancel function for shutdown
}

// NewWebhookService creates a new webhook service.
func NewWebhookService(
	webhooks ports.WebhookStore,
	deliveries ports.DeliveryStore,
	logger zerolog.Logger,
) *WebhookService {
	// Create shutdown context for graceful termination of goroutines
	shutdownCtx, shutdownFn := context.WithCancel(context.Background())

	return &WebhookService{
		webhooks:    webhooks,
		deliveries:  deliveries,
		logger:      logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopCh:      make(chan struct{}),
		shutdownCtx: shutdownCtx,
		shutdownFn:  shutdownFn,
	}
}

// Dispatch dispatches an event to all subscribed webhooks.
// This is the main entry point for sending webhook events.
func (s *WebhookService) Dispatch(ctx context.Context, event webhook.Event) error {
	// Find all webhooks subscribed to this event
	webhooks, err := s.webhooks.ListForEvent(ctx, event.Type)
	if err != nil {
		s.logger.Error().Err(err).
			Str("event_type", string(event.Type)).
			Msg("failed to list webhooks for event")
		return err
	}

	if len(webhooks) == 0 {
		s.logger.Debug().
			Str("event_type", string(event.Type)).
			Msg("no webhooks subscribed to event")
		return nil
	}

	// Build the payload
	payload, err := webhook.BuildPayload(event)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to build webhook payload")
		return err
	}

	payloadBytes, err := webhook.SerializePayload(payload)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to serialize webhook payload")
		return err
	}

	payloadStr := string(payloadBytes)
	now := time.Now()

	// Create a delivery for each webhook and dispatch
	for _, wh := range webhooks {
		delivery := webhook.NewDelivery(wh, event, payloadStr, now)

		// Store the delivery
		if err := s.deliveries.Create(ctx, delivery); err != nil {
			s.logger.Error().Err(err).
				Str("webhook_id", wh.ID).
				Str("event_id", event.ID).
				Msg("failed to create delivery record")
			continue
		}

		// Dispatch asynchronously with timeout derived from shutdown context
		// This prevents goroutine leaks - goroutines will be cancelled on shutdown
		webhookCtx, cancel := context.WithTimeout(s.shutdownCtx, 30*time.Second)
		go func(ctx context.Context, cancelFn context.CancelFunc) {
			defer cancelFn()
			s.sendWebhook(ctx, wh, delivery, payloadBytes)
		}(webhookCtx, cancel)
	}

	s.logger.Info().
		Str("event_type", string(event.Type)).
		Str("event_id", event.ID).
		Int("webhook_count", len(webhooks)).
		Msg("webhook event dispatched")

	return nil
}

// sendWebhook sends a webhook and updates the delivery status.
func (s *WebhookService) sendWebhook(ctx context.Context, wh webhook.Webhook, delivery webhook.Delivery, payload []byte) {
	start := time.Now()

	// Apply webhook-specific timeout via context (not new client)
	// This prevents HTTP client leaks from creating a new client per request
	if wh.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(wh.TimeoutMS)*time.Millisecond)
		defer cancel()
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payload))
	if err != nil {
		s.markFailed(ctx, delivery, 0, "", err.Error(), 0)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "APIGate-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", wh.ID)
	req.Header.Set("X-Event-ID", delivery.EventID)
	req.Header.Set("X-Event-Type", string(delivery.EventType))

	// Sign the payload
	signature := webhook.SignPayload(payload, wh.Secret)
	req.Header.Set("X-Webhook-Signature", signature)

	// Send the request using shared client
	resp, err := s.client.Do(req)
	durationMS := int(time.Since(start).Milliseconds())

	if err != nil {
		s.logger.Warn().Err(err).
			Str("webhook_id", wh.ID).
			Str("url", wh.URL).
			Msg("webhook request failed")
		s.markFailed(ctx, delivery, 0, "", err.Error(), durationMS)
		return
	}
	defer resp.Body.Close()

	// Read response body (limited to prevent memory issues)
	var respBody bytes.Buffer
	respBody.Grow(1024)
	respBody.ReadFrom(resp.Body)
	bodyStr := respBody.String()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.markSuccess(ctx, delivery, resp.StatusCode, bodyStr, durationMS)
	} else {
		s.markFailed(ctx, delivery, resp.StatusCode, bodyStr, "", durationMS)
	}
}

// markSuccess marks a delivery as successful.
func (s *WebhookService) markSuccess(ctx context.Context, d webhook.Delivery, statusCode int, respBody string, durationMS int) {
	updated := webhook.MarkSuccess(d, statusCode, respBody, durationMS, time.Now())
	if err := s.deliveries.Update(ctx, updated); err != nil {
		s.logger.Error().Err(err).
			Str("delivery_id", d.ID).
			Msg("failed to update delivery status")
	}

	s.logger.Debug().
		Str("delivery_id", d.ID).
		Str("webhook_id", d.WebhookID).
		Int("status_code", statusCode).
		Int("duration_ms", durationMS).
		Msg("webhook delivered successfully")
}

// markFailed marks a delivery as failed and schedules retry if needed.
func (s *WebhookService) markFailed(ctx context.Context, d webhook.Delivery, statusCode int, respBody, errMsg string, durationMS int) {
	updated := webhook.MarkFailed(d, statusCode, respBody, errMsg, durationMS, time.Now())
	if err := s.deliveries.Update(ctx, updated); err != nil {
		s.logger.Error().Err(err).
			Str("delivery_id", d.ID).
			Msg("failed to update delivery status")
	}

	if updated.Status == webhook.DeliveryRetrying {
		s.logger.Info().
			Str("delivery_id", d.ID).
			Str("webhook_id", d.WebhookID).
			Int("attempt", d.Attempt).
			Time("next_retry", *updated.NextRetry).
			Msg("webhook delivery scheduled for retry")
	} else {
		s.logger.Warn().
			Str("delivery_id", d.ID).
			Str("webhook_id", d.WebhookID).
			Int("status_code", statusCode).
			Str("error", errMsg).
			Msg("webhook delivery failed permanently")
	}
}

// StartRetryWorker starts a background worker that processes pending retries.
func (s *WebhookService) StartRetryWorker(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Dur("interval", interval).Msg("starting webhook retry worker")

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.processRetries(ctx)
			}
		}
	}()
}

// StopRetryWorker stops the retry worker and cancels pending webhook goroutines.
func (s *WebhookService) StopRetryWorker() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}

	// Cancel shutdown context to terminate any pending webhook goroutines
	if s.shutdownFn != nil {
		s.shutdownFn()
	}
}

// processRetries processes pending webhook retries.
func (s *WebhookService) processRetries(ctx context.Context) {
	now := time.Now()
	pending, err := s.deliveries.ListPending(ctx, now, 100)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list pending deliveries")
		return
	}

	if len(pending) == 0 {
		return
	}

	s.logger.Debug().Int("count", len(pending)).Msg("processing pending webhook deliveries")

	for _, d := range pending {
		// Get the webhook
		wh, err := s.webhooks.Get(ctx, d.WebhookID)
		if err != nil {
			s.logger.Error().Err(err).
				Str("webhook_id", d.WebhookID).
				Msg("failed to get webhook for retry")
			continue
		}

		if !wh.Enabled {
			s.logger.Debug().
				Str("webhook_id", d.WebhookID).
				Msg("skipping retry for disabled webhook")
			continue
		}

		// Increment attempt and update
		updated := webhook.IncrementAttempt(d, now)
		if err := s.deliveries.Update(ctx, updated); err != nil {
			s.logger.Error().Err(err).Msg("failed to increment attempt")
			continue
		}

		// Re-dispatch with timeout derived from shutdown context
		retryCtx, cancel := context.WithTimeout(s.shutdownCtx, 30*time.Second)
		go func(ctx context.Context, cancelFn context.CancelFunc) {
			defer cancelFn()
			s.sendWebhook(ctx, wh, updated, []byte(d.Payload))
		}(retryCtx, cancel)
	}
}

// DispatchEvent is a convenience method for creating and dispatching events.
func (s *WebhookService) DispatchEvent(ctx context.Context, eventType webhook.EventType, userID string, data map[string]interface{}) error {
	event := webhook.Event{
		ID:        webhook.GenerateEventID(),
		Type:      eventType,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      data,
	}
	return s.Dispatch(ctx, event)
}

// TestWebhook sends a test event to a specific webhook.
func (s *WebhookService) TestWebhook(ctx context.Context, webhookID string) error {
	wh, err := s.webhooks.Get(ctx, webhookID)
	if err != nil {
		return err
	}

	event := webhook.Event{
		ID:        webhook.GenerateEventID(),
		Type:      webhook.EventTest,
		UserID:    wh.UserID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message":    "This is a test webhook event",
			"webhook_id": webhookID,
		},
	}

	payload, err := webhook.BuildPayload(event)
	if err != nil {
		return err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	delivery := webhook.NewDelivery(wh, event, string(payloadBytes), time.Now())
	if err := s.deliveries.Create(ctx, delivery); err != nil {
		return err
	}

	// Dispatch with timeout derived from shutdown context
	testCtx, cancel := context.WithTimeout(s.shutdownCtx, 30*time.Second)
	go func() {
		defer cancel()
		s.sendWebhook(testCtx, wh, delivery, payloadBytes)
	}()
	return nil
}
