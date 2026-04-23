package handler

import (
	"net/http"

	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/constant"
	"github.com/LurusTech/lurus-api/internal/adapter/middleware"
	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/pkg/setting"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/console_setting"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/operation_setting"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func TestStatus(c *gin.Context) {
	err := repo.PingDB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "数据库连接失败",
		})
		return
	}
	// 获取HTTP统计信息
	httpStats := middleware.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Server is running",
		"http_stats": httpStats,
	})
	return
}

func GetStatus(c *gin.Context) {

	cs := console_setting.GetConsoleSetting()
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	passkeySetting := system_setting.GetPasskeySettings()
	legalSetting := system_setting.GetLegalSettings()

	// Build login methods configuration for frontend
	loginMethods := gin.H{
		"password": gin.H{
			"enabled":              common.PasswordLoginEnabled,
			"registration_enabled": common.PasswordRegisterEnabled,
		},
		"sms": gin.H{
			"enabled":       common.SMSEnabled,
			"auto_register": common.SMSAutoRegister,
			"bind_enabled":  common.SMSEnabled,
		},
		"github": gin.H{
			"enabled":   common.GitHubOAuthEnabled,
			"client_id": common.GitHubClientId,
		},
		"discord": gin.H{
			"enabled":   system_setting.GetDiscordSettings().Enabled,
			"client_id": system_setting.GetDiscordSettings().ClientId,
		},
		"linuxdo": gin.H{
			"enabled":             common.LinuxDOOAuthEnabled,
			"client_id":           common.LinuxDOClientId,
			"minimum_trust_level": common.LinuxDOMinimumTrustLevel,
		},
		"telegram": gin.H{
			"enabled":  common.TelegramOAuthEnabled,
			"bot_name": common.TelegramBotName,
		},
		"wechat": gin.H{
			"enabled": common.WeChatAuthEnabled,
			"qrcode":  common.WeChatAccountQRCodeImageURL,
		},
		"oidc": gin.H{
			"enabled":                system_setting.GetOIDCSettings().Enabled,
			"client_id":              system_setting.GetOIDCSettings().ClientId,
			"authorization_endpoint": system_setting.GetOIDCSettings().AuthorizationEndpoint,
		},
		"passkey": gin.H{
			"enabled":           passkeySetting.Enabled,
			"display_name":      passkeySetting.RPDisplayName,
			"rp_id":             passkeySetting.RPID,
			"user_verification": passkeySetting.UserVerification,
		},
	}

	// Build registration configuration
	registration := gin.H{
		"mode":                        common.RegistrationMode,
		"enabled":                     common.RegisterEnabled,
		"email_verification_required": common.EmailVerificationEnabled,
		"phone_verification_required": common.RegistrationMode == common.RegistrationModePhoneVerified,
		"invite_code_required":        common.InviteCodeRequired || common.RegistrationMode == common.RegistrationModeInviteOnly,
		"turnstile_required":          common.TurnstileCheckEnabled,
	}

	// Build security configuration
	security := gin.H{
		"2fa_available":     true,
		"passkey_available": passkeySetting.Enabled,
	}

	data := gin.H{
		"version":                     common.Version,
		"start_time":                  common.StartTime,
		"email_verification":          common.EmailVerificationEnabled,
		"github_oauth":                common.GitHubOAuthEnabled,
		"github_client_id":            common.GitHubClientId,
		"discord_oauth":               system_setting.GetDiscordSettings().Enabled,
		"discord_client_id":           system_setting.GetDiscordSettings().ClientId,
		"linuxdo_oauth":               common.LinuxDOOAuthEnabled,
		"linuxdo_client_id":           common.LinuxDOClientId,
		"linuxdo_minimum_trust_level": common.LinuxDOMinimumTrustLevel,
		"telegram_oauth":              common.TelegramOAuthEnabled,
		"telegram_bot_name":           common.TelegramBotName,
		"system_name":                 common.SystemName,
		"logo":                        common.Logo,
		"footer_html":                 common.Footer,
		"wechat_qrcode":               common.WeChatAccountQRCodeImageURL,
		"wechat_login":                common.WeChatAuthEnabled,
		"server_address":              system_setting.ServerAddress,
		"turnstile_check":             common.TurnstileCheckEnabled,
		"turnstile_site_key":          common.TurnstileSiteKey,
		"docs_link":                   operation_setting.GetGeneralSetting().DocsLink,
		"quota_per_unit":              common.QuotaPerUnit,
		// 兼容旧前端：保留 display_in_currency，同时提供新的 quota_display_type
		"display_in_currency":           operation_setting.IsCurrencyDisplay(),
		"quota_display_type":            operation_setting.GetQuotaDisplayType(),
		"custom_currency_symbol":        operation_setting.GetGeneralSetting().CustomCurrencySymbol,
		"custom_currency_exchange_rate": operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate,
		"enable_batch_update":           common.BatchUpdateEnabled,
		"enable_drawing":                common.DrawingEnabled,
		"enable_task":                   common.TaskEnabled,
		"enable_data_export":            common.DataExportEnabled,
		"data_export_default_time":      common.DataExportDefaultTime,
		"default_collapse_sidebar":      common.DefaultCollapseSidebar,
		"mj_notify_enabled":             setting.MjNotifyEnabled,
		"chats":                         setting.Chats,
		"demo_site_enabled":             operation_setting.DemoSiteEnabled,
		"self_use_mode_enabled":         operation_setting.SelfUseModeEnabled,
		"default_use_auto_group":        setting.DefaultUseAutoGroup,

		"usd_exchange_rate": operation_setting.USDExchangeRate,
		"price":             operation_setting.Price,

		// 面板启用开关
		"api_info_enabled":      cs.ApiInfoEnabled,
		"uptime_kuma_enabled":   cs.UptimeKumaEnabled,
		"announcements_enabled": cs.AnnouncementsEnabled,
		"faq_enabled":           cs.FAQEnabled,

		// 模块管理配置
		"HeaderNavModules":    common.OptionMap["HeaderNavModules"],
		"SidebarModulesAdmin": common.OptionMap["SidebarModulesAdmin"],

		"oidc_enabled":                system_setting.GetOIDCSettings().Enabled,
		"oidc_client_id":              system_setting.GetOIDCSettings().ClientId,
		"oidc_authorization_endpoint": system_setting.GetOIDCSettings().AuthorizationEndpoint,
		"passkey_login":               passkeySetting.Enabled,
		"passkey_display_name":        passkeySetting.RPDisplayName,
		"passkey_rp_id":               passkeySetting.RPID,
		"passkey_origins":             passkeySetting.Origins,
		"passkey_allow_insecure":      passkeySetting.AllowInsecureOrigin,
		"passkey_user_verification":   passkeySetting.UserVerification,
		"passkey_attachment":          passkeySetting.AttachmentPreference,
		"setup":                       constant.IsSetup(),
		"user_agreement_enabled":      legalSetting.UserAgreement != "",
		"privacy_policy_enabled":      legalSetting.PrivacyPolicy != "",
		"checkin_enabled":             operation_setting.GetCheckinSetting().Enabled,

		// New: Frontend login configuration
		"login_methods": loginMethods,
		"registration":  registration,
		"security":           security,
		"sms_enabled":        common.SMSEnabled,
	}

	// 根据启用状态注入可选内容
	if cs.ApiInfoEnabled {
		data["api_info"] = console_setting.GetApiInfo()
	}
	if cs.AnnouncementsEnabled {
		data["announcements"] = console_setting.GetAnnouncements()
	}
	if cs.FAQEnabled {
		data["faq"] = console_setting.GetFAQ()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
	return
}

func GetNotice(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Notice"],
	})
	return
}

func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["About"],
	})
	return
}

func GetUserAgreement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().UserAgreement,
	})
	return
}

func GetPrivacyPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().PrivacyPolicy,
	})
	return
}

func GetMidjourney(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Midjourney"],
	})
	return
}

func GetHomePageContent(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["HomePageContent"],
	})
	return
}

