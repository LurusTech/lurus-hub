// Package hub implements data processing capabilities that differentiate
// Lurus Hub from a plain LLM relay. These features are additive — they
// do not modify upstream New API relay logic, keeping upstream sync clean.
package hub

import (
	"math"
	"sync"
	"time"
)

// ChannelScore represents the computed quality score for a channel.
// Higher score = better channel for routing decisions.
type ChannelScore struct {
	ChannelID   int
	Score       float64   // 0.0 - 1.0, higher is better
	Latency     float64   // exponential moving average (seconds)
	ErrorRate   float64   // error rate over the sliding window (0.0 - 1.0)
	SuccessRate float64   // 1 - ErrorRate
	CostFactor  float64   // relative cost (1.0 = baseline)
	UpdatedAt   time.Time // last score computation
}

// channelStats tracks raw request outcomes for scoring.
type channelStats struct {
	successes   int64
	failures    int64
	latencySum  float64
	latencyEMA  float64 // exponential moving average
	lastUpdated time.Time
}

// ChannelScorer computes real-time performance scores for channels.
// It uses a sliding window of observations to compute latency EMA,
// error rate, and an overall quality score for smart routing.
//
// Thread-safe: all methods are safe for concurrent use.
type ChannelScorer struct {
	mu    sync.RWMutex
	stats map[int]*channelStats

	// Scoring weights (configurable)
	latencyWeight float64
	errorWeight   float64
	costWeight    float64

	// EMA smoothing factor (0 < alpha <= 1). Higher = more responsive.
	emaAlpha float64

	// Window: stats older than this are decayed.
	decayInterval time.Duration
}

// NewChannelScorer creates a scorer with production-tuned defaults.
func NewChannelScorer() *ChannelScorer {
	return &ChannelScorer{
		stats:         make(map[int]*channelStats),
		latencyWeight: 0.3,
		errorWeight:   0.5,
		costWeight:    0.2,
		emaAlpha:      0.3,
		decayInterval: 10 * time.Minute,
	}
}

// RecordSuccess records a successful request for a channel.
func (cs *ChannelScorer) RecordSuccess(channelID int, latencySec float64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	s := cs.getOrCreate(channelID)
	s.successes++
	s.updateLatencyEMA(latencySec, cs.emaAlpha)
	s.lastUpdated = time.Now()
}

// RecordFailure records a failed request for a channel.
func (cs *ChannelScorer) RecordFailure(channelID int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	s := cs.getOrCreate(channelID)
	s.failures++
	s.lastUpdated = time.Now()
}

// GetScore computes and returns the current score for a channel.
// Returns nil if the channel has no observations.
func (cs *ChannelScorer) GetScore(channelID int) *ChannelScore {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	s, ok := cs.stats[channelID]
	if !ok {
		return nil
	}

	return cs.computeScore(channelID, s)
}

// GetAllScores returns scores for all tracked channels.
func (cs *ChannelScorer) GetAllScores() []ChannelScore {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	scores := make([]ChannelScore, 0, len(cs.stats))
	for id, s := range cs.stats {
		scores = append(scores, *cs.computeScore(id, s))
	}
	return scores
}

// GetBestChannel returns the channel ID with the highest score among
// the given candidates. Returns -1 if no candidates have scores.
func (cs *ChannelScorer) GetBestChannel(candidateIDs []int) int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	bestID := -1
	bestScore := -1.0

	for _, id := range candidateIDs {
		s, ok := cs.stats[id]
		if !ok {
			continue
		}
		score := cs.computeScore(id, s)
		if score.Score > bestScore {
			bestScore = score.Score
			bestID = id
		}
	}
	return bestID
}

// Decay reduces the weight of old observations. Call periodically
// (e.g., every minute) to keep scores responsive to recent behavior.
func (cs *ChannelScorer) Decay() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cutoff := time.Now().Add(-cs.decayInterval)
	for id, s := range cs.stats {
		if s.lastUpdated.Before(cutoff) {
			// Halve old counts to decay influence gradually
			s.successes = s.successes / 2
			s.failures = s.failures / 2
			if s.successes == 0 && s.failures == 0 {
				delete(cs.stats, id)
			}
		}
	}
}

func (cs *ChannelScorer) getOrCreate(channelID int) *channelStats {
	s, ok := cs.stats[channelID]
	if !ok {
		s = &channelStats{lastUpdated: time.Now()}
		cs.stats[channelID] = s
	}
	return s
}

func (cs *ChannelScorer) computeScore(channelID int, s *channelStats) *ChannelScore {
	total := s.successes + s.failures
	if total == 0 {
		return &ChannelScore{
			ChannelID:   channelID,
			Score:       0.5, // neutral score for unknown channels
			Latency:     0,
			ErrorRate:   0,
			SuccessRate: 1,
			CostFactor:  1,
			UpdatedAt:   s.lastUpdated,
		}
	}

	errorRate := float64(s.failures) / float64(total)
	successRate := 1 - errorRate

	// Normalize latency: 0-1s → 1.0, 1-5s → 0.8-0.4, 5s+ → <0.4
	latencyScore := 1.0
	if s.latencyEMA > 0 {
		latencyScore = math.Max(0, 1.0-s.latencyEMA/10.0)
	}

	// Weighted composite score
	score := cs.latencyWeight*latencyScore +
		cs.errorWeight*successRate +
		cs.costWeight*1.0 // cost factor is 1.0 for now (TODO: integrate pricing)

	// Clamp to [0, 1]
	score = math.Max(0, math.Min(1, score))

	return &ChannelScore{
		ChannelID:   channelID,
		Score:       score,
		Latency:     s.latencyEMA,
		ErrorRate:   errorRate,
		SuccessRate: successRate,
		CostFactor:  1.0,
		UpdatedAt:   s.lastUpdated,
	}
}

func (s *channelStats) updateLatencyEMA(latency, alpha float64) {
	if s.latencyEMA == 0 {
		s.latencyEMA = latency
	} else {
		s.latencyEMA = alpha*latency + (1-alpha)*s.latencyEMA
	}
	s.latencySum += latency
}
