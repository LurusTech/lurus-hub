package app

import (
	"sync"
	"testing"
)

func TestInitTokenEncoders(t *testing.T) {
	// This should not panic
	InitTokenEncoders()

	// After init, defaultTokenEncoder should be non-nil
	if defaultTokenEncoder == nil {
		t.Error("defaultTokenEncoder should not be nil after InitTokenEncoders")
	}
}

func TestGetTokenEncoder(t *testing.T) {
	// Ensure initialized
	InitTokenEncoders()

	tests := []struct {
		name  string
		model string
	}{
		{"gpt-4", "gpt-4"},
		{"gpt-3.5-turbo", "gpt-3.5-turbo"},
		{"unknown model", "some-unknown-model"},
		{"empty model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := getTokenEncoder(tt.model)
			if encoder == nil {
				t.Errorf("getTokenEncoder(%q) returned nil", tt.model)
			}
		})
	}
}

func TestGetTokenEncoder_Caching(t *testing.T) {
	InitTokenEncoders()

	model := "test-cache-model"

	// First call
	encoder1 := getTokenEncoder(model)

	// Second call should return cached encoder
	encoder2 := getTokenEncoder(model)

	// Both should be the same instance (pointer equality)
	if encoder1 != encoder2 {
		t.Error("getTokenEncoder should return cached encoder on second call")
	}
}

func TestGetTokenEncoder_Concurrent(t *testing.T) {
	InitTokenEncoders()

	var wg sync.WaitGroup
	models := []string{"model-a", "model-b", "model-c"}

	// Run concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			model := models[idx%len(models)]
			encoder := getTokenEncoder(model)
			if encoder == nil {
				t.Errorf("getTokenEncoder(%q) returned nil in concurrent call", model)
			}
		}(i)
	}

	wg.Wait()
}

func TestGetTokenNum(t *testing.T) {
	InitTokenEncoders()

	encoder := getTokenEncoder("gpt-4")
	if encoder == nil {
		t.Fatal("failed to get encoder")
	}

	tests := []struct {
		name     string
		text     string
		minCount int
		maxCount int
	}{
		{"empty string", "", 0, 0},
		{"single word", "hello", 1, 3},
		{"sentence", "Hello, how are you?", 3, 10},
		{"numbers", "12345", 1, 5},
		{"chinese", "你好世界", 2, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := getTokenNum(encoder, tt.text)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("getTokenNum(%q) = %d, want between %d and %d",
					tt.text, count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func BenchmarkGetTokenEncoder(b *testing.B) {
	InitTokenEncoders()

	b.Run("cached", func(b *testing.B) {
		// Pre-cache the model
		_ = getTokenEncoder("gpt-4")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = getTokenEncoder("gpt-4")
		}
	})

	b.Run("uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Use different model names to avoid cache hits
			model := "bench-model-" + string(rune('a'+i%26))
			_ = getTokenEncoder(model)
		}
	})
}

func BenchmarkGetTokenNum(b *testing.B) {
	InitTokenEncoders()
	encoder := getTokenEncoder("gpt-4")

	texts := []string{
		"short",
		"This is a medium length sentence with several words.",
		"This is a longer piece of text that contains multiple sentences. It should take more time to tokenize. The tokenizer needs to process each character and determine the appropriate token boundaries.",
	}

	for _, text := range texts {
		b.Run("len="+string(rune('0'+len(text)/10)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				getTokenNum(encoder, text)
			}
		})
	}
}
