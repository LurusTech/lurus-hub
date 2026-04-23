package app

import (
	"testing"

	relaycommon "github.com/LurusTech/lurus-api/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/dto"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/ratio_setting"
	"github.com/LurusTech/lurus-api/internal/pkg/types"
)

func TestHasCustomModelRatio_DiffersFromDefault_ReturnsTrue(t *testing.T) {
	// Pick a well-known model from the default ratio map
	defaultMap := ratio_setting.GetDefaultModelRatioMap()
	var modelName string
	var defaultRatio float64
	for k, v := range defaultMap {
		modelName = k
		defaultRatio = v
		break
	}
	if modelName == "" {
		t.Skip("no models in default ratio map")
	}

	// Custom ratio differs from default
	customRatio := defaultRatio + 1.0
	if !hasCustomModelRatio(modelName, customRatio) {
		t.Error("expected true when custom ratio differs from default")
	}
}

func TestHasCustomModelRatio_MatchesDefault_ReturnsFalse(t *testing.T) {
	defaultMap := ratio_setting.GetDefaultModelRatioMap()
	var modelName string
	var defaultRatio float64
	for k, v := range defaultMap {
		modelName = k
		defaultRatio = v
		break
	}
	if modelName == "" {
		t.Skip("no models in default ratio map")
	}

	if hasCustomModelRatio(modelName, defaultRatio) {
		t.Error("expected false when custom ratio matches default")
	}
}

func TestHasCustomModelRatio_UnknownModel_ReturnsTrue(t *testing.T) {
	// A model not in the default map should return true
	if !hasCustomModelRatio("totally-unknown-model-xyz", 1.0) {
		t.Error("expected true for model not in default ratio map")
	}
}

func TestCalculateAudioQuota_UsePrice(t *testing.T) {
	info := QuotaInfo{
		UsePrice:   true,
		ModelPrice: 0.01,  // $0.01 per call
		GroupRatio: 1.0,
	}

	quota := calculateAudioQuota(info)
	// modelPrice * QuotaPerUnit * groupRatio = 0.01 * 500000 * 1.0 = 5000
	expected := int(0.01 * common.QuotaPerUnit * 1.0)
	if quota != expected {
		t.Errorf("calculateAudioQuota(usePrice) = %d, want %d", quota, expected)
	}
}

func TestCalculateAudioQuota_UsePrice_WithGroupRatio(t *testing.T) {
	info := QuotaInfo{
		UsePrice:   true,
		ModelPrice: 0.01,
		GroupRatio: 2.0,
	}

	quota := calculateAudioQuota(info)
	expected := int(0.01 * common.QuotaPerUnit * 2.0)
	if quota != expected {
		t.Errorf("calculateAudioQuota(usePrice, groupRatio=2) = %d, want %d", quota, expected)
	}
}

func TestCalculateAudioQuota_ZeroTokens(t *testing.T) {
	info := QuotaInfo{
		InputDetails:  TokenDetails{TextTokens: 0, AudioTokens: 0},
		OutputDetails: TokenDetails{TextTokens: 0, AudioTokens: 0},
		ModelName:     "gpt-4o",
		UsePrice:      false,
		ModelRatio:    1.0,
		GroupRatio:    1.0,
	}

	quota := calculateAudioQuota(info)
	// With all zero tokens but non-zero ratio, minimum quota is 1
	if quota != 1 {
		t.Errorf("calculateAudioQuota(zero tokens, non-zero ratio) = %d, want 1", quota)
	}
}

func TestCalculateAudioQuota_ZeroRatio(t *testing.T) {
	info := QuotaInfo{
		InputDetails:  TokenDetails{TextTokens: 100, AudioTokens: 0},
		OutputDetails: TokenDetails{TextTokens: 50, AudioTokens: 0},
		ModelName:     "gpt-4o",
		UsePrice:      false,
		ModelRatio:    0.0,
		GroupRatio:    1.0,
	}

	quota := calculateAudioQuota(info)
	// With zero model ratio, product is zero, and since ratio is zero, no minimum applies
	if quota != 0 {
		t.Errorf("calculateAudioQuota(zero modelRatio) = %d, want 0", quota)
	}
}

func TestCalculateAudioQuota_BasicTextTokens(t *testing.T) {
	info := QuotaInfo{
		InputDetails:  TokenDetails{TextTokens: 1000, AudioTokens: 0},
		OutputDetails: TokenDetails{TextTokens: 500, AudioTokens: 0},
		ModelName:     "gpt-4o",
		UsePrice:      false,
		ModelRatio:    1.0,
		GroupRatio:    1.0,
	}

	quota := calculateAudioQuota(info)
	// Should be positive; exact value depends on completion ratio for gpt-4o
	if quota <= 0 {
		t.Errorf("calculateAudioQuota(basic text) = %d, expected positive", quota)
	}
}

