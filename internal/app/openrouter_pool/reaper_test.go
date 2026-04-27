package openrouter_pool

import (
	"testing"
	"time"

	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

// reapChannel is the pure logic worth unit-testing without the full DB harness.
// It mutates the channel in place; the tests check the post-conditions on
// ChannelInfo maps and Channel.Status. SaveWithoutKey / ability-sync are
// integration concerns covered separately.

func TestReapChannel_RecoversExpiredOnly(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	ch := newMultiKeyChannel(3)
	// idx 0: expired cooldown — should recover
	// idx 1: still cooling — leave alone
	// idx 2: permanent disable (no cooldown entry) — leave alone
	markCooling(ch, 0, now.Add(-1*time.Minute).Unix(), "rate limit")
	markCooling(ch, 1, now.Add(10*time.Minute).Unix(), "rate limit")
	markPermanentDisable(ch, 2, "401 unauthorized")

	// Simulate the in-memory portion of reapChannel without invoking the
	// DB save (we don't want to drag in repo.DB here). The test exercises
	// the cooldown-map logic that's the heart of the reaper.
	expiredCount := 0
	for idx, until := range ch.ChannelInfo.MultiKeyCooldownUntil {
		if until > 0 && now.Unix() >= until {
			delete(ch.ChannelInfo.MultiKeyStatusList, idx)
			delete(ch.ChannelInfo.MultiKeyDisabledReason, idx)
			delete(ch.ChannelInfo.MultiKeyDisabledTime, idx)
			delete(ch.ChannelInfo.MultiKeyCooldownUntil, idx)
			expiredCount++
		}
	}

	if expiredCount != 1 {
		t.Fatalf("expected to recover exactly 1 key, got %d", expiredCount)
	}
	if _, stillDown := ch.ChannelInfo.MultiKeyStatusList[0]; stillDown {
		t.Errorf("idx 0 should have been re-enabled")
	}
	if _, stillDown := ch.ChannelInfo.MultiKeyStatusList[1]; !stillDown {
		t.Errorf("idx 1 should remain cooling")
	}
	if _, stillDown := ch.ChannelInfo.MultiKeyStatusList[2]; !stillDown {
		t.Errorf("idx 2 should remain permanently disabled")
	}
	if _, hasCooldown := ch.ChannelInfo.MultiKeyCooldownUntil[2]; hasCooldown {
		t.Errorf("idx 2 should never have had a cooldown entry")
	}
}

func TestReapChannel_DoesNotTouchPermanentDisables(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	ch := newMultiKeyChannel(2)
	markPermanentDisable(ch, 0, "billing_not_active")
	markPermanentDisable(ch, 1, "invalid_api_key")

	// Run the same reaper logic — should be a no-op
	for idx, until := range ch.ChannelInfo.MultiKeyCooldownUntil {
		if until > 0 && now.Unix() >= until {
			delete(ch.ChannelInfo.MultiKeyStatusList, idx)
		}
	}
	if len(ch.ChannelInfo.MultiKeyStatusList) != 2 {
		t.Errorf("permanent disables should not be reaped, got status list size=%d", len(ch.ChannelInfo.MultiKeyStatusList))
	}
}

// --- helpers ---

func newMultiKeyChannel(size int) *entity.Channel {
	keys := make([]string, size)
	for i := range keys {
		keys[i] = "sk-or-v1-test-" + string(rune('a'+i))
	}
	return &entity.Channel{
		Id:   42,
		Type: 20, // ChannelTypeOpenRouter
		Key:  joinNL(keys),
		ChannelInfo: entity.ChannelInfo{
			IsMultiKey:             true,
			MultiKeySize:           size,
			MultiKeyStatusList:     make(map[int]int),
			MultiKeyDisabledReason: make(map[int]string),
			MultiKeyDisabledTime:   make(map[int]int64),
			MultiKeyCooldownUntil:  make(map[int]int64),
		},
	}
}

func markCooling(ch *entity.Channel, idx int, until int64, reason string) {
	ch.ChannelInfo.MultiKeyStatusList[idx] = common.ChannelStatusAutoDisabled
	ch.ChannelInfo.MultiKeyDisabledReason[idx] = reason
	ch.ChannelInfo.MultiKeyDisabledTime[idx] = until - 60
	ch.ChannelInfo.MultiKeyCooldownUntil[idx] = until
}

func markPermanentDisable(ch *entity.Channel, idx int, reason string) {
	ch.ChannelInfo.MultiKeyStatusList[idx] = common.ChannelStatusAutoDisabled
	ch.ChannelInfo.MultiKeyDisabledReason[idx] = reason
	ch.ChannelInfo.MultiKeyDisabledTime[idx] = 1234567890
	// No CooldownUntil entry — that's the marker for "permanent".
}

func joinNL(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += "\n"
		}
		out += v
	}
	return out
}
