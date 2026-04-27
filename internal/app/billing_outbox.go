package app

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/LurusTech/lurus-hub/internal/pkg/metrics"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxActionSettle  = "settle"
	outboxActionRelease = "release"
	outboxMaxRetries    = 10
)

// billingOutboxDB is set during initialization and used by the outbox worker.
var billingOutboxDB *gorm.DB

// InitBillingOutbox sets the DB handle and auto-migrates the outbox table.
func InitBillingOutbox(db *gorm.DB) error {
	billingOutboxDB = db
	return db.AutoMigrate(&entity.BillingOutbox{})
}

// EnqueueSettle writes a settle action to the outbox for reliable retry.
func EnqueueSettle(accountID, preAuthID int64, amountLB float64) error {
	if billingOutboxDB == nil {
		slog.Error("billing outbox not initialized, settle lost", "preauth_id", preAuthID, "amount", amountLB)
		return fmt.Errorf("billing outbox not initialized")
	}
	entry := entity.BillingOutbox{
		AccountID: accountID,
		PreAuthID: preAuthID,
		Action:    outboxActionSettle,
		AmountLB:  amountLB,
		Status:    "pending",
		NextRetry: time.Now(),
	}
	if err := billingOutboxDB.Create(&entry).Error; err != nil {
		slog.Error("billing outbox enqueue settle failed", "preauth_id", preAuthID, "err", err)
		return fmt.Errorf("enqueue settle: %w", err)
	}
	return nil
}

// EnqueueRelease writes a release action to the outbox for reliable retry.
func EnqueueRelease(accountID, preAuthID int64) error {
	if billingOutboxDB == nil {
		slog.Error("billing outbox not initialized, release lost", "preauth_id", preAuthID)
		return fmt.Errorf("billing outbox not initialized")
	}
	entry := entity.BillingOutbox{
		AccountID: accountID,
		PreAuthID: preAuthID,
		Action:    outboxActionRelease,
		AmountLB:  0,
		Status:    "pending",
		NextRetry: time.Now(),
	}
	if err := billingOutboxDB.Create(&entry).Error; err != nil {
		slog.Error("billing outbox enqueue release failed", "preauth_id", preAuthID, "err", err)
		return fmt.Errorf("enqueue release: %w", err)
	}
	return nil
}

// ProcessBillingOutbox polls pending entries and retries them.
// Uses FOR UPDATE SKIP LOCKED to prevent multi-pod double-processing.
func ProcessBillingOutbox(ctx context.Context) error {
	if billingOutboxDB == nil {
		return nil
	}

	// Accurate pending count for metrics (including not-yet-ready entries)
	var pendingCount int64
	billingOutboxDB.Model(&entity.BillingOutbox{}).Where("status = ?", "pending").Count(&pendingCount)
	metrics.BillingOutboxPending.Set(float64(pendingCount))

	// Claim entries atomically: SELECT ... FOR UPDATE SKIP LOCKED prevents
	// two pods from processing the same entry simultaneously.
	var entries []entity.BillingOutbox
	if err := billingOutboxDB.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND next_retry <= ?", "pending", time.Now()).
		Order("next_retry ASC").
		Limit(50).
		Find(&entries).Error; err != nil {
		return fmt.Errorf("query outbox: %w", err)
	}

	for i := range entries {
		entry := &entries[i]
		processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		var err error

		switch entry.Action {
		case outboxActionSettle:
			_, err = common.SettlePreAuthGRPC(processCtx, entry.PreAuthID, entry.AmountLB)
		case outboxActionRelease:
			err = common.ReleasePreAuthGRPC(processCtx, entry.PreAuthID)
		default:
			err = fmt.Errorf("unknown action: %s", entry.Action)
		}
		cancel()

		if err == nil {
			// Atomic update: only mark done if still pending (guard against race)
			billingOutboxDB.Model(&entity.BillingOutbox{}).
				Where("id = ? AND status = ?", entry.ID, "pending").
				Updates(map[string]any{"status": "done", "error": ""})
			slog.Info("billing outbox processed", "id", entry.ID, "action", entry.Action, "preauth_id", entry.PreAuthID)
		} else {
			entry.RetryCount++
			entry.Error = err.Error()
			newStatus := "pending"
			if entry.RetryCount >= outboxMaxRetries {
				newStatus = "failed"
				metrics.BillingOutboxFailedTotal.Inc()
				slog.Error("billing outbox permanently failed", "id", entry.ID, "action", entry.Action, "preauth_id", entry.PreAuthID, "err", err)
			} else {
				backoff := time.Duration(math.Pow(2, float64(entry.RetryCount))) * 5 * time.Second
				entry.NextRetry = time.Now().Add(backoff)
				slog.Warn("billing outbox retry scheduled", "id", entry.ID, "retry", entry.RetryCount, "next", entry.NextRetry, "err", err)
			}
			// Atomic update: only update if still pending
			billingOutboxDB.Model(&entity.BillingOutbox{}).
				Where("id = ? AND status = ?", entry.ID, "pending").
				Updates(map[string]any{
					"retry_count": entry.RetryCount,
					"next_retry":  entry.NextRetry,
					"status":      newStatus,
					"error":       entry.Error,
				})
		}
	}

	return nil
}
