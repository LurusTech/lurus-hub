package repo

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var testDBCounter atomic.Int64

// SetupTestDB opens an in-memory SQLite database, runs AutoMigrate for all
// required tables, and wires the package-level DB / LOG_DB globals.
// It returns a cleanup function that closes the database and resets globals.
func SetupTestDB(t *testing.T) func() {
	t.Helper()

	// Use unique database name to avoid conflicts between parallel tests
	dbName := fmt.Sprintf("file:repotest%d?mode=memory&cache=shared", testDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	// Migrate tables one by one to handle SQLite global index name conflicts
	tables := []interface{}{
		&User{},
		&Token{},
		&Log{},
		&InternalApiKey{},
		&Subscription{},
		&TopUp{},
		&InvitationCode{},
		&Setup{},
		&Tenant{},
		&UserIdentityMapping{},
		&TenantConfig{},
		&Option{},
		&Channel{},
		&Ability{},
		&Midjourney{},
		&Task{},
		&QuotaData{},
		&Redemption{},
	}
	for _, tbl := range tables {
		if err := db.AutoMigrate(tbl); err != nil {
			// SQLite uses global index names (unlike PostgreSQL which scopes to table).
			// Multiple models share the same composite index name (e.g., idx_tenant_user),
			// causing "already exists" errors that are safe to ignore during migration.
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			t.Fatalf("failed to auto-migrate %T: %v", tbl, err)
		}
	}

	// Save previous state
	prevDB := DB
	prevLogDB := LOG_DB
	prevSQLite := common.UsingSQLite
	prevPG := common.UsingPostgreSQL
	prevMySQL := common.UsingMySQL

	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false
	common.RedisEnabled = false

	// Initialize OptionMap to prevent nil map panic
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMapRWMutex.Unlock()

	initCol()

	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		DB = prevDB
		LOG_DB = prevLogDB
		common.UsingSQLite = prevSQLite
		common.UsingPostgreSQL = prevPG
		common.UsingMySQL = prevMySQL
	}

	t.Cleanup(cleanup)
	return cleanup
}

// SeedTestUsers creates three users: root (role 100), normal (role 1), and
// disabled (role 1, status disabled). All passwords are hashed via
// common.Password2Hash. Returns pointers to the created User records.
func SeedTestUsers(t *testing.T) (root, normal, disabled *User) {
	t.Helper()

	hash := func(plain string) string {
		t.Helper()
		h, err := common.Password2Hash(plain)
		if err != nil {
			t.Fatalf("Password2Hash(%q) failed: %v", plain, err)
		}
		return h
	}

	root = &User{
		Username:    "testroot",
		Password:    hash("rootpassword"),
		DisplayName: "Test Root",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		Email:       "root@test.local",
		Quota:       100_000_000,
		Group:       "default",
		AffCode:     common.GetRandomString(8),
	}

	normal = &User{
		Username:    "testnormal",
		Password:    hash("normalpassword"),
		DisplayName: "Test Normal",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       "normal@test.local",
		Quota:       1_000_000,
		Group:       "default",
		AffCode:     common.GetRandomString(8),
	}

	disabled = &User{
		Username:    "testdisabled",
		Password:    hash("disabledpassword"),
		DisplayName: "Test Disabled",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusDisabled,
		Email:       "disabled@test.local",
		Quota:       0,
		Group:       "default",
		AffCode:     common.GetRandomString(8),
	}

	for _, u := range []*User{root, normal, disabled} {
		if err := DB.Create(u).Error; err != nil {
			t.Fatalf("failed to seed user %q: %v", u.Username, err)
		}
	}

	return root, normal, disabled
}

// SeedTestApiKeys creates two InternalApiKey records: one with full access
// (ScopeAll) and one with read-only scopes. It returns the persisted key
// objects and a slice of raw (unhashed) key strings in the same order
// [fullRawKey, readOnlyRawKey].
func SeedTestApiKeys(t *testing.T) (fullKey, readOnlyKey *InternalApiKey, rawKeys []string) {
	t.Helper()

	makeKey := func(name, rawKey string, scopes []string, createdBy int, expiresAt int64, desc string) *InternalApiKey {
		t.Helper()
		scopesJSON, err := json.Marshal(scopes)
		if err != nil {
			t.Fatalf("json.Marshal scopes: %v", err)
		}
		k := &InternalApiKey{
			Name:        name,
			KeyHash:     hashKey(rawKey),
			KeyPrefix:   rawKey[:16],
			Scopes:      string(scopesJSON),
			CreatedBy:   createdBy,
			CreatedAt:   common.GetTimestamp(),
			ExpiresAt:   expiresAt,
			Enabled:     true,
			Description: desc,
		}
		if err := DB.Create(k).Error; err != nil {
			t.Fatalf("failed to seed api key %q: %v", name, err)
		}
		return k
	}

	fullRaw := "lurus_ik_" + common.GetRandomString(32)
	readOnlyRaw := "lurus_ik_" + common.GetRandomString(32)

	fullKey = makeKey(
		"full-access-key",
		fullRaw,
		[]string{ScopeAll},
		1,
		0, // never expires
		"Full access test key",
	)

	readOnlyKey = makeKey(
		"read-only-key",
		readOnlyRaw,
		[]string{ScopeUserRead, ScopeQuotaRead, ScopeSubscriptionRead, ScopeTokenRead, ScopeBalanceRead},
		1,
		time.Now().Add(24*time.Hour).Unix(),
		"Read-only test key",
	)

	rawKeys = []string{fullRaw, readOnlyRaw}
	return fullKey, readOnlyKey, rawKeys
}

// SeedTestTokens creates several Token records for the given userId:
//   - an active unlimited-quota token
//   - an active token with limited quota (500000 remain)
//   - an expired token
//   - a disabled token
func SeedTestTokens(t *testing.T, userId int) {
	t.Helper()

	now := common.GetTimestamp()

	tokens := []Token{
		{
			UserId:         userId,
			Key:            "sk-test-unlimited-" + common.GetRandomString(24),
			Status:         common.TokenStatusEnabled,
			Name:           "unlimited-token",
			CreatedTime:    now,
			AccessedTime:   now,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "default",
		},
		{
			UserId:         userId,
			Key:            "sk-test-limited-" + common.GetRandomString(26),
			Status:         common.TokenStatusEnabled,
			Name:           "limited-token",
			CreatedTime:    now,
			AccessedTime:   now,
			ExpiredTime:    now + 86400,
			RemainQuota:    int(common.QuotaPerUnit),
			UnlimitedQuota: false,
			Group:          "default",
		},
		{
			UserId:      userId,
			Key:         "sk-test-expired-" + common.GetRandomString(26),
			Status:      common.TokenStatusEnabled,
			Name:        "expired-token",
			CreatedTime: now - 172800,
			ExpiredTime: now - 86400, // expired yesterday
			Group:       "default",
		},
		{
			UserId:         userId,
			Key:            "sk-test-disabled-" + common.GetRandomString(25),
			Status:         common.TokenStatusEnabled + 1, // disabled status
			Name:           "disabled-token",
			CreatedTime:    now,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "default",
		},
	}

	for i := range tokens {
		if err := DB.Create(&tokens[i]).Error; err != nil {
			t.Fatalf("failed to seed token %q: %v", tokens[i].Name, err)
		}
	}
}
