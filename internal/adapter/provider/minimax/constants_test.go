package minimax

import (
	"testing"
)

func TestMinimaxModelList_NonEmpty(t *testing.T) {
	if len(ModelList) == 0 {
		t.Error("ModelList should not be empty")
	}
}

func TestMinimaxModelList_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(ModelList))
	for _, m := range ModelList {
		if seen[m] {
			t.Errorf("duplicate model in ModelList: %q", m)
		}
		seen[m] = true
	}
}

func TestMinimaxModelList_ContainsExpectedModels(t *testing.T) {
	expected := []string{
		"MiniMax-Text-01",
		"MiniMax-01",
		"minimax-text-01",
	}

	modelSet := make(map[string]bool, len(ModelList))
	for _, m := range ModelList {
		modelSet[m] = true
	}

	for _, want := range expected {
		if !modelSet[want] {
			t.Errorf("ModelList missing expected model %q", want)
		}
	}
}

func TestMinimaxChannelName(t *testing.T) {
	if ChannelName != "minimax" {
		t.Errorf("ChannelName = %q, want %q", ChannelName, "minimax")
	}
}
