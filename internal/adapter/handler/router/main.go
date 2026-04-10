package router

import (
	"embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/metrics"
	"github.com/QuantumNous/lurus-api/internal/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gin-gonic/gin"
)

func SetRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	// Add OpenTelemetry tracing middleware (must be first to capture full request)
	router.Use(tracing.Middleware())

	// Add Prometheus metrics middleware
	router.Use(metrics.Middleware())

	// Expose /metrics endpoint for Prometheus scraping (restricted to private/loopback IPs)
	router.GET("/metrics", metricsAuthMiddleware(), gin.WrapH(promhttp.Handler()))

	SetApiRouter(router)
	SetApiV2Router(router)  // Multi-tenant v2 API routes
	SetDashboardRouter(router)
	SetRelayRouter(router)
	SetVideoRouter(router)
	SetInternalApiRouter(router)
	frontendBaseUrl := os.Getenv("FRONTEND_BASE_URL")
	if common.IsMasterNode && frontendBaseUrl != "" {
		frontendBaseUrl = ""
		common.SysLog("FRONTEND_BASE_URL is ignored on master node")
	}
	if frontendBaseUrl == "" {
		SetWebRouter(router, buildFS, indexPage)
	} else {
		frontendBaseUrl = strings.TrimSuffix(frontendBaseUrl, "/")
		router.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
		})
	}
}

// metricsAuthMiddleware restricts /metrics to requests from loopback or private
// (RFC 1918 / RFC 4193) source IPs. In-cluster Prometheus scrapes arrive from
// pod/service CIDRs (10.x, 172.x) which are private. Public internet clients
// are rejected with 403.
func metricsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			// RemoteAddr might not have a port (unlikely for TCP, but be safe)
			host = c.Request.RemoteAddr
		}

		ip := net.ParseIP(host)
		if ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
