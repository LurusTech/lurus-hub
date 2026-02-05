package repo

import (
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
)

func TestTopUp_Insert(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        100,
		Money:         10.0,
		TradeNo:       "trade_insert_" + common.GetRandomString(8),
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := TopUpInsert(topUp); err != nil {
		t.Fatalf("Insert() failed: %v", err)
	}
	if topUp.Id == 0 {
		t.Error("topup ID should be assigned after insert")
	}

	// Verify in DB
	var found TopUp
	if err := DB.First(&found, "id = ?", topUp.Id).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if found.TradeNo != topUp.TradeNo {
		t.Errorf("TradeNo = %q, want %q", found.TradeNo, topUp.TradeNo)
	}
}

func TestTopUp_GetByTradeNo(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	tradeNo := "trade_get_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        50,
		Money:         5.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	DB.Create(topUp)

	found := GetTopUpByTradeNo(tradeNo)
	if found == nil {
		t.Fatal("expected topup, got nil")
	}
	if found.Amount != 50 {
		t.Errorf("Amount = %d, want 50", found.Amount)
	}
}

func TestTopUp_Recharge_Success(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	initialQuota := normal.Quota

	tradeNo := "trade_recharge_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        100,
		Money:         2.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	DB.Create(topUp)

	if err := Recharge(tradeNo, "cus_test123"); err != nil {
		t.Fatalf("Recharge() failed: %v", err)
	}

	// Verify topup status
	recharged := GetTopUpByTradeNo(tradeNo)
	if recharged == nil {
		t.Fatal("topup should exist after recharge")
	}
	if recharged.Status != common.TopUpStatusSuccess {
		t.Errorf("Status = %q, want %q", recharged.Status, common.TopUpStatusSuccess)
	}

	// Verify user quota increased: Money * QuotaPerUnit
	var user User
	DB.First(&user, "id = ?", normal.Id)
	expectedQuota := initialQuota + int(topUp.Money*common.QuotaPerUnit)
	if user.Quota != expectedQuota {
		t.Errorf("user Quota = %d, want %d", user.Quota, expectedQuota)
	}
}

func TestTopUp_Recharge_AlreadySuccess(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	tradeNo := "trade_already_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        100,
		Money:         2.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusSuccess, // already success
	}
	DB.Create(topUp)

	err := Recharge(tradeNo, "cus_test456")
	if err == nil {
		t.Error("expected error for already-success topup, got nil")
	}
}

func TestTopUp_Recharge_OnlyPending(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	tradeNo := "trade_cancelled_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        100,
		Money:         2.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        "cancelled",
	}
	DB.Create(topUp)

	err := Recharge(tradeNo, "cus_test789")
	if err == nil {
		t.Error("expected error for cancelled topup, got nil")
	}
}
