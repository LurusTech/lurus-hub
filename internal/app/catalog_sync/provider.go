package catalog_sync

import (
	"context"

	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
)

// UpstreamModel is one model ID discovered from a vendor Models API.
type UpstreamModel struct {
	ID string
}

// Provider lists models for one channel binding (base URL + key + type).
type Provider interface {
	Name() string
	ListModels(ctx context.Context) ([]UpstreamModel, error)
}

// SupportedChannelTypes are the inventory sources CatalogSync understands.
func SupportedChannelTypes() map[int]bool {
	return map[int]bool{
		constant.ChannelTypeOpenAI:     true, // OpenAI-compatible incl. Grok2API
		constant.ChannelTypeOpenRouter: true,
		constant.ChannelTypeDeepSeek:   true,
		constant.ChannelTypeZhipu_v4:   true,
		constant.ChannelTypeVolcEngine: true,
	}
}

// PriceHint is an optional upstream ratio/price signal for proposals only.
type PriceHint struct {
	ModelName string
	Field     string // model_ratio | completion_ratio | model_price
	Value     float64
	Source    string
}
