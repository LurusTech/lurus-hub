package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/domain/entity"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/currency"
	"github.com/LurusTech/lurus-api/internal/pkg/logger"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// ===== Currency Info =====

// InternalGetCurrencyInfo returns the current currency system configuration.
// GET /internal/currency/info
func InternalGetCurrencyInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"currencies": []gin.H{
				{
					"code":        currency.CodeLuGold,
					"name":        "LuGold",
					"name_cn":     "陆金",
					"tier":        1,
					"description": "Subscription/package unit",
				},
				{
					"code":        currency.CodeLuCoin,
					"name":        "LuCoin",
					"name_cn":     "陆币",
					"tier":        2,
					"description": "Platform-wide credit (1 LUC ~ CNY 1)",
				},
				{
					"code":        currency.CodeLute,
					"name":        "Lute",
					"name_cn":     "路特",
					"tier":        3,
					"description": "API usage credit (1 LUT = 1 internal quota unit)",
				},
			},
			"exchange_rates": gin.H{
				"lug_to_luc": currency.LugToLuc,
				"luc_to_lut": currency.LucToLut(),
				"lug_to_lut": currency.LugToLut(),
			},
			"vip_bonuses": []gin.H{
				{"level": 0, "name": "Standard", "name_cn": "标准", "multiplier": currency.VIPBonusRate(0)},
				{"level": 1, "name": "Silver", "name_cn": "白银", "multiplier": currency.VIPBonusRate(1)},
				{"level": 2, "name": "Gold", "name_cn": "黄金", "multiplier": currency.VIPBonusRate(2)},
				{"level": 3, "name": "Platinum", "name_cn": "铂金", "multiplier": currency.VIPBonusRate(3)},
				{"level": 4, "name": "Diamond", "name_cn": "钻石", "multiplier": currency.VIPBonusRate(4)},
			},
			"conversion_direction": "LUG -> LUC -> LUT (one-way, irreversible)",
		},
	})
}

// ===== Currency Exchange =====

