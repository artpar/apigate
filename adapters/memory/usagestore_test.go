package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/usage"
)

func TestUsageStore_NewUsageStore(t *testing.T) {
	store := memory.NewUsageStore()
	if store == nil {
		t.Fatal("NewUsageStore returned nil")
	}

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("new store should be empty, got %d events", len(all))
	}
}

func TestUsageStore_RecordBatch(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1", Method: "GET", Path: "/api/test", Timestamp: time.Now()},
		{ID: "e2", UserID: "user1", Method: "POST", Path: "/api/test", Timestamp: time.Now()},
		{ID: "e3", UserID: "user1", Method: "PUT", Path: "/api/test", Timestamp: time.Now()},
	}

	err := store.RecordBatch(ctx, events)
	if err != nil {
		t.Fatalf("RecordBatch failed: %v", err)
	}

	all := store.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 events, got %d", len(all))
	}
}

func TestUsageStore_RecordBatch_Empty(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	err := store.RecordBatch(ctx, []usage.Event{})
	if err != nil {
		t.Fatalf("RecordBatch with empty slice failed: %v", err)
	}

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 events, got %d", len(all))
	}
}

func TestUsageStore_RecordBatch_MultipleTimes(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	batch1 := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user1"},
	}
	batch2 := []usage.Event{
		{ID: "e3", UserID: "user1"},
		{ID: "e4", UserID: "user1"},
	}

	store.RecordBatch(ctx, batch1)
	store.RecordBatch(ctx, batch2)

	all := store.GetAll()
	if len(all) != 4 {
		t.Errorf("expected 4 events, got %d", len(all))
	}
}

func TestUsageStore_GetSummary(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	events := []usage.Event{
		{ID: "e1", UserID: "user1", StatusCode: 200, RequestBytes: 100, ResponseBytes: 200, LatencyMs: 50, CostMultiplier: 1.0, Timestamp: now},
		{ID: "e2", UserID: "user1", StatusCode: 200, RequestBytes: 150, ResponseBytes: 250, LatencyMs: 60, CostMultiplier: 1.5, Timestamp: now},
		{ID: "e3", UserID: "user1", StatusCode: 400, RequestBytes: 50, ResponseBytes: 100, LatencyMs: 30, CostMultiplier: 1.0, Timestamp: now},
	}

	store.RecordBatch(ctx, events)

	summary, err := store.GetSummary(ctx, "user1", start, end)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3", summary.RequestCount)
	}
	if summary.BytesIn != 300 {
		t.Errorf("BytesIn = %d, want 300", summary.BytesIn)
	}
	if summary.BytesOut != 550 {
		t.Errorf("BytesOut = %d, want 550", summary.BytesOut)
	}
	if summary.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", summary.ErrorCount)
	}
}

func TestUsageStore_GetSummary_FilterByUser(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	events := []usage.Event{
		{ID: "e1", UserID: "user1", Timestamp: now},
		{ID: "e2", UserID: "user1", Timestamp: now},
		{ID: "e3", UserID: "user2", Timestamp: now},
		{ID: "e4", UserID: "user2", Timestamp: now},
		{ID: "e5", UserID: "user2", Timestamp: now},
	}

	store.RecordBatch(ctx, events)

	summary1, _ := store.GetSummary(ctx, "user1", start, end)
	if summary1.RequestCount != 2 {
		t.Errorf("user1 RequestCount = %d, want 2", summary1.RequestCount)
	}

	summary2, _ := store.GetSummary(ctx, "user2", start, end)
	if summary2.RequestCount != 3 {
		t.Errorf("user2 RequestCount = %d, want 3", summary2.RequestCount)
	}
}

func TestUsageStore_GetSummary_FilterByTimeRange(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	events := []usage.Event{
		{ID: "e1", UserID: "user1", Timestamp: baseTime.Add(-2 * time.Hour)}, // Before range
		{ID: "e2", UserID: "user1", Timestamp: baseTime},                     // In range
		{ID: "e3", UserID: "user1", Timestamp: baseTime.Add(30 * time.Minute)}, // In range
		{ID: "e4", UserID: "user1", Timestamp: baseTime.Add(2 * time.Hour)}, // After range
	}

	store.RecordBatch(ctx, events)

	start := baseTime.Add(-time.Minute)
	end := baseTime.Add(time.Hour)

	summary, _ := store.GetSummary(ctx, "user1", start, end)
	if summary.RequestCount != 2 {
		t.Errorf("RequestCount in time range = %d, want 2", summary.RequestCount)
	}
}

func TestUsageStore_GetSummary_NoEvents(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	summary, err := store.GetSummary(ctx, "user1", now.Add(-time.Hour), now)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0", summary.RequestCount)
	}
}

