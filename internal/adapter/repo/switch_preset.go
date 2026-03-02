package repo

import (
	"context"
	"encoding/json"
	"time"
)

// SwitchConfigPresetRow is the DB model for the switch_config_presets table.
type SwitchConfigPresetRow struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Tool        string    `gorm:"type:varchar(32);not null;index:idx_switch_presets_tool"`
	Name        string    `gorm:"type:varchar(128);not null"`
	Description string    `gorm:"type:text"`
	Category    string    `gorm:"type:varchar(64);index:idx_switch_presets_tool"`
	ConfigJSON  []byte    `gorm:"type:jsonb;not null"`
	IsOfficial  bool      `gorm:"default:true;index:idx_switch_presets_tool"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (SwitchConfigPresetRow) TableName() string { return "switch_config_presets" }

// SwitchPresetDTO is the API-facing representation returned by handler functions.
type SwitchPresetDTO struct {
	ID          string                 `json:"id"`
	Tool        string                 `json:"tool"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	ConfigJSON  map[string]interface{} `json:"config_json"`
	IsOfficial  bool                   `json:"is_official"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ListSwitchConfigPresets queries presets filtered by tool (optional) and category (optional).
// Results are paginated via limit/offset.
func ListSwitchConfigPresets(ctx context.Context, tool, category string, limit, offset int) ([]SwitchPresetDTO, error) {
	q := DB.WithContext(ctx).Model(&SwitchConfigPresetRow{}).Where("is_official = ?", true)
	if tool != "" {
		q = q.Where("tool = ?", tool)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var rows []SwitchConfigPresetRow
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}

	dtos := make([]SwitchPresetDTO, 0, len(rows))
	for _, r := range rows {
		dto := SwitchPresetDTO{
			ID:          r.ID,
			Tool:        r.Tool,
			Name:        r.Name,
			Description: r.Description,
			Category:    r.Category,
			IsOfficial:  r.IsOfficial,
			CreatedAt:   r.CreatedAt,
		}
		var cfg map[string]interface{}
		if err := json.Unmarshal(r.ConfigJSON, &cfg); err == nil {
			dto.ConfigJSON = cfg
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

// CreateSwitchConfigPreset inserts a new preset record and returns the created DTO.
func CreateSwitchConfigPreset(ctx context.Context, tool, name, description, category string, configJSON map[string]interface{}, isOfficial bool) (*SwitchPresetDTO, error) {
	raw, err := json.Marshal(configJSON)
	if err != nil {
		return nil, err
	}

	row := SwitchConfigPresetRow{
		Tool:        tool,
		Name:        name,
		Description: description,
		Category:    category,
		ConfigJSON:  raw,
		IsOfficial:  isOfficial,
	}
	if err := DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}

	dto := &SwitchPresetDTO{
		ID:          row.ID,
		Tool:        row.Tool,
		Name:        row.Name,
		Description: row.Description,
		Category:    row.Category,
		ConfigJSON:  configJSON,
		IsOfficial:  row.IsOfficial,
		CreatedAt:   row.CreatedAt,
	}
	return dto, nil
}
