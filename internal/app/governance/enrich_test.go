package governance

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/lurus-api/internal/domain/entity"
	"github.com/QuantumNous/lurus-api/internal/pkg/constant"
	"github.com/QuantumNous/lurus-api/internal/pkg/dto"
	relaycommon "github.com/QuantumNous/lurus-api/internal/adapter/provider/common"

	"github.com/gin-gonic/gin"
)

func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	return c
}

func TestEnrichLogParams_NilSafety(t *testing.T) {
	c := newTestContext()
	// Should not panic with nil info or params.
	EnrichLogParams(c, nil, &entity.RecordConsumeLogParams{})
	EnrichLogParams(c, &relaycommon.RelayInfo{}, nil)
	EnrichLogParams(nil, nil, nil)
}

func TestEnrichLogParams_ChannelTypeFromMeta(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
		},
		StartTime: time.Now().Add(-100 * time.Millisecond),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.ChannelType != constant.ChannelTypeOpenAI {
		t.Errorf("expected ChannelType=%d, got %d", constant.ChannelTypeOpenAI, params.ChannelType)
	}
}

func TestEnrichLogParams_ChannelTypeFallbackToContext(t *testing.T) {
	c := newTestContext()
	c.Set("channel_type", constant.ChannelTypeAnthropic)
	info := &relaycommon.RelayInfo{
		ChannelMeta: nil, // no meta
		StartTime:   time.Now(),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.ChannelType != constant.ChannelTypeAnthropic {
		t.Errorf("expected ChannelType=%d from context, got %d", constant.ChannelTypeAnthropic, params.ChannelType)
	}
}

func TestEnrichLogParams_RelayMode(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		RelayMode: 42,
		StartTime: time.Now(),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.RelayMode != 42 {
		t.Errorf("expected RelayMode=42, got %d", params.RelayMode)
	}
}

func TestEnrichLogParams_FingerprintFromContext(t *testing.T) {
	c := newTestContext()
	c.Set(ctxKeyFingerprint, "abcdef1234567890")
	info := &relaycommon.RelayInfo{StartTime: time.Now()}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.RequestFingerprint != "abcdef1234567890" {
		t.Errorf("expected fingerprint from context, got %q", params.RequestFingerprint)
	}
}

func TestEnrichLogParams_FingerprintEmpty(t *testing.T) {
	c := newTestContext()
	// No fingerprint in context.
	info := &relaycommon.RelayInfo{StartTime: time.Now()}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.RequestFingerprint != "" {
		t.Errorf("expected empty fingerprint, got %q", params.RequestFingerprint)
	}
}

func TestEnrichLogParams_UpstreamModelFromMeta(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4o-2024-08-06",
		},
		StartTime: time.Now(),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.UpstreamModel != "gpt-4o-2024-08-06" {
		t.Errorf("expected upstream from meta, got %q", params.UpstreamModel)
	}
}

func TestEnrichLogParams_UpstreamModelFallback(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o-mini",
		ChannelMeta:     nil,
		StartTime:       time.Now(),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.UpstreamModel != "gpt-4o-mini" {
		t.Errorf("expected upstream fallback to OriginModelName, got %q", params.UpstreamModel)
	}
}

func TestEnrichLogParams_TotalLatencyMs(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		StartTime: time.Now().Add(-200 * time.Millisecond),
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.TotalLatencyMs < 150 || params.TotalLatencyMs > 500 {
		t.Errorf("expected latency ~200ms, got %dms", params.TotalLatencyMs)
	}
}

func TestEnrichLogParams_LogDetailLevel(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		StartTime: time.Now(),
		UserSetting: dto.UserSetting{
			LogDetailLevel: "none",
		},
	}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.LogDetailLevel != "none" {
		t.Errorf("expected LogDetailLevel=none, got %q", params.LogDetailLevel)
	}
}

func TestEnrichLogParams_DataFlowMetadata(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
		},
		StartTime: time.Now(),
	}
	params := &entity.RecordConsumeLogParams{
		TokenName: "test-token",
		Other:     make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if params.Other["data_flow_source"] != "test-token" {
		t.Errorf("expected data_flow_source=test-token, got %v", params.Other["data_flow_source"])
	}
	dest := params.Other["data_flow_dest"]
	if dest == nil || dest == "" {
		t.Error("expected data_flow_dest to be populated")
	}
}

func TestEnrichLogParams_NilOtherMapInitialized(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{StartTime: time.Now()}
	params := &entity.RecordConsumeLogParams{
		Other: nil, // nil map
	}
	EnrichLogParams(c, info, params)

	if params.Other == nil {
		t.Error("expected Other map to be initialized")
	}
}

func TestEnrichLogParams_NoClientIPInOther(t *testing.T) {
	c := newTestContext()
	info := &relaycommon.RelayInfo{StartTime: time.Now()}
	params := &entity.RecordConsumeLogParams{
		Other: make(map[string]interface{}),
	}
	EnrichLogParams(c, info, params)

	if _, exists := params.Other["client_ip"]; exists {
		t.Error("client_ip should NOT be in Other map (privacy: controlled by RecordIpLog setting)")
	}
}
