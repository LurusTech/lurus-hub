package openai

import (
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/LurusTech/lurus-hub/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
	"github.com/LurusTech/lurus-hub/internal/pkg/dto"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// parseReasoningEffortFromModelSuffix
// ---------------------------------------------------------------------------

func TestParseReasoningEffortFromModelSuffix(t *testing.T) {
	tests := []struct {
		model      string
		wantEffort string
		wantModel  string
	}{
		{"o3-mini-high", "high", "o3-mini"},
		{"o3-mini-low", "low", "o3-mini"},
		{"o3-mini-medium", "medium", "o3-mini"},
		{"o4-mini-minimal", "minimal", "o4-mini"},
		{"o1-none", "none", "o1"},
		{"o3-xhigh", "xhigh", "o3"},
		{"gpt-5-high", "high", "gpt-5"},
		// no suffix
		{"o3-mini", "", "o3-mini"},
		{"gpt-4o", "", "gpt-4o"},
		{"claude-3-opus", "", "claude-3-opus"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			effort, model := parseReasoningEffortFromModelSuffix(tt.model)
			if effort != tt.wantEffort {
				t.Errorf("effort = %q, want %q", effort, tt.wantEffort)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ProcessStreamResponse
// ---------------------------------------------------------------------------

func TestProcessStreamResponse(t *testing.T) {
	t.Run("accumulates content", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						Content: common.GetPointer("hello "),
					},
				},
			},
		}
		var builder strings.Builder
		toolCount := 0
		err := ProcessStreamResponse(resp, &builder, &toolCount)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if builder.String() != "hello " {
			t.Errorf("content = %q, want %q", builder.String(), "hello ")
		}
		if toolCount != 0 {
			t.Errorf("toolCount = %d, want 0", toolCount)
		}
	})

	t.Run("accumulates reasoning content", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						ReasoningContent: common.GetPointer("thinking..."),
					},
				},
			},
		}
		var builder strings.Builder
		toolCount := 0
		_ = ProcessStreamResponse(resp, &builder, &toolCount)
		if builder.String() != "thinking..." {
			t.Errorf("content = %q, want %q", builder.String(), "thinking...")
		}
	})

	t.Run("tool call counting and accumulation", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						ToolCalls: []dto.ToolCallResponse{
							{
								Function: dto.FunctionResponse{
									Name:      "get_weather",
									Arguments: `{"city":"NYC"}`,
								},
							},
						},
					},
				},
			},
		}
		var builder strings.Builder
		toolCount := 0
		_ = ProcessStreamResponse(resp, &builder, &toolCount)
		if toolCount != 1 {
			t.Errorf("toolCount = %d, want 1", toolCount)
		}
		got := builder.String()
		if !strings.Contains(got, "get_weather") {
			t.Errorf("content should contain tool name, got %q", got)
		}
		if !strings.Contains(got, `{"city":"NYC"}`) {
			t.Errorf("content should contain tool arguments, got %q", got)
		}
	})

	t.Run("multiple tool calls updates count", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						ToolCalls: []dto.ToolCallResponse{
							{Function: dto.FunctionResponse{Name: "a"}},
							{Function: dto.FunctionResponse{Name: "b"}},
							{Function: dto.FunctionResponse{Name: "c"}},
						},
					},
				},
			},
		}
		var builder strings.Builder
		toolCount := 0
		_ = ProcessStreamResponse(resp, &builder, &toolCount)
		if toolCount != 3 {
			t.Errorf("toolCount = %d, want 3", toolCount)
		}
	})

	t.Run("empty choices no-op", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{},
		}
		var builder strings.Builder
		toolCount := 0
		_ = ProcessStreamResponse(resp, &builder, &toolCount)
		if builder.Len() != 0 {
			t.Errorf("builder should be empty, got %q", builder.String())
		}
		if toolCount != 0 {
			t.Errorf("toolCount = %d, want 0", toolCount)
		}
	})

	t.Run("content and reasoning combined", func(t *testing.T) {
		resp := dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						Content:          common.GetPointer("answer"),
						ReasoningContent: common.GetPointer("reason"),
					},
				},
			},
		}
		var builder strings.Builder
		toolCount := 0
		_ = ProcessStreamResponse(resp, &builder, &toolCount)
		if builder.String() != "answerreason" {
			t.Errorf("content = %q, want %q", builder.String(), "answerreason")
		}
	})
}

// ---------------------------------------------------------------------------
// Adaptor.ConvertOpenAIRequest
// ---------------------------------------------------------------------------

