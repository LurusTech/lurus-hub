package app

import (
	"testing"

	relaycommon "github.com/LurusTech/lurus-api/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
)

func TestPreConsumeQuota_SufficientQuota_TrustedToken_Succeeds(t *testing.T) {
	db := setupServiceTestDB(t)
	trustQuota := common.GetTrustQuota()
	highQuota := trustQuota + 500_000
	userId := seedTestUser(t, db, highQuota)
	_, tokenId := seedTestToken(t, db, userId, highQuota, false)

	c := createTestGinContext()
	c.Set("token_quota", highQuota) // above trust quota

	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenId:        tokenId,
		TokenKey:       "test-key",
		TokenUnlimited: false,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr != nil {
		t.Errorf("expected nil error, got: %v", apiErr.Error())
	}
	// Trusted path: pre-consumed quota should be zeroed
	if relayInfo.FinalPreConsumedQuota != 0 {
		t.Errorf("expected FinalPreConsumedQuota=0, got %d", relayInfo.FinalPreConsumedQuota)
	}
}

func TestPreConsumeQuota_InsufficientQuota_ReturnsError(t *testing.T) {
	db := setupServiceTestDB(t)
	userId := seedTestUser(t, db, 0) // zero quota

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenUnlimited: true,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr == nil {
		t.Fatal("expected error for user with zero quota")
	}
}

func TestPreConsumeQuota_NegativeQuota_ReturnsError(t *testing.T) {
	db := setupServiceTestDB(t)
	userId := seedTestUser(t, db, 100) // small positive quota

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenUnlimited: true,
	}

	// preConsumedQuota exceeds user quota
	apiErr := PreConsumeQuota(c, 200, relayInfo)
	if apiErr == nil {
		t.Fatal("expected error when preConsumedQuota exceeds user quota")
	}
}

func TestPreConsumeQuota_TrustQuota_SkipsPreConsume(t *testing.T) {
	db := setupServiceTestDB(t)
	// Create user with quota greater than trust quota (10 * QuotaPerUnit)
	trustQuota := common.GetTrustQuota()
	highQuota := trustQuota + 1_000_000
	userId := seedTestUser(t, db, highQuota)
	tokenKey, tokenId := seedTestToken(t, db, userId, highQuota, false)

	c := createTestGinContext()
	c.Set("token_quota", highQuota) // token also has enough quota

	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenId:        tokenId,
		TokenKey:       tokenKey,
		TokenUnlimited: false,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr != nil {
		t.Errorf("expected nil error for trusted user, got: %v", apiErr.Error())
	}
	// When both user and token have sufficient quota (above trust threshold),
	// preConsumedQuota should be set to 0
	if relayInfo.FinalPreConsumedQuota != 0 {
		t.Errorf("expected FinalPreConsumedQuota=0 (trusted), got %d", relayInfo.FinalPreConsumedQuota)
	}
}

func TestPreConsumeQuota_UnlimitedToken_TrustSkips(t *testing.T) {
	db := setupServiceTestDB(t)
	trustQuota := common.GetTrustQuota()
	highQuota := trustQuota + 1_000_000
	userId := seedTestUser(t, db, highQuota)

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenUnlimited: true,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr != nil {
		t.Errorf("expected nil error for unlimited token with high quota, got: %v", apiErr.Error())
	}
	// Unlimited token with high user quota => trusted, no pre-consume
	if relayInfo.FinalPreConsumedQuota != 0 {
		t.Errorf("expected FinalPreConsumedQuota=0, got %d", relayInfo.FinalPreConsumedQuota)
	}
}

func TestPreConsumeQuota_UserQuotaRecordedInRelayInfo(t *testing.T) {
	db := setupServiceTestDB(t)
	trustQuota := common.GetTrustQuota()
	userQuota := trustQuota + 100_000
	userId := seedTestUser(t, db, userQuota)

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenUnlimited: true,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr != nil {
		t.Fatalf("expected nil error, got: %v", apiErr.Error())
	}

	// Verify user quota is recorded in relayInfo
	if relayInfo.UserQuota != userQuota {
		t.Errorf("expected UserQuota=%d, got %d", userQuota, relayInfo.UserQuota)
	}
}

func TestPreConsumeQuota_LowQuotaButPositive_ReturnsError(t *testing.T) {
	db := setupServiceTestDB(t)
	// Quota is positive but less than the preConsumedQuota
	userId := seedTestUser(t, db, 50)

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         userId,
		TokenUnlimited: true,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr == nil {
		t.Fatal("expected error when user quota < preConsumedQuota")
	}
}

func TestPreConsumeQuota_NonExistentUser_ReturnsError(t *testing.T) {
	_ = setupServiceTestDB(t)

	c := createTestGinContext()
	relayInfo := &relaycommon.RelayInfo{
		UserId:         999999, // non-existent
		TokenUnlimited: true,
	}

	apiErr := PreConsumeQuota(c, 100, relayInfo)
	if apiErr == nil {
		t.Fatal("expected error for non-existent user")
	}
}
