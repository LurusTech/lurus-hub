package relay

// used_quota and channel used_quota used to be written before SettleConsume,
// so a failed settlement still counted the request as spent.

import (
	"errors"
	"net/http"
	"testing"

	relaycommon "github.com/LurusTech/lurus-hub/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/app"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
	"github.com/LurusTech/lurus-hub/internal/pkg/dto"
	"github.com/LurusTech/lurus-hub/internal/pkg/types"
)

func TestPostConsumeQuota_SettlementErrorSkipsUsedQuota(t *testing.T) {
	cleanup := setupRelayDB(t)
	defer cleanup()

	prevFn := app.PostConsumeQuotaFn
	app.PostConsumeQuotaFn = func(*relaycommon.RelayInfo, int, int, bool) error {
		return errors.New("settle: platform unreachable")
	}
	t.Cleanup(func() { app.PostConsumeQuotaFn = prevFn })

	u := &repo.User{Username: "used-skip-settle", Quota: 100_000_000}
	if err := repo.DB.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	ch := &repo.Channel{Name: "ch-skip", Key: "k"}
	if err := repo.DB.Create(ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	info := newRelayInfo(u.Id, ch.Id, constant.APITypeOpenAI)
	info.IsPlayground = true
	info.UserQuota = 100_000_000
	info.PriceData = types.PriceData{
		ModelRatio:      2,
		CompletionRatio: 1,
		GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
	}

	c, _ := newJSONContext(http.MethodPost, "/", nil)
	c.Set("token_name", "tkn")
	postConsumeQuota(c, info, &dto.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500})

	var refreshed repo.User
	if err := repo.DB.First(&refreshed, u.Id).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if refreshed.UsedQuota != 0 || refreshed.RequestCount != 0 {
		t.Errorf("user used_quota=%d request_count=%d, want 0/0 after a failed settle", refreshed.UsedQuota, refreshed.RequestCount)
	}
	var refreshedCh repo.Channel
	if err := repo.DB.First(&refreshedCh, ch.Id).Error; err != nil {
		t.Fatalf("reload channel: %v", err)
	}
	if refreshedCh.UsedQuota != 0 {
		t.Errorf("channel used_quota=%d, want 0 after a failed settle", refreshedCh.UsedQuota)
	}
}
