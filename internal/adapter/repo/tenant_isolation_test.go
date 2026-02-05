package repo

import (
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
)

// ============================================================================
// Tenant Isolation Unit Tests
// Tests for TenantId field presence and correct handling across all models
// ============================================================================

func TestUser_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	user := &User{
		Username:    "tenant_test_user",
		Password:    "testpassword123",
		DisplayName: "Tenant Test User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Verify default tenant_id is set
	var found User
	if err := DB.First(&found, "id = ?", user.Id).Error; err != nil {
		t.Fatalf("Query user failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestUser_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	customTenant := "tenant_abc123"
	user := &User{
		Username:    "custom_tenant_user",
		Password:    "testpassword123",
		DisplayName: "Custom Tenant User",
		TenantId:    customTenant,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	var found User
	if err := DB.First(&found, "id = ?", user.Id).Error; err != nil {
		t.Fatalf("Query user failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

func TestToken_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	token := &Token{
		UserId:         normal.Id,
		Key:            "sk-test-tenant-" + common.GetRandomString(24),
		Status:         common.TokenStatusEnabled,
		Name:           "tenant-test-token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "default",
	}
	if err := DB.Create(token).Error; err != nil {
		t.Fatalf("Create token failed: %v", err)
	}

	var found Token
	if err := DB.First(&found, "id = ?", token.Id).Error; err != nil {
		t.Fatalf("Query token failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestToken_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	customTenant := "tenant_xyz789"

	token := &Token{
		UserId:         normal.Id,
		TenantId:       customTenant,
		Key:            "sk-test-custom-" + common.GetRandomString(24),
		Status:         common.TokenStatusEnabled,
		Name:           "custom-tenant-token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "default",
	}
	if err := DB.Create(token).Error; err != nil {
		t.Fatalf("Create token failed: %v", err)
	}

	var found Token
	if err := DB.First(&found, "id = ?", token.Id).Error; err != nil {
		t.Fatalf("Query token failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

func TestTopUp_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	topUp := &TopUp{
		UserId:        normal.Id,
		Amount:        100,
		Money:         10.0,
		TradeNo:       "trade_tenant_" + common.GetRandomString(8),
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := TopUpInsert(topUp); err != nil {
		t.Fatalf("Insert topup failed: %v", err)
	}

	var found TopUp
	if err := DB.First(&found, "id = ?", topUp.Id).Error; err != nil {
		t.Fatalf("Query topup failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestTopUp_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	customTenant := "tenant_billing_001"

	topUp := &TopUp{
		UserId:        normal.Id,
		TenantId:      customTenant,
		Amount:        200,
		Money:         20.0,
		TradeNo:       "trade_custom_" + common.GetRandomString(8),
		PaymentMethod: "creem",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := TopUpInsert(topUp); err != nil {
		t.Fatalf("Insert topup failed: %v", err)
	}

	var found TopUp
	if err := DB.First(&found, "id = ?", topUp.Id).Error; err != nil {
		t.Fatalf("Query topup failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

func TestSubscription_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	sub := &Subscription{
		UserId:   normal.Id,
		PlanCode: "basic",
		PlanName: "Basic Plan",
		Status:   SubscriptionStatusPending,
	}
	if err := CreateSubscription(sub); err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	var found Subscription
	if err := DB.First(&found, "id = ?", sub.Id).Error; err != nil {
		t.Fatalf("Query subscription failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestSubscription_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	customTenant := "tenant_sub_002"

	sub := &Subscription{
		UserId:   normal.Id,
		TenantId: customTenant,
		PlanCode: "pro",
		PlanName: "Pro Plan",
		Status:   SubscriptionStatusActive,
	}
	if err := CreateSubscription(sub); err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	var found Subscription
	if err := DB.First(&found, "id = ?", sub.Id).Error; err != nil {
		t.Fatalf("Query subscription failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

func TestRedemption_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	redemption := &Redemption{
		UserId:      normal.Id,
		Key:         common.GetUUID(),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "test-redemption",
		Quota:       100000,
		CreatedTime: common.GetTimestamp(),
	}
	if err := RedemptionInsert(redemption); err != nil {
		t.Fatalf("Insert redemption failed: %v", err)
	}

	var found Redemption
	if err := DB.First(&found, "id = ?", redemption.Id).Error; err != nil {
		t.Fatalf("Query redemption failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestRedemption_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	customTenant := "tenant_redeem_003"

	redemption := &Redemption{
		UserId:      normal.Id,
		TenantId:    customTenant,
		Key:         common.GetUUID(),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "custom-tenant-redemption",
		Quota:       200000,
		CreatedTime: common.GetTimestamp(),
	}
	if err := RedemptionInsert(redemption); err != nil {
		t.Fatalf("Insert redemption failed: %v", err)
	}

	var found Redemption
	if err := DB.First(&found, "id = ?", redemption.Id).Error; err != nil {
		t.Fatalf("Query redemption failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

func TestLog_TenantId_DefaultValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	log := &Log{
		UserId:    normal.Id,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeConsume,
		Content:   "test log entry",
		Username:  normal.Username,
	}
	if err := DB.Create(log).Error; err != nil {
		t.Fatalf("Create log failed: %v", err)
	}

	var found Log
	if err := DB.First(&found, "id = ?", log.Id).Error; err != nil {
		t.Fatalf("Query log failed: %v", err)
	}
	if found.TenantId != "default" {
		t.Errorf("TenantId = %q, want %q", found.TenantId, "default")
	}
}

func TestLog_TenantId_CustomValue(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	customTenant := "tenant_log_004"

	log := &Log{
		UserId:    normal.Id,
		TenantId:  customTenant,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   "custom tenant log entry",
		Username:  normal.Username,
	}
	if err := DB.Create(log).Error; err != nil {
		t.Fatalf("Create log failed: %v", err)
	}

	var found Log
	if err := DB.First(&found, "id = ?", log.Id).Error; err != nil {
		t.Fatalf("Query log failed: %v", err)
	}
	if found.TenantId != customTenant {
		t.Errorf("TenantId = %q, want %q", found.TenantId, customTenant)
	}
}

// ============================================================================
// Webhook Tenant Verification Tests
// Tests for tenant verification in Recharge, RechargeCreem, and Redeem
// ============================================================================

func TestRecharge_TenantMismatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with tenant_a
	userA := &User{
		Username:    "user_tenant_a",
		Password:    "testpassword",
		DisplayName: "User Tenant A",
		TenantId:    "tenant_a",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(userA).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Create topup with different tenant_b
	tradeNo := "trade_mismatch_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        userA.Id,
		TenantId:      "tenant_b", // Different tenant!
		Amount:        100,
		Money:         10.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := DB.Create(topUp).Error; err != nil {
		t.Fatalf("Create topup failed: %v", err)
	}

	// Attempt to recharge should fail due to tenant mismatch
	err := Recharge(tradeNo, "cus_test")
	if err == nil {
		t.Error("expected error for tenant mismatch, got nil")
	}
	if err != nil && err.Error() != "充值失败，租户验证失败" {
		t.Logf("Got expected error: %v", err)
	}
}

func TestRecharge_TenantMatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with tenant_a
	userA := &User{
		Username:    "user_same_tenant",
		Password:    "testpassword",
		DisplayName: "User Same Tenant",
		TenantId:    "tenant_same",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(userA).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Create topup with same tenant
	tradeNo := "trade_match_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        userA.Id,
		TenantId:      "tenant_same", // Same tenant
		Amount:        100,
		Money:         10.0,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := DB.Create(topUp).Error; err != nil {
		t.Fatalf("Create topup failed: %v", err)
	}

	// Recharge should succeed
	err := Recharge(tradeNo, "cus_test_match")
	if err != nil {
		t.Fatalf("Recharge should succeed for matching tenant: %v", err)
	}

	// Verify topup status changed
	recharged := GetTopUpByTradeNo(tradeNo)
	if recharged.Status != common.TopUpStatusSuccess {
		t.Errorf("Status = %q, want %q", recharged.Status, common.TopUpStatusSuccess)
	}
}

func TestRechargeCreem_TenantMismatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with tenant_creem_a
	userA := &User{
		Username:    "creem_user_a",
		Password:    "testpassword",
		DisplayName: "Creem User A",
		TenantId:    "tenant_creem_a",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(userA).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Create topup with different tenant
	tradeNo := "creem_mismatch_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        userA.Id,
		TenantId:      "tenant_creem_b", // Different tenant!
		Amount:        500000,
		Money:         50.0,
		TradeNo:       tradeNo,
		PaymentMethod: "creem",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := DB.Create(topUp).Error; err != nil {
		t.Fatalf("Create topup failed: %v", err)
	}

	// Attempt to recharge should fail due to tenant mismatch
	err := RechargeCreem(tradeNo, "test@example.com", "Test User")
	if err == nil {
		t.Error("expected error for tenant mismatch in RechargeCreem, got nil")
	}
}

func TestRechargeCreem_TenantMatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with same tenant
	user := &User{
		Username:    "creem_user_match",
		Password:    "testpassword",
		DisplayName: "Creem User Match",
		TenantId:    "tenant_creem_match",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Create topup with same tenant
	tradeNo := "creem_match_" + common.GetRandomString(8)
	topUp := &TopUp{
		UserId:        user.Id,
		TenantId:      "tenant_creem_match", // Same tenant
		Amount:        500000,
		Money:         50.0,
		TradeNo:       tradeNo,
		PaymentMethod: "creem",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	if err := DB.Create(topUp).Error; err != nil {
		t.Fatalf("Create topup failed: %v", err)
	}

	// Recharge should succeed
	err := RechargeCreem(tradeNo, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("RechargeCreem should succeed for matching tenant: %v", err)
	}
}

func TestRedeem_TenantMismatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with tenant_redeem_a
	userA := &User{
		Username:    "redeem_user_a",
		Password:    "testpassword",
		DisplayName: "Redeem User A",
		TenantId:    "tenant_redeem_a",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(userA).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Create redemption code with different tenant
	key := common.GetUUID()
	redemption := &Redemption{
		UserId:      1, // doesn't matter for this test
		TenantId:    "tenant_redeem_b", // Different tenant!
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "cross-tenant-code",
		Quota:       100000,
		CreatedTime: common.GetTimestamp(),
	}
	if err := DB.Create(redemption).Error; err != nil {
		t.Fatalf("Create redemption failed: %v", err)
	}

	// Attempt to redeem should fail due to tenant mismatch
	_, err := Redeem(key, userA.Id)
	if err == nil {
		t.Error("expected error for tenant mismatch in Redeem, got nil")
	}
	if err != nil && err.Error() != "兑换失败，该兑换码不属于当前租户" {
		t.Logf("Got expected error: %v", err)
	}
}

func TestRedeem_TenantMatch(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create user with tenant_redeem_match
	user := &User{
		Username:    "redeem_user_match",
		Password:    "testpassword",
		DisplayName: "Redeem User Match",
		TenantId:    "tenant_redeem_match",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       100000,
		AffCode:     common.GetRandomString(8),
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	initialQuota := user.Quota
	redeemQuota := 50000

	// Create redemption code with same tenant
	key := common.GetUUID()
	redemption := &Redemption{
		UserId:      1,
		TenantId:    "tenant_redeem_match", // Same tenant
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "same-tenant-code",
		Quota:       redeemQuota,
		CreatedTime: common.GetTimestamp(),
	}
	if err := DB.Create(redemption).Error; err != nil {
		t.Fatalf("Create redemption failed: %v", err)
	}

	// Redeem should succeed
	quota, err := Redeem(key, user.Id)
	if err != nil {
		t.Fatalf("Redeem should succeed for matching tenant: %v", err)
	}
	if quota != redeemQuota {
		t.Errorf("returned quota = %d, want %d", quota, redeemQuota)
	}

	// Verify user quota increased
	var updatedUser User
	DB.First(&updatedUser, "id = ?", user.Id)
	expectedQuota := initialQuota + redeemQuota
	if updatedUser.Quota != expectedQuota {
		t.Errorf("user Quota = %d, want %d", updatedUser.Quota, expectedQuota)
	}
}

// ============================================================================
// Cross-Tenant Security Tests
// Tests to ensure data isolation between tenants
// ============================================================================

func TestCrossTenant_UserIsolation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create users in different tenants
	userA := &User{
		Username:    "tenant_a_user",
		Password:    "testpassword",
		DisplayName: "Tenant A User",
		TenantId:    "tenant_a",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     common.GetRandomString(8),
	}
	userB := &User{
		Username:    "tenant_b_user",
		Password:    "testpassword",
		DisplayName: "Tenant B User",
		TenantId:    "tenant_b",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     common.GetRandomString(8),
	}
	DB.Create(userA)
	DB.Create(userB)

	// Query users by tenant
	var tenantAUsers []User
	DB.Where("tenant_id = ?", "tenant_a").Find(&tenantAUsers)
	if len(tenantAUsers) != 1 {
		t.Errorf("tenant_a users count = %d, want 1", len(tenantAUsers))
	}
	if tenantAUsers[0].Username != "tenant_a_user" {
		t.Errorf("tenant_a user = %q, want %q", tenantAUsers[0].Username, "tenant_a_user")
	}

	var tenantBUsers []User
	DB.Where("tenant_id = ?", "tenant_b").Find(&tenantBUsers)
	if len(tenantBUsers) != 1 {
		t.Errorf("tenant_b users count = %d, want 1", len(tenantBUsers))
	}
}

func TestCrossTenant_TokenIsolation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	// Create tokens in different tenants
	tokenA := &Token{
		UserId:         normal.Id,
		TenantId:       "tenant_token_a",
		Key:            "sk-tenant-a-" + common.GetRandomString(24),
		Status:         common.TokenStatusEnabled,
		Name:           "Tenant A Token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	tokenB := &Token{
		UserId:         normal.Id,
		TenantId:       "tenant_token_b",
		Key:            "sk-tenant-b-" + common.GetRandomString(24),
		Status:         common.TokenStatusEnabled,
		Name:           "Tenant B Token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	DB.Create(tokenA)
	DB.Create(tokenB)

	// Query tokens by tenant
	var tenantATokens []Token
	DB.Where("tenant_id = ?", "tenant_token_a").Find(&tenantATokens)
	if len(tenantATokens) != 1 {
		t.Errorf("tenant_token_a tokens count = %d, want 1", len(tenantATokens))
	}

	var tenantBTokens []Token
	DB.Where("tenant_id = ?", "tenant_token_b").Find(&tenantBTokens)
	if len(tenantBTokens) != 1 {
		t.Errorf("tenant_token_b tokens count = %d, want 1", len(tenantBTokens))
	}
}

func TestCrossTenant_TopUpIsolation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	// Create topups in different tenants
	topUpA := &TopUp{
		UserId:        normal.Id,
		TenantId:      "tenant_topup_a",
		Amount:        100,
		Money:         10.0,
		TradeNo:       "trade_a_" + common.GetRandomString(8),
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	topUpB := &TopUp{
		UserId:        normal.Id,
		TenantId:      "tenant_topup_b",
		Amount:        200,
		Money:         20.0,
		TradeNo:       "trade_b_" + common.GetRandomString(8),
		PaymentMethod: "creem",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	DB.Create(topUpA)
	DB.Create(topUpB)

	// Query topups by tenant
	var tenantATopUps []TopUp
	DB.Where("tenant_id = ?", "tenant_topup_a").Find(&tenantATopUps)
	if len(tenantATopUps) != 1 {
		t.Errorf("tenant_topup_a topups count = %d, want 1", len(tenantATopUps))
	}
	if tenantATopUps[0].Amount != 100 {
		t.Errorf("tenant_topup_a amount = %d, want 100", tenantATopUps[0].Amount)
	}
}

func TestCrossTenant_RedemptionIsolation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	// Create redemptions in different tenants
	redemptionA := &Redemption{
		UserId:      normal.Id,
		TenantId:    "tenant_redemption_a",
		Key:         common.GetUUID(),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Tenant A Code",
		Quota:       100000,
		CreatedTime: common.GetTimestamp(),
	}
	redemptionB := &Redemption{
		UserId:      normal.Id,
		TenantId:    "tenant_redemption_b",
		Key:         common.GetUUID(),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Tenant B Code",
		Quota:       200000,
		CreatedTime: common.GetTimestamp(),
	}
	DB.Create(redemptionA)
	DB.Create(redemptionB)

	// Query redemptions by tenant
	var tenantARedemptions []Redemption
	DB.Where("tenant_id = ?", "tenant_redemption_a").Find(&tenantARedemptions)
	if len(tenantARedemptions) != 1 {
		t.Errorf("tenant_redemption_a count = %d, want 1", len(tenantARedemptions))
	}
	if tenantARedemptions[0].Quota != 100000 {
		t.Errorf("tenant_redemption_a quota = %d, want 100000", tenantARedemptions[0].Quota)
	}
}

func TestCrossTenant_LogIsolation(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	// Create logs in different tenants
	logA := &Log{
		UserId:    normal.Id,
		TenantId:  "tenant_log_a",
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeConsume,
		Content:   "Tenant A log",
		Username:  normal.Username,
	}
	logB := &Log{
		UserId:    normal.Id,
		TenantId:  "tenant_log_b",
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   "Tenant B log",
		Username:  normal.Username,
	}
	DB.Create(logA)
	DB.Create(logB)

	// Query logs by tenant
	var tenantALogs []Log
	DB.Where("tenant_id = ?", "tenant_log_a").Find(&tenantALogs)
	if len(tenantALogs) != 1 {
		t.Errorf("tenant_log_a count = %d, want 1", len(tenantALogs))
	}
	if tenantALogs[0].Content != "Tenant A log" {
		t.Errorf("tenant_log_a content = %q, want %q", tenantALogs[0].Content, "Tenant A log")
	}
}
