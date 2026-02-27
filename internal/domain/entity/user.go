package entity

import (
	"encoding/json"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/dto"

	"gorm.io/gorm"
)

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id               int            `json:"id"`
	TenantId         string         `json:"tenant_id" gorm:"type:varchar(36);index;default:'default'"` // Tenant isolation
	Username         string         `json:"username" gorm:"unique;index" validate:"max=20"`
	Password         string         `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	OriginalPassword string         `json:"original_password" gorm:"-:all"` // Password change verification only
	DisplayName      string         `json:"display_name" gorm:"index" validate:"max=20"`
	Role             int            `json:"role" gorm:"type:int;default:1"`   // admin, common
	Status           int            `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email            string         `json:"email" gorm:"index" validate:"max=50"`
	GitHubId         string         `json:"github_id" gorm:"column:github_id;index"`
	DiscordId        string         `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId           string         `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId         string         `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId       string         `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode string         `json:"verification_code" gorm:"-:all"`                                    // Email verification only
	AccessToken      *string        `json:"access_token" gorm:"type:char(32);column:access_token;uniqueIndex"` // system management token
	Quota            int            `json:"quota" gorm:"type:int;default:0"`
	UsedQuota        int            `json:"used_quota" gorm:"type:int;default:0;column:used_quota"`
	RequestCount     int            `json:"request_count" gorm:"type:int;default:0;"`
	Group            string         `json:"group" gorm:"type:varchar(64);default:'default'"`
	DailyQuota       int            `json:"daily_quota" gorm:"type:int;default:0;column:daily_quota"`
	DailyUsed        int            `json:"daily_used" gorm:"type:int;default:0;column:daily_used"`
	LastDailyReset   int64          `json:"last_daily_reset" gorm:"type:bigint;default:0;column:last_daily_reset"`
	BaseGroup        string         `json:"base_group" gorm:"type:varchar(64);column:base_group"`
	FallbackGroup    string         `json:"fallback_group" gorm:"type:varchar(64);column:fallback_group"`
	AffCode          string         `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount         int            `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota         int            `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`
	AffHistoryQuota  int            `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"`
	InviterId        int            `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	LinuxDOId        string         `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	Phone            string         `json:"phone" gorm:"column:phone;index"`
	PhoneVerified    bool           `json:"phone_verified" gorm:"column:phone_verified;default:false"`
	Setting          string         `json:"setting" gorm:"type:text;column:setting"`
	Remark           string         `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
}

func (user *User) ToBaseUser() *UserBase {
	return &UserBase{
		Id:             user.Id,
		Group:          user.Group,
		Quota:          user.Quota,
		Status:         user.Status,
		Username:       user.Username,
		Setting:        user.Setting,
		Email:          user.Email,
		DailyQuota:     user.DailyQuota,
		DailyUsed:      user.DailyUsed,
		LastDailyReset: user.LastDailyReset,
		BaseGroup:      user.BaseGroup,
		FallbackGroup:  user.FallbackGroup,
	}
}

func (user *User) GetAccessToken() string {
	if user.AccessToken == nil {
		return ""
	}
	return *user.AccessToken
}

func (user *User) SetAccessToken(token string) {
	user.AccessToken = &token
}

func (user *User) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := json.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

func (user *User) SetSetting(setting dto.UserSetting) {
	settingBytes, err := json.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal setting: " + err.Error())
		return
	}
	user.Setting = string(settingBytes)
}

// IsSubscriber checks if user has subscriber role or higher
func (user *User) IsSubscriber() bool {
	return user.Role >= common.RoleSubscriberUser
}

// UserBase is a lightweight view of User for caching
type UserBase struct {
	Id             int    `json:"id"`
	TenantId       string `json:"tenant_id"`
	Group          string `json:"group"`
	Email          string `json:"email"`
	Quota          int    `json:"quota"`
	Status         int    `json:"status"`
	Username       string `json:"username"`
	Setting        string `json:"setting"`
	DailyQuota     int    `json:"daily_quota"`
	DailyUsed      int    `json:"daily_used"`
	LastDailyReset int64  `json:"last_daily_reset"`
	BaseGroup      string `json:"base_group"`
	FallbackGroup  string `json:"fallback_group"`
}

func (user *UserBase) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := json.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

// DailyQuotaInfo represents daily quota status for a user
type DailyQuotaInfo struct {
	UserID          int    `json:"user_id"`
	DailyQuota      int    `json:"daily_quota"`
	DailyUsed       int    `json:"daily_used"`
	DailyRemaining  int    `json:"daily_remaining"`
	LastDailyReset  int64  `json:"last_daily_reset"`
	BaseGroup       string `json:"base_group"`
	FallbackGroup   string `json:"fallback_group"`
	CurrentGroup    string `json:"current_group"`
	IsUsingFallback bool   `json:"is_using_fallback"`
	NeedsReset      bool   `json:"needs_reset"`
}

// NeedsDailyReset checks if daily quota needs to be reset based on last reset timestamp
func NeedsDailyReset(lastResetTimestamp int64) bool {
	if lastResetTimestamp == 0 {
		return true
	}
	now := common.GetTimestamp()
	nowDay := now / 86400
	lastResetDay := lastResetTimestamp / 86400
	return nowDay > lastResetDay
}
