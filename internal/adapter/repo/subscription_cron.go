package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"gorm.io/gorm"
)

// StartSubscriptionCronJobs starts background jobs for subscription management.
// Deprecated: Use StartSubscriptionCronJobsWithContext instead.
func StartSubscriptionCronJobs() {
	StartSubscriptionCronJobsWithContext(context.Background())
}

// StartSubscriptionCronJobsWithContext starts background jobs for subscription management with context support.
// All goroutines exit when ctx is cancelled.
func StartSubscriptionCronJobsWithContext(ctx context.Context) {
	// Check expired subscriptions every 5 minutes
	common.SafeGoWithContext(ctx, subscriptionExpiryCheckerWithContext)

	// Cleanup stale pending subscriptions every hour
	common.SafeGoWithContext(ctx, stalePendingSubscriptionCleanerWithContext)

	// Process auto-renewals every hour
	common.SafeGoWithContext(ctx, autoRenewalProcessorWithContext)

	common.SysLog("Subscription cron jobs started")
}

// subscriptionExpiryCheckerWithContext periodically checks and expires subscriptions with context support
func subscriptionExpiryCheckerWithContext(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	processExpiredSubscriptions()

	for {
		select {
		case <-ctx.Done():
			common.SysLog("Subscription expiry checker stopped")
			return
		case <-ticker.C:
			processExpiredSubscriptions()
		}
	}
}

// subscriptionExpiryChecker periodically checks and expires subscriptions
// Deprecated: Use subscriptionExpiryCheckerWithContext instead.
func subscriptionExpiryChecker() {
	subscriptionExpiryCheckerWithContext(context.Background())
}

// processExpiredSubscriptions finds and expires all overdue subscriptions
func processExpiredSubscriptions() {
	batchSize := 100

	for {
		subs, err := GetExpiredSubscriptions(batchSize)
		if err != nil {
			common.SysLog("Failed to get expired subscriptions: " + err.Error())
			return
		}

		if len(subs) == 0 {
			break
		}

		for _, sub := range subs {
			if err := ExpireSubscription(sub); err != nil {
				common.SysLog(fmt.Sprintf("Failed to expire subscription %d: %s", sub.Id, err.Error()))
				continue
			}
			common.SysLog(fmt.Sprintf("Expired subscription for user %d", sub.UserId))
		}

		// If we got less than batch size, we're done
		if len(subs) < batchSize {
			break
		}
	}
}

// ProcessSubscriptionRenewals handles auto-renewal for subscriptions by deducting from user quota balance.
// Subscriptions expiring within 24 hours with auto_renew=true are processed.
func ProcessSubscriptionRenewals() {
	var subs []Subscription
	err := DB.Where(
		"status = ? AND auto_renew = ? AND expires_at < ? AND expires_at > ?",
		SubscriptionStatusActive, true,
		time.Now().Add(24*time.Hour), time.Now(),
	).Find(&subs).Error

	if err != nil {
		common.SysLog("Failed to get subscriptions for renewal: " + err.Error())
		return
	}

	for i := range subs {
		processOneAutoRenewal(&subs[i])
	}
}

// processOneAutoRenewal attempts to auto-renew a single subscription by deducting from user's quota balance.
// Renewal cost = plan.Price (CNY) converted to quota units at QuotaPerUnit rate.
// On success: deducts cost, grants new period TotalQuota, extends expires_at.
// On insufficient balance: logs warning (TODO: send email notification).
func processOneAutoRenewal(sub *Subscription) {
	plan := GetSubscriptionPlanByCode(sub.PlanCode)
	if plan == nil {
		common.SysError(fmt.Sprintf("Auto-renewal skipped: plan %q not found for subscription %d", sub.PlanCode, sub.Id))
		return
	}

	// Renewal cost in quota units (plan.Price is in CNY, converted at QuotaPerUnit rate)
	renewalCost := int(plan.Price * common.QuotaPerUnit)

	// Quick balance pre-check (non-locking, avoids unnecessary transaction overhead)
	var user User
	if err := DB.Select("id, quota").Where("id = ?", sub.UserId).First(&user).Error; err != nil {
		common.SysError(fmt.Sprintf("Auto-renewal skipped: user %d not found: %s", sub.UserId, err.Error()))
		return
	}

	if user.Quota < renewalCost {
		// TODO: Send email notification about insufficient balance for renewal
		common.SysLog(fmt.Sprintf(
			"Auto-renewal skipped: user %d insufficient balance (have %d quota, need %d for plan %s at ¥%.2f)",
			sub.UserId, user.Quota, renewalCost, sub.PlanCode, plan.Price,
		))
		return
	}

	// Execute renewal atomically: deduct cost + grant new period quota + extend expiry
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Re-lock and verify balance under transaction
		var lockedUser User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Select("id, quota").Where("id = ?", sub.UserId).First(&lockedUser).Error; err != nil {
			return fmt.Errorf("lock user failed: %w", err)
		}
		if lockedUser.Quota < renewalCost {
			return fmt.Errorf("insufficient balance after lock: have %d, need %d", lockedUser.Quota, renewalCost)
		}

		// Step 1: Deduct renewal cost (floor at 0 to prevent negative quota)
		if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
			Update("quota", quotaDeductSafe(renewalCost)).Error; err != nil {
			return fmt.Errorf("renewal cost deduction failed: %w", err)
		}

		// Step 2: Grant new period quota only for finite plans.
		// TotalQuota=0 signals unlimited quota - do not modify quota in that case.
		if plan.TotalQuota > 0 {
			if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
				Update("quota", gorm.Expr("quota + ?", plan.TotalQuota)).Error; err != nil {
				return fmt.Errorf("quota grant failed: %w", err)
			}
		}

		// Step 3: Reset daily usage for the new billing period
		if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
			Update("daily_used", 0).Error; err != nil {
			return fmt.Errorf("daily_used reset failed: %w", err)
		}

		// Extend subscription expiry from current expiry (not from now, preserving any early renewal buffer)
		newExpiry := sub.ExpiresAt.AddDate(0, 0, plan.Days)
		if err := tx.Model(sub).Updates(map[string]interface{}{
			"expires_at": newExpiry,
			"status":     SubscriptionStatusActive,
		}).Error; err != nil {
			return fmt.Errorf("subscription extend failed: %w", err)
		}

		return nil
	})

	if err != nil {
		common.SysError(fmt.Sprintf("Auto-renewal failed: subscription %d user %d: %s", sub.Id, sub.UserId, err.Error()))
		return
	}

	RecordLog(sub.UserId, LogTypeTopup, fmt.Sprintf(
		"订阅自动续费成功: %s，续费 %d 天，扣费 ¥%.2f",
		sub.PlanCode, plan.Days, plan.Price,
	))
	common.SysLog(fmt.Sprintf(
		"Auto-renewal successful: subscription %d user %d plan %s +%d days",
		sub.Id, sub.UserId, sub.PlanCode, plan.Days,
	))
}

