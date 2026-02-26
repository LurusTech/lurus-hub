package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrTwoFANotEnabled = errors.New("用户未启用2FA")

type TwoFA struct {
	Id             int            `json:"id" gorm:"primaryKey"`
	TenantId       string         `json:"tenant_id" gorm:"type:varchar(36);index;default:'default'"` // Tenant isolation
	UserId         int            `json:"user_id" gorm:"unique;not null;index"`
	Secret         string         `json:"-" gorm:"type:varchar(255);not null"`
	IsEnabled      bool           `json:"is_enabled"`
	FailedAttempts int            `json:"failed_attempts" gorm:"default:0"`
	LockedUntil    *time.Time     `json:"locked_until,omitempty"`
	LastUsedAt     *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// IsLocked returns true if the 2FA is currently locked
func (t *TwoFA) IsLocked() bool {
	if t.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*t.LockedUntil)
}

type TwoFABackupCode struct {
	Id        int            `json:"id" gorm:"primaryKey"`
	UserId    int            `json:"user_id" gorm:"not null;index"`
	CodeHash  string         `json:"-" gorm:"type:varchar(255);not null"`
	IsUsed    bool           `json:"is_used"`
	UsedAt    *time.Time     `json:"used_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
