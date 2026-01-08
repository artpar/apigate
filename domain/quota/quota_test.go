// Package quota provides pure functions for quota enforcement.
// Tests for all public functions and types.
package quota

import (
	"testing"
	"time"

	"github.com/artpar/apigate/ports"
)

// -----------------------------------------------------------------------------
// Check function tests
// -----------------------------------------------------------------------------

func TestCheck_UnlimitedQuota(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 100,
	}
	cfg := Config{
		RequestsPerMonth: -1, // Unlimited
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 10)

	if !result.Allowed {
		t.Errorf("expected Allowed=true for unlimited quota, got false")
	}
	if result.CurrentUsage != 110 {
		t.Errorf("expected CurrentUsage=110, got %d", result.CurrentUsage)
	}
	if result.Limit != -1 {
		t.Errorf("expected Limit=-1 for unlimited, got %d", result.Limit)
	}
	if result.WarningLevel != WarningNone {
		t.Errorf("expected WarningLevel=WarningNone, got %v", result.WarningLevel)
	}
}

func TestCheck_ZeroIncrement(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 50,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 0)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.CurrentUsage != 50 {
		t.Errorf("expected CurrentUsage=50, got %d", result.CurrentUsage)
	}
	if result.PercentUsed != 50.0 {
		t.Errorf("expected PercentUsed=50.0, got %f", result.PercentUsed)
	}
}

func TestCheck_UnderQuota(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 50,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 10)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.CurrentUsage != 60 {
		t.Errorf("expected CurrentUsage=60, got %d", result.CurrentUsage)
	}
	if result.Limit != 100 {
		t.Errorf("expected Limit=100, got %d", result.Limit)
	}
	if result.PercentUsed != 60.0 {
		t.Errorf("expected PercentUsed=60.0, got %f", result.PercentUsed)
	}
	if result.IsOverQuota {
		t.Errorf("expected IsOverQuota=false, got true")
	}
	if result.WarningLevel != WarningNone {
		t.Errorf("expected WarningLevel=WarningNone, got %v", result.WarningLevel)
	}
}

func TestCheck_WarningLevelApproaching(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 79,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.PercentUsed != 80.0 {
		t.Errorf("expected PercentUsed=80.0, got %f", result.PercentUsed)
	}
	if result.WarningLevel != WarningApproaching {
		t.Errorf("expected WarningLevel=WarningApproaching, got %v", result.WarningLevel)
	}
}

func TestCheck_WarningLevelCritical(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 94,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.PercentUsed != 95.0 {
		t.Errorf("expected PercentUsed=95.0, got %f", result.PercentUsed)
	}
	if result.WarningLevel != WarningCritical {
		t.Errorf("expected WarningLevel=WarningCritical, got %v", result.WarningLevel)
	}
}

func TestCheck_WarningLevelExceeded(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 100,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	if !result.Allowed {
		t.Errorf("expected Allowed=true within grace period, got false")
	}
	if result.PercentUsed != 101.0 {
		t.Errorf("expected PercentUsed=101.0, got %f", result.PercentUsed)
	}
	if result.WarningLevel != WarningExceeded {
		t.Errorf("expected WarningLevel=WarningExceeded, got %v", result.WarningLevel)
	}
	if !result.IsOverQuota {
		t.Errorf("expected IsOverQuota=true, got false")
	}
	if result.OverageAmount != 1 {
		t.Errorf("expected OverageAmount=1, got %d", result.OverageAmount)
	}
}

func TestCheck_EnforceHard_WithinGrace(t *testing.T) {
	// With 5% grace on 100 requests, the grace limit is 105
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 104,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	// 105 is within graced limit (105)
	if !result.Allowed {
		t.Errorf("expected Allowed=true within grace, got false")
	}
	if result.Reason != "" {
		t.Errorf("expected empty Reason, got %q", result.Reason)
	}
}

func TestCheck_EnforceHard_ExceedsGrace(t *testing.T) {
	// With 5% grace on 100 requests, the grace limit is 105
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 105,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	// 106 exceeds graced limit (105)
	if result.Allowed {
		t.Errorf("expected Allowed=false when exceeding grace, got true")
	}
	if result.Reason != "quota_exceeded" {
		t.Errorf("expected Reason='quota_exceeded', got %q", result.Reason)
	}
}

