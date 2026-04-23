package handler

import (
	"net/http"
	"strconv"

	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/adapter/middleware"

	"github.com/gin-gonic/gin"
)

// GetLogsV2 retrieves the current user's logs (v2 API with tenant context)
// Route: GET /api/v2/:tenant_slug/logs
func GetLogsV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse pagination and filter parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	logType, _ := strconv.Atoi(c.DefaultQuery("type", "0"))
	modelName := c.Query("model_name")
	startTime, _ := strconv.ParseInt(c.DefaultQuery("start_time", "0"), 10, 64)
	endTime, _ := strconv.ParseInt(c.DefaultQuery("end_time", "0"), 10, 64)
	tokenName := c.Query("token_name")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build log query params
	params := &repo.LogQueryParams{
		UserID:     tenantCtx.UserID,
		TenantID:   tenantCtx.TenantID,
		LogType:    logType,
		ModelName:  modelName,
		StartTime:  startTime,
		EndTime:    endTime,
		TokenName:  tokenName,
		Offset:     offset,
		Limit:      pageSize,
	}

	// Get logs
	logs, total, err := repo.GetUserLogsWithParams(params)
	if err != nil {
		common.SysError("Failed to get logs: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetAllLogsV2 retrieves all logs for the tenant (admin only, v2 API)
// Route: GET /api/v2/:tenant_slug/logs/all
func GetAllLogsV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse pagination and filter parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	logType, _ := strconv.Atoi(c.DefaultQuery("type", "0"))
	modelName := c.Query("model_name")
	startTime, _ := strconv.ParseInt(c.DefaultQuery("start_time", "0"), 10, 64)
	endTime, _ := strconv.ParseInt(c.DefaultQuery("end_time", "0"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build log query params (no user filter for all logs)
	params := &repo.LogQueryParams{
		TenantID:   tenantCtx.TenantID,
		LogType:    logType,
		ModelName:  modelName,
		StartTime:  startTime,
		EndTime:    endTime,
		TokenName:  tokenName,
		Username:   username,
		Offset:     offset,
		Limit:      pageSize,
	}

	// Get all logs for tenant
	logs, total, err := repo.GetTenantLogsWithParams(params)
	if err != nil {
		common.SysError("Failed to get logs: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
