package resilience

import (
	"testing"
	"time"
)

func TestBreakerClosedAllowsRequests(t *testing.T) {
	r := NewRegistry(Config{Threshold: 3, Timeout: 50 * time.Millisecond})
	if !r.Allow(1) {
		t.Fatal("expected closed breaker to allow request")
	}
}

func TestBreakerTripsAfterThreshold(t *testing.T) {
	transitions := make([]struct{ from, to State }, 0)
	r := NewRegistry(Config{
		Threshold: 3,
		Timeout:   50 * time.Millisecond,
		OnStateChange: func(channelID int, from, to State) {
			transitions = append(transitions, struct{ from, to State }{from, to})
		},
	})

	// 2 failures: still closed.
	r.RecordFailure(1)
	r.RecordFailure(1)
	if !r.Allow(1) {
		t.Fatal("expected breaker to still allow after 2 failures (threshold=3)")
	}

	// 3rd failure: trips to Open.
	r.RecordFailure(1)
	if r.Allow(1) {
		t.Fatal("expected breaker to reject after 3 failures")
	}
	if r.GetState(1) != StateOpen {
		t.Fatalf("expected Open, got %s", r.GetState(1))
	}
	if len(transitions) != 1 || transitions[0].to != StateOpen {
		t.Fatalf("expected one transition to Open, got %+v", transitions)
	}
}

func TestBreakerHalfOpenOnTimeout(t *testing.T) {
	r := NewRegistry(Config{Threshold: 1, Timeout: 20 * time.Millisecond})

	r.RecordFailure(1)
	if r.Allow(1) {
		t.Fatal("expected Open breaker to reject")
	}

	// Wait for timeout.
	time.Sleep(30 * time.Millisecond)
	if !r.Allow(1) {
		t.Fatal("expected breaker to allow one probe after timeout (HalfOpen)")
	}
	if r.GetState(1) != StateHalfOpen {
		t.Fatalf("expected HalfOpen, got %s", r.GetState(1))
	}

	// Second request while HalfOpen should be rejected.
	if r.Allow(1) {
		t.Fatal("expected HalfOpen breaker to reject concurrent requests")
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	r := NewRegistry(Config{Threshold: 1, Timeout: 10 * time.Millisecond})

	r.RecordFailure(1)
	time.Sleep(15 * time.Millisecond)
	r.Allow(1) // Transition to HalfOpen.
	r.RecordSuccess(1)

	if r.GetState(1) != StateClosed {
		t.Fatalf("expected Closed after success, got %s", r.GetState(1))
	}
	if !r.Allow(1) {
		t.Fatal("expected closed breaker to allow request after recovery")
	}
}

func TestBreakerHalfOpenFailureReturnsToOpen(t *testing.T) {
	r := NewRegistry(Config{Threshold: 1, Timeout: 10 * time.Millisecond})

	r.RecordFailure(1) // Open
	time.Sleep(15 * time.Millisecond)
	r.Allow(1) // HalfOpen

	r.RecordFailure(1) // Probe failed → back to Open
	if r.GetState(1) != StateOpen {
		t.Fatalf("expected Open after HalfOpen failure, got %s", r.GetState(1))
	}
}

func TestBreakerPerChannelIsolation(t *testing.T) {
	r := NewRegistry(Config{Threshold: 1, Timeout: time.Minute})

	r.RecordFailure(1) // Channel 1 → Open
	if r.Allow(1) {
		t.Fatal("expected channel 1 to be open")
	}
	if !r.Allow(2) {
		t.Fatal("expected channel 2 to still be closed (independent)")
	}
}

func TestBreakerSuccessResetsConsecutiveCount(t *testing.T) {
	r := NewRegistry(Config{Threshold: 3, Timeout: time.Minute})

	r.RecordFailure(1)
	r.RecordFailure(1)
	r.RecordSuccess(1) // Reset count
	r.RecordFailure(1) // 1st failure after reset
	r.RecordFailure(1) // 2nd

	if !r.Allow(1) {
		t.Fatal("expected breaker to still allow — only 2 consecutive failures after reset")
	}
}
