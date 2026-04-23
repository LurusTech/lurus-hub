package entity

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/constant"
	"github.com/LurusTech/lurus-api/internal/pkg/dto"
	"github.com/LurusTech/lurus-api/internal/pkg/types"
)

type Channel struct {
	Id                 int     `json:"id"`
	TenantId           string  `json:"tenant_id" gorm:"type:varchar(36);index;index:idx_tenant_status,priority:1;default:'default'"` // Tenant isolation
	Type               int     `json:"type" gorm:"default:0"`
	Key                string  `json:"key" gorm:"not null"`
	OpenAIOrganization *string `json:"openai_organization"`
	TestModel          *string `json:"test_model"`
	Status             int     `json:"status" gorm:"default:1;index:idx_tenant_status,priority:2"`
	Name               string  `json:"name" gorm:"index"`
	Weight             *uint   `json:"weight" gorm:"default:0"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	TestTime           int64   `json:"test_time" gorm:"bigint"`
	ResponseTime       int     `json:"response_time"` // in milliseconds
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
	Other              string  `json:"other"`
	Balance            float64 `json:"balance"` // in USD
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
	Models             string  `json:"models"`
	Group              string  `json:"group" gorm:"type:varchar(64);default:'default'"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string `json:"model_mapping" gorm:"type:text"`
	StatusCodeMapping  *string `json:"status_code_mapping" gorm:"type:varchar(1024);default:''"`
	Priority           *int64  `json:"priority" gorm:"bigint;default:0"`
	AutoBan            *int    `json:"auto_ban" gorm:"default:1"`
	OtherInfo          string  `json:"other_info"`
	Tag                *string `json:"tag" gorm:"index"`
	Setting            *string `json:"setting" gorm:"type:text"` // Channel extra settings
	ParamOverride      *string `json:"param_override" gorm:"type:text"`
	HeaderOverride     *string `json:"header_override" gorm:"type:text"`
	Remark             *string `json:"remark" gorm:"type:varchar(255)" validate:"max=255"`
	ChannelInfo        ChannelInfo `json:"channel_info" gorm:"type:json"`
	OtherSettings      string  `json:"settings" gorm:"column:settings"` // Other settings (azure version, etc.)

	// cache info
	Keys []string `json:"-" gorm:"-"`
}

type ChannelInfo struct {
	IsMultiKey             bool                  `json:"is_multi_key"`
	MultiKeySize           int                   `json:"multi_key_size"`
	MultiKeyStatusList     map[int]int           `json:"multi_key_status_list"`
	MultiKeyDisabledReason map[int]string        `json:"multi_key_disabled_reason,omitempty"`
	MultiKeyDisabledTime   map[int]int64         `json:"multi_key_disabled_time,omitempty"`
	MultiKeyPollingIndex   int                   `json:"multi_key_polling_index"`
	MultiKeyMode           constant.MultiKeyMode `json:"multi_key_mode"`
}

// Value implements driver.Valuer interface
func (c ChannelInfo) Value() (driver.Value, error) {
	return common.Marshal(&c)
}

// Scan implements sql.Scanner interface
func (c *ChannelInfo) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return common.Unmarshal(bytesValue, c)
}

func (channel *Channel) GetKeys() []string {
	if channel.Key == "" {
		return []string{}
	}
	if len(channel.Keys) > 0 {
		return channel.Keys
	}
	trimmed := strings.TrimSpace(channel.Key)
	// If the key starts with '[', try to parse it as a JSON array (e.g., for Vertex AI scenarios)
	if strings.HasPrefix(trimmed, "[") {
		var arr []json.RawMessage
		if err := common.Unmarshal([]byte(trimmed), &arr); err == nil {
			res := make([]string, len(arr))
			for i, v := range arr {
				res[i] = string(v)
			}
			return res
		}
	}
	// Otherwise, fall back to splitting by newline
	keys := strings.Split(strings.Trim(channel.Key, "\n"), "\n")
	return keys
}

func (channel *Channel) GetModels() []string {
	if channel.Models == "" {
		return []string{}
	}
	return strings.Split(strings.Trim(channel.Models, ","), ",")
}

func (channel *Channel) GetGroups() []string {
	if channel.Group == "" {
		return []string{}
	}
	groups := strings.Split(strings.Trim(channel.Group, ","), ",")
	for i, group := range groups {
		groups[i] = strings.TrimSpace(group)
	}
	return groups
}