func TestCheck_EnforceWarn(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 150,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceWarn,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 10)

	// EnforceWarn always allows
	if !result.Allowed {
		t.Errorf("expected Allowed=true for EnforceWarn, got false")
	}
	if result.IsOverQuota != true {
		t.Errorf("expected IsOverQuota=true, got false")
	}
}

func TestCheck_EnforceSoft(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 200,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceSoft,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 10)

	// EnforceSoft always allows
	if !result.Allowed {
		t.Errorf("expected Allowed=true for EnforceSoft, got false")
	}
	if result.IsOverQuota != true {
		t.Errorf("expected IsOverQuota=true, got false")
	}
	if result.OverageAmount != 110 {
		t.Errorf("expected OverageAmount=110, got %d", result.OverageAmount)
	}
}

func TestCheck_DefaultEnforceMode(t *testing.T) {
	// Test with an unknown/empty enforce mode (should default to hard)
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 105,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceMode("unknown"),
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	// Default to hard enforcement - 106 exceeds graced limit (105)
	if result.Allowed {
		t.Errorf("expected Allowed=false for default enforcement, got true")
	}
	if result.Reason != "quota_exceeded" {
		t.Errorf("expected Reason='quota_exceeded', got %q", result.Reason)
	}
}

func TestCheck_ZeroLimit(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 0,
	}
	cfg := Config{
		RequestsPerMonth: 0, // Zero limit (edge case)
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	// With 0 limit and 0 grace, graced limit is 0, so 1 request exceeds
	if result.Allowed {
		t.Errorf("expected Allowed=false with zero limit, got true")
	}
}

func TestCheck_LargeNumbers(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 999999990,
	}
	cfg := Config{
		RequestsPerMonth: 1000000000,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 10)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.CurrentUsage != 1000000000 {
		t.Errorf("expected CurrentUsage=1000000000, got %d", result.CurrentUsage)
	}
}

func TestCheck_NoGrace(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 100,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0, // No grace
	}

	result := Check(state, cfg, 1)

	// Without grace, 101 requests should be blocked
	if result.Allowed {
		t.Errorf("expected Allowed=false without grace, got true")
	}
}

func TestCheck_ExactlyAtLimit(t *testing.T) {
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 99,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, 1)

	if !result.Allowed {
		t.Errorf("expected Allowed=true at exact limit, got false")
	}
	if result.PercentUsed != 100.0 {
		t.Errorf("expected PercentUsed=100.0, got %f", result.PercentUsed)
	}
	if result.IsOverQuota {
		t.Errorf("expected IsOverQuota=false at exactly 100%%, got true")
	}
	// At exactly 100%, warning level should be WarningCritical (>=95% and <=100%)
	if result.WarningLevel != WarningCritical {
		t.Errorf("expected WarningLevel=WarningCritical at 100%%, got %v", result.WarningLevel)
	}
}

// -----------------------------------------------------------------------------
// PeriodBounds function tests
// -----------------------------------------------------------------------------

