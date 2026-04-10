package hub

import (
	"sync"
	"testing"
)

func TestAdjustWeights_NoHub(t *testing.T) {
	// Reset singleton for test isolation
	instance = nil
	once = sync.Once{}

	result := AdjustWeights([]int{1, 2, 3}, []int{100, 100, 100})
	if result != nil {
		t.Error("expected nil when Hub is not initialized")
	}
}

func TestAdjustWeights_NoScores(t *testing.T) {
	instance = nil
	once = sync.Once{}
	Init(nil)
	defer func() { instance = nil; once = sync.Once{} }()

	// No scores recorded — should return nil
	result := AdjustWeights([]int{1, 2, 3}, []int{100, 100, 100})
	if result != nil {
		t.Error("expected nil when no scores available")
	}
}

func TestAdjustWeights_WithScores(t *testing.T) {
	instance = nil
	once = sync.Once{}
	h := Init(nil)
	defer func() { instance = nil; once = sync.Once{} }()

	// Channel 1: perfect (low latency, no errors)
	for i := 0; i < 20; i++ {
		h.Scorer.RecordSuccess(1, 0.1)
	}
	// Channel 2: mediocre (some errors)
	for i := 0; i < 15; i++ {
		h.Scorer.RecordSuccess(2, 0.5)
	}
	for i := 0; i < 5; i++ {
		h.Scorer.RecordFailure(2)
	}
	// Channel 3: no data (unknown)

	result := AdjustWeights([]int{1, 2, 3}, []int{100, 100, 100})
	if result == nil {
		t.Fatal("expected adjusted weights")
	}

	// Channel 1 should have highest weight (score ~1.0 → factor ~1.5 → weight ~150)
	if result[0] <= result[1] {
		t.Errorf("channel 1 (perfect) should have higher weight than channel 2 (mediocre): %d vs %d", result[0], result[1])
	}

	// Channel 3 should keep original weight (no data)
	if result[2] != 100 {
		t.Errorf("channel 3 (no data) should keep original weight 100, got %d", result[2])
	}
}

func TestAdjustWeights_MinWeightIsOne(t *testing.T) {
	instance = nil
	once = sync.Once{}
	h := Init(nil)
	defer func() { instance = nil; once = sync.Once{} }()

	// Channel with all failures, very low score
	for i := 0; i < 100; i++ {
		h.Scorer.RecordFailure(1)
	}

	result := AdjustWeights([]int{1}, []int{1})
	if result == nil {
		t.Fatal("expected adjusted weights")
	}
	if result[0] < 1 {
		t.Errorf("weight should never be less than 1, got %d", result[0])
	}
}

func TestAdjustWeights_BoostHighPerformers(t *testing.T) {
	instance = nil
	once = sync.Once{}
	h := Init(nil)
	defer func() { instance = nil; once = sync.Once{} }()

	// Fast, reliable channel
	for i := 0; i < 50; i++ {
		h.Scorer.RecordSuccess(1, 0.05)
	}

	result := AdjustWeights([]int{1}, []int{100})
	if result == nil {
		t.Fatal("expected adjusted weights")
	}

	// Score should be near 1.0, factor near 1.5, weight near 150
	if result[0] < 130 {
		t.Errorf("high-performing channel should get significant boost, got %d (expected ~150)", result[0])
	}
}
