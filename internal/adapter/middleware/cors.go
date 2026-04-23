package middleware

import (
	"github.com/LurusTech/lurus-api/internal/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()
	// Load allowed origins from centralized config (env: ALLOWED_ORIGINS)
	corsConfig.AllowOrigins = config.Get().CORS.AllowedOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"*"}
	return cors.New(corsConfig)
}
