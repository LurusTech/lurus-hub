package hub

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUsageAggregator_Record(t *testing.T) {
	ua := NewUsageAggregator(func(ctx context.Context, buckets []UsageBucket) error {
		return nil
	})

	now := time.Now()
	ua.Record(UsageEvent{
		TenantID:     "t1",
		ModelName:    "gpt-4",
		ChannelID:    1,
		Timestamp:    now,
		InputTokens:  100,
		OutputTokens: 50,
		LatencySec:   0.5,
		Quota:        10,
		CostCNY:      0.01,
		Success:      true,
	})
	ua.Record(UsageEvent{
		TenantID:     "t1",
		ModelName:    "gpt-4",
		ChannelID:    1,
		Timestamp:    now.Add(time.Second),
		InputTokens:  200,
		OutputTokens: 100,
		LatencySec:   1.0,
		Quota:        20,
		CostCNY:      0.02,
		Success:      false,
	})

	if ua.PendingBucketCount() != 1 {
		t.Errorf("expected 1 bucket, got %d", ua.PendingBucketCount())
	}

	// Verify aggregation
	ua.mu.Lock()
	for _, b := range ua.buckets {
		if b.RequestCount != 2 {
			t.Errorf("expected 2 requests, got %d", b.RequestCount)
		}
		if b.ErrorCount != 1 {
			t.Errorf("expected 1 error, got %d", b.ErrorCount)
		}
		if b.InputTokens != 300 {
			t.Errorf("expected 300 input tokens, got %d", b.InputTokens)
		}
		if b.OutputTokens != 150 {
			t.Errorf("expected 150 output tokens, got %d", b.OutputTokens)
		}
	}
	ua.mu.Unlock()
}

func TestUsageAggregator_DifferentWindows(t *testing.T) {
	ua := NewUsageAggregator(func(ctx context.Context, buckets []UsageBucket) error {
		return nil
	})

	// Events in different hours should go to different buckets
	t1 := time.Date(2026, 4, 10, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 10, 11, 30, 0, 0, time.UTC)

	ua.Record(UsageEvent{TenantID: "t1", ModelName: "gpt-4", ChannelID: 1, Timestamp: t1, Success: true})
	ua.Record(UsageEvent{TenantID: "t1", ModelName: "gpt-4", ChannelID: 1, Timestamp: t2, Success: true})

	if ua.PendingBucketCount() != 2 {
		t.Errorf("expected 2 buckets for different hours, got %d", ua.PendingBucketCount())
	}
}

func TestUsageAggregator_FlushSuccess(t *testing.T) {
	var flushed int64

	ua := NewUsageAggregator(func(ctx context.Context, buckets []UsageBucket) error {
		atomic.AddInt64(&flushed, int64(len(buckets)))
		return nil
	})
	ua.flushInterval = 10 * time.Millisecond

	now := time.Now()
	ua.Record(UsageEvent{TenantID: "t1", ModelName: "gpt-4", ChannelID: 1, Timestamp: now, Success: true})
	ua.Record(UsageEvent{TenantID: "t2", ModelName: "gpt-4", ChannelID: 2, Timestamp: now, Success: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ua.Run(ctx)

	if atomic.LoadInt64(&flushed) != 2 {
		t.Errorf("expected 2 buckets flushed, got %d", atomic.LoadInt64(&flushed))
	}
	if ua.PendingBucketCount() != 0 {
		t.Errorf("expected 0 pending after flush, got %d", ua.PendingBucketCount())
	}
}

func TestUsageAggregator_FlushFailurePreservesData(t *testing.T) {
	failCount := 0
	ua := NewUsageAggregator(func(ctx context.Context, buckets []UsageBucket) error {
		failCount++
		if failCount <= 1 {
			return context.DeadlineExceeded
		}
		return nil
	})

	now := time.Now()
	ua.Record(UsageEvent{TenantID: "t1", ModelName: "gpt-4", ChannelID: 1, Timestamp: now, Success: true, InputTokens: 100})

	// First flush fails — data should be preserved
	ua.flushBuckets(context.Background())
	if ua.PendingBucketCount() != 1 {
		t.Errorf("expected data preserved after failed flush, got %d buckets", ua.PendingBucketCount())
	}

	// Second flush succeeds
	ua.flushBuckets(context.Background())
	if ua.PendingBucketCount() != 0 {
		t.Errorf("expected 0 pending after successful flush, got %d", ua.PendingBucketCount())
	}
}

func TestUsageAggregator_Concurrent(t *testing.T) {
	var flushedCount atomic.Int64

	ua := NewUsageAggregator(func(ctx context.Context, buckets []UsageBucket) error {
		flushedCount.Add(int64(len(buckets)))
		return nil
	})

	var wg sync.WaitGroup
	now := time.Now()

	// 10 goroutines recording events concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ua.Record(UsageEvent{
					TenantID:     tenant,
					ModelName:    "gpt-4",
					ChannelID:    1,
					Timestamp:    now,
					InputTokens:  10,
					OutputTokens: 5,
					LatencySec:   0.1,
					Success:      true,
				})
			}
		}(string(rune('A' + i)))
	}

	wg.Wait()

	// Flush should work after concurrent writes
	ua.flushBuckets(context.Background())
	if ua.PendingBucketCount() != 0 {
		t.Errorf("expected 0 pending after flush, got %d", ua.PendingBucketCount())
	}
}

func TestUsageBucket_AvgLatency(t *testing.T) {
	b := &UsageBucket{RequestCount: 4, TotalLatency: 10.0}
	if b.AvgLatency() != 2.5 {
		t.Errorf("expected 2.5, got %f", b.AvgLatency())
	}

	empty := &UsageBucket{}
	if empty.AvgLatency() != 0 {
		t.Error("expected 0 for empty bucket")
	}
}

func TestUsageBucket_ErrorRate(t *testing.T) {
	b := &UsageBucket{RequestCount: 10, ErrorCount: 3}
	if b.ErrorRate() != 0.3 {
		t.Errorf("expected 0.3, got %f", b.ErrorRate())
	}
}
