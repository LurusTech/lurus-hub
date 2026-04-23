package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/LurusTech/lurus-api/internal/domain/entity"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/gin-gonic/gin"
)

// GetToolVersions handles GET /api/v2/switch/tools/versions
// No authentication required. Returns the latest cached tool versions from Redis.
// If Redis is unavailable the handler returns an empty data map (graceful degradation).
func GetToolVersions(c *gin.Context) {
	if !common.RedisEnabled || common.RDB == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    map[string]string{},
		})
		return
	}

	ctx := context.Background()
	raw, err := common.RDB.HGetAll(ctx, toolVersionRedisKey).Result()
	if err != nil || len(raw) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    map[string]string{},
		})
		return
	}

	versions := make(map[string]string, len(raw))
	for tool, jsonStr := range raw {
		var ver entity.ToolVersion
		if err := json.Unmarshal([]byte(jsonStr), &ver); err == nil && ver.Version != "" {
			versions[tool] = ver.Version
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    versions,
	})
}
