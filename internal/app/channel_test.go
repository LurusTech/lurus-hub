package app

import (
	"errors"
	"net/http"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting/operation_setting"
	"github.com/LurusTech/lurus-hub/internal/pkg/types"
)

// saveAndRestoreAutoBanFlags saves and restores the automatic channel disable/enable flags.
func saveAndRestoreAutoBanFlags(t *testing.T) {
	t.Helper()
	origDisable := common.AutomaticDisableChannelEnabled
	origEnable := common.AutomaticEnableChannelEnabled
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = origDisable
		common.AutomaticEnableChannelEnabled = origEnable
	})
}

// saveAndRestoreDisableKeywords saves and restores the automatic disable keywords.
func saveAndRestoreDisableKeywords(t *testing.T) {
	t.Helper()
	orig := make([]string, len(operation_setting.AutomaticDisableKeywords))
	copy(orig, operation_setting.AutomaticDisableKeywords)
	t.Cleanup(func() {
		operation_setting.AutomaticDisableKeywords = orig
	})
}

func TestShouldDisableChannel_AutoBanDisabled_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = false

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "test error",
		Type:    "error",
		Code:    "test",
	}, http.StatusInternalServerError)

	if ShouldDisableChannel(1, apiErr) {
		t.Error("expected false when auto ban disabled")
	}
}

func TestShouldDisableChannel_NilError_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	if ShouldDisableChannel(1, nil) {
		t.Error("expected false for nil error")
	}
}

func TestShouldDisableChannel_ChannelError_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	// A channel-prefixed error code triggers IsChannelError
	apiErr := types.NewError(
		errors.New("no available key"),
		types.ErrorCodeChannelNoAvailableKey,
	)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for channel error")
	}
}

func TestShouldDisableChannel_SkipRetryError_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.NewError(
		errors.New("insufficient user quota"),
		types.ErrorCodeInsufficientUserQuota,
		types.ErrOptionWithSkipRetry(),
	)

	if ShouldDisableChannel(1, apiErr) {
		t.Error("expected false for skip-retry error (insufficient quota)")
	}
}

func TestShouldDisableChannel_Unauthorized_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "invalid credentials",
		Type:    "error",
		Code:    "unauthorized",
	}, http.StatusUnauthorized)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for 401 Unauthorized")
	}
}

func TestShouldDisableChannel_ForbiddenGemini_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "forbidden",
		Type:    "error",
		Code:    "forbidden",
	}, http.StatusForbidden)

	if !ShouldDisableChannel(constant.ChannelTypeGemini, apiErr) {
		t.Error("expected true for 403 on Gemini channel")
	}
}

func TestShouldDisableChannel_ForbiddenNonGemini_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	saveAndRestoreDisableKeywords(t)
	common.AutomaticDisableChannelEnabled = true

	// Clear keywords so the keyword match path doesn't fire
	operation_setting.AutomaticDisableKeywords = []string{}

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "forbidden",
		Type:    "error",
		Code:    "forbidden_but_not_gemini",
	}, http.StatusForbidden)

	if ShouldDisableChannel(1, apiErr) {
		t.Error("expected false for 403 on non-Gemini channel without matching keywords")
	}
}

func TestShouldDisableChannel_InvalidApiKey_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "invalid api key provided",
		Type:    "error",
		Code:    "invalid_api_key",
	}, http.StatusUnauthorized)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for invalid_api_key code")
	}
}

func TestShouldDisableChannel_AccountDeactivated_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "account deactivated",
		Type:    "error",
		Code:    "account_deactivated",
	}, http.StatusForbidden)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for account_deactivated code")
	}
}

func TestShouldDisableChannel_BillingNotActive_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "billing not active",
		Type:    "error",
		Code:    "billing_not_active",
	}, http.StatusForbidden)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for billing_not_active code")
	}
}

func TestShouldDisableChannel_InsufficientQuotaType_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "you have exceeded your quota",
		Type:    "insufficient_quota",
		Code:    "some_code",
	}, http.StatusPaymentRequired)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for insufficient_quota type")
	}
}

