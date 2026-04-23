package claude

import (
	"testing"

	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/dto"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// stopReasonClaude2OpenAI
// ---------------------------------------------------------------------------

func TestStopReasonClaude2OpenAI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"stop_sequence", "stop"},
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"unknown_reason", "unknown_reason"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stopReasonClaude2OpenAI(tt.input)
			if got != tt.want {
				t.Errorf("stopReasonClaude2OpenAI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RequestOpenAI2ClaudeComplete
// ---------------------------------------------------------------------------

func TestRequestOpenAI2ClaudeComplete(t *testing.T) {
	t.Run("basic messages to prompt format", func(t *testing.T) {
		req := dto.GeneralOpenAIRequest{
			Model:       "claude-2.1",
			Stream:      true,
			Temperature: common.GetPointer(0.7),
			TopP:        0.9,
			Messages: []dto.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "How are you?"},
			},
		}
		result := RequestOpenAI2ClaudeComplete(req)

		if result.Model != "claude-2.1" {
			t.Errorf("Model = %q, want %q", result.Model, "claude-2.1")
		}
		if result.Stream != true {
			t.Error("Stream should be true")
		}

		// prompt should contain Human: and Assistant: prefixes
		wantPrompt := "\n\nHuman: Hello\n\nAssistant: Hi there\n\nHuman: How are you?\n\nAssistant:"
		if result.Prompt != wantPrompt {
			t.Errorf("Prompt = %q, want %q", result.Prompt, wantPrompt)
		}
	})

	t.Run("system message becomes initial prompt", func(t *testing.T) {
		req := dto.GeneralOpenAIRequest{
			Model: "claude-2",
			Messages: []dto.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hi"},
			},
		}
		result := RequestOpenAI2ClaudeComplete(req)

		wantPrompt := "You are helpful.\n\nHuman: Hi\n\nAssistant:"
		if result.Prompt != wantPrompt {
			t.Errorf("Prompt = %q, want %q", result.Prompt, wantPrompt)
		}
	})

	t.Run("default MaxTokensToSample is 4096", func(t *testing.T) {
		req := dto.GeneralOpenAIRequest{
			Model:    "claude-2",
			Messages: []dto.Message{{Role: "user", Content: "Hi"}},
		}
		result := RequestOpenAI2ClaudeComplete(req)

		if result.MaxTokensToSample != 4096 {
			t.Errorf("MaxTokensToSample = %d, want 4096", result.MaxTokensToSample)
		}
	})

	t.Run("model and params preserved", func(t *testing.T) {
		temp := 0.5
		req := dto.GeneralOpenAIRequest{
			Model:       "claude-instant-1.2",
			Temperature: &temp,
			TopP:        0.8,
			TopK:        40,
			Messages:    []dto.Message{{Role: "user", Content: "test"}},
		}
		result := RequestOpenAI2ClaudeComplete(req)

		if result.Model != "claude-instant-1.2" {
			t.Errorf("Model = %q, want %q", result.Model, "claude-instant-1.2")
		}
		if result.Temperature == nil || *result.Temperature != 0.5 {
			t.Errorf("Temperature = %v, want 0.5", result.Temperature)
		}
		if result.TopP != 0.8 {
			t.Errorf("TopP = %f, want 0.8", result.TopP)
		}
		if result.TopK != 40 {
			t.Errorf("TopK = %d, want 40", result.TopK)
		}
	})
}

// ---------------------------------------------------------------------------
// StreamResponseClaude2OpenAI
// ---------------------------------------------------------------------------

