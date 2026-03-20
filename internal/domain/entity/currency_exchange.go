package entity

import "time"

// CurrencyExchange records a single currency exchange transaction.
// Append-only: rows are never updated or deleted after creation.
type CurrencyExchange struct {
	Id               int       `json:"id" gorm:"primaryKey"`
	UserId           int       `json:"user_id" gorm:"index;not null"`
	SourceCurrency   string    `json:"source_currency" gorm:"size:3;not null"`  // "LUC"
	SourceAmount     float64   `json:"source_amount" gorm:"not null"`           // e.g. 10.0 LUC
	TargetCurrency   string    `json:"target_currency" gorm:"size:3;not null"`  // "LUT"
	TargetAmount     int       `json:"target_amount" gorm:"not null"`           // e.g. 5000000 LUT
	ExchangeRate     float64   `json:"exchange_rate" gorm:"not null"`           // effective rate applied
	VIPLevel         int       `json:"vip_level" gorm:"default:0"`             // VIP level at time of exchange
	VIPBonus         float64   `json:"vip_bonus" gorm:"default:1.0"`           // bonus multiplier applied
	ReferenceId      string    `json:"reference_id" gorm:"size:64;uniqueIndex"` // idempotency key
	PlatformOrderNo  string    `json:"platform_order_no" gorm:"size:64;index"`  // platform wallet tx reference
	SourceService    string    `json:"source_service" gorm:"size:32"`           // "lurus-platform", "lurus-api"
	Note             string    `json:"note" gorm:"size:255"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (CurrencyExchange) TableName() string {
	return "currency_exchanges"
}
