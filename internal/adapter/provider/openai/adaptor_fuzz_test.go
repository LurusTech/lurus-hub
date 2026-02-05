package openai

import (
	"strings"
	"testing"
)

// FuzzParseReasoningEffortFromModelSuffix tests the model suffix parser with random inputs
// Run: go test -fuzz=FuzzParseReasoningEffortFromModelSuffix -fuzztime=30s ./internal/adapter/provider/openai/...
func FuzzParseReasoningEffortFromModelSuffix(f *testing.F) {
	// Seed corpus with known valid inputs
	seeds := []string{
		"o3-mini-high",
		"o3-mini-low",
		"o3-mini-medium",
		"o4-mini-minimal",
		"o1-none",
		"o3-xhigh",
		"gpt-5-high",
		"o3-mini",
		"gpt-4o",
		"claude-3-opus",
		"",
		"-high",
		"--high",
		"-",
		"model-with-many-dashes-high",
		"model\x00with\x00nulls-high",
		"model\nwith\nnewlines-low",
		strings.Repeat("a", 1000) + "-high",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, model string) {
		effort, parsedModel := parseReasoningEffortFromModelSuffix(model)

		// Invariant 1: effort must be one of the known values or empty
		validEfforts := map[string]bool{
			"":        true,
			"high":    true,
			"low":     true,
			"medium":  true,
			"minimal": true,
			"none":    true,
			"xhigh":   true,
		}
		if !validEfforts[effort] {
			t.Errorf("invalid effort %q for model %q", effort, model)
		}

		// Invariant 2: parsedModel + "-" + effort should equal original model (if effort found)
		// Note: parsedModel CAN still end with the suffix (e.g., "-high-high" -> parsedModel="-high")
		// This is correct behavior - we only strip one suffix occurrence

		// Invariant 3: parsedModel + suffix should reconstruct original (if effort found)
		if effort != "" {
			reconstructed := parsedModel + "-" + effort
			if reconstructed != model {
				t.Errorf("reconstruction failed: %q + %q != %q", parsedModel, effort, model)
			}
		}

		// Invariant 4: if no effort found, parsedModel should equal original
		if effort == "" && parsedModel != model {
			t.Errorf("no effort but parsedModel %q != model %q", parsedModel, model)
		}

		// Invariant 5: should never panic (implicit - if we reach here, no panic)
	})
}