func TestPeriodBounds_StartOfMonth(t *testing.T) {
	input := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.January, 31, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_MiddleOfMonth(t *testing.T) {
	input := time.Date(2024, time.March, 15, 14, 30, 45, 123, time.UTC)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.March, 31, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_EndOfMonth(t *testing.T) {
	input := time.Date(2024, time.February, 29, 23, 59, 59, 999999999, time.UTC) // Leap year
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.February, 29, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_December(t *testing.T) {
	input := time.Date(2024, time.December, 25, 12, 0, 0, 0, time.UTC)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.December, 31, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_NonUTCTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone not available: %v", err)
	}

	input := time.Date(2024, time.June, 15, 10, 0, 0, 0, loc)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.June, 1, 0, 0, 0, 0, loc)
	expectedEnd := time.Date(2024, time.June, 30, 23, 59, 59, 999999999, loc)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_LeapYear(t *testing.T) {
	input := time.Date(2024, time.February, 15, 0, 0, 0, 0, time.UTC)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.February, 29, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

func TestPeriodBounds_NonLeapYear(t *testing.T) {
	input := time.Date(2023, time.February, 15, 0, 0, 0, 0, time.UTC)
	start, end := PeriodBounds(input)

	expectedStart := time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2023, time.February, 28, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start=%v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end=%v, got %v", expectedEnd, end)
	}
}

// -----------------------------------------------------------------------------
// ConfigFromPlan function tests
// -----------------------------------------------------------------------------

func TestConfigFromPlan_HardEnforce(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-1",
		Name:             "Basic",
		RequestsPerMonth: 1000,
		QuotaEnforceMode: ports.QuotaEnforceHard,
		QuotaGracePct:    0.10,
	}

	cfg := ConfigFromPlan(plan)

	if cfg.RequestsPerMonth != 1000 {
		t.Errorf("expected RequestsPerMonth=1000, got %d", cfg.RequestsPerMonth)
	}
	if cfg.EnforceMode != EnforceHard {
		t.Errorf("expected EnforceMode=EnforceHard, got %v", cfg.EnforceMode)
	}
	if cfg.GracePct != 0.10 {
		t.Errorf("expected GracePct=0.10, got %f", cfg.GracePct)
	}
	if cfg.BytesPerMonth != 0 {
		t.Errorf("expected BytesPerMonth=0 (not implemented), got %d", cfg.BytesPerMonth)
	}
}

func TestConfigFromPlan_WarnEnforce(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-2",
		Name:             "Pro",
		RequestsPerMonth: 10000,
		QuotaEnforceMode: ports.QuotaEnforceWarn,
		QuotaGracePct:    0.15,
	}

	cfg := ConfigFromPlan(plan)

	if cfg.EnforceMode != EnforceWarn {
		t.Errorf("expected EnforceMode=EnforceWarn, got %v", cfg.EnforceMode)
	}
	if cfg.GracePct != 0.15 {
		t.Errorf("expected GracePct=0.15, got %f", cfg.GracePct)
	}
}

func TestConfigFromPlan_SoftEnforce(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-3",
		Name:             "Enterprise",
		RequestsPerMonth: 100000,
		QuotaEnforceMode: ports.QuotaEnforceSoft,
		QuotaGracePct:    0.20,
	}

	cfg := ConfigFromPlan(plan)

	if cfg.EnforceMode != EnforceSoft {
		t.Errorf("expected EnforceMode=EnforceSoft, got %v", cfg.EnforceMode)
	}
	if cfg.GracePct != 0.20 {
		t.Errorf("expected GracePct=0.20, got %f", cfg.GracePct)
	}
}

func TestConfigFromPlan_DefaultGracePct(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-4",
		Name:             "Default Grace",
		RequestsPerMonth: 5000,
		QuotaEnforceMode: ports.QuotaEnforceHard,
		QuotaGracePct:    0, // Zero should trigger default
	}

	cfg := ConfigFromPlan(plan)

	if cfg.GracePct != 0.05 {
		t.Errorf("expected default GracePct=0.05, got %f", cfg.GracePct)
	}
}

func TestConfigFromPlan_EmptyEnforceMode(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-5",
		Name:             "No Mode",
		RequestsPerMonth: 5000,
		QuotaEnforceMode: "", // Empty should default to hard in the switch
		QuotaGracePct:    0.05,
	}

	cfg := ConfigFromPlan(plan)

	// Empty string doesn't match any case, so defaults to EnforceHard at initialization
	if cfg.EnforceMode != EnforceHard {
		t.Errorf("expected EnforceMode=EnforceHard for empty mode, got %v", cfg.EnforceMode)
	}
}

func TestConfigFromPlan_UnlimitedRequests(t *testing.T) {
	plan := ports.Plan{
		ID:               "plan-6",
		Name:             "Unlimited",
		RequestsPerMonth: -1,
		QuotaEnforceMode: ports.QuotaEnforceHard,
		QuotaGracePct:    0.05,
	}

	cfg := ConfigFromPlan(plan)

	if cfg.RequestsPerMonth != -1 {
		t.Errorf("expected RequestsPerMonth=-1, got %d", cfg.RequestsPerMonth)
	}
}

