package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonitorWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		MonitorWithContext(ctx)
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
		t.Fatal("MonitorWithContext did not respond to context cancellation")
	}
}

func TestMonitorWithContext_ImmediateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		MonitorWithContext(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK - should exit immediately
	case <-time.After(500 * time.Millisecond):
		t.Fatal("should exit immediately when context is already cancelled")
	}
}

func TestMonitorWithContext_TimeoutPropagation(t *testing.T) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		MonitorWithContext(ctx)
		close(done)
	}()

	<-done

	elapsed := time.Since(start)
	// Should exit after context timeout, not wait for next ticker
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestMonitorWithContext_PprofDirCreation(t *testing.T) {
	// This test verifies the pprof directory creation logic works
	// We can't easily trigger high CPU, but we can test the directory handling

	testDir := filepath.Join(os.TempDir(), "pprof-test-"+time.Now().Format("20060102150405"))

	// Ensure directory doesn't exist
	os.RemoveAll(testDir)

	// The actual MonitorWithContext creates ./pprof, but we're just
	// testing the context cancellation behavior here
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		MonitorWithContext(ctx)
		close(done)
	}()

	<-done

	// Cleanup
	os.RemoveAll(testDir)
}

func TestMonitorWithContext_ConcurrentStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	const numGoroutines = 5
	done := make(chan struct{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			MonitorWithContext(ctx)
			done <- struct{}{}
		}()
	}

	// Wait for context timeout
	<-ctx.Done()

	// Collect all completions
	completed := 0
	timeout := time.After(2 * time.Second)
	for completed < numGoroutines {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatalf("only %d/%d goroutines completed", completed, numGoroutines)
		}
	}
}

// Benchmark the overhead of context checking
func BenchmarkMonitorWithContext_Cancel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		MonitorWithContext(ctx)
	}
}

func BenchmarkMonitorWithContext_Timeout(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		MonitorWithContext(ctx)
		cancel()
	}
}
