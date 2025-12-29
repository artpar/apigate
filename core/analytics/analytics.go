// Package analytics provides runtime analytics and metrics collection.
// It automatically tracks all action executions with resource usage.
package analytics

import (
	"context"
	"time"
)

// Event represents a single execution event.
type Event struct {
	// Identity
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// What was executed
	Channel  string `json:"channel"`  // http, cli, tty, grpc, websocket
	Module   string `json:"module"`   // user, route, upstream, etc.
	Action   string `json:"action"`   // list, get, create, update, delete, custom
	RecordID string `json:"record_id,omitempty"`

	// Who made the request
	UserID   string `json:"user_id,omitempty"`
	APIKeyID string `json:"api_key_id,omitempty"`
	RemoteIP string `json:"remote_ip,omitempty"`

	// Resource usage
	DurationNS    int64 `json:"duration_ns"`    // Nanoseconds
	MemoryBytes   int64 `json:"memory_bytes"`   // Memory allocated
	RequestBytes  int64 `json:"request_bytes"`  // Request size
	ResponseBytes int64 `json:"response_bytes"` // Response size

	// Result
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Summary represents aggregated analytics for a time period.
type Summary struct {
	// Grouping
	Channel string `json:"channel,omitempty"`
	Module  string `json:"module,omitempty"`
	Action  string `json:"action,omitempty"`
	Period  string `json:"period"` // minute, hour, day

	// Time range
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`

	// Counts
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	ErrorRequests   int64 `json:"error_requests"`

	// Latency (nanoseconds)
	AvgDurationNS int64 `json:"avg_duration_ns"`
	MinDurationNS int64 `json:"min_duration_ns"`
	MaxDurationNS int64 `json:"max_duration_ns"`
	P50DurationNS int64 `json:"p50_duration_ns"`
	P95DurationNS int64 `json:"p95_duration_ns"`
	P99DurationNS int64 `json:"p99_duration_ns"`

	// Resource usage
	TotalMemoryBytes   int64 `json:"total_memory_bytes"`
	TotalRequestBytes  int64 `json:"total_request_bytes"`
	TotalResponseBytes int64 `json:"total_response_bytes"`

	// Cost
	CostUnits float64 `json:"cost_units"`
}

// QueryOptions configures analytics queries.
type QueryOptions struct {
	// Time range
	Start time.Time
	End   time.Time

	// Filters
	Channel  string
	Module   string
	Action   string
	UserID   string
	APIKeyID string
	Success  *bool

	// Pagination
	Limit  int
	Offset int

	// Ordering
	OrderBy   string // timestamp, duration_ns, memory_bytes
	OrderDesc bool
}

// AggregateOptions configures aggregation queries.
type AggregateOptions struct {
	// Time range
	Start time.Time
	End   time.Time

	// Grouping
	GroupBy []string // module, action, channel, user_id, api_key_id
	Period  string   // minute, hour, day

	// Filters
	Channel string
	Module  string
	Action  string
}

// Collector collects and stores analytics events.
type Collector interface {
	// Record stores an event (non-blocking, best-effort).
	Record(event Event)

	// RecordAsync stores an event asynchronously.
	RecordAsync(event Event)

	// Flush forces pending events to be written.
	Flush(ctx context.Context) error

	// Close shuts down the collector.
	Close() error
}

// Store provides analytics storage and querying.
type Store interface {
	// Write writes events to storage.
	Write(ctx context.Context, events []Event) error

	// Query retrieves events matching the options.
	Query(ctx context.Context, opts QueryOptions) ([]Event, int64, error)

	// Aggregate returns summarized analytics.
	Aggregate(ctx context.Context, opts AggregateOptions) ([]Summary, error)

	// Delete removes events older than the given time.
	Delete(ctx context.Context, before time.Time) (int64, error)

	// Close shuts down the store.
	Close() error
}

// Analytics combines collection and querying.
type Analytics interface {
	Collector
	Store
}

// CostCalculator calculates cost units for an event.
type CostCalculator interface {
	Calculate(event Event) float64
}

// DefaultCostCalculator provides a simple cost calculation.
type DefaultCostCalculator struct {
	// Cost per request
	BaseRequestCost float64

	// Cost per microsecond of CPU time
	CPUCostPerUS float64

	// Cost per KB of memory
	MemoryCostPerKB float64

	// Cost per KB of data transfer
	TransferCostPerKB float64
}

// Calculate computes cost units for an event.
func (c *DefaultCostCalculator) Calculate(event Event) float64 {
	cost := c.BaseRequestCost

	// CPU cost
	cost += float64(event.DurationNS/1000) * c.CPUCostPerUS

	// Memory cost
	cost += float64(event.MemoryBytes/1024) * c.MemoryCostPerKB

	// Transfer cost
	totalBytes := event.RequestBytes + event.ResponseBytes
	cost += float64(totalBytes/1024) * c.TransferCostPerKB

	return cost
}

// NewDefaultCostCalculator creates a cost calculator with sensible defaults.
func NewDefaultCostCalculator() *DefaultCostCalculator {
	return &DefaultCostCalculator{
		BaseRequestCost:   0.001,   // $0.001 per request
		CPUCostPerUS:      0.00001, // $0.01 per second
		MemoryCostPerKB:   0.000001,
		TransferCostPerKB: 0.00001,
	}
}
