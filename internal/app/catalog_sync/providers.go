package catalog_sync

import (
	"strings"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
)

// ProviderForChannel returns a ListModels implementation for the channel type.
func ProviderForChannel(ch *repo.Channel) (Provider, error) {
	base := ""
	if ch.BaseURL != nil {
		base = strings.TrimSpace(*ch.BaseURL)
	}
	key := ""
	if keys := ch.GetKeys(); len(keys) > 0 {
		key = keys[0]
	} else {
		key = ch.Key
	}

	switch ch.Type {
	case constant.ChannelTypeVolcEngine:
		if base == "" {
			base = constant.ChannelBaseURLs[constant.ChannelTypeVolcEngine]
		}
		return &VolcengineProvider{BaseURL: base, APIKey: key}, nil
	case constant.ChannelTypeOpenRouter:
		if base == "" {
			base = constant.ChannelBaseURLs[constant.ChannelTypeOpenRouter]
		}
		return &OpenAICompatibleProvider{Label: "openrouter", BaseURL: base, APIKey: key}, nil
	case constant.ChannelTypeDeepSeek:
		if base == "" {
			base = constant.ChannelBaseURLs[constant.ChannelTypeDeepSeek]
		}
		return &OpenAICompatibleProvider{Label: "deepseek", BaseURL: base, APIKey: key}, nil
	case constant.ChannelTypeZhipu_v4:
		if base == "" {
			base = constant.ChannelBaseURLs[constant.ChannelTypeZhipu_v4]
		}
		return &OpenAICompatibleProvider{Label: "glm", BaseURL: base, APIKey: key}, nil
	case constant.ChannelTypeOpenAI:
		if base == "" {
			base = constant.ChannelBaseURLs[constant.ChannelTypeOpenAI]
		}
		return &OpenAICompatibleProvider{Label: "openai_compatible", BaseURL: base, APIKey: key}, nil
	default:
		return nil, ErrUnsupportedChannelType
	}
}
