package entity

import (
	"time"

	"gorm.io/gorm"
)

type TenantConfig struct {
	Id          int            `json:"id" gorm:"primaryKey;autoIncrement"`
	TenantID    string         `json:"tenant_id" gorm:"column:tenant_id;size:36;not null;index"`
	ConfigKey   string         `json:"config_key" gorm:"column:config_key;size:128;not null"`
	ConfigValue string         `json:"config_value" gorm:"column:config_value;type:text"`
	ConfigType  string         `json:"config_type" gorm:"column:config_type;size:32;default:'string'"`
	Description string         `json:"description" gorm:"size:255"`
	IsSystem    bool           `json:"is_system" gorm:"column:is_system;default:false;index"`
	IsEncrypted bool           `json:"is_encrypted" gorm:"column:is_encrypted;default:false"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for TenantConfig model
func (TenantConfig) TableName() string {
	return "tenant_configs"
}

// Config type constants
const (
	ConfigTypeString = "string"
	ConfigTypeInt    = "int"
	ConfigTypeBool   = "bool"
	ConfigTypeJSON   = "json"
	ConfigTypeFloat  = "float"
)
