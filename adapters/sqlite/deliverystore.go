package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/artpar/apigate/ports"
)

// deliveryStore implements ports.DeliveryStore using SQLite.
type deliveryStore struct {
	db *sql.DB
}

// NewDeliveryStore creates a new SQLite delivery store.
func NewDeliveryStore(db *sql.DB) ports.DeliveryStore {
	return &deliveryStore{db: db}
}

func (s *deliveryStore) List(ctx context.Context, webhookID string, limit int) ([]webhook.Delivery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event_id, event_type, payload, status,
		       attempt, max_attempts, status_code, response_body, error,
		       duration_ms, next_retry, created_at, updated_at
		FROM webhook_deliveries
		WHERE webhook_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *deliveryStore) ListPending(ctx context.Context, before time.Time, limit int) ([]webhook.Delivery, error) {
	// List pending deliveries and retryable deliveries whose next_retry time has passed
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event_id, event_type, payload, status,
		       attempt, max_attempts, status_code, response_body, error,
		       duration_ms, next_retry, created_at, updated_at
		FROM webhook_deliveries
		WHERE (status = 'pending') OR (status = 'retrying' AND next_retry <= ?)
		ORDER BY created_at ASC
		LIMIT ?
	`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *deliveryStore) Get(ctx context.Context, id string) (webhook.Delivery, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, webhook_id, event_id, event_type, payload, status,
		       attempt, max_attempts, status_code, response_body, error,
		       duration_ms, next_retry, created_at, updated_at
		FROM webhook_deliveries
		WHERE id = ?
	`, id)

	return s.scanRow(row)
}

func (s *deliveryStore) Create(ctx context.Context, d webhook.Delivery) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event_id, event_type, payload, status,
		                                attempt, max_attempts, status_code, response_body, error,
		                                duration_ms, next_retry, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.WebhookID, d.EventID, string(d.EventType), d.Payload, string(d.Status),
		d.Attempt, d.MaxAttempts, d.StatusCode, d.ResponseBody, d.Error,
		d.DurationMS, d.NextRetry, d.CreatedAt, d.UpdatedAt)

	return err
}

func (s *deliveryStore) Update(ctx context.Context, d webhook.Delivery) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status = ?, attempt = ?, status_code = ?, response_body = ?,
		    error = ?, duration_ms = ?, next_retry = ?, updated_at = ?
		WHERE id = ?
	`, string(d.Status), d.Attempt, d.StatusCode, d.ResponseBody,
		d.Error, d.DurationMS, d.NextRetry, d.UpdatedAt, d.ID)

	return err
}

func (s *deliveryStore) scanRows(rows *sql.Rows) ([]webhook.Delivery, error) {
	var deliveries []webhook.Delivery
	for rows.Next() {
		d, err := s.scanFromRows(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}

func (s *deliveryStore) scanFromRows(rows *sql.Rows) (webhook.Delivery, error) {
	var d webhook.Delivery
	var eventType string
	var status string
	var payload sql.NullString
	var responseBody sql.NullString
	var errorMsg sql.NullString
	var nextRetry sql.NullTime

	err := rows.Scan(
		&d.ID, &d.WebhookID, &d.EventID, &eventType, &payload, &status,
		&d.Attempt, &d.MaxAttempts, &d.StatusCode, &responseBody, &errorMsg,
		&d.DurationMS, &nextRetry, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return webhook.Delivery{}, err
	}

	d.EventType = webhook.EventType(eventType)
	d.Status = webhook.DeliveryStatus(status)
	d.Payload = payload.String
	d.ResponseBody = responseBody.String
	d.Error = errorMsg.String
	if nextRetry.Valid {
		d.NextRetry = &nextRetry.Time
	}

	return d, nil
}

func (s *deliveryStore) scanRow(row *sql.Row) (webhook.Delivery, error) {
	var d webhook.Delivery
	var eventType string
	var status string
	var payload sql.NullString
	var responseBody sql.NullString
	var errorMsg sql.NullString
	var nextRetry sql.NullTime

	err := row.Scan(
		&d.ID, &d.WebhookID, &d.EventID, &eventType, &payload, &status,
		&d.Attempt, &d.MaxAttempts, &d.StatusCode, &responseBody, &errorMsg,
		&d.DurationMS, &nextRetry, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return webhook.Delivery{}, err
	}

	d.EventType = webhook.EventType(eventType)
	d.Status = webhook.DeliveryStatus(status)
	d.Payload = payload.String
	d.ResponseBody = responseBody.String
	d.Error = errorMsg.String
	if nextRetry.Valid {
		d.NextRetry = &nextRetry.Time
	}

	return d, nil
}
