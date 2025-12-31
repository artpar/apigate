package exporter

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/artpar/apigate/core/analytics"
)

// MockStore implements analytics.Store for testing
type MockStore struct {
	events      []analytics.Event
	summaries   []analytics.Summary
	aggregateErr error
	writeErr     error
}

func (m *MockStore) Write(ctx context.Context, events []analytics.Event) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.events = append(m.events, events...)
	return nil
}

func (m *MockStore) Query(ctx context.Context, opts analytics.QueryOptions) ([]analytics.Event, int64, error) {
	return m.events, int64(len(m.events)), nil
}

func (m *MockStore) Aggregate(ctx context.Context, opts analytics.AggregateOptions) ([]analytics.Summary, error) {
	if m.aggregateErr != nil {
		return nil, m.aggregateErr
	}
	return m.summaries, nil
}

func (m *MockStore) Delete(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *MockStore) Close() error {
	return nil
}

// MockExporter implements Exporter for testing
type MockExporter struct {
	name      string
	startErr  error
	stopErr   error
	started   bool
	stopped   bool
}

func (m *MockExporter) Name() string {
	return m.name
}

func (m *MockExporter) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *MockExporter) Stop(ctx context.Context) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

// MockPullExporter implements PullExporter for testing
type MockPullExporter struct {
	MockExporter
	collectErr error
}

func (m *MockPullExporter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("metrics"))
	})
}

func (m *MockPullExporter) Collect(ctx context.Context) error {
	return m.collectErr
}