// -----------------------------------------------------------------------------
// WarningLevel String method tests
// -----------------------------------------------------------------------------

func TestWarningLevel_String_None(t *testing.T) {
	level := WarningNone
	if level.String() != "none" {
		t.Errorf("expected 'none', got %q", level.String())
	}
}

func TestWarningLevel_String_Approaching(t *testing.T) {
	level := WarningApproaching
	if level.String() != "approaching" {
		t.Errorf("expected 'approaching', got %q", level.String())
	}
}

func TestWarningLevel_String_Critical(t *testing.T) {
	level := WarningCritical
	if level.String() != "critical" {
		t.Errorf("expected 'critical', got %q", level.String())
	}
}

func TestWarningLevel_String_Exceeded(t *testing.T) {
	level := WarningExceeded
	if level.String() != "exceeded" {
		t.Errorf("expected 'exceeded', got %q", level.String())
	}
}

func TestWarningLevel_String_Unknown(t *testing.T) {
	level := WarningLevel(99) // Unknown value
	if level.String() != "unknown" {
		t.Errorf("expected 'unknown', got %q", level.String())
	}
}

// -----------------------------------------------------------------------------
// Type constant tests
// -----------------------------------------------------------------------------

func TestEnforceModeConstants(t *testing.T) {
	if EnforceHard != "hard" {
		t.Errorf("expected EnforceHard='hard', got %q", EnforceHard)
	}
	if EnforceWarn != "warn" {
		t.Errorf("expected EnforceWarn='warn', got %q", EnforceWarn)
	}
	if EnforceSoft != "soft" {
		t.Errorf("expected EnforceSoft='soft', got %q", EnforceSoft)
	}
}

func TestWarningLevelConstants(t *testing.T) {
	if WarningNone != 0 {
		t.Errorf("expected WarningNone=0, got %d", WarningNone)
	}
	if WarningApproaching != 1 {
		t.Errorf("expected WarningApproaching=1, got %d", WarningApproaching)
	}
	if WarningCritical != 2 {
		t.Errorf("expected WarningCritical=2, got %d", WarningCritical)
	}
	if WarningExceeded != 3 {
		t.Errorf("expected WarningExceeded=3, got %d", WarningExceeded)
	}
}

// -----------------------------------------------------------------------------
// Config struct tests
// -----------------------------------------------------------------------------

func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config

	if cfg.RequestsPerMonth != 0 {
		t.Errorf("expected zero value RequestsPerMonth=0, got %d", cfg.RequestsPerMonth)
	}
	if cfg.BytesPerMonth != 0 {
		t.Errorf("expected zero value BytesPerMonth=0, got %d", cfg.BytesPerMonth)
	}
	if cfg.EnforceMode != "" {
		t.Errorf("expected zero value EnforceMode='', got %q", cfg.EnforceMode)
	}
	if cfg.GracePct != 0 {
		t.Errorf("expected zero value GracePct=0, got %f", cfg.GracePct)
	}
}

// -----------------------------------------------------------------------------
// CheckResult struct tests
// -----------------------------------------------------------------------------

func TestCheckResult_ZeroValue(t *testing.T) {
	var result CheckResult

	if result.Allowed {
		t.Errorf("expected zero value Allowed=false, got true")
	}
	if result.CurrentUsage != 0 {
		t.Errorf("expected zero value CurrentUsage=0, got %d", result.CurrentUsage)
	}
	if result.Limit != 0 {
		t.Errorf("expected zero value Limit=0, got %d", result.Limit)
	}
	if result.PercentUsed != 0 {
		t.Errorf("expected zero value PercentUsed=0, got %f", result.PercentUsed)
	}
	if result.IsOverQuota {
		t.Errorf("expected zero value IsOverQuota=false, got true")
	}
	if result.OverageAmount != 0 {
		t.Errorf("expected zero value OverageAmount=0, got %d", result.OverageAmount)
	}
	if result.WarningLevel != WarningNone {
		t.Errorf("expected zero value WarningLevel=WarningNone, got %v", result.WarningLevel)
	}
	if result.Reason != "" {
		t.Errorf("expected zero value Reason='', got %q", result.Reason)
	}
}

