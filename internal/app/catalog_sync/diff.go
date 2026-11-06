package catalog_sync

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func splitModels(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		m := strings.TrimSpace(p)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func joinModels(models []string) string {
	return strings.Join(models, ",")
}

func setOf(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// DiffResult compares local channel model list vs upstream IDs.
type DiffResult struct {
	Added   []string
	Stale   []string
	Kept    []string
	Version string // stable hash of sorted upstream IDs
}

func DiffModels(localCSV string, upstream []UpstreamModel) DiffResult {
	local := setOf(splitModels(localCSV))
	up := map[string]bool{}
	for _, u := range upstream {
		id := strings.TrimSpace(u.ID)
		if id == "" {
			continue
		}
		up[id] = true
	}
	var added, stale, kept []string
	for id := range up {
		if local[id] {
			kept = append(kept, id)
		} else {
			added = append(added, id)
		}
	}
	for id := range local {
		if !up[id] {
			stale = append(stale, id)
		}
	}
	sort.Strings(added)
	sort.Strings(stale)
	sort.Strings(kept)
	return DiffResult{
		Added:   added,
		Stale:   stale,
		Kept:    kept,
		Version: hashIDs(sortedKeys(up)),
	}
}

func hashIDs(ids []string) string {
	h := sha256.New()
	for _, id := range ids {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func circuitBreakerThreshold(baseline int) int {
	half := baseline / 2
	if half > 10 {
		return half
	}
	return 10
}

// ShouldTripBreaker returns true when fetch looks suspiciously small vs baseline.
func ShouldTripBreaker(fetched, baseline int) bool {
	if baseline <= 0 {
		return fetched == 0
	}
	return fetched < circuitBreakerThreshold(baseline)
}
