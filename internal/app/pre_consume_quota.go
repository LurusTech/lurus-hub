package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/logger"
	"github.com/QuantumNous/lurus-api/internal/pkg/metrics"
	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	relaycommon "github.com/QuantumNous/lurus-api/internal/adapter/provider/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/types"

	"github.com/gin-gonic/gin"
)

func ReturnPreConsumedQuota(c *gin.Context, relayInfo *relaycommon.RelayInfo) {
	if relayInfo.FinalPreConsumedQuota != 0 {
		logger.LogInfo(c, fmt.Sprintf("用户 %d 请求失败, 返还预扣费额度 %s", relayInfo.UserId, logger.FormatQuota(relayInfo.FinalPreConsumedQuota)))
		// Execute refund synchronously to prevent quota loss if process crashes
		err := PostConsumeQuota(relayInfo, -relayInfo.FinalPreConsumedQuota, 0, false)
		if err != nil {
			common.SysLog("error return pre-consumed quota: " + err.Error())
		}
	}

	// Release platform pre-auth if one was created and relay failed
	if relayInfo.PlatformPreAuthID > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := common.ReleasePreAuthGRPC(ctx, relayInfo.PlatformPreAuthID); err != nil {
			common.SysLog(fmt.Sprintf("failed to release pre-auth %d on refund, enqueuing: %s", relayInfo.PlatformPreAuthID, err.Error()))
			if enqErr := EnqueueRelease(relayInfo.IdentityAccountID, relayInfo.PlatformPreAuthID); enqErr != nil {
				common.SysError(fmt.Sprintf("CRITICAL: release AND outbox both failed for preauth %d: %s",
					relayInfo.PlatformPreAuthID, enqErr.Error()))
			}
		}
	}
}

// PreConsumeQuota checks if the user has enough quota to pre-consume.
// When BILLING_UNIFIED_ENABLED and IdentityAccountID > 0, it also calls
// platform PreAuthorize to freeze wallet balance (the authoritative check).
func PreConsumeQuota(c *gin.Context, preConsumedQuota int, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
	// Platform pre-authorization path: freeze wallet balance before relay
	if common.BillingUnifiedEnabled && relayInfo.IdentityAccountID > 0 && preConsumedQuota > 0 {
		estimatedLB := float64(preConsumedQuota) / common.QuotaPerUnit
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		preAuthStart := time.Now()
		result, err := common.PreAuthorizeGRPC(ctx, relayInfo.IdentityAccountID, estimatedLB,
			"lurus-api", "", fmt.Sprintf("relay userId=%d model=%s", relayInfo.UserId, relayInfo.OriginModelName), 300)
		metrics.BillingPreAuthDuration.Observe(time.Since(preAuthStart).Seconds())
		if err != nil {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("insufficient balance: %w", err),
				types.ErrorCodeInsufficientUserQuota, http.StatusPaymentRequired,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		relayInfo.PlatformPreAuthID = result.PreAuthID
		logger.LogInfo(c, fmt.Sprintf("platform pre-auth created: id=%d amount=%.4f LB account=%d",
			result.PreAuthID, estimatedLB, relayInfo.IdentityAccountID))
	}

	// Legacy local quota check (always runs for backward compat)
	userQuota, err := repo.GetUserQuota(relayInfo.UserId, false)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	if userQuota <= 0 {
		return types.NewErrorWithStatusCode(fmt.Errorf("用户额度不足, 剩余额度: %s", logger.FormatQuota(userQuota)), types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}
	if userQuota-preConsumedQuota < 0 {
		return types.NewErrorWithStatusCode(fmt.Errorf("预扣费额度失败, 用户剩余额度: %s, 需要预扣费额度: %s", logger.FormatQuota(userQuota), logger.FormatQuota(preConsumedQuota)), types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}

	trustQuota := common.GetTrustQuota()

	relayInfo.UserQuota = userQuota
	if userQuota > trustQuota {
		if !relayInfo.TokenUnlimited {
			tokenQuota := c.GetInt("token_quota")
			if tokenQuota > trustQuota {
				preConsumedQuota = 0
				logger.LogInfo(c, fmt.Sprintf("用户 %d 剩余额度 %s 且令牌 %d 额度 %d 充足, 信任且不需要预扣费", relayInfo.UserId, logger.FormatQuota(userQuota), relayInfo.TokenId, tokenQuota))
			}
		} else {
			preConsumedQuota = 0
			logger.LogInfo(c, fmt.Sprintf("用户 %d 额度充足且为无限额度令牌, 信任且不需要预扣费", relayInfo.UserId))
		}
	}

	if preConsumedQuota > 0 {
		err := PreConsumeTokenQuota(relayInfo, preConsumedQuota)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		err = repo.DecreaseUserQuota(relayInfo.UserId, preConsumedQuota)
		if err != nil {
			return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
		}
		logger.LogInfo(c, fmt.Sprintf("用户 %d 预扣费 %s, 预扣费后剩余额度: %s", relayInfo.UserId, logger.FormatQuota(preConsumedQuota), logger.FormatQuota(userQuota-preConsumedQuota)))
	}
	relayInfo.FinalPreConsumedQuota = preConsumedQuota
	return nil
}
