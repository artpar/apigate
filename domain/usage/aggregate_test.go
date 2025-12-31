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

func TestAggregate_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		events           []usage.Event
		wantRequestCount int64
		wantComputeUnits float64
		wantBytesIn      int64
		wantBytesOut     int64
		wantErrorCount   int64
		wantAvgLatency   int64
		wantUserID       string
	}{
		{
			name:             "empty events",
			events:           []usage.Event{},
			wantRequestCount: 0,
			wantComputeUnits: 0,
			wantBytesIn:      0,
			wantBytesOut:     0,
			wantErrorCount:   0,
			wantAvgLatency:   0,
			wantUserID:       "",
		},
		{
			name: "single event",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 1.5},
			},
			wantRequestCount: 1,
			wantComputeUnits: 1.5,
			wantBytesIn:      500,
			wantBytesOut:     1000,
			wantErrorCount:   0,
			wantAvgLatency:   100,
			wantUserID:       "user1",
		},
		{
			name: "4xx errors counted",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 400, LatencyMs: 10, RequestBytes: 100, ResponseBytes: 50, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 401, LatencyMs: 10, RequestBytes: 100, ResponseBytes: 50, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 403, LatencyMs: 10, RequestBytes: 100, ResponseBytes: 50, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 404, LatencyMs: 10, RequestBytes: 100, ResponseBytes: 50, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 429, LatencyMs: 10, RequestBytes: 100, ResponseBytes: 50, CostMultiplier: 1.0},
			},
			wantRequestCount: 5,
			wantComputeUnits: 5.0,
			wantBytesIn:      500,
			wantBytesOut:     250,
			wantErrorCount:   5, // All 4xx are errors
			wantAvgLatency:   10,
			wantUserID:       "user1",
		},
		{
			name: "5xx errors counted",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 500, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 502, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 503, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
			},
			wantRequestCount: 3,
			wantComputeUnits: 3.0,
			wantBytesIn:      300,
			wantBytesOut:     300,
			wantErrorCount:   3,
			wantAvgLatency:   50,
			wantUserID:       "user1",
		},
		{
			name: "mixed success and errors",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 201, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 204, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 0, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 301, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 302, LatencyMs: 50, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 400, LatencyMs: 20, RequestBytes: 50, ResponseBytes: 50, CostMultiplier: 1.0},
				{UserID: "user1", StatusCode: 500, LatencyMs: 20, RequestBytes: 50, ResponseBytes: 50, CostMultiplier: 1.0},
			},
			wantRequestCount: 7,
			wantComputeUnits: 7.0,
			wantBytesIn:      1800, // 500+500+500+100+100+50+50
			wantBytesOut:     2300, // 1000+1000+0+100+100+50+50
			wantErrorCount:   2,    // Only 400 and 500
			wantAvgLatency:   62,   // (100+100+100+50+50+20+20)/7 = 440/7 = 62
			wantUserID:       "user1",
		},
		{
			name: "zero cost multipliers",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 0},
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 0},
			},
			wantRequestCount: 2,
			wantComputeUnits: 0,
			wantBytesIn:      1000,
			wantBytesOut:     2000,
			wantErrorCount:   0,
			wantAvgLatency:   100,
			wantUserID:       "user1",
		},
		{
			name: "high cost multipliers",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 10.5},
				{UserID: "user1", StatusCode: 200, LatencyMs: 100, RequestBytes: 500, ResponseBytes: 1000, CostMultiplier: 5.5},
			},
			wantRequestCount: 2,
			wantComputeUnits: 16.0,
			wantBytesIn:      1000,
			wantBytesOut:     2000,
			wantErrorCount:   0,
			wantAvgLatency:   100,
			wantUserID:       "user1",
		},
		{
			name: "zero bytes",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 204, LatencyMs: 5, RequestBytes: 0, ResponseBytes: 0, CostMultiplier: 1.0},
			},
			wantRequestCount: 1,
			wantComputeUnits: 1.0,
			wantBytesIn:      0,
			wantBytesOut:     0,
			wantErrorCount:   0,
			wantAvgLatency:   5,
			wantUserID:       "user1",
		},
		{
			name: "zero latency",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 0, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
			},
			wantRequestCount: 1,
			wantComputeUnits: 1.0,
			wantBytesIn:      100,
			wantBytesOut:     100,
			wantErrorCount:   0,
			wantAvgLatency:   0,
			wantUserID:       "user1",
		},
		{
			name: "boundary status code 399 not error",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 399, LatencyMs: 100, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
			},
			wantRequestCount: 1,
			wantComputeUnits: 1.0,
			wantBytesIn:      100,
			wantBytesOut:     100,
			wantErrorCount:   0, // 399 is not an error
			wantAvgLatency:   100,
			wantUserID:       "user1",
		},
		{
			name: "boundary status code 400 is error",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 400, LatencyMs: 100, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
			},
			wantRequestCount: 1,
			wantComputeUnits: 1.0,
			wantBytesIn:      100,
			wantBytesOut:     100,
			wantErrorCount:   1, // 400 is an error
			wantAvgLatency:   100,
			wantUserID:       "user1",
		},
		{
			name: "multiple users takes first",
			events: []usage.Event{
				{UserID: "first_user", StatusCode: 200, LatencyMs: 100, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
				{UserID: "second_user", StatusCode: 200, LatencyMs: 100, RequestBytes: 100, ResponseBytes: 100, CostMultiplier: 1.0},
			},
			wantRequestCount: 2,
			wantComputeUnits: 2.0,
			wantBytesIn:      200,
			wantBytesOut:     200,
			wantErrorCount:   0,
			wantAvgLatency:   100,
			wantUserID:       "first_user",
		},
		{
			name: "large values",
			events: []usage.Event{
				{UserID: "user1", StatusCode: 200, LatencyMs: 1000000, RequestBytes: 1000000000, ResponseBytes: 2000000000, CostMultiplier: 100.0},
			},
			wantRequestCount: 1,
			wantComputeUnits: 100.0,
			wantBytesIn:      1000000000,
			wantBytesOut:     2000000000,
			wantErrorCount:   0,
			wantAvgLatency:   1000000,
			wantUserID:       "user1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := usage.Aggregate(tt.events, periodStart, periodEnd)

			if summary.RequestCount != tt.wantRequestCount {
				t.Errorf("RequestCount = %d, want %d", summary.RequestCount, tt.wantRequestCount)
			}
			if summary.ComputeUnits != tt.wantComputeUnits {
				t.Errorf("ComputeUnits = %f, want %f", summary.ComputeUnits, tt.wantComputeUnits)
			}
			if summary.BytesIn != tt.wantBytesIn {
				t.Errorf("BytesIn = %d, want %d", summary.BytesIn, tt.wantBytesIn)
			}
			if summary.BytesOut != tt.wantBytesOut {
				t.Errorf("BytesOut = %d, want %d", summary.BytesOut, tt.wantBytesOut)
			}
			if summary.ErrorCount != tt.wantErrorCount {
				t.Errorf("ErrorCount = %d, want %d", summary.ErrorCount, tt.wantErrorCount)
			}
			if summary.AvgLatencyMs != tt.wantAvgLatency {
				t.Errorf("AvgLatencyMs = %d, want %d", summary.AvgLatencyMs, tt.wantAvgLatency)
			}
			if summary.UserID != tt.wantUserID {
				t.Errorf("UserID = %s, want %s", summary.UserID, tt.wantUserID)
			}
			if !summary.PeriodStart.Equal(periodStart) {
				t.Errorf("PeriodStart = %v, want %v", summary.PeriodStart, periodStart)
			}
			if !summary.PeriodEnd.Equal(periodEnd) {
				t.Errorf("PeriodEnd = %v, want %v", summary.PeriodEnd, periodEnd)
			}
		})
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

