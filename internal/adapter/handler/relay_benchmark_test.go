package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/dto"
	"github.com/gin-gonic/gin"
)

// BenchmarkRequestValidation benchmarks the request validation path
func BenchmarkRequestValidation(b *testing.B) {
	gin.SetMode(gin.TestMode)

	reqBody := dto.GeneralOpenAIRequest{
		Model: "gpt-4",
		Messages: []dto.Message{
			{Role: "user", Content: "Hello, world!"},
		},
		MaxTokens: 100,
		Stream:    false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		// Simulate request body reading
		var req dto.GeneralOpenAIRequest
		_ = c.ShouldBindJSON(&req)
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization for responses
func BenchmarkJSONSerialization(b *testing.B) {
	response := dto.OpenAITextResponse{
		Id:      "chatcmpl-" + common.GetRandomString(24),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   "gpt-4",
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: "Hello! How can I help you today?",
				},
				FinishReason: "stop",
			},
		},
		Usage: dto.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(response)
	}
}

// BenchmarkJSONDeserialization benchmarks JSON deserialization for requests
func BenchmarkJSONDeserialization(b *testing.B) {
	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"Hello, world!"}],"max_tokens":100}`
	bodyBytes := []byte(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req dto.GeneralOpenAIRequest
		_ = json.Unmarshal(bodyBytes, &req)
	}
}

// BenchmarkContextSetup benchmarks context setup for relay
func BenchmarkContextSetup(b *testing.B) {
	gin.SetMode(gin.TestMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Simulate context setup
		c.Set(common.RequestIdKey, common.GetRandomString(16))
		c.Set("channel_id", 1)
		c.Set("channel_type", 1)
		c.Set("original_model", "gpt-4")
		c.Set("token_id", 1)
		c.Set("user_id", 1)

		// Read back values
		_ = c.GetString(common.RequestIdKey)
		_ = c.GetInt("channel_id")
	}
}

// BenchmarkParallelRequests benchmarks parallel request handling
func BenchmarkParallelRequests(b *testing.B) {
	gin.SetMode(gin.TestMode)

	reqBody := dto.GeneralOpenAIRequest{
		Model: "gpt-4",
		Messages: []dto.Message{
			{Role: "user", Content: "Hello!"},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			var req dto.GeneralOpenAIRequest
			_ = c.ShouldBindJSON(&req)
		}
	})
}

// BenchmarkWithContext tests context cancellation overhead
func BenchmarkWithContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		_ = ctx
		cancel()
	}
}

// BenchmarkRandomString benchmarks random string generation used in request IDs
func BenchmarkRandomString(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = common.GetRandomString(16)
	}
}

// BenchmarkLargeMessageSerialization benchmarks serialization of large messages
func BenchmarkLargeMessageSerialization(b *testing.B) {
	// Create a large message (simulating a long conversation)
	messages := make([]dto.Message, 50)
	for i := 0; i < 50; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages[i] = dto.Message{
			Role:    role,
			Content: "This is message number " + string(rune('0'+i%10)) + " with some content that makes it longer to simulate real-world usage patterns in chat applications.",
		}
	}

	reqBody := dto.GeneralOpenAIRequest{
		Model:     "gpt-4",
		Messages:  messages,
		MaxTokens: 4096,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(reqBody)
	}
}

// BenchmarkStreamResponse benchmarks streaming response chunk creation
func BenchmarkStreamResponse(b *testing.B) {
	content := "Hello"
	chunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl-abc123",
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   "gpt-4",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Index: 0,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: &content,
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(chunk)
		// Simulate SSE format
		_ = append([]byte("data: "), append(data, []byte("\n\n")...)...)
	}
}
