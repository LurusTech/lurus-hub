package router

import (
	"github.com/QuantumNous/lurus-api/internal/adapter/handler"
	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"

	"github.com/gin-gonic/gin"
)

// SetApiV2Router sets up v2 API routes with multi-tenant support
// All v2 routes use Zitadel OAuth authentication
func SetApiV2Router(router *gin.Engine) {
	// V2 API group
	apiV2 := router.Group("/api/v2")
	{
		// ================================================================
		// OAuth Authentication Routes (No authentication required)
		// OAuth 认证路由（无需认证）
		// ================================================================

		// OAuth login redirect - redirects to Zitadel login page
		// OAuth 登录跳转 - 跳转到 Zitadel 登录页面
		apiV2.GET("/:tenant_slug/auth/login", handler.ZitadelLoginRedirect)

		// OAuth callback - handles Zitadel OAuth callback
		// OAuth 回调 - 处理 Zitadel OAuth 回调
		apiV2.GET("/oauth/callback", handler.ZitadelCallback)

		// Session info - returns current user data for frontend after OAuth callback
		// Session 信息 - OAuth 回调后返回当前用户数据供前端使用
		apiV2.GET("/auth/session-info", handler.GetSessionInfo)

		// OAuth logout - logs out from Zitadel
		// OAuth 登出 - 从 Zitadel 登出
		apiV2.POST("/oauth/logout", handler.ZitadelLogout)

		// OAuth token refresh - refreshes access token
		// OAuth Token 刷新 - 刷新访问令牌
		apiV2.POST("/oauth/refresh", handler.RefreshAccessToken)

		// ================================================================
		// Tenant-Specific Routes (Require Zitadel JWT authentication)
		// 租户路由（需要 Zitadel JWT 认证）
		// ================================================================

		tenantRoute := apiV2.Group("/:tenant_slug")
		tenantRoute.Use(middleware.ZitadelAuth()) // Zitadel JWT verification
		{
			// User routes
			// 用户路由
			tenantRoute.GET("/user/me", handler.GetSelfV2)
			tenantRoute.PUT("/user/me", handler.UpdateSelfV2)
			// TODO: Add more user routes

			// Channel routes (Admin only)
			// 渠道路由（仅管理员）
			channelRoute := tenantRoute.Group("/channels")
			{
				channelRoute.GET("", handler.ListChannelsV2)
				channelRoute.GET("/:id", handler.GetChannelV2)

				// Admin-only channel management
				channelRoute.POST("", middleware.RequireRole("admin"), handler.CreateChannelV2)
				channelRoute.PUT("/:id", middleware.RequireRole("admin"), handler.UpdateChannelV2)
				channelRoute.DELETE("/:id", middleware.RequireRole("admin"), handler.DeleteChannelV2)
			}

			// Token (API key) routes
			// Token（API密钥）路由
			tokenRoute := tenantRoute.Group("/tokens")
			{
				tokenRoute.GET("", handler.ListTokensV2)
				tokenRoute.POST("", handler.CreateTokenV2)
				tokenRoute.PUT("/:id", handler.UpdateTokenV2)
				tokenRoute.DELETE("/:id", handler.DeleteTokenV2)
			}

			// Log routes
			// 日志路由
			logRoute := tenantRoute.Group("/logs")
			{
				logRoute.GET("", handler.GetLogsV2)
				// Admin can view all users' logs
				logRoute.GET("/all", middleware.RequireRole("admin"), handler.GetAllLogsV2)
			}

			// Tenant configuration routes (Admin only)
			// 租户配置路由（仅管理员）
			configRoute := tenantRoute.Group("/config")
			configRoute.Use(middleware.RequireRole("admin"))
			{
				configRoute.GET("", handler.GetTenantConfigs)
				configRoute.PUT("/:key", handler.UpdateTenantConfig)
			}

			// Billing routes (wallet-to-quota transfer)
			// 计费路由（钱包转配额）
			billingRoute := tenantRoute.Group("/billing")
			{
				billingRoute.GET("/topups", handler.GetTopUpsV2)
				billingRoute.POST("/topup", middleware.TopupRateLimit(), handler.TopUpV2)
			}

			// Redemption code routes
			// 兑换码路由
			redemptionRoute := tenantRoute.Group("/redemptions")
			{
				// Users can redeem codes (rate limited: 5 attempts/min per IP to prevent brute-force)
				redemptionRoute.POST("/redeem", middleware.RedemptionRateLimit(), handler.RedeemCodeV2)

				// Admin can manage redemption codes
				redemptionRoute.GET("", middleware.RequireRole("admin"), handler.ListRedemptionsV2)
				redemptionRoute.POST("", middleware.RequireRole("admin"), handler.CreateRedemptionV2)
				redemptionRoute.DELETE("/:id", middleware.RequireRole("admin"), handler.DeleteRedemptionV2)
			}
		}

		// ================================================================
		// Switch Public Routes (no authentication required)
		// lurus-switch 公共路由（无需认证）
		// ================================================================

		switchGroup := apiV2.Group("/switch")
		{
			// Tool version endpoint — polled by lurus-switch to check for updates.
			switchGroup.GET("/tools/versions", handler.GetToolVersions)

			// Config preset library — read-only for clients.
			switchGroup.GET("/presets", handler.ListSwitchPresets)
		}

		// Tool download manifest — platform-aware binary/npm download links.
		// No authentication required; cached by CDN/proxy for 1 hour.
		apiV2.GET("/tools/download-manifest", handler.GetToolDownloadManifest)

		// ================================================================
		// Client API (FlexAuth: Zitadel JWT or API Token sk-xxx)
		// 客户端 API — 供其他 Lurus 产品查询用户数据
		// ================================================================

		clientRoute := apiV2.Group("/client")
		clientRoute.Use(middleware.FlexAuth())
		{
			clientRoute.GET("/profile", handler.ClientGetProfile)
			clientRoute.GET("/tokens", handler.ClientGetTokens)
			clientRoute.GET("/sessions", handler.ClientGetSessions)

			clientUsage := clientRoute.Group("/usage")
			{
				clientUsage.GET("/summary", handler.ClientGetUsageSummary)
				clientUsage.GET("/models", handler.ClientGetUsageByModel)
				clientUsage.GET("/daily", handler.ClientGetUsageDaily)
			}
		}

		// ================================================================
		// Platform-wide User Routes (Zitadel JWT auth, no tenant context)
		// 平台用户路由（Zitadel JWT 认证，无需 tenant context）
		// ================================================================

		platformUser := apiV2.Group("/user")
		platformUser.Use(middleware.ZitadelAuth())
		{
			// Identity overview — returns VIP level, Lubell wallet balance and subscription status.
			platformUser.GET("/identity-overview", handler.GetIdentityOverview)

			// Billing — unified wallet topup via lurus-platform
			billingRoute := platformUser.Group("/billing")
			{
				billingRoute.GET("/summary", handler.GetBillingSummary)
				billingRoute.GET("/payment-methods", handler.GetBillingPaymentMethods)
				billingRoute.POST("/checkout", handler.CreateBillingCheckout)
				billingRoute.GET("/checkout/:order_no/status", handler.GetBillingCheckoutStatus)
			}
		}

		// ================================================================
		// Platform Admin Routes (System-level, requires Platform Admin role)
		// 平台管理员路由（系统级，需要平台管理员角色）
		// ================================================================

		adminRoute := apiV2.Group("/admin")
		// Note: For Platform Admin routes, we use v1 authentication (session-based)
		// since Platform Admins may manage multiple tenants
		// 注意：平台管理员路由使用 v1 认证（基于 session）
		// 因为平台管理员需要管理多个租户
		adminRoute.Use(middleware.UserAuth(), middleware.RootAuth())
		{
			// Tenant management
			// 租户管理
			tenantMgmt := adminRoute.Group("/tenants")
			{
				tenantMgmt.GET("", handler.ListTenants)
				tenantMgmt.POST("", handler.CreateTenant)
				tenantMgmt.GET("/:id", handler.GetTenant)
				tenantMgmt.PUT("/:id", handler.UpdateTenant)
				tenantMgmt.DELETE("/:id", handler.DeleteTenant)

				// Tenant status management
				tenantMgmt.POST("/:id/enable", handler.EnableTenant)
				tenantMgmt.POST("/:id/disable", handler.DisableTenant)
				tenantMgmt.POST("/:id/suspend", handler.SuspendTenant)

				// Tenant statistics
				tenantMgmt.GET("/:id/stats", handler.GetTenantStats)
			}

			// User identity mapping management (Platform Admin)
			// 用户身份映射管理（平台管理员）
			mappingRoute := adminRoute.Group("/mappings")
			{
				mappingRoute.GET("", handler.ListUserMappingsV2)
				mappingRoute.GET("/:id", handler.GetUserMappingV2)
				mappingRoute.DELETE("/:id", handler.DeleteUserMappingV2)
			}

			// System-wide statistics (Platform Admin)
			// 系统级统计（平台管理员）
			adminRoute.GET("/stats", handler.GetSystemStatsV2)

			// Switch preset management (Platform Admin)
			// switch 配置预设管理（平台管理员）
			adminRoute.POST("/switch/presets", handler.CreateSwitchPreset)
		}
	}
}
