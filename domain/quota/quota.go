// Package quota provides pure functions for quota enforcement.
// All functions are deterministic with no side effects.
package quota

import (
	"time"

	"github.com/artpar/apigate/ports"
)

// MeterType determines which metric to use for quota enforcement.
type MeterType string

const (
	MeterTypeRequests     MeterType = "requests"      // Count raw API requests (default)
	MeterTypeComputeUnits MeterType = "compute_units" // Count weighted units (tokens, etc.)
)

// Config represents quota limits and enforcement settings (value type).
type Config struct {
	RequestsPerMonth int64            // -1 = unlimited (also used as UnitsPerMonth for compute_units)
	BytesPerMonth    int64            // 0 = unlimited
	EnforceMode      EnforceMode      // How to handle quota exceeded
	GracePct         float64          // Grace percentage before hard block (e.g., 0.05 = 5%)
	MeterType        MeterType        // Which metric to enforce: requests or compute_units
	EstimatedCost    float64          // Estimated cost per request for pre-check (compute_units mode)
}

// EnforceMode determines how quota limits are enforced.
type EnforceMode string

const (
	EnforceHard EnforceMode = "hard" // Reject requests when quota exceeded
	EnforceWarn EnforceMode = "warn" // Allow but add warning headers
	EnforceSoft EnforceMode = "soft" // Allow and bill overage
)

// WarningLevel indicates how close to or over quota the user is.
type WarningLevel int

const (
	WarningNone       WarningLevel = iota // < 80%
	WarningApproaching                    // >= 80%
	WarningCritical                       // >= 95%
	WarningExceeded                       // > 100%
)

// CheckResult represents the outcome of a quota check (value type).
type CheckResult struct {
	Allowed       bool
	CurrentUsage  int64
	Limit         int64
	PercentUsed   float64
	IsOverQuota   bool
	OverageAmount int64
	WarningLevel  WarningLevel
	Reason        string
}

// Check performs a quota check against current state.
// This is a PURE function - no side effects.
// The increment parameter is interpreted based on MeterType:
//   - For requests: number of requests to add (typically 1)
//   - For compute_units: estimated units to add (from EstimatedCost or actual cost)
func Check(state ports.QuotaState, cfg Config, increment int64) CheckResult {
	// Determine which metric to use based on MeterType
	var currentUsage int64
	switch cfg.MeterType {
	case MeterTypeComputeUnits:
		currentUsage = int64(state.ComputeUnits)
	default: // MeterTypeRequests or empty (backward compatible)
		currentUsage = state.RequestCount
	}

	// Handle unlimited quota
	if cfg.RequestsPerMonth < 0 {
		return CheckResult{
			Allowed:      true,
			CurrentUsage: currentUsage + increment,
			Limit:        -1,
			WarningLevel: WarningNone,
		}
	}

	newCount := currentUsage + increment
	limit := cfg.RequestsPerMonth
	gracedLimit := int64(float64(limit) * (1 + cfg.GracePct))

	var percentUsed float64
	if limit > 0 {
		percentUsed = float64(newCount) / float64(limit) * 100
	}

	result := CheckResult{
		CurrentUsage: newCount,
		Limit:        limit,
		PercentUsed:  percentUsed,
		IsOverQuota:  newCount > limit,
	}

	// Determine warning level
	switch {
	case percentUsed > 100:
		result.WarningLevel = WarningExceeded
		result.OverageAmount = newCount - limit
	case percentUsed >= 95:
		result.WarningLevel = WarningCritical
	case percentUsed >= 80:
		result.WarningLevel = WarningApproaching
	default:
		result.WarningLevel = WarningNone
	}

	// Determine if request is allowed based on enforcement mode
	switch cfg.EnforceMode {
	case EnforceHard:
		result.Allowed = newCount <= gracedLimit
		if !result.Allowed {
			result.Reason = "quota_exceeded"
		}
	case EnforceWarn, EnforceSoft:
		result.Allowed = true
	default:
		// Default to hard enforcement
		result.Allowed = newCount <= gracedLimit
		if !result.Allowed {
			result.Reason = "quota_exceeded"
		}
	}

	return result
}

// PeriodBounds returns the start and end of a billing period for a given time.
// This is a PURE function.
func PeriodBounds(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return
}

// ConfigFromPlan creates a quota Config from plan settings.
// This is a PURE function.
func ConfigFromPlan(p ports.Plan) Config {
	mode := EnforceHard
	switch p.QuotaEnforceMode {
	case ports.QuotaEnforceWarn:
		mode = EnforceWarn
	case ports.QuotaEnforceSoft:
		mode = EnforceSoft
	case ports.QuotaEnforceHard:
		mode = EnforceHard
	}

	gracePct := p.QuotaGracePct
	if gracePct == 0 {
		gracePct = 0.05 // Default 5% grace
	}

	// Map ports.MeterType to quota.MeterType
	meterType := MeterTypeRequests
	switch p.MeterType {
	case ports.MeterTypeComputeUnits:
		meterType = MeterTypeComputeUnits
	}

	estimatedCost := p.EstimatedCostPerReq
	if estimatedCost <= 0 {
		estimatedCost = 1.0 // Default to 1 unit per request
	}

	return Config{
		RequestsPerMonth: p.RequestsPerMonth,
		BytesPerMonth:    0, // Not yet implemented in Plan
		EnforceMode:      mode,
		GracePct:         gracePct,
		MeterType:        meterType,
		EstimatedCost:    estimatedCost,
	}
}

// WarningLevelString returns the string representation of a warning level.
func (w WarningLevel) String() string {
	switch w {
	case WarningNone:
		return "none"
	case WarningApproaching:
		return "approaching"
	case WarningCritical:
		return "critical"
	case WarningExceeded:
		return "exceeded"
	default:
		return "unknown"
	}
}
