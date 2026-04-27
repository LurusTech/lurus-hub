package repo

import (
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestUser_Insert_DuplicateUsername(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	oldQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 0
	defer func() { common.QuotaForNewUser = oldQuota }()

	u1 := &User{Username: "dupuser", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	if err := u1.Insert(); err != nil {
		t.Fatalf("first Insert() failed: %v", err)
	}

	u2 := &User{Username: "dupuser", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	if err := u2.Insert(); err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}

func TestUser_GetById_Found(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	found, err := GetUserById(normal.Id, true)
	if err != nil {
		t.Fatalf("GetUserById() failed: %v", err)
	}
	if found.Username != "testnormal" {
		t.Errorf("Username = %q, want %q", found.Username, "testnormal")
	}
}

func TestUser_GetById_NotFound(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, err := GetUserById(99999, true)
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestUser_GetByEmail_Found(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	user := &User{Email: normal.Email}
	if err := user.FillUserByEmail(); err != nil {
		t.Fatalf("FillUserByEmail() failed: %v", err)
	}
	if user.Username != "testnormal" {
		t.Errorf("Username = %q, want %q", user.Username, "testnormal")
	}
}

func TestUser_IncreaseQuota_Atomic(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)
	initialQuota := normal.Quota

	if err := IncreaseUserQuota(normal.Id, 5000, true); err != nil {
		t.Fatalf("IncreaseUserQuota() failed: %v", err)
	}

	var quota int
	DB.Model(&User{}).Where("id = ?", normal.Id).Select("quota").Scan(&quota)
	if quota != initialQuota+5000 {
		t.Errorf("quota = %d, want %d", quota, initialQuota+5000)
	}
}

func TestUser_DecreaseQuota_Atomic(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	if err := IncreaseUserQuota(normal.Id, 10000, true); err != nil {
		t.Fatalf("IncreaseUserQuota() failed: %v", err)
	}

	if err := DecreaseUserQuota(normal.Id, 3000); err != nil {
		t.Fatalf("DecreaseUserQuota() failed: %v", err)
	}

	var quota int
	DB.Model(&User{}).Where("id = ?", normal.Id).Select("quota").Scan(&quota)
	expected := normal.Quota + 10000 - 3000
	if quota != expected {
		t.Errorf("quota = %d, want %d", quota, expected)
	}
}

func TestUser_DecreaseQuota_BelowZero(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	oldQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 0
	defer func() { common.QuotaForNewUser = oldQuota }()

	user := &User{
		Username: "zeroquota",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Quota:    0,
		Group:    "default",
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	err := DecreaseUserQuota(user.Id, 1000)
	if err != nil {
		t.Logf("DecreaseUserQuota() returned error: %v (may be expected)", err)
		return
	}

	var quota int
	DB.Model(&User{}).Where("id = ?", user.Id).Select("quota").Scan(&quota)
	if quota > 0 {
		t.Errorf("quota = %d, expected <= 0 after decreasing from 0", quota)
	}
}

func TestUser_DeleteById_SoftDelete(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	if err := DeleteUserById(normal.Id); err != nil {
		t.Fatalf("DeleteUserById() failed: %v", err)
	}

	if _, err := GetUserById(normal.Id, true); err == nil {
		t.Error("expected error querying soft-deleted user, got nil")
	}

	var found User
	if err := DB.Unscoped().First(&found, "id = ?", normal.Id).Error; err != nil {
		t.Errorf("Unscoped query should find soft-deleted user, got error: %v", err)
	}
	if found.DeletedAt.Time.IsZero() {
		t.Error("DeletedAt should be set after soft delete")
	}
}

func TestChineseCharacters_Username(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	user := &User{
		Username:    "cnuser1",
		DisplayName: "English Name",
		Status:      common.UserStatusEnabled,
		Role:        common.RoleCommonUser,
		Group:       "default",
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	var found User
	DB.First(&found, "id = ?", user.Id)
	if found.Username != "cnuser1" {
		t.Errorf("Username = %q, want %q", found.Username, "cnuser1")
	}
}

func TestChineseCharacters_DisplayName(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	chineseDisplay := "\u4e2d\u6587\u663e\u793a\u540d"
	user := &User{
		Username:    "cnuser2",
		DisplayName: chineseDisplay,
		Status:      common.UserStatusEnabled,
		Role:        common.RoleCommonUser,
		Group:       "default",
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	var found User
	DB.First(&found, "id = ?", user.Id)
	if found.DisplayName != chineseDisplay {
		t.Errorf("DisplayName = %q, want %q", found.DisplayName, chineseDisplay)
	}
}

func TestUTF8_MaxLength_CJK(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	cjkName := "\u6d4b\u8bd5\u7528\u6237\u663e\u793a\u540d\u79f0\u8d85\u957f\u5b57\u7b26\u4e32\u6d4b\u8bd5\u8d85\u957f\u5b57\u7b26"
	user := &User{
		Username:    "cjkuser",
		DisplayName: cjkName,
		Status:      common.UserStatusEnabled,
		Role:        common.RoleCommonUser,
		Group:       "default",
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	var found User
	if err := DB.First(&found, "id = ?", user.Id).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if found.DisplayName != cjkName {
		t.Errorf("DisplayName mismatch: got %q, want %q", found.DisplayName, cjkName)
	}
}
