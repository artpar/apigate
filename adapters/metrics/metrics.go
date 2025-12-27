// Package metrics provides Prometheus metrics collection for APIGate.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all Prometheus metrics for APIGate.
type Collector struct {
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	// Auth metrics
	AuthFailures *prometheus.CounterVec

	// Rate limit metrics
	RateLimitHits   *prometheus.CounterVec
	RateLimitTokens *prometheus.GaugeVec

	// Usage metrics
	UsageRequests *prometheus.CounterVec
	UsageBytes    *prometheus.CounterVec

	// Upstream metrics
	UpstreamDuration  *prometheus.HistogramVec
	UpstreamErrors    *prometheus.CounterVec
	UpstreamInFlight  prometheus.Gauge

	// Config metrics
	ConfigReloads      prometheus.Counter
	ConfigReloadErrors prometheus.Counter
	ConfigLastReload   prometheus.Gauge
}

// New creates a new metrics collector with all metrics registered.
func New() *Collector {
	return &Collector{
		// Request metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "requests_total",
				Help:      "Total number of requests processed",
			},
			[]string{"method", "path", "status", "plan_id"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "apigate",
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "requests_in_flight",
				Help:      "Number of requests currently being processed",
			},
		),

		// Auth metrics
		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"reason"},
		),

		// Rate limit metrics
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "rate_limit_hits_total",
				Help:      "Total number of rate limit hits",
			},
			[]string{"plan_id", "user_id"},
		),
		RateLimitTokens: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "rate_limit_tokens",
				Help:      "Current rate limit tokens available",
			},
			[]string{"plan_id", "user_id"},
		),

		// Usage metrics
		UsageRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "usage_requests_total",
				Help:      "Total requests by user/plan",
			},
			[]string{"user_id", "plan_id"},
		),
		UsageBytes: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "usage_bytes_total",
				Help:      "Total bytes transferred by user/plan",
			},
			[]string{"user_id", "plan_id", "direction"},
		),

		// Upstream metrics
		UpstreamDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "apigate",
				Name:      "upstream_duration_seconds",
				Help:      "Upstream request duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"method", "status"},
		),
		UpstreamErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "upstream_errors_total",
				Help:      "Total number of upstream errors",
			},
			[]string{"type"},
		),
		UpstreamInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "upstream_requests_in_flight",
				Help:      "Number of requests currently being sent to upstream",
			},
		),

		// Config metrics
		ConfigReloads: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "config_reloads_total",
				Help:      "Total number of successful config reloads",
			},
		),
		ConfigReloadErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "config_reload_errors_total",
				Help:      "Total number of config reload errors",
			},
		),
		ConfigLastReload: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "config_last_reload_timestamp",
				Help:      "Unix timestamp of last successful config reload",
			},
		),
	}
}

// NewWithRegistry creates a new metrics collector with a custom registry.
// Useful for testing to avoid global state.
func NewWithRegistry(reg prometheus.Registerer) *Collector {
	factory := promauto.With(reg)

	return &Collector{
		RequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "requests_total",
				Help:      "Total number of requests processed",
			},
			[]string{"method", "path", "status", "plan_id"},
		),
		RequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "apigate",
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		RequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "requests_in_flight",
				Help:      "Number of requests currently being processed",
			},
		),
		AuthFailures: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"reason"},
		),
		RateLimitHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "rate_limit_hits_total",
				Help:      "Total number of rate limit hits",
			},
			[]string{"plan_id", "user_id"},
		),
		RateLimitTokens: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "rate_limit_tokens",
				Help:      "Current rate limit tokens available",
			},
			[]string{"plan_id", "user_id"},
		),
		UsageRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "usage_requests_total",
				Help:      "Total requests by user/plan",
			},
			[]string{"user_id", "plan_id"},
		),
		UsageBytes: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "usage_bytes_total",
				Help:      "Total bytes transferred by user/plan",
			},
			[]string{"user_id", "plan_id", "direction"},
		),
		UpstreamDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "apigate",
				Name:      "upstream_duration_seconds",
				Help:      "Upstream request duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"method", "status"},
		),
		UpstreamErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "upstream_errors_total",
				Help:      "Total number of upstream errors",
			},
			[]string{"type"},
		),
		UpstreamInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "upstream_requests_in_flight",
				Help:      "Number of requests currently being sent to upstream",
			},
		),
		ConfigReloads: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "config_reloads_total",
				Help:      "Total number of successful config reloads",
			},
		),
		ConfigReloadErrors: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "apigate",
				Name:      "config_reload_errors_total",
				Help:      "Total number of config reload errors",
			},
		),
		ConfigLastReload: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "apigate",
				Name:      "config_last_reload_timestamp",
				Help:      "Unix timestamp of last successful config reload",
			},
		),
	}
}

// NormalizePath reduces cardinality by normalizing path patterns.
// e.g., /users/123/orders/456 -> /users/:id/orders/:id
func NormalizePath(path string) string {
	// For now, just return the path as-is
	// TODO: Add path normalization for common patterns
	// This prevents high cardinality from dynamic path segments
	if len(path) > 50 {
		return path[:50] + "..."
	}
	return path
}
