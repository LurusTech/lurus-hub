package app

import (
	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/currency"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting/operation_setting"
)

// SubscriptionQuotaInfo holds the computed display amounts for quota information.
type SubscriptionQuotaInfo struct {
	TotalAmount     float64
	UsedAmount      float64
	RemainingAmount float64
	UnlimitedQuota  bool
	ExpiredTime     int64
}

// CalculateDisplayAmount converts a raw quota value to the display amount
// based on the configured display type (USD, CNY, Tokens, or Lute).
func CalculateDisplayAmount(quota int) float64 {
	amount := float64(quota)
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		amount = amount / common.QuotaPerUnit * operation_setting.USDExchangeRate
	case operation_setting.QuotaDisplayTypeLute:
		// Lute: 1 LUT = 1 quota unit, display as LUC equivalent
		amount = currency.LutToLucDisplay(quota)
	case operation_setting.QuotaDisplayTypeTokens:
		// Keep raw token count (= LUT amount, since 1 LUT = 1 quota unit)
	default:
		// USD
		amount = amount / common.QuotaPerUnit
	}
	return amount
}

// GetSubscriptionQuotaInfo computes the SubscriptionQuotaInfo for OpenAI-compatible subscription endpoint.
func GetSubscriptionQuotaInfo(userId int, tokenId int, displayTokenStat bool) (*SubscriptionQuotaInfo, error) {
	info := &SubscriptionQuotaInfo{}

	if displayTokenStat {
		token, err := repo.GetTokenById(tokenId)
		if err != nil {
			return nil, err
		}
		info.TotalAmount = CalculateDisplayAmount(token.RemainQuota + token.UsedQuota)
		info.UsedAmount = CalculateDisplayAmount(token.UsedQuota)
		info.RemainingAmount = CalculateDisplayAmount(token.RemainQuota)
		info.UnlimitedQuota = token.UnlimitedQuota
		info.ExpiredTime = token.ExpiredTime
		if info.ExpiredTime <= 0 {
			info.ExpiredTime = 0
		}
		if token.UnlimitedQuota {
			info.TotalAmount = 100000000
		}
	} else {
		remainQuota, err := repo.GetUserQuota(userId, false)
		if err != nil {
			return nil, err
		}
		usedQuota, err := repo.GetUserUsedQuota(userId)
		if err != nil {
			return nil, err
		}
		info.TotalAmount = CalculateDisplayAmount(remainQuota + usedQuota)
		info.UsedAmount = CalculateDisplayAmount(usedQuota)
		info.RemainingAmount = CalculateDisplayAmount(remainQuota)
	}

	return info, nil
}

// GetUsageAmount computes the usage amount for OpenAI-compatible usage endpoint.
func GetUsageAmount(userId int, tokenId int, displayTokenStat bool) (float64, error) {
	var quota int
	var err error

	if displayTokenStat {
		token, err := repo.GetTokenById(tokenId)
		if err != nil {
			return 0, err
		}
		quota = token.UsedQuota
	} else {
		quota, err = repo.GetUserUsedQuota(userId)
		if err != nil {
			return 0, err
		}
	}

	return CalculateDisplayAmount(quota) * 100, nil
}
