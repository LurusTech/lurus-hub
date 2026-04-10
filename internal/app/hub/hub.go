package hub

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Hub is the singleton entry point for all Hub data processing capabilities.
// It is initialized once at startup and provides thread-safe access to
// the ChannelScorer and UsageAggregator.
var (
	instance *Hub
	once     sync.Once
)

// Hub holds the data processing components.
type Hub struct {
	Scorer     *ChannelScorer
	Aggregator *UsageAggregator
}

// Init initializes the Hub singleton with the given flush function.
// Safe to call multiple times — only the first call takes effect.
func Init(flushFn FlushFunc) *Hub {
	once.Do(func() {
		instance = &Hub{
			Scorer:     NewChannelScorer(),
			Aggregator: NewUsageAggregator(flushFn),
		}
		slog.Info("hub: initialized")
	})
	return instance
}

// Get returns the Hub singleton. Returns nil if Init() was not called.
func Get() *Hub {
	return instance
}

// RunBackgroundTasks starts all Hub background goroutines.
// Call this after Init() during application startup.
// Blocks until ctx is cancelled.
func (h *Hub) RunBackgroundTasks(ctx context.Context) {
	var wg sync.WaitGroup

	// 1. Usage aggregator flush loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.Aggregator.Run(ctx)
	}()

	// 2. Channel scorer decay loop (every 1 minute)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("hub/scorer-decay: stopped")
				return
			case <-ticker.C:
				h.Scorer.Decay()
			}
		}
	}()

	wg.Wait()
}

// RecordRelayOutcome records the outcome of a relay request for both
// channel scoring and usage aggregation. This is the primary integration
// point called from the relay handler.
func RecordRelayOutcome(channelID int, success bool, latencySec float64,
	tenantID, modelName string, inputTokens, outputTokens int64,
	quota int64, costCNY float64) {

	h := Get()
	if h == nil {
		return // Hub not initialized (e.g., during tests)
	}

	// Update channel scorer
	if success {
		h.Scorer.RecordSuccess(channelID, latencySec)
	} else {
		h.Scorer.RecordFailure(channelID)
	}

	// Emit usage event to aggregator
	h.Aggregator.Record(UsageEvent{
		TenantID:     tenantID,
		ModelName:    modelName,
		ChannelID:    channelID,
		Timestamp:    time.Now(),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		LatencySec:   latencySec,
		Quota:        quota,
		CostCNY:      costCNY,
		Success:      success,
	})
}