func (channel *Channel) GetOtherInfo() map[string]interface{} {
	otherInfo := make(map[string]interface{})
	if channel.OtherInfo != "" {
		err := common.Unmarshal([]byte(channel.OtherInfo), &otherInfo)
		if err != nil {
			common.SysLog("failed to unmarshal other info for channel " + channel.Name + ": " + err.Error())
		}
	}
	return otherInfo
}

func (channel *Channel) SetOtherInfo(otherInfo map[string]interface{}) {
	otherInfoBytes, err := json.Marshal(otherInfo)
	if err != nil {
		common.SysLog("failed to marshal other info for channel " + channel.Name + ": " + err.Error())
		return
	}
	channel.OtherInfo = string(otherInfoBytes)
}

func (channel *Channel) GetTag() string {
	if channel.Tag == nil {
		return ""
	}
	return *channel.Tag
}

func (channel *Channel) SetTag(tag string) {
	channel.Tag = &tag
}

func (channel *Channel) GetAutoBan() bool {
	if channel.AutoBan == nil {
		return false
	}
	return *channel.AutoBan == 1
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

func (channel *Channel) GetWeight() int {
	if channel.Weight == nil {
		return 0
	}
	return int(*channel.Weight)
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	url := *channel.BaseURL
	if url == "" {
		url = constant.ChannelBaseURLs[channel.Type]
	}
	return url
}

func (channel *Channel) GetModelMapping() string {
	if channel.ModelMapping == nil {
		return ""
	}
	return *channel.ModelMapping
}

func (channel *Channel) GetStatusCodeMapping() string {
	if channel.StatusCodeMapping == nil {
		return ""
	}
	return *channel.StatusCodeMapping
}

func (channel *Channel) ValidateSettings() error {
	channelParams := &dto.ChannelSettings{}
	if channel.Setting != nil && *channel.Setting != "" {
		err := common.Unmarshal([]byte(*channel.Setting), channelParams)
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) SetSetting(setting dto.ChannelSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal setting for channel: " + err.Error())
		return
	}
	channel.Setting = common.GetPointer[string](string(settingBytes))
}

func (channel *Channel) SetOtherSettings(setting dto.ChannelOtherSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal other settings for channel: " + err.Error())
		return
	}
	channel.OtherSettings = string(settingBytes)
}

func (channel *Channel) GetParamOverride() map[string]interface{} {
	paramOverride := make(map[string]interface{})
	if channel.ParamOverride != nil && *channel.ParamOverride != "" {
		err := common.Unmarshal([]byte(*channel.ParamOverride), &paramOverride)
		if err != nil {
			common.SysLog("failed to unmarshal param override for channel: " + err.Error())
		}
	}
	return paramOverride
}

func (channel *Channel) GetHeaderOverride() map[string]interface{} {
	headerOverride := make(map[string]interface{})
	if channel.HeaderOverride != nil && *channel.HeaderOverride != "" {
		err := common.Unmarshal([]byte(*channel.HeaderOverride), &headerOverride)
		if err != nil {
			common.SysLog("failed to unmarshal header override for channel: " + err.Error())
		}
	}
	return headerOverride
}

// GetNextEnabledKeyIndex returns (key, keyIndex, error).
// Note: multi-key polling state updates must be handled by the caller (repo layer).
func (channel *Channel) GetNextEnabledKeyIndex(statusList map[int]int) (string, int, *types.NewAPIError) {
	if !channel.ChannelInfo.IsMultiKey {
		return channel.Key, 0, nil
	}

	keys := channel.GetKeys()
	if len(keys) == 0 {
		return "", 0, types.NewError(nil, types.ErrorCodeChannelNoAvailableKey)
	}

	getStatus := func(idx int) int {
		if statusList == nil {
			return common.ChannelStatusEnabled
		}
		if status, ok := statusList[idx]; ok {
			return status
		}
		return common.ChannelStatusEnabled
	}

	enabledIdx := make([]int, 0, len(keys))
	for i := range keys {
		if getStatus(i) == common.ChannelStatusEnabled {
			enabledIdx = append(enabledIdx, i)
		}
	}
	if len(enabledIdx) == 0 {
		return "", 0, types.NewError(nil, types.ErrorCodeChannelNoAvailableKey)
	}

	return keys[enabledIdx[0]], enabledIdx[0], nil
}
