package repo

import (
	"context"
	"time"

	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
	"gorm.io/gorm/clause"
)

// ModelUsageStat is aliased from the canonical entity definition.
type ModelUsageStat = entity.ModelUsageStat

// UpsertModelUsageStats writes or updates usage counts for a batch of (model, channelType) rows.
// On conflict, count_24h and last_updated_at are overwritten.
func UpsertModelUsageStats(ctx context.Context, rows []ModelUsageStat) error {
	if len(rows) == 0 {
		return nil
	}
	return DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "model_name"}, {Name: "channel_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"count_24h",
			"last_updated_at",
		}),
	}).Create(&rows).Error
}

// GetModelUsageStatsByChannelType returns all stats for a given channel type.
// Caller is responsible for inspecting LastUpdatedAt to detect stale data.
func GetModelUsageStatsByChannelType(channelType int) ([]ModelUsageStat, error) {
	var stats []ModelUsageStat
	err := DB.Where("channel_type = ?", channelType).Find(&stats).Error
	return stats, err
}

// DeleteStaleModelUsageStats removes rows older than the given threshold.
func DeleteStaleModelUsageStats(channelType int, olderThan time.Time) error {
	return DB.Where("channel_type = ? AND last_updated_at < ?", channelType, olderThan).
		Delete(&ModelUsageStat{}).Error
}
