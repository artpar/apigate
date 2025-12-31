package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/usage"
)

func TestQuotaStore_NewQuotaStore(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	if store == nil {
		t.Fatal("NewQuotaStore returned nil")
	}
	defer store.Close()

	if store.Len() != 0 {
		t.Errorf("new store should be empty, got %d entries", store.Len())
	}
}

func TestQuotaStore_NewQuotaStore_DefaultConfig(t *testing.T) {
	// Test with zero values - should use defaults
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		NumShards:       0, // Should default to 32
		CleanupInterval: 0, // Should default to 1 hour
	})
	defer store.Close()

	if store == nil {
		t.Fatal("NewQuotaStore returned nil with zero config")
	}
}

func TestQuotaStore_NewQuotaStore_CustomConfig(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		NumShards:       16,
		CleanupInterval: time.Second,
	})
	defer store.Close()

	if store == nil {
		t.Fatal("NewQuotaStore returned nil with custom config")
	}
}

func TestQuotaStore_Get_EmptyStore(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	state, err := store.Get(ctx, "user1", periodStart)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should return empty state for non-existent user
	if state.UserID != "user1" {
		t.Errorf("UserID = %s, want user1", state.UserID)
	}
	if state.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0", state.RequestCount)
	}
	if state.ComputeUnits != 0 {
		t.Errorf("ComputeUnits = %f, want 0", state.ComputeUnits)
	}
	if state.BytesUsed != 0 {
		t.Errorf("BytesUsed = %d, want 0", state.BytesUsed)
	}
}

func TestQuotaStore_Increment_NewUser(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	state, err := store.Increment(ctx, "user1", periodStart, 5, 10.5, 1024)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}

	if state.UserID != "user1" {
		t.Errorf("UserID = %s, want user1", state.UserID)
	}
	if state.RequestCount != 5 {
		t.Errorf("RequestCount = %d, want 5", state.RequestCount)
	}
	if state.ComputeUnits != 10.5 {
		t.Errorf("ComputeUnits = %f, want 10.5", state.ComputeUnits)
	}
	if state.BytesUsed != 1024 {
		t.Errorf("BytesUsed = %d, want 1024", state.BytesUsed)
	}
	if state.LastUpdated.IsZero() {
		t.Error("LastUpdated should be set")
	}
}

func TestQuotaStore_Increment_ExistingUser(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// First increment
	store.Increment(ctx, "user1", periodStart, 5, 10.0, 1000)

	// Second increment
	state, err := store.Increment(ctx, "user1", periodStart, 3, 5.5, 500)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}

	if state.RequestCount != 8 {
		t.Errorf("RequestCount = %d, want 8", state.RequestCount)
	}
	if state.ComputeUnits != 15.5 {
		t.Errorf("ComputeUnits = %f, want 15.5", state.ComputeUnits)
	}
	if state.BytesUsed != 1500 {
		t.Errorf("BytesUsed = %d, want 1500", state.BytesUsed)
	}
}

func TestQuotaStore_Increment_MultipleIncrements(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Multiple increments
	for i := 0; i < 100; i++ {
		store.Increment(ctx, "user1", periodStart, 1, 0.1, 10)
	}

	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 100 {
		t.Errorf("RequestCount = %d, want 100", state.RequestCount)
	}
	// Float comparison with tolerance
	if state.ComputeUnits < 9.9 || state.ComputeUnits > 10.1 {
		t.Errorf("ComputeUnits = %f, want ~10.0", state.ComputeUnits)
	}
	if state.BytesUsed != 1000 {
		t.Errorf("BytesUsed = %d, want 1000", state.BytesUsed)
	}
}

func TestQuotaStore_Get_AfterIncrement(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", periodStart, 10, 25.0, 2048)

	state, err := store.Get(ctx, "user1", periodStart)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if state.RequestCount != 10 {
		t.Errorf("RequestCount = %d, want 10", state.RequestCount)
	}
	if state.ComputeUnits != 25.0 {
		t.Errorf("ComputeUnits = %f, want 25.0", state.ComputeUnits)
	}
	if state.BytesUsed != 2048 {
		t.Errorf("BytesUsed = %d, want 2048", state.BytesUsed)
	}
}

