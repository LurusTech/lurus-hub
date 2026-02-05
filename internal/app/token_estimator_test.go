package app

import (
	"strings"
	"testing"
)

func TestIsCJK(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// Chinese characters
		{"chinese char 中", '中', true},
		{"chinese char 文", '文', true},
		{"chinese char 你", '你', true},
		{"chinese char 好", '好', true},

		// Japanese Hiragana (0x3040-0x309F)
		{"hiragana あ", 'あ', true},
		{"hiragana ひ", 'ひ', true},

		// Japanese Katakana (0x30A0-0x30FF)
		{"katakana ア", 'ア', true},
		{"katakana カ", 'カ', true},

		// Korean Hangul (0xAC00-0xD7A3)
		{"korean 한", '한', true},
		{"korean 글", '글', true},

		// Non-CJK
		{"latin a", 'a', false},
		{"latin Z", 'Z', false},
		{"digit 5", '5', false},
		{"space", ' ', false},
		{"emoji", '😀', false},
		{"punctuation", '.', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCJK(tt.input)
			if result != tt.expected {
				t.Errorf("isCJK(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsLatinOrNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// Latin letters
		{"lowercase a", 'a', true},
		{"lowercase z", 'z', true},
		{"uppercase A", 'A', true},
		{"uppercase Z", 'Z', true},

		// Numbers
		{"digit 0", '0', true},
		{"digit 5", '5', true},
		{"digit 9", '9', true},

		// Non-latin, non-number
		{"space", ' ', false},
		{"dot", '.', false},
		{"comma", ',', false},
		// Note: unicode.IsLetter returns true for CJK, so Chinese chars ARE letters
		{"chinese is letter", '中', true},
		{"emoji", '😀', false},
		{"at sign", '@', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLatinOrNumber(tt.input)
			if result != tt.expected {
				t.Errorf("isLatinOrNumber(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsEmoji(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// Common emojis (in the defined ranges)
		{"emoji face", '😀', true},
		{"emoji heart", '❤', true},    // U+2764 in Dingbats range
		{"emoji sun", '☀', true},      // U+2600 in Misc Symbols range
		{"emoji check", '✅', true},    // U+2705 in Dingbats range
		// Note: ⭐ (U+2B50) is NOT in the isEmoji ranges, classified differently
		{"star not emoji", '⭐', false},

		// Non-emoji
		{"letter a", 'a', false},
		{"digit 1", '1', false},
		{"chinese", '中', false},
		{"space", ' ', false},
		{"punctuation", '.', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmoji(tt.input)
			if result != tt.expected {
				t.Errorf("isEmoji(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsMathSymbol(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// Math symbols
		{"sum", '∑', true},
		{"integral", '∫', true},
		{"partial", '∂', true},
		{"sqrt", '√', true},
		{"infinity", '∞', true},
		{"less equal", '≤', true},
		{"greater equal", '≥', true},
		{"not equal", '≠', true},
		{"approx", '≈', true},
		{"plus minus", '±', true},
		{"superscript 2", '²', true},
		{"superscript 3", '³', true},
		{"subscript 1", '₁', true},

		// Non-math
		{"plus", '+', false},
		{"minus", '-', false},
		{"equals", '=', false},
		{"letter", 'a', false},
		{"digit", '1', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMathSymbol(tt.input)
			if result != tt.expected {
				t.Errorf("isMathSymbol(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsURLDelim(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// URL delimiters
		{"slash", '/', true},
		{"colon", ':', true},
		{"question", '?', true},
		{"ampersand", '&', true},
		{"equals", '=', true},
		{"semicolon", ';', true},
		{"hash", '#', true},
		{"percent", '%', true},

		// Non-URL delimiters
		{"dot", '.', false},
		{"comma", ',', false},
		{"letter", 'a', false},
		{"digit", '1', false},
		{"at", '@', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isURLDelim(tt.input)
			if result != tt.expected {
				t.Errorf("isURLDelim(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEstimateToken(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		text     string
		minToken int // minimum expected tokens
		maxToken int // maximum expected tokens (for range check)
	}{
		// Empty string
		{"empty OpenAI", OpenAI, "", 0, 0},
		{"empty Gemini", Gemini, "", 0, 0},
		{"empty Claude", Claude, "", 0, 0},

		// Simple English words
		{"single word", OpenAI, "hello", 1, 3},
		{"two words", OpenAI, "hello world", 2, 5},
		{"sentence", OpenAI, "The quick brown fox", 3, 8},

		// Numbers
		{"single number", OpenAI, "123", 1, 3},
		{"mixed word number", OpenAI, "version3", 2, 5},

		// CJK characters
		{"chinese", OpenAI, "你好世界", 3, 6},
		{"japanese", OpenAI, "こんにちは", 4, 8},
		{"korean", OpenAI, "안녕하세요", 4, 8},

		// Emojis
		{"single emoji", OpenAI, "😀", 1, 4},
		{"multiple emojis", OpenAI, "😀🎉❤️", 3, 10},

		// URLs
		{"simple url", OpenAI, "https://example.com/path", 5, 15},

		// Math symbols
		{"math expression", OpenAI, "∑∫∂√", 4, 15},

		// Mixed content
		{"mixed en-cn", OpenAI, "Hello 你好", 2, 6},
		{"mixed all", OpenAI, "Hello 你好 123 😀", 4, 12},

		// Newlines and spaces (each word + newline/tab adds to count)
		{"with newlines", OpenAI, "line1\nline2\nline3", 3, 12},
		{"with tabs", OpenAI, "col1\tcol2\tcol3", 3, 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateToken(tt.provider, tt.text)
			if result < tt.minToken || result > tt.maxToken {
				t.Errorf("EstimateToken(%s, %q) = %d, want between %d and %d",
					tt.provider, tt.text, result, tt.minToken, tt.maxToken)
			}
		})
	}
}

func TestEstimateToken_DifferentProviders(t *testing.T) {
	// Same text should produce different results for different providers
	text := "Hello 你好 world 世界 123"

	openaiTokens := EstimateToken(OpenAI, text)
	geminiTokens := EstimateToken(Gemini, text)
	claudeTokens := EstimateToken(Claude, text)

	// All should be positive
	if openaiTokens <= 0 {
		t.Errorf("OpenAI tokens = %d, want > 0", openaiTokens)
	}
	if geminiTokens <= 0 {
		t.Errorf("Gemini tokens = %d, want > 0", geminiTokens)
	}
	if claudeTokens <= 0 {
		t.Errorf("Claude tokens = %d, want > 0", claudeTokens)
	}

	// Log the results for reference (not a strict assertion since ratios may vary)
	t.Logf("Token counts for %q: OpenAI=%d, Gemini=%d, Claude=%d",
		text, openaiTokens, geminiTokens, claudeTokens)
}

func TestEstimateToken_UnknownProvider(t *testing.T) {
	// Unknown provider should fall back to OpenAI
	text := "hello world"
	unknownResult := EstimateToken(Unknown, text)
	openaiResult := EstimateToken(OpenAI, text)

	if unknownResult != openaiResult {
		t.Errorf("Unknown provider result %d != OpenAI result %d", unknownResult, openaiResult)
	}
}

func TestEstimateTokenByModel(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		text         string
		expectedType Provider
	}{
		// OpenAI models
		{"gpt-4", "gpt-4", "hello", OpenAI},
		{"gpt-4o", "gpt-4o", "hello", OpenAI},
		{"gpt-3.5-turbo", "gpt-3.5-turbo", "hello", OpenAI},

		// Gemini models
		{"gemini-pro", "gemini-pro", "hello", Gemini},
		{"gemini-1.5-flash", "gemini-1.5-flash", "hello", Gemini},
		{"GEMINI-PRO", "GEMINI-PRO", "hello", Gemini}, // case insensitive

		// Claude models
		{"claude-3-opus", "claude-3-opus", "hello", Claude},
		{"claude-3-sonnet", "claude-3-sonnet", "hello", Claude},
		{"CLAUDE-3-HAIKU", "CLAUDE-3-HAIKU", "hello", Claude}, // case insensitive

		// Default to OpenAI
		{"unknown model", "some-unknown-model", "hello", OpenAI},
		{"empty model", "", "hello", OpenAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokenByModel(tt.model, tt.text)
			expected := EstimateToken(tt.expectedType, tt.text)

			if result != expected {
				t.Errorf("EstimateTokenByModel(%q, %q) = %d, want %d (from %s)",
					tt.model, tt.text, result, expected, tt.expectedType)
			}
		})
	}
}

func TestEstimateTokenByModel_EmptyText(t *testing.T) {
	result := EstimateTokenByModel("gpt-4", "")
	if result != 0 {
		t.Errorf("EstimateTokenByModel with empty text = %d, want 0", result)
	}
}

func TestGetMultipliers(t *testing.T) {
	// Test that multipliers are returned correctly for each provider
	providers := []Provider{OpenAI, Gemini, Claude, Unknown}

	for _, p := range providers {
		m := getMultipliers(p)

		// All multipliers should be positive (except BasePad which can be 0)
		if m.Word <= 0 {
			t.Errorf("getMultipliers(%s).Word = %f, want > 0", p, m.Word)
		}
		if m.CJK <= 0 {
			t.Errorf("getMultipliers(%s).CJK = %f, want > 0", p, m.CJK)
		}
	}
}

func BenchmarkEstimateToken(b *testing.B) {
	text := strings.Repeat("Hello world 你好世界 123 😀 ", 100)

	b.Run("OpenAI", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EstimateToken(OpenAI, text)
		}
	})

	b.Run("Gemini", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EstimateToken(Gemini, text)
		}
	})

	b.Run("Claude", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EstimateToken(Claude, text)
		}
	})
}

func BenchmarkEstimateTokenByModel(b *testing.B) {
	text := strings.Repeat("Hello world 你好世界 ", 100)
	models := []string{"gpt-4", "gemini-pro", "claude-3-opus"}

	for _, model := range models {
		b.Run(model, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				EstimateTokenByModel(model, text)
			}
		})
	}
}
