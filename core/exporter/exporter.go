// Package exporter provides pluggable metrics export capabilities.
// Implementations include Prometheus, DataDog, StatsD, etc.
package exporter

import (
	"context"
	"net/http"

	"github.com/artpar/apigate/core/analytics"
)

// Exporter is the base interface for all metrics exporters.
type Exporter interface {
	// Name returns the exporter identifier (e.g., "prometheus", "datadog").
	Name() string

	// Start starts the exporter.
	Start(ctx context.Context) error

	// Stop stops the exporter gracefully.
	Stop(ctx context.Context) error
}

// PushExporter pushes metrics to a remote endpoint.
// Examples: DataDog, StatsD, InfluxDB, CloudWatch.
type PushExporter interface {
	Exporter

	// Push sends metrics to the remote endpoint.
	Push(ctx context.Context, summaries []analytics.Summary) error

	// SetInterval sets the push interval.
	SetInterval(seconds int)
}

// PullExporter exposes metrics for scraping.
// Examples: Prometheus, OpenMetrics.
type PullExporter interface {
	Exporter

	// Handler returns an HTTP handler for the metrics endpoint.
	Handler() http.Handler

	// Collect gathers current metrics for exposure.
	Collect(ctx context.Context) error
}

// StreamExporter streams metrics in real-time.
// Examples: Kafka, NATS, WebSocket.
type StreamExporter interface {
	Exporter

	// Stream sends an event immediately.
	Stream(ctx context.Context, event analytics.Event) error
}

// Config is the base configuration for exporters.
type Config struct {
	// Enabled determines if the exporter is active.
	Enabled bool

	// Name is the exporter type (prometheus, datadog, etc.).
	Name string

	// Endpoint is the target URL for push exporters.
	Endpoint string

	// Interval is the push interval in seconds (for push exporters).
	Interval int

	// Labels are additional labels/tags to add to all metrics.
	Labels map[string]string

	// Extra holds exporter-specific configuration.
	Extra map[string]any
}

// Metric represents a single metric data point.
type Metric struct {
	// Name is the metric name (e.g., "apigate_requests_total").
	Name string

	// Type is the metric type.
	Type MetricType

	// Value is the metric value.
	Value float64

	// Labels are the metric dimensions.
	Labels map[string]string

	// Help is the metric description.
	Help string
}

// MetricType defines the type of metric.
type MetricType int

const (
	// Counter is a monotonically increasing value.
	Counter MetricType = iota

	// Gauge is a value that can go up and down.
	Gauge

	// Histogram is a distribution of values.
	Histogram

	// Summary is a distribution with quantiles.
	Summary
)

// String returns the string representation of the metric type.
func (t MetricType) String() string {
	switch t {
	case Counter:
		return "counter"
	case Gauge:
		return "gauge"
	case Histogram:
		return "histogram"
	case Summary:
		return "summary"
	default:
		return "unknown"
	}
}

// Registry manages multiple exporters.
type Registry struct {
	exporters map[string]Exporter
	analytics analytics.Store
}

// NewRegistry creates a new exporter registry.
func NewRegistry(store analytics.Store) *Registry {
	return &Registry{
		exporters: make(map[string]Exporter),
		analytics: store,
	}
}

// Register adds an exporter to the registry.
func (r *Registry) Register(exp Exporter) error {
	r.exporters[exp.Name()] = exp
	return nil
}

// Get returns an exporter by name.
func (r *Registry) Get(name string) (Exporter, bool) {
	exp, ok := r.exporters[name]
	return exp, ok
}

// All returns all registered exporters.
func (r *Registry) All() []Exporter {
	result := make([]Exporter, 0, len(r.exporters))
	for _, exp := range r.exporters {
		result = append(result, exp)
	}
	return result
}

// PullExporters returns all pull-based exporters (for HTTP handler mounting).
func (r *Registry) PullExporters() []PullExporter {
	var result []PullExporter
	for _, exp := range r.exporters {
		if pull, ok := exp.(PullExporter); ok {
			result = append(result, pull)
		}
	}
	return result
}

