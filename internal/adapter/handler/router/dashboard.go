package router

import (
	"github.com/LurusTech/lurus-api/internal/adapter/handler"
	"github.com/LurusTech/lurus-api/internal/adapter/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetDashboardRouter(router *gin.Engine) {
	apiRouter := router.Group("/")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	apiRouter.Use(middleware.CORS())
	apiRouter.Use(middleware.TokenAuth())
	{
		apiRouter.GET("/dashboard/billing/subscription", handler.GetSubscription)
		apiRouter.GET("/v1/dashboard/billing/subscription", handler.GetSubscription)
		apiRouter.GET("/dashboard/billing/usage", handler.GetUsage)
		apiRouter.GET("/v1/dashboard/billing/usage", handler.GetUsage)
	}
}
