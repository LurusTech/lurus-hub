package router

import (
	"github.com/QuantumNous/lurus-api/internal/adapter/handler"
	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"

	"github.com/gin-gonic/gin"
)

// SetApiV2Router sets up v2 API routes.
// OAuth login/callback, multi-tenant, and FlexAuth routes have been removed.
// Admin operations use AdminJWTAuth; billing uses ZitadelAuth.
func SetApiV2Router(router *gin.Engine) {
	apiV2 := router.Group("/api/v2")
	{
		// ================================================================
		// Switch Public Routes (no authentication required)
		// ================================================================

		switchGroup := apiV2.Group("/switch")
		{
			switchGroup.GET("/tools/versions", handler.GetToolVersions)
			switchGroup.GET("/presets", handler.ListSwitchPresets)
		}

		apiV2.GET("/tools/download-manifest", handler.GetToolDownloadManifest)

		// ================================================================
		// Platform User Routes (Zitadel JWT auth)
		// ================================================================

		platformUser := apiV2.Group("/user")
		platformUser.Use(middleware.ZitadelAuth())
		{
			platformUser.GET("/identity-overview", handler.GetIdentityOverview)

			billingRoute := platformUser.Group("/billing")
			{
				billingRoute.GET("/summary", handler.GetBillingSummary)
				billingRoute.GET("/payment-methods", handler.GetBillingPaymentMethods)
				billingRoute.POST("/checkout", handler.CreateBillingCheckout)
				billingRoute.GET("/checkout/:order_no/status", handler.GetBillingCheckoutStatus)
			}
		}

		// ================================================================
		// Platform Admin Routes (AdminJWTAuth with root role)
		// ================================================================

		adminRoute := apiV2.Group("/admin")
		adminRoute.Use(middleware.RootJWTAuth())
		{
			tenantMgmt := adminRoute.Group("/tenants")
			{
				tenantMgmt.GET("", handler.ListTenants)
				tenantMgmt.POST("", handler.CreateTenant)
				tenantMgmt.GET("/:id", handler.GetTenant)
				tenantMgmt.PUT("/:id", handler.UpdateTenant)
				tenantMgmt.DELETE("/:id", handler.DeleteTenant)
				tenantMgmt.POST("/:id/enable", handler.EnableTenant)
				tenantMgmt.POST("/:id/disable", handler.DisableTenant)
				tenantMgmt.POST("/:id/suspend", handler.SuspendTenant)
				tenantMgmt.GET("/:id/stats", handler.GetTenantStats)
			}

			mappingRoute := adminRoute.Group("/mappings")
			{
				mappingRoute.GET("", handler.ListUserMappingsV2)
				mappingRoute.GET("/:id", handler.GetUserMappingV2)
				mappingRoute.DELETE("/:id", handler.DeleteUserMappingV2)
			}

			adminRoute.GET("/stats", handler.GetSystemStatsV2)
			adminRoute.POST("/switch/presets", handler.CreateSwitchPreset)

			// Governance (rate-limited: heavy aggregation queries)
			govRoute := adminRoute.Group("/governance")
			govRoute.Use(middleware.CriticalRateLimit())
			{
				govRoute.GET("/channels", handler.GetGovernanceChannelDistribution)
				govRoute.GET("/fingerprints", handler.GetGovernanceFingerprintStats)
				govRoute.GET("/latency", handler.GetGovernanceLatencyStats)
				govRoute.GET("/efficiency", handler.GetGovernanceEfficiencyStats)
			}
			adminRoute.GET("/audit/events", middleware.CriticalRateLimit(), handler.GetAuditEvents)
		}
	}
}
