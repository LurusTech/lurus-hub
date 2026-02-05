package app

import (
	"strings"
	"testing"
)

func TestSundaySearch(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pattern  string
		expected bool
	}{
		// Basic matches
		{"exact match", "hello", "hello", true},
		{"substring match", "hello world", "world", true},
		{"prefix match", "hello world", "hello", true},
		{"middle match", "the quick brown fox", "quick", true},

		// No matches
		{"no match", "hello world", "xyz", false},
		{"case sensitive no match", "Hello", "hello", false},
		{"partial no match", "abc", "abcd", false},

		// Edge cases
		{"empty text", "", "pattern", false},
		{"empty pattern", "text", "", true}, // empty pattern always matches
		{"both empty", "", "", true},
		{"single char match", "a", "a", true},
		{"single char no match", "a", "b", false},
		{"pattern longer than text", "ab", "abc", false},

		// Multiple occurrences
		{"multiple occurrences", "abcabc", "abc", true},
		{"overlapping pattern", "aaa", "aa", true},

		// Special patterns
		{"repeated chars", "aaabbbccc", "bbb", true},
		{"at end", "hello world", "ld", true},
		{"at start", "hello world", "he", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SundaySearch(tt.text, tt.pattern)
			if result != tt.expected {
				t.Errorf("SundaySearch(%q, %q) = %v, want %v",
					tt.text, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestSundaySearch_LongStrings(t *testing.T) {
	// Test with longer strings
	longText := strings.Repeat("abc", 1000) + "xyz" + strings.Repeat("def", 1000)

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"find in middle of long text", "xyz", true},
		{"find at start of long text", "abc", true},
		{"find at end of long text", "def", true},
		{"not found in long text", "qqq", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SundaySearch(longText, tt.pattern)
			if result != tt.expected {
				t.Errorf("SundaySearch(longText, %q) = %v, want %v",
					tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestRemoveDuplicate(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		// Basic cases
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"all duplicates", []string{"a", "a", "a"}, []string{"a"}},
		{"some duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},

		// Edge cases
		{"empty slice", []string{}, []string{}},
		{"single element", []string{"a"}, []string{"a"}},
		{"two same elements", []string{"a", "a"}, []string{"a"}},
		{"two different elements", []string{"a", "b"}, []string{"a", "b"}},

		// Preserves order
		{"preserves first occurrence order", []string{"c", "a", "b", "a", "c"}, []string{"c", "a", "b"}},

		// Empty strings
		{"empty strings", []string{"", "", "a"}, []string{"", "a"}},
		{"only empty strings", []string{"", "", ""}, []string{""}},

		// Special characters
		{"special chars", []string{"a\nb", "a\tb", "a\nb"}, []string{"a\nb", "a\tb"}},
		{"unicode", []string{"中文", "日本語", "中文"}, []string{"中文", "日本語"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveDuplicate(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("RemoveDuplicate(%v) = %v, want %v",
					tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("RemoveDuplicate(%v)[%d] = %q, want %q",
						tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestRemoveDuplicate_LargeInput(t *testing.T) {
	// Test with large input
	input := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		input[i] = string(rune('a' + (i % 26))) // Only 26 unique values
	}

	result := RemoveDuplicate(input)
	if len(result) != 26 {
		t.Errorf("RemoveDuplicate(large input) got %d unique elements, want 26", len(result))
	}

	// Verify order is preserved (first occurrence)
	for i := 0; i < 26; i++ {
		expected := string(rune('a' + i))
		if result[i] != expected {
			t.Errorf("RemoveDuplicate(large input)[%d] = %q, want %q", i, result[i], expected)
		}
	}
}

func TestAcKey(t *testing.T) {
	tests := []struct {
		name        string
		dict        []string
		shouldBeKey bool // true if should produce non-empty key
	}{
		{"empty dict", []string{}, false},
		{"single word", []string{"test"}, true},
		{"multiple words", []string{"hello", "world"}, true},
		{"only whitespace", []string{"  ", "\t", "\n"}, false},
		{"mixed with empty", []string{"test", "", "  "}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := acKey(tt.dict)
			if tt.shouldBeKey && result == "" {
				t.Errorf("acKey(%v) returned empty, expected non-empty key", tt.dict)
			}
			if !tt.shouldBeKey && result != "" {
				t.Errorf("acKey(%v) = %q, expected empty", tt.dict, result)
			}
		})
	}
}

func TestAcKey_Deterministic(t *testing.T) {
	dict := []string{"apple", "banana", "cherry"}

	// Same input should produce same key
	key1 := acKey(dict)
	key2 := acKey(dict)
	if key1 != key2 {
		t.Errorf("acKey is not deterministic: %q != %q", key1, key2)
	}

	// Different order should produce same key (sorted internally)
	dictReordered := []string{"cherry", "apple", "banana"}
	key3 := acKey(dictReordered)
	if key1 != key3 {
		t.Errorf("acKey should be order-independent: %q != %q", key1, key3)
	}

	// Case insensitive (normalized to lower)
	dictUpper := []string{"APPLE", "BANANA", "CHERRY"}
	key4 := acKey(dictUpper)
	if key1 != key4 {
		t.Errorf("acKey should be case-insensitive: %q != %q", key1, key4)
	}
}

func TestAcSearch(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		dict              []string
		stopImmediately   bool
		expectedFound     bool
		expectedMinHits   int // minimum expected hits when not stopping immediately
	}{
		// Basic searches
		{"single word found", "hello world", []string{"world"}, false, true, 1},
		{"single word not found", "hello world", []string{"xyz"}, false, false, 0},
		{"multiple words some found", "the quick brown fox", []string{"quick", "fox", "xyz"}, false, true, 2},
		{"all words found", "apple banana cherry", []string{"apple", "banana"}, false, true, 2},

		// Stop immediately
		{"stop immediately true", "apple banana cherry", []string{"apple", "banana", "cherry"}, true, true, 1},

		// Edge cases
		{"empty text", "", []string{"word"}, false, false, 0},
		{"empty dict", "some text", []string{}, false, false, 0},
		{"both empty", "", []string{}, false, false, 0},

		// Case sensitivity: AC machine converts dict to lowercase, but NOT the input text
		// So search is only case-insensitive if input text happens to be lowercase
		{"lowercase text matches lowercase dict", "hello world", []string{"hello"}, false, true, 1},
		{"uppercase text does NOT match lowercase dict", "HELLO WORLD", []string{"hello"}, false, false, 0},

		// Overlapping patterns
		{"overlapping", "abcabc", []string{"abc"}, false, true, 2},

		// Unicode
		{"unicode search", "你好世界", []string{"你好", "世界"}, false, true, 2},
		{"mixed unicode ascii", "hello你好world", []string{"hello", "你好"}, false, true, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, hits := AcSearch(tt.text, tt.dict, tt.stopImmediately)

			if found != tt.expectedFound {
				t.Errorf("AcSearch(%q, %v, %v) found = %v, want %v",
					tt.text, tt.dict, tt.stopImmediately, found, tt.expectedFound)
			}

			if tt.expectedFound && len(hits) < tt.expectedMinHits {
				t.Errorf("AcSearch(%q, %v, %v) hits = %v, want at least %d hits",
					tt.text, tt.dict, tt.stopImmediately, hits, tt.expectedMinHits)
			}

			if !tt.expectedFound && len(hits) > 0 {
				t.Errorf("AcSearch(%q, %v, %v) returned hits %v when not found",
					tt.text, tt.dict, tt.stopImmediately, hits)
			}
		})
	}
}

func TestAcSearch_Cache(t *testing.T) {
	dict := []string{"test", "word"}

	// First call builds cache
	found1, _ := AcSearch("test word here", dict, false)
	if !found1 {
		t.Error("First AcSearch should find match")
	}

	// Second call should use cache
	found2, _ := AcSearch("another test string", dict, false)
	if !found2 {
		t.Error("Second AcSearch should find match (using cache)")
	}

	// Different dict should build new cache entry
	dict2 := []string{"other", "words"}
	found3, _ := AcSearch("other content", dict2, false)
	if !found3 {
		t.Error("AcSearch with different dict should work")
	}
}

func BenchmarkSundaySearch(b *testing.B) {
	text := strings.Repeat("the quick brown fox jumps over the lazy dog ", 100)
	pattern := "lazy"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SundaySearch(text, pattern)
	}
}

func BenchmarkRemoveDuplicate(b *testing.B) {
	input := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		input[i] = string(rune('a' + (i % 26)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RemoveDuplicate(input)
	}
}

func BenchmarkAcSearch(b *testing.B) {
	text := strings.Repeat("the quick brown fox jumps over the lazy dog ", 100)
	dict := []string{"quick", "fox", "lazy", "dog"}

	// Warm up cache
	AcSearch(text, dict, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AcSearch(text, dict, false)
	}
}
