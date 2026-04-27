// Package openrouter_sync implements the periodic OpenRouter free-model
// import pipeline: fetch → classify → rank → diff → write. See
// doc/active_task.md for the full design.
package openrouter_sync

import (
	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
)

// modalitySet is a tiny string-set helper.
type modalitySet map[string]struct{}

func newModalitySet(ms []string) modalitySet {
	s := make(modalitySet, len(ms))
	for _, m := range ms {
		s[m] = struct{}{}
	}
	return s
}

func (s modalitySet) has(m string) bool { _, ok := s[m]; return ok }

// hasModality is a helper that doesn't allocate the set when only one lookup is needed.
func hasModality(list []string, target string) bool {
	for _, m := range list {
		if m == target {
			return true
		}
	}
	return false
}

// onlyText reports whether the modality slice contains text and nothing else.
func onlyText(list []string) bool {
	if len(list) == 0 {
		return false
	}
	for _, m := range list {
		if m != "text" {
			return false
		}
	}
	return true
}

// ClassifyOne returns the category bucket(s) a model belongs to.
// A model can hit multiple buckets (e.g. a model that takes audio and emits text
// is `asr`; if it also emitted audio it would be both `asr` and `tts`).
func ClassifyOne(in, out []string) []string {
	cats := make([]string, 0, 2)

	// llm_reasoning: pure text-in → text-included-out
	if onlyText(in) && hasModality(out, "text") {
		cats = append(cats, entity.OpenRouterCategoryLLMReasoning)
	}
	// vision: image in addition to text input, text-only output
	if hasModality(in, "image") && onlyText(out) {
		cats = append(cats, entity.OpenRouterCategoryVision)
	}
	// image_gen: any output that includes image
	if hasModality(out, "image") {
		cats = append(cats, entity.OpenRouterCategoryImageGen)
	}
	// asr: audio in, text-only out
	if hasModality(in, "audio") && onlyText(out) {
		cats = append(cats, entity.OpenRouterCategoryASR)
	}
	// tts: audio in output
	if hasModality(out, "audio") {
		cats = append(cats, entity.OpenRouterCategoryTTS)
	}

	return cats
}

// MatchesAny returns true iff any classification of (in, out) intersects `wanted`.
func MatchesAny(in, out []string, wanted []string) bool {
	if len(wanted) == 0 {
		return false
	}
	want := newModalitySet(wanted)
	for _, c := range ClassifyOne(in, out) {
		if want.has(c) {
			return true
		}
	}
	return false
}
