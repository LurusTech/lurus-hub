package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/adapter/middleware"
	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// Test helpers — builds a /api/v2/client/* router with mock FlexAuth
// ============================================================================

func setupClientAPIRouter(t *testing.T) *V2TestContext {
	t.Helper()
	ctx := SetupV2TestRouter(t)

	// Add client API routes to the existing test router using a mock FlexAuth
	// that reads from test headers (same pattern as the main test setup).
	mockFlexAuth := func(c *gin.Context) {
		tenantID := c.GetHeader("X-Test-Tenant-ID")
		if tenantID == "" {
			tenantID = ctx.TenantID
		}
		userIDStr := c.GetHeader("X-Test-User-ID")
		userID := ctx.UserID
		if userIDStr != "" {
			if id, err := strconv.Atoi(userIDStr); err == nil {
				userID = id
			}
		}
		c.Set("id", userID)
		c.Set("auth_method", "token")
		// Also set tenant_context for handlers that might need it
		c.Set("tenant_context", &middleware.TenantContext{
			TenantID: tenantID,
			UserID:   userID,
		})
		c.Next()
	}

	client := ctx.Router.Group("/api/v2/client")
	client.Use(mockFlexAuth)
	{
		client.GET("/profile", ClientGetProfile)
		client.GET("/tokens", ClientGetTokens)
		client.GET("/sessions", ClientGetSessions)
		client.GET("/usage/summary", ClientGetUsageSummary)
		client.GET("/usage/models", ClientGetUsageByModel)
		client.GET("/usage/daily", ClientGetUsageDaily)
	}

	return ctx
}

func clientRequest(ctx *V2TestContext, path string) map[string]interface{} {
	headers := map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.UserID),
	}
	w := V2Request(ctx.Router, http.MethodGet, path, nil, headers)
	if w.Code != http.StatusOK {
		return nil
	}
	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

// ============================================================================
// ClientGetProfile
// ============================================================================

func TestClientGetProfile_Success(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/profile", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})

	if data["username"] != ctx.NormalUser.Username {
		t.Errorf("expected username=%s, got %v", ctx.NormalUser.Username, data["username"])
	}
	if int(data["quota"].(float64)) != ctx.NormalUser.Quota {
		t.Errorf("expected quota=%d, got %v", ctx.NormalUser.Quota, data["quota"])
	}
	if _, ok := data["remaining_quota"]; !ok {
		t.Error("expected remaining_quota field")
	}
	if _, ok := data["display_currency"]; !ok {
		t.Error("expected display_currency field")
	}
	if _, ok := data["display_amount"]; !ok {
		t.Error("expected display_amount field")
	}
}

func TestClientGetProfile_DifferentUsers(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	// Admin user
	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/profile", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.AdminUser.Id),
	})
	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	if data["username"] != ctx.AdminUser.Username {
		t.Errorf("expected admin username=%s, got %v", ctx.AdminUser.Username, data["username"])
	}
}

// ============================================================================
// ClientGetUsageSummary
// ============================================================================

func TestClientGetUsageSummary_Success(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/summary", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})

	if _, ok := data["quota"]; !ok {
		t.Error("expected quota field")
	}
	if _, ok := data["used_quota"]; !ok {
		t.Error("expected used_quota field")
	}
	if _, ok := data["rpm"]; !ok {
		t.Error("expected rpm field")
	}
	if _, ok := data["tpm"]; !ok {
		t.Error("expected tpm field")
	}
}

// ============================================================================
// ClientGetUsageByModel
// ============================================================================

func TestClientGetUsageByModel_Empty(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/models", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	models := data["models"].([]interface{})
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestClientGetUsageByModel_WithData(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	// Seed quota_data entries
	now := common.GetTimestamp()
	for _, model := range []string{"gpt-4", "gpt-4", "claude-3"} {
		qd := &repo.QuotaData{
			UserID:    ctx.NormalUser.Id,
			Username:  ctx.NormalUser.Username,
			ModelName: model,
			CreatedAt: now,
			Quota:     1000,
			TokenUsed: 500,
			Count:     1,
		}
		ctx.DB.Create(qd)
	}

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/models", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	models := data["models"].([]interface{})
	if len(models) != 2 { // gpt-4 and claude-3
		t.Errorf("expected 2 models, got %d", len(models))
	}
}

// ============================================================================
// ClientGetUsageDaily
// ============================================================================

func TestClientGetUsageDaily_DefaultDays(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/daily", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	daily := data["daily"].([]interface{})
	if len(daily) != 30 {
		t.Errorf("expected 30 days, got %d", len(daily))
	}
}

func TestClientGetUsageDaily_CustomDays(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/daily?days=7", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	daily := data["daily"].([]interface{})
	if len(daily) != 7 {
		t.Errorf("expected 7 days, got %d", len(daily))
	}
}

func TestClientGetUsageDaily_MaxDays(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/usage/daily?days=200", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	daily := data["daily"].([]interface{})
	if len(daily) != 90 { // capped at 90
		t.Errorf("expected 90 days (capped), got %d", len(daily))
	}
}

// ============================================================================
// ClientGetTokens
// ============================================================================

func TestClientGetTokens_Empty(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/tokens", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(items))
	}
}

func TestClientGetTokens_WithData(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	SeedV2Token(t, ctx, ctx.NormalUser.Id, "test-token-1")
	SeedV2Token(t, ctx, ctx.NormalUser.Id, "test-token-2")

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/tokens", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(items))
	}
	total := int(data["total"].(float64))
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestClientGetTokens_Pagination(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	for i := 0; i < 25; i++ {
		SeedV2Token(t, ctx, ctx.NormalUser.Id, fmt.Sprintf("token-%d", i))
	}

	// Page 2 with size 10
	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/tokens?p=2&size=10", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 10 {
		t.Errorf("expected 10 tokens on page 2, got %d", len(items))
	}
}

func TestClientGetTokens_UserIsolation(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	SeedV2Token(t, ctx, ctx.NormalUser.Id, "normal-token")
	SeedV2Token(t, ctx, ctx.AdminUser.Id, "admin-token-1")
	SeedV2Token(t, ctx, ctx.AdminUser.Id, "admin-token-2")

	// Normal user sees only their token
	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/tokens", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 1 {
		t.Errorf("normal user: expected 1 token, got %v", data["total"])
	}

	// Admin user sees only their tokens
	w = V2Request(ctx.Router, http.MethodGet, "/api/v2/client/tokens", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.AdminUser.Id),
	})
	resp = AssertV2Success(t, w)
	data = resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 2 {
		t.Errorf("admin user: expected 2 tokens, got %v", data["total"])
	}
}

// ============================================================================
// ClientGetSessions
// ============================================================================

func TestClientGetSessions_Success(t *testing.T) {
	ctx := setupClientAPIRouter(t)
	defer ctx.Cleanup()

	w := V2Request(ctx.Router, http.MethodGet, "/api/v2/client/sessions", nil, map[string]string{
		"X-Test-User-ID": fmt.Sprintf("%d", ctx.NormalUser.Id),
	})

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)
	data := resp["data"].(map[string]interface{})

	if data["auth_method"] != "token" {
		t.Errorf("expected auth_method=token, got %v", data["auth_method"])
	}
	if data["username"] != ctx.NormalUser.Username {
		t.Errorf("expected username=%s, got %v", ctx.NormalUser.Username, data["username"])
	}
}
