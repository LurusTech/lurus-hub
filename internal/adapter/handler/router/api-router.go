package router

import (
	"github.com/QuantumNous/lurus-api/internal/adapter/handler"
	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.CORS()) // Enable CORS for cross-domain SSO
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", handler.GetSetup)
		apiRouter.POST("/setup", handler.PostSetup)
		apiRouter.GET("/status", handler.GetStatus)
		apiRouter.GET("/uptime/status", handler.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), handler.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), handler.TestStatus)
		apiRouter.GET("/notice", handler.GetNotice)
		apiRouter.GET("/user-agreement", handler.GetUserAgreement)
		apiRouter.GET("/privacy-policy", handler.GetPrivacyPolicy)
		apiRouter.GET("/about", handler.GetAbout)
		//apiRouter.GET("/midjourney", handler.GetMidjourney)
		apiRouter.GET("/home_page_content", handler.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.TryUserAuth(), handler.GetPricing)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), handler.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), handler.ResetPassword)
		apiRouter.GET("/oauth/github", middleware.CriticalRateLimit(), handler.GitHubOAuth)
		apiRouter.GET("/oauth/discord", middleware.CriticalRateLimit(), handler.DiscordOAuth)
		apiRouter.GET("/oauth/oidc", middleware.CriticalRateLimit(), handler.OidcAuth)
		apiRouter.GET("/oauth/linuxdo", middleware.CriticalRateLimit(), handler.LinuxdoOAuth)
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), handler.GenerateOAuthCode)
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), handler.WeChatAuth)
		apiRouter.GET("/oauth/wechat/bind", middleware.CriticalRateLimit(), handler.WeChatBind)
		apiRouter.GET("/oauth/email/bind", middleware.CriticalRateLimit(), handler.EmailBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), handler.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), handler.TelegramBind)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), handler.GetRatioConfig)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), handler.UniversalVerify)
		apiRouter.GET("/verify/status", middleware.UserAuth(), handler.GetVerificationStatus)

		// SMS verification routes
		smsRoute := apiRouter.Group("/sms")
		{
			smsRoute.GET("/status", handler.GetSMSStatus)
			smsRoute.POST("/send", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.SendSmsVerification)
		}

		// Invitation code validation (public)
		apiRouter.GET("/invitation/validate", handler.ValidateInviteCode)

		// Auth/Session endpoints (for SSO support)
		authRoute := apiRouter.Group("/auth")
		{
			authRoute.GET("/session", handler.GetSessionInfo) // Session check for cross-domain SSO
		}

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), handler.Verify2FALogin)
			userRoute.POST("/login_sms", middleware.CriticalRateLimit(), handler.LoginWithSms)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), handler.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), handler.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), handler.TokenLog)
			userRoute.GET("/logout", handler.Logout)
			userRoute.GET("/groups", handler.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", handler.GetUserGroups)
				selfRoute.GET("/self", handler.GetSelf)
				selfRoute.POST("/bind_phone", middleware.CriticalRateLimit(), handler.BindPhone)
				selfRoute.GET("/models", handler.GetUserModels)
				selfRoute.PUT("/self", handler.UpdateSelf)
				selfRoute.DELETE("/self", handler.DeleteSelf)
				selfRoute.GET("/token", handler.GenerateAccessToken)
				selfRoute.GET("/passkey", handler.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", handler.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", handler.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", handler.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", handler.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", handler.PasskeyDelete)
				selfRoute.GET("/aff", handler.GetAffCode)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), handler.TopUp)
				selfRoute.POST("/aff_transfer", handler.TransferAffQuota)
				selfRoute.PUT("/setting", handler.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", handler.Get2FAStatus)
				selfRoute.POST("/2fa/setup", handler.Setup2FA)
				selfRoute.POST("/2fa/enable", handler.Enable2FA)
				selfRoute.POST("/2fa/disable", handler.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", handler.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", handler.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), handler.DoCheckin)

				// User verification status
				selfRoute.GET("/verification-status", middleware.GetUserVerificationStatus)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", handler.GetAllUsers)
				adminRoute.GET("/search", handler.SearchUsers)
				adminRoute.GET("/:id", handler.GetUser)
				adminRoute.POST("/", handler.CreateUser)
				adminRoute.POST("/manage", handler.ManageUser)
				adminRoute.PUT("/", handler.UpdateUser)
				adminRoute.DELETE("/:id", handler.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", handler.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", handler.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", handler.AdminDisable2FA)

				}
		}
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", handler.GetOptions)
			optionRoute.PUT("/", handler.UpdateOption)
			optionRoute.POST("/rest_model_ratio", handler.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", handler.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		// Login configuration management (Root only)
		loginConfigRoute := apiRouter.Group("/admin/login-config")
		loginConfigRoute.Use(middleware.RootAuth())
		{
			loginConfigRoute.GET("/", handler.AdminGetLoginConfig)
			loginConfigRoute.PUT("/", handler.AdminUpdateLoginConfig)
			loginConfigRoute.GET("/modes", handler.AdminGetConfigModes)
		}

		// Invitation code management (Admin)
		invitationRoute := apiRouter.Group("/admin/invitation-codes")
		invitationRoute.Use(middleware.AdminAuth())
		{
			invitationRoute.GET("/", handler.AdminListInviteCodes)
			invitationRoute.GET("/search", handler.AdminSearchInviteCodes)
			invitationRoute.GET("/stats", handler.AdminGetInviteCodeStats)
			invitationRoute.POST("/", handler.AdminCreateInviteCodes)
			invitationRoute.DELETE("/cleanup", handler.AdminCleanupExpiredInviteCodes)
			invitationRoute.DELETE("/:id", handler.AdminDeleteInviteCode)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootAuth())
		{
			ratioSyncRoute.GET("/channels", handler.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", handler.FetchUpstreamRatios)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth())
		{
			channelRoute.GET("/", handler.GetAllChannels)
			channelRoute.GET("/search", handler.SearchChannels)
			channelRoute.GET("/models", handler.ChannelListModels)
			channelRoute.GET("/models_enabled", handler.EnabledListModels)
			channelRoute.GET("/:id", handler.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), handler.GetChannelKey)
			channelRoute.GET("/test", handler.TestAllChannels)
			channelRoute.GET("/test/:id", handler.TestChannel)
			channelRoute.GET("/update_balance", handler.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", handler.UpdateChannelBalance)
			channelRoute.POST("/", handler.AddChannel)
			channelRoute.PUT("/", handler.UpdateChannel)
			channelRoute.DELETE("/disabled", handler.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", handler.DisableTagChannels)
			channelRoute.POST("/tag/enabled", handler.EnableTagChannels)
			channelRoute.PUT("/tag", handler.EditTagChannels)
			channelRoute.DELETE("/:id", handler.DeleteChannel)
			channelRoute.POST("/batch", handler.DeleteChannelBatch)
			channelRoute.POST("/fix", handler.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", handler.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", handler.FetchModels)
			channelRoute.POST("/ollama/pull", handler.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", handler.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", handler.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", handler.OllamaVersion)
			channelRoute.POST("/batch/tag", handler.BatchSetChannelTag)
			channelRoute.GET("/tag/models", handler.GetTagModels)
			channelRoute.POST("/copy/:id", handler.CopyChannel)
			channelRoute.POST("/multi_key/manage", handler.ManageMultiKeys)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", handler.GetAllTokens)
			tokenRoute.GET("/search", handler.SearchTokens)
			tokenRoute.GET("/:id", handler.GetToken)
			tokenRoute.POST("/", handler.AddToken)
			tokenRoute.PUT("/", handler.UpdateToken)
			tokenRoute.DELETE("/:id", handler.DeleteToken)
			tokenRoute.POST("/batch", handler.DeleteTokenBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuth())
			{
				tokenUsageRoute.GET("/", handler.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", handler.GetAllRedemptions)
			redemptionRoute.GET("/search", handler.SearchRedemptions)
			redemptionRoute.GET("/:id", handler.GetRedemption)
			redemptionRoute.POST("/", handler.AddRedemption)
			redemptionRoute.PUT("/", handler.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", handler.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", handler.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), handler.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), handler.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), handler.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), handler.GetLogsSelfStat)
		logRoute.GET("/search", middleware.AdminAuth(), handler.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), handler.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), handler.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), handler.GetAllQuotaDates)
		dataRoute.GET("/self", middleware.UserAuth(), handler.GetUserQuotaDates)

		logRoute.Use(middleware.CORS())
		{
			logRoute.GET("/token", handler.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", handler.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminAuth())
		{
			prefillGroupRoute.GET("/", handler.GetPrefillGroups)
			prefillGroupRoute.POST("/", handler.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", handler.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", handler.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), handler.GetUserMidjourney)
		mjRoute.GET("/", middleware.AdminAuth(), handler.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), handler.GetUserTask)
			taskRoute.GET("/", middleware.AdminAuth(), handler.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AdminAuth())
		{
			vendorRoute.GET("/", handler.GetAllVendors)
			vendorRoute.GET("/search", handler.SearchVendors)
			vendorRoute.GET("/:id", handler.GetVendorMeta)
			vendorRoute.POST("/", handler.CreateVendorMeta)
			vendorRoute.PUT("/", handler.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", handler.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", handler.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", handler.SyncUpstreamModels)
			modelsRoute.POST("/sync_channels", handler.SyncAllChannelsNow)
			modelsRoute.GET("/pricing_info", handler.GetModelsPricingInfo)
			modelsRoute.GET("/missing", handler.GetMissingModels)
			modelsRoute.GET("/", handler.GetAllModelsMeta)
			modelsRoute.GET("/search", handler.SearchModelsMeta)
			modelsRoute.GET("/:id", handler.GetModelMeta)
			modelsRoute.POST("/", handler.CreateModelMeta)
			modelsRoute.PUT("/", handler.UpdateModelMeta)
			modelsRoute.DELETE("/:id", handler.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.AdminAuth())
		{
			deploymentsRoute.GET("/settings", handler.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", handler.TestIoNetConnection)
			deploymentsRoute.GET("/", handler.GetAllDeployments)
			deploymentsRoute.GET("/search", handler.SearchDeployments)
			deploymentsRoute.POST("/test-connection", handler.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", handler.GetHardwareTypes)
			deploymentsRoute.GET("/locations", handler.GetLocations)
			deploymentsRoute.GET("/available-replicas", handler.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", handler.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", handler.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", handler.CreateDeployment)

			deploymentsRoute.GET("/:id", handler.GetDeployment)
			deploymentsRoute.GET("/:id/logs", handler.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", handler.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", handler.GetContainerDetails)
			deploymentsRoute.PUT("/:id", handler.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", handler.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", handler.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", handler.DeleteDeployment)
		}

			// Internal API Key management (admin only)
		apiKeyRoute := apiRouter.Group("/api-keys")
		apiKeyRoute.Use(middleware.AdminAuth())
		{
			apiKeyRoute.GET("/", handler.AdminListApiKeys)
			apiKeyRoute.GET("/scopes", handler.AdminGetApiKeyScopes)
			apiKeyRoute.POST("/", handler.AdminCreateApiKey)
			apiKeyRoute.PUT("/:id", handler.AdminUpdateApiKey)
			apiKeyRoute.DELETE("/:id", handler.AdminDeleteApiKey)
			apiKeyRoute.PUT("/:id/toggle", handler.AdminToggleApiKey)
		}

		// Release/Download management (public, no auth required)
		releaseRoute := apiRouter.Group("/releases")
		{
			releaseRoute.GET("/", handler.ListReleases)
			releaseRoute.GET("/latest/:product_id", handler.GetLatestRelease)
			releaseRoute.GET("/:id", handler.GetReleaseByID)
			releaseRoute.GET("/:id/changelog", handler.GetChangelog)
			releaseRoute.GET("/:id/download/:artifact_id", middleware.DownloadRateLimit(), handler.DownloadArtifact)
		}
	}
}
