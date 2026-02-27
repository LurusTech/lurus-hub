package handler

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// MockRouter creates a minimal Gin engine for unit testing.
func MockRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ============================================================================
// Utility Function Unit Tests
// Pure functions — no DB or HTTP setup required
// ============================================================================

func TestHasRole(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		roles := []string{"admin", "editor", "viewer"}
		if !hasRole(roles, "admin") {
			t.Error("expected hasRole to return true for 'admin'")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		roles := []string{"admin", "editor"}
		if hasRole(roles, "superadmin") {
			t.Error("expected hasRole to return false for 'superadmin'")
		}
	})

	t.Run("empty_roles", func(t *testing.T) {
		if hasRole(nil, "admin") {
			t.Error("expected hasRole to return false for nil roles")
		}
		if hasRole([]string{}, "admin") {
			t.Error("expected hasRole to return false for empty roles")
		}
	})
}

func TestMaskSingleKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "short_key_under_8",
			key:      "abc",
			expected: "****",
		},
		{
			name:     "exact_8_chars",
			key:      "12345678",
			expected: "****",
		},
		{
			name:     "normal_key",
			key:      "sk-abcdefghijklmnop",
			expected: "sk-a****mnop",
		},
		{
			name:     "9_chars_boundary",
			key:      "123456789",
			expected: "1234****6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskSingleKey(tt.key)
			if got != tt.expected {
				t.Errorf("maskSingleKey(%q) = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "empty",
			key:      "",
			expected: "",
		},
		{
			name:     "single_key",
			key:      "sk-abcdefghijklmnop",
			expected: "sk-a****mnop",
		},
		{
			name:     "multi_line_keys",
			key:      "sk-abcdefghijklmnop\nsk-12345678901234xy",
			expected: "sk-a****mnop\nsk-1****34xy",
		},
		{
			name:     "trailing_newline",
			key:      "sk-abcdefghijklmnop\n",
			expected: "sk-a****mnop\n****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskKey(tt.key)
			if got != tt.expected {
				t.Errorf("maskKey(%q) = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}

func TestMaskRedemptionKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "short_key",
			key:      "abcd",
			expected: "****",
		},
		{
			name:     "exact_8_chars",
			key:      "12345678",
			expected: "****",
		},
		{
			name:     "normal_32_char_key",
			key:      "abcdefghijklmnopqrstuvwxyz123456",
			expected: "abcd************************3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskRedemptionKey(tt.key)
			if got != tt.expected {
				t.Errorf("maskRedemptionKey(%q) = %q, want %q", tt.key, got, tt.expected)
			}
			// Verify masked key contains the mask pattern for normal keys
			if len(tt.key) > 8 && !strings.Contains(got, "****") {
				t.Errorf("expected masked key to contain '****', got %q", got)
			}
		})
	}
}
