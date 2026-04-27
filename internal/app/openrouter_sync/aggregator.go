package openrouter_sync

import (
	"context"
	"fmt"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
)

// aggregateQueryTimeout caps the GROUP BY query so a heavy logs table can't
// stall the hourly ticker. Beyond this, ranking falls back to model.Created.
const aggregateQueryTimeout = 5 * time.Second

// aggregateLimit caps the number of rows returned. TopN is at most ~50 in
// practice; 200 leaves a safe buffer while keeping the query cheap.
const aggregateLimit = 200

// AggregateRow is the row shape returned by the GROUP BY query.
type AggregateRow struct {
	ModelName string
	Count24h  int64
}

// AggregateOpenRouterUsage reads the logs table for the past 24 hours filtered
// to OpenRouter `:free` models, groups by model_name ordered by count desc,
// and upserts the result into model_usage_stats.
//
// Master-only: callers must guard with common.IsMasterNode before scheduling.
// Failure (including timeout) is non-fatal: the sync engine's ranker degrades
// to model.Created when stats are stale or empty.
func AggregateOpenRouterUsage(parent context.Context) error {
	if repo.DB == nil {
		return fmt.Errorf("aggregator: DB not initialized")
	}

	ctx, cancel := context.WithTimeout(parent, aggregateQueryTimeout)
	defer cancel()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour).Unix()

	var rows []AggregateRow
	err := repo.LOG_DB.WithContext(ctx).
		Table("logs").
		Select("model_name AS model_name, COUNT(*) AS count_24h").
		Where("channel_type = ?", constant.ChannelTypeOpenRouter).
		Where("created_at > ?", cutoff).
		Where("model_name LIKE ?", "%:free").
		Group("model_name").
		Order("count_24h DESC").
		Limit(aggregateLimit).
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("aggregator: query logs: %w", err)
	}

	if len(rows) == 0 {
		common.SysLog("openrouter aggregator: no usage rows in last 24h")
		return nil
	}

	stats := make([]repo.ModelUsageStat, 0, len(rows))
	for _, r := range rows {
		stats = append(stats, repo.ModelUsageStat{
			ModelName:     r.ModelName,
			ChannelType:   constant.ChannelTypeOpenRouter,
			Count24h:      r.Count24h,
			LastUpdatedAt: now,
		})
	}

	if err := repo.UpsertModelUsageStats(parent, stats); err != nil {
		return fmt.Errorf("aggregator: upsert stats: %w", err)
	}

	// Drop rows that haven't been refreshed in over 24h (model went idle/disappeared).
	if err := repo.DeleteStaleModelUsageStats(constant.ChannelTypeOpenRouter, now.Add(-24*time.Hour)); err != nil {
		// Non-fatal — stale rows will be filtered by BuildUsageMap anyway.
		common.SysLog("openrouter aggregator: cleanup stale rows failed: " + err.Error())
	}

	common.SysLog(fmt.Sprintf("openrouter aggregator: refreshed %d model usage rows", len(stats)))
	return nil
}

// LoadUsageStats reads stats from DB and converts them into the Ranker's input format.
func LoadUsageStats() ([]Stat, error) {
	rows, err := repo.GetModelUsageStatsByChannelType(constant.ChannelTypeOpenRouter)
	if err != nil {
		return nil, err
	}
	out := make([]Stat, 0, len(rows))
	for _, r := range rows {
		out = append(out, Stat{
			ModelName:     r.ModelName,
			Count24h:      r.Count24h,
			LastUpdatedAt: r.LastUpdatedAt,
		})
	}
	return out, nil
}
