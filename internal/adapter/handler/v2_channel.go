package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LurusTech/lurus-api/internal/adapter/middleware"
	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/app/governance"
	"github.com/LurusTech/lurus-api/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// V2 Channel Controllers
// Channels are tenant-level resources managed by tenant admins
// ============================================================================

// ListChannelsV2 retrieves channels for the tenant (admin only)
// Route: GET /api/v2/:tenant_slug/channels
func ListChannelsV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Check admin role
	if !hasRole(tenantCtx.Roles, "admin") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin role required",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	startIdx := (page - 1) * pageSize

	// Parse filter parameters
	keyword := c.Query("keyword")
	group := c.Query("group")
	modelFilter := c.Query("model")
	tag := c.Query("tag")
	idSort := c.DefaultQuery("id_sort", "false") == "true"

	var channels []*repo.Channel
	var total int64

	// Build query based on filters
	if keyword != "" || group != "" || modelFilter != "" {
		// Use search function
		allChannels, searchErr := repo.SearchChannels(keyword, group, modelFilter, idSort)
		if searchErr != nil {
			common.SysError("Failed to search channels: " + searchErr.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to search channels",
			})
			return
		}
		total = int64(len(allChannels))
		// Manual pagination
		end := startIdx + pageSize
		if end > len(allChannels) {
			end = len(allChannels)
		}
		if startIdx < len(allChannels) {
			channels = allChannels[startIdx:end]
		}
	} else if tag != "" {
		// Filter by tag
		allChannels, tagErr := repo.GetChannelsByTag(tag, idSort, false)
		if tagErr != nil {
			common.SysError("Failed to get channels by tag: " + tagErr.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to get channels",
			})
			return
		}
		total = int64(len(allChannels))
		// Manual pagination
		end := startIdx + pageSize
		if end > len(allChannels) {
			end = len(allChannels)
		}
		if startIdx < len(allChannels) {
			channels = allChannels[startIdx:end]
		}
	} else {
		// Get all channels with pagination
		var getAllErr error
		channels, getAllErr = repo.GetAllChannels(startIdx, pageSize, false, idSort)
		if getAllErr != nil {
			common.SysError("Failed to get channels: " + getAllErr.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to get channels",
			})
			return
		}
		total, _ = repo.CountAllChannels()
	}

	// Mask sensitive keys in response
	for _, ch := range channels {
		ch.Key = maskKey(ch.Key)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"channels":  channels,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetChannelV2 retrieves a specific channel (admin only)
// Route: GET /api/v2/:tenant_slug/channels/:id
func GetChannelV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Check admin role
	if !hasRole(tenantCtx.Roles, "admin") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin role required",
		})
		return
	}

	// Get channel ID from URL
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid channel ID",
		})
		return
	}

	// Get channel
	channel, err := repo.GetChannelById(channelID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Channel not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    channel,
	})
}

// CreateChannelV2 creates a new channel (admin only)
// Route: POST /api/v2/:tenant_slug/channels
func CreateChannelV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Check admin role
	if !hasRole(tenantCtx.Roles, "admin") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin role required",
		})
		return
	}

	// Parse request body
	var channel repo.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Validate required fields
	if channel.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Channel name is required",
		})
		return
	}
	if channel.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Channel key is required",
		})
		return
	}
	if channel.Models == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "At least one model is required",
		})
		return
	}

	// Validate name length
	if len(channel.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Channel name too long (max 100 characters)",
		})
		return
	}

	// Set defaults
	channel.CreatedTime = common.GetTimestamp()
	if channel.Status == 0 {
		channel.Status = common.ChannelStatusEnabled
	}
	if channel.Group == "" {
		channel.Group = "default"
	}

	// Set tenant ID from context
	if tenantId, err := repo.GetTenantID(c); err == nil {
		channel.TenantId = tenantId
	} else {
		channel.TenantId = "default"
	}

	// Validate settings if provided
	if err := channel.ValidateSettings(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Insert channel
	if err := channel.Insert(); err != nil {
		common.SysError("Failed to create channel: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create channel",
		})
		return
	}

	// Refresh channel cache
	go repo.InitChannelCache()
	governance.RecordAuditEvent(governance.NewAuditEvent(c, governance.ActorAdmin, tenantCtx.UserID,
		governance.ActionChannelUpdated, governance.ResourceChannel, channel.Id, ""))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Channel created successfully",
		"data": gin.H{
			"id":   channel.Id,
			"name": channel.Name,
		},
	})
}

