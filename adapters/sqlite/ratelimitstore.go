package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/ports"
)

// RateLimitStore implements ports.RateLimitStore using SQLite.
type RateLimitStore struct {
	db *DB
}

// NewRateLimitStore creates a new SQLite rate limit store.
func NewRateLimitStore(db *DB) *RateLimitStore {
	return &RateLimitStore{db: db}
}

// Get retrieves current rate limit state for a key.
func (s *RateLimitStore) Get(ctx context.Context, keyID string) (ratelimit.WindowState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT count, window_end, burst_used
		FROM rate_limit_state
		WHERE key_id = ?
	`, keyID)

	var state ratelimit.WindowState
	err := row.Scan(&state.Count, &state.WindowEnd, &state.BurstUsed)
	if errors.Is(err, sql.ErrNoRows) {
		// Return empty state if not found (key has no rate limit history)
		return ratelimit.WindowState{}, nil
	}
	if err != nil {
		return ratelimit.WindowState{}, err
	}
	return state, nil
}

// Set updates rate limit state for a key.
func (s *RateLimitStore) Set(ctx context.Context, keyID string, state ratelimit.WindowState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rate_limit_state (key_id, count, window_end, burst_used)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key_id) DO UPDATE SET
			count = excluded.count,
			window_end = excluded.window_end,
			burst_used = excluded.burst_used
	`, keyID, state.Count, state.WindowEnd, state.BurstUsed)
	return err
}

// Delete removes rate limit state for a key.
func (s *RateLimitStore) Delete(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rate_limit_state WHERE key_id = ?`, keyID)
	return err
}

// Cleanup removes expired rate limit entries.
func (s *RateLimitStore) Cleanup(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM rate_limit_state
		WHERE window_end < datetime('now', '-1 hour')
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Ensure interface compliance.
var _ ports.RateLimitStore = (*RateLimitStore)(nil)