// -----------------------------------------------------------------------------
// Edge case and boundary tests
// -----------------------------------------------------------------------------

func TestCheck_BoundaryAt80Percent(t *testing.T) {
	tests := []struct {
		name     string
		count    int64
		incr     int64
		expected WarningLevel
	}{
		{"79 percent", 78, 1, WarningNone},
		{"exactly 80 percent", 79, 1, WarningApproaching},
		{"81 percent", 80, 1, WarningApproaching},
	}

	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := ports.QuotaState{RequestCount: tt.count}
			result := Check(state, cfg, tt.incr)
			if result.WarningLevel != tt.expected {
				t.Errorf("expected WarningLevel=%v, got %v (percent=%.1f)", tt.expected, result.WarningLevel, result.PercentUsed)
			}
		})
	}
}

func TestCheck_BoundaryAt95Percent(t *testing.T) {
	tests := []struct {
		name     string
		count    int64
		incr     int64
		expected WarningLevel
	}{
		{"94 percent", 93, 1, WarningApproaching},
		{"exactly 95 percent", 94, 1, WarningCritical},
		{"96 percent", 95, 1, WarningCritical},
	}

	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := ports.QuotaState{RequestCount: tt.count}
			result := Check(state, cfg, tt.incr)
			if result.WarningLevel != tt.expected {
				t.Errorf("expected WarningLevel=%v, got %v (percent=%.1f)", tt.expected, result.WarningLevel, result.PercentUsed)
			}
		})
	}
}

func TestCheck_BoundaryAt100Percent(t *testing.T) {
	tests := []struct {
		name     string
		count    int64
		incr     int64
		expected WarningLevel
	}{
		{"99 percent", 98, 1, WarningCritical},
		{"exactly 100 percent", 99, 1, WarningCritical},
		{"101 percent", 100, 1, WarningExceeded},
	}

	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := ports.QuotaState{RequestCount: tt.count}
			result := Check(state, cfg, tt.incr)
			if result.WarningLevel != tt.expected {
				t.Errorf("expected WarningLevel=%v, got %v (percent=%.1f)", tt.expected, result.WarningLevel, result.PercentUsed)
			}
		})
	}
}

func TestCheck_NegativeIncrement(t *testing.T) {
	// Although unusual, the function should handle negative increments
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 100,
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	result := Check(state, cfg, -10)

	if result.CurrentUsage != 90 {
		t.Errorf("expected CurrentUsage=90 with negative increment, got %d", result.CurrentUsage)
	}
	if result.PercentUsed != 90.0 {
		t.Errorf("expected PercentUsed=90.0, got %f", result.PercentUsed)
	}
}

// -----------------------------------------------------------------------------
// Integration-style tests (combining multiple scenarios)
// -----------------------------------------------------------------------------

func TestCheck_FullLifecycle(t *testing.T) {
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
	}

	// Start fresh
	state := ports.QuotaState{UserID: "user-1", RequestCount: 0}

	// Make requests up to 75%
	result := Check(state, cfg, 75)
	if result.WarningLevel != WarningNone {
		t.Errorf("at 75%%: expected WarningNone, got %v", result.WarningLevel)
	}
	state.RequestCount = 75

	// Make requests to reach 85%
	result = Check(state, cfg, 10)
	if result.WarningLevel != WarningApproaching {
		t.Errorf("at 85%%: expected WarningApproaching, got %v", result.WarningLevel)
	}
	state.RequestCount = 85

	// Make requests to reach 97%
	result = Check(state, cfg, 12)
	if result.WarningLevel != WarningCritical {
		t.Errorf("at 97%%: expected WarningCritical, got %v", result.WarningLevel)
	}
	state.RequestCount = 97

	// Make requests to exceed quota
	result = Check(state, cfg, 5)
	if result.WarningLevel != WarningExceeded {
		t.Errorf("at 102%%: expected WarningExceeded, got %v", result.WarningLevel)
	}
	if !result.IsOverQuota {
		t.Errorf("at 102%%: expected IsOverQuota=true")
	}
	if !result.Allowed {
		t.Errorf("at 102%%: expected Allowed=true (within grace)")
	}
	state.RequestCount = 102

	// Make requests to exceed grace
	result = Check(state, cfg, 4)
	if result.Allowed {
		t.Errorf("at 106%%: expected Allowed=false (exceeds grace)")
	}
}

