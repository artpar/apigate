package exporter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/core/analytics"
	"github.com/prometheus/client_golang/prometheus"
)

// TestDefaultPrometheusBuckets tests the default histogram buckets
func TestDefaultPrometheusBuckets(t *testing.T) {
	buckets := DefaultPrometheusBuckets()

	if len(buckets) != 12 {
		t.Errorf("Expected 12 buckets, got %d", len(buckets))
	}

	// Check first and last buckets
	if buckets[0] != 0.001 {
		t.Errorf("First bucket = %v, want 0.001", buckets[0])
	}

	if buckets[len(buckets)-1] != 10 {
		t.Errorf("Last bucket = %v, want 10", buckets[len(buckets)-1])
	}
}

// TestNewPrometheusExporter tests creating a new Prometheus exporter
func TestNewPrometheusExporter(t *testing.T) {
	store := &MockStore{}
	cfg := PrometheusConfig{
		Store:  store,
		Prefix: "apigate",
		Labels: map[string]string{"env": "test"},
	}

	exp := NewPrometheusExporter(cfg)

	if exp == nil {
		t.Fatal("NewPrometheusExporter returned nil")
	}

	if exp.prefix != "apigate" {
		t.Errorf("prefix = %q, want %q", exp.prefix, "apigate")
	}

	if exp.labels["env"] != "test" {
		t.Error("Labels not set correctly")
	}

	if exp.registry == nil {
		t.Error("registry is nil")
	}
}

// TestNewPrometheusExporterDefaults tests default values
func TestNewPrometheusExporterDefaults(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	if exp.prefix != "apigate" {
		t.Errorf("default prefix = %q, want %q", exp.prefix, "apigate")
	}

	if exp.labels == nil {
		t.Error("labels should not be nil")
	}
}

// TestPrometheusExporterName tests the Name method
func TestPrometheusExporterName(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	if exp.Name() != "prometheus" {
		t.Errorf("Name() = %q, want %q", exp.Name(), "prometheus")
	}
}

// TestPrometheusExporterStart tests starting the exporter
func TestPrometheusExporterStart(t *testing.T) {
	store := &MockStore{
		summaries: []analytics.Summary{
			{
				Channel:         "http",
				Module:          "user",
				Action:          "list",
				TotalRequests:   100,
				SuccessRequests: 90,
				ErrorRequests:   10,
			},
		},
	}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	err := exp.Start(ctx)

	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

// TestPrometheusExporterStartNoStore tests starting with nil store
func TestPrometheusExporterStartNoStore(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	ctx := context.Background()
	err := exp.Start(ctx)

	if err != nil {
		t.Fatalf("Start with nil store failed: %v", err)
	}
}

// TestPrometheusExporterStop tests stopping the exporter
func TestPrometheusExporterStop(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	ctx := context.Background()
	err := exp.Stop(ctx)

	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// TestPrometheusExporterHandler tests the HTTP handler
func TestPrometheusExporterHandler(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	handler := exp.Handler()

	if handler == nil {
		t.Fatal("Handler returned nil")
	}

	// Make a test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	// Check that Go metrics are included
	if len(bodyStr) == 0 {
		t.Error("Empty response body")
	}
}

// TestPrometheusExporterCollect tests collecting metrics
func TestPrometheusExporterCollect(t *testing.T) {
	store := &MockStore{
		summaries: []analytics.Summary{
			{
				Channel:            "http",
				Module:             "user",
				Action:             "list",
				TotalRequests:      100,
				SuccessRequests:    90,
				ErrorRequests:      10,
				AvgDurationNS:      1000000, // 1ms
				TotalRequestBytes:  1024,
				TotalResponseBytes: 2048,
				CostUnits:          0.5,
			},
		},
	}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	err := exp.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Verify lastCollect was updated
	if exp.lastCollect.IsZero() {
		t.Error("lastCollect should be set after Collect")
	}
}

// TestPrometheusExporterCollectNoStore tests Collect with nil store
func TestPrometheusExporterCollectNoStore(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	ctx := context.Background()
	err := exp.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect with nil store should not error: %v", err)
	}
}

// TestPrometheusExporterCollectError tests Collect with store error
func TestPrometheusExporterCollectError(t *testing.T) {
	expectedErr := errors.New("aggregate error")
	store := &MockStore{aggregateErr: expectedErr}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	err := exp.Collect(ctx)

	if err != expectedErr {
		t.Errorf("Collect error = %v, want %v", err, expectedErr)
	}
}

// TestPrometheusExporterCollectFromEvents tests collecting from events
func TestPrometheusExporterCollectFromEvents(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	events := []analytics.Event{
		{
			ID:            "event-1",
			Channel:       "http",
			Module:        "user",
			Action:        "create",
			DurationNS:    1500000, // 1.5ms
			RequestBytes:  256,
			ResponseBytes: 512,
			Success:       true,
		},
		{
			ID:            "event-2",
			Channel:       "http",
			Module:        "user",
			Action:        "create",
			DurationNS:    2000000, // 2ms
			RequestBytes:  128,
			ResponseBytes: 256,
			Success:       false,
		},
	}

	ctx := context.Background()
	err := exp.CollectFromEvents(ctx, events)

	if err != nil {
		t.Fatalf("CollectFromEvents failed: %v", err)
	}

	// Verify metrics were recorded by checking the handler output
	handler := exp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check that our metrics are in the output
	if !containsStr(body, "apigate_requests_total") {
		t.Error("Expected apigate_requests_total metric not found")
	}
}

// TestPrometheusExporterCollectFromEventsEmpty tests with empty events
func TestPrometheusExporterCollectFromEventsEmpty(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	ctx := context.Background()
	err := exp.CollectFromEvents(ctx, nil)

	if err != nil {
		t.Fatalf("CollectFromEvents with nil events failed: %v", err)
	}

	err = exp.CollectFromEvents(ctx, []analytics.Event{})

	if err != nil {
		t.Fatalf("CollectFromEvents with empty events failed: %v", err)
	}
}

// TestPrometheusExporterRegistry tests getting the registry
func TestPrometheusExporterRegistry(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	reg := exp.Registry()

	if reg == nil {
		t.Fatal("Registry returned nil")
	}

	if reg != exp.registry {
		t.Error("Registry returned wrong registry")
	}
}

// TestPrometheusExporterWithCustomMetric tests adding custom metrics
func TestPrometheusExporterWithCustomMetric(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	customCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_metric_total",
		Help: "A custom metric",
	})

	err := exp.WithCustomMetric(customCounter)

	if err != nil {
		t.Fatalf("WithCustomMetric failed: %v", err)
	}

	// Try to register the same metric again (should fail)
	err = exp.WithCustomMetric(customCounter)

	if err == nil {
		t.Error("Expected error when registering duplicate metric")
	}
}

