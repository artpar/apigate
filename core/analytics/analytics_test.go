package analytics

import (
	"math"
	"testing"
	"time"
)

func TestDefaultCostCalculator_Calculate(t *testing.T) {
	tests := []struct {
		name     string
		calc     *DefaultCostCalculator
		event    Event
		expected float64
	}{
		{
			name: "zero values returns base cost only",
			calc: &DefaultCostCalculator{
				BaseRequestCost:   0.001,
				CPUCostPerUS:      0.00001,
				MemoryCostPerKB:   0.000001,
				TransferCostPerKB: 0.00001,
			},
			event:    Event{},
			expected: 0.001,
		},
		{
			name: "calculates CPU cost",
			calc: &DefaultCostCalculator{
				BaseRequestCost:   0.001,
				CPUCostPerUS:      0.01,
				MemoryCostPerKB:   0.0,
				TransferCostPerKB: 0.0,
			},
			event: Event{
				DurationNS: 1000000, // 1ms = 1000us
			},
			expected: 0.001 + (1000 * 0.01), // base + cpu cost
		},
		{
			name: "calculates memory cost",
			calc: &DefaultCostCalculator{
				BaseRequestCost:   0.001,
				CPUCostPerUS:      0.0,
				MemoryCostPerKB:   0.000001,
				TransferCostPerKB: 0.0,
			},
			event: Event{
				MemoryBytes: 2048, // 2KB
			},
			expected: 0.001 + (2 * 0.000001), // base + memory cost
		},
		{
			name: "calculates transfer cost",
			calc: &DefaultCostCalculator{
				BaseRequestCost:   0.001,
				CPUCostPerUS:      0.0,
				MemoryCostPerKB:   0.0,
				TransferCostPerKB: 0.00001,
			},
			event: Event{
				RequestBytes:  1024, // 1KB
				ResponseBytes: 2048, // 2KB
			},
			expected: 0.001 + (3 * 0.00001), // base + transfer cost (3KB total)
		},
		{
			name: "calculates combined cost",
			calc: &DefaultCostCalculator{
				BaseRequestCost:   0.001,
				CPUCostPerUS:      0.00001,
				MemoryCostPerKB:   0.000001,
				TransferCostPerKB: 0.00001,
			},
			event: Event{
				DurationNS:    2000000, // 2ms = 2000us
				MemoryBytes:   4096,    // 4KB
				RequestBytes:  1024,    // 1KB
				ResponseBytes: 2048,    // 2KB
			},
			expected: 0.001 + (2000 * 0.00001) + (4 * 0.000001) + (3 * 0.00001),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.calc.Calculate(tt.event)
			if result != tt.expected {
				t.Errorf("Calculate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewDefaultCostCalculator(t *testing.T) {
	calc := NewDefaultCostCalculator()

	if calc == nil {
		t.Fatal("NewDefaultCostCalculator() returned nil")
	}

	if calc.BaseRequestCost != 0.001 {
		t.Errorf("BaseRequestCost = %v, want %v", calc.BaseRequestCost, 0.001)
	}

	if calc.CPUCostPerUS != 0.00001 {
		t.Errorf("CPUCostPerUS = %v, want %v", calc.CPUCostPerUS, 0.00001)
	}

	if calc.MemoryCostPerKB != 0.000001 {
		t.Errorf("MemoryCostPerKB = %v, want %v", calc.MemoryCostPerKB, 0.000001)
	}

	if calc.TransferCostPerKB != 0.00001 {
		t.Errorf("TransferCostPerKB = %v, want %v", calc.TransferCostPerKB, 0.00001)
	}
}

func TestDefaultCostCalculator_CalculateWithDefaultValues(t *testing.T) {
	calc := NewDefaultCostCalculator()

	event := Event{
		DurationNS:    1000000,  // 1ms
		MemoryBytes:   1024,     // 1KB
		RequestBytes:  512,      // 0.5KB
		ResponseBytes: 512,      // 0.5KB
	}

	result := calc.Calculate(event)

	// base: 0.001
	// cpu: (1000000/1000) * 0.00001 = 1000 * 0.00001 = 0.01
	// memory: (1024/1024) * 0.000001 = 1 * 0.000001 = 0.000001
	// transfer: ((512+512)/1024) * 0.00001 = 1 * 0.00001 = 0.00001
	expected := 0.001 + 0.01 + 0.000001 + 0.00001

	// Use tolerance for float comparison due to floating point precision
	if math.Abs(result-expected) > 1e-12 {
		t.Errorf("Calculate() = %v, want %v", result, expected)
	}
}

func TestEvent_Fields(t *testing.T) {
	now := time.Now()
	event := Event{
		ID:            "test-id-123",
		Timestamp:     now,
		Channel:       "http",
		Module:        "user",
		Action:        "get",
		RecordID:      "record-1",
		UserID:        "user-1",
		APIKeyID:      "key-1",
		RemoteIP:      "192.168.1.1",
		DurationNS:    1000000,
		MemoryBytes:   2048,
		RequestBytes:  512,
		ResponseBytes: 1024,
		Success:       true,
		StatusCode:    200,
		Error:         "",
	}

	if event.ID != "test-id-123" {
		t.Errorf("ID = %v, want %v", event.ID, "test-id-123")
	}
	if event.Timestamp != now {
		t.Errorf("Timestamp = %v, want %v", event.Timestamp, now)
	}
	if event.Channel != "http" {
		t.Errorf("Channel = %v, want %v", event.Channel, "http")
	}
	if event.Module != "user" {
		t.Errorf("Module = %v, want %v", event.Module, "user")
	}
	if event.Action != "get" {
		t.Errorf("Action = %v, want %v", event.Action, "get")
	}
	if event.RecordID != "record-1" {
		t.Errorf("RecordID = %v, want %v", event.RecordID, "record-1")
	}
	if event.UserID != "user-1" {
		t.Errorf("UserID = %v, want %v", event.UserID, "user-1")
	}
	if event.APIKeyID != "key-1" {
		t.Errorf("APIKeyID = %v, want %v", event.APIKeyID, "key-1")
	}
	if event.RemoteIP != "192.168.1.1" {
		t.Errorf("RemoteIP = %v, want %v", event.RemoteIP, "192.168.1.1")
	}
	if event.DurationNS != 1000000 {
		t.Errorf("DurationNS = %v, want %v", event.DurationNS, 1000000)
	}
	if event.MemoryBytes != 2048 {
		t.Errorf("MemoryBytes = %v, want %v", event.MemoryBytes, 2048)
	}
	if event.RequestBytes != 512 {
		t.Errorf("RequestBytes = %v, want %v", event.RequestBytes, 512)
	}
	if event.ResponseBytes != 1024 {
		t.Errorf("ResponseBytes = %v, want %v", event.ResponseBytes, 1024)
	}
	if !event.Success {
		t.Errorf("Success = %v, want %v", event.Success, true)
	}
	if event.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want %v", event.StatusCode, 200)
	}
	if event.Error != "" {
		t.Errorf("Error = %v, want empty", event.Error)
	}
}

func TestSummary_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Hour)
	summary := Summary{
		Channel:            "http",
		Module:             "user",
		Action:             "list",
		Period:             "hour",
		Start:              start,
		End:                end,
		TotalRequests:      100,
		SuccessRequests:    95,
		ErrorRequests:      5,
		AvgDurationNS:      1000000,
		MinDurationNS:      500000,
		MaxDurationNS:      2000000,
		P50DurationNS:      900000,
		P95DurationNS:      1800000,
		P99DurationNS:      1950000,
		TotalMemoryBytes:   102400,
		TotalRequestBytes:  51200,
		TotalResponseBytes: 204800,
		CostUnits:          0.5,
	}

	if summary.Channel != "http" {
		t.Errorf("Channel = %v, want %v", summary.Channel, "http")
	}
	if summary.Module != "user" {
		t.Errorf("Module = %v, want %v", summary.Module, "user")
	}
	if summary.Action != "list" {
		t.Errorf("Action = %v, want %v", summary.Action, "list")
	}
	if summary.Period != "hour" {
		t.Errorf("Period = %v, want %v", summary.Period, "hour")
	}
	if summary.TotalRequests != 100 {
		t.Errorf("TotalRequests = %v, want %v", summary.TotalRequests, 100)
	}
	if summary.SuccessRequests != 95 {
		t.Errorf("SuccessRequests = %v, want %v", summary.SuccessRequests, 95)
	}
	if summary.ErrorRequests != 5 {
		t.Errorf("ErrorRequests = %v, want %v", summary.ErrorRequests, 5)
	}
	if summary.CostUnits != 0.5 {
		t.Errorf("CostUnits = %v, want %v", summary.CostUnits, 0.5)
	}
}

func TestQueryOptions_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Hour)
	success := true

	opts := QueryOptions{
		Start:     start,
		End:       end,
		Channel:   "http",
		Module:    "user",
		Action:    "get",
		UserID:    "user-1",
		APIKeyID:  "key-1",
		Success:   &success,
		Limit:     50,
		Offset:    10,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}

	if opts.Start != start {
		t.Errorf("Start = %v, want %v", opts.Start, start)
	}
	if opts.End != end {
		t.Errorf("End = %v, want %v", opts.End, end)
	}
	if opts.Channel != "http" {
		t.Errorf("Channel = %v, want %v", opts.Channel, "http")
	}
	if opts.Limit != 50 {
		t.Errorf("Limit = %v, want %v", opts.Limit, 50)
	}
	if opts.Offset != 10 {
		t.Errorf("Offset = %v, want %v", opts.Offset, 10)
	}
	if !opts.OrderDesc {
		t.Errorf("OrderDesc = %v, want %v", opts.OrderDesc, true)
	}
}

