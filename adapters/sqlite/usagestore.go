package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// UsageStore implements ports.UsageStore using SQLite.
type UsageStore struct {
	db *DB
}

// NewUsageStore creates a new SQLite usage store.
func NewUsageStore(db *DB) *UsageStore {
	return &UsageStore{db: db}
}

// RecordBatch stores multiple usage events.
func (s *UsageStore) RecordBatch(ctx context.Context, events []usage.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO usage_events (
			id, key_id, user_id, method, path, status_code, latency_ms,
			request_bytes, response_bytes, cost_multiplier, ip_address, user_agent, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		// Store timestamp in UTC for consistent querying
		_, err := stmt.ExecContext(ctx,
			e.ID, e.KeyID, e.UserID, e.Method, e.Path, e.StatusCode, e.LatencyMs,
			e.RequestBytes, e.ResponseBytes, e.CostMultiplier, e.IPAddress, e.UserAgent, e.Timestamp.UTC(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSummary returns aggregated usage for a period.
func (s *UsageStore) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	// Format times as ISO8601 strings for SQLite comparison
	// Convert to UTC since timestamps are stored in UTC
	startStr := start.UTC().Format("2006-01-02 15:04:05")
	endStr := end.UTC().Format("2006-01-02 15:04:05")
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as request_count,
			COALESCE(SUM(cost_multiplier), 0) as compute_units,
			COALESCE(SUM(request_bytes), 0) as bytes_in,
			COALESCE(SUM(response_bytes), 0) as bytes_out,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as error_count,
			CAST(COALESCE(AVG(latency_ms), 0) AS INTEGER) as avg_latency
		FROM usage_events
		WHERE user_id = ? AND datetime(timestamp) >= datetime(?) AND datetime(timestamp) < datetime(?)
	`, userID, startStr, endStr)

	var summary usage.Summary
	summary.UserID = userID
	summary.PeriodStart = start
	summary.PeriodEnd = end

	err := row.Scan(
		&summary.RequestCount,
		&summary.ComputeUnits,
		&summary.BytesIn,
		&summary.BytesOut,
		&summary.ErrorCount,
		&summary.AvgLatencyMs,
	)
	if err != nil {
		return usage.Summary{}, err
	}

	return summary, nil
}

// GetHistory returns usage summaries for past periods.
func (s *UsageStore) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	// Get monthly summaries for the past N months
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			strftime('%Y-%m-01', timestamp) as period_start,
			COUNT(*) as request_count,
			COALESCE(SUM(cost_multiplier), 0) as compute_units,
			COALESCE(SUM(request_bytes), 0) as bytes_in,
			COALESCE(SUM(response_bytes), 0) as bytes_out,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as error_count,
			CAST(COALESCE(AVG(latency_ms), 0) AS INTEGER) as avg_latency
		FROM usage_events
		WHERE user_id = ?
		GROUP BY strftime('%Y-%m', timestamp)
		ORDER BY period_start DESC
		LIMIT ?
	`, userID, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []usage.Summary
	for rows.Next() {
		var summary usage.Summary
		var periodStart string
		summary.UserID = userID

		err := rows.Scan(
			&periodStart,
			&summary.RequestCount,
			&summary.ComputeUnits,
			&summary.BytesIn,
			&summary.BytesOut,
			&summary.ErrorCount,
			&summary.AvgLatencyMs,
		)
		if err != nil {
			return nil, err
		}

		// Parse period start and calculate period end
		summary.PeriodStart, _ = time.Parse("2006-01-02", periodStart)
		summary.PeriodEnd = summary.PeriodStart.AddDate(0, 1, 0)

		summaries = append(summaries, summary)
	}

	return summaries, rows.Err()
}

// GetRecentRequests returns recent request logs.
func (s *UsageStore) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key_id, user_id, method, path, status_code, latency_ms,
		       request_bytes, response_bytes, cost_multiplier, ip_address, user_agent, timestamp
		FROM usage_events
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []usage.Event
	for rows.Next() {
		var e usage.Event
		var ipAddress, userAgent sql.NullString

		err := rows.Scan(
			&e.ID, &e.KeyID, &e.UserID, &e.Method, &e.Path, &e.StatusCode, &e.LatencyMs,
			&e.RequestBytes, &e.ResponseBytes, &e.CostMultiplier, &ipAddress, &userAgent, &e.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if ipAddress.Valid {
			e.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			e.UserAgent = userAgent.String
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

// SaveSummary persists a pre-aggregated summary.
func (s *UsageStore) SaveSummary(ctx context.Context, summary usage.Summary) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO usage_summaries (
			user_id, period_start, period_end, request_count, compute_units,
			bytes_in, bytes_out, error_count, avg_latency_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, period_start) DO UPDATE SET
			request_count = request_count + excluded.request_count,
			compute_units = compute_units + excluded.compute_units,
			bytes_in = bytes_in + excluded.bytes_in,
			bytes_out = bytes_out + excluded.bytes_out,
			error_count = error_count + excluded.error_count,
			avg_latency_ms = (avg_latency_ms + excluded.avg_latency_ms) / 2
	`, summary.UserID, summary.PeriodStart, summary.PeriodEnd,
		summary.RequestCount, summary.ComputeUnits, summary.BytesIn,
		summary.BytesOut, summary.ErrorCount, summary.AvgLatencyMs)
	return err
}

// Cleanup removes old usage events.
func (s *UsageStore) Cleanup(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM usage_events WHERE timestamp < ?
	`, olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Ensure interface compliance.
var _ ports.UsageStore = (*UsageStore)(nil)
