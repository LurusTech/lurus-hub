package governance

import (
	"strings"
	"testing"
)

func TestComputeFingerprint_Deterministic(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	fp1 := ComputeFingerprint(42, "gpt-4o", body)
	fp2 := ComputeFingerprint(42, "gpt-4o", body)
	if fp1 != fp2 {
		t.Errorf("fingerprint not deterministic: %q vs %q", fp1, fp2)
	}
}

func TestComputeFingerprint_Length(t *testing.T) {
	body := []byte(`test`)
	fp := ComputeFingerprint(1, "model", body)
	if len(fp) != 16 {
		t.Errorf("expected 16 hex chars, got %d: %q", len(fp), fp)
	}
	// Verify all characters are valid hex
	for _, c := range fp {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("non-hex character %c in fingerprint %q", c, fp)
		}
	}
}

func TestComputeFingerprint_DifferentTokenID(t *testing.T) {
	body := []byte(`same body`)
	fp1 := ComputeFingerprint(1, "model", body)
	fp2 := ComputeFingerprint(2, "model", body)
	if fp1 == fp2 {
		t.Error("different tokenIDs should produce different fingerprints")
	}
}

func TestComputeFingerprint_DifferentModel(t *testing.T) {
	body := []byte(`same body`)
	fp1 := ComputeFingerprint(1, "gpt-4o", body)
	fp2 := ComputeFingerprint(1, "gpt-4o-mini", body)
	if fp1 == fp2 {
		t.Error("different models should produce different fingerprints")
	}
}

func TestComputeFingerprint_Truncation(t *testing.T) {
	// Body larger than 4096 bytes — should truncate.
	smallBody := make([]byte, 4096)
	for i := range smallBody {
		smallBody[i] = 'A'
	}
	largeBody := make([]byte, 8192)
	copy(largeBody, smallBody)
	for i := 4096; i < 8192; i++ {
		largeBody[i] = 'B'
	}
	fp1 := ComputeFingerprint(1, "m", smallBody)
	fp2 := ComputeFingerprint(1, "m", largeBody)
	if fp1 != fp2 {
		t.Errorf("large body should be truncated to match small body fingerprint: %q vs %q", fp1, fp2)
	}
}

func TestComputeFingerprint_EmptyBody(t *testing.T) {
	fp := ComputeFingerprint(1, "model", nil)
	if len(fp) != 16 {
		t.Errorf("expected 16 hex chars for empty body, got %d: %q", len(fp), fp)
	}
}

func TestComputeFingerprint_BoundaryConfusion(t *testing.T) {
	// Ensure "1|ab|..." != "1|a|b..." via length-prefixed fields.
	body := []byte(`x`)
	fp1 := ComputeFingerprint(1, "ab", body)
	fp2 := ComputeFingerprint(1, "a", append([]byte("b|"), body...))
	if fp1 == fp2 {
		t.Error("length-prefixed fields should prevent boundary confusion")
	}
}
