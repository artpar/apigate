package terminology

import "testing"

func TestForUnit_Requests(t *testing.T) {
	labels := ForUnit("requests")
	if labels.UsageUnit != "request" {
		t.Errorf("UsageUnit = %s, want request", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "requests" {
		t.Errorf("UsageUnitPlural = %s, want requests", labels.UsageUnitPlural)
	}
	if labels.QuotaLabel != "Monthly Quota" {
		t.Errorf("QuotaLabel = %s, want Monthly Quota", labels.QuotaLabel)
	}
	if labels.RateLimitLabel != "req/min" {
		t.Errorf("RateLimitLabel = %s, want req/min", labels.RateLimitLabel)
	}
	if labels.OverageLabel != "per request" {
		t.Errorf("OverageLabel = %s, want per request", labels.OverageLabel)
	}
}

func TestForUnit_Tokens(t *testing.T) {
	labels := ForUnit("tokens")
	if labels.UsageUnit != "token" {
		t.Errorf("UsageUnit = %s, want token", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "tokens" {
		t.Errorf("UsageUnitPlural = %s, want tokens", labels.UsageUnitPlural)
	}
	if labels.QuotaLabel != "Monthly Token Quota" {
		t.Errorf("QuotaLabel = %s, want Monthly Token Quota", labels.QuotaLabel)
	}
	if labels.RateLimitLabel != "tokens/min" {
		t.Errorf("RateLimitLabel = %s, want tokens/min", labels.RateLimitLabel)
	}
	if labels.OverageLabel != "per token" {
		t.Errorf("OverageLabel = %s, want per token", labels.OverageLabel)
	}
}

func TestForUnit_DataPoints(t *testing.T) {
	labels := ForUnit("data_points")
	if labels.UsageUnit != "data point" {
		t.Errorf("UsageUnit = %s, want data point", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "data points" {
		t.Errorf("UsageUnitPlural = %s, want data points", labels.UsageUnitPlural)
	}
	if labels.QuotaLabel != "Monthly Data Points" {
		t.Errorf("QuotaLabel = %s, want Monthly Data Points", labels.QuotaLabel)
	}
	if labels.RateLimitLabel != "points/min" {
		t.Errorf("RateLimitLabel = %s, want points/min", labels.RateLimitLabel)
	}
	if labels.OverageLabel != "per data point" {
		t.Errorf("OverageLabel = %s, want per data point", labels.OverageLabel)
	}
}

func TestForUnit_Bytes(t *testing.T) {
	labels := ForUnit("bytes")
	if labels.UsageUnit != "KB" {
		t.Errorf("UsageUnit = %s, want KB", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "KB" {
		t.Errorf("UsageUnitPlural = %s, want KB", labels.UsageUnitPlural)
	}
	if labels.QuotaLabel != "Monthly Data Transfer" {
		t.Errorf("QuotaLabel = %s, want Monthly Data Transfer", labels.QuotaLabel)
	}
	if labels.RateLimitLabel != "KB/min" {
		t.Errorf("RateLimitLabel = %s, want KB/min", labels.RateLimitLabel)
	}
	if labels.OverageLabel != "per KB" {
		t.Errorf("OverageLabel = %s, want per KB", labels.OverageLabel)
	}
}

func TestForUnit_Unknown(t *testing.T) {
	// Unknown unit should default to requests
	labels := ForUnit("unknown")
	if labels.UsageUnit != "request" {
		t.Errorf("UsageUnit = %s, want request for unknown unit", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "requests" {
		t.Errorf("UsageUnitPlural = %s, want requests for unknown unit", labels.UsageUnitPlural)
	}
}

func TestForUnit_EmptyString(t *testing.T) {
	// Empty string should default to requests
	labels := ForUnit("")
	if labels.UsageUnit != "request" {
		t.Errorf("UsageUnit = %s, want request for empty string", labels.UsageUnit)
	}
}

func TestDefault(t *testing.T) {
	labels := Default()
	if labels.UsageUnit != "request" {
		t.Errorf("UsageUnit = %s, want request", labels.UsageUnit)
	}
	if labels.UsageUnitPlural != "requests" {
		t.Errorf("UsageUnitPlural = %s, want requests", labels.UsageUnitPlural)
	}
	if labels.QuotaLabel != "Monthly Quota" {
		t.Errorf("QuotaLabel = %s, want Monthly Quota", labels.QuotaLabel)
	}
	if labels.RateLimitLabel != "req/min" {
		t.Errorf("RateLimitLabel = %s, want req/min", labels.RateLimitLabel)
	}
	if labels.OverageLabel != "per request" {
		t.Errorf("OverageLabel = %s, want per request", labels.OverageLabel)
	}
}

func TestMeteringUnitConstants(t *testing.T) {
	// Verify constants are defined correctly
	tests := []struct {
		unit     MeteringUnit
		expected string
	}{
		{UnitRequests, "requests"},
		{UnitTokens, "tokens"},
		{UnitDataPoints, "data_points"},
		{UnitBytes, "bytes"},
	}

	for _, tt := range tests {
		if string(tt.unit) != tt.expected {
			t.Errorf("MeteringUnit %v = %s, want %s", tt.unit, string(tt.unit), tt.expected)
		}
	}
}
