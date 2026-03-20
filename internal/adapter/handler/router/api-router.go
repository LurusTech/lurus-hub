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
	apiRouter.Use(middleware.CORS())
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", handler.GetSetup)
		apiRouter.POST("/setup", handler.PostSetup)
		apiRouter.GET("/status", handler.GetStatus)
		apiRouter.GET("/health", handler.GetHealthDetailed)
		apiRouter.GET("/uptime/status", handler.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.AdminJWTAuth(), handler.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminJWTAuth(), handler.TestStatus)
		apiRouter.GET("/notice", handler.GetNotice)
		apiRouter.GET("/user-agreement", handler.GetUserAgreement)
		apiRouter.GET("/privacy-policy", handler.GetPrivacyPolicy)
		apiRouter.GET("/about", handler.GetAbout)
		apiRouter.GET("/home_page_content", handler.GetHomePageContent)
		apiRouter.GET("/pricing", handler.GetPricing)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), handler.GetRatioConfig)

		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootJWTAuth())
		{
			optionRoute.GET("/", handler.GetOptions)
			optionRoute.PUT("/", handler.UpdateOption)
			optionRoute.POST("/rest_model_ratio", handler.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", handler.MigrateConsoleSetting)
		}

		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootJWTAuth())
		{
			ratioSyncRoute.GET("/channels", handler.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", handler.FetchUpstreamRatios)
		}

		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminJWTAuth())
		{
			channelRoute.GET("/", handler.GetAllChannels)
			channelRoute.GET("/search", handler.SearchChannels)
			channelRoute.GET("/models", handler.ChannelListModels)
			channelRoute.GET("/models_enabled", handler.EnabledListModels)
			channelRoute.GET("/:id", handler.GetChannel)
			channelRoute.GET("/:id/key", middleware.RootJWTAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), handler.GetChannelKey)
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
		tokenRoute.Use(middleware.AdminJWTAuth())
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
		redemptionRoute.Use(middleware.AdminJWTAuth())
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
		logRoute.GET("/", middleware.AdminJWTAuth(), handler.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminJWTAuth(), handler.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminJWTAuth(), handler.GetLogsStat)
		logRoute.GET("/search", middleware.AdminJWTAuth(), handler.SearchAllLogs)
		logRoute.Use(middleware.CORS())
		{
			logRoute.GET("/token", handler.GetLogByKey)
		}

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminJWTAuth(), handler.GetAllQuotaDates)

		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminJWTAuth())
		{
			groupRoute.GET("/", handler.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminJWTAuth())
		{
			prefillGroupRoute.GET("/", handler.GetPrefillGroups)
			prefillGroupRoute.POST("/", handler.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", handler.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", handler.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/", middleware.AdminJWTAuth(), handler.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/", middleware.AdminJWTAuth(), handler.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AdminJWTAuth())
		{
			vendorRoute.GET("/", handler.GetAllVendors)
			vendorRoute.GET("/search", handler.SearchVendors)
			vendorRoute.GET("/:id", handler.GetVendorMeta)
			vendorRoute.POST("/", handler.CreateVendorMeta)
			vendorRoute.PUT("/", handler.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", handler.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AdminJWTAuth())
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

		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.AdminJWTAuth())
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

		apiKeyRoute := apiRouter.Group("/api-keys")
		apiKeyRoute.Use(middleware.AdminJWTAuth())
		{
			apiKeyRoute.GET("/", handler.AdminListApiKeys)
			apiKeyRoute.GET("/scopes", handler.AdminGetApiKeyScopes)
			apiKeyRoute.POST("/", handler.AdminCreateApiKey)
			apiKeyRoute.PUT("/:id", handler.AdminUpdateApiKey)
			apiKeyRoute.DELETE("/:id", handler.AdminDeleteApiKey)
			apiKeyRoute.PUT("/:id/toggle", handler.AdminToggleApiKey)
		}

		releaseRoute := apiRouter.Group("/releases")
		{
			releaseRoute.GET("/", handler.ListReleases)
			releaseRoute.GET("/latest/:product_id", handler.GetLatestRelease)
			releaseRoute.GET("/:id", handler.GetReleaseByID)
			releaseRoute.GET("/:id/changelog", handler.GetChangelog)
			releaseRoute.GET("/:id/download/:artifact_id", middleware.DownloadRateLimit(), handler.DownloadArtifact)
		}

		// User management (admin only)
		userRoute := apiRouter.Group("/user")
		userRoute.Use(middleware.AdminJWTAuth())
		{
			userRoute.GET("/", handler.GetAllUsers)
			userRoute.GET("/search", handler.SearchUsers)
			userRoute.GET("/:id", handler.GetUser)
			userRoute.PUT("/", handler.UpdateUser)
		}
	}
}
