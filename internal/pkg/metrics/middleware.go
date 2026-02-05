package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware that records request metrics
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Track active connections
		ActiveConnections.Inc()
		defer ActiveConnections.Dec()

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
