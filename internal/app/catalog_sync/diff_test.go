package catalog_sync

import "testing"

func TestDiffModels_AddAndStale(t *testing.T) {
	diff := DiffModels("a,b,c", []UpstreamModel{{ID: "b"}, {ID: "c"}, {ID: "d"}})
	if len(diff.Added) != 1 || diff.Added[0] != "d" {
		t.Fatalf("Added = %#v, want [d]", diff.Added)
	}
	if len(diff.Stale) != 1 || diff.Stale[0] != "a" {
		t.Fatalf("Stale = %#v, want [a]", diff.Stale)
	}
	if len(diff.Kept) != 2 {
		t.Fatalf("Kept = %#v", diff.Kept)
	}
	if diff.Version == "" {
		t.Fatal("expected non-empty version hash")
	}
}

func TestShouldTripBreaker(t *testing.T) {
	if !ShouldTripBreaker(0, 0) {
		t.Fatal("empty fetch with no baseline should trip")
	}
	if ShouldTripBreaker(50, 40) {
		t.Fatal("healthy fetch should not trip")
	}
	if !ShouldTripBreaker(5, 40) {
		t.Fatal("half baseline threshold should trip for 5 vs 40")
	}
}

func TestCircuitBreakerThresholdFloor(t *testing.T) {
	if got := circuitBreakerThreshold(10); got != 10 {
		t.Fatalf("threshold(10)=%d want 10", got)
	}
	if got := circuitBreakerThreshold(100); got != 50 {
		t.Fatalf("threshold(100)=%d want 50", got)
	}
}
