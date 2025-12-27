package ratelimit_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/ratelimit"
)

var (
	baseTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	cfg      = ratelimit.Config{
		Limit:       10,
		Window:      time.Minute,
		BurstTokens: 2,
	}
)

func TestCheck_AllowsWithinLimit(t *testing.T) {
	state := ratelimit.WindowState{
		Count:     5,
		WindowEnd: baseTime.Add(30 * time.Second),
	}

	result, newState := ratelimit.Check(state, cfg, baseTime)

	if !result.Allowed {
		t.Error("expected request to be allowed")
	}
	if result.Remaining != 4 { // 10 - 6 = 4
		t.Errorf("remaining = %d, want 4", result.Remaining)
	}
	if newState.Count != 6 {
		t.Errorf("count = %d, want 6", newState.Count)
	}
}

func TestCheck_DeniesOverLimit(t *testing.T) {
	state := ratelimit.WindowState{
		Count:     10,
		WindowEnd: baseTime.Add(30 * time.Second),
		BurstUsed: 2, // Burst exhausted
	}

	result, newState := ratelimit.Check(state, cfg, baseTime)

	if result.Allowed {
		t.Error("expected request to be denied")
	}
	if result.Reason != ratelimit.ReasonLimitExceeded {
		t.Errorf("reason = %q, want %q", result.Reason, ratelimit.ReasonLimitExceeded)
	}
	if newState.Count != 10 { // Count unchanged
		t.Errorf("count = %d, want 10", newState.Count)
	}
}

func TestCheck_UsesBurstTokens(t *testing.T) {
	state := ratelimit.WindowState{
		Count:     10, // At limit
		WindowEnd: baseTime.Add(30 * time.Second),
		BurstUsed: 0, // Burst available
	}

	result, newState := ratelimit.Check(state, cfg, baseTime)

	if !result.Allowed {
		t.Error("expected request to be allowed via burst")
	}
	if result.Remaining != 0 {
		t.Errorf("remaining = %d, want 0", result.Remaining)
	}
	if newState.BurstUsed != 1 {
		t.Errorf("burstUsed = %d, want 1", newState.BurstUsed)
	}
}

func TestCheck_ResetsExpiredWindow(t *testing.T) {
	pastWindow := baseTime.Add(-time.Hour)
	state := ratelimit.WindowState{
		Count:     100, // Way over limit
		WindowEnd: pastWindow,
		BurstUsed: 10,
	}

	result, newState := ratelimit.Check(state, cfg, baseTime)

	if !result.Allowed {
		t.Error("expected request to be allowed after window reset")
	}
	if result.Remaining != 9 { // 10 - 1 = 9
		t.Errorf("remaining = %d, want 9", result.Remaining)
	}
	if newState.Count != 1 {
		t.Errorf("count = %d, want 1 (reset)", newState.Count)
	}
	if newState.BurstUsed != 0 {
		t.Errorf("burstUsed = %d, want 0 (reset)", newState.BurstUsed)
	}
}

func TestCheck_HandlesZeroState(t *testing.T) {
	state := ratelimit.WindowState{} // Zero value

	result, newState := ratelimit.Check(state, cfg, baseTime)

	if !result.Allowed {
		t.Error("expected first request to be allowed")
	}
	if result.Remaining != 9 {
		t.Errorf("remaining = %d, want 9", result.Remaining)
	}
	if newState.Count != 1 {
		t.Errorf("count = %d, want 1", newState.Count)
	}
}

func TestCheck_Deterministic(t *testing.T) {
	// Same input should always produce same output
	state := ratelimit.WindowState{
		Count:     7,
		WindowEnd: baseTime.Add(30 * time.Second),
	}

	result1, state1 := ratelimit.Check(state, cfg, baseTime)
	result2, state2 := ratelimit.Check(state, cfg, baseTime)

	if result1 != result2 {
		t.Error("Check should be deterministic")
	}
	if state1 != state2 {
		t.Error("Check should be deterministic")
	}
}

func TestCalculateDelay(t *testing.T) {
	tests := []struct {
		name      string
		result    ratelimit.CheckResult
		now       time.Time
		wantDelay time.Duration
	}{
		{
			name: "allowed returns zero",
			result: ratelimit.CheckResult{
				Allowed: true,
				ResetAt: baseTime.Add(time.Minute),
			},
			now:       baseTime,
			wantDelay: 0,
		},
		{
			name: "denied returns time to reset",
			result: ratelimit.CheckResult{
				Allowed: false,
				ResetAt: baseTime.Add(30 * time.Second),
			},
			now:       baseTime,
			wantDelay: 30 * time.Second,
		},
		{
			name: "past reset returns zero",
			result: ratelimit.CheckResult{
				Allowed: false,
				ResetAt: baseTime.Add(-time.Second),
			},
			now:       baseTime,
			wantDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratelimit.CalculateDelay(tt.result, tt.now)
			if got != tt.wantDelay {
				t.Errorf("CalculateDelay() = %v, want %v", got, tt.wantDelay)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	window1 := baseTime.Add(30 * time.Second)
	window2 := baseTime.Add(45 * time.Second)

	states := []ratelimit.WindowState{
		{Count: 5, WindowEnd: window2, BurstUsed: 1},
		{Count: 8, WindowEnd: window1, BurstUsed: 0},
		{Count: 3, WindowEnd: window2, BurstUsed: 2},
	}

	result := ratelimit.Merge(states...)

	if result.Count != 8 { // Max count
		t.Errorf("count = %d, want 8", result.Count)
	}
	if result.BurstUsed != 2 { // Max burst
		t.Errorf("burstUsed = %d, want 2", result.BurstUsed)
	}
	if !result.WindowEnd.Equal(window1) { // Earliest window
		t.Errorf("windowEnd = %v, want %v", result.WindowEnd, window1)
	}
}

// Benchmark to ensure rate limit check is fast
func BenchmarkCheck(b *testing.B) {
	state := ratelimit.WindowState{
		Count:     5,
		WindowEnd: baseTime.Add(30 * time.Second),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ratelimit.Check(state, cfg, baseTime)
	}
}