// stalePendingSubscriptionCleanerWithContext cleans up pending subscriptions older than 24 hours with context support
func stalePendingSubscriptionCleanerWithContext(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run immediately on start (with delay to allow system to stabilize)
	select {
	case <-ctx.Done():
		common.SysLog("Stale pending subscription cleaner stopped before initial run")
		return
	case <-time.After(30 * time.Second):
		cleanupStalePendingSubscriptions()
	}

	for {
		select {
		case <-ctx.Done():
			common.SysLog("Stale pending subscription cleaner stopped")
			return
		case <-ticker.C:
			cleanupStalePendingSubscriptions()
		}
	}
}

// stalePendingSubscriptionCleaner cleans up pending subscriptions older than 24 hours
// Deprecated: Use stalePendingSubscriptionCleanerWithContext instead.
func stalePendingSubscriptionCleaner() {
	stalePendingSubscriptionCleanerWithContext(context.Background())
}

// cleanupStalePendingSubscriptions marks old pending subscriptions as expired
func cleanupStalePendingSubscriptions() {
	batchSize := 100
	// Subscriptions pending for more than 24 hours are considered stale
	staleDuration := 24 * time.Hour
	totalCleaned := 0

	for {
		subs, err := GetPendingSubscriptionsOlderThan(staleDuration, batchSize)
		if err != nil {
			common.SysError("Failed to get stale pending subscriptions: " + err.Error())
			return
		}

		if len(subs) == 0 {
			break
		}

		for _, sub := range subs {
			if err := CleanupStalePendingSubscription(sub); err != nil {
				common.SysError(fmt.Sprintf("Failed to cleanup stale subscription %d: %s", sub.Id, err.Error()))
				continue
			}
			totalCleaned++
		}

		if len(subs) < batchSize {
			break
		}
	}

	if totalCleaned > 0 {
		common.SysLog(fmt.Sprintf("Cleaned up %d stale pending subscriptions", totalCleaned))
	}
}

// autoRenewalProcessorWithContext handles automatic subscription renewals with context support
func autoRenewalProcessorWithContext(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run immediately on start (with delay)
	select {
	case <-ctx.Done():
		common.SysLog("Auto renewal processor stopped before initial run")
		return
	case <-time.After(1 * time.Minute):
		ProcessSubscriptionRenewals()
	}

	for {
		select {
		case <-ctx.Done():
			common.SysLog("Auto renewal processor stopped")
			return
		case <-ticker.C:
			ProcessSubscriptionRenewals()
		}
	}
}

// autoRenewalProcessor handles automatic subscription renewals
// Deprecated: Use autoRenewalProcessorWithContext instead.
func autoRenewalProcessor() {
	autoRenewalProcessorWithContext(context.Background())
}

// SendExpirationWarning sends warning to users whose subscriptions are expiring soon
func SendExpirationWarning() {
	// Find subscriptions expiring within 3 days
	var subs []Subscription
	err := DB.Where(
		"status = ? AND expires_at > ? AND expires_at < ?",
		SubscriptionStatusActive, time.Now(), time.Now().Add(72*time.Hour),
	).Find(&subs).Error

	if err != nil {
		common.SysError("Failed to get expiring subscriptions for warning: " + err.Error())
		return
	}

	for _, sub := range subs {
		daysRemaining := int(time.Until(sub.ExpiresAt).Hours() / 24)
		common.SysLog(fmt.Sprintf("Subscription expiring soon: user_id=%d, plan=%s, days_remaining=%d",
			sub.UserId, sub.PlanCode, daysRemaining))
		// TODO: Send email/notification to user
	}
}
