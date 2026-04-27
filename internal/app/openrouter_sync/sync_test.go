package openrouter_sync

import (
	"sort"
	"testing"
)

// These tests cover the pure logic that doesn't touch DB / HTTP:
//   - circuit breaker threshold math
//   - set helpers (used to compute manual-preserved + diff)
//
// Higher-level tests (transaction + concurrent jobs + circuit breaker integration)
// will be added once the proto replace is restored and the test DB harness can
// boot.

func TestCircuitBreakerThreshold(t *testing.T) {
	tests := []struct {
		baseline int
		want     int
	}{
		{baseline: 0, want: 10},   // first run — floor of 10
		{baseline: 5, want: 10},   // tiny baseline — still floor of 10
		{baseline: 19, want: 10},  // half is 9 → floor wins
		{baseline: 20, want: 10},  // half is 10 → tie, half not strictly greater, floor wins
		{baseline: 100, want: 50}, // half wins
		{baseline: 999, want: 499},
	}
	for _, tc := range tests {
		if got := circuitBreakerThreshold(tc.baseline); got != tc.want {
			t.Errorf("threshold(%d) = %d, want %d", tc.baseline, got, tc.want)
		}
	}
}

func TestSetHelpers(t *testing.T) {
	a := setFromSlice([]string{"x", "y", "z"})
	b := setFromSlice([]string{"y"})

	diff := setDifference(a, b)
	if len(diff) != 2 || hasInSet(diff, "y") {
		t.Errorf("setDifference broken: %v", diff)
	}

	uni := setUnion(a, b)
	if len(uni) != 3 {
		t.Errorf("setUnion broken: %v", uni)
	}

	sorted := setToSortedSlice(a)
	if !sort.StringsAreSorted(sorted) || len(sorted) != 3 {
		t.Errorf("setToSortedSlice broken: %v", sorted)
	}

	// CSV input with whitespace and empty entries should be cleaned.
	c := setFromSlice([]string{"  m  ", "", "n"})
	if len(c) != 2 || !hasInSet(c, "m") || !hasInSet(c, "n") {
		t.Errorf("setFromSlice trimming broken: %v", c)
	}
}

func TestSortedDiff(t *testing.T) {
	a := setFromSlice([]string{"a", "b", "c"})
	b := setFromSlice([]string{"b"})
	added := sortedDiff(a, b)
	want := []string{"a", "c"}
	if len(added) != len(want) {
		t.Fatalf("sortedDiff = %v, want %v", added, want)
	}
	for i, v := range want {
		if added[i] != v {
			t.Fatalf("sortedDiff = %v, want %v", added, want)
		}
	}
}

func hasInSet(s map[string]struct{}, k string) bool {
	_, ok := s[k]
	return ok
}
