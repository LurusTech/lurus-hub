package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// BillingUnifiedEnabled controls whether the pre-authorize billing path is active.
// When true and IdentityAccountID > 0, relay requests use freeze/settle/release.
// When false, the legacy fire-and-forget DebitWallet path is used.
var BillingUnifiedEnabled = os.Getenv("BILLING_UNIFIED_ENABLED") == "true"

// PreAuthResult holds the response from a wallet pre-authorization call.
type PreAuthResult struct {
	PreAuthID int64   `json:"preauth_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	ExpiresAt string  `json:"expires_at"`
}

// PreAuthorize freezes an estimated amount in the wallet before LLM relay.
// Returns the pre-auth ID for later settle/release, or an error (e.g. insufficient balance).
func PreAuthorize(ctx context.Context, accountID int64, amount float64, productID, referenceID, description string, ttlSeconds int) (*PreAuthResult, error) {
	if IdentityServiceURL == "" {
		return nil, fmt.Errorf("identity service not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"amount":      amount,
		"product_id":  productID,
		"reference_id": referenceID,
		"description": description,
		"ttl_seconds": ttlSeconds,
	})
	url := fmt.Sprintf("%s/internal/v1/accounts/%d/wallet/pre-authorize", IdentityServiceURL, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pre-authorize request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("pre-authorize failed: status %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("pre-authorize failed: status %d", resp.StatusCode)
	}
	var result PreAuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// SettlePreAuthResult holds the response from a settle call.
type SettlePreAuthResult struct {
	PreAuthID    int64   `json:"preauth_id"`
	Status       string  `json:"status"`
	HeldAmount   float64 `json:"held_amount"`
	ActualAmount float64 `json:"actual_amount"`
}

// SettlePreAuth settles a pre-authorization with the actual consumed amount.
func SettlePreAuth(ctx context.Context, preAuthID int64, actualAmount float64) (*SettlePreAuthResult, error) {
	if IdentityServiceURL == "" {
		return nil, fmt.Errorf("identity service not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"actual_amount": actualAmount,
	})
	url := fmt.Sprintf("%s/internal/v1/wallet/pre-auth/%d/settle", IdentityServiceURL, preAuthID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("settle request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("settle failed: status %d", resp.StatusCode)
	}
	var result SettlePreAuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ReleasePreAuth cancels a pre-authorization and unfreezes the held amount.
func ReleasePreAuth(ctx context.Context, preAuthID int64) error {
	if IdentityServiceURL == "" {
		return fmt.Errorf("identity service not configured")
	}
	url := fmt.Sprintf("%s/internal/v1/wallet/pre-auth/%d/release", IdentityServiceURL, preAuthID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		return fmt.Errorf("release request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release failed: status %d", resp.StatusCode)
	}
	return nil
}

// billingGRPCTimeout wraps a context with a 5s timeout for billing gRPC calls.
func billingGRPCTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(grpcCtx(ctx), 5*time.Second)
}
