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

// IdentityServiceURL is the base URL for lurus-identity service.
var IdentityServiceURL = getIdentityServiceURL()

func getIdentityServiceURL() string {
	if url := os.Getenv("IDENTITY_SERVICE_URL"); url != "" {
		return url
	}
	return "http://identity-service.lurus-identity.svc.cluster.local:18104"
}

// IdentityServiceInternalKey is the bearer token for /internal/v1/* endpoints.
var IdentityServiceInternalKey = os.Getenv("IDENTITY_SERVICE_INTERNAL_KEY")

var identityClient = &http.Client{
	Timeout: 5 * time.Second,
}

// IdentityMapping represents the unified user identity mapping returned by lurus-identity.
type IdentityMapping struct {
	ID          int64     `json:"id"`
	LurusID     string    `json:"lurus_id"`
	ZitadelSub  string    `json:"zitadel_sub"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Status      int16     `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Entitlements is a key→value map describing an account's product permissions.
type Entitlements map[string]string

// GetString returns a string entitlement value, falling back to defaultVal.
func (e Entitlements) GetString(key, defaultVal string) string {
	if v, ok := e[key]; ok {
		return v
	}
	return defaultVal
}

// GetInt returns an integer entitlement value, falling back to defaultVal.
func (e Entitlements) GetInt(key string, defaultVal int) int {
	v := e.GetString(key, "")
	if v == "" {
		return defaultVal
	}
	var i int
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
		return defaultVal
	}
	return i
}

// GetBool returns a boolean entitlement value, falling back to defaultVal.
func (e Entitlements) GetBool(key string, defaultVal bool) bool {
	v := e.GetString(key, "")
	switch v {
	case "true":
		return true
	case "false":
		return false
	default:
		return defaultVal
	}
}

// GetAccountByZitadelSub retrieves account info from lurus-identity by Zitadel OIDC sub.
// Returns nil on not-found or network errors (callers degrade gracefully).
func GetAccountByZitadelSub(ctx context.Context, sub string) (*IdentityMapping, error) {
	if IdentityServiceURL == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		IdentityServiceURL+"/internal/v1/accounts/by-zitadel-sub/"+sub,
		nil,
	)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		SysLog(fmt.Sprintf("identity GetAccountByZitadelSub: %v", err))
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		SysLog(fmt.Sprintf("identity GetAccountByZitadelSub: status %d", resp.StatusCode))
		return nil, nil
	}
	var a IdentityMapping
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, nil
	}
	return &a, nil
}

// UpsertAccount creates or updates an account in lurus-identity (called on OIDC login).
func UpsertAccount(ctx context.Context, zitadelSub, email, displayName, avatarURL string) (*IdentityMapping, error) {
	if IdentityServiceURL == "" {
		return nil, nil
	}
	body, _ := json.Marshal(map[string]string{
		"zitadel_sub":  zitadelSub,
		"email":        email,
		"display_name": displayName,
		"avatar_url":   avatarURL,
	})
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		IdentityServiceURL+"/internal/v1/accounts/upsert",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		SysLog(fmt.Sprintf("identity UpsertAccount: %v", err))
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		SysLog(fmt.Sprintf("identity UpsertAccount: status %d", resp.StatusCode))
		return nil, nil
	}
	var a IdentityMapping
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, nil
	}
	return &a, nil
}

// GetEntitlements retrieves product entitlements for an account (Redis-cached in identity service).
// Falls back to empty Entitlements map on any error — callers must handle the free/default case.
func GetEntitlements(ctx context.Context, accountID int64, productID string) (Entitlements, error) {
	if IdentityServiceURL == "" {
		return Entitlements{"plan_code": "free"}, nil
	}
	url := fmt.Sprintf("%s/internal/v1/accounts/%d/entitlements/%s", IdentityServiceURL, accountID, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Entitlements{"plan_code": "free"}, nil
	}
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		SysLog(fmt.Sprintf("identity GetEntitlements: %v", err))
		return Entitlements{"plan_code": "free"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Entitlements{"plan_code": "free"}, nil
	}
	var em Entitlements
	if err := json.NewDecoder(resp.Body).Decode(&em); err != nil {
		return Entitlements{"plan_code": "free"}, nil
	}
	return em, nil
}

// AccountOverview mirrors the aggregated read model from lurus-identity's overview endpoint.
type AccountOverview struct {
	Account struct {
		ID          int64  `json:"id"`
		LurusID     string `json:"lurus_id"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	} `json:"account"`
	VIP struct {
		Level          int16  `json:"level"`
		LevelName      string `json:"level_name"`
		LevelEN        string `json:"level_en"`
		Points         int64  `json:"points"`
		LevelExpiresAt *struct {
			Time string `json:"time"`
		} `json:"level_expires_at"`
	} `json:"vip"`
	Wallet struct {
		Balance float64 `json:"balance"`
		Frozen  float64 `json:"frozen"`
	} `json:"wallet"`
	Subscription *struct {
		ProductID string  `json:"product_id"`
		PlanCode  string  `json:"plan_code"`
		Status    string  `json:"status"`
		ExpiresAt *string `json:"expires_at"`
		AutoRenew bool    `json:"auto_renew"`
	} `json:"subscription"`
	TopupURL string `json:"topup_url"`
}

// GetAccountOverview retrieves the aggregated overview for an account from lurus-identity.
// Returns nil, nil on network errors or when identity service is not configured — callers degrade gracefully.
func GetAccountOverview(ctx context.Context, accountID int64, productID string) (*AccountOverview, error) {
	if IdentityServiceURL == "" {
		return nil, nil
	}
	url := fmt.Sprintf("%s/internal/v1/accounts/%d/overview", IdentityServiceURL, accountID)
	if productID != "" {
		url += "?product_id=" + productID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)

	resp, err := identityClient.Do(req)
	if err != nil {
		SysLog(fmt.Sprintf("identity GetAccountOverview: %v", err))
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		SysLog(fmt.Sprintf("identity GetAccountOverview: status %d", resp.StatusCode))
		return nil, nil
	}
	var ov AccountOverview
	if err := json.NewDecoder(resp.Body).Decode(&ov); err != nil {
		return nil, nil
	}
	return &ov, nil
}

// ReportLLMUsage sends a usage record to lurus-identity for VIP accumulation.
// Fire-and-forget — errors are logged but not propagated.
func ReportLLMUsage(ctx context.Context, accountID int64, amountCNY float64) {
	if IdentityServiceURL == "" {
		return
	}
	body, _ := json.Marshal(map[string]any{
		"account_id": accountID,
		"amount_cny": amountCNY,
	})
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		IdentityServiceURL+"/internal/v1/usage/report",
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+IdentityServiceInternalKey)
	resp, err := identityClient.Do(req)
	if err != nil {
		SysLog(fmt.Sprintf("identity ReportLLMUsage: %v", err))
		return
	}
	resp.Body.Close()
}

