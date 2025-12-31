package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func createTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func createTestStore(t *testing.T) (*SQLiteStore, *sql.DB) {
	t.Helper()
	db := createTestDB(t)
	cfg := SQLiteConfig{
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}
	store, err := NewSQLiteStore(db, cfg)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	return store, db
}

func TestNewSQLiteStore(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	cfg := SQLiteConfig{
		BatchSize:     50,
		FlushInterval: 500 * time.Millisecond,
		BufferSize:    500,
	}

	store, err := NewSQLiteStore(db, cfg)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	if store.batchSize != 50 {
		t.Errorf("batchSize = %v, want %v", store.batchSize, 50)
	}
	if store.flushInterval != 500*time.Millisecond {
		t.Errorf("flushInterval = %v, want %v", store.flushInterval, 500*time.Millisecond)
	}
}

func TestNewSQLiteStore_Defaults(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	cfg := SQLiteConfig{} // All zero values

	store, err := NewSQLiteStore(db, cfg)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	if store.batchSize != 100 {
		t.Errorf("batchSize = %v, want default %v", store.batchSize, 100)
	}
	if store.flushInterval != time.Second {
		t.Errorf("flushInterval = %v, want default %v", store.flushInterval, time.Second)
	}
}

func TestDefaultSQLiteConfig(t *testing.T) {
	cfg := DefaultSQLiteConfig()

	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %v, want %v", cfg.BatchSize, 100)
	}
	if cfg.FlushInterval != time.Second {
		t.Errorf("FlushInterval = %v, want %v", cfg.FlushInterval, time.Second)
	}
	if cfg.BufferSize != 10000 {
		t.Errorf("BufferSize = %v, want %v", cfg.BufferSize, 10000)
	}
	if cfg.CostCalculator == nil {
		t.Error("CostCalculator should not be nil")
	}
}

func TestSQLiteStore_Write(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			ID:            "test-1",
			Timestamp:     time.Now(),
			Channel:       "http",
			Module:        "user",
			Action:        "get",
			DurationNS:    1000000,
			MemoryBytes:   2048,
			RequestBytes:  512,
			ResponseBytes: 1024,
			Success:       true,
			StatusCode:    200,
		},
		{
			ID:            "test-2",
			Timestamp:     time.Now(),
			Channel:       "http",
			Module:        "user",
			Action:        "list",
			DurationNS:    2000000,
			Success:       false,
			StatusCode:    500,
			Error:         "internal error",
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify data was written
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM analytics").Scan(&count)
	if err != nil {
		t.Fatalf("Query count error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count = %v, want %v", count, 2)
	}
}

func TestSQLiteStore_Write_EmptyEvents(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	err := store.Write(ctx, []Event{})
	if err != nil {
		t.Fatalf("Write() with empty events should not error, got = %v", err)
	}
}

func TestSQLiteStore_Write_AutoGenerateID(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			DurationNS: 1000000,
			Success:    true,
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	var id string
	err = db.QueryRow("SELECT id FROM analytics LIMIT 1").Scan(&id)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if id == "" {
		t.Error("ID should be auto-generated")
	}
}

func TestSQLiteStore_Write_AutoGenerateTimestamp(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			ID:         "test-1",
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			DurationNS: 1000000,
			Success:    true,
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	var ts string
	err = db.QueryRow("SELECT timestamp FROM analytics LIMIT 1").Scan(&ts)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if ts == "" {
		t.Error("Timestamp should be auto-generated")
	}
}

func TestSQLiteStore_Record(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	event := Event{
		ID:         "record-test-1",
		Channel:    "http",
		Module:     "user",
		Action:     "get",
		Timestamp:  time.Now(),
		DurationNS: 1000000,
		Success:    true,
	}

	store.Record(event)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM analytics WHERE id = 'record-test-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %v, want %v", count, 1)
	}
}

func TestSQLiteStore_RecordAsync(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	event := Event{
		ID:         "async-test-1",
		Channel:    "http",
		Module:     "user",
		Action:     "get",
		Timestamp:  time.Now(),
		DurationNS: 1000000,
		Success:    true,
	}

	store.RecordAsync(event)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM analytics WHERE id = 'async-test-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %v, want %v", count, 1)
	}
}