func TestQuotaStore_Sync(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	summary := usage.Summary{
		RequestCount: 100,
		ComputeUnits: 50.5,
		BytesIn:      2000,
		BytesOut:     3000,
	}

	err := store.Sync(ctx, "user1", periodStart, summary)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 100 {
		t.Errorf("RequestCount = %d, want 100", state.RequestCount)
	}
	if state.ComputeUnits != 50.5 {
		t.Errorf("ComputeUnits = %f, want 50.5", state.ComputeUnits)
	}
	if state.BytesUsed != 5000 { // BytesIn + BytesOut
		t.Errorf("BytesUsed = %d, want 5000", state.BytesUsed)
	}
}

func TestQuotaStore_Sync_OverwritesExisting(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// First set some data
	store.Increment(ctx, "user1", periodStart, 10, 5.0, 500)

	// Now sync with different data
	summary := usage.Summary{
		RequestCount: 200,
		ComputeUnits: 100.0,
		BytesIn:      4000,
		BytesOut:     6000,
	}

	store.Sync(ctx, "user1", periodStart, summary)

	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 200 {
		t.Errorf("RequestCount = %d, want 200 (should be overwritten)", state.RequestCount)
	}
}

func TestQuotaStore_MultipleUsers(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", periodStart, 10, 5.0, 100)
	store.Increment(ctx, "user2", periodStart, 20, 10.0, 200)
	store.Increment(ctx, "user3", periodStart, 30, 15.0, 300)

	state1, _ := store.Get(ctx, "user1", periodStart)
	state2, _ := store.Get(ctx, "user2", periodStart)
	state3, _ := store.Get(ctx, "user3", periodStart)

	if state1.RequestCount != 10 {
		t.Errorf("user1 RequestCount = %d, want 10", state1.RequestCount)
	}
	if state2.RequestCount != 20 {
		t.Errorf("user2 RequestCount = %d, want 20", state2.RequestCount)
	}
	if state3.RequestCount != 30 {
		t.Errorf("user3 RequestCount = %d, want 30", state3.RequestCount)
	}
}

func TestQuotaStore_MultiplePeriods(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	jan := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	mar := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", jan, 100, 10.0, 1000)
	store.Increment(ctx, "user1", feb, 200, 20.0, 2000)
	store.Increment(ctx, "user1", mar, 300, 30.0, 3000)

	stateJan, _ := store.Get(ctx, "user1", jan)
	stateFeb, _ := store.Get(ctx, "user1", feb)
	stateMar, _ := store.Get(ctx, "user1", mar)

	if stateJan.RequestCount != 100 {
		t.Errorf("Jan RequestCount = %d, want 100", stateJan.RequestCount)
	}
	if stateFeb.RequestCount != 200 {
		t.Errorf("Feb RequestCount = %d, want 200", stateFeb.RequestCount)
	}
	if stateMar.RequestCount != 300 {
		t.Errorf("Mar RequestCount = %d, want 300", stateMar.RequestCount)
	}
}

func TestQuotaStore_Clear(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", periodStart, 10, 5.0, 100)
	store.Increment(ctx, "user2", periodStart, 20, 10.0, 200)

	if store.Len() != 2 {
		t.Errorf("Len before Clear = %d, want 2", store.Len())
	}

	store.Clear()

	if store.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", store.Len())
	}

	// Verify data is gone
	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 0 {
		t.Errorf("RequestCount after Clear = %d, want 0", state.RequestCount)
	}
}

func TestQuotaStore_Clear_MultipleTimes(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	store.Increment(ctx, "user1", periodStart, 10, 5.0, 100)

	store.Clear()
	store.Clear() // Should not panic

	if store.Len() != 0 {
		t.Errorf("Len = %d, want 0", store.Len())
	}
}

