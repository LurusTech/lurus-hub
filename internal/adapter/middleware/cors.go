package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	// Cannot use AllowAllOrigins with AllowCredentials
	// Explicitly list .lurus.cn subdomains for SSO support
	config.AllowOrigins = []string{
		"https://www.lurus.cn",
		"https://gushen.lurus.cn",
		"https://webmail.lurus.cn",
		"http://localhost:5173", // Development
		"http://localhost:3000", // Development
	}
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	return cors.New(config)
}
