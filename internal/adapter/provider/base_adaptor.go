package provider

import (
	"errors"
	"io"
	"net/http"

	"github.com/LurusTech/lurus-hub/internal/pkg/dto"
	relaycommon "github.com/LurusTech/lurus-hub/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/types"

	"github.com/gin-gonic/gin"
)

// ErrNotImplemented is the standard error returned by unimplemented adaptor methods.
var ErrNotImplemented = errors.New("not implemented")

// BaseAdaptor provides default implementations for all Adaptor interface methods.
// Adaptors can embed this struct and only override the methods they actually support.
// This eliminates boilerplate stub code and prevents panic in unimplemented methods.
type BaseAdaptor struct{}

func (a *BaseAdaptor) Init(info *relaycommon.RelayInfo) {}

func (a *BaseAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return "", ErrNotImplemented
}

func (a *BaseAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	return ErrNotImplemented
}

func (a *BaseAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	return nil, types.NewErrorWithStatusCode(ErrNotImplemented, "not_implemented", http.StatusNotImplemented)
}

func (a *BaseAdaptor) GetModelList() []string {
	return nil
}

func (a *BaseAdaptor) GetChannelName() string {
	return ""
}

func (a *BaseAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, ErrNotImplemented
}

func (a *BaseAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, ErrNotImplemented
}