// TestMergeSummaries tests the MergeSummaries function
func TestMergeSummaries(t *testing.T) {
	t.Run("empty summaries", func(t *testing.T) {
		result := usage.MergeSummaries()
		if result.RequestCount != 0 {
			t.Errorf("RequestCount = %d, want 0", result.RequestCount)
		}
	})

	t.Run("single summary", func(t *testing.T) {
		s := usage.Summary{
			UserID:       "user1",
			PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			RequestCount: 100,
			ComputeUnits: 50.5,
			BytesIn:      1000,
			BytesOut:     2000,
			ErrorCount:   5,
			AvgLatencyMs: 150,
		}
		result := usage.MergeSummaries(s)

		if result.RequestCount != 100 {
			t.Errorf("RequestCount = %d, want 100", result.RequestCount)
		}
		if result.ComputeUnits != 50.5 {
			t.Errorf("ComputeUnits = %f, want 50.5", result.ComputeUnits)
		}
		if result.BytesIn != 1000 {
			t.Errorf("BytesIn = %d, want 1000", result.BytesIn)
		}
		if result.BytesOut != 2000 {
			t.Errorf("BytesOut = %d, want 2000", result.BytesOut)
		}
		if result.ErrorCount != 5 {
			t.Errorf("ErrorCount = %d, want 5", result.ErrorCount)
		}
		if result.AvgLatencyMs != 150 {
			t.Errorf("AvgLatencyMs = %d, want 150", result.AvgLatencyMs)
		}
	})

	t.Run("two summaries", func(t *testing.T) {
		s1 := usage.Summary{
			UserID:       "user1",
			PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
			RequestCount: 100,
			ComputeUnits: 50.0,
			BytesIn:      1000,
			BytesOut:     2000,
			ErrorCount:   5,
			AvgLatencyMs: 100,
		}
		s2 := usage.Summary{
			UserID:       "user1",
			PeriodStart:  time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			RequestCount: 200,
			ComputeUnits: 100.0,
			BytesIn:      2000,
			BytesOut:     4000,
			ErrorCount:   10,
			AvgLatencyMs: 200,
		}
		result := usage.MergeSummaries(s1, s2)

		if result.RequestCount != 300 {
			t.Errorf("RequestCount = %d, want 300", result.RequestCount)
		}
		if result.ComputeUnits != 150.0 {
			t.Errorf("ComputeUnits = %f, want 150.0", result.ComputeUnits)
		}
		if result.BytesIn != 3000 {
			t.Errorf("BytesIn = %d, want 3000", result.BytesIn)
		}
		if result.BytesOut != 6000 {
			t.Errorf("BytesOut = %d, want 6000", result.BytesOut)
		}
		if result.ErrorCount != 15 {
			t.Errorf("ErrorCount = %d, want 15", result.ErrorCount)
		}
		// Period should expand to cover both
		if !result.PeriodStart.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("PeriodStart = %v, want 2024-01-01", result.PeriodStart)
		}
		if !result.PeriodEnd.Equal(time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)) {
			t.Errorf("PeriodEnd = %v, want 2024-01-31", result.PeriodEnd)
		}
	})

	t.Run("multiple summaries", func(t *testing.T) {
		summaries := []usage.Summary{
			{
				PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				RequestCount: 100,
				ComputeUnits: 25.0,
				BytesIn:      500,
				BytesOut:     1000,
				ErrorCount:   2,
				AvgLatencyMs: 50,
			},
			{
				PeriodStart:  time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
				RequestCount: 200,
				ComputeUnits: 50.0,
				BytesIn:      1000,
				BytesOut:     2000,
				ErrorCount:   4,
				AvgLatencyMs: 100,
			},
			{
				PeriodStart:  time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
				RequestCount: 300,
				ComputeUnits: 75.0,
				BytesIn:      1500,
				BytesOut:     3000,
				ErrorCount:   6,
				AvgLatencyMs: 150,
			},
		}
		result := usage.MergeSummaries(summaries...)

		if result.RequestCount != 600 {
			t.Errorf("RequestCount = %d, want 600", result.RequestCount)
		}
		if result.ComputeUnits != 150.0 {
			t.Errorf("ComputeUnits = %f, want 150.0", result.ComputeUnits)
		}
		if result.BytesIn != 3000 {
			t.Errorf("BytesIn = %d, want 3000", result.BytesIn)
		}
		if result.BytesOut != 6000 {
			t.Errorf("BytesOut = %d, want 6000", result.BytesOut)
		}
		if result.ErrorCount != 12 {
			t.Errorf("ErrorCount = %d, want 12", result.ErrorCount)
		}
	})

	t.Run("period bounds expansion", func(t *testing.T) {
		// Second summary has earlier start
		s1 := usage.Summary{
			PeriodStart:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			RequestCount: 100,
		}
		s2 := usage.Summary{
			PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
			RequestCount: 100,
		}
		result := usage.MergeSummaries(s1, s2)

		// Should expand to earliest start
		if !result.PeriodStart.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("PeriodStart = %v, want 2024-01-01", result.PeriodStart)
		}
		// Should expand to latest end
		if !result.PeriodEnd.Equal(time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("PeriodEnd = %v, want 2024-03-31", result.PeriodEnd)
		}
	})

	t.Run("zero request count summaries", func(t *testing.T) {
		s1 := usage.Summary{
			PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			RequestCount: 0,
			AvgLatencyMs: 0,
		}
		s2 := usage.Summary{
			PeriodStart:  time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			RequestCount: 0,
			AvgLatencyMs: 0,
		}
		result := usage.MergeSummaries(s1, s2)

		if result.RequestCount != 0 {
			t.Errorf("RequestCount = %d, want 0", result.RequestCount)
		}
	})
}

