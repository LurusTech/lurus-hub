package entity

import (
	"time"

	"gorm.io/gorm"
)

// UserIdentityMapping maps Lurus users to Zitadel identities for multi-tenant support
type UserIdentityMapping struct {
	Id                int            `json:"id" gorm:"primaryKey;autoIncrement"`
	LurusUserID       int            `json:"lurus_user_id" gorm:"column:lurus_user_id;not null;index"`
	ZitadelUserID     string         `json:"zitadel_user_id" gorm:"column:zitadel_user_id;size:128;not null;index"`
	TenantID          string         `json:"tenant_id" gorm:"column:tenant_id;size:36;not null;index"`
	Email             string         `json:"email" gorm:"size:255;index"`
	DisplayName       string         `json:"display_name" gorm:"column:display_name;size:128"`
	PreferredUsername string         `json:"preferred_username" gorm:"column:preferred_username;size:128"`
	LastSyncAt        *time.Time     `json:"last_sync_at" gorm:"column:last_sync_at"`
	IsActive          bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for UserIdentityMapping model
func (UserIdentityMapping) TableName() string {
	return "user_identity_mapping"
}

// ZitadelUserClaims represents claims extracted from Zitadel OIDC token
type ZitadelUserClaims struct {
	Sub               string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	OrgID             string
	OrgDomain         string
}
