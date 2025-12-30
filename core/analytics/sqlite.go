package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SQLiteStore implements Analytics with SQLite backend.
type SQLiteStore struct {
	db     *sql.DB
	buffer chan Event
	done   chan struct{}
	wg     sync.WaitGroup

	// Configuration
	batchSize     int
	flushInterval time.Duration
	costCalc      CostCalculator
}

// SQLiteConfig configures the SQLite analytics store.
type SQLiteConfig struct {
	// BatchSize is the number of events to batch before writing.
	BatchSize int

	// FlushInterval is the maximum time between flushes.
	FlushInterval time.Duration

	// BufferSize is the size of the in-memory event buffer.
	BufferSize int

	// CostCalculator computes cost for events.
	CostCalculator CostCalculator
}

// DefaultSQLiteConfig returns sensible defaults.
func DefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		BatchSize:      100,
		FlushInterval:  time.Second,
		BufferSize:     10000,
		CostCalculator: NewDefaultCostCalculator(),
	}
}

// NewSQLiteStore creates a new SQLite-backed analytics store.
func NewSQLiteStore(db *sql.DB, cfg SQLiteConfig) (*SQLiteStore, error) {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 10000
	}
	if cfg.CostCalculator == nil {
		cfg.CostCalculator = NewDefaultCostCalculator()
	}

	s := &SQLiteStore{
		db:            db,
		buffer:        make(chan Event, cfg.BufferSize),
		done:          make(chan struct{}),
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		costCalc:      cfg.CostCalculator,
	}

	if err := s.createTable(); err != nil {
		return nil, err
	}

	// Start background flusher
	s.wg.Add(1)
	go s.flusher()

	return s, nil
}