// TestMetricTypeString tests the String method of MetricType
func TestMetricTypeString(t *testing.T) {
	tests := []struct {
		name     string
		typ      MetricType
		expected string
	}{
		{"Counter", Counter, "counter"},
		{"Gauge", Gauge, "gauge"},
		{"Histogram", Histogram, "histogram"},
		{"Summary", Summary, "summary"},
		{"Unknown", MetricType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ.String()
			if result != tt.expected {
				t.Errorf("MetricType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestNewRegistry tests creating a new Registry
func TestNewRegistry(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if reg.exporters == nil {
		t.Error("Registry exporters map is nil")
	}

	if reg.analytics != store {
		t.Error("Registry analytics store not set correctly")
	}
}

// TestRegistryRegister tests registering exporters
func TestRegistryRegister(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	exp1 := &MockExporter{name: "test1"}
	exp2 := &MockExporter{name: "test2"}

	err := reg.Register(exp1)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = reg.Register(exp2)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify they're registered
	if len(reg.exporters) != 2 {
		t.Errorf("Expected 2 exporters, got %d", len(reg.exporters))
	}
}

// TestRegistryGet tests getting an exporter by name
func TestRegistryGet(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	exp := &MockExporter{name: "test"}
	reg.Register(exp)

	// Test getting existing exporter
	got, ok := reg.Get("test")
	if !ok {
		t.Error("Get returned false for existing exporter")
	}
	if got.Name() != "test" {
		t.Errorf("Got wrong exporter, name = %q", got.Name())
	}

	// Test getting non-existing exporter
	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Get returned true for non-existing exporter")
	}
}

// TestRegistryAll tests getting all exporters
func TestRegistryAll(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	exp1 := &MockExporter{name: "test1"}
	exp2 := &MockExporter{name: "test2"}
	reg.Register(exp1)
	reg.Register(exp2)

	all := reg.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d exporters, want 2", len(all))
	}
}

// TestRegistryPullExporters tests filtering pull exporters
func TestRegistryPullExporters(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	// Add a regular exporter
	exp1 := &MockExporter{name: "regular"}
	reg.Register(exp1)

	// Add a pull exporter
	pullExp := &MockPullExporter{MockExporter: MockExporter{name: "pull"}}
	reg.Register(pullExp)

	pullExporters := reg.PullExporters()
	if len(pullExporters) != 1 {
		t.Errorf("PullExporters() returned %d exporters, want 1", len(pullExporters))
	}

	if len(pullExporters) > 0 && pullExporters[0].Name() != "pull" {
		t.Error("PullExporters() returned wrong exporter")
	}
}

// TestRegistryStart tests starting all exporters
func TestRegistryStart(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	exp1 := &MockExporter{name: "test1"}
	exp2 := &MockExporter{name: "test2"}
	reg.Register(exp1)
	reg.Register(exp2)

	ctx := context.Background()
	err := reg.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !exp1.started || !exp2.started {
		t.Error("Not all exporters were started")
	}
}

// TestRegistryStartError tests start error handling
func TestRegistryStartError(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	expectedErr := errors.New("start error")
	exp := &MockExporter{name: "test", startErr: expectedErr}
	reg.Register(exp)

	ctx := context.Background()
	err := reg.Start(ctx)
	if err != expectedErr {
		t.Errorf("Start error = %v, want %v", err, expectedErr)
	}
}

// TestRegistryStop tests stopping all exporters
func TestRegistryStop(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	exp1 := &MockExporter{name: "test1"}
	exp2 := &MockExporter{name: "test2"}
	reg.Register(exp1)
	reg.Register(exp2)

	ctx := context.Background()
	err := reg.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if !exp1.stopped || !exp2.stopped {
		t.Error("Not all exporters were stopped")
	}
}

// TestRegistryStopError tests stop error handling
func TestRegistryStopError(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	expectedErr := errors.New("stop error")
	exp := &MockExporter{name: "test", stopErr: expectedErr}
	reg.Register(exp)

	ctx := context.Background()
	err := reg.Stop(ctx)
	if err != expectedErr {
		t.Errorf("Stop error = %v, want %v", err, expectedErr)
	}
}

// TestRegistryAnalytics tests getting the analytics store
func TestRegistryAnalytics(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	if reg.Analytics() != store {
		t.Error("Analytics() returned wrong store")
	}
}

// TestNewMetricBuilder tests creating a new MetricBuilder
func TestNewMetricBuilder(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	if builder == nil {
		t.Fatal("NewMetricBuilder returned nil")
	}

	if builder.prefix != "apigate" {
		t.Errorf("prefix = %q, want %q", builder.prefix, "apigate")
	}

	if builder.labels == nil {
		t.Error("labels map is nil")
	}
}

// TestMetricBuilderWithLabels tests adding labels to builder
func TestMetricBuilderWithLabels(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	result := builder.WithLabels(map[string]string{
		"env":     "test",
		"version": "1.0",
	})

	// Check fluent return
	if result != builder {
		t.Error("WithLabels did not return the builder")
	}

	if builder.labels["env"] != "test" {
		t.Errorf("env label = %q, want %q", builder.labels["env"], "test")
	}

	if builder.labels["version"] != "1.0" {
		t.Errorf("version label = %q, want %q", builder.labels["version"], "1.0")
	}
}

// TestMetricBuilderCounter tests creating counter metrics
func TestMetricBuilderCounter(t *testing.T) {
	builder := NewMetricBuilder("apigate").WithLabels(map[string]string{"env": "test"})

	metric := builder.Counter("requests_total", 100, map[string]string{"module": "user"}, "Total requests")

	if metric.Name != "apigate_requests_total" {
		t.Errorf("metric name = %q, want %q", metric.Name, "apigate_requests_total")
	}

	if metric.Type != Counter {
		t.Errorf("metric type = %v, want %v", metric.Type, Counter)
	}

	if metric.Value != 100 {
		t.Errorf("metric value = %v, want %v", metric.Value, 100.0)
	}

	if metric.Labels["env"] != "test" {
		t.Error("Default label not included")
	}

	if metric.Labels["module"] != "user" {
		t.Error("Provided label not included")
	}

	if metric.Help != "Total requests" {
		t.Errorf("metric help = %q", metric.Help)
	}
}

// TestMetricBuilderGauge tests creating gauge metrics
func TestMetricBuilderGauge(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	metric := builder.Gauge("active_connections", 42, nil, "Active connections")

	if metric.Name != "apigate_active_connections" {
		t.Errorf("metric name = %q", metric.Name)
	}

	if metric.Type != Gauge {
		t.Errorf("metric type = %v, want %v", metric.Type, Gauge)
	}

	if metric.Value != 42 {
		t.Errorf("metric value = %v", metric.Value)
	}
}

// TestMetricBuilderHistogram tests creating histogram metrics
func TestMetricBuilderHistogram(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	metric := builder.Histogram("request_duration", 0.5, nil, "Request duration")

	if metric.Name != "apigate_request_duration" {
		t.Errorf("metric name = %q", metric.Name)
	}

	if metric.Type != Histogram {
		t.Errorf("metric type = %v, want %v", metric.Type, Histogram)
	}
}

// TestMetricBuilderNoPrefix tests metric builder without prefix
func TestMetricBuilderNoPrefix(t *testing.T) {
	builder := NewMetricBuilder("")

	metric := builder.Counter("requests_total", 100, nil, "Total requests")

	if metric.Name != "requests_total" {
		t.Errorf("metric name = %q, want %q", metric.Name, "requests_total")
	}
}

// TestMetricBuilderLabelMerging tests that labels are properly merged
func TestMetricBuilderLabelMerging(t *testing.T) {
	builder := NewMetricBuilder("test").WithLabels(map[string]string{
		"env":    "test",
		"common": "value1",
	})

	metric := builder.Counter("test", 1, map[string]string{
		"specific": "value2",
		"common":   "override", // Should override default
	}, "")

	if metric.Labels["env"] != "test" {
		t.Error("Default env label missing")
	}

	if metric.Labels["specific"] != "value2" {
		t.Error("Specific label missing")
	}

	if metric.Labels["common"] != "override" {
		t.Errorf("Label override failed, common = %q", metric.Labels["common"])
	}
}

// TestSummaryToMetrics tests converting summaries to metrics
func TestSummaryToMetrics(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	summaries := []analytics.Summary{
		{
			Module:             "user",
			Action:             "list",
			TotalRequests:      100,
			SuccessRequests:    90,
			ErrorRequests:      10,
			AvgDurationNS:      1000000,
			MinDurationNS:      500000,
			MaxDurationNS:      5000000,
			TotalMemoryBytes:   1024000,
			TotalRequestBytes:  2048000,
			TotalResponseBytes: 4096000,
			CostUnits:          0.5,
		},
	}

	metrics := SummaryToMetrics(summaries, builder)

	// Should have 10 metrics per summary
	if len(metrics) != 10 {
		t.Errorf("SummaryToMetrics returned %d metrics, want 10", len(metrics))
	}

	// Check that module/action labels are set
	for _, m := range metrics {
		if m.Labels["module"] != "user" {
			t.Errorf("metric %s missing module label", m.Name)
		}
		if m.Labels["action"] != "list" {
			t.Errorf("metric %s missing action label", m.Name)
		}
	}

	// Verify some specific metrics
	foundRequestsTotal := false
	foundCostUnits := false
	for _, m := range metrics {
		if m.Name == "apigate_requests_total" && m.Value == 100 {
			foundRequestsTotal = true
		}
		if m.Name == "apigate_cost_units_total" && m.Value == 0.5 {
			foundCostUnits = true
		}
	}

	if !foundRequestsTotal {
		t.Error("requests_total metric not found or has wrong value")
	}

	if !foundCostUnits {
		t.Error("cost_units_total metric not found or has wrong value")
	}
}

// TestSummaryToMetricsEmpty tests with empty summaries
func TestSummaryToMetricsEmpty(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	metrics := SummaryToMetrics(nil, builder)
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics for nil summaries, got %d", len(metrics))
	}

	metrics = SummaryToMetrics([]analytics.Summary{}, builder)
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics for empty summaries, got %d", len(metrics))
	}
}

// TestSummaryToMetricsNoModuleAction tests summaries without module/action
func TestSummaryToMetricsNoModuleAction(t *testing.T) {
	builder := NewMetricBuilder("apigate")

	summaries := []analytics.Summary{
		{
			TotalRequests: 50,
		},
	}

	metrics := SummaryToMetrics(summaries, builder)

	// Labels should not include empty module/action
	for _, m := range metrics {
		if _, ok := m.Labels["module"]; ok {
			t.Error("Empty module should not be included in labels")
		}
		if _, ok := m.Labels["action"]; ok {
			t.Error("Empty action should not be included in labels")
		}
	}
}

// TestSummaryToMetricsMultipleSummaries tests with multiple summaries
func TestSummaryToMetricsMultipleSummaries(t *testing.T) {
	builder := NewMetricBuilder("test")

	summaries := []analytics.Summary{
		{Module: "user", Action: "list", TotalRequests: 100},
		{Module: "route", Action: "get", TotalRequests: 200},
		{Module: "upstream", Action: "create", TotalRequests: 50},
	}

	metrics := SummaryToMetrics(summaries, builder)

	// 10 metrics per summary * 3 summaries = 30 metrics
	if len(metrics) != 30 {
		t.Errorf("Expected 30 metrics, got %d", len(metrics))
	}
}

// TestConfigStruct tests the Config struct initialization
func TestConfigStruct(t *testing.T) {
	cfg := Config{
		Enabled:  true,
		Name:     "test",
		Endpoint: "http://localhost:9090",
		Interval: 30,
		Labels: map[string]string{
			"env": "production",
		},
		Extra: map[string]any{
			"custom": "value",
		},
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}

	if cfg.Name != "test" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test")
	}

	if cfg.Endpoint != "http://localhost:9090" {
		t.Errorf("Endpoint = %q", cfg.Endpoint)
	}

	if cfg.Interval != 30 {
		t.Errorf("Interval = %d, want 30", cfg.Interval)
	}

	if cfg.Labels["env"] != "production" {
		t.Error("Labels not set correctly")
	}

	if cfg.Extra["custom"] != "value" {
		t.Error("Extra not set correctly")
	}
}

