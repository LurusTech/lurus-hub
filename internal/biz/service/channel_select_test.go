package service

import (
	"testing"
)

func TestRetryParam_GetRetry_NilRetry_ReturnsZero(t *testing.T) {
	p := &RetryParam{}
	if got := p.GetRetry(); got != 0 {
		t.Errorf("GetRetry() = %d, want 0", got)
	}
}

func TestRetryParam_GetRetry_ReturnsValue(t *testing.T) {
	val := 5
	p := &RetryParam{Retry: &val}
	if got := p.GetRetry(); got != 5 {
		t.Errorf("GetRetry() = %d, want 5", got)
	}
}

func TestRetryParam_SetRetry(t *testing.T) {
	p := &RetryParam{}
	p.SetRetry(3)
	if p.Retry == nil {
		t.Fatal("expected Retry to be non-nil after SetRetry")
	}
	if *p.Retry != 3 {
		t.Errorf("Retry = %d, want 3", *p.Retry)
	}
}

func TestRetryParam_SetRetry_OverwritesPrevious(t *testing.T) {
	val := 1
	p := &RetryParam{Retry: &val}
	p.SetRetry(10)
	if *p.Retry != 10 {
		t.Errorf("Retry = %d, want 10", *p.Retry)
	}
}

func TestRetryParam_IncreaseRetry_FromNil(t *testing.T) {
	p := &RetryParam{}
	p.IncreaseRetry()
	if p.Retry == nil {
		t.Fatal("expected Retry to be non-nil after IncreaseRetry")
	}
	if *p.Retry != 1 {
		t.Errorf("Retry = %d, want 1", *p.Retry)
	}
}

func TestRetryParam_IncreaseRetry_Increments(t *testing.T) {
	val := 3
	p := &RetryParam{Retry: &val}
	p.IncreaseRetry()
	if *p.Retry != 4 {
		t.Errorf("Retry = %d, want 4", *p.Retry)
	}
}

func TestRetryParam_IncreaseRetry_MultipleTimes(t *testing.T) {
	p := &RetryParam{}
	p.IncreaseRetry()
	p.IncreaseRetry()
	p.IncreaseRetry()
	if *p.Retry != 3 {
		t.Errorf("Retry = %d, want 3 after 3 increases", *p.Retry)
	}
}

func TestRetryParam_ResetRetryNextTry_SkipsOneIncrease(t *testing.T) {
	val := 2
	p := &RetryParam{Retry: &val}

	// Flag that the next IncreaseRetry should be skipped
	p.ResetRetryNextTry()

	// This IncreaseRetry should be a no-op
	p.IncreaseRetry()
	if *p.Retry != 2 {
		t.Errorf("Retry = %d, want 2 (should not have increased)", *p.Retry)
	}

	// Next IncreaseRetry should work normally
	p.IncreaseRetry()
	if *p.Retry != 3 {
		t.Errorf("Retry = %d, want 3 (should have increased normally)", *p.Retry)
	}
}

func TestRetryParam_ResetRetryNextTry_OnlySkipsOnce(t *testing.T) {
	val := 0
	p := &RetryParam{Retry: &val}

	p.ResetRetryNextTry()
	p.IncreaseRetry() // skipped
	p.IncreaseRetry() // should work
	p.IncreaseRetry() // should work

	if *p.Retry != 2 {
		t.Errorf("Retry = %d, want 2 (one skip + two increases)", *p.Retry)
	}
}
