package repo

import (
	"github.com/QuantumNous/lurus-api/internal/domain/entity"
	"github.com/QuantumNous/lurus-api/internal/pkg/constant"
)

// GovernanceChannelStat holds aggregated stats grouped by channel type.
type GovernanceChannelStat struct {
	ChannelType int    `json:"channel_type"`
	ChannelName string `json:"channel_name"`
	Count       int64  `json:"count"`
	TotalQuota  int64  `json:"total_quota"`
}

// GovernanceFingerprintStat holds deduplication stats per token.
type GovernanceFingerprintStat struct {
	TokenID       int   `json:"token_id"`
	Total         int64 `json:"total"`
	UniqueCount   int64 `json:"unique_count"`
	DuplicateRate float64 `json:"duplicate_rate"`
}

// GovernanceLatencyStat holds latency aggregation per model.
type GovernanceLatencyStat struct {
	ModelName string  `json:"model_name"`
	AvgMs     float64 `json:"avg_ms"`
	MaxMs     float64 `json:"max_ms"`
	Count     int64   `json:"count"`
}

// GetChannelDistribution returns request count and quota grouped by channel_type.
func GetChannelDistribution(startTime int64) ([]GovernanceChannelStat, error) {
	var results []GovernanceChannelStat
	// Filter out channel_type=0 (historical data before governance columns were added).
	err := LOG_DB.Model(&entity.Log{}).
		Select("channel_type, COUNT(*) as count, COALESCE(SUM(quota), 0) as total_quota").
		Where("type = ? AND created_at >= ? AND channel_type > 0", LogTypeConsume, startTime).
		Group("channel_type").
		Order("total_quota DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].ChannelName = constant.GetChannelTypeName(results[i].ChannelType)
	}
	return results, nil
}

// GetFingerprintDedupStats returns fingerprint deduplication stats per token
// for tokens with more than minCount requests since startTime.
func GetFingerprintDedupStats(startTime int64, minCount int) ([]GovernanceFingerprintStat, error) {
	var results []GovernanceFingerprintStat
	err := LOG_DB.Model(&entity.Log{}).
		Select("token_id, COUNT(*) as total, COUNT(DISTINCT request_fingerprint) as unique_count").
		Where("type = ? AND created_at >= ? AND request_fingerprint != ''", LogTypeConsume, startTime).
		Group("token_id").
		Having("COUNT(*) > ?", minCount).
		Order("total DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].Total > 0 {
			results[i].DuplicateRate = 1.0 - float64(results[i].UniqueCount)/float64(results[i].Total)
		}
	}
	return results, nil
}

// GetLatencyStats returns average and max latency per model.
// Uses AVG/MAX for cross-DB compatibility (PostgreSQL, MySQL, SQLite).
func GetLatencyStats(startTime int64) ([]GovernanceLatencyStat, error) {
	var results []GovernanceLatencyStat
	err := LOG_DB.Model(&entity.Log{}).
		Select("model_name, AVG(total_latency_ms) as avg_ms, MAX(total_latency_ms) as max_ms, COUNT(*) as count").
		Where("type = ? AND created_at >= ? AND total_latency_ms > 0", LogTypeConsume, startTime).
		Group("model_name").
		Having("COUNT(*) > 1").
		Order("avg_ms DESC").
		Find(&results).Error
	return results, err
}

// GovernanceEfficiencyStat holds cost efficiency metrics per model.
// Aligned with Portkey 4-dimension cost governance: Usage / Efficiency / Routing / Budgets.
type GovernanceEfficiencyStat struct {
	ModelName       string  `json:"model_name"`
	TotalRequests   int64   `json:"total_requests"`
	TotalQuota      int64   `json:"total_quota"`
	TotalTokens     int64   `json:"total_tokens"`
	AvgQuotaPerReq  float64 `json:"avg_quota_per_request"`
	AvgTokensPerReq float64 `json:"avg_tokens_per_request"`
	QuotaPerToken   float64 `json:"quota_per_token"`
}

// GetEfficiencyStats returns cost-per-request and cost-per-token metrics per model.
func GetEfficiencyStats(startTime int64) ([]GovernanceEfficiencyStat, error) {
	var results []GovernanceEfficiencyStat
	err := LOG_DB.Model(&entity.Log{}).
		Select(`model_name,
			COUNT(*) as total_requests,
			COALESCE(SUM(quota), 0) as total_quota,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens`).
		Where("type = ? AND created_at >= ? AND quota > 0", LogTypeConsume, startTime).
		Group("model_name").
		Having("COUNT(*) > 0").
		Order("total_quota DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].TotalRequests > 0 {
			results[i].AvgQuotaPerReq = float64(results[i].TotalQuota) / float64(results[i].TotalRequests)
			results[i].AvgTokensPerReq = float64(results[i].TotalTokens) / float64(results[i].TotalRequests)
		}
		if results[i].TotalTokens > 0 {
			results[i].QuotaPerToken = float64(results[i].TotalQuota) / float64(results[i].TotalTokens)
		}
	}
	return results, nil
}