// InternalExchangeLucToLut converts LuCoin (platform credit) to Lute (API credit).
// This is called by lurus-platform after debiting the user's wallet.
// Idempotent via reference_id.
//
// POST /internal/currency/exchange
func InternalExchangeLucToLut(c *gin.Context) {
	var req struct {
		UserId          int     `json:"user_id" binding:"required"`
		LucAmount       float64 `json:"luc_amount" binding:"required"`
		VIPLevel        int     `json:"vip_level"`
		ReferenceId     string  `json:"reference_id" binding:"required"`
		PlatformOrderNo string  `json:"platform_order_no"`
		Note            string  `json:"note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.UserId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	if req.LucAmount <= 0 || req.LucAmount > 100000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "LUC amount must be between 0 and 100,000",
			"error_code": "VALIDATION_FAILED",
		})
		return
	}

	// Idempotency check
	var existingCount int64
	repo.DB.Model(&entity.CurrencyExchange{}).Where("reference_id = ?", req.ReferenceId).Count(&existingCount)
	if existingCount > 0 {
		var existing entity.CurrencyExchange
		repo.DB.Where("reference_id = ?", req.ReferenceId).First(&existing)
		currentQuota, _ := repo.GetUserQuota(req.UserId, true)
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"idempotent": true,
			"data": gin.H{
				"exchange_id":    existing.Id,
				"luc_amount":     existing.SourceAmount,
				"lut_amount":     existing.TargetAmount,
				"exchange_rate":  existing.ExchangeRate,
				"user_balance":   currentQuota,
				"balance_luc":    currency.LutToLucDisplay(currentQuota),
				"balance_cn":     currency.FormatLutCN(currentQuota),
			},
		})
		return
	}

	// Verify user exists
	user, err := repo.GetUserById(req.UserId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found",
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	if user.Status == common.UserStatusDisabled {
		c.JSON(http.StatusForbidden, gin.H{
			"success":    false,
			"message":    "User is disabled",
			"error_code": "USER_DISABLED",
		})
		return
	}

	// Calculate exchange
	info := currency.CalculateExchange(req.LucAmount, req.VIPLevel)

	if info.TargetAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "Exchange amount too small, resulting in 0 LUT",
			"error_code": "AMOUNT_TOO_SMALL",
		})
		return
	}

	// Begin transaction: record exchange + credit quota atomically
	tx := repo.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to begin transaction: " + tx.Error.Error(),
		})
		return
	}

	// Record exchange transaction
	exchange := &entity.CurrencyExchange{
		UserId:          req.UserId,
		SourceCurrency:  currency.CodeLuCoin,
		SourceAmount:    req.LucAmount,
		TargetCurrency:  currency.CodeLute,
		TargetAmount:    info.TargetAmount,
		ExchangeRate:    info.ExchangeRate,
		VIPLevel:        req.VIPLevel,
		VIPBonus:        info.VIPBonus,
		ReferenceId:     req.ReferenceId,
		PlatformOrderNo: req.PlatformOrderNo,
		SourceService:   "lurus-platform",
		Note:            req.Note,
	}

	if err := tx.Create(exchange).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to record exchange: " + err.Error(),
		})
		return
	}

	// Credit LUT (quota) to user
	if err := tx.Model(&repo.User{}).Where("id = ?", req.UserId).
		Update("quota", repo.DB.Raw("quota + ?", info.TargetAmount)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to credit LUT to user: " + err.Error(),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to commit exchange: " + err.Error(),
		})
		return
	}

	// Log the operation
	keyName := c.GetString("internal_api_key_name")
	repo.RecordLog(req.UserId, repo.LogTypeTopup,
		fmt.Sprintf("Currency exchange: %.4f LUC -> %s (rate=%.0f, VIP=%d). Ref: %s. Key: %s",
			req.LucAmount, logger.LogQuota(info.TargetAmount), info.ExchangeRate, req.VIPLevel, req.ReferenceId, keyName))

	// Fetch updated quota
	newQuota, _ := repo.GetUserQuota(req.UserId, true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"exchange_id":    exchange.Id,
			"luc_amount":     req.LucAmount,
			"lut_amount":     info.TargetAmount,
			"exchange_rate":  info.ExchangeRate,
			"vip_bonus":      info.VIPBonus,
			"user_balance":   newQuota,
			"balance_luc":    currency.LutToLucDisplay(newQuota),
			"balance_cn":     currency.FormatLutCN(newQuota),
		},
	})
}

// ===== Model Pricing =====

// InternalGetModelPricing returns model pricing in Lute (LUT) per 1K tokens.
// GET /internal/currency/models/pricing
func InternalGetModelPricing(c *gin.Context) {
	groupName := c.DefaultQuery("group", "default")
	modelFilter := c.Query("model")

	groupRatio := ratio_setting.GetGroupRatio(groupName)

	type modelPrice struct {
		Model          string  `json:"model"`
		InputPer1K     float64 `json:"input_lut_per_1k"`
		OutputPer1K    float64 `json:"output_lut_per_1k"`
		ModelRatio     float64 `json:"model_ratio"`
		CompletionRate float64 `json:"completion_ratio"`
		GroupRatio     float64 `json:"group_ratio"`
		InputLucPer1K  float64 `json:"input_luc_per_1k"`
		OutputLucPer1K float64 `json:"output_luc_per_1k"`
	}

	allRatios := ratio_setting.GetModelRatioCopy()
	result := make([]modelPrice, 0, len(allRatios))

	for model, ratio := range allRatios {
		if modelFilter != "" && model != modelFilter {
			continue
		}

		completionRatio := ratio_setting.GetCompletionRatio(model)
		inputLut, outputLut := currency.ModelPriceLut(ratio, completionRatio, groupRatio)

		result = append(result, modelPrice{
			Model:          model,
			InputPer1K:     inputLut,
			OutputPer1K:    outputLut,
			ModelRatio:     ratio,
			CompletionRate: completionRatio,
			GroupRatio:     groupRatio,
			InputLucPer1K:  currency.LutToLucDisplay(int(inputLut)),
			OutputLucPer1K: currency.LutToLucDisplay(int(outputLut)),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"currency":       currency.CodeLute,
			"currency_cn":    "路特",
			"group":          groupName,
			"group_ratio":    groupRatio,
			"exchange_rate":  currency.LucToLut(),
			"models":         result,
			"total":          len(result),
		},
	})
}

// ===== User Balance (enriched with Lute) =====

// InternalGetUserBalanceLute returns user balance with full currency breakdown.
// GET /internal/currency/balance/:id
func InternalGetUserBalanceLute(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	user, err := repo.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found",
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	// Count lifetime exchanges for this user
	var totalExchanged int64
	repo.DB.Model(&entity.CurrencyExchange{}).
		Where("user_id = ?", userId).
		Select("COALESCE(SUM(target_amount), 0)").
		Scan(&totalExchanged)

	var totalLucSpent float64
	repo.DB.Model(&entity.CurrencyExchange{}).
		Where("user_id = ?", userId).
		Select("COALESCE(SUM(source_amount), 0)").
		Scan(&totalLucSpent)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id": user.Id,
			"balance": gin.H{
				"lut":         user.Quota,
				"lut_display": currency.FormatLutCN(user.Quota),
				"luc_equiv":   currency.LutToLucDisplay(user.Quota),
			},
			"used": gin.H{
				"lut":         user.UsedQuota,
				"lut_display": currency.FormatLutCN(user.UsedQuota),
				"luc_equiv":   currency.LutToLucDisplay(user.UsedQuota),
			},
			"daily": gin.H{
				"quota":          user.DailyQuota,
				"used":           user.DailyUsed,
				"last_reset":     user.LastDailyReset,
			},
			"lifetime_exchanges": gin.H{
				"total_lut_received": totalExchanged,
				"total_luc_spent":    totalLucSpent,
			},
			"exchange_rate": gin.H{
				"luc_to_lut": currency.LucToLut(),
				"currency":   currency.CodeLute,
			},
		},
	})
}

// InternalGetExchangeHistory returns paginated exchange history for a user.
// GET /internal/currency/exchanges/:id
func InternalGetExchangeHistory(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var exchanges []entity.CurrencyExchange
	var total int64

	repo.DB.Model(&entity.CurrencyExchange{}).Where("user_id = ?", userId).Count(&total)
	repo.DB.Where("user_id = ?", userId).
		Order("id DESC").
		Offset(offset).Limit(pageSize).
		Find(&exchanges)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"exchanges": exchanges,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
