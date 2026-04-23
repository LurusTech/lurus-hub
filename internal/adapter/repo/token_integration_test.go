package repo

import (
	"testing"

	"github.com/LurusTech/lurus-api/internal/pkg/common"
)

func TestToken_Insert_GeneratesKey(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	token := &Token{
		UserId:         1,
		Key:            "sk-" + common.GetRandomString(32),
		Status:         common.TokenStatusEnabled,
		Name:           "test-token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		RemainQuota:    100000,
		UnlimitedQuota: false,
		Group:          "default",
	}
	if err := DB.Create(token).Error; err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if token.Key == "" {
		t.Error("token key should not be empty")
	}
	if token.Id == 0 {
		t.Error("token ID should be assigned after insert")
	}
}

func TestToken_GetById(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	token := &Token{
		UserId:      1,
		Key:         "sk-" + common.GetRandomString(32),
		Status:      common.TokenStatusEnabled,
		Name:        "getbyid-token",
		CreatedTime: common.GetTimestamp(),
		ExpiredTime: -1,
		RemainQuota: 50000,
		Group:       "default",
	}
	DB.Create(token)

	found, err := GetTokenById(token.Id)
	if err != nil {
		t.Fatalf("GetTokenById() failed: %v", err)
	}
	if found.Name != "getbyid-token" {
		t.Errorf("Name = %q, want %q", found.Name, "getbyid-token")
	}
}

func TestToken_CountUserTokens(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	userId := 42
	for i := 0; i < 3; i++ {
		token := &Token{
			UserId:      userId,
			Key:         "sk-" + common.GetRandomString(32),
			Status:      common.TokenStatusEnabled,
			Name:        "count-token",
			CreatedTime: common.GetTimestamp(),
			ExpiredTime: -1,
			RemainQuota: 10000,
			Group:       "default",
		}
		DB.Create(token)
	}

	count, err := CountUserTokens(userId)
	if err != nil {
		t.Fatalf("CountUserTokens() failed: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestToken_GetAllUserTokens_Pagination(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	userId := 99
	for i := 0; i < 5; i++ {
		token := &Token{
			UserId:      userId,
			Key:         "sk-" + common.GetRandomString(32),
			Status:      common.TokenStatusEnabled,
			Name:        "page-token",
			CreatedTime: common.GetTimestamp(),
			ExpiredTime: -1,
			RemainQuota: 10000,
			Group:       "default",
		}
		DB.Create(token)
	}

	tokens, err := GetAllUserTokens(userId, 0, 2)
	if err != nil {
		t.Fatalf("GetAllUserTokens() failed: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("len(tokens) = %d, want 2", len(tokens))
	}
}

func TestToken_Delete_SoftDelete(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	token := &Token{
		UserId:      1,
		Key:         "sk-" + common.GetRandomString(32),
		Status:      common.TokenStatusEnabled,
		Name:        "delete-token",
		CreatedTime: common.GetTimestamp(),
		ExpiredTime: -1,
		RemainQuota: 10000,
		Group:       "default",
	}
	DB.Create(token)

	if err := token.Delete(); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Normal query should fail
	_, err := GetTokenById(token.Id)
	if err == nil {
		t.Error("expected error querying soft-deleted token, got nil")
	}

	// Unscoped should still find it
	var found Token
	if err := DB.Unscoped().First(&found, "id = ?", token.Id).Error; err != nil {
		t.Errorf("Unscoped query should find soft-deleted token: %v", err)
	}
}

func TestToken_ValidateUserToken_Enabled(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	key := "sk-" + common.GetRandomString(32)
	token := &Token{
		UserId:         1,
		Key:            key,
		Status:         common.TokenStatusEnabled,
		Name:           "valid-token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		RemainQuota:    100000,
		UnlimitedQuota: false,
		Group:          "default",
	}
	DB.Create(token)

	validated, err := ValidateUserToken(key)
	if err != nil {
		t.Fatalf("ValidateUserToken() failed: %v", err)
	}
	if validated.Id != token.Id {
		t.Errorf("validated token Id = %d, want %d", validated.Id, token.Id)
	}
}

func TestToken_ValidateUserToken_Expired(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	key := "sk-" + common.GetRandomString(32)
	token := &Token{
		UserId:      1,
		Key:         key,
		Status:      common.TokenStatusEnabled,
		Name:        "expired-token",
		CreatedTime: common.GetTimestamp() - 172800,
		ExpiredTime: common.GetTimestamp() - 86400, // expired yesterday
		RemainQuota: 100000,
		Group:       "default",
	}
	DB.Create(token)

	_, err := ValidateUserToken(key)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestToken_ValidateUserToken_Exhausted(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	key := "sk-" + common.GetRandomString(32)
	token := &Token{
		UserId:         1,
		Key:            key,
		Status:         common.TokenStatusEnabled,
		Name:           "exhausted-token",
		CreatedTime:    common.GetTimestamp(),
		ExpiredTime:    -1,
		RemainQuota:    0,
		UnlimitedQuota: false,
		Group:          "default",
	}
	DB.Create(token)

	_, err := ValidateUserToken(key)
	if err == nil {
		t.Error("expected error for exhausted token, got nil")
	}
}