// -----------------------------------------------------------------------------
// MeterType tests - compute_units mode
// -----------------------------------------------------------------------------

func TestCheck_MeterTypeComputeUnits_Basic(t *testing.T) {
	// Test that compute_units mode uses ComputeUnits field instead of RequestCount
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 1000, // High request count should be ignored
		ComputeUnits: 50,   // This should be used
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
		MeterType:        MeterTypeComputeUnits,
	}

	result := Check(state, cfg, 10)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.CurrentUsage != 60 {
		t.Errorf("expected CurrentUsage=60 (ComputeUnits), got %d", result.CurrentUsage)
	}
	if result.PercentUsed != 60.0 {
		t.Errorf("expected PercentUsed=60.0, got %f", result.PercentUsed)
	}
}

func TestCheck_MeterTypeComputeUnits_Exceeded(t *testing.T) {
	// Test compute_units mode when quota exceeded
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 5,   // Low request count (ignored)
		ComputeUnits: 100, // At limit
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
		MeterType:        MeterTypeComputeUnits,
	}

	result := Check(state, cfg, 10)

	// 110 units exceeds grace limit of 105
	if result.Allowed {
		t.Errorf("expected Allowed=false when exceeding grace, got true")
	}
	if result.CurrentUsage != 110 {
		t.Errorf("expected CurrentUsage=110, got %d", result.CurrentUsage)
	}
	if result.Reason != "quota_exceeded" {
		t.Errorf("expected Reason='quota_exceeded', got %q", result.Reason)
	}
}

func TestCheck_MeterTypeComputeUnits_Unlimited(t *testing.T) {
	// Test unlimited quota with compute_units mode
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 100,
		ComputeUnits: 999999,
	}
	cfg := Config{
		RequestsPerMonth: -1, // Unlimited
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
		MeterType:        MeterTypeComputeUnits,
	}

	result := Check(state, cfg, 1000)

	if !result.Allowed {
		t.Errorf("expected Allowed=true for unlimited quota, got false")
	}
	if result.CurrentUsage != 1000999 {
		t.Errorf("expected CurrentUsage=1000999, got %d", result.CurrentUsage)
	}
	if result.Limit != -1 {
		t.Errorf("expected Limit=-1, got %d", result.Limit)
	}
}

func TestCheck_MeterTypeRequests_Default(t *testing.T) {
	// Test that empty/default MeterType uses RequestCount (backward compatible)
	state := ports.QuotaState{
		UserID:       "user-1",
		RequestCount: 50,
		ComputeUnits: 9999, // High compute units should be ignored
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceHard,
		GracePct:         0.05,
		// MeterType not set (defaults to empty string)
	}

	result := Check(state, cfg, 10)

	if !result.Allowed {
		t.Errorf("expected Allowed=true, got false")
	}
	if result.CurrentUsage != 60 {
		t.Errorf("expected CurrentUsage=60 (RequestCount), got %d", result.CurrentUsage)
	}
}