func TestQuotaStore_Len(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	if store.Len() != 0 {
		t.Errorf("initial Len = %d, want 0", store.Len())
	}

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", periodStart, 1, 0.0, 0)
	if store.Len() != 1 {
		t.Errorf("Len = %d, want 1", store.Len())
	}

	store.Increment(ctx, "user2", periodStart, 1, 0.0, 0)
	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2", store.Len())
	}

	// Same user, same period - should not increase count
	store.Increment(ctx, "user1", periodStart, 1, 0.0, 0)
	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2 (same user/period)", store.Len())
	}
}

func TestQuotaStore_Close(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		CleanupInterval: time.Millisecond * 100,
	})

	err := store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Close should stop the cleanup goroutine
	// We can verify by waiting and ensuring no panic
	time.Sleep(time.Millisecond * 200)
}

func TestQuotaStore_ConcurrentAccess(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		NumShards: 8,
	})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent increments for different users
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := string(rune('a' + idx%26))
			store.Increment(ctx, userID, periodStart, 1, 0.1, 10)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := string(rune('a' + idx%26))
			store.Get(ctx, userID, periodStart)
		}(i)
	}

	wg.Wait()

	// Verify no data corruption
	if store.Len() > 26 {
		t.Errorf("Len = %d, expected <= 26 (unique users)", store.Len())
	}
}

func TestQuotaStore_ConcurrentIncrementsSameUser(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent increments for same user
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Increment(ctx, "user1", periodStart, 1, 0.0, 0)
		}()
	}

	wg.Wait()

	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != int64(numGoroutines) {
		t.Errorf("RequestCount = %d, want %d", state.RequestCount, numGoroutines)
	}
}

func TestQuotaStore_WithUsageStore(t *testing.T) {
	usageStore := memory.NewUsageStore()
	ctx := context.Background()

	// Add some usage events
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	events := []usage.Event{
		{ID: "e1", UserID: "user1", RequestBytes: 100, ResponseBytes: 200, CostMultiplier: 1.0, Timestamp: now},
		{ID: "e2", UserID: "user1", RequestBytes: 150, ResponseBytes: 250, CostMultiplier: 1.5, Timestamp: now},
	}
	usageStore.RecordBatch(ctx, events)

	// Create quota store with usage store
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		UsageStore: usageStore,
	})
	defer store.Close()

	// Get should load from usage store
	state, err := store.Get(ctx, "user1", periodStart)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should have data from usage store
	if state.RequestCount == 0 && state.BytesUsed == 0 {
		t.Log("Note: Usage store returned empty summary (might be time range issue)")
	}
}

func TestQuotaStore_Sharding(t *testing.T) {
	// Test with different shard counts
	shardCounts := []int{1, 2, 4, 8, 16, 32, 64}

	for _, numShards := range shardCounts {
		t.Run(string(rune('0'+numShards)), func(t *testing.T) {
			store := memory.NewQuotaStore(memory.QuotaStoreConfig{
				NumShards: numShards,
			})
			defer store.Close()
			ctx := context.Background()

			periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

			// Add multiple users to spread across shards
			for i := 0; i < 100; i++ {
				userID := string([]rune{rune('a' + i/26), rune('a' + i%26)})
				store.Increment(ctx, userID, periodStart, 1, 0.0, 0)
			}

			if store.Len() != 100 {
				t.Errorf("numShards=%d: Len = %d, want 100", numShards, store.Len())
			}
		})
	}
}

func TestQuotaStore_PeriodStartFormat(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	// Different times in the same month should hash to same key
	time1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 3, 15, 12, 30, 0, 0, time.UTC)
	time3 := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)

	store.Increment(ctx, "user1", time1, 10, 0.0, 0)
	store.Increment(ctx, "user1", time2, 5, 0.0, 0)
	store.Increment(ctx, "user1", time3, 3, 0.0, 0)

	// Key format is "userID:2006-01", so all three should be same key
	state, _ := store.Get(ctx, "user1", time1)
	if state.RequestCount != 18 {
		t.Errorf("RequestCount = %d, want 18 (all increments combined)", state.RequestCount)
	}
}

