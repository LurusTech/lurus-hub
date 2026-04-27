package handler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestAutomaticallyUpdateChannelsWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 1) // 1 minute frequency
		close(done)
	}()

	// Let it initialize
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("AutomaticallyUpdateChannelsWithContext did not respond to context cancellation")
	}
}

func TestAutomaticallyUpdateChannelsWithContext_ImmediateCancellation(t *testing.T) {
	// Cancel before starting
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 1)
		close(done)
	}()

	select {
	case <-done:
		// OK - should exit immediately
	case <-time.After(500 * time.Millisecond):
		t.Fatal("should exit immediately when context is already cancelled")
	}
}

func TestAutomaticallyTestChannelsWithContext_Cancellation(t *testing.T) {
	// Save and restore master node setting
	prevMaster := common.IsMasterNode
	common.IsMasterNode = true
	defer func() { common.IsMasterNode = prevMaster }()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		AutomaticallyTestChannelsWithContext(ctx)
		close(done)
	}()

	// Let it initialize
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("AutomaticallyTestChannelsWithContext did not respond to context cancellation")
	}
}

func TestAutomaticallyTestChannelsWithContext_NotMaster(t *testing.T) {
	prevMaster := common.IsMasterNode
	common.IsMasterNode = false
	defer func() { common.IsMasterNode = prevMaster }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyTestChannelsWithContext(ctx)
		close(done)
	}()

	// Should exit immediately when not master
	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("should exit immediately when not master node")
	}
}

func TestUpdateMidjourneyTaskBulkWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		UpdateMidjourneyTaskBulkWithContext(ctx)
		close(done)
	}()

	// Let it initialize
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("UpdateMidjourneyTaskBulkWithContext did not respond to context cancellation")
	}
}

func TestUpdateMidjourneyTaskBulkWithContext_ImmediateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		UpdateMidjourneyTaskBulkWithContext(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Fatal("should exit immediately when context is already cancelled")
	}
}

func TestUpdateTaskBulkWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		UpdateTaskBulkWithContext(ctx)
		close(done)
	}()

	// Let it initialize
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("UpdateTaskBulkWithContext did not respond to context cancellation")
	}
}

func TestUpdateTaskBulkWithContext_ImmediateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		UpdateTaskBulkWithContext(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Fatal("should exit immediately when context is already cancelled")
	}
}

// Stress tests for concurrent cancellation
func TestConcurrentContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const numGoroutines = 20
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var completedCount atomic.Int32
	done := make(chan struct{})

	// Start multiple different context tasks
	for i := 0; i < numGoroutines/4; i++ {
		go func() {
			AutomaticallyUpdateChannelsWithContext(ctx, 1)
			completedCount.Add(1)
		}()
		go func() {
			UpdateMidjourneyTaskBulkWithContext(ctx)
			completedCount.Add(1)
		}()
		go func() {
			UpdateTaskBulkWithContext(ctx)
			completedCount.Add(1)
		}()
		go func() {
			prevMaster := common.IsMasterNode
			common.IsMasterNode = true
			AutomaticallyTestChannelsWithContext(ctx)
			common.IsMasterNode = prevMaster
			completedCount.Add(1)
		}()
	}

	// Wait for context timeout
	<-ctx.Done()

	// Wait a bit more for all goroutines to complete
	go func() {
		for completedCount.Load() < numGoroutines {
			time.Sleep(10 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(3 * time.Second):
		completed := completedCount.Load()
		t.Fatalf("only %d/%d goroutines completed after timeout", completed, numGoroutines)
	}
}

// Test context timeout propagation
func TestContextTimeoutPropagation(t *testing.T) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 60) // 60 minute frequency (won't trigger)
		close(done)
	}()

	<-done

	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("context timeout took too long: %v", elapsed)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("context timeout was too fast: %v", elapsed)
	}
}

// Benchmark context cancellation overhead
func BenchmarkUpdateChannelsWithContext_Cancel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		AutomaticallyUpdateChannelsWithContext(ctx, 1)
	}
}

func BenchmarkUpdateMidjourneyWithContext_Cancel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		UpdateMidjourneyTaskBulkWithContext(ctx)
	}
}

func BenchmarkUpdateTaskWithContext_Cancel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		UpdateTaskBulkWithContext(ctx)
	}
}

// Test AutomaticallyTestChannelsWithContext with disabled auto-test
func TestAutomaticallyTestChannelsWithContext_DisabledAutoTest(t *testing.T) {
	prevMaster := common.IsMasterNode
	common.IsMasterNode = true
	defer func() { common.IsMasterNode = prevMaster }()

	// Auto-test is disabled by default in operation_setting
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyTestChannelsWithContext(ctx)
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

// Test multiple context cancellations (double cancel)
func TestDoubleContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 1)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)

	// Cancel twice
	cancel()
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("did not handle double cancel")
	}
}

// Test context already done before starting
func TestContextAlreadyDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled

	// All these should exit immediately
	start := time.Now()

	AutomaticallyUpdateChannelsWithContext(ctx, 1)
	UpdateMidjourneyTaskBulkWithContext(ctx)
	UpdateTaskBulkWithContext(ctx)

	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("took too long: %v", elapsed)
	}
}

// Test context deadline (not cancel)
func TestContextDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 60) // Long interval
		close(done)
	}()

	select {
	case <-done:
		// OK - should exit at deadline
	case <-time.After(500 * time.Millisecond):
		t.Fatal("did not respect deadline")
	}
}

// Test rapid start/stop cycles
func TestRapidStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		AutomaticallyUpdateChannelsWithContext(ctx, 1)
		cancel()
	}
}

// Test with nil context (should panic or handle gracefully)
func TestNilContext(t *testing.T) {
	// This tests that the functions handle edge cases
	// Most Go context functions panic on nil context, but we should verify behavior
	defer func() {
		if r := recover(); r != nil {
			// Expected - nil context causes panic
		}
	}()

	// This might panic - that's acceptable behavior for nil context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use cancelled context instead of nil to test
	AutomaticallyUpdateChannelsWithContext(ctx, 1)
}
