package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/artpar/apigate/ports"
)

// webhookStore implements ports.WebhookStore using SQLite.
type webhookStore struct {
	db *sql.DB
}

// NewWebhookStore creates a new SQLite webhook store.
func NewWebhookStore(db *sql.DB) ports.WebhookStore {
	return &webhookStore{db: db}
}

func (s *webhookStore) List(ctx context.Context) ([]webhook.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, description, url, secret, events,
		       retry_count, timeout_ms, enabled, created_at, updated_at
		FROM webhooks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *webhookStore) ListByUser(ctx context.Context, userID string) ([]webhook.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, description, url, secret, events,
		       retry_count, timeout_ms, enabled, created_at, updated_at
		FROM webhooks
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *webhookStore) ListEnabled(ctx context.Context) ([]webhook.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, description, url, secret, events,
		       retry_count, timeout_ms, enabled, created_at, updated_at
		FROM webhooks
		WHERE enabled = 1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *webhookStore) ListForEvent(ctx context.Context, eventType webhook.EventType) ([]webhook.Webhook, error) {
	// SQLite JSON_EACH to search within the events array
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.user_id, w.name, w.description, w.url, w.secret, w.events,
		       w.retry_count, w.timeout_ms, w.enabled, w.created_at, w.updated_at
		FROM webhooks w
		WHERE w.enabled = 1
		  AND EXISTS (
			SELECT 1 FROM json_each(w.events)
			WHERE json_each.value = ?
		  )
		ORDER BY w.created_at DESC
	`, string(eventType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *webhookStore) Get(ctx context.Context, id string) (webhook.Webhook, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, description, url, secret, events,
		       retry_count, timeout_ms, enabled, created_at, updated_at
		FROM webhooks
		WHERE id = ?
	`, id)

	return s.scanRow(row)
}

func (s *webhookStore) Create(ctx context.Context, w webhook.Webhook) error {
	eventsJSON, err := json.Marshal(w.Events)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, user_id, name, description, url, secret, events,
		                      retry_count, timeout_ms, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.UserID, w.Name, w.Description, w.URL, w.Secret, string(eventsJSON),
		w.RetryCount, w.TimeoutMS, w.Enabled, w.CreatedAt, w.UpdatedAt)

	return err
}

func (s *webhookStore) Update(ctx context.Context, w webhook.Webhook) error {
	eventsJSON, err := json.Marshal(w.Events)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE webhooks
		SET name = ?, description = ?, url = ?, secret = ?, events = ?,
		    retry_count = ?, timeout_ms = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, w.Name, w.Description, w.URL, w.Secret, string(eventsJSON),
		w.RetryCount, w.TimeoutMS, w.Enabled, w.UpdatedAt, w.ID)

	return err
}

func (s *webhookStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, id)
	return err
}

func (s *webhookStore) scanRows(rows *sql.Rows) ([]webhook.Webhook, error) {
	var webhooks []webhook.Webhook
	for rows.Next() {
		w, err := s.scanFromRows(rows)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (s *webhookStore) scanFromRows(rows *sql.Rows) (webhook.Webhook, error) {
	var w webhook.Webhook
	var eventsJSON string
	var description sql.NullString

	err := rows.Scan(
		&w.ID, &w.UserID, &w.Name, &description, &w.URL, &w.Secret, &eventsJSON,
		&w.RetryCount, &w.TimeoutMS, &w.Enabled, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return webhook.Webhook{}, err
	}

	w.Description = description.String

	// Parse events JSON
	if eventsJSON != "" {
		var events []string
		if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
			// Try comma-separated as fallback
			events = strings.Split(eventsJSON, ",")
		}
		for _, e := range events {
			w.Events = append(w.Events, webhook.EventType(strings.TrimSpace(e)))
		}
	}

	return w, nil
}

func (s *webhookStore) scanRow(row *sql.Row) (webhook.Webhook, error) {
	var w webhook.Webhook
	var eventsJSON string
	var description sql.NullString

	err := row.Scan(
		&w.ID, &w.UserID, &w.Name, &description, &w.URL, &w.Secret, &eventsJSON,
		&w.RetryCount, &w.TimeoutMS, &w.Enabled, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return webhook.Webhook{}, err
	}

	w.Description = description.String

	// Parse events JSON
	if eventsJSON != "" {
		var events []string
		if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
			events = strings.Split(eventsJSON, ",")
		}
		for _, e := range events {
			w.Events = append(w.Events, webhook.EventType(strings.TrimSpace(e)))
		}
	}

	return w, nil
}
