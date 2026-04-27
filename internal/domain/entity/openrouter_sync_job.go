package entity

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

// OpenRouterSyncJob defines a recurring rule for importing OpenRouter free models
// into a target channel, filtered by category and ranked by internal usage.
type OpenRouterSyncJob struct {
	Id              int        `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"type:varchar(128);not null"`
	TargetChannelId int        `json:"target_channel_id" gorm:"index;not null"`
	Categories      string     `json:"categories" gorm:"type:text;not null"`
	TopN            int        `json:"top_n" gorm:"default:0"`
	Schedule        string     `json:"schedule" gorm:"type:varchar(32);default:'manual'"`
	Enabled         bool       `json:"enabled" gorm:"default:true"`
	LastRunAt       *time.Time `json:"last_run_at"`
	LastError       string     `json:"last_error" gorm:"type:text"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (OpenRouterSyncJob) TableName() string {
	return "openrouter_sync_jobs"
}

// Supported category keys for the Categories JSON array.
const (
	OpenRouterCategoryLLMReasoning = "llm_reasoning"
	OpenRouterCategoryVision       = "vision"
	OpenRouterCategoryImageGen     = "image_gen"
	OpenRouterCategoryASR          = "asr"
	OpenRouterCategoryTTS          = "tts"
)

// Schedule values
const (
	OpenRouterScheduleDaily  = "daily"
	OpenRouterScheduleWeekly = "weekly"
	OpenRouterScheduleManual = "manual"
)

// GetCategories parses the JSON-encoded Categories field.
func (j *OpenRouterSyncJob) GetCategories() []string {
	if strings.TrimSpace(j.Categories) == "" {
		return []string{}
	}
	var cats []string
	if err := json.Unmarshal([]byte(j.Categories), &cats); err != nil {
		common.SysLog("OpenRouterSyncJob: failed to unmarshal Categories: " + err.Error())
		return []string{}
	}
	return cats
}

// SetCategories serializes the given categories into the Categories field.
func (j *OpenRouterSyncJob) SetCategories(cats []string) error {
	b, err := json.Marshal(cats)
	if err != nil {
		return err
	}
	j.Categories = string(b)
	return nil
}

// ShouldRun decides whether this job is due based on its Schedule and LastRunAt.
// Manual-only jobs are never picked up by the scheduler.
func (j *OpenRouterSyncJob) ShouldRun(now time.Time) bool {
	if !j.Enabled {
		return false
	}
	switch j.Schedule {
	case OpenRouterScheduleDaily:
		if j.LastRunAt == nil {
			return true
		}
		return now.Sub(*j.LastRunAt) >= 24*time.Hour
	case OpenRouterScheduleWeekly:
		if j.LastRunAt == nil {
			return true
		}
		return now.Sub(*j.LastRunAt) >= 7*24*time.Hour
	default: // manual or unknown — never auto-runs
		return false
	}
}
