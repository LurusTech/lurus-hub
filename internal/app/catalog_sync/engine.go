package catalog_sync

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting/ratio_setting"
)

// Engine runs one catalog sync cycle across eligible channels.
type Engine struct {
	Now            func() time.Time
	ProviderFor    func(*repo.Channel) (Provider, error)
	PriceHintsFor  func(ctx context.Context, ch *repo.Channel, models []UpstreamModel) ([]PriceHint, error)
	UpdateChannel  func(ch *repo.Channel) error
	ListChannels   func() ([]*repo.Channel, error)
	AllowAutoAdd   bool // when false, only record diffs (dry preview path)
	ForceBreakerOff bool
}

func NewEngine() *Engine {
	return &Engine{
		Now:         time.Now,
		ProviderFor: ProviderForChannel,
		PriceHintsFor: OfficialRatioHints,
		UpdateChannel: func(ch *repo.Channel) error {
			return ch.Update() // persists Models and refreshes abilities
		},
		ListChannels: listEligibleChannels,
		AllowAutoAdd: true,
	}
}

// ChannelResult summarizes one channel pass.
type ChannelResult struct {
	ChannelId      int      `json:"channel_id"`
	FetchedTotal   int      `json:"fetched_total"`
	Added          []string `json:"added"`
	Stale          []string `json:"stale"`
	ProposedPrices int      `json:"proposed_prices"`
	CircuitBreaker bool     `json:"circuit_breaker"`
	Skipped        bool     `json:"skipped"`
	Error          string   `json:"error,omitempty"`
	SourceVersion  string   `json:"source_version"`
}

// RunResult aggregates a full tick.
type RunResult struct {
	Channels []ChannelResult `json:"channels"`
}

func listEligibleChannels() ([]*repo.Channel, error) {
	channels, err := repo.GetAllChannels(0, 0, true, true)
	if err != nil {
		return nil, err
	}
	allow := parseChannelAllowlist(os.Getenv("CATALOG_SYNC_CHANNEL_IDS"))
	supported := SupportedChannelTypes()
	out := make([]*repo.Channel, 0)
	for _, ch := range channels {
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if !supported[ch.Type] {
			continue
		}
		if len(allow) > 0 && !allow[ch.Id] {
			continue
		}
		out = append(out, ch)
	}
	return out, nil
}

func parseChannelAllowlist(raw string) map[int]bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	m := map[int]bool{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			continue
		}
		m[id] = true
	}
	return m
}

// Run executes one sync cycle.
func (e *Engine) Run(ctx context.Context) (*RunResult, error) {
	if e.Now == nil {
		e.Now = time.Now
	}
	if e.ProviderFor == nil {
		e.ProviderFor = ProviderForChannel
	}
	if e.ListChannels == nil {
		e.ListChannels = listEligibleChannels
	}
	channels, err := e.ListChannels()
	if err != nil {
		return nil, err
	}
	result := &RunResult{}
	for _, ch := range channels {
		cr := e.syncChannel(ctx, ch)
		result.Channels = append(result.Channels, cr)
	}
	return result, nil
}

func (e *Engine) syncChannel(ctx context.Context, ch *repo.Channel) ChannelResult {
	now := e.Now().UTC()
	cr := ChannelResult{ChannelId: ch.Id}
	provider, err := e.ProviderFor(ch)
	if err != nil {
		cr.Error = err.Error()
		_ = saveRun(&entity.CatalogSyncRun{ChannelId: ch.Id, Error: cr.Error, CreatedAt: now})
		return cr
	}
	models, err := provider.ListModels(ctx)
	if err != nil {
		cr.Error = err.Error()
		_ = saveRun(&entity.CatalogSyncRun{ChannelId: ch.Id, Error: cr.Error, CreatedAt: now})
		return cr
	}
	cr.FetchedTotal = len(models)
	if !e.ForceBreakerOff && ShouldTripBreaker(len(models), ch.LastSyncFetchCount) {
		cr.CircuitBreaker = true
		cr.Skipped = true
		cr.Error = fmt.Sprintf("%v: fetched=%d baseline=%d", ErrCircuitBreaker, len(models), ch.LastSyncFetchCount)
		_ = saveRun(&entity.CatalogSyncRun{
			ChannelId: ch.Id, FetchedTotal: len(models), CircuitBreaker: true,
			Error: cr.Error, CreatedAt: now,
		})
		return cr
	}

	diff := DiffModels(ch.Models, models)
	cr.Added = diff.Added
	cr.Stale = diff.Stale
	cr.SourceVersion = diff.Version

	present := setOf(append(append([]string{}, diff.Kept...), diff.Added...))
	if err := upsertStale(ch.Id, diff.Stale, diff.Version, now); err != nil {
		cr.Error = "stale upsert: " + err.Error()
	}
	if err := clearStaleResolvedIfBack(ch.Id, present, now); err != nil && cr.Error == "" {
		cr.Error = "stale resolve: " + err.Error()
	}

	if e.AllowAutoAdd && len(diff.Added) > 0 {
		merged := append(splitModels(ch.Models), diff.Added...)
		ch.Models = joinModels(merged)
		ch.LastSyncFetchCount = len(models)
		if e.UpdateChannel != nil {
			if err := e.UpdateChannel(ch); err != nil {
				cr.Error = "channel update: " + err.Error()
			}
		}
	} else {
		ch.LastSyncFetchCount = len(models)
		_ = repo.DB.Model(ch).Update("last_sync_fetch_count", ch.LastSyncFetchCount).Error
	}

	if e.PriceHintsFor != nil {
		hints, herr := e.PriceHintsFor(ctx, ch, models)
		if herr != nil && cr.Error == "" {
			cr.Error = "price hints: " + herr.Error()
		} else if len(hints) > 0 {
			n, perr := upsertPriceProposals(hints, ratio_setting.GetModelRatioCopy(), diff.Version)
			if perr != nil && cr.Error == "" {
				cr.Error = "price proposals: " + perr.Error()
			}
			cr.ProposedPrices = n
		}
	}

	_ = saveRun(&entity.CatalogSyncRun{
		ChannelId:      ch.Id,
		FetchedTotal:   len(models),
		AddedCount:     len(diff.Added),
		StaleCount:     len(diff.Stale),
		ProposedPrices: cr.ProposedPrices,
		CircuitBreaker: false,
		Error:          cr.Error,
		SourceVersion:  diff.Version,
		CreatedAt:      now,
	})
	return cr
}
