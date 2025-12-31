package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/ratelimit"
)

func TestShardedRateLimitStore_NewShardedRateLimitStore(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	if store == nil {
		t.Fatal("NewShardedRateLimitStore returned nil")
	}
	defer store.Close()

	if store.Len() != 0 {
		t.Errorf("new store should be empty, got %d entries", store.Len())
	}
}

func TestShardedRateLimitStore_DefaultConfig(t *testing.T) {
	// Test with zero values - should use defaults
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards:       0, // Should default to 32
		CleanupInterval: 0, // Should default to 5 minutes
	})
	defer store.Close()

	if store == nil {
		t.Fatal("NewShardedRateLimitStore returned nil with zero config")
	}
}

func TestShardedRateLimitStore_CustomConfig(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards:       16,
		CleanupInterval: time.Second,
	})
	defer store.Close()

	if store == nil {
		t.Fatal("NewShardedRateLimitStore returned nil with custom config")
	}
}

func TestShardedRateLimitStore_GetSet(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	state := ratelimit.WindowState{
		Count:     10,
		WindowEnd: time.Now().Add(time.Minute),
		BurstUsed: 2,
	}

	err := store.Set(ctx, "key1", state)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Count != 10 {
		t.Errorf("Count = %d, want 10", got.Count)
	}
	if got.BurstUsed != 2 {
		t.Errorf("BurstUsed = %d, want 2", got.BurstUsed)
	}
}

func TestShardedRateLimitStore_Get_NotFound(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	state, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should return zero state for nonexistent key
	if state.Count != 0 {
		t.Errorf("Count = %d, want 0 for nonexistent key", state.Count)
	}
	if state.BurstUsed != 0 {
		t.Errorf("BurstUsed = %d, want 0", state.BurstUsed)
	}
}

func TestShardedRateLimitStore_Set_Overwrite(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 5})
	store.Set(ctx, "key1", ratelimit.WindowState{Count: 15})

	state, _ := store.Get(ctx, "key1")
	if state.Count != 15 {
		t.Errorf("Count = %d, want 15 (should be overwritten)", state.Count)
	}
}

func TestShardedRateLimitStore_GetAndCheck(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       10,
		Window:      time.Minute,
		BurstTokens: 2,
	}

	now := time.Now()

	// First check should be allowed
	result, err := store.GetAndCheck(ctx, "key1", cfg, now)
	if err != nil {
		t.Fatalf("GetAndCheck failed: %v", err)
	}

	if !result.Allowed {
		t.Error("first request should be allowed")
	}
	if result.Remaining != 9 {
		t.Errorf("Remaining = %d, want 9", result.Remaining)
	}
}

func TestShardedRateLimitStore_GetAndCheck_RateLimited(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       3,
		Window:      time.Minute,
		BurstTokens: 1,
	}

	now := time.Now()

	// First 3 requests should be allowed (within limit)
	for i := 0; i < 3; i++ {
		result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should use burst token
	result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
	if !result.Allowed {
		t.Error("4th request should be allowed (burst token)")
	}
	if result.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", result.Remaining)
	}

	// 5th request should be denied
	result, _ = store.GetAndCheck(ctx, "key1", cfg, now)
	if result.Allowed {
		t.Error("5th request should be denied")
	}
	if result.Reason != ratelimit.ReasonLimitExceeded {
		t.Errorf("Reason = %s, want %s", result.Reason, ratelimit.ReasonLimitExceeded)
	}
}

func TestShardedRateLimitStore_GetAndCheck_NewWindow(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       2,
		Window:      time.Minute,
		BurstTokens: 0,
	}

	now := time.Now().Truncate(time.Minute)

	// Use up the limit
	store.GetAndCheck(ctx, "key1", cfg, now)
	store.GetAndCheck(ctx, "key1", cfg, now)

	result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
	if result.Allowed {
		t.Error("should be rate limited")
	}

	// Move to next window (need to be AFTER the current window ends)
	// The window ends at now + 1 minute, so we need to be after that
	nextWindow := now.Add(time.Minute + time.Second)
	result, _ = store.GetAndCheck(ctx, "key1", cfg, nextWindow)
	if !result.Allowed {
		t.Error("should be allowed in new window")
	}
	if result.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1", result.Remaining)
	}
}

func TestShardedRateLimitStore_GetAndCheck_ResetAt(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:  10,
		Window: time.Minute,
	}

	now := time.Now().Truncate(time.Minute)

	result, _ := store.GetAndCheck(ctx, "key1", cfg, now)

	expectedReset := now.Add(time.Minute)
	if !result.ResetAt.Equal(expectedReset) {
		t.Errorf("ResetAt = %v, want %v", result.ResetAt, expectedReset)
	}
}

