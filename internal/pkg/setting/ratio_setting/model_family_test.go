package ratio_setting

import (
	"testing"
)

func TestModelFamilyFallback(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantRatio float64
		wantMatch bool
	}{
		// Gemini families
		{"gemini flash-lite", "gemini-3.1-flash-lite-preview", 0.05, true},
		{"gemini flash-image", "gemini-3.1-flash-image-preview", 0.075, true},
		{"gemini flash", "gemini-3-flash-preview", 0.075, true},
		{"gemini pro-image", "gemini-3-pro-image-preview", 0.625, true},
		{"gemini pro", "gemini-3-pro-preview", 0.625, true},
		{"gemini unknown tier", "gemini-99-unknown", 0.075, true},

		// Claude families
		{"claude haiku", "claude-haiku-5-20260101", 0.5, true},
		{"claude sonnet", "claude-sonnet-5-20260101", 1.5, true},
		{"claude opus", "claude-opus-5-20260101", 7.5, true},
		{"claude unknown", "claude-unknown-99", 1.5, true},

		// GPT families
		{"gpt nano", "gpt-6-nano", 0.05, true},
		{"gpt mini", "gpt-6-mini", 0.2, true},
		{"gpt turbo", "gpt-5-turbo", 5.0, true},
		{"gpt default", "gpt-6", 1.25, true},

		// O-series
		{"o3 mini", "o3-mini-2025-99-99", 0.55, true},
		{"o3 pro", "o3-pro-2025-99-99", 10.0, true},
		{"o4 mini", "o4-mini-2025-99-99", 0.55, true},

		// DeepSeek
		{"deepseek chat", "deepseek-chat-v3", 0.07, true},
		{"deepseek reasoner", "deepseek-reasoner-v2", 0.275, true},
		{"deepseek unknown", "deepseek-v99", 0.07, true},

		// Unknown provider — no match
		{"unknown", "llama-3-70b", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio, matched := ModelFamilyFallback(tt.model)
			if matched != tt.wantMatch {
				t.Errorf("ModelFamilyFallback(%q) matched=%v, want %v", tt.model, matched, tt.wantMatch)
			}
			if matched && ratio != tt.wantRatio {
				t.Errorf("ModelFamilyFallback(%q) ratio=%v, want %v", tt.model, ratio, tt.wantRatio)
			}
		})
	}
}

func TestModelFamilyFallback_TierSpecificity(t *testing.T) {
	// flash-lite must match before flash
	ratio, ok := ModelFamilyFallback("gemini-4-flash-lite-exp")
	if !ok || ratio != 0.05 {
		t.Errorf("flash-lite should match 0.05, got %v (matched=%v)", ratio, ok)
	}

	// pro-image must match before pro
	ratio, ok = ModelFamilyFallback("gemini-4-pro-image-exp")
	if !ok || ratio != 0.625 {
		t.Errorf("pro-image should match 0.625, got %v (matched=%v)", ratio, ok)
	}
}
