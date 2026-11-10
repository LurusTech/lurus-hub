package app

// A hard failure debiting a real limited tenant pool used to return the same
// false as "no pool", and PostConsumeQuota still settled the platform wallet.
// Advisory must not override that: the wallet is the charge we are refusing.
// Phase 2.5 never fails the user-facing response.

import (
	"context"
	"sync/atomic"
	"testing"

	relaycommon "github.com/LurusTech/lurus-hub/internal/adapter/provider/common"
	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestPostConsumeQuota_LostPoolDebit_SkipsPlatformWallet(t *testing.T) {
	db := setupServiceTestDB(t)
	seedPoolTables(t, db)
	isolateBizTPMWindow(t)
	legacyDebitEnv(t)

	userId := seedTestUser(t, db, 100_000)
	tokenId := seedTenantToken(t, db, userId, "t-pool-lost-wallet")
	if _, err := repo.CreateTenantCreditPool("t-pool-lost-wallet", userId, 1_000, repo.PoolResetMonthly, 80); err != nil {
		t.Fatalf("create pool: %v", err)
	}
	// Balance stays 0, so DebitPool returns exhausted and the overdraft draw
	// insert fails with the ledger table gone. The pool row itself remains:
	// this is a real non-unlimited pool whose debit was lost, not a skip.
	if err := db.Migrator().DropTable(&entity.TenantCreditPoolDraw{}); err != nil {
		t.Fatalf("drop draws: %v", err)
	}

	var settles, debits int32
	prevSettle := settleWithBreaker
	settleWithBreaker = func(context.Context, int64, float64) (*common.SettlePreAuthResult, error) {
		atomic.AddInt32(&settles, 1)
		return &common.SettlePreAuthResult{}, nil
	}
	t.Cleanup(func() { settleWithBreaker = prevSettle })
	prevDebit := debitWalletGRPC
	debitWalletGRPC = func(context.Context, int64, float64, string, string, string, string) (*common.DebitWalletResult, error) {
		atomic.AddInt32(&debits, 1)
		return &common.DebitWalletResult{}, nil
	}
	t.Cleanup(func() { debitWalletGRPC = prevDebit })
	prevAsync := AsyncGo
	AsyncGo = func(f func()) { f() }
	t.Cleanup(func() { AsyncGo = prevAsync })

	// WalletAuthoritative forces advisory. Reusing localQuotaConsistent=false
	// would still settle under advisory, which is the bug.
	preAuth := &relaycommon.RelayInfo{
		UserId:              userId,
		TokenId:             tokenId,
		IdentityAccountID:   77,
		PlatformPreAuthID:   91001,
		PlatformGoverned:    true,
		WalletAuthoritative: true,
	}
	if err := PostConsumeQuota(preAuth, 700, 0, false); err != nil {
		t.Fatalf("pre-auth PostConsumeQuota: %v — a lost pool debit must not fail the response", err)
	}
	if preAuth.PlatformPreAuthID != 0 {
		t.Errorf("pre-auth id = %d, want 0 — the hold must be released, not left frozen", preAuth.PlatformPreAuthID)
	}
	if got := atomic.LoadInt32(&settles); got != 0 {
		t.Errorf("settle calls = %d, want 0 — wallet settled after the pool debit was lost", got)
	}

	legacy := &relaycommon.RelayInfo{
		UserId:              userId,
		TokenId:             tokenId,
		IdentityAccountID:   78,
		PlatformPreAuthID:   0,
		PlatformGoverned:    true,
		WalletAuthoritative: true,
	}
	if err := PostConsumeQuota(legacy, 700, 0, false); err != nil {
		t.Fatalf("legacy PostConsumeQuota: %v — a lost pool debit must not fail the response", err)
	}
	if got := atomic.LoadInt32(&debits); got != 0 {
		t.Errorf("debitWalletGRPC calls = %d, want 0 — legacy wallet debit fired after the pool debit was lost", got)
	}
	if legacy.WalletChargeCNY4 != 0 {
		t.Errorf("WalletChargeCNY4 = %d, want 0", legacy.WalletChargeCNY4)
	}

	got, err := repo.GetTenantCreditPool("t-pool-lost-wallet")
	if err != nil {
		t.Fatalf("readback pool: %v", err)
	}
	if got.CurrentBalance != 0 {
		t.Errorf("pool balance = %d, want 0 (lost debit must not move the pool)", got.CurrentBalance)
	}
	if q := userQuota(t, db, userId); q != 100_000-700-700 {
		t.Errorf("user quota = %d, want %d — local user ledger still settles; only the wallet is skipped", q, 100_000-700-700)
	}
}