// Start starts all registered exporters.
func (r *Registry) Start(ctx context.Context) error {
	for _, exp := range r.exporters {
		if err := exp.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops all registered exporters.
func (r *Registry) Stop(ctx context.Context) error {
	for _, exp := range r.exporters {
		if err := exp.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Analytics returns the analytics store for exporters to query.
func (r *Registry) Analytics() analytics.Store {
	return r.analytics
}

// MetricBuilder helps construct metrics from analytics data.
type MetricBuilder struct {
	prefix string
	labels map[string]string
}

// NewMetricBuilder creates a new metric builder.
func NewMetricBuilder(prefix string) *MetricBuilder {
	return &MetricBuilder{
		prefix: prefix,
		labels: make(map[string]string),
	}
}

// WithLabels adds default labels to all metrics.
func (b *MetricBuilder) WithLabels(labels map[string]string) *MetricBuilder {
	for k, v := range labels {
		b.labels[k] = v
	}
	return b
}

// Counter creates a counter metric.
func (b *MetricBuilder) Counter(name string, value float64, labels map[string]string, help string) Metric {
	return b.metric(name, Counter, value, labels, help)
}

// Gauge creates a gauge metric.
func (b *MetricBuilder) Gauge(name string, value float64, labels map[string]string, help string) Metric {
	return b.metric(name, Gauge, value, labels, help)
}

// Histogram creates a histogram metric.
func (b *MetricBuilder) Histogram(name string, value float64, labels map[string]string, help string) Metric {
	return b.metric(name, Histogram, value, labels, help)
}

func (b *MetricBuilder) metric(name string, typ MetricType, value float64, labels map[string]string, help string) Metric {
	// Merge default labels with provided labels
	allLabels := make(map[string]string)
	for k, v := range b.labels {
		allLabels[k] = v
	}
	for k, v := range labels {
		allLabels[k] = v
	}

	fullName := name
	if b.prefix != "" {
		fullName = b.prefix + "_" + name
	}

	return Metric{
		Name:   fullName,
		Type:   typ,
		Value:  value,
		Labels: allLabels,
		Help:   help,
	}
}

// SummaryToMetrics converts analytics summaries to exportable metrics.
func SummaryToMetrics(summaries []analytics.Summary, builder *MetricBuilder) []Metric {
	var metrics []Metric

	for _, s := range summaries {
		labels := map[string]string{}
		if s.Module != "" {
			labels["module"] = s.Module
		}
		if s.Action != "" {
			labels["action"] = s.Action
		}

		// Request counts
		metrics = append(metrics,
			builder.Counter("requests_total", float64(s.TotalRequests), labels,
				"Total number of requests"),
			builder.Counter("requests_success", float64(s.SuccessRequests), labels,
				"Number of successful requests"),
			builder.Counter("requests_error", float64(s.ErrorRequests), labels,
				"Number of failed requests"),
		)

		// Latency
		metrics = append(metrics,
			builder.Gauge("duration_avg_ns", float64(s.AvgDurationNS), labels,
				"Average request duration in nanoseconds"),
			builder.Gauge("duration_min_ns", float64(s.MinDurationNS), labels,
				"Minimum request duration in nanoseconds"),
			builder.Gauge("duration_max_ns", float64(s.MaxDurationNS), labels,
				"Maximum request duration in nanoseconds"),
		)

		// Resource usage
		metrics = append(metrics,
			builder.Counter("memory_bytes_total", float64(s.TotalMemoryBytes), labels,
				"Total memory used in bytes"),
			builder.Counter("request_bytes_total", float64(s.TotalRequestBytes), labels,
				"Total request size in bytes"),
			builder.Counter("response_bytes_total", float64(s.TotalResponseBytes), labels,
				"Total response size in bytes"),
		)

		// Cost
		metrics = append(metrics,
			builder.Counter("cost_units_total", s.CostUnits, labels,
				"Total cost units consumed"),
		)
	}

	return metrics
}
