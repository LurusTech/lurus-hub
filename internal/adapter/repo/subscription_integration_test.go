package repo

import (
	"testing"
	"time"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
)

func TestSubscription_Create(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:        normal.Id,
		PlanCode:      "monthly",
		PlanName:      "Monthly Plan",
		Status:        SubscriptionStatusPending,
		DailyQuota:    1000000,
		TotalQuota:    5000000,
		BaseGroup:     "premium",
		FallbackGroup: "default",
		StartedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
		PaymentMethod: "stripe",
		Amount:        9.99,
		Currency:      "USD",
	}

	if err := CreateSubscription(sub); err != nil {
		t.Fatalf("CreateSubscription() failed: %v", err)
	}
	if sub.Id == 0 {
		t.Error("subscription ID should be assigned after create")
	}

	// Verify in DB
	var found Subscription
	if err := DB.First(&found, "id = ?", sub.Id).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if found.PlanCode != "monthly" {
		t.Errorf("PlanCode = %q, want %q", found.PlanCode, "monthly")
	}
}

func TestSubscription_GetActive_Found(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:    normal.Id,
		PlanCode:  "monthly",
		PlanName:  "Monthly Plan",
		Status:    SubscriptionStatusActive,
		StartedAt: time.Now().Add(-24 * time.Hour),
		ExpiresAt: time.Now().Add(29 * 24 * time.Hour),
	}
	DB.Create(sub)

	active, err := GetActiveSubscription(normal.Id)
	if err != nil {
		t.Fatalf("GetActiveSubscription() failed: %v", err)
	}
	if active == nil {
		t.Fatal("expected active subscription, got nil")
	}
	if active.Id != sub.Id {
		t.Errorf("active subscription Id = %d, want %d", active.Id, sub.Id)
	}
}

func TestSubscription_GetActive_ExpiredNotReturned(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:    normal.Id,
		PlanCode:  "monthly",
		PlanName:  "Monthly Plan",
		Status:    SubscriptionStatusActive,
		StartedAt: time.Now().Add(-60 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * 24 * time.Hour), // expired yesterday
	}
	DB.Create(sub)

	active, err := GetActiveSubscription(normal.Id)
	if err != nil {
		t.Fatalf("GetActiveSubscription() failed: %v", err)
	}
	if active != nil {
		t.Errorf("expected nil for expired subscription, got Id=%d", active.Id)
	}
}

func TestSubscription_GetById(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:    normal.Id,
		PlanCode:  "yearly",
		PlanName:  "Yearly Plan",
		Status:    SubscriptionStatusActive,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}
	DB.Create(sub)

	found, err := GetSubscriptionById(sub.Id)
	if err != nil {
		t.Fatalf("GetSubscriptionById() failed: %v", err)
	}
	if found.PlanCode != "yearly" {
		t.Errorf("PlanCode = %q, want %q", found.PlanCode, "yearly")
	}
}

func TestSubscription_UpdateStatus(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:    normal.Id,
		PlanCode:  "monthly",
		PlanName:  "Monthly Plan",
		Status:    SubscriptionStatusActive,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	DB.Create(sub)

	if err := UpdateSubscriptionStatus(sub.Id, SubscriptionStatusCancelled); err != nil {
		t.Fatalf("UpdateSubscriptionStatus() failed: %v", err)
	}

	var updated Subscription
	DB.First(&updated, "id = ?", sub.Id)
	if updated.Status != SubscriptionStatusCancelled {
		t.Errorf("Status = %q, want %q", updated.Status, SubscriptionStatusCancelled)
	}
}

func TestSubscription_ActivateSubscription(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:        normal.Id,
		PlanCode:      "monthly",
		PlanName:      "Monthly Plan",
		Status:        SubscriptionStatusPending,
		DailyQuota:    500000,
		TotalQuota:    2000000,
		BaseGroup:     "premium",
		FallbackGroup: "default",
		StartedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	}
	DB.Create(sub)

	if err := ActivateSubscription(sub); err != nil {
		t.Fatalf("ActivateSubscription() failed: %v", err)
	}

	// Verify subscription status
	var updatedSub Subscription
	DB.First(&updatedSub, "id = ?", sub.Id)
	if updatedSub.Status != SubscriptionStatusActive {
		t.Errorf("sub Status = %q, want %q", updatedSub.Status, SubscriptionStatusActive)
	}

	// Verify user fields were synced
	var user User
	DB.First(&user, "id = ?", normal.Id)

	if user.DailyQuota != 500000 {
		t.Errorf("user DailyQuota = %d, want 500000", user.DailyQuota)
	}
	if user.BaseGroup != "premium" {
		t.Errorf("user BaseGroup = %q, want %q", user.BaseGroup, "premium")
	}
	if user.Group != "premium" {
		t.Errorf("user Group = %q, want %q", user.Group, "premium")
	}
	if user.Role != common.RoleSubscriberUser {
		t.Errorf("user Role = %d, want %d (subscriber)", user.Role, common.RoleSubscriberUser)
	}
	// TotalQuota is added to existing quota
	expectedQuota := normal.Quota + 2000000
	if user.Quota != expectedQuota {
		t.Errorf("user Quota = %d, want %d", user.Quota, expectedQuota)
	}
}
