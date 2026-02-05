package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
)

// ============================================================================
// V2 Billing Controller Tests
// ============================================================================

func TestTopUpV2_InvalidPaymentMethod(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	body := map[string]interface{}{
		"amount":         1000,
		"money":          10.0,
		"payment_method": "bitcoin", // Invalid payment method
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/topup", body, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Invalid payment method. Supported: stripe, epay, creem" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestTopUpV2_AmountLimit(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Try to topup more than 10,000,000 cents
	body := map[string]interface{}{
		"amount":         10000001, // Exceeds 10M limit
		"money":          100000.01,
		"payment_method": "stripe",
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/topup", body, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Amount exceeds maximum limit" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestTopUpV2_Success(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	body := map[string]interface{}{
		"amount":         1000,
		"money":          10.0,
		"payment_method": "stripe",
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/topup", body, nil)

	AssertV2Status(t, w, http.StatusCreated)
	resp := AssertV2Success(t, w)

	data := resp["data"].(map[string]interface{})
	if data["trade_no"] == nil || data["trade_no"] == "" {
		t.Error("expected trade_no to be returned")
	}
	if data["amount"].(float64) != 1000 {
		t.Errorf("expected amount=1000, got %v", data["amount"])
	}
	if data["status"] != common.TopUpStatusPending {
		t.Errorf("expected status='pending', got %v", data["status"])
	}
}

func TestTopUpV2_RequiredFields(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing amount",
			body: map[string]interface{}{"money": 10.0, "payment_method": "stripe"},
		},
		{
			name: "missing money",
			body: map[string]interface{}{"amount": 1000, "payment_method": "stripe"},
		},
		{
			name: "missing payment_method",
			body: map[string]interface{}{"amount": 1000, "money": 10.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/topup", tt.body, nil)
			AssertV2Status(t, w, http.StatusBadRequest)
		})
	}
}

func TestGetTopUpsV2_Success(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create some topups
	SeedV2TopUp(t, ctx, ctx.NormalUser.Id, common.TopUpStatusPending)
	SeedV2TopUp(t, ctx, ctx.NormalUser.Id, common.TopUpStatusSuccess)

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodGet, "/api/v2/test-tenant/billing/topups", nil, nil)

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)

	data := resp["data"].(map[string]interface{})
	topups := data["topups"].([]interface{})
	if len(topups) != 2 {
		t.Errorf("expected 2 topups, got %d", len(topups))
	}
}

func TestSubscribeV2_AlreadySubscribed(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create an active subscription
	SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusActive)

	// Try to subscribe again
	body := map[string]interface{}{
		"plan_code":      "monthly",
		"payment_method": "stripe",
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscribe", body, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "You already have an active subscription" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestSubscribeV2_InvalidPlanCode(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	body := map[string]interface{}{
		"plan_code":      "invalid_plan",
		"payment_method": "stripe",
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscribe", body, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Invalid plan code. Supported: weekly, monthly, quarterly, yearly" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestSubscribeV2_Success(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	body := map[string]interface{}{
		"plan_code":      "monthly",
		"payment_method": "stripe",
		"auto_renew":     true,
	}

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscribe", body, nil)

	AssertV2Status(t, w, http.StatusCreated)
	resp := AssertV2Success(t, w)

	data := resp["data"].(map[string]interface{})
	if data["plan_code"] != "monthly" {
		t.Errorf("expected plan_code='monthly', got %v", data["plan_code"])
	}
	if data["status"].(string) != repo.SubscriptionStatusPending {
		t.Errorf("expected status=pending, got %v", data["status"])
	}
}

func TestCancelSubscriptionV2_NotOwned(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create a subscription for admin user
	sub := SeedV2Subscription(t, ctx, ctx.AdminUser.Id, repo.SubscriptionStatusActive)

	// Try to cancel as normal user
	path := fmt.Sprintf("/api/v2/test-tenant/billing/subscriptions/%d/cancel", sub.Id)
	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, path, nil, nil)

	AssertV2Status(t, w, http.StatusForbidden)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Access denied" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestCancelSubscriptionV2_NotActive(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create a pending subscription
	sub := SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusPending)

	// Try to cancel
	path := fmt.Sprintf("/api/v2/test-tenant/billing/subscriptions/%d/cancel", sub.Id)
	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, path, nil, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Only active subscriptions can be cancelled" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestCancelSubscriptionV2_Success(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create an active subscription
	sub := SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusActive)

	// Cancel it
	path := fmt.Sprintf("/api/v2/test-tenant/billing/subscriptions/%d/cancel", sub.Id)
	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, path, nil, nil)

	AssertV2Status(t, w, http.StatusOK)
	AssertV2Success(t, w)

	// Verify it's cancelled
	var updatedSub repo.Subscription
	ctx.DB.First(&updatedSub, sub.Id)
	if updatedSub.Status != repo.SubscriptionStatusCancelled {
		t.Errorf("expected status=cancelled, got %s", updatedSub.Status)
	}
}

