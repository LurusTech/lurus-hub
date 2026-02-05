package entity

import "time"

// Subscription represents a user's subscription record
type Subscription struct {
	Id       int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId   int    `json:"user_id" gorm:"index;not null"`
	TenantId string `json:"tenant_id" gorm:"type:varchar(36);index;default:'default'"` // Tenant isolation
	PlanCode string `json:"plan_code" gorm:"type:varchar(32);not null"`                // weekly/monthly/quarterly/yearly
	PlanName string `json:"plan_name" gorm:"type:varchar(64);not null"`
	Status   string `json:"status" gorm:"type:varchar(16);default:'active'"` // active/expired/cancelled/pending

	// Quota configuration (synced to User table on activation)
	DailyQuota    int    `json:"daily_quota" gorm:"type:int;default:0"`
	TotalQuota    int    `json:"total_quota" gorm:"type:int;default:0"`
	BaseGroup     string `json:"base_group" gorm:"type:varchar(64)"`
	FallbackGroup string `json:"fallback_group" gorm:"type:varchar(64)"`

	// Time
	StartedAt time.Time `json:"started_at" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`

	// Payment
	PaymentMethod string  `json:"payment_method" gorm:"type:varchar(32)"`    // stripe/epay/creem
	PaymentId     string  `json:"payment_id" gorm:"type:varchar(128);index"` // External payment ID
	Amount        float64 `json:"amount" gorm:"type:decimal(10,2)"`
	Currency      string  `json:"currency" gorm:"type:varchar(8);default:'CNY'"`
	AutoRenew     bool    `json:"auto_renew" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

// SubscriptionStatus constants
const (
	SubscriptionStatusPending   = "pending"   // Payment pending
	SubscriptionStatusActive    = "active"    // Currently active
	SubscriptionStatusExpired   = "expired"   // Expired
	SubscriptionStatusCancelled = "cancelled" // Cancelled by user
)

// SubscriptionPlan represents a subscription plan configuration
type SubscriptionPlan struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Days          int     `json:"days"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	DailyQuota    int     `json:"daily_quota"`
	TotalQuota    int     `json:"total_quota"`
	BaseGroup     string  `json:"base_group"`
	FallbackGroup string  `json:"fallback_group"`
	Enabled       bool    `json:"enabled"`
	SortOrder     int     `json:"sort_order"`
}

