package hub

import (
	"testing"
	"time"
)

func TestChannelScorer_BasicScoring(t *testing.T) {
	cs := NewChannelScorer()

	// Record some successes with low latency
	for i := 0; i < 10; i++ {
		cs.RecordSuccess(1, 0.5)
	}

	score := cs.GetScore(1)
	if score == nil {
		t.Fatal("expected score for channel 1")
	}
	if score.Score < 0.8 {
		t.Errorf("expected high score for healthy channel, got %f", score.Score)
	}
	if score.ErrorRate != 0 {
		t.Errorf("expected 0 error rate, got %f", score.ErrorRate)
	}
}

func TestChannelScorer_ErrorsLowerScore(t *testing.T) {
	cs := NewChannelScorer()

	cs.RecordSuccess(1, 0.5)
	cs.RecordSuccess(1, 0.5)
	cs.RecordFailure(1)
	cs.RecordFailure(1)

	score := cs.GetScore(1)
	if score == nil {
		t.Fatal("expected score")
	}
	if score.ErrorRate != 0.5 {
		t.Errorf("expected 50%% error rate, got %f", score.ErrorRate)
	}
	// Score should be lower than a healthy channel (with 50% errors and
	// cost weight contributing 0.2 baseline, score ≈ 0.3*latency + 0.5*0.5 + 0.2 = 0.7x)
	if score.Score > 0.8 {
		t.Errorf("expected lower score with 50%% errors, got %f", score.Score)
	}
}

func TestChannelScorer_HighLatencyLowersScore(t *testing.T) {
	cs := NewChannelScorer()

	// Fast channel
	for i := 0; i < 10; i++ {
		cs.RecordSuccess(1, 0.1)
	}
	// Slow channel
	for i := 0; i < 10; i++ {
		cs.RecordSuccess(2, 8.0)
	}

	fast := cs.GetScore(1)
	slow := cs.GetScore(2)

	if fast.Score <= slow.Score {
		t.Errorf("fast channel (%f) should score higher than slow (%f)", fast.Score, slow.Score)
	}
}

func TestChannelScorer_GetBestChannel(t *testing.T) {
	cs := NewChannelScorer()

	// Channel 1: perfect
	for i := 0; i < 20; i++ {
		cs.RecordSuccess(1, 0.2)
	}
	// Channel 2: some errors
	for i := 0; i < 15; i++ {
		cs.RecordSuccess(2, 0.3)
	}
	for i := 0; i < 5; i++ {
		cs.RecordFailure(2)
	}
	// Channel 3: very slow
	for i := 0; i < 20; i++ {
		cs.RecordSuccess(3, 9.0)
	}

	best := cs.GetBestChannel([]int{1, 2, 3})
	if best != 1 {
		t.Errorf("expected channel 1 as best, got %d", best)
	}

	// Unknown channel should not be selected
	best = cs.GetBestChannel([]int{999})
	if best != -1 {
		t.Errorf("expected -1 for unknown channels, got %d", best)
	}
}

func TestChannelScorer_UnknownChannel(t *testing.T) {
	cs := NewChannelScorer()

	score := cs.GetScore(999)
	if score != nil {
		t.Error("expected nil for untracked channel")
	}
}

func TestChannelScorer_Decay(t *testing.T) {
	cs := NewChannelScorer()
	cs.decayInterval = 0 // immediate decay for testing

	cs.RecordSuccess(1, 0.5)
	cs.RecordSuccess(1, 0.5)

	// Force time to be old
	cs.mu.Lock()
	cs.stats[1].lastUpdated = time.Now().Add(-time.Hour)
	cs.mu.Unlock()

	cs.Decay()

	// After decay, counts should be halved
	cs.mu.RLock()
	s := cs.stats[1]
	cs.mu.RUnlock()

	if s.successes != 1 {
		t.Errorf("expected successes to be halved to 1, got %d", s.successes)
	}
}

func TestChannelScorer_GetAllScores(t *testing.T) {
	cs := NewChannelScorer()

	cs.RecordSuccess(1, 0.5)
	cs.RecordSuccess(2, 1.0)
	cs.RecordFailure(3)

	scores := cs.GetAllScores()
	if len(scores) != 3 {
		t.Errorf("expected 3 scores, got %d", len(scores))
	}
}

func TestChannelScorer_Concurrent(t *testing.T) {
	cs := NewChannelScorer()
	done := make(chan struct{})

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cs.RecordSuccess(id%3, 0.5)
				cs.RecordFailure(id % 3)
			}
			done <- struct{}{}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cs.GetAllScores()
				cs.GetBestChannel([]int{0, 1, 2})
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 15; i++ {
		<-done
	}
}
