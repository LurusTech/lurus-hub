package controller

import (
	"github.com/QuantumNous/lurus-api/internal/biz/service"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/types"
	"github.com/gin-gonic/gin"
)

func GetSubscription(c *gin.Context) {
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	info, err := service.GetSubscriptionQuotaInfo(userId, tokenId, common.DisplayTokenStatEnabled)
	if err != nil {
		openAIError := types.OpenAIError{
			Message: err.Error(),
			Type:    "upstream_error",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
		return
	}
	subscription := OpenAISubscriptionResponse{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       info.TotalAmount,
		HardLimitUSD:       info.TotalAmount,
		SystemHardLimitUSD: info.TotalAmount,
		AccessUntil:        info.ExpiredTime,
	}
	c.JSON(200, subscription)
}

func GetUsage(c *gin.Context) {
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	amount, err := service.GetUsageAmount(userId, tokenId, common.DisplayTokenStatEnabled)
	if err != nil {
		openAIError := types.OpenAIError{
			Message: err.Error(),
			Type:    "new_api_error",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
		return
	}
	usage := OpenAIUsageResponse{
		Object:     "list",
		TotalUsage: amount,
	}
	c.JSON(200, usage)
}
