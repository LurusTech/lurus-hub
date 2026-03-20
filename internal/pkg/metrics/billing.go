package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BillingPreAuthDuration measures pre-authorization latency.
	BillingPreAuthDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "billing",
			Name:      "preauth_duration_seconds",
			Help:      "Pre-authorization call latency in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1},
		},
	)

	// BillingSettleTotal counts settle operations by status.
	BillingSettleTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "billing",
			Name:      "settle_total",
			Help:      "Total settle operations by status (success/error)",
		},
		[]string{"status"},
	)

	// BillingOutboxPending tracks the number of pending outbox entries.
	BillingOutboxPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "billing",
			Name:      "outbox_pending",
			Help:      "Number of pending billing outbox entries",
		},
	)

	// BillingOutboxFailedTotal counts permanently failed outbox entries.
	BillingOutboxFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "billing",
			Name:      "outbox_failed_total",
			Help:      "Total billing outbox entries that permanently failed after max retries",
		},
	)
)
