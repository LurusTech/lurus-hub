package app

import (
	"testing"

	"github.com/LurusTech/lurus-api/internal/pkg/dto"
)

func TestValidUsage(t *testing.T) {
	tests := []struct {
		name     string
		usage    *dto.Usage
		expected bool
	}{
		{
			name:     "nil usage",
			usage:    nil,
			expected: false,
		},
		{
			name: "zero usage",
			usage: &dto.Usage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			},
			expected: false,
		},
		{
			name: "only prompt tokens",
			usage: &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 0,
				TotalTokens:      100,
			},
			expected: true,
		},
		{
			name: "only completion tokens",
			usage: &dto.Usage{
				PromptTokens:     0,
				CompletionTokens: 50,
				TotalTokens:      50,
			},
			expected: true,
		},
		{
			name: "both tokens",
			usage: &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
			expected: true,
		},
		{
			name: "negative tokens still valid",
			usage: &dto.Usage{
				PromptTokens:     -1, // unusual but non-zero
				CompletionTokens: 0,
				TotalTokens:      -1,
			},
			expected: true, // non-zero prompt tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidUsage(tt.usage)
			if result != tt.expected {
				t.Errorf("ValidUsage(%+v) = %v, want %v", tt.usage, result, tt.expected)
			}
		})
	}
}

func TestValidUsage_EdgeCases(t *testing.T) {
	// Test with large numbers
	largeUsage := &dto.Usage{
		PromptTokens:     1000000000,
		CompletionTokens: 1000000000,
		TotalTokens:      2000000000,
	}
	if !ValidUsage(largeUsage) {
		t.Error("ValidUsage should return true for large token counts")
	}

	// Test pointer behavior
	var nilUsage *dto.Usage = nil
	if ValidUsage(nilUsage) {
		t.Error("ValidUsage should return false for nil pointer")
	}
}
