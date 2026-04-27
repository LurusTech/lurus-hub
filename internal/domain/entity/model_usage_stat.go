package entity

import "time"

// ModelUsageStat is a small aggregate table that holds 24-hour call counts
// for each (model, channel_type) pair. Populated hourly by the openrouter
// rank aggregator; read by the sync engine's ranker. Never queried in the
// sync hot path against the raw logs table.
type ModelUsageStat struct {
	ModelName     string    `json:"model_name" gorm:"type:varchar(128);primaryKey"`
	ChannelType   int       `json:"channel_type" gorm:"primaryKey"`
	Count24h      int64     `json:"count_24h" gorm:"default:0"`
	LastUpdatedAt time.Time `json:"last_updated_at" gorm:"index"`
}

func (ModelUsageStat) TableName() string {
	return "model_usage_stats"
}