// TestMetricStruct tests the Metric struct
func TestMetricStruct(t *testing.T) {
	metric := Metric{
		Name:  "test_metric",
		Type:  Counter,
		Value: 123.45,
		Labels: map[string]string{
			"key": "value",
		},
		Help: "Test metric help",
	}

	if metric.Name != "test_metric" {
		t.Errorf("Name = %q", metric.Name)
	}

	if metric.Type != Counter {
		t.Errorf("Type = %v", metric.Type)
	}

	if metric.Value != 123.45 {
		t.Errorf("Value = %v", metric.Value)
	}

	if metric.Labels["key"] != "value" {
		t.Error("Labels not set correctly")
	}

	if metric.Help != "Test metric help" {
		t.Errorf("Help = %q", metric.Help)
	}
}

// TestRegistryEmptyPullExporters tests PullExporters with no pull exporters
func TestRegistryEmptyPullExporters(t *testing.T) {
	store := &MockStore{}
	reg := NewRegistry(store)

	// Only add regular exporters
	reg.Register(&MockExporter{name: "regular1"})
	reg.Register(&MockExporter{name: "regular2"})

	pullExporters := reg.PullExporters()
	if len(pullExporters) != 0 {
		t.Errorf("Expected 0 pull exporters, got %d", len(pullExporters))
	}
}

// TestRegistryNilAnalytics tests registry with nil analytics store
func TestRegistryNilAnalytics(t *testing.T) {
	reg := NewRegistry(nil)

	if reg.Analytics() != nil {
		t.Error("Analytics() should return nil for nil store")
	}
}