func TestStreamResponseClaude2OpenAI(t *testing.T) {
	t.Run("message_start extracts id and model", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "message_start",
			Message: &dto.ClaudeMediaMessage{
				Id:    "msg_123",
				Model: "claude-3-opus-20240229",
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if resp.Id != "msg_123" {
			t.Errorf("Id = %q, want %q", resp.Id, "msg_123")
		}
		if resp.Model != "claude-3-opus-20240229" {
			t.Errorf("Model = %q, want %q", resp.Model, "claude-3-opus-20240229")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].Delta.Role != "assistant" {
			t.Errorf("Delta.Role = %q, want %q", resp.Choices[0].Delta.Role, "assistant")
		}
	})

	t.Run("content_block_start text", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "content_block_start",
			ContentBlock: &dto.ClaudeMediaMessage{
				Type: "text",
				Text: common.GetPointer("Hello"),
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].Delta.GetContentString() != "Hello" {
			t.Errorf("content = %q, want %q", resp.Choices[0].Delta.GetContentString(), "Hello")
		}
	})

	t.Run("content_block_start tool_use", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "content_block_start",
			ContentBlock: &dto.ClaudeMediaMessage{
				Type: "tool_use",
				Id:   "toolu_abc",
				Name: "get_weather",
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		tools := resp.Choices[0].Delta.ToolCalls
		if len(tools) != 1 {
			t.Fatalf("len(ToolCalls) = %d, want 1", len(tools))
		}
		if tools[0].ID != "toolu_abc" {
			t.Errorf("ToolCall.ID = %q, want %q", tools[0].ID, "toolu_abc")
		}
		if tools[0].Type != "function" {
			t.Errorf("ToolCall.Type = %q, want %q", tools[0].Type, "function")
		}
		if tools[0].Function.Name != "get_weather" {
			t.Errorf("ToolCall.Function.Name = %q, want %q", tools[0].Function.Name, "get_weather")
		}
	})

	t.Run("content_block_start nil ContentBlock returns nil", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type:         "content_block_start",
			ContentBlock: nil,
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)
		if resp != nil {
			t.Errorf("expected nil response for nil ContentBlock, got %+v", resp)
		}
	})

	t.Run("content_block_delta text", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "content_block_delta",
			Delta: &dto.ClaudeMediaMessage{
				Type: "text_delta",
				Text: common.GetPointer(" world"),
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].Delta.Content == nil {
			t.Fatal("Delta.Content should not be nil")
		}
		if *resp.Choices[0].Delta.Content != " world" {
			t.Errorf("content = %q, want %q", *resp.Choices[0].Delta.Content, " world")
		}
	})

	t.Run("content_block_delta thinking", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "content_block_delta",
			Delta: &dto.ClaudeMediaMessage{
				Type:     "thinking_delta",
				Thinking: common.GetPointer("Let me think..."),
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].Delta.ReasoningContent == nil {
			t.Fatal("ReasoningContent should not be nil")
		}
		if *resp.Choices[0].Delta.ReasoningContent != "Let me think..." {
			t.Errorf("reasoning = %q, want %q", *resp.Choices[0].Delta.ReasoningContent, "Let me think...")
		}
	})

	t.Run("content_block_delta input_json_delta", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "content_block_delta",
			Delta: &dto.ClaudeMediaMessage{
				Type:        "input_json_delta",
				PartialJson: common.GetPointer(`{"city":`),
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		tools := resp.Choices[0].Delta.ToolCalls
		if len(tools) != 1 {
			t.Fatalf("len(ToolCalls) = %d, want 1", len(tools))
		}
		if tools[0].Function.Arguments != `{"city":` {
			t.Errorf("Arguments = %q, want %q", tools[0].Function.Arguments, `{"city":`)
		}
	})

	t.Run("message_delta stop reason", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Type: "message_delta",
			Delta: &dto.ClaudeMediaMessage{
				StopReason: common.GetPointer("end_turn"),
			},
		}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
		}
		if resp.Choices[0].FinishReason == nil {
			t.Fatal("FinishReason should not be nil")
		}
		if *resp.Choices[0].FinishReason != "stop" {
			t.Errorf("FinishReason = %q, want %q", *resp.Choices[0].FinishReason, "stop")
		}
	})

	t.Run("message_stop returns nil", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{Type: "message_stop"}
		resp := StreamResponseClaude2OpenAI(RequestModeMessage, claudeResp)
		if resp != nil {
			t.Errorf("expected nil for message_stop, got %+v", resp)
		}
	})

	t.Run("completion mode", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Completion: "Hello there",
			StopReason: "stop_sequence",
		}
		resp := StreamResponseClaude2OpenAI(RequestModeCompletion, claudeResp)

		if resp == nil {
			t.Fatal("response should not be nil")
		}
		if resp.Choices[0].Delta.GetContentString() != "Hello there" {
			t.Errorf("content = %q, want %q", resp.Choices[0].Delta.GetContentString(), "Hello there")
		}
		if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "stop" {
			t.Errorf("FinishReason = %v, want %q", resp.Choices[0].FinishReason, "stop")
		}
	})
}

// ---------------------------------------------------------------------------
// ResponseClaude2OpenAI
// ---------------------------------------------------------------------------

