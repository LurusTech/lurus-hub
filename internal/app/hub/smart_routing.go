package hub

// AdjustWeights modifies channel weights based on real-time performance scores.
// Each channel's weight is multiplied by a factor derived from its Hub score:
//
//   - Score 1.0 (perfect) → weight * 1.5 (50% boost)
//   - Score 0.5 (average) → weight * 1.0 (unchanged)
//   - Score 0.0 (terrible) → weight * 0.5 (50% reduction)
//   - Unknown channel → weight * 1.0 (no change, falls back to original behavior)
//
// This ensures that channels with better latency, lower error rates, and lower
// cost are selected more frequently, while still giving all channels a chance
// (no channel is fully excluded by the scorer).
//
// The function returns adjusted weights in the same order as the input channelIDs.
// If Hub is not initialized, returns nil (caller should use original weights).
func AdjustWeights(channelIDs []int, originalWeights []int) []int {
	h := Get()
	if h == nil {
		return nil
	}

	adjusted := make([]int, len(channelIDs))
	anyAdjusted := false

	for i, id := range channelIDs {
		score := h.Scorer.GetScore(id)
		if score == nil {
			// No data for this channel — keep original weight
			adjusted[i] = originalWeights[i]
			continue
		}

		// Linear interpolation: score 0→0.5x, score 0.5→1.0x, score 1.0→1.5x
		factor := 0.5 + score.Score // range: [0.5, 1.5]
		w := int(float64(originalWeights[i]) * factor)
		if w < 1 {
			w = 1 // never reduce to zero — every channel gets a chance
		}
		adjusted[i] = w
		anyAdjusted = true
	}

	if !anyAdjusted {
		return nil // no scores available, let caller use original weights
	}
	return adjusted
}
