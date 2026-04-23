package provider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/LurusTech/lurus-api/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-api/internal/pkg/dto"

	"github.com/gin-gonic/gin"
)

// Compile-time interface compliance check
var _ Adaptor = (*BaseAdaptor)(nil)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestBaseAdaptor_Init(t *testing.T) {
	t.Run("no panic on nil info", func(t *testing.T) {
		a := &BaseAdaptor{}
		a.Init(nil)
	})

	t.Run("no panic on valid info", func(t *testing.T) {
		a := &BaseAdaptor{}
		a.Init(&relaycommon.RelayInfo{})
	})
}

func TestBaseAdaptor_GetRequestURL(t *testing.T) {
	a := &BaseAdaptor{}
	url, err := a.GetRequestURL(&relaycommon.RelayInfo{})
	if url != "" {
		t.Errorf("GetRequestURL() url = %q, want empty", url)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("GetRequestURL() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_SetupRequestHeader(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h := make(http.Header)
	err := a.SetupRequestHeader(c, &h, &relaycommon.RelayInfo{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("SetupRequestHeader() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertOpenAIRequest(c, &relaycommon.RelayInfo{}, &dto.GeneralOpenAIRequest{})
	if result != nil {
		t.Errorf("ConvertOpenAIRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertOpenAIRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertRerankRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertRerankRequest(c, 0, dto.RerankRequest{})
	if result != nil {
		t.Errorf("ConvertRerankRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertRerankRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertEmbeddingRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertEmbeddingRequest(c, &relaycommon.RelayInfo{}, dto.EmbeddingRequest{})
	if result != nil {
		t.Errorf("ConvertEmbeddingRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertEmbeddingRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertAudioRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertAudioRequest(c, &relaycommon.RelayInfo{}, dto.AudioRequest{})
	if result != nil {
		t.Errorf("ConvertAudioRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertAudioRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertImageRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertImageRequest(c, &relaycommon.RelayInfo{}, dto.ImageRequest{})
	if result != nil {
		t.Errorf("ConvertImageRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertImageRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertOpenAIResponsesRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertOpenAIResponsesRequest(c, &relaycommon.RelayInfo{}, dto.OpenAIResponsesRequest{})
	if result != nil {
		t.Errorf("ConvertOpenAIResponsesRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertOpenAIResponsesRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_DoRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.DoRequest(c, &relaycommon.RelayInfo{}, nil)
	if result != nil {
		t.Errorf("DoRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("DoRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_DoResponse(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	usage, apiErr := a.DoResponse(c, nil, &relaycommon.RelayInfo{})
	if usage != nil {
		t.Errorf("DoResponse() usage = %v, want nil", usage)
	}
	if apiErr == nil {
		t.Fatal("DoResponse() should return non-nil error")
	}
	if apiErr.StatusCode != http.StatusNotImplemented {
		t.Errorf("DoResponse() statusCode = %d, want %d", apiErr.StatusCode, http.StatusNotImplemented)
	}
}

func TestBaseAdaptor_GetModelList(t *testing.T) {
	a := &BaseAdaptor{}
	result := a.GetModelList()
	if result != nil {
		t.Errorf("GetModelList() = %v, want nil", result)
	}
}

func TestBaseAdaptor_GetChannelName(t *testing.T) {
	a := &BaseAdaptor{}
	result := a.GetChannelName()
	if result != "" {
		t.Errorf("GetChannelName() = %q, want empty", result)
	}
}

func TestBaseAdaptor_ConvertClaudeRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertClaudeRequest(c, &relaycommon.RelayInfo{}, &dto.ClaudeRequest{})
	if result != nil {
		t.Errorf("ConvertClaudeRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertClaudeRequest() err = %v, want ErrNotImplemented", err)
	}
}

func TestBaseAdaptor_ConvertGeminiRequest(t *testing.T) {
	a := &BaseAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	result, err := a.ConvertGeminiRequest(c, &relaycommon.RelayInfo{}, &dto.GeminiChatRequest{})
	if result != nil {
		t.Errorf("ConvertGeminiRequest() result = %v, want nil", result)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ConvertGeminiRequest() err = %v, want ErrNotImplemented", err)
	}
}
