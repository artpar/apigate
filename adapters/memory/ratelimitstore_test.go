package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/ratelimit"
)

// RateLimitStore tests

func TestRateLimitStore_GetSet(t *testing.T) {
	store := memory.NewRateLimitStore()
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

func TestRateLimitStore_Get_NotFound(t *testing.T) {
	store := memory.NewRateLimitStore()
	ctx := context.Background()

	state, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should return zero state
	if state.Count != 0 {
		t.Errorf("expected zero count for nonexistent key")
	}
}

func TestRateLimitStore_Clear(t *testing.T) {
	store := memory.NewRateLimitStore()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 5})
	store.Set(ctx, "key2", ratelimit.WindowState{Count: 10})

	store.Clear()

	state, _ := store.Get(ctx, "key1")
	if state.Count != 0 {
		t.Errorf("expected 0 after Clear, got %d", state.Count)
	}
}
