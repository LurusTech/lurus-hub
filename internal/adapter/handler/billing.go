package handler

import (
	"net/http"

	"github.com/QuantumNous/lurus-api/internal/adapter/middleware"
	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/setting/operation_setting"
	"github.com/QuantumNous/lurus-api/internal/pkg/types"
	"github.com/gin-gonic/gin"
)

// GetIdentityOverview returns the authenticated user's aggregated identity overview
// from lurus-identity (VIP level, Lubell balance, subscription status).
// Degrades gracefully when lurus-identity is unavailable.
// GET /api/v2/user/identity-overview?product_id=<pid>
func GetIdentityOverview(c *gin.Context) {
	tenantCtx, err := middleware.GetTenantContext(c)
	if err != nil || tenantCtx.ZitadelUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	im, err := common.GetAccountByZitadelSub(c.Request.Context(), tenantCtx.ZitadelUserID)
	if err != nil || im == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity account not found"})
		return
	}

	productID := c.DefaultQuery("product_id", "lurus-api")
	ov, _ := common.GetAccountOverview(c.Request.Context(), im.ID, productID)
	if ov == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "identity service unavailable"})
		return
	}

	c.JSON(http.StatusOK, ov)
}

// calculateDisplayAmount converts a raw quota value to the display amount
// based on the configured display type (USD, CNY, or Tokens).
func calculateDisplayAmount(quota int) float64 {
	amount := float64(quota)
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		amount = amount / common.QuotaPerUnit * operation_setting.USDExchangeRate
	case operation_setting.QuotaDisplayTypeTokens:
		// Keep raw token count
	default:
		// USD
		amount = amount / common.QuotaPerUnit
	}
	return amount
}

func GetSubscription(c *gin.Context) {
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")

	var totalAmount, expiredTime float64
	var expired int64

	if common.DisplayTokenStatEnabled {
		token, err := repo.GetTokenById(tokenId)
		if err != nil {
			c.JSON(200, gin.H{"error": types.OpenAIError{Message: err.Error(), Type: "upstream_error"}})
			return
		}
		totalAmount = calculateDisplayAmount(token.RemainQuota + token.UsedQuota)
		expired = token.ExpiredTime
		if expired <= 0 {
			expired = 0
		}
		if token.UnlimitedQuota {
			totalAmount = 100000000
		}
		_ = expiredTime
	} else {
		remainQuota, err := repo.GetUserQuota(userId, false)
		if err != nil {
			c.JSON(200, gin.H{"error": types.OpenAIError{Message: err.Error(), Type: "upstream_error"}})
			return
		}
		usedQuota, err := repo.GetUserUsedQuota(userId)
		if err != nil {
			c.JSON(200, gin.H{"error": types.OpenAIError{Message: err.Error(), Type: "upstream_error"}})
			return
		}
		totalAmount = calculateDisplayAmount(remainQuota + usedQuota)
	}

	subscription := OpenAISubscriptionResponse{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       totalAmount,
		HardLimitUSD:       totalAmount,
		SystemHardLimitUSD: totalAmount,
		AccessUntil:        expired,
	}
	c.JSON(200, subscription)
}

func GetUsage(c *gin.Context) {
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")

	var quota int
	if common.DisplayTokenStatEnabled {
		token, err := repo.GetTokenById(tokenId)
		if err != nil {
			c.JSON(200, gin.H{"error": types.OpenAIError{Message: err.Error(), Type: "new_api_error"}})
			return
		}
		quota = token.UsedQuota
	} else {
		var err error
		quota, err = repo.GetUserUsedQuota(userId)
		if err != nil {
			c.JSON(200, gin.H{"error": types.OpenAIError{Message: err.Error(), Type: "new_api_error"}})
			return
		}
	}

	usage := OpenAIUsageResponse{
		Object:     "list",
		TotalUsage: calculateDisplayAmount(quota) * 100,
	}
	c.JSON(200, usage)
}