func TestResponseClaude2OpenAI(t *testing.T) {
	t.Run("completion mode trims prefix", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Completion: " Hello world",
			StopReason: "stop_sequence",
			Model:      "claude-2.1",
		}
		resp := ResponseClaude2OpenAI(RequestModeCompletion, claudeResp)

		if resp.Model != "claude-2.1" {
			t.Errorf("Model = %q, want %q", resp.Model, "claude-2.1")
		}
		if resp.Object != "chat.completion" {
			t.Errorf("Object = %q, want %q", resp.Object, "chat.completion")
		}
		// In completion mode, there are 2 choices (the completion one + the final one)
		// but all should have "assistant" role
		found := false
		for _, choice := range resp.Choices {
			if choice.Message.Role == "assistant" {
				found = true
			}
		}
		if !found {
			t.Error("expected at least one choice with assistant role")
		}
	})

	t.Run("message mode with text content", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Id: "msg_456",
			Content: []dto.ClaudeMediaMessage{
				{Type: "text", Text: common.GetPointer("The answer is 42")},
			},
			StopReason: "end_turn",
			Model:      "claude-3-opus-20240229",
		}
		resp := ResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if resp.Id != "msg_456" {
			t.Errorf("Id = %q, want %q", resp.Id, "msg_456")
		}
		if resp.Model != "claude-3-opus-20240229" {
			t.Errorf("Model = %q, want %q", resp.Model, "claude-3-opus-20240229")
		}
		if len(resp.Choices) == 0 {
			t.Fatal("expected at least one choice")
		}
	})

	t.Run("message mode with tool_use", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Id: "msg_789",
			Content: []dto.ClaudeMediaMessage{
				{Type: "text", Text: common.GetPointer("")},
				{
					Type:  "tool_use",
					Id:    "toolu_abc",
					Name:  "get_weather",
					Input: map[string]interface{}{"city": "NYC"},
				},
			},
			StopReason: "tool_use",
			Model:      "claude-3-opus-20240229",
		}
		resp := ResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if len(resp.Choices) == 0 {
			t.Fatal("expected at least one choice")
		}
		lastChoice := resp.Choices[len(resp.Choices)-1]
		if lastChoice.FinishReason != "tool_calls" {
			t.Errorf("FinishReason = %q, want %q", lastChoice.FinishReason, "tool_calls")
		}
	})

	t.Run("message mode with thinking block", func(t *testing.T) {
		claudeResp := &dto.ClaudeResponse{
			Id: "msg_think",
			Content: []dto.ClaudeMediaMessage{
				{Type: "thinking", Thinking: common.GetPointer("Let me analyze...")},
				{Type: "text", Text: common.GetPointer("The answer is 42")},
			},
			StopReason: "end_turn",
			Model:      "claude-3-7-sonnet-20250219",
		}
		resp := ResponseClaude2OpenAI(RequestModeMessage, claudeResp)

		if len(resp.Choices) == 0 {
			t.Fatal("expected at least one choice")
		}
		lastChoice := resp.Choices[len(resp.Choices)-1]
		if lastChoice.Message.ReasoningContent != "Let me analyze..." {
			t.Errorf("ReasoningContent = %q, want %q", lastChoice.Message.ReasoningContent, "Let me analyze...")
		}
	})
}

// ---------------------------------------------------------------------------
// mapToolChoice
// ---------------------------------------------------------------------------

func TestMapToolChoice(t *testing.T) {
	t.Run("auto string", func(t *testing.T) {
		result := mapToolChoice("auto", nil)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "auto" {
			t.Errorf("Type = %q, want %q", result.Type, "auto")
		}
	})

	t.Run("required maps to any", func(t *testing.T) {
		result := mapToolChoice("required", nil)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "any" {
			t.Errorf("Type = %q, want %q", result.Type, "any")
		}
	})

	t.Run("none string", func(t *testing.T) {
		result := mapToolChoice("none", nil)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "none" {
			t.Errorf("Type = %q, want %q", result.Type, "none")
		}
	})

	t.Run("object with function name", func(t *testing.T) {
		choice := map[string]interface{}{
			"function": map[string]interface{}{
				"name": "get_weather",
			},
		}
		result := mapToolChoice(choice, nil)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool" {
			t.Errorf("Type = %q, want %q", result.Type, "tool")
		}
		if result.Name != "get_weather" {
			t.Errorf("Name = %q, want %q", result.Name, "get_weather")
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		result := mapToolChoice(nil, nil)
		if result != nil {
			t.Errorf("expected nil for nil toolChoice, got %+v", result)
		}
	})

	t.Run("parallel tool calls true", func(t *testing.T) {
		parallel := true
		result := mapToolChoice("auto", &parallel)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DisableParallelToolUse != false {
			t.Error("DisableParallelToolUse should be false when parallel=true")
		}
	})

	t.Run("parallel tool calls false", func(t *testing.T) {
		parallel := false
		result := mapToolChoice("auto", &parallel)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DisableParallelToolUse != true {
			t.Error("DisableParallelToolUse should be true when parallel=false")
		}
	})

	t.Run("parallel tool calls without tool choice creates auto default", func(t *testing.T) {
		parallel := true
		result := mapToolChoice(nil, &parallel)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "auto" {
			t.Errorf("Type = %q, want %q", result.Type, "auto")
		}
	})
}