func TestShardedRateLimitStore_MultipleKeys(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 10})
	store.Set(ctx, "key2", ratelimit.WindowState{Count: 20})
	store.Set(ctx, "key3", ratelimit.WindowState{Count: 30})

	state1, _ := store.Get(ctx, "key1")
	state2, _ := store.Get(ctx, "key2")
	state3, _ := store.Get(ctx, "key3")

	if state1.Count != 10 {
		t.Errorf("key1 Count = %d, want 10", state1.Count)
	}
	if state2.Count != 20 {
		t.Errorf("key2 Count = %d, want 20", state2.Count)
	}
	if state3.Count != 30 {
		t.Errorf("key3 Count = %d, want 30", state3.Count)
	}
}

func TestShardedRateLimitStore_Clear(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 10})
	store.Set(ctx, "key2", ratelimit.WindowState{Count: 20})

	if store.Len() != 2 {
		t.Errorf("Len before Clear = %d, want 2", store.Len())
	}

	store.Clear()

	if store.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", store.Len())
	}

	state, _ := store.Get(ctx, "key1")
	if state.Count != 0 {
		t.Errorf("Count after Clear = %d, want 0", state.Count)
	}
}

func TestShardedRateLimitStore_Clear_MultipleTimes(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 10})
	store.Clear()
	store.Clear() // Should not panic

	if store.Len() != 0 {
		t.Errorf("Len = %d, want 0", store.Len())
	}
}

func TestShardedRateLimitStore_Len(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	if store.Len() != 0 {
		t.Errorf("initial Len = %d, want 0", store.Len())
	}

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 1})
	if store.Len() != 1 {
		t.Errorf("Len = %d, want 1", store.Len())
	}

	store.Set(ctx, "key2", ratelimit.WindowState{Count: 2})
	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2", store.Len())
	}

	// Overwriting same key should not increase count
	store.Set(ctx, "key1", ratelimit.WindowState{Count: 100})
	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2 (overwrite)", store.Len())
	}
}

func TestShardedRateLimitStore_Close(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		CleanupInterval: time.Millisecond * 100,
	})

	err := store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Close should stop the cleanup goroutine
	time.Sleep(time.Millisecond * 200)
}

func TestShardedRateLimitStore_ConcurrentAccess(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards: 8,
	})
	defer store.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent sets
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx%26))
			store.Set(ctx, key, ratelimit.WindowState{Count: idx})
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx%26))
			store.Get(ctx, key)
		}(i)
	}

	wg.Wait()
}

func TestShardedRateLimitStore_ConcurrentGetAndCheck(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       1000,
		Window:      time.Minute,
		BurstTokens: 100,
	}

	now := time.Now()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent GetAndCheck for same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetAndCheck(ctx, "key1", cfg, now)
		}()
	}

	wg.Wait()

	// Verify state is consistent
	state, _ := store.Get(ctx, "key1")
	if state.Count != numGoroutines {
		t.Errorf("Count = %d, want %d", state.Count, numGoroutines)
	}
}

func TestShardedRateLimitStore_ConcurrentGetAndCheck_MultipleKeys(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards: 16,
	})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       100,
		Window:      time.Minute,
		BurstTokens: 10,
	}

	now := time.Now()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent GetAndCheck for different keys
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx%10))
			store.GetAndCheck(ctx, key, cfg, now)
		}(i)
	}

	wg.Wait()

	// Should have 10 unique keys
	if store.Len() != 10 {
		t.Errorf("Len = %d, want 10", store.Len())
	}
}

func TestShardedRateLimitStore_Sharding(t *testing.T) {
	// Test with different shard counts
	shardCounts := []int{1, 2, 4, 8, 16, 32, 64}

	for _, numShards := range shardCounts {
		t.Run(string(rune('0'+numShards)), func(t *testing.T) {
			store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
				NumShards: numShards,
			})
			defer store.Close()
			ctx := context.Background()

			// Add multiple keys to spread across shards
			for i := 0; i < 100; i++ {
				key := string([]rune{rune('a' + i/26), rune('a' + i%26)})
				store.Set(ctx, key, ratelimit.WindowState{Count: i})
			}

			if store.Len() != 100 {
				t.Errorf("numShards=%d: Len = %d, want 100", numShards, store.Len())
			}
		})
	}
}