// UpdateChannelV2 updates a channel (admin only)
// Route: PUT /api/v2/:tenant_slug/channels/:id
func UpdateChannelV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Check admin role
	if !hasRole(tenantCtx.Roles, "admin") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin role required",
		})
		return
	}

	// Get channel ID from URL
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid channel ID",
		})
		return
	}

	// Get existing channel
	existingChannel, err := repo.GetChannelById(channelID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Channel not found",
		})
		return
	}

	// Parse request body
	var updateReq repo.Channel
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Update fields if provided
	if updateReq.Name != "" {
		if len(updateReq.Name) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Channel name too long (max 100 characters)",
			})
			return
		}
		existingChannel.Name = updateReq.Name
	}
	if updateReq.Key != "" {
		existingChannel.Key = updateReq.Key
	}
	if updateReq.Models != "" {
		existingChannel.Models = updateReq.Models
	}
	if updateReq.Group != "" {
		existingChannel.Group = updateReq.Group
	}
	if updateReq.BaseURL != nil {
		existingChannel.BaseURL = updateReq.BaseURL
	}
	if updateReq.Status != 0 {
		existingChannel.Status = updateReq.Status
	}
	if updateReq.Type != 0 {
		existingChannel.Type = updateReq.Type
	}
	if updateReq.Weight != nil {
		existingChannel.Weight = updateReq.Weight
	}
	if updateReq.Priority != nil {
		existingChannel.Priority = updateReq.Priority
	}
	if updateReq.ModelMapping != nil {
		existingChannel.ModelMapping = updateReq.ModelMapping
	}
	if updateReq.Tag != nil {
		existingChannel.Tag = updateReq.Tag
	}
	if updateReq.Remark != nil {
		existingChannel.Remark = updateReq.Remark
	}
	if updateReq.Setting != nil {
		existingChannel.Setting = updateReq.Setting
	}
	if updateReq.ParamOverride != nil {
		existingChannel.ParamOverride = updateReq.ParamOverride
	}
	if updateReq.HeaderOverride != nil {
		existingChannel.HeaderOverride = updateReq.HeaderOverride
	}

	// Validate settings
	if err := existingChannel.ValidateSettings(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Save channel
	if err := existingChannel.Update(); err != nil {
		common.SysError("Failed to update channel: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update channel",
		})
		return
	}

	// Refresh channel cache
	go repo.InitChannelCache()
	governance.RecordAuditEvent(governance.NewAuditEvent(c, governance.ActorAdmin, tenantCtx.UserID,
		governance.ActionChannelUpdated, governance.ResourceChannel, existingChannel.Id, ""))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Channel updated successfully",
		"data": gin.H{
			"id":   existingChannel.Id,
			"name": existingChannel.Name,
		},
	})
}

// DeleteChannelV2 deletes a channel (admin only)
// Route: DELETE /api/v2/:tenant_slug/channels/:id
func DeleteChannelV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Check admin role
	if !hasRole(tenantCtx.Roles, "admin") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin role required",
		})
		return
	}

	// Get channel ID from URL
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid channel ID",
		})
		return
	}

	// Get existing channel to verify it exists
	channel, err := repo.GetChannelById(channelID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Channel not found",
		})
		return
	}

	// Delete channel
	if err := channel.Delete(); err != nil {
		common.SysError("Failed to delete channel: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete channel",
		})
		return
	}

	// Refresh channel cache
	go repo.InitChannelCache()
	governance.RecordAuditEvent(governance.NewAuditEvent(c, governance.ActorAdmin, tenantCtx.UserID,
		governance.ActionChannelDeleted, governance.ResourceChannel, channelID, ""))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Channel deleted successfully",
	})
}

// hasRole checks if the user has a specific role
func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// maskKey masks sensitive API keys for display
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	// If key contains newlines (multi-key), mask each
	if strings.Contains(key, "\n") {
		lines := strings.Split(key, "\n")
		masked := make([]string, len(lines))
		for i, line := range lines {
			masked[i] = maskSingleKey(line)
		}
		return strings.Join(masked, "\n")
	}
	return maskSingleKey(key)
}

// maskSingleKey masks a single API key
func maskSingleKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
