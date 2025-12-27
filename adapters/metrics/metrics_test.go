package metrics_test

import (
	"testing"

	"github.com/artpar/apigate/adapters/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	// Use a new registry to avoid conflicts with other tests
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	if m == nil {
		t.Fatal("NewWithRegistry returned nil")
	}

	// Verify all metrics are initialized
	if m.RequestsTotal == nil {
		t.Error("RequestsTotal is nil")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration is nil")
	}
	if m.RequestsInFlight == nil {
		t.Error("RequestsInFlight is nil")
	}
	if m.AuthFailures == nil {
		t.Error("AuthFailures is nil")
	}
	if m.RateLimitHits == nil {
		t.Error("RateLimitHits is nil")
	}
	if m.UsageRequests == nil {
		t.Error("UsageRequests is nil")
	}
	if m.UpstreamDuration == nil {
		t.Error("UpstreamDuration is nil")
	}
	if m.ConfigReloads == nil {
		t.Error("ConfigReloads is nil")
	}
}

func TestRequestsTotal(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	// Record some requests
	m.RequestsTotal.WithLabelValues("GET", "/api/test", "2xx", "free").Inc()
	m.RequestsTotal.WithLabelValues("POST", "/api/data", "4xx", "pro").Add(5)

	// Verify metrics were gathered
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "apigate_requests_total" {
			found = true
			if len(f.GetMetric()) != 2 {
				t.Errorf("expected 2 metric series, got %d", len(f.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("apigate_requests_total metric not found")
	}
}

func TestRequestDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	// Record some durations
	m.RequestDuration.WithLabelValues("GET", "/api/test", "2xx").Observe(0.05)
	m.RequestDuration.WithLabelValues("GET", "/api/test", "2xx").Observe(0.1)
	m.RequestDuration.WithLabelValues("GET", "/api/test", "2xx").Observe(0.5)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "apigate_request_duration_seconds" {
			found = true
		}
	}
	if !found {
		t.Error("apigate_request_duration_seconds metric not found")
	}
}

func TestAuthFailures(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	m.AuthFailures.WithLabelValues("invalid_api_key").Inc()
	m.AuthFailures.WithLabelValues("missing_api_key").Add(3)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "apigate_auth_failures_total" {
			found = true
			if len(f.GetMetric()) != 2 {
				t.Errorf("expected 2 metric series, got %d", len(f.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("apigate_auth_failures_total metric not found")
	}
}

func TestRateLimitHits(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	m.RateLimitHits.WithLabelValues("free", "user1").Inc()
	m.RateLimitHits.WithLabelValues("free", "user2").Inc()
	m.RateLimitHits.WithLabelValues("pro", "user3").Inc()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "apigate_rate_limit_hits_total" {
			found = true
			if len(f.GetMetric()) != 3 {
				t.Errorf("expected 3 metric series, got %d", len(f.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("apigate_rate_limit_hits_total metric not found")
	}
}

func TestUsageMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	m.UsageRequests.WithLabelValues("user1", "free").Inc()
	m.UsageBytes.WithLabelValues("user1", "free", "request").Add(1024)
	m.UsageBytes.WithLabelValues("user1", "free", "response").Add(2048)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	foundRequests := false
	foundBytes := false
	for _, f := range families {
		if f.GetName() == "apigate_usage_requests_total" {
			foundRequests = true
		}
		if f.GetName() == "apigate_usage_bytes_total" {
			foundBytes = true
		}
	}
	if !foundRequests {
		t.Error("apigate_usage_requests_total metric not found")
	}
	if !foundBytes {
		t.Error("apigate_usage_bytes_total metric not found")
	}
}

func TestConfigReloads(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	m.ConfigReloads.Inc()
	m.ConfigLastReload.SetToCurrentTime()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	foundReloads := false
	foundLastReload := false
	for _, f := range families {
		if f.GetName() == "apigate_config_reloads_total" {
			foundReloads = true
		}
		if f.GetName() == "apigate_config_last_reload_timestamp" {
			foundLastReload = true
		}
	}
	if !foundReloads {
		t.Error("apigate_config_reloads_total metric not found")
	}
	if !foundLastReload {
		t.Error("apigate_config_last_reload_timestamp metric not found")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/test", "/api/test"},
		{"/api/users/123", "/api/users/123"},
		{"/short", "/short"},
	}

	for _, tt := range tests {
		result := metrics.NormalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizePath(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}

	// Test long path truncation
	longPath := "/very/long/path/that/exceeds/fifty/characters/in/total/length"
	result := metrics.NormalizePath(longPath)
	if len(result) > 53 { // 50 chars + "..."
		t.Errorf("NormalizePath should truncate long paths, got len=%d", len(result))
	}
	if result[len(result)-3:] != "..." {
		t.Errorf("truncated path should end with '...', got %s", result)
	}
}

func TestRequestsInFlight(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(reg)

	// Simulate requests in flight
	m.RequestsInFlight.Inc()
	m.RequestsInFlight.Inc()
	m.RequestsInFlight.Dec()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "apigate_requests_in_flight" {
			found = true
			if len(f.GetMetric()) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.GetMetric()))
			}
			// Value should be 1 (2 inc - 1 dec)
			val := f.GetMetric()[0].GetGauge().GetValue()
			if val != 1 {
				t.Errorf("expected value 1, got %f", val)
			}
		}
	}
	if !found {
		t.Error("apigate_requests_in_flight metric not found")
	}
}