func TestShardedRateLimitStore_WindowStateWithBurst(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{})
	defer store.Close()
	ctx := context.Background()

	windowEnd := time.Now().Add(5 * time.Minute)
	state := ratelimit.WindowState{
		Count:     50,
		WindowEnd: windowEnd,
		BurstUsed: 3,
	}

	store.Set(ctx, "key1", state)

	got, _ := store.Get(ctx, "key1")
	if got.Count != 50 {
		t.Errorf("Count = %d, want 50", got.Count)
	}
	if got.BurstUsed != 3 {
		t.Errorf("BurstUsed = %d, want 3", got.BurstUsed)
	}
	if !got.WindowEnd.Equal(windowEnd) {
		t.Errorf("WindowEnd = %v, want %v", got.WindowEnd, windowEnd)
	}
}

func TestShardedRateLimitStore_FullLifecycle(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		NumShards:       4,
		CleanupInterval: time.Hour,
	})
	defer store.Close()
	ctx := context.Background()

	cfg := ratelimit.Config{
		Limit:       5,
		Window:      time.Minute,
		BurstTokens: 2,
	}

	now := time.Now().Truncate(time.Minute)

	// Initial state is empty
	state, _ := store.Get(ctx, "key1")
	if state.Count != 0 {
		t.Errorf("initial Count = %d, want 0", state.Count)
	}

	// Use GetAndCheck to consume limit
	for i := 0; i < 5; i++ {
		result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Consume burst tokens
	for i := 0; i < 2; i++ {
		result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
		if !result.Allowed {
			t.Errorf("burst request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	result, _ := store.GetAndCheck(ctx, "key1", cfg, now)
	if result.Allowed {
		t.Error("request should be denied after exhausting limit and burst")
	}

	// Verify state
	state, _ = store.Get(ctx, "key1")
	if state.Count != 7 {
		t.Errorf("Count = %d, want 7", state.Count)
	}
	if state.BurstUsed != 2 {
		t.Errorf("BurstUsed = %d, want 2", state.BurstUsed)
	}

	// Clear
	store.Clear()

	// Verify empty
	if store.Len() != 0 {
		t.Errorf("Len after clear = %d, want 0", store.Len())
	}
}

func TestShardedRateLimitStore_CleanupRemovesExpired(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		CleanupInterval: time.Millisecond * 50, // Fast cleanup for testing
	})
	defer store.Close()
	ctx := context.Background()

	// Add an expired entry (window ended more than 1 hour ago)
	expiredWindowEnd := time.Now().Add(-2 * time.Hour)
	store.Set(ctx, "expired_key", ratelimit.WindowState{
		Count:     10,
		WindowEnd: expiredWindowEnd,
	})

	// Add a fresh entry
	freshWindowEnd := time.Now().Add(time.Minute)
	store.Set(ctx, "fresh_key", ratelimit.WindowState{
		Count:     5,
		WindowEnd: freshWindowEnd,
	})

	if store.Len() != 2 {
		t.Errorf("Len = %d, want 2", store.Len())
	}

	// Wait for cleanup to run
	time.Sleep(time.Millisecond * 100)

	// Expired entry should be cleaned up, fresh should remain
	if store.Len() != 1 {
		t.Errorf("Len after cleanup = %d, want 1", store.Len())
	}

	// Verify expired is gone
	expiredState, _ := store.Get(ctx, "expired_key")
	if expiredState.Count != 0 {
		t.Errorf("expired key should be cleaned up, Count = %d", expiredState.Count)
	}

	// Verify fresh is still there
	freshState, _ := store.Get(ctx, "fresh_key")
	if freshState.Count != 5 {
		t.Errorf("fresh key should remain, Count = %d", freshState.Count)
	}
}

func TestShardedRateLimitStore_CleanupDoesNotRemoveFresh(t *testing.T) {
	store := memory.NewShardedRateLimitStore(memory.ShardedRateLimitConfig{
		CleanupInterval: time.Millisecond * 50,
	})
	defer store.Close()
	ctx := context.Background()

	// Add entries that should NOT be cleaned up
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		store.Set(ctx, key, ratelimit.WindowState{
			Count:     i,
			WindowEnd: time.Now().Add(time.Minute),
		})
	}

	if store.Len() != 10 {
		t.Errorf("Len = %d, want 10", store.Len())
	}

	// Wait for cleanup to run
	time.Sleep(time.Millisecond * 100)

	// All should remain
	if store.Len() != 10 {
		t.Errorf("Len after cleanup = %d, want 10 (nothing should be removed)", store.Len())
	}
}
