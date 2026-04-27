package repo

import (
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

// BenchmarkGetRandomSatisfiedChannel benchmarks the channel selection hot path
func BenchmarkGetRandomSatisfiedChannel(b *testing.B) {
	// Enable memory cache for benchmark
	common.MemoryCacheEnabled = true

	// Setup test data
	setupBenchmarkChannelCache()

	b.ResetTimer()
	b.Run("SingleChannel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = GetRandomSatisfiedChannel("default", "gpt-4-single", 0)
		}
	})

	b.Run("MultipleChannels", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = GetRandomSatisfiedChannel("default", "gpt-4", 0)
		}
	})

	b.Run("WithRetry", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = GetRandomSatisfiedChannel("default", "gpt-4", 1)
		}
	})

	b.Run("ParallelRequests", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = GetRandomSatisfiedChannel("default", "gpt-4", 0)
			}
		})
	})
}

// setupBenchmarkChannelCache sets up test channel data for benchmarks
func setupBenchmarkChannelCache() {
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()

	// Create test channels
	channelsIDM = make(map[int]*Channel)
	group2model2channels = make(map[string]map[string][]int)
	group2model2channels["default"] = make(map[string][]int)

	// Add channels with different priorities and weights
	for i := 1; i <= 10; i++ {
		priority := int64(i % 3) // 3 different priorities: 0, 1, 2
		weight := uint(i * 10)   // weights: 10, 20, 30, ...
		channelsIDM[i] = &Channel{
			Id:       i,
			Name:     "test-channel-" + string(rune('0'+i)),
			Status:   common.ChannelStatusEnabled,
			Priority: &priority,
			Weight:   &weight,
		}
		group2model2channels["default"]["gpt-4"] = append(
			group2model2channels["default"]["gpt-4"], i,
		)
	}

	// Add single channel for single channel test
	singlePriority := int64(1)
	singleWeight := uint(100)
	channelsIDM[100] = &Channel{
		Id:       100,
		Name:     "single-channel",
		Status:   common.ChannelStatusEnabled,
		Priority: &singlePriority,
		Weight:   &singleWeight,
	}
	group2model2channels["default"]["gpt-4-single"] = []int{100}
}