func TestGetSubscriptionsV2_Success(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create subscriptions
	SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusActive)
	SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusExpired)

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodGet, "/api/v2/test-tenant/billing/subscriptions", nil, nil)

	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)

	data := resp["data"].(map[string]interface{})
	subscriptions := data["subscriptions"].([]interface{})
	if len(subscriptions) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subscriptions))
	}

	// Should have an active subscription
	active := data["active"]
	if active == nil {
		t.Error("expected active subscription to be returned")
	}
}

func TestCancelSubscriptionV2_TenantMismatch(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create a subscription in a different tenant
	sub := SeedV2Subscription(t, ctx, ctx.NormalUser.Id, repo.SubscriptionStatusActive)
	sub.TenantId = "other-tenant-123"
	ctx.DB.Save(sub)

	// Try to cancel from the test tenant
	path := fmt.Sprintf("/api/v2/test-tenant/billing/subscriptions/%d/cancel", sub.Id)
	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, path, nil, nil)

	AssertV2Status(t, w, http.StatusForbidden)
}

func TestSubscribeV2_AllPlanCodes(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	plans := []string{"weekly", "monthly", "quarterly", "yearly"}

	for i, plan := range plans {
		// Use a different user for each plan to avoid "already subscribed" error
		// For simplicity, we'll just delete any existing subscriptions
		ctx.DB.Where("user_id = ?", ctx.NormalUser.Id).Delete(&repo.Subscription{})

		body := map[string]interface{}{
			"plan_code":      plan,
			"payment_method": "stripe",
		}

		w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscribe", body, nil)

		if w.Code != http.StatusCreated {
			t.Errorf("plan %s: expected status 201, got %d", plan, w.Code)
			continue
		}

		resp := AssertV2Success(t, w)
		data := resp["data"].(map[string]interface{})
		if data["plan_code"] != plan {
			t.Errorf("iteration %d: expected plan_code=%s, got %v", i, plan, data["plan_code"])
		}

		// Clean up for next iteration
		ctx.DB.Where("user_id = ?", ctx.NormalUser.Id).Delete(&repo.Subscription{})
	}
}

func TestCancelSubscriptionV2_InvalidID(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscriptions/invalid/cancel", nil, nil)

	AssertV2Status(t, w, http.StatusBadRequest)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Invalid subscription ID" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestCancelSubscriptionV2_NotFound(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodPost, "/api/v2/test-tenant/billing/subscriptions/99999/cancel", nil, nil)

	AssertV2Status(t, w, http.StatusNotFound)
	resp := ParseV2Response(t, w)
	if msg, ok := resp["message"].(string); ok {
		if msg != "Subscription not found" {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}

func TestGetTopUpsV2_Pagination(t *testing.T) {
	ctx := SetupV2TestRouter(t)
	defer ctx.Cleanup()

	// Create 15 topups
	for i := 0; i < 15; i++ {
		SeedV2TopUp(t, ctx, ctx.NormalUser.Id, common.TopUpStatusPending)
	}

	// Get first page
	w := V2RequestAsUser(ctx, ctx.NormalUser, http.MethodGet, "/api/v2/test-tenant/billing/topups?page=1&page_size=10", nil, nil)
	AssertV2Status(t, w, http.StatusOK)
	resp := AssertV2Success(t, w)

	data := resp["data"].(map[string]interface{})
	topups := data["topups"].([]interface{})
	if len(topups) != 10 {
		t.Errorf("expected 10 topups on first page, got %d", len(topups))
	}

	total := int(data["total"].(float64))
	if total != 15 {
		t.Errorf("expected total=15, got %d", total)
	}

	// Get second page
	w = V2RequestAsUser(ctx, ctx.NormalUser, http.MethodGet, "/api/v2/test-tenant/billing/topups?page=2&page_size=10", nil, nil)
	resp = AssertV2Success(t, w)

	data = resp["data"].(map[string]interface{})
	topups = data["topups"].([]interface{})
	if len(topups) != 5 {
		t.Errorf("expected 5 topups on second page, got %d", len(topups))
	}
}
