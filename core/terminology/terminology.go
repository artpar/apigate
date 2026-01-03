// Package terminology provides configurable UI labels for different metering modes.
// This allows the UI to display "tokens" instead of "requests" for LLM APIs,
// "data points" for analytics APIs, "KB" for storage APIs, etc.
package terminology

// Labels contains UI labels for metering units.
// These are used throughout the admin and portal interfaces.
type Labels struct {
	// UsageUnit is the singular form (e.g., "request", "token", "data point")
	UsageUnit string

	// UsageUnitPlural is the plural form (e.g., "requests", "tokens", "data points")
	UsageUnitPlural string

	// QuotaLabel is the label for quota limits (e.g., "Monthly Quota", "Token Quota")
	QuotaLabel string

	// RateLimitLabel is the label for rate limits (e.g., "req/min", "tokens/min")
	RateLimitLabel string

	// OverageLabel is the label for overage pricing (e.g., "per request", "per token")
	OverageLabel string
}

// MeteringUnit represents the type of unit being metered.
type MeteringUnit string

const (
	UnitRequests   MeteringUnit = "requests"
	UnitTokens     MeteringUnit = "tokens"
	UnitDataPoints MeteringUnit = "data_points"
	UnitBytes      MeteringUnit = "bytes"
)

// ForUnit returns the appropriate Labels for a given metering unit.
func ForUnit(unit string) Labels {
	switch MeteringUnit(unit) {
	case UnitTokens:
		return Labels{
			UsageUnit:       "token",
			UsageUnitPlural: "tokens",
			QuotaLabel:      "Monthly Token Quota",
			RateLimitLabel:  "tokens/min",
			OverageLabel:    "per token",
		}
	case UnitDataPoints:
		return Labels{
			UsageUnit:       "data point",
			UsageUnitPlural: "data points",
			QuotaLabel:      "Monthly Data Points",
			RateLimitLabel:  "points/min",
			OverageLabel:    "per data point",
		}
	case UnitBytes:
		return Labels{
			UsageUnit:       "KB",
			UsageUnitPlural: "KB",
			QuotaLabel:      "Monthly Data Transfer",
			RateLimitLabel:  "KB/min",
			OverageLabel:    "per KB",
		}
	default: // UnitRequests or unknown
		return Labels{
			UsageUnit:       "request",
			UsageUnitPlural: "requests",
			QuotaLabel:      "Monthly Quota",
			RateLimitLabel:  "req/min",
			OverageLabel:    "per request",
		}
	}
}

// Default returns the default labels (requests).
func Default() Labels {
	return ForUnit(string(UnitRequests))
}
