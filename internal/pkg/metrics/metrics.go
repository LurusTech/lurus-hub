// Package metrics provides Prometheus metrics for the API gateway.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "lurus"
	subsystem = "gateway"
)

var (
	// RequestsTotal counts total requests by method, path, and status
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// RequestDuration measures request latency in seconds
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// RelayRequestsTotal counts relay requests by provider and model
	RelayRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "relay_requests_total",
			Help:      "Total number of relay requests to upstream providers",
		},
		[]string{"provider", "model", "status"},
	)

	// RelayDuration measures upstream API latency
	RelayDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "relay_duration_seconds",
			Help:      "Upstream provider API latency in seconds",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"provider", "model"},
	)

	// ChannelSelectDuration measures channel selection latency
	ChannelSelectDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "channel_select_duration_seconds",
			Help:      "Channel selection latency in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25},
		},
	)

	// TokensProcessed counts tokens processed
	TokensProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "tokens_processed_total",
			Help:      "Total tokens processed (input + output)",
		},
		[]string{"provider", "model", "type"}, // type: input, output
	)

	// QuotaConsumed tracks quota consumption
	QuotaConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "quota_consumed_total",
			Help:      "Total quota consumed",
		},
		[]string{"tenant_id", "user_id"},
	)

	// RetryAttempts counts retry attempts
	RetryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "retry_attempts_total",
			Help:      "Total retry attempts",
		},
		[]string{"provider", "reason"},
	)

	// ActiveConnections tracks current active connections
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "active_connections",
			Help:      "Number of active connections",
		},
	)

	// ChannelHealth tracks channel availability
	ChannelHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "channel_health",
			Help:      "Channel health status (1=healthy, 0=unhealthy)",
		},
		[]string{"channel_id", "channel_name", "provider"},
	)

	// CacheHits tracks cache hit/miss ratio
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hits_total",
			Help:      "Cache hit/miss counts",
		},
		[]string{"cache_type", "result"}, // result: hit, miss
	)
)

// RecordRelayRequest records a relay request with its outcome
func RecordRelayRequest(provider, model, status string, durationSec float64) {
	RelayRequestsTotal.WithLabelValues(provider, model, status).Inc()
	RelayDuration.WithLabelValues(provider, model).Observe(durationSec)
}

// RecordTokens records token usage
func RecordTokens(provider, model string, inputTokens, outputTokens int) {
	if inputTokens > 0 {
		TokensProcessed.WithLabelValues(provider, model, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		TokensProcessed.WithLabelValues(provider, model, "output").Add(float64(outputTokens))
	}
}

// RecordQuotaConsumed records quota consumption
func RecordQuotaConsumed(tenantID, userID string, quota int64) {
	QuotaConsumed.WithLabelValues(tenantID, userID).Add(float64(quota))
}
