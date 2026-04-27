package openrouter_sync

import (
	"testing"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/provider/openrouter"
)

func TestBuildUsageMap_StaleDropped(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	stats := []Stat{
		{ModelName: "fresh", Count24h: 100, LastUpdatedAt: now.Add(-2 * time.Hour)},
		{ModelName: "stale", Count24h: 9999, LastUpdatedAt: now.Add(-48 * time.Hour)},
	}
	m := BuildUsageMap(stats, now)
	if _, ok := m["fresh"]; !ok {
		t.Errorf("fresh row should be kept")
	}
	if _, ok := m["stale"]; ok {
		t.Errorf("stale row should be dropped")
	}
}

func TestBuildUsageMap_AllStaleReturnsNil(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	stats := []Stat{
		{ModelName: "a", Count24h: 1, LastUpdatedAt: now.Add(-50 * time.Hour)},
	}
	if got := BuildUsageMap(stats, now); got != nil {
		t.Errorf("expected nil when all rows stale, got %v", got)
	}
}

func TestRankAndTrim_ByUsage(t *testing.T) {
	candidates := []openrouter.Model{
		{ID: "low", Created: 100},
		{ID: "high", Created: 50},
		{ID: "mid", Created: 75},
	}
	usage := UsageMap{"low": 1, "mid": 5, "high": 100}
	got := RankAndTrim(candidates, usage, 2)
	if len(got) != 2 || got[0].ID != "high" || got[1].ID != "mid" {
		t.Fatalf("expected [high mid], got %v", got)
	}
}

func TestRankAndTrim_ColdStartByCreated(t *testing.T) {
	candidates := []openrouter.Model{
		{ID: "old", Created: 100},
		{ID: "new", Created: 200},
		{ID: "ancient", Created: 50},
	}
	got := RankAndTrim(candidates, nil, 2)
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("expected [new old] (newer Created wins), got %v", got)
	}
}

func TestRankAndTrim_ZeroTopNNoLimit(t *testing.T) {
	candidates := []openrouter.Model{
		{ID: "a", Created: 1},
		{ID: "b", Created: 2},
		{ID: "c", Created: 3},
	}
	got := RankAndTrim(candidates, nil, 0)
	if len(got) != 3 {
		t.Fatalf("topN=0 should keep all, got %d", len(got))
	}
}

func TestRankAndTrim_TieBreakStable(t *testing.T) {
	// Same usage, same Created → must tiebreak on ID for determinism
	candidates := []openrouter.Model{
		{ID: "z", Created: 100},
		{ID: "a", Created: 100},
		{ID: "m", Created: 100},
	}
	got := RankAndTrim(candidates, UsageMap{"a": 1, "m": 1, "z": 1}, 0)
	if got[0].ID != "a" || got[1].ID != "m" || got[2].ID != "z" {
		t.Fatalf("expected lex order on tie, got %v", got)
	}
}
