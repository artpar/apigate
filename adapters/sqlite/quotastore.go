package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// QuotaStore implements ports.QuotaStore using SQLite for persistence.
// This ensures quota state survives server restarts.
type QuotaStore struct {
	db *DB
}

// NewQuotaStore creates a new SQLite quota store.
func NewQuotaStore(db *DB) *QuotaStore {
	return &QuotaStore{db: db}
}

// Get retrieves current quota state for a user's billing period.
func (s *QuotaStore) Get(ctx context.Context, userID string, periodStart time.Time) (ports.QuotaState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT user_id, period_start, request_count, compute_units, bytes_used, last_updated
		FROM quota_state
		WHERE user_id = ? AND period_start = ?
	`, userID, periodStart)

	var state ports.QuotaState
	err := row.Scan(
		&state.UserID,
		&state.PeriodStart,
		&state.RequestCount,
		&state.ComputeUnits,
		&state.BytesUsed,
		&state.LastUpdated,
	)
	if err == sql.ErrNoRows {
		// Return empty state for new period
		return ports.QuotaState{
			UserID:      userID,
			PeriodStart: periodStart,
		}, nil
	}
	if err != nil {
		return ports.QuotaState{}, err
	}

	return state, nil
}

// Increment atomically adds to quota counters, returns new state.
func (s *QuotaStore) Increment(ctx context.Context, userID string, periodStart time.Time,
	requests int64, computeUnits float64, bytes int64) (ports.QuotaState, error) {

	now := time.Now().UTC()

	// Use INSERT OR REPLACE with atomic increment via subquery
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO quota_state (user_id, period_start, request_count, compute_units, bytes_used, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, period_start) DO UPDATE SET
			request_count = request_count + excluded.request_count,
			compute_units = compute_units + excluded.compute_units,
			bytes_used = bytes_used + excluded.bytes_used,
			last_updated = excluded.last_updated
	`, userID, periodStart, requests, computeUnits, bytes, now)
	if err != nil {
		return ports.QuotaState{}, err
	}

	// Read back the updated state
	return s.Get(ctx, userID, periodStart)
}

// Sync reconciles quota state from usage store (background job).
func (s *QuotaStore) Sync(ctx context.Context, userID string, periodStart time.Time, summary usage.Summary) error {
	now := time.Now().UTC()

	// Replace quota state with values from usage summary
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO quota_state (user_id, period_start, request_count, compute_units, bytes_used, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, period_start) DO UPDATE SET
			request_count = excluded.request_count,
			compute_units = excluded.compute_units,
			bytes_used = excluded.bytes_used,
			last_updated = excluded.last_updated
	`, userID, periodStart, summary.RequestCount, summary.ComputeUnits, summary.BytesIn+summary.BytesOut, now)

	return err
}

// CleanupOldPeriods removes quota states for periods older than the given cutoff.
// This should be called periodically to prevent unbounded table growth.
func (s *QuotaStore) CleanupOldPeriods(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM quota_state WHERE period_start < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Ensure interface compliance.
var _ ports.QuotaStore = (*QuotaStore)(nil)
