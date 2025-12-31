package exporter

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/core/analytics"
	"github.com/rs/zerolog"
)

// TestNewLogExporter tests creating a new LogExporter
func TestNewLogExporter(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	store := &MockStore{}
	cfg := LogConfig{
		Logger:   logger,
		Store:    store,
		Interval: 60,
	}

	exp := NewLogExporter(cfg)

	if exp == nil {
		t.Fatal("NewLogExporter returned nil")
	}

	if exp.interval != 60 {
		t.Errorf("interval = %d, want 60", exp.interval)
	}

	if exp.done == nil {
		t.Error("done channel is nil")
	}
}

// TestLogExporterName tests the Name method
func TestLogExporterName(t *testing.T) {
	exp := NewLogExporter(LogConfig{})

	if exp.Name() != "log" {
		t.Errorf("Name() = %q, want %q", exp.Name(), "log")
	}
}

// TestLogExporterStart tests starting the exporter
func TestLogExporterStart(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	ctx := context.Background()
	err := exp.Start(ctx)

	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check that log message was written
	if !bytes.Contains(buf.Bytes(), []byte("log exporter started")) {
		t.Error("Expected start log message not found")
	}
}

// TestLogExporterStop tests stopping the exporter
func TestLogExporterStop(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	ctx := context.Background()
	err := exp.Stop(ctx)

	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Check that log message was written
	if !bytes.Contains(buf.Bytes(), []byte("log exporter stopped")) {
		t.Error("Expected stop log message not found")
	}
}

// TestLogExporterPush tests pushing summaries
func TestLogExporterPush(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	summaries := []analytics.Summary{
		{
			Module:          "user",
			Action:          "list",
			TotalRequests:   100,
			SuccessRequests: 90,
			ErrorRequests:   10,
			AvgDurationNS:   1000000,
			CostUnits:       0.5,
		},
		{
			Module:          "route",
			Action:          "get",
			TotalRequests:   50,
			SuccessRequests: 50,
			ErrorRequests:   0,
			AvgDurationNS:   500000,
			CostUnits:       0.25,
		},
	}

	ctx := context.Background()
	err := exp.Push(ctx, summaries)

	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	output := buf.String()

	// Check that summary info was logged
	if !bytes.Contains(buf.Bytes(), []byte("user")) {
		t.Error("Expected 'user' module not found in output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("route")) {
		t.Error("Expected 'route' module not found in output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("metrics")) {
		t.Errorf("Expected 'metrics' message not found in output: %s", output)
	}
}

// TestLogExporterPushEmpty tests pushing empty summaries
func TestLogExporterPushEmpty(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	ctx := context.Background()
	err := exp.Push(ctx, nil)

	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	err = exp.Push(ctx, []analytics.Summary{})

	if err != nil {
		t.Fatalf("Push with empty slice failed: %v", err)
	}
}

// TestLogExporterSetInterval tests setting the interval
func TestLogExporterSetInterval(t *testing.T) {
	exp := NewLogExporter(LogConfig{Interval: 30})

	if exp.interval != 30 {
		t.Errorf("initial interval = %d, want 30", exp.interval)
	}

	exp.SetInterval(60)

	if exp.interval != 60 {
		t.Errorf("updated interval = %d, want 60", exp.interval)
	}
}

// TestLogExporterStream tests streaming events
func TestLogExporterStream(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	event := analytics.Event{
		ID:           "test-123",
		Timestamp:    time.Now(),
		Channel:      "http",
		Module:       "user",
		Action:       "create",
		RecordID:     "user-456",
		UserID:       "admin",
		DurationNS:   1500000,
		MemoryBytes:  1024,
		RequestBytes: 256,
		Success:      true,
		StatusCode:   201,
	}

	ctx := context.Background()
	err := exp.Stream(ctx, event)

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Check that event was logged as JSON
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("analytics event")) {
		t.Errorf("Expected 'analytics event' message not found: %s", output)
	}
}

// TestLogExporterStreamWithError tests streaming event with error
func TestLogExporterStreamWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exp := NewLogExporter(LogConfig{Logger: logger})

	event := analytics.Event{
		ID:         "test-error",
		Module:     "user",
		Action:     "create",
		Success:    false,
		StatusCode: 500,
		Error:      "internal server error",
	}

	ctx := context.Background()
	err := exp.Stream(ctx, event)

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
}

// TestNewNoopExporter tests creating a NoopExporter
func TestNewNoopExporter(t *testing.T) {
	exp := NewNoopExporter()

	if exp == nil {
		t.Fatal("NewNoopExporter returned nil")
	}
}

// TestNoopExporterName tests the NoopExporter Name method
func TestNoopExporterName(t *testing.T) {
	exp := NewNoopExporter()

	if exp.Name() != "noop" {
		t.Errorf("Name() = %q, want %q", exp.Name(), "noop")
	}
}

// TestNoopExporterStart tests the NoopExporter Start method
func TestNoopExporterStart(t *testing.T) {
	exp := NewNoopExporter()
	ctx := context.Background()

	err := exp.Start(ctx)

	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

// TestNoopExporterStop tests the NoopExporter Stop method
func TestNoopExporterStop(t *testing.T) {
	exp := NewNoopExporter()
	ctx := context.Background()

	err := exp.Stop(ctx)

	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// TestNoopExporterPush tests the NoopExporter Push method
func TestNoopExporterPush(t *testing.T) {
	exp := NewNoopExporter()
	ctx := context.Background()

	summaries := []analytics.Summary{
		{Module: "test", TotalRequests: 100},
	}

	err := exp.Push(ctx, summaries)

	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

// TestNoopExporterSetInterval tests the NoopExporter SetInterval method
func TestNoopExporterSetInterval(t *testing.T) {
	exp := NewNoopExporter()

	// This should be a no-op and not panic
	exp.SetInterval(60)
}

// TestNoopExporterStream tests the NoopExporter Stream method
func TestNoopExporterStream(t *testing.T) {
	exp := NewNoopExporter()
	ctx := context.Background()

	event := analytics.Event{
		ID:     "test-123",
		Module: "user",
		Action: "create",
	}

	err := exp.Stream(ctx, event)

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
}

// TestLogConfigDefaults tests LogConfig with default values
func TestLogConfigDefaults(t *testing.T) {
	cfg := LogConfig{}

	if cfg.Interval != 0 {
		t.Errorf("Default interval = %d, want 0", cfg.Interval)
	}

	if cfg.Store != nil {
		t.Error("Default store should be nil")
	}
}

// TestLogExporterWithStore tests LogExporter with a store
func TestLogExporterWithStore(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	store := &MockStore{
		summaries: []analytics.Summary{
			{Module: "test", TotalRequests: 50},
		},
	}

	exp := NewLogExporter(LogConfig{
		Logger: logger,
		Store:  store,
	})

	if exp.store != store {
		t.Error("Store not set correctly")
	}
}

// TestLogExporterDoneChannel tests that the done channel is closed on stop
func TestLogExporterDoneChannel(t *testing.T) {
	exp := NewLogExporter(LogConfig{
		Logger: zerolog.New(nil),
	})

	ctx := context.Background()
	err := exp.Stop(ctx)

	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Check that done channel is closed
	select {
	case <-exp.done:
		// Channel is closed, this is expected
	default:
		t.Error("done channel should be closed after Stop")
	}
}