func TestShouldDisableChannel_AuthenticationErrorType_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithClaudeError(types.ClaudeError{
		Message: "authentication failed",
		Type:    "authentication_error",
	}, http.StatusUnauthorized)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for authentication_error type")
	}
}

func TestShouldDisableChannel_PermissionErrorType_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticDisableChannelEnabled = true

	apiErr := types.WithClaudeError(types.ClaudeError{
		Message: "permission denied",
		Type:    "permission_error",
	}, http.StatusForbidden)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for permission_error type")
	}
}

func TestShouldDisableChannel_KeywordMatch_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	saveAndRestoreDisableKeywords(t)
	common.AutomaticDisableChannelEnabled = true

	operation_setting.AutomaticDisableKeywords = []string{
		"your credit balance is too low",
	}

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "Your credit balance is too low to continue",
		Type:    "error",
		Code:    "some_unknown_code",
	}, http.StatusPaymentRequired)

	if !ShouldDisableChannel(1, apiErr) {
		t.Error("expected true for keyword match in error message")
	}
}

func TestShouldDisableChannel_NoMatch_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	saveAndRestoreDisableKeywords(t)
	common.AutomaticDisableChannelEnabled = true
	operation_setting.AutomaticDisableKeywords = []string{}

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "some random error that doesn't match anything",
		Type:    "error",
		Code:    "some_random_code",
	}, http.StatusBadGateway)

	if ShouldDisableChannel(1, apiErr) {
		t.Error("expected false when no disable conditions match")
	}
}

func TestShouldEnableChannel_AutoEnableDisabled_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticEnableChannelEnabled = false

	if ShouldEnableChannel(nil, common.ChannelStatusAutoDisabled) {
		t.Error("expected false when auto enable disabled")
	}
}

func TestShouldEnableChannel_NilError_AutoDisabled_ReturnsTrue(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticEnableChannelEnabled = true

	if !ShouldEnableChannel(nil, common.ChannelStatusAutoDisabled) {
		t.Error("expected true when error is nil and channel is auto-disabled")
	}
}

func TestShouldEnableChannel_NonNilError_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticEnableChannelEnabled = true

	apiErr := types.NewError(errors.New("some error"), types.ErrorCodeBadResponse)
	if ShouldEnableChannel(apiErr, common.ChannelStatusAutoDisabled) {
		t.Error("expected false when error is non-nil")
	}
}

func TestShouldEnableChannel_NotAutoDisabledStatus_ReturnsFalse(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	common.AutomaticEnableChannelEnabled = true

	if ShouldEnableChannel(nil, common.ChannelStatusEnabled) {
		t.Error("expected false when channel status is not auto-disabled")
	}
}

func TestShouldDisableChannel_TableDriven(t *testing.T) {
	saveAndRestoreAutoBanFlags(t)
	saveAndRestoreDisableKeywords(t)
	common.AutomaticDisableChannelEnabled = true
	operation_setting.AutomaticDisableKeywords = []string{}

	tests := []struct {
		name        string
		channelType int
		err         *types.NewAPIError
		want        bool
	}{
		{
			name: "Arrearage code",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "arrearage", Type: "error", Code: "Arrearage",
			}, http.StatusPaymentRequired),
			want: true,
		},
		{
			name: "pre_consume_token_quota_failed code",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "token quota failed", Type: "error", Code: "pre_consume_token_quota_failed",
			}, http.StatusForbidden),
			want: true,
		},
		{
			name: "insufficient_user_quota type",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "insufficient", Type: "insufficient_user_quota", Code: "x",
			}, http.StatusPaymentRequired),
			want: true,
		},
		{
			name: "forbidden type",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "forbidden", Type: "forbidden", Code: "x",
			}, http.StatusForbidden),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldDisableChannel(tc.channelType, tc.err)
			if got != tc.want {
				t.Errorf("ShouldDisableChannel() = %v, want %v", got, tc.want)
			}
		})
	}
}
