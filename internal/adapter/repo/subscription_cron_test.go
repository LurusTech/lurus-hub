package repo

import (
	"testing"
	"time"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
)

// resetPlanCache clears the in-memory subscription plan cache.
// Called in defer to ensure plan isolation between tests.
func resetPlanCache() {
	subscriptionPlansMu.Lock()
	subscriptionPlansCache = nil
	subscriptionPlansMu.Unlock()
}

// TestProcessOneAutoRenewal tests the core auto-renewal logic (P0-1).
func TestProcessOneAutoRenewal(t *testing.T) {
	t.Run("monthly_plan_success", func(t *testing.T) {
		cleanup := SetupTestDB(t)
		defer cleanup()
		defer resetPlanCache()

		_, normal, _ := SeedTestUsers(t)

		// Inject a finite plan (TotalQuota=5M, Days=30, Price=1.0 CNY)
		renewalCost := int(1.0 * common.QuotaPerUnit)
		testPlan := SubscriptionPlan{
			Code:       "test_monthly",
			Name:       "Test Monthly",
			Days:       30,
			Price:      1.0,
			TotalQuota: 5_000_000,
			Enabled:    true,
		}
		subscriptionPlansMu.Lock()
		subscriptionPlansCache = []SubscriptionPlan{testPlan}
		subscriptionPlansMu.Unlock()

		// Give user enough quota to cover renewalCost
		startQuota := renewalCost + 1_000_000
		DB.Model(&User{}).Where("id = ?", normal.Id).Updates(map[string]interface{}{
			"quota":      startQuota,
			"daily_used": 500,
		})

		expiresAt := time.Now().Add(12 * time.Hour)
		sub := &Subscription{
			UserId:    normal.Id,
			PlanCode:  "test_monthly",
			PlanName:  "Test Monthly",
			Status:    SubscriptionStatusActive,
			AutoRenew: true,
			StartedAt: time.Now().Add(-30 * 24 * time.Hour),
			ExpiresAt: expiresAt,
		}
		DB.Create(sub)

		processOneAutoRenewal(sub)

		// Verify quota: startQuota - renewalCost + 5M
		var user User
		DB.First(&user, "id = ?", normal.Id)
		expectedQuota := startQuota - renewalCost + 5_000_000
		if user.Quota != expectedQuota {
			t.Errorf("Quota = %d, want %d (startQuota=%d - cost=%d + 5M)", user.Quota, expectedQuota, startQuota, renewalCost)
		}

		// Verify daily_used reset to 0
		if user.DailyUsed != 0 {
			t.Errorf("DailyUsed = %d, want 0", user.DailyUsed)
		}

		// Verify expires_at extended by 30 days
		var updatedSub Subscription
		DB.First(&updatedSub, "id = ?", sub.Id)
		expectedExpiry := expiresAt.AddDate(0, 0, 30)
		diff := updatedSub.ExpiresAt.Sub(expectedExpiry)
		if diff < -5*time.Second || diff > 5*time.Second {
			t.Errorf("ExpiresAt = %v, want ~%v", updatedSub.ExpiresAt, expectedExpiry)
		}
	})

	t.Run("yearly_plan_unlimited_quota", func(t *testing.T) {
		cleanup := SetupTestDB(t)
		defer cleanup()
		defer resetPlanCache()

		_, normal, _ := SeedTestUsers(t)

		// TotalQuota=0 means unlimited - should NOT add quota on renewal
		testPlan := SubscriptionPlan{
			Code:       "test_yearly",
			Name:       "Test Yearly",
			Days:       365,
			Price:      1.0,
			TotalQuota: 0, // unlimited
			Enabled:    true,
		}
		subscriptionPlansMu.Lock()
		subscriptionPlansCache = []SubscriptionPlan{testPlan}
		subscriptionPlansMu.Unlock()

		renewalCost := int(1.0 * common.QuotaPerUnit)
		startQuota := renewalCost + 2_000_000
		DB.Model(&User{}).Where("id = ?", normal.Id).Updates(map[string]interface{}{
			"quota":      startQuota,
			"daily_used": 999,
		})

		expiresAt := time.Now().Add(12 * time.Hour)
		sub := &Subscription{
			UserId:    normal.Id,
			PlanCode:  "test_yearly",
			Status:    SubscriptionStatusActive,
			AutoRenew: true,
			StartedAt: time.Now().Add(-365 * 24 * time.Hour),
			ExpiresAt: expiresAt,
		}
		DB.Create(sub)

		processOneAutoRenewal(sub)

		var user User
		DB.First(&user, "id = ?", normal.Id)

		// Only deduct cost, no TotalQuota addition (unlimited plan)
		expectedQuota := startQuota - renewalCost
		if user.Quota != expectedQuota {
			t.Errorf("Quota = %d, want %d (startQuota=%d - cost=%d, no TotalQuota grant)", user.Quota, expectedQuota, startQuota, renewalCost)
		}

		// daily_used must be reset
		if user.DailyUsed != 0 {
			t.Errorf("DailyUsed = %d, want 0", user.DailyUsed)
		}
	})

	t.Run("insufficient_balance_skipped", func(t *testing.T) {
		cleanup := SetupTestDB(t)
		defer cleanup()
		defer resetPlanCache()

		_, normal, _ := SeedTestUsers(t)

		// Plan costs 5M quota units but user only has 1000
		testPlan := SubscriptionPlan{
			Code:       "test_expensive",
			Name:       "Test Expensive",
			Days:       30,
			Price:      10.0, // renewalCost = 10 * QuotaPerUnit = 5M
			TotalQuota: 10_000_000,
			Enabled:    true,
		}
		subscriptionPlansMu.Lock()
		subscriptionPlansCache = []SubscriptionPlan{testPlan}
		subscriptionPlansMu.Unlock()

		startQuota := 1000
		DB.Model(&User{}).Where("id = ?", normal.Id).Update("quota", startQuota)

		expiresAt := time.Now().Add(12 * time.Hour)
		sub := &Subscription{
			UserId:    normal.Id,
			PlanCode:  "test_expensive",
			Status:    SubscriptionStatusActive,
			AutoRenew: true,
			StartedAt: time.Now().Add(-30 * 24 * time.Hour),
			ExpiresAt: expiresAt,
		}
		DB.Create(sub)

		processOneAutoRenewal(sub)

		// Quota must not change (renewal skipped)
		var user User
		DB.First(&user, "id = ?", normal.Id)
		if user.Quota != startQuota {
			t.Errorf("Quota = %d, want %d (unchanged due to insufficient balance)", user.Quota, startQuota)
		}

		// ExpiresAt must not change
		var updatedSub Subscription
		DB.First(&updatedSub, "id = ?", sub.Id)
		diff := updatedSub.ExpiresAt.Sub(expiresAt)
		if diff < -2*time.Second || diff > 2*time.Second {
			t.Errorf("ExpiresAt changed from %v to %v, expected no change", expiresAt, updatedSub.ExpiresAt)
		}
	})

	t.Run("plan_not_found", func(t *testing.T) {
		cleanup := SetupTestDB(t)
		defer cleanup()
		defer resetPlanCache()

		_, normal, _ := SeedTestUsers(t)

		// Cache has no plans
		subscriptionPlansMu.Lock()
		subscriptionPlansCache = []SubscriptionPlan{}
		subscriptionPlansMu.Unlock()

		startQuota := normal.Quota
		expiresAt := time.Now().Add(12 * time.Hour)
		sub := &Subscription{
			UserId:    normal.Id,
			PlanCode:  "nonexistent_plan",
			Status:    SubscriptionStatusActive,
			AutoRenew: true,
			ExpiresAt: expiresAt,
		}
		DB.Create(sub)

		processOneAutoRenewal(sub)

		// Nothing must change - plan not found causes early return
		var user User
		DB.First(&user, "id = ?", normal.Id)
		if user.Quota != startQuota {
			t.Errorf("Quota = %d, want %d (unchanged, plan not found)", user.Quota, startQuota)
		}
	})

	t.Run("daily_used_reset", func(t *testing.T) {
		cleanup := SetupTestDB(t)
		defer cleanup()
		defer resetPlanCache()

		_, normal, _ := SeedTestUsers(t)

		testPlan := SubscriptionPlan{
			Code:       "test_reset",
			Name:       "Test Reset",
			Days:       30,
			Price:      1.0,
			TotalQuota: 5_000_000,
			Enabled:    true,
		}
		subscriptionPlansMu.Lock()
		subscriptionPlansCache = []SubscriptionPlan{testPlan}
		subscriptionPlansMu.Unlock()

		renewalCost := int(1.0 * common.QuotaPerUnit)
		startQuota := renewalCost + 1_000_000

		// Set daily_used to a non-zero value
		DB.Model(&User{}).Where("id = ?", normal.Id).Updates(map[string]interface{}{
			"quota":      startQuota,
			"daily_used": 99999,
		})

		sub := &Subscription{
			UserId:    normal.Id,
			PlanCode:  "test_reset",
			Status:    SubscriptionStatusActive,
			AutoRenew: true,
			ExpiresAt: time.Now().Add(12 * time.Hour),
		}
		DB.Create(sub)

		processOneAutoRenewal(sub)

		var user User
		DB.First(&user, "id = ?", normal.Id)
		if user.DailyUsed != 0 {
			t.Errorf("DailyUsed = %d after renewal, want 0 (must be reset for new billing period)", user.DailyUsed)
		}
	})
}
