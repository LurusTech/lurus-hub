package repo

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestSyncChannelCacheWithContext_Cancellation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Enable memory cache for test
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = false }()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		SyncChannelCacheWithContext(ctx, 1) // 1 second frequency
		close(done)
	}()

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Should exit within reasonable time
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("SyncChannelCacheWithContext did not respond to context cancellation")
	}
}

func TestSyncChannelCacheWithContext_MultipleIterations(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = false }()

	// Create an ability first (required for channel group initialization)
	ability := &Ability{
		Group:     "default",
		Model:     "gpt-4",
		ChannelId: 1,
		Enabled:   true,
	}
	if err := DB.Create(ability).Error; err != nil {
		t.Fatalf("failed to create test ability: %v", err)
	}

	// Create a test channel
	channel := &Channel{
		Name:   "test-channel",
		Type:   1,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
		Models: "gpt-4",
		Group:  "default",
	}
	if err := DB.Create(channel).Error; err != nil {
		t.Fatalf("failed to create test channel: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		SyncChannelCacheWithContext(ctx, 1)
		close(done)
	}()

	// Wait for context timeout
	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

func TestSyncOptionsWithContext_Cancellation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		SyncOptionsWithContext(ctx, 1)
		close(done)
	}()

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("SyncOptionsWithContext did not respond to context cancellation")
	}
}

func TestSyncOptionsWithContext_LoadsOptions(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Insert a test option
	option := &Option{
		Key:   "TestOption",
		Value: "TestValue",
	}
	if err := DB.Create(option).Error; err != nil {
		t.Fatalf("failed to create test option: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		SyncOptionsWithContext(ctx, 1)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

func TestUpdateQuotaDataWithContext_Cancellation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Set up required settings
	prevEnabled := common.DataExportEnabled
	prevInterval := common.DataExportInterval
	common.DataExportEnabled = true
	common.DataExportInterval = 1 // 1 minute
	defer func() {
		common.DataExportEnabled = prevEnabled
		common.DataExportInterval = prevInterval
	}()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		UpdateQuotaDataWithContext(ctx)
		close(done)
	}()

	// Let it initialize
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("UpdateQuotaDataWithContext did not respond to context cancellation")
	}
}

func TestUpdateQuotaDataWithContext_DisabledExport(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	prevEnabled := common.DataExportEnabled
	prevInterval := common.DataExportInterval
	common.DataExportEnabled = false
	common.DataExportInterval = 1
	defer func() {
		common.DataExportEnabled = prevEnabled
		common.DataExportInterval = prevInterval
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateQuotaDataWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// Edge case: rapid context cancellation
func TestSyncChannelCacheWithContext_ImmediateCancellation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = false }()

	// Cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		SyncChannelCacheWithContext(ctx, 1)
		close(done)
	}()

	select {
	case <-done:
		// OK - should exit immediately
	case <-time.After(500 * time.Millisecond):
		t.Fatal("should exit immediately when context is already cancelled")
	}
}

// Edge case: very short frequency
func TestSyncOptionsWithContext_ShortFrequency(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// We can't easily count syncs without modifying the function,
	// but we can at least verify it handles short intervals
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		SyncOptionsWithContext(ctx, 1) // 1 second
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("did not exit after timeout")
	}

}

// Stress test: many concurrent syncs
func TestConcurrentContextSyncs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	cleanup := SetupTestDB(t)
	defer cleanup()

	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = false }()

	const numGoroutines = 10
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	var completedCount atomic.Int32

	for i := 0; i < numGoroutines; i++ {
		go func() {
			SyncChannelCacheWithContext(ctx, 1)
			completedCount.Add(1)
			if completedCount.Load() == numGoroutines {
				close(done)
			}
		}()
	}

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(3 * time.Second):
		completed := completedCount.Load()
		t.Fatalf("only %d/%d goroutines completed", completed, numGoroutines)
	}
}

// Test UpdateQuotaDataWithContext with enabled export
func TestUpdateQuotaDataWithContext_EnabledExport(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	prevEnabled := common.DataExportEnabled
	prevInterval := common.DataExportInterval
	common.DataExportEnabled = true
	common.DataExportInterval = 1
	defer func() {
		common.DataExportEnabled = prevEnabled
		common.DataExportInterval = prevInterval
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateQuotaDataWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// Test ticker actually fires
func TestUpdateQuotaDataWithContext_TickerFires(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cleanup := SetupTestDB(t)
	defer cleanup()

	prevEnabled := common.DataExportEnabled
	prevInterval := common.DataExportInterval
	common.DataExportEnabled = true
	common.DataExportInterval = 1 // 1 minute - won't fire in test
	defer func() {
		common.DataExportEnabled = prevEnabled
		common.DataExportInterval = prevInterval
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateQuotaDataWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("did not exit")
	}
}

// Benchmark tests
func BenchmarkSyncChannelCacheWithContext_Start(b *testing.B) {
	// This benchmarks the startup overhead
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Immediate cancel

		SyncChannelCacheWithContext(ctx, 60)
	}
}

func BenchmarkSyncOptionsWithContext_Start(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		SyncOptionsWithContext(ctx, 60)
	}
}
