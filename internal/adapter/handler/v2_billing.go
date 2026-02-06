package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"
	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/app"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/setting"
	"github.com/QuantumNous/lurus-api/internal/pkg/setting/operation_setting"
	"github.com/QuantumNous/lurus-api/internal/pkg/setting/system_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
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

// GetSubscriptionPlansV2 returns available subscription plans
// Route: GET /api/v2/:tenant_slug/billing/plans
func GetSubscriptionPlansV2(c *gin.Context) {
	plans := repo.GetSubscriptionPlans()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    plans,
	})
}

// GetTopUpInfoV2 returns available payment methods and configuration
// Route: GET /api/v2/:tenant_slug/billing/topup-info
func GetTopUpInfoV2(c *gin.Context) {
	payMethods := operation_setting.PayMethods

	// Add Stripe to methods if configured
	if setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "" {
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}
		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "rgba(var(--semi-purple-5), 1)",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enable_online_topup": operation_setting.PayAddress != "" && operation_setting.EpayId != "" && operation_setting.EpayKey != "",
			"enable_stripe_topup": setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "",
			"enable_creem_topup":  setting.CreemApiKey != "" && setting.CreemProducts != "[]",
			"creem_products":      setting.CreemProducts,
			"pay_methods":         payMethods,
			"min_topup":           operation_setting.MinTopUp,
			"stripe_min_topup":    setting.StripeMinTopUp,
			"amount_options":      operation_setting.GetPaymentSetting().AmountOptions,
			"discount":            operation_setting.GetPaymentSetting().AmountDiscount,
		},
	})
}

// InitiatePaymentV2 initiates payment for a pending topup order
// Route: POST /api/v2/:tenant_slug/billing/pay
func InitiatePaymentV2(c *gin.Context) {
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Tenant context not found",
		})
		return
	}

	var req struct {
		TradeNo       string `json:"trade_no" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// Look up the pending topup order
	topup := repo.GetTopUpByTradeNo(req.TradeNo)
	if topup == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Order not found",
		})
		return
	}

	// Verify ownership
	if topup.UserId != tenantCtx.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied",
		})
		return
	}

	// Verify order status
	if topup.Status != common.TopUpStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Order is not in pending status",
		})
		return
	}

	user, err := repo.GetUserById(tenantCtx.UserID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to load user data",
		})
		return
	}

	tenantSlug := c.Param("tenant_slug")

	switch req.PaymentMethod {
	case "stripe":
		if setting.StripeApiSecret == "" || setting.StripeWebhookSecret == "" || setting.StripePriceId == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Stripe payment is not configured",
			})
			return
		}
		reference := fmt.Sprintf("v2-stripe-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
		referenceId := "ref_" + common.Sha1([]byte(reference))

		payLink, err := genStripeLink(referenceId, user.StripeCustomer, user.Email, topup.Amount)
		if err != nil {
			log.Printf("Failed to generate Stripe checkout link: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to initiate payment",
			})
			return
		}

		// Update topup with Stripe reference
		topup.TradeNo = referenceId
		topup.PaymentMethod = "stripe"
		repo.TopUpUpdate(topup)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"payment_url": payLink,
				"trade_no":    referenceId,
			},
		})

	case "creem":
		if setting.CreemApiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Creem payment is not configured",
			})
			return
		}

		var products []CreemProduct
		if err := json.Unmarshal([]byte(setting.CreemProducts), &products); err != nil || len(products) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "No Creem products configured",
			})
			return
		}

		// Find a product matching the topup amount
		var selectedProduct *CreemProduct
		for _, p := range products {
			if p.Quota == topup.Amount {
				selectedProduct = &p
				break
			}
		}
		if selectedProduct == nil {
			// Use first product as fallback
			selectedProduct = &products[0]
		}

		reference := fmt.Sprintf("v2-creem-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
		referenceId := "ref_" + common.Sha1([]byte(reference))

		checkoutUrl, err := genCreemLink(referenceId, selectedProduct, user.Email, user.Username)
		if err != nil {
			log.Printf("Failed to generate Creem checkout link: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to initiate payment",
			})
			return
		}

		// Update topup with Creem reference
		topup.TradeNo = referenceId
		topup.PaymentMethod = "creem"
		repo.TopUpUpdate(topup)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"payment_url": checkoutUrl,
				"trade_no":    referenceId,
			},
		})

	default:
		// Epay payment
		if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Online payment is not configured",
			})
			return
		}

		if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Unsupported payment method",
			})
			return
		}

		callBackAddress := app.GetCallbackAddress()
		returnUrlStr := fmt.Sprintf("%s/console/topup?payment=success&tenant=%s", system_setting.ServerAddress, tenantSlug)
		notifyUrlStr := callBackAddress + "/api/user/epay/notify"

		client := GetEpayClient()
		if client == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Payment service unavailable",
			})
			return
		}

		group, _ := repo.GetUserGroup(tenantCtx.UserID, true)
		payMoney := getPayMoney(topup.Amount, group)

		returnUrlParsed, _ := url.Parse(returnUrlStr)
		notifyUrlParsed, _ := url.Parse(notifyUrlStr)

		uri, params, err := client.Purchase(&epay.PurchaseArgs{
			Type:           req.PaymentMethod,
			ServiceTradeNo: topup.TradeNo,
			Name:           fmt.Sprintf("TUC%d", topup.Amount),
			Money:          strconv.FormatFloat(payMoney, 'f', 2, 64),
			Device:         epay.PC,
			NotifyUrl:      notifyUrlParsed,
			ReturnUrl:      returnUrlParsed,
		})
		if err != nil {
			log.Printf("Failed to create Epay order: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to initiate payment",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"payment_url": uri,
				"params":      params,
				"trade_no":    topup.TradeNo,
			},
		})
	}
}
