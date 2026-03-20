package common

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const (
	// walletCachePrefix is the Redis key prefix for cached wallet available balance.
	walletCachePrefix = "wallet:avail:"
	// walletCacheTTL is how long a cached balance is trusted (30 seconds).
	walletCacheTTL = 30 * time.Second
	// walletTrustThreshold is the minimum available balance (in LB) to skip pre-auth.
	// Users with balance above this threshold get fast-path (no pre-auth call).
	walletTrustThreshold = 10.0 // 10 LB — well above typical single-request cost
)

// GetCachedWalletBalance returns the cached available wallet balance for an account.
// Returns (balance, true) if cached, or (0, false) if not cached or Redis unavailable.
func GetCachedWalletBalance(accountID int64) (float64, bool) {
	if !RedisEnabled || RDB == nil {
		return 0, false
	}
	key := fmt.Sprintf("%s%d", walletCachePrefix, accountID)
	val, err := RDB.Get(context.Background(), key).Result()
	if err != nil {
		return 0, false
	}
	balance, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}
	return balance, true
}

// SetCachedWalletBalance updates the cached wallet balance after a successful pre-auth or settle.
func SetCachedWalletBalance(accountID int64, balance float64) {
	if !RedisEnabled || RDB == nil {
		return
	}
	key := fmt.Sprintf("%s%d", walletCachePrefix, accountID)
	RDB.Set(context.Background(), key, fmt.Sprintf("%.4f", balance), walletCacheTTL)
}

// InvalidateCachedWalletBalance removes the cached balance (e.g., after settle or topup).
func InvalidateCachedWalletBalance(accountID int64) {
	if !RedisEnabled || RDB == nil {
		return
	}
	key := fmt.Sprintf("%s%d", walletCachePrefix, accountID)
	RDB.Del(context.Background(), key)
}

// ShouldSkipPreAuth returns true if the user has a high cached balance,
// meaning we can trust them to pay and skip the synchronous pre-auth call.
// This reduces latency for premium users who rarely exhaust their balance.
func ShouldSkipPreAuth(accountID int64, estimatedLB float64) bool {
	balance, ok := GetCachedWalletBalance(accountID)
	if !ok {
		return false
	}
	return balance > walletTrustThreshold && balance > estimatedLB*3
}
