package helper

import (
	"testing"

	relaycommon "github.com/QuantumNous/lurus-api/internal/adapter/provider/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/QuantumNous/lurus-api/internal/pkg/dto"
	"github.com/QuantumNous/lurus-api/internal/pkg/types"
)

func TestEstimateQuotaFromUsage_NilInputs(t *testing.T) {
	if got := EstimateQuotaFromUsage(nil, &dto.Usage{}); got != 0 {
		t.Errorf("nil info: got %d, want 0", got)
	}
	if got := EstimateQuotaFromUsage(&relaycommon.RelayInfo{}, nil); got != 0 {
		t.Errorf("nil usage: got %d, want 0", got)
	}
}

func TestEstimateQuotaFromUsage_FixedPrice(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			UsePrice:   true,
			ModelPrice: 0.01,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1.5,
			},
		},
	}
	usage := &dto.Usage{PromptTokens: 100, CompletionTokens: 50}
	got := EstimateQuotaFromUsage(info, usage)
	// 0.01 * QuotaPerUnit * 1.5 = 0.01 * 500000 * 1.5 = 7500
	want := 7500
	if got != want {
		t.Errorf("fixed price: got %d, want %d", got, want)
	}
}

func TestEstimateQuotaFromUsage_TokenBased(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			ModelRatio:      2.0,
			CompletionRatio: 3.0,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1.0,
			},
		},
	}
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
	}
	got := EstimateQuotaFromUsage(info, usage)
	// (100 + 50*3) * 2.0 * 1.0 = 250 * 2 = 500
	want := 500
	if got != want {
		t.Errorf("token based: got %d, want %d", got, want)
	}
}

func TestComputeLurusExtension_BillingModes(t *testing.T) {
	tests := []struct {
		name            string
		preAuthID       int64
		identityAcctID  int64
		wantBillingMode string
	}{
		{"pre_auth", 123, 456, "pre_auth"},
		{"trust_cache", 0, 456, "trust_cache"},
		{"legacy", 0, 0, "legacy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				PlatformPreAuthID: tt.preAuthID,
				IdentityAccountID: tt.identityAcctID,
				UserQuota:         1000000,
				PriceData: types.PriceData{
					ModelRatio: 1.0,
					GroupRatioInfo: types.GroupRatioInfo{
						GroupRatio: 1.0,
					},
				},
			}
			ext := ComputeLurusExtension(info, &dto.Usage{}, 500)
			if ext.BillingMode != tt.wantBillingMode {
				t.Errorf("billing mode: got %q, want %q", ext.BillingMode, tt.wantBillingMode)
			}
		})
	}
}

func TestComputeLurusExtension_CostAndBalance(t *testing.T) {
	info := &relaycommon.RelayInfo{
		UserQuota: int(common.QuotaPerUnit * 10), // 10 LB
		PriceData: types.PriceData{
			ModelRatio: 2.0,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1.5,
			},
		},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 42},
	}
	quota := int(common.QuotaPerUnit * 2) // 2 LB cost
	ext := ComputeLurusExtension(info, usage, quota)

	if ext.CostLB != 2.0 {
		t.Errorf("cost: got %f, want 2.0", ext.CostLB)
	}
	if ext.BalanceRemaining != 8.0 {
		t.Errorf("balance: got %f, want 8.0", ext.BalanceRemaining)
	}
	if ext.CachedTokens != 42 {
		t.Errorf("cached tokens: got %d, want 42", ext.CachedTokens)
	}
	if ext.ModelRatio != 2.0 {
		t.Errorf("model ratio: got %f, want 2.0", ext.ModelRatio)
	}
	if ext.GroupRatio != 1.5 {
		t.Errorf("group ratio: got %f, want 1.5", ext.GroupRatio)
	}
}

func TestComputeLurusExtension_NegativeBalance(t *testing.T) {
	info := &relaycommon.RelayInfo{
		UserQuota: 100,
		PriceData: types.PriceData{
			ModelRatio: 1.0,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1.0,
			},
		},
	}
	ext := ComputeLurusExtension(info, &dto.Usage{}, 200)
	if ext.BalanceRemaining != 0 {
		t.Errorf("negative balance should be 0: got %f", ext.BalanceRemaining)
	}
}
