package usage_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/usage"
)

var (
	periodStart = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd   = time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
)

func TestAggregate(t *testing.T) {
	events := []usage.Event{
		{UserID: "u1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 1.0},
		{UserID: "u1", StatusCode: 200, LatencyMs: 200, RequestBytes: 600, ResponseBytes: 1200, CostMultiplier: 1.0},
		{UserID: "u1", StatusCode: 500, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 200, CostMultiplier: 2.0},
	}

	summary := usage.Aggregate(events, periodStart, periodEnd)

	if summary.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3", summary.RequestCount)
	}
	if summary.ComputeUnits != 4.0 { // 1 + 1 + 2
		t.Errorf("ComputeUnits = %f, want 4.0", summary.ComputeUnits)
	}
	if summary.BytesIn != 1200 { // 500 + 600 + 100
		t.Errorf("BytesIn = %d, want 1200", summary.BytesIn)
	}
	if summary.BytesOut != 2400 { // 1000 + 1200 + 200
		t.Errorf("BytesOut = %d, want 2400", summary.BytesOut)
	}
	if summary.ErrorCount != 1 { // One 5xx
		t.Errorf("ErrorCount = %d, want 1", summary.ErrorCount)
	}
	if summary.AvgLatencyMs != 116 { // (100+200+50)/3 = 116.67 truncated
		t.Errorf("AvgLatencyMs = %d, want 116", summary.AvgLatencyMs)
	}
}

func TestAggregate_Empty(t *testing.T) {
	summary := usage.Aggregate(nil, periodStart, periodEnd)

	if summary.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0", summary.RequestCount)
	}
	if !summary.PeriodStart.Equal(periodStart) {
		t.Errorf("PeriodStart = %v, want %v", summary.PeriodStart, periodStart)
	}
}

func TestCheckQuota(t *testing.T) {
	tests := []struct {
		name         string
		summary      usage.Summary
		quota        usage.Quota
		wantOver     bool
		wantOverage  int64
		wantPercent  float64
	}{
		{
			name:        "under quota",
			summary:     usage.Summary{RequestCount: 500},
			quota:       usage.Quota{RequestsPerMonth: 1000},
			wantOver:    false,
			wantPercent: 50.0,
		},
		{
			name:        "at quota",
			summary:     usage.Summary{RequestCount: 1000},
			quota:       usage.Quota{RequestsPerMonth: 1000},
			wantOver:    false,
			wantPercent: 100.0,
		},
		{
			name:        "over quota",
			summary:     usage.Summary{RequestCount: 1500},
			quota:       usage.Quota{RequestsPerMonth: 1000},
			wantOver:    true,
			wantOverage: 500,
			wantPercent: 150.0,
		},
		{
			name:        "unlimited quota",
			summary:     usage.Summary{RequestCount: 999999},
			quota:       usage.Quota{RequestsPerMonth: -1},
			wantOver:    false,
			wantPercent: 0, // No limit to compare against
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := usage.CheckQuota(tt.summary, tt.quota)

			if status.IsOverQuota != tt.wantOver {
				t.Errorf("IsOverQuota = %v, want %v", status.IsOverQuota, tt.wantOver)
			}
			if status.OverageRequests != tt.wantOverage {
				t.Errorf("OverageRequests = %d, want %d", status.OverageRequests, tt.wantOverage)
			}
			if status.RequestsPercent != tt.wantPercent {
				t.Errorf("RequestsPercent = %f, want %f", status.RequestsPercent, tt.wantPercent)
			}
		})
	}
}

func TestCalculateOverage(t *testing.T) {
	tests := []struct {
		name     string
		usage    int64
		included int64
		price    int64
		want     int64
	}{
		{"under limit", 500, 1000, 10, 0},
		{"at limit", 1000, 1000, 10, 0},
		{"over limit", 1500, 1000, 10, 5000}, // 500 * 10
		{"unlimited", 99999, -1, 10, 0},      // -1 = unlimited
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usage.CalculateOverage(tt.usage, tt.included, tt.price)
			if got != tt.want {
				t.Errorf("CalculateOverage() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPeriodBounds(t *testing.T) {
	// Test middle of month
	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	start, end := usage.PeriodBounds(ts)

	wantStart := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2024, 3, 31, 23, 59, 59, 999999999, time.UTC)

	if !start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", end, wantEnd)
	}
}

// Benchmark aggregation
func BenchmarkAggregate(b *testing.B) {
	events := make([]usage.Event, 1000)
	for i := range events {
		events[i] = usage.Event{
			UserID:         "user-1",
			StatusCode:     200,
			LatencyMs:      100,
			RequestBytes:   500,
			ResponseBytes:  1000,
			CostMultiplier: 1.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage.Aggregate(events, periodStart, periodEnd)
	}
}
