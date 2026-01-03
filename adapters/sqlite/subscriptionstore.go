package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
)

// SubscriptionStore implements ports.SubscriptionStore using SQLite.
type SubscriptionStore struct {
	db *DB
}

// NewSubscriptionStore creates a new SQLite subscription store.
func NewSubscriptionStore(db *DB) *SubscriptionStore {
	return &SubscriptionStore{db: db}
}

// Get retrieves a subscription by ID.
func (s *SubscriptionStore) Get(ctx context.Context, id string) (billing.Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, plan_id, provider, provider_id, provider_item_id,
		       status, current_period_start, current_period_end,
		       cancel_at_period_end, cancelled_at, created_at, updated_at
		FROM subscriptions
		WHERE id = ?
	`, id)
	return scanSubscription(row)
}

// GetByUser retrieves the active subscription for a user.
func (s *SubscriptionStore) GetByUser(ctx context.Context, userID string) (billing.Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, plan_id, provider, provider_id, provider_item_id,
		       status, current_period_start, current_period_end,
		       cancel_at_period_end, cancelled_at, created_at, updated_at
		FROM subscriptions
		WHERE user_id = ? AND status IN ('active', 'trialing', 'past_due')
		ORDER BY created_at DESC
		LIMIT 1
	`, userID)
	return scanSubscription(row)
}

// Create stores a new subscription.
func (s *SubscriptionStore) Create(ctx context.Context, sub billing.Subscription) error {
	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	if sub.UpdatedAt.IsZero() {
		sub.UpdatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, user_id, plan_id, provider, provider_id, provider_item_id,
			status, current_period_start, current_period_end,
			cancel_at_period_end, cancelled_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sub.ID, sub.UserID, sub.PlanID, sub.Provider, sub.ProviderID, sub.ProviderItemID,
		string(sub.Status), sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		boolToInt(sub.CancelAtPeriodEnd), nullTime(sub.CancelledAt),
		sub.CreatedAt, sub.UpdatedAt,
	)

	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

// Update modifies a subscription.
func (s *SubscriptionStore) Update(ctx context.Context, sub billing.Subscription) error {
	sub.UpdatedAt = time.Now().UTC()

	result, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET plan_id = ?, provider = ?, provider_id = ?, provider_item_id = ?,
		    status = ?, current_period_start = ?, current_period_end = ?,
		    cancel_at_period_end = ?, cancelled_at = ?, updated_at = ?
		WHERE id = ?
	`,
		sub.PlanID, sub.Provider, sub.ProviderID, sub.ProviderItemID,
		string(sub.Status), sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		boolToInt(sub.CancelAtPeriodEnd), nullTime(sub.CancelledAt),
		sub.UpdatedAt, sub.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSubscription(row *sql.Row) (billing.Subscription, error) {
	var sub billing.Subscription
	var status string
	var providerID, providerItemID sql.NullString
	var cancelledAt sql.NullTime
	var cancelAtPeriodEnd int

	err := row.Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.Provider, &providerID, &providerItemID,
		&status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&cancelAtPeriodEnd, &cancelledAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return billing.Subscription{}, ErrNotFound
	}
	if err != nil {
		return billing.Subscription{}, err
	}

	sub.Status = billing.SubscriptionStatus(status)
	sub.CancelAtPeriodEnd = cancelAtPeriodEnd == 1
	if providerID.Valid {
		sub.ProviderID = providerID.String
	}
	if providerItemID.Valid {
		sub.ProviderItemID = providerItemID.String
	}
	if cancelledAt.Valid {
		sub.CancelledAt = &cancelledAt.Time
	}

	return sub, nil
}

// Ensure interface compliance.
var _ ports.SubscriptionStore = (*SubscriptionStore)(nil)