// createTable creates the analytics table.
func (s *SQLiteStore) createTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS analytics (
			id TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL,
			channel TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			record_id TEXT,
			user_id TEXT,
			api_key_id TEXT,
			remote_ip TEXT,
			duration_ns INTEGER NOT NULL,
			memory_bytes INTEGER DEFAULT 0,
			request_bytes INTEGER DEFAULT 0,
			response_bytes INTEGER DEFAULT 0,
			success INTEGER NOT NULL,
			status_code INTEGER,
			error TEXT,
			cost_units REAL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_timestamp ON analytics(timestamp);
		CREATE INDEX IF NOT EXISTS idx_analytics_module_action ON analytics(module, action);
		CREATE INDEX IF NOT EXISTS idx_analytics_user ON analytics(user_id);
		CREATE INDEX IF NOT EXISTS idx_analytics_api_key ON analytics(api_key_id);
	`)
	return err
}

// Record stores an event (non-blocking).
func (s *SQLiteStore) Record(event Event) {
	select {
	case s.buffer <- event:
	default:
		// Buffer full, drop event (best-effort)
	}
}

// RecordAsync is an alias for Record (already async).
func (s *SQLiteStore) RecordAsync(event Event) {
	s.Record(event)
}

// Flush forces pending events to be written.
func (s *SQLiteStore) Flush(ctx context.Context) error {
	events := s.drain()
	if len(events) == 0 {
		return nil
	}
	return s.Write(ctx, events)
}

// drain collects all pending events from the buffer.
func (s *SQLiteStore) drain() []Event {
	var events []Event
	for {
		select {
		case e := <-s.buffer:
			events = append(events, e)
		default:
			return events
		}
	}
}

// flusher periodically flushes events to storage.
func (s *SQLiteStore) flusher() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	var batch []Event

	for {
		select {
		case <-s.done:
			// Flush remaining
			if len(batch) > 0 {
				s.Write(context.Background(), batch)
			}
			// Drain buffer
			remaining := s.drain()
			if len(remaining) > 0 {
				s.Write(context.Background(), remaining)
			}
			return

		case e := <-s.buffer:
			batch = append(batch, e)
			if len(batch) >= s.batchSize {
				s.Write(context.Background(), batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.Write(context.Background(), batch)
				batch = nil
			}
		}
	}
}

// Write writes events to storage.
func (s *SQLiteStore) Write(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO analytics (
			id, timestamp, channel, module, action, record_id,
			user_id, api_key_id, remote_ip,
			duration_ns, memory_bytes, request_bytes, response_bytes,
			success, status_code, error, cost_units
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now()
		}

		cost := s.costCalc.Calculate(e)

		successInt := 0
		if e.Success {
			successInt = 1
		}

		_, err := stmt.ExecContext(ctx,
			e.ID, e.Timestamp.Format(time.RFC3339Nano),
			e.Channel, e.Module, e.Action, e.RecordID,
			e.UserID, e.APIKeyID, e.RemoteIP,
			e.DurationNS, e.MemoryBytes, e.RequestBytes, e.ResponseBytes,
			successInt, e.StatusCode, e.Error, cost,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query retrieves events matching the options.
func (s *SQLiteStore) Query(ctx context.Context, opts QueryOptions) ([]Event, int64, error) {
	var conditions []string
	var args []any

	if !opts.Start.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, opts.Start.Format(time.RFC3339Nano))
	}
	if !opts.End.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, opts.End.Format(time.RFC3339Nano))
	}
	if opts.Channel != "" {
		conditions = append(conditions, "channel = ?")
		args = append(args, opts.Channel)
	}
	if opts.Module != "" {
		conditions = append(conditions, "module = ?")
		args = append(args, opts.Module)
	}
	if opts.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, opts.Action)
	}
	if opts.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, opts.UserID)
	}
	if opts.APIKeyID != "" {
		conditions = append(conditions, "api_key_id = ?")
		args = append(args, opts.APIKeyID)
	}
	if opts.Success != nil {
		if *opts.Success {
			conditions = append(conditions, "success = 1")
		} else {
			conditions = append(conditions, "success = 0")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM analytics " + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Order - whitelist allowed columns to prevent SQL injection
	allowedOrderCols := map[string]bool{
		"timestamp":      true,
		"duration_ns":    true,
		"memory_bytes":   true,
		"request_bytes":  true,
		"response_bytes": true,
		"module":         true,
		"action":         true,
		"channel":        true,
	}
	orderBy := "timestamp"
	if opts.OrderBy != "" && allowedOrderCols[opts.OrderBy] {
		orderBy = opts.OrderBy
	}
	order := "DESC"
	if !opts.OrderDesc {
		order = "ASC"
	}

	// Limit/offset
	limit := 100
	if opts.Limit > 0 {
		limit = opts.Limit
	}

	query := fmt.Sprintf(`
		SELECT id, timestamp, channel, module, action, record_id,
			user_id, api_key_id, remote_ip,
			duration_ns, memory_bytes, request_bytes, response_bytes,
			success, status_code, error
		FROM analytics %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, where, orderBy, order)

	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var ts string
		var success int
		var recordID, userID, apiKeyID, remoteIP, errMsg sql.NullString
		var statusCode sql.NullInt64

		err := rows.Scan(
			&e.ID, &ts, &e.Channel, &e.Module, &e.Action, &recordID,
			&userID, &apiKeyID, &remoteIP,
			&e.DurationNS, &e.MemoryBytes, &e.RequestBytes, &e.ResponseBytes,
			&success, &statusCode, &errMsg,
		)
		if err != nil {
			return nil, 0, err
		}

		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		e.Success = success == 1
		e.RecordID = recordID.String
		e.UserID = userID.String
		e.APIKeyID = apiKeyID.String
		e.RemoteIP = remoteIP.String
		e.StatusCode = int(statusCode.Int64)
		e.Error = errMsg.String

		events = append(events, e)
	}

	return events, total, rows.Err()
}

