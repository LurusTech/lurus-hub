package entity

import "time"

// CatalogSyncRun is one inventory/price-proposal pass over a channel.
type CatalogSyncRun struct {
	Id             int64     `json:"id" gorm:"primaryKey"`
	ChannelId      int       `json:"channel_id" gorm:"index;not null"`
	FetchedTotal   int       `json:"fetched_total"`
	AddedCount     int       `json:"added_count"`
	StaleCount     int       `json:"stale_count"`
	ProposedPrices int       `json:"proposed_prices"`
	CircuitBreaker bool      `json:"circuit_breaker"`
	Error          string    `json:"error" gorm:"type:text"`
	SourceVersion  string    `json:"source_version" gorm:"type:varchar(64);index"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (CatalogSyncRun) TableName() string { return "catalog_sync_runs" }

// CatalogStalePending marks a channel model missing from upstream.
// Removals require human confirm — never auto-deleted by the daily job.
type CatalogStalePending struct {
	Id            int64      `json:"id" gorm:"primaryKey"`
	ChannelId     int        `json:"channel_id" gorm:"uniqueIndex:ux_catalog_stale_channel_model;not null"`
	ModelName     string     `json:"model_name" gorm:"type:varchar(191);uniqueIndex:ux_catalog_stale_channel_model;not null"`
	FirstSeenAt   time.Time  `json:"first_seen_at" gorm:"autoCreateTime"`
	LastSeenAt    time.Time  `json:"last_seen_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	SourceVersion string     `json:"source_version" gorm:"type:varchar(64)"`
}

func (CatalogStalePending) TableName() string { return "catalog_stale_pending" }

// CatalogPriceProposal is a suggested ratio/price change; never auto-applied.
type CatalogPriceProposal struct {
	Id            int64      `json:"id" gorm:"primaryKey"`
	ModelName     string     `json:"model_name" gorm:"type:varchar(191);uniqueIndex:ux_catalog_price_model_field;not null"`
	Field         string     `json:"field" gorm:"type:varchar(64);uniqueIndex:ux_catalog_price_model_field;not null"`
	LocalValue    float64    `json:"local_value"`
	UpstreamValue float64    `json:"upstream_value"`
	Source        string     `json:"source" gorm:"type:varchar(128)"`
	SourceVersion string     `json:"source_version" gorm:"type:varchar(64);index"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	AppliedAt     *time.Time `json:"applied_at"`
	DismissedAt   *time.Time `json:"dismissed_at"`
}

func (CatalogPriceProposal) TableName() string { return "catalog_price_proposals" }