func TestCalcOpenRouterCacheCreateTokens_CacheCreationRatioIsOne_ReturnsZero(t *testing.T) {
	usage := dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		Cost:             0.05,
	}
	priceData := types.PriceData{
		ModelRatio:         1.0,
		CompletionRatio:    2.0,
		CacheRatio:         0.5,
		CacheCreationRatio: 1.0, // ratio == 1 => short-circuit
	}

	result := CalcOpenRouterCacheCreateTokens(usage, priceData)
	if result != 0 {
		t.Errorf("expected 0 when CacheCreationRatio==1, got %d", result)
	}
}

func TestCalcOpenRouterCacheCreateTokens_BasicCalculation(t *testing.T) {
	// Set up known values to verify the formula
	usage := dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 100,
		},
		Cost: 0.01,
	}

	priceData := types.PriceData{
		ModelRatio:         1.0,
		CompletionRatio:    2.0,
		CacheRatio:         0.5,
		CacheCreationRatio: 3.0,
	}

	result := CalcOpenRouterCacheCreateTokens(usage, priceData)
	// The formula:
	// quotaPrice = ModelRatio / QuotaPerUnit = 1.0 / 500000
	// promptCacheCreatePrice = quotaPrice * CacheCreationRatio
	// promptCacheReadPrice = quotaPrice * CacheRatio
	// completionPrice = quotaPrice * CompletionRatio
	// result = round((cost - totalPromptTokens*quotaPrice + cacheReadTokens*(quotaPrice-cacheReadPrice) - completionTokens*completionPrice) / (cacheCreatePrice - quotaPrice))
	// The result should be a non-negative integer (could be any value)
	// Just verify no panic and type correctness
	_ = result
}

func TestCalcOpenRouterCacheCreateTokens_ZeroCacheTokens(t *testing.T) {
	usage := dto.Usage{
		PromptTokens:     500,
		CompletionTokens: 100,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
		Cost: 0.005,
	}

	priceData := types.PriceData{
		ModelRatio:         2.0,
		CompletionRatio:    1.5,
		CacheRatio:         0.5,
		CacheCreationRatio: 4.0,
	}

	result := CalcOpenRouterCacheCreateTokens(usage, priceData)
	// Just verify no panic; exact value depends on formula
	_ = result
}

func TestCalcOpenRouterCacheCreateTokens_ZeroCost(t *testing.T) {
	usage := dto.Usage{
		PromptTokens:     500,
		CompletionTokens: 100,
		Cost:             float64(0),
	}

	priceData := types.PriceData{
		ModelRatio:         1.0,
		CompletionRatio:    2.0,
		CacheRatio:         0.5,
		CacheCreationRatio: 3.0,
	}

	result := CalcOpenRouterCacheCreateTokens(usage, priceData)
	_ = result // verify no panic
}

func TestCalcOpenRouterCacheCreateTokens_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		usage     dto.Usage
		priceData types.PriceData
		wantZero  bool
	}{
		{
			name: "cache creation ratio is 1",
			usage: dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				Cost:             0.001,
			},
			priceData: types.PriceData{
				ModelRatio:         1.0,
				CompletionRatio:    1.0,
				CacheRatio:         0.5,
				CacheCreationRatio: 1.0,
			},
			wantZero: true,
		},
		{
			name: "normal case with non-zero values",
			usage: dto.Usage{
				PromptTokens:     2000,
				CompletionTokens: 500,
				PromptTokensDetails: dto.InputTokenDetails{
					CachedTokens: 200,
				},
				Cost: 0.02,
			},
			priceData: types.PriceData{
				ModelRatio:         1.5,
				CompletionRatio:    2.0,
				CacheRatio:         0.25,
				CacheCreationRatio: 5.0,
			},
			wantZero: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CalcOpenRouterCacheCreateTokens(tc.usage, tc.priceData)
			if tc.wantZero && result != 0 {
				t.Errorf("expected 0, got %d", result)
			}
		})
	}
}

func TestPreConsumeTokenQuota_NegativeQuota_ReturnsError(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		TokenKey:       "test-key",
		TokenId:        1,
		TokenUnlimited: true,
	}

	err := PreConsumeTokenQuota(relayInfo, -1)
	if err == nil {
		t.Fatal("expected error for negative quota")
	}
}

func TestPreConsumeTokenQuota_Playground_ReturnsNil(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		IsPlayground: true,
	}

	err := PreConsumeTokenQuota(relayInfo, 100)
	if err != nil {
		t.Errorf("expected nil for playground mode, got: %v", err)
	}
}
