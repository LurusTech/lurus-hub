package handler

import (
	"net/http"
	"strconv"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/adapter/middleware"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// V2 Redemption Controllers
// Redemption code management with tenant isolation
// ============================================================================

// RedeemCodeV2 redeems a code for quota
// Route: POST /api/v2/:tenant_slug/redeem
func RedeemCodeV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse request body — accept both "key" (frontend/v1 compat) and "code"
	var req struct {
		Key  string `json:"key"`
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Use "key" if provided, fall back to "code"
	redeemCode := req.Key
	if redeemCode == "" {
		redeemCode = req.Code
	}
	if redeemCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Redemption code is required",
		})
		return
	}

	// Validate code format
	if len(redeemCode) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid redemption code format",
		})
		return
	}

	// Redeem the code
	quota, err := repo.Redeem(redeemCode, tenantCtx.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Code redeemed successfully",
		"data": gin.H{
			"quota_added": quota,
		},
	})
}

// ListRedemptionsV2 lists redemption codes (admin only)
// Route: GET /api/v2/:tenant_slug/redemptions
func ListRedemptionsV2(c *gin.Context) {
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

	// Parse search keyword
	keyword := c.Query("keyword")

	var redemptions []*repo.Redemption
	var total int64

	if keyword != "" {
		// Search redemptions
		redemptions, total, err = repo.SearchRedemptions(keyword, startIdx, pageSize)
	} else {
		// Get all redemptions
		redemptions, total, err = repo.GetAllRedemptions(startIdx, pageSize)
	}

	if err != nil {
		common.SysError("Failed to get redemptions: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve redemptions",
		})
		return
	}

	// Mask redemption keys for security
	for _, r := range redemptions {
		if r.Status == common.RedemptionCodeStatusEnabled {
			// Only mask enabled codes
			r.Key = maskRedemptionKey(r.Key)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"redemptions": redemptions,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

// CreateRedemptionV2 creates new redemption codes (admin only)
// Route: POST /api/v2/:tenant_slug/redemptions
func CreateRedemptionV2(c *gin.Context) {
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
	var req struct {
		Name        string `json:"name" binding:"required"`
		Quota       int    `json:"quota" binding:"required,min=1"`
		Count       int    `json:"count" binding:"required,min=1,max=100"` // Number of codes to generate
		ExpiredTime int64  `json:"expired_time"`                           // 0 means never expires
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Validate name length
	if len(req.Name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Name too long (max 50 characters)",
		})
		return
	}

	// Validate quota value
	maxQuota := int(1000000000 * common.QuotaPerUnit)
	if req.Quota > maxQuota {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Quota value exceeds maximum allowed",
		})
		return
	}

	// Validate expired time
	if req.ExpiredTime != 0 && req.ExpiredTime < common.GetTimestamp() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Expiration time must be in the future",
		})
		return
	}

	// Generate redemption codes
	createdCodes := make([]gin.H, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		key := common.GetRandomString(32)

		redemption := &repo.Redemption{
			UserId:      tenantCtx.UserID,
			TenantId:    tenantCtx.TenantID,
			Key:         key,
			Name:        req.Name,
			Quota:       req.Quota,
			Status:      common.RedemptionCodeStatusEnabled,
			CreatedTime: common.GetTimestamp(),
			ExpiredTime: req.ExpiredTime,
		}

		if err := repo.RedemptionInsert(redemption); err != nil {
			common.SysError("Failed to create redemption: " + err.Error())
			// Continue with remaining codes
			continue
		}

		createdCodes = append(createdCodes, gin.H{
			"id":   redemption.Id,
			"key":  key, // Return full key on creation
			"name": redemption.Name,
		})
	}

	if len(createdCodes) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create any redemption codes",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Redemption codes created successfully",
		"data": gin.H{
			"codes":     createdCodes,
			"count":     len(createdCodes),
			"requested": req.Count,
		},
	})
}

// DeleteRedemptionV2 deletes a redemption code (admin only)
// Route: DELETE /api/v2/:tenant_slug/redemptions/:id
func DeleteRedemptionV2(c *gin.Context) {
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

	// Get redemption ID from URL
	redemptionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid redemption ID",
		})
		return
	}

	// Get existing redemption to verify it exists
	redemption, err := repo.GetRedemptionById(redemptionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Redemption code not found",
		})
		return
	}

	// Verify tenant ownership
	if redemption.TenantId != tenantCtx.TenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied",
		})
		return
	}

	// Delete redemption
	if err := repo.RedemptionDelete(redemption); err != nil {
		common.SysError("Failed to delete redemption: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete redemption code",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Redemption code deleted successfully",
	})
}

// maskRedemptionKey masks a redemption key for display
func maskRedemptionKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "************************" + key[len(key)-4:]
}
