package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/dto"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetResponseID(t *testing.T) {
	t.Run("returns chatcmpl prefix with request id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(common.RequestIdKey, "abc123")

		got := GetResponseID(c)
		if got != "chatcmpl-abc123" {
			t.Errorf("GetResponseID() = %q, want %q", got, "chatcmpl-abc123")
		}
	})

	t.Run("returns chatcmpl prefix with empty request id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		got := GetResponseID(c)
		if got != "chatcmpl-" {
			t.Errorf("GetResponseID() = %q, want %q", got, "chatcmpl-")
		}
	})
}

func TestGenerateStartEmptyResponse(t *testing.T) {
	t.Run("basic structure with nil fingerprint", func(t *testing.T) {
		resp := GenerateStartEmptyResponse("test-id", 1234567890, "gpt-4", nil)

		if resp.Id != "test-id" {
			t.Errorf("Id = %q, want %q", resp.Id, "test-id")
		}
		if resp.Object != "chat.completion.chunk" {
			t.Errorf("Object = %q, want %q", resp.Object, "chat.completion.chunk")
		}
		if resp.Created != 1234567890 {
			t.Errorf("Created = %d, want %d", resp.Created, 1234567890)
		}
		if resp.Model != "gpt-4" {
			t.Errorf("Model = %q, want %q", resp.Model, "gpt-4")
		}
		if resp.SystemFingerprint != nil {
			t.Errorf("SystemFingerprint = %v, want nil", resp.SystemFingerprint)
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].Delta.Role != "assistant" {
			t.Errorf("Delta.Role = %q, want %q", resp.Choices[0].Delta.Role, "assistant")
		}
		if resp.Choices[0].Delta.GetContentString() != "" {
			t.Errorf("Delta.Content = %q, want empty", resp.Choices[0].Delta.GetContentString())
		}
	})

	t.Run("with system fingerprint", func(t *testing.T) {
		fp := "fp_abc123"
		resp := GenerateStartEmptyResponse("test-id", 0, "gpt-4", &fp)

		if resp.SystemFingerprint == nil {
			t.Fatal("SystemFingerprint should not be nil")
		}
		if *resp.SystemFingerprint != fp {
			t.Errorf("SystemFingerprint = %q, want %q", *resp.SystemFingerprint, fp)
		}
	})
}

func TestGenerateStopResponse(t *testing.T) {
	tests := []struct {
		name         string
		finishReason string
	}{
		{"stop", "stop"},
		{"length", "length"},
		{"tool_calls", "tool_calls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := GenerateStopResponse("id-1", 100, "gpt-4", tt.finishReason)

			if resp.Id != "id-1" {
				t.Errorf("Id = %q, want %q", resp.Id, "id-1")
			}
			if resp.Object != "chat.completion.chunk" {
				t.Errorf("Object = %q, want %q", resp.Object, "chat.completion.chunk")
			}
			if resp.Created != 100 {
				t.Errorf("Created = %d, want %d", resp.Created, 100)
			}
			if resp.SystemFingerprint != nil {
				t.Errorf("SystemFingerprint = %v, want nil", resp.SystemFingerprint)
			}
			if len(resp.Choices) != 1 {
				t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
			}
			if resp.Choices[0].FinishReason == nil {
				t.Fatal("FinishReason should not be nil")
			}
			if *resp.Choices[0].FinishReason != tt.finishReason {
				t.Errorf("FinishReason = %q, want %q", *resp.Choices[0].FinishReason, tt.finishReason)
			}
		})
	}
}

func TestGenerateFinalUsageResponse(t *testing.T) {
	t.Run("basic structure with usage", func(t *testing.T) {
		usage := dto.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		}
		resp := GenerateFinalUsageResponse("id-2", 200, "gpt-4", usage)

		if resp.Id != "id-2" {
			t.Errorf("Id = %q, want %q", resp.Id, "id-2")
		}
		if resp.Object != "chat.completion.chunk" {
			t.Errorf("Object = %q, want %q", resp.Object, "chat.completion.chunk")
		}
		if resp.Created != 200 {
			t.Errorf("Created = %d, want %d", resp.Created, 200)
		}
		if resp.Model != "gpt-4" {
			t.Errorf("Model = %q, want %q", resp.Model, "gpt-4")
		}
		if resp.SystemFingerprint != nil {
			t.Errorf("SystemFingerprint should be nil")
		}
		if len(resp.Choices) != 0 {
			t.Errorf("len(Choices) = %d, want 0", len(resp.Choices))
		}
		if resp.Usage == nil {
			t.Fatal("Usage should not be nil")
		}
		if resp.Usage.PromptTokens != 100 {
			t.Errorf("PromptTokens = %d, want 100", resp.Usage.PromptTokens)
		}
		if resp.Usage.CompletionTokens != 50 {
			t.Errorf("CompletionTokens = %d, want 50", resp.Usage.CompletionTokens)
		}
		if resp.Usage.TotalTokens != 150 {
			t.Errorf("TotalTokens = %d, want 150", resp.Usage.TotalTokens)
		}
	})

	t.Run("zero usage fields preserved", func(t *testing.T) {
		usage := dto.Usage{}
		resp := GenerateFinalUsageResponse("id-3", 300, "gpt-3.5", usage)

		if resp.Usage == nil {
			t.Fatal("Usage should not be nil")
		}
		if resp.Usage.PromptTokens != 0 || resp.Usage.CompletionTokens != 0 || resp.Usage.TotalTokens != 0 {
			t.Errorf("zero usage fields not preserved: prompt=%d, completion=%d, total=%d",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
	})
}
