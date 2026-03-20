package handler

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/pkg/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// InternalGetUserLogs returns paginated usage logs for a specific user.
// GET /internal/log/user/:id?page=1&per_page=20
func InternalGetUserLogs(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	logs, total, err := repo.GetUserLogsInternal(userID, (page-1)*perPage, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query logs failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// InternalGetUserLogStat returns aggregated usage statistics for a user.
// GET /internal/log/user/:id/stat?group_by=model|day
func InternalGetUserLogStat(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	groupBy := c.DefaultQuery("group_by", "model")
	if groupBy != "model" && groupBy != "day" {
		groupBy = "model"
	}
	stats, err := repo.GetUserLogStatInternal(userID, groupBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query stats failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// InternalGetTokenLogs returns usage logs filtered by token ID.
// GET /internal/log/token/:token_id?page=1&per_page=20
func InternalGetTokenLogs(c *gin.Context) {
	tokenID, err := strconv.Atoi(c.Param("token_id"))
	if err != nil || tokenID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	logs, total, err := repo.GetTokenLogsInternal(tokenID, (page-1)*perPage, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query logs failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// ModelCatalogEntry represents a model with its pricing info.
type ModelCatalogEntry struct {
	ID         string  `json:"id"`
	ModelRatio float64 `json:"model_ratio"`
	GroupRatio float64 `json:"group_ratio"`
	Available  bool    `json:"available"`
}

// InternalGetModelCatalog returns the available model catalog with pricing.
// GET /internal/models/catalog?group=default
func InternalGetModelCatalog(c *gin.Context) {
	group := c.DefaultQuery("group", "default")
	groupRatio := ratio_setting.GetGroupRatio(group)

	models := ratio_setting.GetDefaultModelRatioMap()
	catalog := make([]ModelCatalogEntry, 0, len(models))
	for name, ratio := range models {
		catalog = append(catalog, ModelCatalogEntry{
			ID:         name,
			ModelRatio: ratio,
			GroupRatio: groupRatio,
			Available:  true,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": catalog, "group": group, "group_ratio": groupRatio})
}
