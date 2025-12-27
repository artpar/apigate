// Package ratelimit provides pure rate limiting algorithms.
// All functions are deterministic - same input always produces same output.
package ratelimit

import "time"

// WindowState represents the current state of a rate limit window (value type).
type WindowState struct {
	Count     int       // Requests in current window
	WindowEnd time.Time // When current window ends
	BurstUsed int       // Burst tokens used
}

// CheckResult represents the outcome of a rate limit check (value type).
type CheckResult struct {
	Allowed   bool
	Remaining int       // Requests remaining in window
	ResetAt   time.Time // When limit resets
	Reason    string    // If not allowed, why
}

// Config holds rate limit configuration (value type).
type Config struct {
	Limit       int           // Requests per window
	Window      time.Duration // Window duration
	BurstTokens int           // Extra tokens for bursts
}

// Reasons for denial
const (
	ReasonLimitExceeded = "rate_limit_exceeded"
)

// Check performs a rate limit check.
// This is a PURE function - no side effects, deterministic.
//
// Parameters:
//   - state: current window state
//   - cfg: rate limit configuration
//   - now: current timestamp
//
// Returns:
//   - result: whether request is allowed and metadata
//   - newState: updated state (caller must persist if needed)
func Check(state WindowState, cfg Config, now time.Time) (CheckResult, WindowState) {
	// Calculate window boundaries
	windowStart := now.Truncate(cfg.Window)
	windowEnd := windowStart.Add(cfg.Window)

	// Check if we're in a new window
	if now.After(state.WindowEnd) || state.WindowEnd.IsZero() {
		// New window - reset counters
		state = WindowState{
			Count:     0,
			WindowEnd: windowEnd,
			BurstUsed: 0,
		}
	}

	// Check if within normal limit
	if state.Count < cfg.Limit {
		state.Count++
		return CheckResult{
			Allowed:   true,
			Remaining: cfg.Limit - state.Count,
			ResetAt:   state.WindowEnd,
		}, state
	}

	// Over limit - try burst tokens
	burstRemaining := cfg.BurstTokens - state.BurstUsed
	if burstRemaining > 0 {
		state.Count++
		state.BurstUsed++
		return CheckResult{
			Allowed:   true,
			Remaining: 0, // Normal limit exhausted, using burst
			ResetAt:   state.WindowEnd,
		}, state
	}

	// Completely over limit
	return CheckResult{
		Allowed:   false,
		Remaining: 0,
		ResetAt:   state.WindowEnd,
		Reason:    ReasonLimitExceeded,
	}, state
}

// CalculateDelay returns how long to wait before retrying.
// This is a PURE function.
func CalculateDelay(result CheckResult, now time.Time) time.Duration {
	if result.Allowed {
		return 0
	}
	delay := result.ResetAt.Sub(now)
	if delay < 0 {
		return 0
	}
	return delay
}

// Merge combines multiple window states (for distributed rate limiting).
// Takes the maximum count and earliest window end.
// This is a PURE function.
func Merge(states ...WindowState) WindowState {
	if len(states) == 0 {
		return WindowState{}
	}

	result := states[0]
	for _, s := range states[1:] {
		if s.Count > result.Count {
			result.Count = s.Count
		}
		if s.BurstUsed > result.BurstUsed {
			result.BurstUsed = s.BurstUsed
		}
		// Use earliest window end for safety
		if !s.WindowEnd.IsZero() && (result.WindowEnd.IsZero() || s.WindowEnd.Before(result.WindowEnd)) {
			result.WindowEnd = s.WindowEnd
		}
	}

	return result
}
