package openrouter_pool

import (
	"strconv"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
	"github.com/LurusTech/lurus-hub/internal/pkg/types"
)

// MaybeMarkCooldown is the relay-side hook for OpenRouter rate-limited keys.
// Called from processChannelError; no-op unless:
//   - channel type is OpenRouter
//   - the channel is in multi-key mode (has a key pool)
//   - the upstream returned 429
//
// All other error paths (401/billing/etc.) keep their existing AutoBan behavior.
func MaybeMarkCooldown(channelErr types.ChannelError, apiErr *types.NewAPIError) {
	if apiErr == nil {
		return
	}
	if channelErr.ChannelType != constant.ChannelTypeOpenRouter {
		return
	}
	if !channelErr.IsMultiKey {
		return
	}
	if apiErr.StatusCode != 429 {
		return
	}
	if channelErr.UsingKey == "" {
		return
	}

	until := ParseCooldownUntil(apiErr.UpstreamHeader, []byte(apiErr.UpstreamBodyHint), time.Now())
	if until <= 0 {
		return
	}
	if !repo.MarkMultiKeyCooldown(channelErr.ChannelId, channelErr.UsingKey, until, "rate limit (auto cooldown)") {
		return
	}
	common.SysLog("openrouter pool: marked key cooldown on channel " +
		strconv.Itoa(channelErr.ChannelId) +
		" until " + time.Unix(until, 0).UTC().Format(time.RFC3339))
}
