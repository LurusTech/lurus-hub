package app

import (
	"errors"
	"fmt"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
)

const (
	TokenNameMaxLength = 50
	MaxQuotaMultiplier = 1000000000
)

// ValidateTokenName checks that the token name does not exceed the maximum length.
func ValidateTokenName(name string) error {
	if len(name) > TokenNameMaxLength {
		return errors.New("令牌名称过长")
	}
	return nil
}

// ValidateTokenQuota validates the token quota values.
// If unlimitedQuota is true, no quota validation is performed.
func ValidateTokenQuota(remainQuota int, unlimitedQuota bool) error {
	if unlimitedQuota {
		return nil
	}
	if remainQuota < 0 {
		return errors.New("额度值不能为负数")
	}
	maxQuotaValue := int(MaxQuotaMultiplier * common.QuotaPerUnit)
	if remainQuota > maxQuotaValue {
		return fmt.Errorf("额度值超出有效范围，最大值为 %d", maxQuotaValue)
	}
	return nil
}

// CanEnableToken checks whether a token can be enabled based on its current state.
// Returns nil if the token can be enabled, or an error describing why it cannot.
func CanEnableToken(token *repo.Token) error {
	if token.Status == common.TokenStatusExpired &&
		token.ExpiredTime <= common.GetTimestamp() &&
		token.ExpiredTime != -1 {
		return errors.New("令牌已过期，无法启用，请先修改令牌过期时间，或者设置为永不过期")
	}
	if token.Status == common.TokenStatusExhausted &&
		token.RemainQuota <= 0 &&
		!token.UnlimitedQuota {
		return errors.New("令牌可用额度已用尽，无法启用，请先修改令牌剩余额度，或者设置为无限额度")
	}
	return nil
}

// GenerateTokenKey generates a new unique token key.
func GenerateTokenKey() (string, error) {
	key, err := common.GenerateKey()
	if err != nil {
		common.SysLog("failed to generate token key: " + err.Error())
		return "", errors.New("生成令牌失败")
	}
	return key, nil
}

// BuildCleanToken creates a sanitized Token struct for insertion.
func BuildCleanToken(userId int, tenantId string, token *repo.Token, key string) repo.Token {
	return repo.Token{
		UserId:             userId,
		TenantId:           tenantId,
		Name:               token.Name,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		CrossGroupRetry:    token.CrossGroupRetry,
	}
}

// ApplyTokenUpdate copies update fields from the source token to the target.
// Returns the updated target token.
func ApplyTokenUpdate(target *repo.Token, source *repo.Token) {
	target.Name = source.Name
	target.ExpiredTime = source.ExpiredTime
	target.RemainQuota = source.RemainQuota
	target.UnlimitedQuota = source.UnlimitedQuota
	target.ModelLimitsEnabled = source.ModelLimitsEnabled
	target.ModelLimits = source.ModelLimits
	target.AllowIps = source.AllowIps
	target.Group = source.Group
	target.CrossGroupRetry = source.CrossGroupRetry
}