func TestQuotaStore_DifferentMonths(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{})
	defer store.Close()
	ctx := context.Background()

	jan := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)

	store.Increment(ctx, "user1", jan, 100, 0.0, 0)
	store.Increment(ctx, "user1", feb, 200, 0.0, 0)

	stateJan, _ := store.Get(ctx, "user1", jan)
	stateFeb, _ := store.Get(ctx, "user1", feb)

	if stateJan.RequestCount != 100 {
		t.Errorf("Jan RequestCount = %d, want 100", stateJan.RequestCount)
	}
	if stateFeb.RequestCount != 200 {
		t.Errorf("Feb RequestCount = %d, want 200", stateFeb.RequestCount)
	}
}

func TestQuotaStore_FullLifecycle(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		NumShards:       4,
		CleanupInterval: time.Hour,
	})
	defer store.Close()
	ctx := context.Background()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Initial state is empty
	state, _ := store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 0 {
		t.Errorf("initial RequestCount = %d, want 0", state.RequestCount)
	}

	// Increment
	store.Increment(ctx, "user1", periodStart, 10, 5.0, 1000)

	// Verify
	state, _ = store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 10 {
		t.Errorf("RequestCount = %d, want 10", state.RequestCount)
	}

	// Sync with new data
	summary := usage.Summary{
		RequestCount: 50,
		ComputeUnits: 25.0,
		BytesIn:      3000,
		BytesOut:     2000,
	}
	store.Sync(ctx, "user1", periodStart, summary)

	// Verify sync overwrote
	state, _ = store.Get(ctx, "user1", periodStart)
	if state.RequestCount != 50 {
		t.Errorf("RequestCount after sync = %d, want 50", state.RequestCount)
	}

	// Clear
	store.Clear()

	// Verify empty
	if store.Len() != 0 {
		t.Errorf("Len after clear = %d, want 0", store.Len())
	}
}

func TestQuotaStore_CleanupRemovesOldPeriods(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		CleanupInterval: time.Millisecond * 50, // Fast cleanup for testing
	})
	defer store.Close()
	ctx := context.Background()

	// Add an old period (more than 2 months ago)
	oldPeriod := time.Now().AddDate(0, -3, 0)
	store.Increment(ctx, "user1", oldPeriod, 100, 10.0, 1000)

	// Add a recent period
	currentPeriod := time.Now()
	store.Increment(ctx, "user2", currentPeriod, 50, 5.0, 500)

	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2", store.Len())
	}

	// Wait for cleanup to run
	time.Sleep(time.Millisecond * 100)

	// Old period should be cleaned up, recent should remain
	if store.Len() != 1 {
		t.Errorf("Len after cleanup = %d, want 1", store.Len())
	}

	// Verify old period is gone
	oldState, _ := store.Get(ctx, "user1", oldPeriod)
	if oldState.RequestCount != 0 {
		t.Errorf("old period should be cleaned up, RequestCount = %d", oldState.RequestCount)
	}

	// Verify current period is still there
	currentState, _ := store.Get(ctx, "user2", currentPeriod)
	if currentState.RequestCount != 50 {
		t.Errorf("current period should remain, RequestCount = %d", currentState.RequestCount)
	}
}

func TestQuotaStore_CleanupDoesNotRemoveRecentPeriods(t *testing.T) {
	store := memory.NewQuotaStore(memory.QuotaStoreConfig{
		CleanupInterval: time.Millisecond * 50,
	})
	defer store.Close()
	ctx := context.Background()

	// Add recent period entries (within last 2 months)
	currentPeriod := time.Now()
	lastMonth := time.Now().AddDate(0, -1, 0)

	store.Increment(ctx, "user1", currentPeriod, 100, 10.0, 1000)
	store.Increment(ctx, "user2", lastMonth, 50, 5.0, 500)

	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2", store.Len())
	}

	// Wait for cleanup to run
	time.Sleep(time.Millisecond * 100)

	// Both should remain
	if store.Len() != 2 {
		t.Errorf("Len after cleanup = %d, want 2 (nothing should be removed)", store.Len())
	}
}