func TestSQLiteStore_Flush(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()

	// Write directly to ensure data is there
	events := []Event{
		{
			ID:         "flush-test-1",
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			Timestamp:  time.Now(),
			DurationNS: 1000000,
			Success:    true,
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Now test flush on buffered event
	event := Event{
		ID:         "flush-test-2",
		Channel:    "http",
		Module:     "user",
		Action:     "get",
		Timestamp:  time.Now(),
		DurationNS: 1000000,
		Success:    true,
	}

	store.Record(event)

	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM analytics").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count < 1 {
		t.Errorf("Count = %v, want at least %v", count, 1)
	}
}

func TestSQLiteStore_Flush_Empty(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	err := store.Flush(ctx)
	if err != nil {
		t.Fatalf("Flush() with no events should not error, got = %v", err)
	}
}

func TestSQLiteStore_Query_Basic(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			ID:         "query-1",
			Timestamp:  time.Now(),
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			DurationNS: 1000000,
			Success:    true,
			StatusCode: 200,
		},
		{
			ID:         "query-2",
			Timestamp:  time.Now(),
			Channel:    "http",
			Module:     "user",
			Action:     "list",
			DurationNS: 2000000,
			Success:    true,
			StatusCode: 200,
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total = %v, want %v", total, 2)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %v, want %v", len(result), 2)
	}
}

func TestSQLiteStore_Query_FilterByChannel(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "ch-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "ch-2", Timestamp: time.Now(), Channel: "grpc", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "ch-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{Channel: "http"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total = %v, want %v", total, 2)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %v, want %v", len(result), 2)
	}
}

func TestSQLiteStore_Query_FilterByModule(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "mod-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "mod-2", Timestamp: time.Now(), Channel: "http", Module: "route", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{Module: "user"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}

func TestSQLiteStore_Query_FilterByAction(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "act-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "act-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
		{ID: "act-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{Action: "get"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total = %v, want %v", total, 2)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %v, want %v", len(result), 2)
	}
}