func TestUsageStore_GetHistory(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	// Create events in different months
	jan := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)
	mar := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

	events := []usage.Event{
		{ID: "e1", UserID: "user1", Timestamp: jan},
		{ID: "e2", UserID: "user1", Timestamp: jan},
		{ID: "e3", UserID: "user1", Timestamp: feb},
		{ID: "e4", UserID: "user1", Timestamp: mar},
		{ID: "e5", UserID: "user1", Timestamp: mar},
		{ID: "e6", UserID: "user1", Timestamp: mar},
	}

	store.RecordBatch(ctx, events)

	summaries, err := store.GetHistory(ctx, "user1", 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(summaries) != 3 {
		t.Errorf("expected 3 periods, got %d", len(summaries))
	}
}

func TestUsageStore_GetHistory_LimitPeriods(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	// Create events in 5 different months
	for m := 1; m <= 5; m++ {
		t := time.Date(2024, time.Month(m), 15, 12, 0, 0, 0, time.UTC)
		events := []usage.Event{{ID: string(rune('a' + m)), UserID: "user1", Timestamp: t}}
		store.RecordBatch(ctx, events)
	}

	summaries, err := store.GetHistory(ctx, "user1", 3)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(summaries) != 3 {
		t.Errorf("expected 3 periods with limit, got %d", len(summaries))
	}
}

func TestUsageStore_GetHistory_FilterByUser(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	events := []usage.Event{
		{ID: "e1", UserID: "user1", Timestamp: now},
		{ID: "e2", UserID: "user1", Timestamp: now},
		{ID: "e3", UserID: "user2", Timestamp: now},
	}

	store.RecordBatch(ctx, events)

	summaries, _ := store.GetHistory(ctx, "user1", 0)
	if len(summaries) == 0 {
		t.Error("expected at least 1 period for user1")
	}

	// Check that only user1 events are counted
	if summaries[0].RequestCount != 2 {
		t.Errorf("user1 RequestCount = %d, want 2", summaries[0].RequestCount)
	}
}

func TestUsageStore_GetHistory_NoEvents(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	summaries, err := store.GetHistory(ctx, "user1", 10)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(summaries) != 0 {
		t.Errorf("expected 0 periods, got %d", len(summaries))
	}
}

func TestUsageStore_GetRecentRequests(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	events := []usage.Event{
		{ID: "e1", UserID: "user1", Method: "GET", Timestamp: now.Add(-5 * time.Minute)},
		{ID: "e2", UserID: "user1", Method: "POST", Timestamp: now.Add(-4 * time.Minute)},
		{ID: "e3", UserID: "user1", Method: "PUT", Timestamp: now.Add(-3 * time.Minute)},
		{ID: "e4", UserID: "user1", Method: "DELETE", Timestamp: now.Add(-2 * time.Minute)},
		{ID: "e5", UserID: "user1", Method: "GET", Timestamp: now.Add(-1 * time.Minute)},
	}

	store.RecordBatch(ctx, events)

	recent, err := store.GetRecentRequests(ctx, "user1", 3)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}

	if len(recent) != 3 {
		t.Errorf("expected 3 recent requests, got %d", len(recent))
	}

	// Should return most recent first (reversed order)
	if recent[0].ID != "e5" {
		t.Errorf("most recent should be e5, got %s", recent[0].ID)
	}
}

func TestUsageStore_GetRecentRequests_FilterByUser(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user2"},
		{ID: "e3", UserID: "user1"},
		{ID: "e4", UserID: "user2"},
		{ID: "e5", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)

	recent, _ := store.GetRecentRequests(ctx, "user1", 10)
	if len(recent) != 3 {
		t.Errorf("expected 3 requests for user1, got %d", len(recent))
	}
}

func TestUsageStore_GetRecentRequests_LimitLargerThanTotal(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)

	recent, _ := store.GetRecentRequests(ctx, "user1", 100)
	if len(recent) != 2 {
		t.Errorf("expected 2 requests, got %d", len(recent))
	}
}

func TestUsageStore_GetRecentRequests_NoEvents(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	recent, err := store.GetRecentRequests(ctx, "user1", 10)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}

	if len(recent) != 0 {
		t.Errorf("expected 0 requests, got %d", len(recent))
	}
}

func TestUsageStore_GetAll(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user2"},
		{ID: "e3", UserID: "user3"},
	}

	store.RecordBatch(ctx, events)

	all := store.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 events, got %d", len(all))
	}
}

func TestUsageStore_GetAll_ReturnsCopy(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)

	all1 := store.GetAll()
	all2 := store.GetAll()

	// Verify they're independent copies
	if &all1[0] == &all2[0] {
		t.Error("GetAll should return a copy, not the original slice")
	}
}

