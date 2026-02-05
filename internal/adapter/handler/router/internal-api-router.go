package router

import (
	"github.com/QuantumNous/lurus-api/internal/adapter/handler"
	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"
	"github.com/QuantumNous/lurus-api/internal/adapter/repo"

	"github.com/gin-gonic/gin"
)

// SetInternalApiRouter sets up internal API routes for service-to-service communication
// These routes use API Key authentication instead of user session auth
func SetInternalApiRouter(router *gin.Engine) {
	internalGroup := router.Group("/internal")
	internalGroup.Use(middleware.InternalApiAuth())

	// User APIs - query user information
	userGroup := internalGroup.Group("/user")
	userGroup.Use(middleware.RequireScope(repo.ScopeUserRead))
	{
		userGroup.GET("/:id", handler.InternalGetUser)
		userGroup.GET("/by-email/:email", handler.InternalGetUserByEmail)
		userGroup.GET("/by-phone/:phone", handler.InternalGetUserByPhone)
	}

	// User write APIs - modify user information
	userWriteGroup := internalGroup.Group("/user")
	userWriteGroup.Use(middleware.RequireScope(repo.ScopeUserWrite))
	{
		userWriteGroup.PUT("/:id", handler.InternalUpdateUser)
	}

	// Subscription APIs - read subscription information
	subReadGroup := internalGroup.Group("/subscription")
	subReadGroup.Use(middleware.RequireScope(repo.ScopeSubscriptionRead))
	{
		subReadGroup.GET("/user/:id", handler.InternalGetUserSubscription)
	}

	// Subscription APIs - grant subscriptions
	subWriteGroup := internalGroup.Group("/subscription")
	subWriteGroup.Use(middleware.RequireScope(repo.ScopeSubscriptionWrite))
	{
		subWriteGroup.POST("/grant", handler.InternalGrantSubscription)
	}

	// Quota APIs - read user quota
	quotaReadGroup := internalGroup.Group("/quota")
	quotaReadGroup.Use(middleware.RequireScope(repo.ScopeQuotaRead))
	{
		quotaReadGroup.GET("/user/:id", handler.InternalGetUserQuota)
	}

	// Quota APIs - adjust user quota
	quotaWriteGroup := internalGroup.Group("/quota")
	quotaWriteGroup.Use(middleware.RequireScope(repo.ScopeQuotaWrite))
	{
		quotaWriteGroup.POST("/adjust", handler.InternalAdjustQuota)
	}

	// Balance APIs - read user balance
	balanceReadGroup := internalGroup.Group("/balance")
	balanceReadGroup.Use(middleware.RequireScope(repo.ScopeBalanceRead))
	{
		balanceReadGroup.GET("/user/:id", handler.InternalGetUserBalance)
	}

	// Balance APIs - top up user balance
	balanceWriteGroup := internalGroup.Group("/balance")
	balanceWriteGroup.Use(middleware.RequireScope(repo.ScopeBalanceWrite))
	{
		balanceWriteGroup.POST("/topup", handler.InternalTopupBalance)
	}
}