func TestSQLiteStore_Query_FilterByUserID(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "usr-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true, UserID: "user-1"},
		{ID: "usr-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true, UserID: "user-2"},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{UserID: "user-1"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}

func TestSQLiteStore_Query_FilterByAPIKeyID(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "key-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true, APIKeyID: "api-key-1"},
		{ID: "key-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true, APIKeyID: "api-key-2"},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{APIKeyID: "api-key-1"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}

func TestSQLiteStore_Query_FilterBySuccess(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "suc-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "suc-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: false, Error: "failed"},
		{ID: "suc-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Filter success=true
	successTrue := true
	result, total, err := store.Query(ctx, QueryOptions{Success: &successTrue})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total = %v, want %v", total, 2)
	}

	// Filter success=false
	successFalse := false
	result, total, err = store.Query(ctx, QueryOptions{Success: &successFalse})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}

func TestSQLiteStore_Query_FilterByTimeRange(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "time-1", Timestamp: now.Add(-2 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "time-2", Timestamp: now.Add(-1 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "time-3", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Query with start time
	result, total, err := store.Query(ctx, QueryOptions{
		Start: now.Add(-90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total = %v, want %v", total, 2)
	}

	// Query with end time
	result, total, err = store.Query(ctx, QueryOptions{
		End: now.Add(-90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}

func TestSQLiteStore_Query_Pagination(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := make([]Event, 10)
	for i := 0; i < 10; i++ {
		events[i] = Event{
			ID:         "page-" + string(rune('0'+i)),
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			DurationNS: 1000000,
			Success:    true,
		}
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// First page
	result, total, err := store.Query(ctx, QueryOptions{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 10 {
		t.Errorf("total = %v, want %v", total, 10)
	}
	if len(result) != 3 {
		t.Errorf("len(result) = %v, want %v", len(result), 3)
	}

	// Second page
	result, total, err = store.Query(ctx, QueryOptions{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("len(result) = %v, want %v", len(result), 3)
	}
}

func TestSQLiteStore_Query_OrderBy(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "order-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 3000000, Success: true},
		{ID: "order-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "order-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 2000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Order by duration ascending
	result, _, err := store.Query(ctx, QueryOptions{OrderBy: "duration_ns", OrderDesc: false})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result[0].DurationNS != 1000000 {
		t.Errorf("First result DurationNS = %v, want %v", result[0].DurationNS, 1000000)
	}

	// Order by duration descending
	result, _, err = store.Query(ctx, QueryOptions{OrderBy: "duration_ns", OrderDesc: true})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result[0].DurationNS != 3000000 {
		t.Errorf("First result DurationNS = %v, want %v", result[0].DurationNS, 3000000)
	}
}

func TestSQLiteStore_Query_InvalidOrderBy(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "inv-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Invalid order by should fall back to timestamp
	_, _, err = store.Query(ctx, QueryOptions{OrderBy: "invalid_column"})
	if err != nil {
		t.Fatalf("Query() with invalid OrderBy should not error, got = %v", err)
	}
}

func TestSQLiteStore_Aggregate_Basic(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "agg-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, MemoryBytes: 1024, RequestBytes: 512, ResponseBytes: 1024, Success: true},
		{ID: "agg-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 2000000, MemoryBytes: 2048, RequestBytes: 1024, ResponseBytes: 2048, Success: true},
		{ID: "agg-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 3000000, MemoryBytes: 1024, RequestBytes: 512, ResponseBytes: 1024, Success: false},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %v, want %v", len(summaries), 1)
	}

	sum := summaries[0]
	if sum.TotalRequests != 3 {
		t.Errorf("TotalRequests = %v, want %v", sum.TotalRequests, 3)
	}
	if sum.SuccessRequests != 2 {
		t.Errorf("SuccessRequests = %v, want %v", sum.SuccessRequests, 2)
	}
	if sum.ErrorRequests != 1 {
		t.Errorf("ErrorRequests = %v, want %v", sum.ErrorRequests, 1)
	}
	if sum.AvgDurationNS != 2000000 {
		t.Errorf("AvgDurationNS = %v, want %v", sum.AvgDurationNS, 2000000)
	}
	if sum.MinDurationNS != 1000000 {
		t.Errorf("MinDurationNS = %v, want %v", sum.MinDurationNS, 1000000)
	}
	if sum.MaxDurationNS != 3000000 {
		t.Errorf("MaxDurationNS = %v, want %v", sum.MaxDurationNS, 3000000)
	}
}

func TestSQLiteStore_Aggregate_GroupByModule(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "grp-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "grp-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
		{ID: "grp-3", Timestamp: time.Now(), Channel: "http", Module: "route", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{GroupBy: []string{"module"}})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %v, want %v", len(summaries), 2)
	}
}

func TestSQLiteStore_Aggregate_GroupByAction(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "act-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "act-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
		{ID: "act-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{GroupBy: []string{"action"}})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %v, want %v", len(summaries), 2)
	}
}

func TestSQLiteStore_Aggregate_GroupByChannel(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "ch-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "ch-2", Timestamp: time.Now(), Channel: "grpc", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "ch-3", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{GroupBy: []string{"channel"}})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %v, want %v", len(summaries), 2)
	}
}

func TestSQLiteStore_Aggregate_MultipleGroupBy(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "multi-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "multi-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
		{ID: "multi-3", Timestamp: time.Now(), Channel: "http", Module: "route", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "multi-4", Timestamp: time.Now(), Channel: "grpc", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{GroupBy: []string{"module", "action"}})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 3 {
		t.Errorf("len(summaries) = %v, want %v", len(summaries), 3)
	}
}

func TestSQLiteStore_Aggregate_WithPeriod(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "period-1", Timestamp: now.Add(-2 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "period-2", Timestamp: now.Add(-1 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "period-3", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Group by hour
	summaries, err := store.Aggregate(ctx, AggregateOptions{Period: "hour"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) < 1 {
		t.Errorf("len(summaries) = %v, want at least 1", len(summaries))
	}
}

func TestSQLiteStore_Aggregate_WithFilters(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "filter-1", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "filter-2", Timestamp: now, Channel: "grpc", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "filter-3", Timestamp: now, Channel: "http", Module: "route", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{Channel: "http"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %v, want %v", len(summaries), 1)
	}
	if summaries[0].TotalRequests != 2 {
		t.Errorf("TotalRequests = %v, want %v", summaries[0].TotalRequests, 2)
	}
}

func TestSQLiteStore_Aggregate_WithTimeRange(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "range-1", Timestamp: now.Add(-2 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "range-2", Timestamp: now.Add(-1 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "range-3", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{
		Start: now.Add(-90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %v, want %v", len(summaries), 1)
	}
	if summaries[0].TotalRequests != 2 {
		t.Errorf("TotalRequests = %v, want %v", summaries[0].TotalRequests, 2)
	}
}

func TestSQLiteStore_Aggregate_PeriodMinute(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "min-1", Timestamp: now.Add(-2 * time.Minute), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "min-2", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{Period: "minute"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) < 1 {
		t.Errorf("len(summaries) = %v, want at least 1", len(summaries))
	}
}

func TestSQLiteStore_Aggregate_PeriodDay(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "day-1", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "day-2", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{Period: "day"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("len(summaries) = %v, want %v", len(summaries), 1)
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "del-1", Timestamp: now.Add(-2 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "del-2", Timestamp: now.Add(-1 * time.Hour), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "del-3", Timestamp: now, Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Delete events older than 90 minutes
	deleted, err := store.Delete(ctx, now.Add(-90*time.Minute))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %v, want %v", deleted, 1)
	}

	// Verify remaining events
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM analytics").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 2 {
		t.Errorf("Remaining count = %v, want %v", count, 2)
	}
}

func TestSQLiteStore_Close(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()

	// Record some events
	event := Event{
		ID:         "close-test-1",
		Channel:    "http",
		Module:     "user",
		Action:     "get",
		Timestamp:  time.Now(),
		DurationNS: 1000000,
		Success:    true,
	}
	store.Record(event)

	// Close should flush remaining events
	err := store.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify event was flushed
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM analytics WHERE id = 'close-test-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %v, want %v", count, 1)
	}
}

func TestSQLiteStore_BatchFlush(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	cfg := SQLiteConfig{
		BatchSize:     5,
		FlushInterval: 10 * time.Second, // Long interval so batch size triggers flush
		BufferSize:    100,
	}

	store, err := NewSQLiteStore(db, cfg)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	// Record enough events to trigger batch flush
	for i := 0; i < 6; i++ {
		store.Record(Event{
			ID:         "batch-" + string(rune('0'+i)),
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			Timestamp:  time.Now(),
			DurationNS: 1000000,
			Success:    true,
		})
	}

	// Wait a bit for batch to flush
	time.Sleep(100 * time.Millisecond)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM analytics").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	// At least 5 should be flushed (batch size)
	if count < 5 {
		t.Errorf("Count = %v, want at least %v", count, 5)
	}
}

func TestSQLiteStore_Query_AllFiltersNullableFields(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			ID:            "null-1",
			Timestamp:     time.Now(),
			Channel:       "http",
			Module:        "user",
			Action:        "get",
			RecordID:      "rec-1",
			UserID:        "user-1",
			APIKeyID:      "key-1",
			RemoteIP:      "192.168.1.1",
			DurationNS:    1000000,
			MemoryBytes:   2048,
			RequestBytes:  512,
			ResponseBytes: 1024,
			Success:       true,
			StatusCode:    200,
			Error:         "",
		},
		{
			ID:         "null-2",
			Timestamp:  time.Now(),
			Channel:    "http",
			Module:     "user",
			Action:     "get",
			DurationNS: 1000000,
			Success:    false,
			StatusCode: 500,
			Error:      "internal error",
		},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, _, err := store.Query(ctx, QueryOptions{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("len(result) = %v, want %v", len(result), 2)
	}

	// Verify nullable fields are correctly read
	for _, e := range result {
		if e.ID == "null-1" {
			if e.RecordID != "rec-1" {
				t.Errorf("RecordID = %v, want %v", e.RecordID, "rec-1")
			}
			if e.UserID != "user-1" {
				t.Errorf("UserID = %v, want %v", e.UserID, "user-1")
			}
			if e.APIKeyID != "key-1" {
				t.Errorf("APIKeyID = %v, want %v", e.APIKeyID, "key-1")
			}
			if e.RemoteIP != "192.168.1.1" {
				t.Errorf("RemoteIP = %v, want %v", e.RemoteIP, "192.168.1.1")
			}
		}
		if e.ID == "null-2" {
			if e.Error != "internal error" {
				t.Errorf("Error = %v, want %v", e.Error, "internal error")
			}
		}
	}
}

func TestSQLiteStore_Aggregate_ModuleFilter(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "mf-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "mf-2", Timestamp: time.Now(), Channel: "http", Module: "route", Action: "get", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{Module: "user"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %v, want %v", len(summaries), 1)
	}
	if summaries[0].TotalRequests != 1 {
		t.Errorf("TotalRequests = %v, want %v", summaries[0].TotalRequests, 1)
	}
}

func TestSQLiteStore_Aggregate_ActionFilter(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{ID: "af-1", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "get", DurationNS: 1000000, Success: true},
		{ID: "af-2", Timestamp: time.Now(), Channel: "http", Module: "user", Action: "list", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	summaries, err := store.Aggregate(ctx, AggregateOptions{Action: "get"})
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %v, want %v", len(summaries), 1)
	}
	if summaries[0].TotalRequests != 1 {
		t.Errorf("TotalRequests = %v, want %v", summaries[0].TotalRequests, 1)
	}
}

func TestSQLiteStore_CustomCostCalculator(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	customCalc := &DefaultCostCalculator{
		BaseRequestCost:   0.1,
		CPUCostPerUS:      0.001,
		MemoryCostPerKB:   0.0001,
		TransferCostPerKB: 0.001,
	}

	cfg := SQLiteConfig{
		BatchSize:      10,
		FlushInterval:  100 * time.Millisecond,
		BufferSize:     100,
		CostCalculator: customCalc,
	}

	store, err := NewSQLiteStore(db, cfg)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	events := []Event{
		{
			ID:            "cost-1",
			Timestamp:     time.Now(),
			Channel:       "http",
			Module:        "user",
			Action:        "get",
			DurationNS:    1000000, // 1ms = 1000us
			MemoryBytes:   1024,    // 1KB
			RequestBytes:  512,
			ResponseBytes: 512,
			Success:       true,
		},
	}

	err = store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	var cost float64
	err = db.QueryRow("SELECT cost_units FROM analytics WHERE id = 'cost-1'").Scan(&cost)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}

	// Expected: 0.1 (base) + 1000*0.001 (cpu) + 1*0.0001 (memory) + 1*0.001 (transfer)
	expected := 0.1 + 1.0 + 0.0001 + 0.001
	if cost != expected {
		t.Errorf("cost = %v, want %v", cost, expected)
	}
}

func TestSQLiteStore_Query_CombinedFilters(t *testing.T) {
	store, db := createTestStore(t)
	defer db.Close()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	events := []Event{
		{ID: "cf-1", Timestamp: now, Channel: "http", Module: "user", Action: "get", UserID: "user-1", DurationNS: 1000000, Success: true},
		{ID: "cf-2", Timestamp: now, Channel: "http", Module: "user", Action: "list", UserID: "user-1", DurationNS: 1000000, Success: true},
		{ID: "cf-3", Timestamp: now, Channel: "grpc", Module: "user", Action: "get", UserID: "user-1", DurationNS: 1000000, Success: true},
		{ID: "cf-4", Timestamp: now, Channel: "http", Module: "route", Action: "get", UserID: "user-1", DurationNS: 1000000, Success: true},
		{ID: "cf-5", Timestamp: now, Channel: "http", Module: "user", Action: "get", UserID: "user-2", DurationNS: 1000000, Success: true},
	}

	err := store.Write(ctx, events)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	result, total, err := store.Query(ctx, QueryOptions{
		Channel: "http",
		Module:  "user",
		Action:  "get",
		UserID:  "user-1",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %v, want %v", total, 1)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %v, want %v", len(result), 1)
	}
}
