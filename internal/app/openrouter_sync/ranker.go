package openrouter_sync

import (
	"sort"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/provider/openrouter"
)

// stalenessThreshold defines when pre-aggregated stats are considered too old
// to use as a ranking signal. The hourly aggregator targets <1h freshness;
// 24h means it has missed many runs (or never produced data) — fall back.
const stalenessThreshold = 24 * time.Hour

// UsageMap maps model_name → 24-hour call count.
// Empty map ⇒ no usage data; ranker falls back to model.Created.
type UsageMap map[string]int64

// Stat is a small struct mirroring repo.ModelUsageStat to avoid an import cycle.
// The aggregator constructs []Stat and the ranker consumes it.
type Stat struct {
	ModelName     string
	Count24h      int64
	LastUpdatedAt time.Time
}

// BuildUsageMap turns a slice of stats into a UsageMap, dropping rows that are
// too stale to trust. Returns nil if no usable data remained.
func BuildUsageMap(stats []Stat, now time.Time) UsageMap {
	if len(stats) == 0 {
		return nil
	}
	m := make(UsageMap, len(stats))
	for _, s := range stats {
		if now.Sub(s.LastUpdatedAt) > stalenessThreshold {
			continue
		}
		m[s.ModelName] = s.Count24h
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// RankAndTrim sorts the candidate models by descending usage count, breaking ties
// (and handling absence-of-usage entirely) by descending `created` timestamp.
// If topN > 0, the result is truncated to topN entries; topN <= 0 means no limit.
//
// usage may be nil — in that case the function ranks purely by Created (cold-start
// fallback when the aggregator has produced nothing yet).
func RankAndTrim(candidates []openrouter.Model, usage UsageMap, topN int) []openrouter.Model {
	if len(candidates) == 0 {
		return candidates
	}
	out := make([]openrouter.Model, len(candidates))
	copy(out, candidates)

	sort.SliceStable(out, func(i, j int) bool {
		ui := usage[out[i].ID]
		uj := usage[out[j].ID]
		if ui != uj {
			return ui > uj
		}
		// Tie-break on created (newer first); empty Created falls to the bottom.
		if out[i].Created != out[j].Created {
			return out[i].Created > out[j].Created
		}
		// Final stable tiebreak on ID for determinism (so test results don't flap).
		return out[i].ID < out[j].ID
	})

	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}
	return out
}
