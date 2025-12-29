package exporter

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/artpar/apigate/core/analytics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusExporter exposes metrics for Prometheus scraping.
type PrometheusExporter struct {
	mu       sync.RWMutex
	store    analytics.Store
	registry *prometheus.Registry
	prefix   string
	labels   map[string]string

	// Metrics
	requestsTotal    *prometheus.CounterVec
	requestsSuccess  *prometheus.CounterVec
	requestsError    *prometheus.CounterVec
	durationSeconds  *prometheus.HistogramVec
	requestBytes     *prometheus.CounterVec
	responseBytes    *prometheus.CounterVec
	costUnits        *prometheus.CounterVec

	// Last collected values for delta calculation
	lastCollect time.Time
	lastValues  map[string]float64
}

// PrometheusConfig configures the Prometheus exporter.
type PrometheusConfig struct {
	// Store is the analytics store to query.
	Store analytics.Store

	// Prefix is added to all metric names (default: "apigate").
	Prefix string

	// Labels are added to all metrics.
	Labels map[string]string

	// Buckets for duration histogram (in seconds).
	// Default: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
	Buckets []float64
}

// DefaultPrometheusBuckets returns default histogram buckets.
func DefaultPrometheusBuckets() []float64 {
	return []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
}

// NewPrometheusExporter creates a new Prometheus exporter.
func NewPrometheusExporter(cfg PrometheusConfig) *PrometheusExporter {
	if cfg.Prefix == "" {
		cfg.Prefix = "apigate"
	}
	if cfg.Buckets == nil {
		cfg.Buckets = DefaultPrometheusBuckets()
	}
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}

	reg := prometheus.NewRegistry()

	// Standard labels for all metrics
	labelNames := []string{"channel", "module", "action"}

	e := &PrometheusExporter{
		store:      cfg.Store,
		registry:   reg,
		prefix:     cfg.Prefix,
		labels:     cfg.Labels,
		lastValues: make(map[string]float64),
	}

	// Define metrics
	e.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_requests_total",
			Help: "Total number of requests processed",
		},
		labelNames,
	)

	e.requestsSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_requests_success_total",
			Help: "Total number of successful requests",
		},
		labelNames,
	)

	e.requestsError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_requests_error_total",
			Help: "Total number of failed requests",
		},
		labelNames,
	)

	e.durationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    cfg.Prefix + "_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: cfg.Buckets,
		},
		labelNames,
	)

	e.requestBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_request_bytes_total",
			Help: "Total request size in bytes",
		},
		labelNames,
	)

	e.responseBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_response_bytes_total",
			Help: "Total response size in bytes",
		},
		labelNames,
	)

	e.costUnits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.Prefix + "_cost_units_total",
			Help: "Total cost units consumed",
		},
		labelNames,
	)

	// Register all metrics
	reg.MustRegister(
		e.requestsTotal,
		e.requestsSuccess,
		e.requestsError,
		e.durationSeconds,
		e.requestBytes,
		e.responseBytes,
		e.costUnits,
	)

	// Register default Go metrics
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return e
}

// Name returns the exporter name.
func (e *PrometheusExporter) Name() string {
	return "prometheus"
}

// Start starts the Prometheus exporter.
func (e *PrometheusExporter) Start(ctx context.Context) error {
	// Initial collection
	return e.Collect(ctx)
}

// Stop stops the Prometheus exporter.
func (e *PrometheusExporter) Stop(ctx context.Context) error {
	return nil
}

// Handler returns the HTTP handler for /metrics endpoint.
func (e *PrometheusExporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Collect gathers metrics from the analytics store.
func (e *PrometheusExporter) Collect(ctx context.Context) error {
	if e.store == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Query aggregated analytics
	summaries, err := e.store.Aggregate(ctx, analytics.AggregateOptions{
		GroupBy: []string{"module", "action", "channel"},
	})
	if err != nil {
		return err
	}

	for _, s := range summaries {
		labels := prometheus.Labels{
			"channel": s.Channel,
			"module":  s.Module,
			"action":  s.Action,
		}

		// Update counters (Prometheus counters are cumulative)
		e.requestsTotal.With(labels).Add(float64(s.TotalRequests))
		e.requestsSuccess.With(labels).Add(float64(s.SuccessRequests))
		e.requestsError.With(labels).Add(float64(s.ErrorRequests))
		e.requestBytes.With(labels).Add(float64(s.TotalRequestBytes))
		e.responseBytes.With(labels).Add(float64(s.TotalResponseBytes))
		e.costUnits.With(labels).Add(s.CostUnits)

		// For histogram, we'd need individual events
		// Using average as an approximation
		if s.TotalRequests > 0 {
			avgSeconds := float64(s.AvgDurationNS) / 1e9
			e.durationSeconds.With(labels).Observe(avgSeconds)
		}
	}

	e.lastCollect = time.Now()
	return nil
}

// CollectFromEvents updates metrics from individual events.
// This provides more accurate histogram data.
func (e *PrometheusExporter) CollectFromEvents(ctx context.Context, events []analytics.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, ev := range events {
		labels := prometheus.Labels{
			"channel": ev.Channel,
			"module":  ev.Module,
			"action":  ev.Action,
		}

		e.requestsTotal.With(labels).Inc()
		if ev.Success {
			e.requestsSuccess.With(labels).Inc()
		} else {
			e.requestsError.With(labels).Inc()
		}

		e.requestBytes.With(labels).Add(float64(ev.RequestBytes))
		e.responseBytes.With(labels).Add(float64(ev.ResponseBytes))

		// Duration histogram
		durationSeconds := float64(ev.DurationNS) / 1e9
		e.durationSeconds.With(labels).Observe(durationSeconds)
	}

	return nil
}

// Registry returns the underlying Prometheus registry.
// Useful for adding custom metrics.
func (e *PrometheusExporter) Registry() *prometheus.Registry {
	return e.registry
}

// WithCustomMetric registers a custom metric with the exporter.
func (e *PrometheusExporter) WithCustomMetric(collector prometheus.Collector) error {
	return e.registry.Register(collector)
}