func TestCheckQuota(t *testing.T) {
	tests := []struct {
		name            string
		summary         usage.Summary
		quota           usage.Quota
		wantOver        bool
		wantOverage     int64
		wantReqPercent  float64
		wantBytePercent float64
	}{
		{
			name:           "under quota",
			summary:        usage.Summary{RequestCount: 500},
			quota:          usage.Quota{RequestsPerMonth: 1000},
			wantOver:       false,
			wantReqPercent: 50.0,
		},
		{
			name:           "at quota",
			summary:        usage.Summary{RequestCount: 1000},
			quota:          usage.Quota{RequestsPerMonth: 1000},
			wantOver:       false,
			wantReqPercent: 100.0,
		},
		{
			name:           "over quota",
			summary:        usage.Summary{RequestCount: 1500},
			quota:          usage.Quota{RequestsPerMonth: 1000},
			wantOver:       true,
			wantOverage:    500,
			wantReqPercent: 150.0,
		},
		{
			name:           "unlimited quota",
			summary:        usage.Summary{RequestCount: 999999},
			quota:          usage.Quota{RequestsPerMonth: -1},
			wantOver:       false,
			wantReqPercent: 0, // No limit to compare against
		},
		{
			name:            "bytes under quota",
			summary:         usage.Summary{BytesIn: 500, BytesOut: 500},
			quota:           usage.Quota{BytesPerMonth: 2000},
			wantOver:        false,
			wantBytePercent: 50.0, // 1000 / 2000 * 100
		},
		{
			name:            "bytes at quota",
			summary:         usage.Summary{BytesIn: 1000, BytesOut: 1000},
			quota:           usage.Quota{BytesPerMonth: 2000},
			wantOver:        false,
			wantBytePercent: 100.0,
		},
		{
			name:            "bytes over quota",
			summary:         usage.Summary{BytesIn: 1500, BytesOut: 1500},
			quota:           usage.Quota{BytesPerMonth: 2000},
			wantOver:        true,
			wantBytePercent: 150.0, // 3000 / 2000 * 100
		},
		{
			name:            "bytes unlimited",
			summary:         usage.Summary{BytesIn: 999999, BytesOut: 999999},
			quota:           usage.Quota{BytesPerMonth: 0}, // 0 = unlimited
			wantOver:        false,
			wantBytePercent: 0,
		},
		{
			name:            "both requests and bytes over quota",
			summary:         usage.Summary{RequestCount: 1500, BytesIn: 1000, BytesOut: 2000},
			quota:           usage.Quota{RequestsPerMonth: 1000, BytesPerMonth: 2000},
			wantOver:        true,
			wantOverage:     500,
			wantReqPercent:  150.0,
			wantBytePercent: 150.0, // 3000 / 2000 * 100
		},
		{
			name:            "requests under but bytes over quota",
			summary:         usage.Summary{RequestCount: 500, BytesIn: 2000, BytesOut: 2000},
			quota:           usage.Quota{RequestsPerMonth: 1000, BytesPerMonth: 2000},
			wantOver:        true,
			wantOverage:     0,
			wantReqPercent:  50.0,
			wantBytePercent: 200.0,
		},
		{
			name:            "requests over but bytes under quota",
			summary:         usage.Summary{RequestCount: 1500, BytesIn: 500, BytesOut: 500},
			quota:           usage.Quota{RequestsPerMonth: 1000, BytesPerMonth: 2000},
			wantOver:        true,
			wantOverage:     500,
			wantReqPercent:  150.0,
			wantBytePercent: 50.0,
		},
		{
			name:           "zero quota limits",
			summary:        usage.Summary{RequestCount: 100, BytesIn: 100, BytesOut: 100},
			quota:          usage.Quota{RequestsPerMonth: 0, BytesPerMonth: 0},
			wantOver:       false,
			wantReqPercent: 0,
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
			if status.RequestsPercent != tt.wantReqPercent {
				t.Errorf("RequestsPercent = %f, want %f", status.RequestsPercent, tt.wantReqPercent)
			}
			if status.BytesPercent != tt.wantBytePercent {
				t.Errorf("BytesPercent = %f, want %f", status.BytesPercent, tt.wantBytePercent)
			}
			// Verify that RequestsUsed and BytesUsed are correctly set
			if status.RequestsUsed != tt.summary.RequestCount {
				t.Errorf("RequestsUsed = %d, want %d", status.RequestsUsed, tt.summary.RequestCount)
			}
			expectedBytes := tt.summary.BytesIn + tt.summary.BytesOut
			if status.BytesUsed != expectedBytes {
				t.Errorf("BytesUsed = %d, want %d", status.BytesUsed, expectedBytes)
			}
			if status.RequestsLimit != tt.quota.RequestsPerMonth {
				t.Errorf("RequestsLimit = %d, want %d", status.RequestsLimit, tt.quota.RequestsPerMonth)
			}
			if status.BytesLimit != tt.quota.BytesPerMonth {
				t.Errorf("BytesLimit = %d, want %d", status.BytesLimit, tt.quota.BytesPerMonth)
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
		{"zero usage", 0, 1000, 10, 0},
		{"zero included", 100, 0, 10, 1000}, // 100 * 10
		{"zero price", 1500, 1000, 0, 0},    // 500 * 0 = 0
		{"negative included means unlimited", 100, -100, 10, 0},
		{"large overage", 1000000, 100000, 1, 900000},           // 900000 * 1
		{"high price per unit", 1100, 1000, 1000, 100000},       // 100 * 1000
		{"exactly one over", 1001, 1000, 10, 10},                // 1 * 10
		{"large values", 9999999999, 1000000000, 1, 8999999999}, // Large overflow check
		{"usage equals included minus one", 999, 1000, 10, 0},
		{"usage equals included plus one", 1001, 1000, 10, 10},
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
	tests := []struct {
		name      string
		input     time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "middle of month",
			input:     time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
			wantStart: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "first day of month",
			input:     time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "last day of month",
			input:     time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
			wantStart: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "January",
			input:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 1, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "February non-leap year",
			input:     time.Date(2023, 2, 15, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2023, 2, 28, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "February leap year",
			input:     time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 2, 29, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "December",
			input:     time.Date(2024, 12, 25, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "April 30 days",
			input:     time.Date(2024, 4, 15, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 4, 30, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "year boundary December to January",
			input:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			wantStart: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:      "different timezone - PST",
			input:     time.Date(2024, 6, 15, 12, 0, 0, 0, time.FixedZone("PST", -8*60*60)),
			wantStart: time.Date(2024, 6, 1, 0, 0, 0, 0, time.FixedZone("PST", -8*60*60)),
			wantEnd:   time.Date(2024, 6, 30, 23, 59, 59, 999999999, time.FixedZone("PST", -8*60*60)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := usage.PeriodBounds(tt.input)

			if !start.Equal(tt.wantStart) {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if !end.Equal(tt.wantEnd) {
				t.Errorf("end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

// TestEventFields tests that Event struct fields are properly populated
func TestEventFields(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	event := usage.Event{
		ID:             "evt-123",
		KeyID:          "key-456",
		UserID:         "user-789",
		Method:         "POST",
		Path:           "/api/v1/users",
		StatusCode:     201,
		LatencyMs:      150,
		RequestBytes:   1024,
		ResponseBytes:  2048,
		CostMultiplier: 1.5,
		IPAddress:      "192.168.1.1",
		UserAgent:      "Mozilla/5.0",
		Timestamp:      ts,
	}

	if event.ID != "evt-123" {
		t.Errorf("ID = %s, want evt-123", event.ID)
	}
	if event.KeyID != "key-456" {
		t.Errorf("KeyID = %s, want key-456", event.KeyID)
	}
	if event.UserID != "user-789" {
		t.Errorf("UserID = %s, want user-789", event.UserID)
	}
	if event.Method != "POST" {
		t.Errorf("Method = %s, want POST", event.Method)
	}
	if event.Path != "/api/v1/users" {
		t.Errorf("Path = %s, want /api/v1/users", event.Path)
	}
	if event.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", event.StatusCode)
	}
	if event.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d, want 150", event.LatencyMs)
	}
	if event.RequestBytes != 1024 {
		t.Errorf("RequestBytes = %d, want 1024", event.RequestBytes)
	}
	if event.ResponseBytes != 2048 {
		t.Errorf("ResponseBytes = %d, want 2048", event.ResponseBytes)
	}
	if event.CostMultiplier != 1.5 {
		t.Errorf("CostMultiplier = %f, want 1.5", event.CostMultiplier)
	}
	if event.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %s, want 192.168.1.1", event.IPAddress)
	}
	if event.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %s, want Mozilla/5.0", event.UserAgent)
	}
	if !event.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", event.Timestamp, ts)
	}
}

// TestSummaryFields tests that Summary struct fields are properly populated
func TestSummaryFields(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
	summary := usage.Summary{
		UserID:       "user-123",
		PeriodStart:  start,
		PeriodEnd:    end,
		RequestCount: 1000,
		ComputeUnits: 500.5,
		BytesIn:      1024000,
		BytesOut:     2048000,
		ErrorCount:   50,
		AvgLatencyMs: 125,
	}

	if summary.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", summary.UserID)
	}
	if !summary.PeriodStart.Equal(start) {
		t.Errorf("PeriodStart = %v, want %v", summary.PeriodStart, start)
	}
	if !summary.PeriodEnd.Equal(end) {
		t.Errorf("PeriodEnd = %v, want %v", summary.PeriodEnd, end)
	}
	if summary.RequestCount != 1000 {
		t.Errorf("RequestCount = %d, want 1000", summary.RequestCount)
	}
	if summary.ComputeUnits != 500.5 {
		t.Errorf("ComputeUnits = %f, want 500.5", summary.ComputeUnits)
	}
	if summary.BytesIn != 1024000 {
		t.Errorf("BytesIn = %d, want 1024000", summary.BytesIn)
	}
	if summary.BytesOut != 2048000 {
		t.Errorf("BytesOut = %d, want 2048000", summary.BytesOut)
	}
	if summary.ErrorCount != 50 {
		t.Errorf("ErrorCount = %d, want 50", summary.ErrorCount)
	}
	if summary.AvgLatencyMs != 125 {
		t.Errorf("AvgLatencyMs = %d, want 125", summary.AvgLatencyMs)
	}
}

// TestQuotaFields tests that Quota struct fields are properly populated
func TestQuotaFields(t *testing.T) {
	quota := usage.Quota{
		RequestsPerMonth: 100000,
		BytesPerMonth:    1073741824, // 1GB
	}

	if quota.RequestsPerMonth != 100000 {
		t.Errorf("RequestsPerMonth = %d, want 100000", quota.RequestsPerMonth)
	}
	if quota.BytesPerMonth != 1073741824 {
		t.Errorf("BytesPerMonth = %d, want 1073741824", quota.BytesPerMonth)
	}
}

// TestQuotaStatusFields tests that QuotaStatus struct fields are properly populated
func TestQuotaStatusFields(t *testing.T) {
	status := usage.QuotaStatus{
		RequestsUsed:    50000,
		RequestsLimit:   100000,
		RequestsPercent: 50.0,
		BytesUsed:       536870912,
		BytesLimit:      1073741824,
		BytesPercent:    50.0,
		IsOverQuota:     false,
		OverageRequests: 0,
	}

	if status.RequestsUsed != 50000 {
		t.Errorf("RequestsUsed = %d, want 50000", status.RequestsUsed)
	}
	if status.RequestsLimit != 100000 {
		t.Errorf("RequestsLimit = %d, want 100000", status.RequestsLimit)
	}
	if status.RequestsPercent != 50.0 {
		t.Errorf("RequestsPercent = %f, want 50.0", status.RequestsPercent)
	}
	if status.BytesUsed != 536870912 {
		t.Errorf("BytesUsed = %d, want 536870912", status.BytesUsed)
	}
	if status.BytesLimit != 1073741824 {
		t.Errorf("BytesLimit = %d, want 1073741824", status.BytesLimit)
	}
	if status.BytesPercent != 50.0 {
		t.Errorf("BytesPercent = %f, want 50.0", status.BytesPercent)
	}
	if status.IsOverQuota != false {
		t.Errorf("IsOverQuota = %v, want false", status.IsOverQuota)
	}
	if status.OverageRequests != 0 {
		t.Errorf("OverageRequests = %d, want 0", status.OverageRequests)
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

// BenchmarkMergeSummaries benchmarks the MergeSummaries function
func BenchmarkMergeSummaries(b *testing.B) {
	summaries := make([]usage.Summary, 100)
	for i := range summaries {
		summaries[i] = usage.Summary{
			UserID:       "user-1",
			PeriodStart:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:    time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			RequestCount: 100,
			ComputeUnits: 50.0,
			BytesIn:      1000,
			BytesOut:     2000,
			ErrorCount:   5,
			AvgLatencyMs: 100,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage.MergeSummaries(summaries...)
	}
}

// BenchmarkCheckQuota benchmarks the CheckQuota function
func BenchmarkCheckQuota(b *testing.B) {
	summary := usage.Summary{
		RequestCount: 50000,
		BytesIn:      500000000,
		BytesOut:     500000000,
	}
	quota := usage.Quota{
		RequestsPerMonth: 100000,
		BytesPerMonth:    1000000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage.CheckQuota(summary, quota)
	}
}

// BenchmarkCalculateOverage benchmarks the CalculateOverage function
func BenchmarkCalculateOverage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage.CalculateOverage(1500, 1000, 10)
	}
}

// BenchmarkPeriodBounds benchmarks the PeriodBounds function
func BenchmarkPeriodBounds(b *testing.B) {
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage.PeriodBounds(ts)
	}
}
