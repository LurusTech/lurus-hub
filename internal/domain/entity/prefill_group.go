package entity

import (
	"database/sql/driver"
	"encoding/json"

	"gorm.io/gorm"
)

// JSONValue based on json.RawMessage for GORM value object
type JSONValue json.RawMessage

func (j JSONValue) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONValue) Scan(value interface{}) error {
	if value == nil {
		*j = JSONValue("null")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = JSONValue(v)
	case string:
		*j = JSONValue(v)
	}
	return nil
}

func (j JSONValue) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONValue) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = JSONValue(data)
	return nil
}

type PrefillGroup struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex:uk_prefill_name,where:deleted_at IS NULL"`
	Type        string         `json:"type" gorm:"size:32;index;not null"`
	Items       JSONValue      `json:"items" gorm:"type:json"`
	Description string         `json:"description,omitempty" gorm:"type:varchar(255)"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