func TestAggregateOptions_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Hour)

	opts := AggregateOptions{
		Start:   start,
		End:     end,
		GroupBy: []string{"module", "action", "channel"},
		Period:  "hour",
		Channel: "http",
		Module:  "user",
		Action:  "get",
	}

	if opts.Start != start {
		t.Errorf("Start = %v, want %v", opts.Start, start)
	}
	if opts.End != end {
		t.Errorf("End = %v, want %v", opts.End, end)
	}
	if opts.Period != "hour" {
		t.Errorf("Period = %v, want %v", opts.Period, "hour")
	}
	if len(opts.GroupBy) != 3 {
		t.Errorf("GroupBy length = %v, want %v", len(opts.GroupBy), 3)
	}
	if opts.GroupBy[0] != "module" {
		t.Errorf("GroupBy[0] = %v, want %v", opts.GroupBy[0], "module")
	}
}

func TestDefaultCostCalculator_ZeroCosts(t *testing.T) {
	calc := &DefaultCostCalculator{
		BaseRequestCost:   0,
		CPUCostPerUS:      0,
		MemoryCostPerKB:   0,
		TransferCostPerKB: 0,
	}

	event := Event{
		DurationNS:    1000000,
		MemoryBytes:   2048,
		RequestBytes:  1024,
		ResponseBytes: 2048,
	}

	result := calc.Calculate(event)
	if result != 0 {
		t.Errorf("Calculate() with zero costs = %v, want 0", result)
	}
}

func TestDefaultCostCalculator_LargeValues(t *testing.T) {
	calc := NewDefaultCostCalculator()

	event := Event{
		DurationNS:    1000000000000, // 1000 seconds
		MemoryBytes:   1073741824,    // 1GB
		RequestBytes:  1073741824,    // 1GB
		ResponseBytes: 1073741824,    // 1GB
	}

	result := calc.Calculate(event)

	// Should calculate without overflow
	if result <= 0 {
		t.Errorf("Calculate() with large values = %v, want positive value", result)
	}
}
