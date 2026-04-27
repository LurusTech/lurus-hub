package app

import (
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/dto"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting"
)

func TestSensitiveWordContains(t *testing.T) {
	// Save original and restore after test
	originalWords := setting.SensitiveWords
	defer func() { setting.SensitiveWords = originalWords }()

	tests := []struct {
		name          string
		words         []string
		text          string
		expectedFound bool
		expectedLen   int // expected number of words found
	}{
		{
			name:          "empty words list",
			words:         []string{},
			text:          "some text with bad words",
			expectedFound: false,
			expectedLen:   0,
		},
		{
			name:          "empty text",
			words:         []string{"bad", "evil"},
			text:          "",
			expectedFound: false,
			expectedLen:   0,
		},
		{
			name:          "no match",
			words:         []string{"bad", "evil"},
			text:          "this is a good text",
			expectedFound: false,
			expectedLen:   0,
		},
		{
			name:          "single match",
			words:         []string{"bad", "evil"},
			text:          "this is a bad text",
			expectedFound: true,
			expectedLen:   1,
		},
		{
			name:          "multiple matches",
			words:         []string{"bad", "evil"},
			text:          "bad and evil are both bad",
			expectedFound: true,
			expectedLen:   1, // stopImmediately=true, so only 1
		},
		{
			name:          "case insensitive",
			words:         []string{"bad"},
			text:          "This is BAD text",
			expectedFound: true,
			expectedLen:   1,
		},
		{
			name:          "chinese sensitive words",
			words:         []string{"敏感词", "违禁"},
			text:          "这里有敏感词存在",
			expectedFound: true,
			expectedLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting.SensitiveWords = tt.words

			found, words := SensitiveWordContains(tt.text)

			if found != tt.expectedFound {
				t.Errorf("SensitiveWordContains() found = %v, want %v", found, tt.expectedFound)
			}
			if len(words) < tt.expectedLen {
				t.Errorf("SensitiveWordContains() words count = %d, want >= %d", len(words), tt.expectedLen)
			}
		})
	}
}

func TestCheckSensitiveText(t *testing.T) {
	// Save original and restore after test
	originalWords := setting.SensitiveWords
	defer func() { setting.SensitiveWords = originalWords }()

	setting.SensitiveWords = []string{"forbidden", "banned"}

	tests := []struct {
		name          string
		text          string
		expectedFound bool
	}{
		{"clean text", "this is clean", false},
		{"has forbidden", "this is forbidden", true},
		{"has banned", "this is banned", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, _ := CheckSensitiveText(tt.text)
			if found != tt.expectedFound {
				t.Errorf("CheckSensitiveText(%q) = %v, want %v", tt.text, found, tt.expectedFound)
			}
		})
	}
}

func TestCheckSensitiveMessages(t *testing.T) {
	// Save original and restore after test
	originalWords := setting.SensitiveWords
	defer func() { setting.SensitiveWords = originalWords }()

	setting.SensitiveWords = []string{"badword", "harmful"}

	tests := []struct {
		name          string
		messages      []dto.Message
		expectError   bool
		expectWords   int
	}{
		{
			name:          "empty messages",
			messages:      []dto.Message{},
			expectError:   false,
			expectWords:   0,
		},
		{
			name: "clean messages",
			messages: []dto.Message{
				{Content: "This is a clean message"},
				{Content: "Another clean one"},
			},
			expectError: false,
			expectWords: 0,
		},
		{
			name: "message with sensitive word",
			messages: []dto.Message{
				{Content: "This contains badword here"},
			},
			expectError: true,
			expectWords: 1,
		},
		{
			name: "image url message skipped",
			messages: []dto.Message{
				{Content: []map[string]interface{}{
					{"type": "image_url", "image_url": map[string]interface{}{"url": "https://example.com/badword.png"}},
				}},
			},
			expectError: false,
			expectWords: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, err := CheckSensitiveMessages(tt.messages)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(words) != tt.expectWords {
				t.Errorf("got %d words, want %d", len(words), tt.expectWords)
			}
		})
	}
}

func TestSensitiveWordReplace(t *testing.T) {
	// Save original and restore after test
	originalWords := setting.SensitiveWords
	defer func() { setting.SensitiveWords = originalWords }()

	tests := []struct {
		name               string
		words              []string
		text               string
		returnImmediately  bool
		expectedFound      bool
		expectedContains   string // substring that should be in result
		expectedNotContain string // substring that should NOT be in result
	}{
		{
			name:              "empty words",
			words:             []string{},
			text:              "some text",
			returnImmediately: false,
			expectedFound:     false,
		},
		{
			name:              "no match",
			words:             []string{"bad"},
			text:              "good text",
			returnImmediately: false,
			expectedFound:     false,
		},
		{
			name:               "single replacement",
			words:              []string{"bad"},
			text:               "this is bad text",
			returnImmediately:  false,
			expectedFound:      true,
			expectedContains:   "**###**",
			expectedNotContain: "bad",
		},
		{
			name:              "return immediately",
			words:             []string{"bad", "evil"},
			text:              "bad and evil text",
			returnImmediately: true,
			expectedFound:     true,
			expectedContains:  "**###**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting.SensitiveWords = tt.words

			found, _, result := SensitiveWordReplace(tt.text, tt.returnImmediately)

			if found != tt.expectedFound {
				t.Errorf("SensitiveWordReplace() found = %v, want %v", found, tt.expectedFound)
			}
			if tt.expectedContains != "" && !contains(result, tt.expectedContains) {
				t.Errorf("result %q should contain %q", result, tt.expectedContains)
			}
			if tt.expectedNotContain != "" && contains(result, tt.expectedNotContain) {
				t.Errorf("result %q should NOT contain %q", result, tt.expectedNotContain)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
