package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// V2 Billing Controllers
// TopUp and Subscription management with tenant isolation
// ============================================================================

// GetTopUpsV2 retrieves the current user's topup history
// Route: GET /api/v2/:tenant_slug/billing/topups
func GetTopUpsV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
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

	pageInfo := &common.PageInfo{
		Page:     page,
		PageSize: pageSize,
	}

	// Get topups for user
	topups, total, err := repo.GetUserTopUps(tenantCtx.UserID, pageInfo)
	if err != nil {
		common.SysError("Failed to get topups: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve topup history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"topups":    topups,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// TopUpV2 initiates a new topup (creates pending order)
// Route: POST /api/v2/:tenant_slug/billing/topup
func TopUpV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse request body
	var req struct {
		Amount        int64   `json:"amount" binding:"required,min=1"`        // Amount in cents
		PaymentMethod string  `json:"payment_method" binding:"required"`      // stripe/epay/creem
		Money         float64 `json:"money" binding:"required,gt=0"`          // Money in CNY/USD
		Currency      string  `json:"currency"`                               // CNY/USD
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Validate payment method
	validMethods := map[string]bool{"stripe": true, "epay": true, "creem": true}
	if !validMethods[req.PaymentMethod] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid payment method. Supported: stripe, epay, creem",
		})
		return
	}

	// Validate amount limits
	if req.Amount > 10000000 { // Max 100,000 CNY
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Amount exceeds maximum limit",
		})
		return
	}

	// Generate unique trade number
	tradeNo := common.GetRandomString(32)

	// Create topup record
	topup := &repo.TopUp{
		UserId:        tenantCtx.UserID,
		TenantId:      tenantCtx.TenantID,
		Amount:        req.Amount,
		Money:         req.Money,
		TradeNo:       tradeNo,
		PaymentMethod: req.PaymentMethod,
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}

	if err := repo.TopUpInsert(topup); err != nil {
		common.SysError("Failed to create topup: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create topup order",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Topup order created",
		"data": gin.H{
			"id":             topup.Id,
			"trade_no":       topup.TradeNo,
			"amount":         topup.Amount,
			"money":          topup.Money,
			"payment_method": topup.PaymentMethod,
			"status":         topup.Status,
		},
	})
}

// GetSubscriptionsV2 retrieves the current user's subscription history
// Route: GET /api/v2/:tenant_slug/billing/subscriptions
func GetSubscriptionsV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse limit parameter
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get subscriptions for user
	subscriptions, err := repo.GetUserSubscriptions(tenantCtx.UserID, limit)
	if err != nil {
		common.SysError("Failed to get subscriptions: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve subscriptions",
		})
		return
	}

	// Get active subscription
	activeSub, _ := repo.GetActiveSubscription(tenantCtx.UserID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subscriptions": subscriptions,
			"active":        activeSub,
		},
	})
}

// SubscribeV2 creates a new subscription (pending payment)
// Route: POST /api/v2/:tenant_slug/billing/subscribe
func SubscribeV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Parse request body
	var req struct {
		PlanCode      string  `json:"plan_code" binding:"required"`       // weekly/monthly/quarterly/yearly
		PaymentMethod string  `json:"payment_method" binding:"required"`  // stripe/epay/creem
		AutoRenew     bool    `json:"auto_renew"`                         // Auto renew subscription
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Validate plan code
	validPlans := map[string]struct {
		Name     string
		Duration time.Duration
		Amount   float64
	}{
		"weekly":    {"Weekly Plan", 7 * 24 * time.Hour, 29.0},
		"monthly":   {"Monthly Plan", 30 * 24 * time.Hour, 99.0},
		"quarterly": {"Quarterly Plan", 90 * 24 * time.Hour, 269.0},
		"yearly":    {"Yearly Plan", 365 * 24 * time.Hour, 999.0},
	}

	plan, ok := validPlans[req.PlanCode]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid plan code. Supported: weekly, monthly, quarterly, yearly",
		})
		return
	}

	// Validate payment method
	validMethods := map[string]bool{"stripe": true, "epay": true, "creem": true}
	if !validMethods[req.PaymentMethod] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid payment method. Supported: stripe, epay, creem",
		})
		return
	}

	// Check if user already has an active subscription
	activeSub, _ := repo.GetActiveSubscription(tenantCtx.UserID)
	if activeSub != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "You already have an active subscription",
			"data": gin.H{
				"current_plan": activeSub.PlanCode,
				"expires_at":   activeSub.ExpiresAt,
			},
		})
		return
	}

	// Create subscription record
	now := time.Now()
	subscription := &repo.Subscription{
		UserId:        tenantCtx.UserID,
		TenantId:      tenantCtx.TenantID,
		PlanCode:      req.PlanCode,
		PlanName:      plan.Name,
		Status:        repo.SubscriptionStatusPending,
		StartedAt:     now,
		ExpiresAt:     now.Add(plan.Duration),
		PaymentMethod: req.PaymentMethod,
		Amount:        plan.Amount,
		Currency:      "CNY",
		AutoRenew:     req.AutoRenew,
	}

	if err := repo.CreateSubscription(subscription); err != nil {
		common.SysError("Failed to create subscription: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create subscription",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Subscription created (pending payment)",
		"data": gin.H{
			"id":             subscription.Id,
			"plan_code":      subscription.PlanCode,
			"plan_name":      subscription.PlanName,
			"amount":         subscription.Amount,
			"currency":       subscription.Currency,
			"payment_method": subscription.PaymentMethod,
			"expires_at":     subscription.ExpiresAt,
			"status":         subscription.Status,
		},
	})
}

// CancelSubscriptionV2 cancels an active subscription
// Route: POST /api/v2/:tenant_slug/billing/subscriptions/:id/cancel
func CancelSubscriptionV2(c *gin.Context) {
	// Get tenant context from middleware
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	// Get subscription ID from URL
	subscriptionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid subscription ID",
		})
		return
	}

	// Get subscription
	subscription, err := repo.GetSubscriptionById(subscriptionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Subscription not found",
		})
		return
	}

	// Verify ownership
	if subscription.UserId != tenantCtx.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied",
		})
		return
	}

	// Verify tenant
	if subscription.TenantId != tenantCtx.TenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied",
		})
		return
	}

	// Check if subscription can be cancelled
	if subscription.Status != repo.SubscriptionStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Only active subscriptions can be cancelled",
			"data": gin.H{
				"current_status": subscription.Status,
			},
		})
		return
	}

	// Cancel subscription
	if err := repo.UpdateSubscriptionStatus(subscriptionID, repo.SubscriptionStatusCancelled); err != nil {
		common.SysError("Failed to cancel subscription: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to cancel subscription",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscription cancelled successfully",
		"data": gin.H{
			"id":         subscriptionID,
			"plan_code":  subscription.PlanCode,
			"expires_at": subscription.ExpiresAt,
		},
	})
}
