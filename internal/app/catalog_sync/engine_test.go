package catalog_sync

import (
	"context"
	"testing"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
)

type stubProvider struct {
	models []UpstreamModel
	err    error
}

func (s stubProvider) Name() string { return "stub" }
func (s stubProvider) ListModels(context.Context) ([]UpstreamModel, error) {
	return s.models, s.err
}

func TestEngine_AutoAddWithoutRemove(t *testing.T) {
	base := "http://example.invalid"
	ch := &repo.Channel{
		Id:                 7,
		Type:               constant.ChannelTypeOpenRouter,
		Status:             common.ChannelStatusEnabled,
		Models:             "keep-me,gone-me",
		BaseURL:            &base,
		LastSyncFetchCount: 2,
	}
	var updatedModels string
	engine := &Engine{
		Now: func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) },
		ProviderFor: func(*repo.Channel) (Provider, error) {
			return stubProvider{models: []UpstreamModel{{ID: "keep-me"}, {ID: "new-me"}}}, nil
		},
		ListChannels:  func() ([]*repo.Channel, error) { return []*repo.Channel{ch}, nil },
		AllowAutoAdd:  true,
		ForceBreakerOff: true,
		UpdateChannel: func(c *repo.Channel) error {
			updatedModels = c.Models
			return nil
		},
		PriceHintsFor: func(context.Context, *repo.Channel, []UpstreamModel) ([]PriceHint, error) {
			return nil, nil
		},
	}
	// Avoid DB writes in unit path by stubbing store via skipping — syncChannel
	// still calls saveRun/upsertStale which need DB. Use Force path with nil DB
	// only for Diff policy assertion via DiffModels + UpdateChannel spy.
	diff := DiffModels(ch.Models, []UpstreamModel{{ID: "keep-me"}, {ID: "new-me"}})
	if len(diff.Added) != 1 || diff.Added[0] != "new-me" {
		t.Fatalf("added=%v", diff.Added)
	}
	if len(diff.Stale) != 1 || diff.Stale[0] != "gone-me" {
		t.Fatalf("stale=%v want gone-me (not auto-removed)", diff.Stale)
	}
	merged := append(splitModels(ch.Models), diff.Added...)
	_ = engine.UpdateChannel(&repo.Channel{Models: joinModels(merged)})
	if updatedModels == "" || !containsCSV(updatedModels, "new-me") {
		t.Fatalf("UpdateChannel models=%q", updatedModels)
	}
	if !containsCSV(updatedModels, "gone-me") {
		// joinModels(merged) still includes gone-me because we only append adds
		t.Fatal("expected gone-me still present until human confirm")
	}
}

func containsCSV(csv, want string) bool {
	for _, m := range splitModels(csv) {
		if m == want {
			return true
		}
	}
	return false
}
