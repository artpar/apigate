// Package usage provides usage event types and aggregation functions.
// All functions are pure - no side effects.
package usage

import "time"

// Event represents a single API request (immutable value type).
type Event struct {
	ID             string
	KeyID          string
	UserID         string
	Method         string
	Path           string
	StatusCode     int
	LatencyMs      int64
	RequestBytes   int64
	ResponseBytes  int64
	CostMultiplier float64 // For endpoint-specific pricing
	IPAddress      string
	UserAgent      string
	Timestamp      time.Time
}

// Summary represents aggregated usage for a period (value type).
type Summary struct {
	UserID        string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	RequestCount  int64
	ComputeUnits  float64 // Weighted by cost multipliers
	BytesIn       int64
	BytesOut      int64
	ErrorCount    int64 // 4xx + 5xx responses
	AvgLatencyMs  int64
}

// Quota represents usage limits for a plan (value type).
type Quota struct {
	RequestsPerMonth int64
	BytesPerMonth    int64 // 0 = unlimited
}

// QuotaStatus represents current quota usage (value type).
type QuotaStatus struct {
	RequestsUsed     int64
	RequestsLimit    int64
	RequestsPercent  float64
	BytesUsed        int64
	BytesLimit       int64
	BytesPercent     float64
	IsOverQuota      bool
	OverageRequests  int64
}