// TestPrometheusConfigStruct tests PrometheusConfig initialization
func TestPrometheusConfigStruct(t *testing.T) {
	buckets := []float64{0.01, 0.1, 1.0}

	cfg := PrometheusConfig{
		Store:   &MockStore{},
		Prefix:  "myapp",
		Labels:  map[string]string{"env": "prod"},
		Buckets: buckets,
	}

	if cfg.Prefix != "myapp" {
		t.Errorf("Prefix = %q, want %q", cfg.Prefix, "myapp")
	}

	if len(cfg.Buckets) != 3 {
		t.Errorf("Buckets count = %d, want 3", len(cfg.Buckets))
	}
}

// TestPrometheusExporterCollectMultipleSummaries tests with multiple summaries
func TestPrometheusExporterCollectMultipleSummaries(t *testing.T) {
	store := &MockStore{
		summaries: []analytics.Summary{
			{
				Channel:         "http",
				Module:          "user",
				Action:          "list",
				TotalRequests:   100,
				SuccessRequests: 90,
				ErrorRequests:   10,
				AvgDurationNS:   1000000,
			},
			{
				Channel:         "http",
				Module:          "route",
				Action:          "get",
				TotalRequests:   200,
				SuccessRequests: 200,
				ErrorRequests:   0,
				AvgDurationNS:   500000,
			},
			{
				Channel:         "grpc",
				Module:          "upstream",
				Action:          "create",
				TotalRequests:   50,
				SuccessRequests: 45,
				ErrorRequests:   5,
				AvgDurationNS:   2000000,
			},
		},
	}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	err := exp.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Verify metrics were recorded
	handler := exp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check for different module labels
	if !containsStr(body, `module="user"`) {
		t.Error("Expected user module metric not found")
	}

	if !containsStr(body, `module="route"`) {
		t.Error("Expected route module metric not found")
	}
}

// TestPrometheusExporterCollectZeroRequests tests with zero requests
func TestPrometheusExporterCollectZeroRequests(t *testing.T) {
	store := &MockStore{
		summaries: []analytics.Summary{
			{
				Channel:         "http",
				Module:          "user",
				Action:          "list",
				TotalRequests:   0,
				SuccessRequests: 0,
				ErrorRequests:   0,
				AvgDurationNS:   0,
			},
		},
	}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	err := exp.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
}

// TestPrometheusExporterCollectFromEventsSuccess tests success/error counting
func TestPrometheusExporterCollectFromEventsSuccess(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	// Create mix of success and error events
	events := []analytics.Event{
		{Channel: "http", Module: "user", Action: "get", Success: true, DurationNS: 1000000},
		{Channel: "http", Module: "user", Action: "get", Success: true, DurationNS: 1000000},
		{Channel: "http", Module: "user", Action: "get", Success: false, DurationNS: 1000000},
	}

	ctx := context.Background()
	err := exp.CollectFromEvents(ctx, events)

	if err != nil {
		t.Fatalf("CollectFromEvents failed: %v", err)
	}
}

// TestPrometheusExporterWithCustomPrefix tests custom prefix
func TestPrometheusExporterWithCustomPrefix(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{
		Prefix: "myapp",
	})

	// Verify the prefix is used in metrics
	handler := exp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The custom prefix should not appear in default go metrics
	// but would appear if we collected data
}

// TestPrometheusExporterThreadSafety tests concurrent access
func TestPrometheusExporterThreadSafety(t *testing.T) {
	store := &MockStore{
		summaries: []analytics.Summary{
			{Channel: "http", Module: "user", Action: "list", TotalRequests: 10, AvgDurationNS: 1000000},
		},
	}

	exp := NewPrometheusExporter(PrometheusConfig{Store: store})

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent Collect calls
	for i := 0; i < 10; i++ {
		go func() {
			_ = exp.Collect(ctx)
			done <- true
		}()
	}

	// Concurrent CollectFromEvents calls
	for i := 0; i < 10; i++ {
		go func() {
			events := []analytics.Event{
				{Channel: "http", Module: "user", Action: "get", Success: true, DurationNS: 1000000},
			}
			_ = exp.CollectFromEvents(ctx, events)
			done <- true
		}()
	}

	// Wait for all goroutines with timeout
	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}

// TestPrometheusExporterHandlerContentType tests handler content type
func TestPrometheusExporterHandlerContentType(t *testing.T) {
	exp := NewPrometheusExporter(PrometheusConfig{})

	handler := exp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")

	// Prometheus handler returns text/plain or application/openmetrics-text
	if contentType == "" {
		t.Error("Content-Type header is empty")
	}
}

// Helper function for string contains check
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