func TestAdaptor_ConvertOpenAIRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	t.Run("nil request returns error", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		_, err := a.ConvertOpenAIRequest(c, &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
		}, nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Errorf("error = %q, want contains 'nil'", err.Error())
		}
	})

	t.Run("o-series MaxTokens to MaxCompletionTokens rewrite", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "o3-mini",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model:     "o3-mini",
			MaxTokens: 1000,
			Messages:  []dto.Message{{Role: "user", Content: "Hi"}},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.MaxTokens != 0 {
			t.Errorf("MaxTokens = %d, want 0", r.MaxTokens)
		}
		if r.MaxCompletionTokens != 1000 {
			t.Errorf("MaxCompletionTokens = %d, want 1000", r.MaxCompletionTokens)
		}
	})

	t.Run("o-series temperature cleared", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		temp := 0.7
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "o3",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model:       "o3",
			Temperature: &temp,
			Messages:    []dto.Message{{Role: "user", Content: "Hi"}},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.Temperature != nil {
			t.Errorf("Temperature = %v, want nil", r.Temperature)
		}
	})

	t.Run("gpt-5 parameter strip", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		temp := 0.5
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "gpt-5",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model:       "gpt-5",
			Temperature: &temp,
			TopP:        0.9,
			LogProbs:    true,
			MaxTokens:   500,
			Messages:    []dto.Message{{Role: "user", Content: "Hi"}},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.Temperature != nil {
			t.Errorf("Temperature = %v, want nil", r.Temperature)
		}
		if r.TopP != 0 {
			t.Errorf("TopP = %f, want 0", r.TopP)
		}
		if r.LogProbs != false {
			t.Error("LogProbs should be false")
		}
		// MaxTokens should move to MaxCompletionTokens
		if r.MaxCompletionTokens != 500 {
			t.Errorf("MaxCompletionTokens = %d, want 500", r.MaxCompletionTokens)
		}
	})

	t.Run("o-series system to developer rewrite", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "o3",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model: "o3",
			Messages: []dto.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hi"},
			},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.Messages[0].Role != "developer" {
			t.Errorf("Messages[0].Role = %q, want %q", r.Messages[0].Role, "developer")
		}
	})

	t.Run("o1-mini preserves system role", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "o1-mini",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model: "o1-mini",
			Messages: []dto.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hi"},
			},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.Messages[0].Role != "system" {
			t.Errorf("Messages[0].Role = %q, want %q (o1-mini should preserve system)", r.Messages[0].Role, "system")
		}
	})

	t.Run("model suffix effort extraction", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "o3-mini-high",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model:    "o3-mini-high",
			Messages: []dto.Message{{Role: "user", Content: "Hi"}},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		if r.ReasoningEffort != "high" {
			t.Errorf("ReasoningEffort = %q, want %q", r.ReasoningEffort, "high")
		}
		if r.Model != "o3-mini" {
			t.Errorf("Model = %q, want %q", r.Model, "o3-mini")
		}
		if info.UpstreamModelName != "o3-mini" {
			t.Errorf("info.UpstreamModelName = %q, want %q", info.UpstreamModelName, "o3-mini")
		}
	})

	t.Run("non-o non-gpt5 model passthrough", func(t *testing.T) {
		a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
		temp := 0.7
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "gpt-4o",
			},
		}
		req := &dto.GeneralOpenAIRequest{
			Model:       "gpt-4o",
			Temperature: &temp,
			TopP:        0.9,
			Messages:    []dto.Message{{Role: "system", Content: "Be helpful"}, {Role: "user", Content: "Hi"}},
		}
		result, err := a.ConvertOpenAIRequest(c, info, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result.(*dto.GeneralOpenAIRequest)
		// system role should NOT be changed for non-o models
		if r.Messages[0].Role != "system" {
			t.Errorf("Messages[0].Role = %q, want %q", r.Messages[0].Role, "system")
		}
		if r.Temperature == nil || *r.Temperature != 0.7 {
			t.Errorf("Temperature should be 0.7, got %v", r.Temperature)
		}
	})
}

// ---------------------------------------------------------------------------
// Adaptor.GetChannelName / GetModelList / Init
// ---------------------------------------------------------------------------

func TestAdaptor_GetChannelName(t *testing.T) {
	a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
	name := a.GetChannelName()
	if name != "openai" {
		t.Errorf("GetChannelName() = %q, want %q", name, "openai")
	}
}

func TestAdaptor_GetModelList(t *testing.T) {
	a := &Adaptor{ChannelType: constant.ChannelTypeOpenAI}
	list := a.GetModelList()
	if list == nil {
		t.Fatal("GetModelList() should not be nil")
	}
	if len(list) == 0 {
		t.Error("GetModelList() should return non-empty list")
	}
}

func TestAdaptor_Init(t *testing.T) {
	t.Run("sets channel type from info", func(t *testing.T) {
		a := &Adaptor{}
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
		}
		a.Init(info)
		if a.ChannelType != constant.ChannelTypeOpenAI {
			t.Errorf("ChannelType = %d, want %d", a.ChannelType, constant.ChannelTypeOpenAI)
		}
	})
}