// Aggregate returns summarized analytics.
func (s *SQLiteStore) Aggregate(ctx context.Context, opts AggregateOptions) ([]Summary, error) {
	var conditions []string
	var args []any

	if !opts.Start.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, opts.Start.Format(time.RFC3339Nano))
	}
	if !opts.End.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, opts.End.Format(time.RFC3339Nano))
	}
	if opts.Channel != "" {
		conditions = append(conditions, "channel = ?")
		args = append(args, opts.Channel)
	}
	if opts.Module != "" {
		conditions = append(conditions, "module = ?")
		args = append(args, opts.Module)
	}
	if opts.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, opts.Action)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build GROUP BY
	var groupCols []string
	var selectCols []string

	for _, g := range opts.GroupBy {
		switch g {
		case "module":
			groupCols = append(groupCols, "module")
			selectCols = append(selectCols, "module")
		case "action":
			groupCols = append(groupCols, "action")
			selectCols = append(selectCols, "action")
		case "channel":
			groupCols = append(groupCols, "channel")
			selectCols = append(selectCols, "channel")
		}
	}

	// Time period grouping
	periodExpr := ""
	switch opts.Period {
	case "minute":
		periodExpr = "strftime('%Y-%m-%d %H:%M', timestamp)"
	case "hour":
		periodExpr = "strftime('%Y-%m-%d %H', timestamp)"
	case "day":
		periodExpr = "strftime('%Y-%m-%d', timestamp)"
	}

	if periodExpr != "" {
		groupCols = append(groupCols, periodExpr)
		selectCols = append(selectCols, periodExpr+" as period")
	}

	groupBy := ""
	if len(groupCols) > 0 {
		groupBy = "GROUP BY " + strings.Join(groupCols, ", ")
	}

	selectPart := strings.Join(selectCols, ", ")
	if selectPart != "" {
		selectPart += ","
	}

	query := fmt.Sprintf(`
		SELECT %s
			COUNT(*) as total_requests,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_requests,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as error_requests,
			CAST(COALESCE(AVG(duration_ns), 0) AS INTEGER) as avg_duration_ns,
			MIN(duration_ns) as min_duration_ns,
			MAX(duration_ns) as max_duration_ns,
			SUM(memory_bytes) as total_memory_bytes,
			SUM(request_bytes) as total_request_bytes,
			SUM(response_bytes) as total_response_bytes,
			SUM(cost_units) as cost_units,
			MIN(timestamp) as start_time,
			MAX(timestamp) as end_time
		FROM analytics %s %s
		ORDER BY start_time DESC
	`, selectPart, where, groupBy)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var sum Summary
		var startStr, endStr string
		var module, action, channel, period sql.NullString

		// Build scan targets based on groupBy
		scanTargets := make([]any, 0)
		for _, g := range opts.GroupBy {
			switch g {
			case "module":
				scanTargets = append(scanTargets, &module)
			case "action":
				scanTargets = append(scanTargets, &action)
			case "channel":
				scanTargets = append(scanTargets, &channel)
			}
		}
		if opts.Period != "" {
			scanTargets = append(scanTargets, &period)
		}

		scanTargets = append(scanTargets,
			&sum.TotalRequests, &sum.SuccessRequests, &sum.ErrorRequests,
			&sum.AvgDurationNS, &sum.MinDurationNS, &sum.MaxDurationNS,
			&sum.TotalMemoryBytes, &sum.TotalRequestBytes, &sum.TotalResponseBytes,
			&sum.CostUnits, &startStr, &endStr,
		)

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}

		sum.Channel = channel.String
		sum.Module = module.String
		sum.Action = action.String
		sum.Period = period.String
		sum.Start, _ = time.Parse(time.RFC3339Nano, startStr)
		sum.End, _ = time.Parse(time.RFC3339Nano, endStr)

		summaries = append(summaries, sum)
	}

	return summaries, rows.Err()
}

// Delete removes events older than the given time.
func (s *SQLiteStore) Delete(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM analytics WHERE timestamp < ?",
		before.Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Close shuts down the store.
func (s *SQLiteStore) Close() error {
	close(s.done)
	s.wg.Wait()
	return nil
}
