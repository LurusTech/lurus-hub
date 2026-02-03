package common

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSafeGo_NormalExecution(t *testing.T) {
	var executed atomic.Bool

	SafeGo(func() {
		executed.Store(true)
	})

	// Wait for goroutine to complete
	time.Sleep(50 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGo did not execute the function")
	}
}

func TestSafeGo_PanicRecovery(t *testing.T) {
	var completed atomic.Bool

	// This should not crash the test
	SafeGo(func() {
		defer func() { completed.Store(true) }()
		panic("test panic")
	})

	// Wait for goroutine to complete
	time.Sleep(50 * time.Millisecond)

	if !completed.Load() {
		t.Error("SafeGo did not recover from panic")
	}
}

func TestSafeGoWithContext_NormalExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var executed atomic.Bool

	SafeGoWithContext(ctx, func(c context.Context) {
		executed.Store(true)
	})

	time.Sleep(50 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGoWithContext did not execute the function")
	}
}

func TestSafeGoWithContext_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var started atomic.Bool
	var stopped atomic.Bool

	SafeGoWithContext(ctx, func(c context.Context) {
		started.Store(true)
		<-c.Done()
		stopped.Store(true)
	})

	time.Sleep(50 * time.Millisecond)
	if !started.Load() {
		t.Error("function did not start")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if !stopped.Load() {
		t.Error("function did not respond to context cancellation")
	}
}

func TestSafeGoWithContext_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	var completed atomic.Bool

	SafeGoWithContext(ctx, func(c context.Context) {
		defer func() { completed.Store(true) }()
		panic("test panic")
	})

	time.Sleep(50 * time.Millisecond)

	if !completed.Load() {
		t.Error("SafeGoWithContext did not recover from panic")
	}
}

func TestSafeGoNamed_NormalExecution(t *testing.T) {
	var executed atomic.Bool

	SafeGoNamed("test-task", func() {
		executed.Store(true)
	})

	time.Sleep(50 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGoNamed did not execute the function")
	}
}

func TestSafeGoNamed_PanicRecovery(t *testing.T) {
	var completed atomic.Bool

	SafeGoNamed("panic-task", func() {
		defer func() { completed.Store(true) }()
		panic("test panic")
	})

	time.Sleep(50 * time.Millisecond)

	if !completed.Load() {
		t.Error("SafeGoNamed did not recover from panic")
	}
}

func TestMustGo_NormalExecution(t *testing.T) {
	var executed atomic.Bool

	MustGo(func() {
		executed.Store(true)
	}, 3)

	time.Sleep(50 * time.Millisecond)

	if !executed.Load() {
		t.Error("MustGo did not execute the function")
	}
}

func TestMustGo_PanicRecovery(t *testing.T) {
	var completed atomic.Bool

	MustGo(func() {
		defer func() { completed.Store(true) }()
		panic("test panic")
	}, 3)

	time.Sleep(100 * time.Millisecond)

	if !completed.Load() {
		t.Error("MustGo did not complete after panic")
	}
}

// Concurrent stress test
func TestSafeGo_ConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const numGoroutines = 100
	var completed atomic.Int32

	for i := 0; i < numGoroutines; i++ {
		SafeGo(func() {
			time.Sleep(10 * time.Millisecond)
			completed.Add(1)
		})
	}

	// Wait for all goroutines to complete
	time.Sleep(200 * time.Millisecond)

	if completed.Load() != numGoroutines {
		t.Errorf("expected %d completions, got %d", numGoroutines, completed.Load())
	}
}

// Benchmark tests
func BenchmarkSafeGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		SafeGo(func() {
			close(done)
		})
		<-done
	}
}

func BenchmarkSafeGoWithContext(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		SafeGoWithContext(ctx, func(c context.Context) {
			close(done)
		})
		<-done
	}
}

func BenchmarkRawGoroutine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		go func() {
			close(done)
		}()
		<-done
	}
}
