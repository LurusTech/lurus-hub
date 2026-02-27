package dto

import (
	"encoding/json"
	"testing"
)

func TestGeminiUsageMetadata_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name                    string
		jsonStr                 string
		wantPrompt              int
		wantCandidates          int
		wantTotal               int
		wantThoughts            int
		wantToolUse             int
		wantDetailsLen          int
	}{
		{
			name: "all fields",
			jsonStr: `{
				"promptTokenCount": 100,
				"candidatesTokenCount": 50,
				"totalTokenCount": 200,
				"thoughtsTokenCount": 30,
				"toolUsePromptTokenCount": 20,
				"promptTokensDetails": [{"modality": "TEXT", "tokenCount": 80}]
			}`,
			wantPrompt:     100,
			wantCandidates: 50,
			wantTotal:      200,
			wantThoughts:   30,
			wantToolUse:    20,
			wantDetailsLen: 1,
		},
		{
			name: "omitted toolUse",
			jsonStr: `{
				"promptTokenCount": 100,
				"candidatesTokenCount": 50,
				"totalTokenCount": 150,
				"thoughtsTokenCount": 0
			}`,
			wantPrompt:     100,
			wantCandidates: 50,
			wantTotal:      150,
			wantThoughts:   0,
			wantToolUse:    0,
			wantDetailsLen: 0,
		},
		{
			name: "explicit null toolUse",
			jsonStr: `{
				"promptTokenCount": 100,
				"candidatesTokenCount": 50,
				"totalTokenCount": 150,
				"toolUsePromptTokenCount": null
			}`,
			wantPrompt:     100,
			wantCandidates: 50,
			wantTotal:      150,
			wantToolUse:    0,
			wantDetailsLen: 0,
		},
		{
			name: "explicit zero toolUse",
			jsonStr: `{
				"promptTokenCount": 100,
				"candidatesTokenCount": 50,
				"totalTokenCount": 150,
				"toolUsePromptTokenCount": 0
			}`,
			wantPrompt:     100,
			wantCandidates: 50,
			wantTotal:      150,
			wantToolUse:    0,
			wantDetailsLen: 0,
		},
		{
			name:           "empty details array",
			jsonStr:        `{"promptTokenCount": 50, "promptTokensDetails": []}`,
			wantPrompt:     50,
			wantDetailsLen: 0,
		},
		{
			name:    "all zeros",
			jsonStr: `{"promptTokenCount": 0, "candidatesTokenCount": 0, "totalTokenCount": 0, "thoughtsTokenCount": 0, "toolUsePromptTokenCount": 0}`,
		},
		{
			name:    "empty object",
			jsonStr: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta GeminiUsageMetadata
			if err := json.Unmarshal([]byte(tt.jsonStr), &meta); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if meta.PromptTokenCount != tt.wantPrompt {
				t.Errorf("PromptTokenCount = %d, want %d", meta.PromptTokenCount, tt.wantPrompt)
			}
			if meta.CandidatesTokenCount != tt.wantCandidates {
				t.Errorf("CandidatesTokenCount = %d, want %d", meta.CandidatesTokenCount, tt.wantCandidates)
			}
			if meta.TotalTokenCount != tt.wantTotal {
				t.Errorf("TotalTokenCount = %d, want %d", meta.TotalTokenCount, tt.wantTotal)
			}
			if meta.ThoughtsTokenCount != tt.wantThoughts {
				t.Errorf("ThoughtsTokenCount = %d, want %d", meta.ThoughtsTokenCount, tt.wantThoughts)
			}
			if meta.ToolUsePromptTokenCount != tt.wantToolUse {
				t.Errorf("ToolUsePromptTokenCount = %d, want %d", meta.ToolUsePromptTokenCount, tt.wantToolUse)
			}
			if len(meta.PromptTokensDetails) != tt.wantDetailsLen {
				t.Errorf("len(PromptTokensDetails) = %d, want %d", len(meta.PromptTokensDetails), tt.wantDetailsLen)
			}
		})
	}
}