func TestCheck_MeterType_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		meterType    MeterType
		requestCount int64
		computeUnits float64
		limit        int64
		increment    int64
		wantAllowed  bool
		wantUsage    int64
	}{
		{
			name:         "requests_under_quota",
			meterType:    MeterTypeRequests,
			requestCount: 50,
			computeUnits: 500,
			limit:        100,
			increment:    10,
			wantAllowed:  true,
			wantUsage:    60,
		},
		{
			name:         "requests_over_quota",
			meterType:    MeterTypeRequests,
			requestCount: 100,
			computeUnits: 50,
			limit:        100,
			increment:    10,
			wantAllowed:  false, // 110 > 105 grace
			wantUsage:    110,
		},
		{
			name:         "units_under_quota",
			meterType:    MeterTypeComputeUnits,
			requestCount: 1000,
			computeUnits: 50,
			limit:        100,
			increment:    10,
			wantAllowed:  true,
			wantUsage:    60,
		},
		{
			name:         "units_over_quota",
			meterType:    MeterTypeComputeUnits,
			requestCount: 10,
			computeUnits: 100,
			limit:        100,
			increment:    10,
			wantAllowed:  false, // 110 > 105 grace
			wantUsage:    110,
		},
		{
			name:         "units_within_grace",
			meterType:    MeterTypeComputeUnits,
			requestCount: 10,
			computeUnits: 100,
			limit:        100,
			increment:    5,
			wantAllowed:  true, // 105 == 105 grace
			wantUsage:    105,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := ports.QuotaState{
				RequestCount: tt.requestCount,
				ComputeUnits: tt.computeUnits,
			}
			cfg := Config{
				RequestsPerMonth: tt.limit,
				EnforceMode:      EnforceHard,
				GracePct:         0.05,
				MeterType:        tt.meterType,
			}

			result := Check(state, cfg, tt.increment)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
			if result.CurrentUsage != tt.wantUsage {
				t.Errorf("CurrentUsage = %d, want %d", result.CurrentUsage, tt.wantUsage)
			}
		})
	}
}

func TestCheck_MeterTypeComputeUnits_SoftEnforce(t *testing.T) {
	// Test soft enforcement with compute_units - should allow and track overage
	state := ports.QuotaState{
		UserID:       "user-1",
		ComputeUnits: 200, // Already over quota
	}
	cfg := Config{
		RequestsPerMonth: 100,
		EnforceMode:      EnforceSoft,
		GracePct:         0.05,
		MeterType:        MeterTypeComputeUnits,
	}

	result := Check(state, cfg, 50)

	if !result.Allowed {
		t.Errorf("expected Allowed=true for soft enforcement, got false")
	}
	if !result.IsOverQuota {
		t.Errorf("expected IsOverQuota=true, got false")
	}
	if result.OverageAmount != 150 {
		t.Errorf("expected OverageAmount=150, got %d", result.OverageAmount)
	}
}

func TestConfigFromPlan_MeterType(t *testing.T) {
	tests := []struct {
		name          string
		planMeterType ports.MeterType
		wantMeterType MeterType
	}{
		{"requests", ports.MeterTypeRequests, MeterTypeRequests},
		{"compute_units", ports.MeterTypeComputeUnits, MeterTypeComputeUnits},
		{"empty_defaults_to_requests", "", MeterTypeRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := ports.Plan{
				RequestsPerMonth: 1000,
				MeterType:        tt.planMeterType,
			}

			cfg := ConfigFromPlan(plan)

			if cfg.MeterType != tt.wantMeterType {
				t.Errorf("MeterType = %v, want %v", cfg.MeterType, tt.wantMeterType)
			}
		})
	}
}

func TestConfigFromPlan_EstimatedCost(t *testing.T) {
	tests := []struct {
		name          string
		estimatedCost float64
		wantCost      float64
	}{
		{"zero_defaults_to_1", 0, 1.0},
		{"negative_defaults_to_1", -5, 1.0},
		{"positive_value", 10.5, 10.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := ports.Plan{
				RequestsPerMonth:    1000,
				EstimatedCostPerReq: tt.estimatedCost,
			}

			cfg := ConfigFromPlan(plan)

			if cfg.EstimatedCost != tt.wantCost {
				t.Errorf("EstimatedCost = %v, want %v", cfg.EstimatedCost, tt.wantCost)
			}
		})
	}
}

func TestMeterTypeConstants(t *testing.T) {
	if MeterTypeRequests != "requests" {
		t.Errorf("expected MeterTypeRequests='requests', got %q", MeterTypeRequests)
	}
	if MeterTypeComputeUnits != "compute_units" {
		t.Errorf("expected MeterTypeComputeUnits='compute_units', got %q", MeterTypeComputeUnits)
	}
}
