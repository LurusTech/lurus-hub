package hub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// UsageBucket represents an aggregated usage snapshot for a
// (tenant, model, channel) tuple within a time window.
type UsageBucket struct {
	TenantID    string
	ModelName   string
	ChannelID   int
	WindowStart time.Time     // start of the aggregation window (truncated to hour)
	Duration    time.Duration // window size (typically 1 hour)

	RequestCount  int64
	ErrorCount    int64
	InputTokens   int64
	OutputTokens  int64
	TotalLatency  float64 // sum of latencies in seconds
	QuotaConsumed int64   // internal quota units
	CostCNY       float64 // estimated cost in CNY
}

// AvgLatency returns the average latency for the bucket.
func (b *UsageBucket) AvgLatency() float64 {
	if b.RequestCount == 0 {
		return 0
	}
	return b.TotalLatency / float64(b.RequestCount)
}

// ErrorRate returns the error rate for the bucket.
func (b *UsageBucket) ErrorRate() float64 {
	if b.RequestCount == 0 {
		return 0
	}
	return float64(b.ErrorCount) / float64(b.RequestCount)
}

// BucketKey uniquely identifies a usage bucket.
type BucketKey struct {
	TenantID    string
	ModelName   string
	ChannelID   int
	WindowStart int64 // unix timestamp
}

// UsageEvent is a single usage observation emitted by the relay pipeline.
type UsageEvent struct {
	TenantID     string
	ModelName    string
	ChannelID    int
	Timestamp    time.Time
	InputTokens  int64
	OutputTokens int64
	LatencySec   float64
	Quota        int64
	CostCNY      float64
	Success      bool
}

// FlushFunc is called when accumulated buckets are ready to be persisted.
// The implementation should write to the database and return any error.
type FlushFunc func(ctx context.Context, buckets []UsageBucket) error

// UsageAggregator accumulates usage events in memory and periodically
// flushes aggregated buckets to persistent storage. This decouples the
// hot relay path from database writes, improving latency and reliability.
//
// Design decisions:
//   - In-memory accumulation avoids per-request DB writes on the hot path
//   - Hourly bucketing enables efficient time-series queries
//   - Flush failures are logged but do not block new events (graceful degradation)
//   - Thread-safe: Record() can be called from any goroutine
type UsageAggregator struct {
	mu      sync.Mutex
	buckets map[BucketKey]*UsageBucket
	flush   FlushFunc

	windowSize    time.Duration
	flushInterval time.Duration
}

// NewUsageAggregator creates an aggregator with the given flush function.
// windowSize controls the time bucketing granularity (default: 1 hour).
// flushInterval controls how often accumulated data is flushed (default: 5 minutes).
func NewUsageAggregator(flush FlushFunc) *UsageAggregator {
	return &UsageAggregator{
		buckets:       make(map[BucketKey]*UsageBucket),
		flush:         flush,
		windowSize:    time.Hour,
		flushInterval: 5 * time.Minute,
	}
}

// Record adds a usage event to the appropriate bucket.
// This is designed for the hot path — no allocations, no I/O.
func (ua *UsageAggregator) Record(event UsageEvent) {
	windowStart := event.Timestamp.Truncate(ua.windowSize)
	key := BucketKey{
		TenantID:    event.TenantID,
		ModelName:   event.ModelName,
		ChannelID:   event.ChannelID,
		WindowStart: windowStart.Unix(),
	}

	ua.mu.Lock()
	defer ua.mu.Unlock()

	b, ok := ua.buckets[key]
	if !ok {
		b = &UsageBucket{
			TenantID:    event.TenantID,
			ModelName:   event.ModelName,
			ChannelID:   event.ChannelID,
			WindowStart: windowStart,
			Duration:    ua.windowSize,
		}
		ua.buckets[key] = b
	}

	b.RequestCount++
	b.InputTokens += event.InputTokens
	b.OutputTokens += event.OutputTokens
	b.TotalLatency += event.LatencySec
	b.QuotaConsumed += event.Quota
	b.CostCNY += event.CostCNY
	if !event.Success {
		b.ErrorCount++
	}
}

// Run starts the periodic flush loop. Blocks until ctx is cancelled.
func (ua *UsageAggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(ua.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush on shutdown
			ua.flushBuckets(context.Background())
			slog.Info("hub/usage-aggregator: stopped")
			return
		case <-ticker.C:
			ua.flushBuckets(ctx)
		}
	}
}

// flushBuckets drains accumulated buckets and calls the flush function.
func (ua *UsageAggregator) flushBuckets(ctx context.Context) {
	ua.mu.Lock()
	if len(ua.buckets) == 0 {
		ua.mu.Unlock()
		return
	}
	// Swap the map to minimize lock hold time
	old := ua.buckets
	ua.buckets = make(map[BucketKey]*UsageBucket, len(old)/2)
	ua.mu.Unlock()

	buckets := make([]UsageBucket, 0, len(old))
	for _, b := range old {
		buckets = append(buckets, *b)
	}

	if err := ua.flush(ctx, buckets); err != nil {
		slog.Error("hub/usage-aggregator: flush failed, data will be retried next cycle",
			"bucket_count", len(buckets),
			"err", err,
		)
		// Re-merge failed buckets back (best-effort data preservation)
		ua.mu.Lock()
		for _, b := range buckets {
			key := BucketKey{
				TenantID:    b.TenantID,
				ModelName:   b.ModelName,
				ChannelID:   b.ChannelID,
				WindowStart: b.WindowStart.Unix(),
			}
			if existing, ok := ua.buckets[key]; ok {
				existing.RequestCount += b.RequestCount
				existing.ErrorCount += b.ErrorCount
				existing.InputTokens += b.InputTokens
				existing.OutputTokens += b.OutputTokens
				existing.TotalLatency += b.TotalLatency
				existing.QuotaConsumed += b.QuotaConsumed
				existing.CostCNY += b.CostCNY
			} else {
				copied := b
				ua.buckets[key] = &copied
			}
		}
		ua.mu.Unlock()
		return
	}

	slog.Info("hub/usage-aggregator: flushed",
		"bucket_count", len(buckets),
		"total_requests", sumRequests(buckets),
	)
}

// PendingBucketCount returns the number of unflushed buckets (for monitoring).
func (ua *UsageAggregator) PendingBucketCount() int {
	ua.mu.Lock()
	defer ua.mu.Unlock()
	return len(ua.buckets)
}

func sumRequests(buckets []UsageBucket) int64 {
	var total int64
	for _, b := range buckets {
		total += b.RequestCount
	}
	return total
}

// FormatBucketSummary returns a human-readable summary for logging.
func FormatBucketSummary(b *UsageBucket) string {
	return fmt.Sprintf("tenant=%s model=%s channel=%d requests=%d errors=%d tokens=%d+%d cost=%.4f",
		b.TenantID, b.ModelName, b.ChannelID,
		b.RequestCount, b.ErrorCount,
		b.InputTokens, b.OutputTokens,
		b.CostCNY,
	)
}
