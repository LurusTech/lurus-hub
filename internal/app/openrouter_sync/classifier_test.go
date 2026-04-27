package openrouter_sync

import (
	"sort"
	"testing"

	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
)

func TestClassifyOne(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
		want []string
	}{
		{
			name: "pure text → llm_reasoning",
			in:   []string{"text"},
			out:  []string{"text"},
			want: []string{entity.OpenRouterCategoryLLMReasoning},
		},
		{
			name: "text+image in, text out → vision (not llm)",
			in:   []string{"text", "image"},
			out:  []string{"text"},
			want: []string{entity.OpenRouterCategoryVision},
		},
		{
			name: "text in, image out → image_gen",
			in:   []string{"text"},
			out:  []string{"image"},
			want: []string{entity.OpenRouterCategoryImageGen},
		},
		{
			name: "audio in, text out → asr",
			in:   []string{"audio"},
			out:  []string{"text"},
			want: []string{entity.OpenRouterCategoryASR},
		},
		{
			name: "text in, audio out → tts",
			in:   []string{"text"},
			out:  []string{"audio"},
			want: []string{entity.OpenRouterCategoryTTS},
		},
		{
			name: "text+image in, text+image out → vision drops (output not only text), image_gen wins",
			in:   []string{"text", "image"},
			out:  []string{"text", "image"},
			want: []string{entity.OpenRouterCategoryImageGen},
		},
		{
			name: "empty input → no category",
			in:   []string{},
			out:  []string{"text"},
			want: []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyOne(tc.in, tc.out)
			if got == nil {
				got = []string{}
			}
			sort.Strings(got)
			want := append([]string{}, tc.want...)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("got %v, want %v", got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestMatchesAny(t *testing.T) {
	if !MatchesAny([]string{"text"}, []string{"text"}, []string{entity.OpenRouterCategoryLLMReasoning}) {
		t.Errorf("text→text should match llm_reasoning")
	}
	if MatchesAny([]string{"text"}, []string{"text"}, []string{entity.OpenRouterCategoryVision}) {
		t.Errorf("text→text should NOT match vision")
	}
	if !MatchesAny([]string{"text", "image"}, []string{"text"},
		[]string{entity.OpenRouterCategoryLLMReasoning, entity.OpenRouterCategoryVision}) {
		t.Errorf("vision model should match the {llm,vision} filter")
	}
	if MatchesAny([]string{"text"}, []string{"text"}, []string{}) {
		t.Errorf("empty wanted set should never match")
	}
}
