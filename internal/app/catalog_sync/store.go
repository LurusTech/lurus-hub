package catalog_sync

import (
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func db() *gorm.DB { return repo.DB }

func saveRun(run *entity.CatalogSyncRun) error {
	return db().Create(run).Error
}

func upsertStale(channelID int, models []string, sourceVersion string, now time.Time) error {
	for _, model := range models {
		row := entity.CatalogStalePending{
			ChannelId:     channelID,
			ModelName:     model,
			LastSeenAt:    now,
			SourceVersion: sourceVersion,
		}
		err := db().Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "channel_id"}, {Name: "model_name"}},
			DoUpdates: clause.Assignments(map[string]any{
				"last_seen_at":   now,
				"source_version": sourceVersion,
				"resolved_at":    nil,
			}),
		}).Create(&row).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func clearStaleResolvedIfBack(channelID int, present map[string]bool, now time.Time) error {
	var rows []entity.CatalogStalePending
	if err := db().Where("channel_id = ? AND resolved_at IS NULL", channelID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if present[row.ModelName] {
			resolved := now
			if err := db().Model(&entity.CatalogStalePending{}).Where("id = ?", row.Id).
				Update("resolved_at", resolved).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func upsertPriceProposals(hints []PriceHint, local map[string]float64, sourceVersion string) (int, error) {
	n := 0
	now := time.Now().UTC()
	for _, h := range hints {
		cur, ok := local[h.ModelName]
		if h.Field != "model_ratio" {
			// MVP: only propose model_ratio diffs from hints keyed as ratios.
			continue
		}
		if ok && nearlyEqual(cur, h.Value) {
			continue
		}
		row := entity.CatalogPriceProposal{
			ModelName:     h.ModelName,
			Field:         h.Field,
			LocalValue:    cur,
			UpstreamValue: h.Value,
			Source:        h.Source,
			SourceVersion: sourceVersion,
			CreatedAt:     now,
		}
		err := db().Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "model_name"}, {Name: "field"}},
			DoUpdates: clause.Assignments(map[string]any{
				"local_value":    cur,
				"upstream_value": h.Value,
				"source":         h.Source,
				"source_version": sourceVersion,
				"created_at":     now,
				"applied_at":     nil,
				"dismissed_at":   nil,
			}),
		}).Create(&row).Error
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func nearlyEqual(a, b float64) bool {
	const eps = 1e-9
	if a > b {
		return a-b < eps
	}
	return b-a < eps
}

// ListOpenStale returns unresolved stale rows, optionally filtered by channel.
func ListOpenStale(channelID int) ([]entity.CatalogStalePending, error) {
	q := db().Where("resolved_at IS NULL")
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	var rows []entity.CatalogStalePending
	err := q.Order("channel_id, model_name").Find(&rows).Error
	return rows, err
}

// ListOpenPriceProposals returns unapplied, undismissed proposals.
func ListOpenPriceProposals() ([]entity.CatalogPriceProposal, error) {
	var rows []entity.CatalogPriceProposal
	err := db().Where("applied_at IS NULL AND dismissed_at IS NULL").
		Order("model_name, field").Find(&rows).Error
	return rows, err
}

// ListRecentRuns returns the latest sync runs.
func ListRecentRuns(limit int) ([]entity.CatalogSyncRun, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []entity.CatalogSyncRun
	err := db().Order("id desc").Limit(limit).Find(&rows).Error
	return rows, err
}
