package exporter

import (
	"context"
	"encoding/json"

	"github.com/artpar/apigate/core/analytics"
	"github.com/rs/zerolog"
)

// LogExporter exports metrics to a logger.
// Useful for debugging and development.
type LogExporter struct {
	logger   zerolog.Logger
	store    analytics.Store
	interval int
	done     chan struct{}
}

// LogConfig configures the log exporter.
type LogConfig struct {
	// Logger is the zerolog logger to use.
	Logger zerolog.Logger

	// Store is the analytics store to query.
	Store analytics.Store

	// Interval is the logging interval in seconds (0 = disabled).
	Interval int
}

// NewLogExporter creates a new log exporter.
func NewLogExporter(cfg LogConfig) *LogExporter {
	return &LogExporter{
		logger:   cfg.Logger,
		store:    cfg.Store,
		interval: cfg.Interval,
		done:     make(chan struct{}),
	}
}

// Name returns the exporter name.
func (e *LogExporter) Name() string {
	return "log"
}

// Start starts the log exporter.
func (e *LogExporter) Start(ctx context.Context) error {
	e.logger.Info().Msg("log exporter started")
	return nil
}

// Stop stops the log exporter.
func (e *LogExporter) Stop(ctx context.Context) error {
	close(e.done)
	e.logger.Info().Msg("log exporter stopped")
	return nil
}

// Push logs the metrics.
func (e *LogExporter) Push(ctx context.Context, summaries []analytics.Summary) error {
	for _, s := range summaries {
		e.logger.Info().
			Str("module", s.Module).
			Str("action", s.Action).
			Int64("total_requests", s.TotalRequests).
			Int64("success_requests", s.SuccessRequests).
			Int64("error_requests", s.ErrorRequests).
			Int64("avg_duration_ns", s.AvgDurationNS).
			Float64("cost_units", s.CostUnits).
			Msg("metrics")
	}
	return nil
}

// SetInterval sets the push interval.
func (e *LogExporter) SetInterval(seconds int) {
	e.interval = seconds
}

// Stream logs an event immediately.
func (e *LogExporter) Stream(ctx context.Context, event analytics.Event) error {
	data, _ := json.Marshal(event)
	e.logger.Debug().RawJSON("event", data).Msg("analytics event")
	return nil
}

// NoopExporter discards all metrics.
// Useful as a placeholder or for testing.
type NoopExporter struct{}

// NewNoopExporter creates a new noop exporter.
func NewNoopExporter() *NoopExporter {
	return &NoopExporter{}
}

// Name returns the exporter name.
func (e *NoopExporter) Name() string {
	return "noop"
}

// Start is a no-op.
func (e *NoopExporter) Start(ctx context.Context) error {
	return nil
}

// Stop is a no-op.
func (e *NoopExporter) Stop(ctx context.Context) error {
	return nil
}

// Push discards the metrics.
func (e *NoopExporter) Push(ctx context.Context, summaries []analytics.Summary) error {
	return nil
}

// SetInterval is a no-op.
func (e *NoopExporter) SetInterval(seconds int) {}

// Stream discards the event.
func (e *NoopExporter) Stream(ctx context.Context, event analytics.Event) error {
	return nil
}