func TestUsageStore_Drain(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user1"},
		{ID: "e3", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)

	drained := store.Drain()
	if len(drained) != 3 {
		t.Errorf("expected 3 drained events, got %d", len(drained))
	}

	// Verify store is empty after drain
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 events after Drain, got %d", len(all))
	}
}

func TestUsageStore_Drain_Empty(t *testing.T) {
	store := memory.NewUsageStore()

	drained := store.Drain()
	if len(drained) != 0 {
		t.Errorf("expected 0 drained events from empty store, got %d", len(drained))
	}
}

func TestUsageStore_Drain_MultipleTimes(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)

	drained1 := store.Drain()
	if len(drained1) != 1 {
		t.Errorf("first drain: expected 1, got %d", len(drained1))
	}

	drained2 := store.Drain()
	if len(drained2) != 0 {
		t.Errorf("second drain: expected 0, got %d", len(drained2))
	}
}

func TestUsageStore_Clear(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
		{ID: "e2", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)
	store.Clear()

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 events after Clear, got %d", len(all))
	}
}

func TestUsageStore_Clear_MultipleTimes(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	events := []usage.Event{
		{ID: "e1", UserID: "user1"},
	}

	store.RecordBatch(ctx, events)
	store.Clear()
	store.Clear() // Should not panic

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 events, got %d", len(all))
	}
}

func TestUsageStore_ConcurrentAccess(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			events := []usage.Event{
				{ID: string(rune('a' + idx%26)), UserID: "user1", Timestamp: time.Now()},
			}
			store.RecordBatch(ctx, events)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetAll()
			store.GetRecentRequests(ctx, "user1", 10)
			store.GetSummary(ctx, "user1", time.Now().Add(-time.Hour), time.Now())
			store.GetHistory(ctx, "user1", 3)
		}()
	}

	wg.Wait()
}

func TestUsageStore_EventWithAllFields(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()
	events := []usage.Event{
		{
			ID:             "complete-event",
			KeyID:          "key-123",
			UserID:         "user-456",
			Method:         "POST",
			Path:           "/api/v1/resource",
			StatusCode:     201,
			LatencyMs:      150,
			RequestBytes:   1024,
			ResponseBytes:  2048,
			CostMultiplier: 2.5,
			IPAddress:      "192.168.1.1",
			UserAgent:      "TestClient/1.0",
			Timestamp:      now,
		},
	}

	store.RecordBatch(ctx, events)

	all := store.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 event, got %d", len(all))
	}

	e := all[0]
	if e.ID != "complete-event" {
		t.Errorf("ID = %s", e.ID)
	}
	if e.KeyID != "key-123" {
		t.Errorf("KeyID = %s", e.KeyID)
	}
	if e.Method != "POST" {
		t.Errorf("Method = %s", e.Method)
	}
	if e.Path != "/api/v1/resource" {
		t.Errorf("Path = %s", e.Path)
	}
	if e.StatusCode != 201 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
	if e.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d", e.LatencyMs)
	}
	if e.CostMultiplier != 2.5 {
		t.Errorf("CostMultiplier = %f", e.CostMultiplier)
	}
	if e.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %s", e.IPAddress)
	}
	if e.UserAgent != "TestClient/1.0" {
		t.Errorf("UserAgent = %s", e.UserAgent)
	}
}

func TestUsageStore_FullLifecycle(t *testing.T) {
	store := memory.NewUsageStore()
	ctx := context.Background()

	now := time.Now()

	// Record batch
	events := []usage.Event{
		{ID: "e1", UserID: "user1", StatusCode: 200, Timestamp: now},
		{ID: "e2", UserID: "user1", StatusCode: 400, Timestamp: now},
		{ID: "e3", UserID: "user1", StatusCode: 500, Timestamp: now},
	}
	store.RecordBatch(ctx, events)

	// Get summary
	summary, _ := store.GetSummary(ctx, "user1", now.Add(-time.Hour), now.Add(time.Hour))
	if summary.RequestCount != 3 {
		t.Errorf("RequestCount = %d", summary.RequestCount)
	}
	if summary.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d", summary.ErrorCount)
	}

	// Get recent requests
	recent, _ := store.GetRecentRequests(ctx, "user1", 2)
	if len(recent) != 2 {
		t.Errorf("recent len = %d", len(recent))
	}

	// Get history
	history, _ := store.GetHistory(ctx, "user1", 1)
	if len(history) != 1 {
		t.Errorf("history len = %d", len(history))
	}

	// Drain
	drained := store.Drain()
	if len(drained) != 3 {
		t.Errorf("drained len = %d", len(drained))
	}

	// Verify empty
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("should be empty after drain, got %d", len(all))
	}

	// Add more and clear
	store.RecordBatch(ctx, events)
	store.Clear()

	all = store.GetAll()
	if len(all) != 0 {
		t.Errorf("should be empty after clear, got %d", len(all))
	}
}
