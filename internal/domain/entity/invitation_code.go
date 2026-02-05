package entity

import "time"

// InvitationCode represents a one-time use invitation code for registration
type InvitationCode struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	Code      string `json:"code" gorm:"uniqueIndex;size:32"`
	CreatedBy int    `json:"created_by" gorm:"index"`
	UsedBy    *int   `json:"used_by"`
	UsedAt    *int64 `json:"used_at"`
	ExpiresAt *int64 `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
}

func (c InvitationCode) TableName() string {
	return "invitation_codes"
}

// IsValid checks if the invitation code is still valid
func (c *InvitationCode) IsValid() bool {
	if c == nil {
		return false
	}
	if c.UsedBy != nil {
		return false
	}
	// ExpiresAt == nil means no expiration
	if c.ExpiresAt != nil && *c.ExpiresAt < time.Now().Unix() {
		return false
	}
	return true
}
