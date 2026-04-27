package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestApiKey_Create_SHA256Hash(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	rawKey, apiKey, err := CreateInternalApiKey("test-key", []string{ScopeAll}, 1, 0, "test")
	if err != nil {
		t.Fatalf("CreateInternalApiKey() failed: %v", err)
	}

	// Verify hash matches SHA256 of raw key
	h := sha256.Sum256([]byte(rawKey))
	expectedHash := hex.EncodeToString(h[:])
	if apiKey.KeyHash != expectedHash {
		t.Errorf("KeyHash = %q, want SHA256(%q) = %q", apiKey.KeyHash, rawKey, expectedHash)
	}
}

func TestApiKey_Validate_ValidKey(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	rawKey, _, err := CreateInternalApiKey("valid-key", []string{ScopeAll}, 1, 0, "test")
	if err != nil {
		t.Fatalf("CreateInternalApiKey() failed: %v", err)
	}

	validated, err := ValidateInternalApiKey(rawKey)
	if err != nil {
		t.Fatalf("ValidateInternalApiKey() failed: %v", err)
	}
	if validated.Name != "valid-key" {
		t.Errorf("Name = %q, want %q", validated.Name, "valid-key")
	}
}

func TestApiKey_Validate_InvalidKey(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, err := ValidateInternalApiKey("lurus_ik_nonexistent_random_string_here")
	if err == nil {
		t.Error("expected error for invalid key, got nil")
	}
}

func TestApiKey_Validate_DisabledKey(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	rawKey, apiKey, err := CreateInternalApiKey("disabled-key", []string{ScopeAll}, 1, 0, "test")
	if err != nil {
		t.Fatalf("CreateInternalApiKey() failed: %v", err)
	}

	// Disable the key
	DB.Model(&InternalApiKey{}).Where("id = ?", apiKey.Id).Update("enabled", false)

	_, err = ValidateInternalApiKey(rawKey)
	if err == nil {
		t.Error("expected error for disabled key, got nil")
	}
}

func TestApiKey_Validate_ExpiredKey(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	// Create key that expired 1 hour ago
	pastExpiry := time.Now().Add(-1 * time.Hour).Unix()
	rawKey, _, err := CreateInternalApiKey("expired-key", []string{ScopeAll}, 1, pastExpiry, "test")
	if err != nil {
		t.Fatalf("CreateInternalApiKey() failed: %v", err)
	}

	_, err = ValidateInternalApiKey(rawKey)
	if err == nil {
		t.Error("expected error for expired key, got nil")
	}
}

func TestApiKey_Validate_UpdatesLastUsed(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	rawKey, apiKey, err := CreateInternalApiKey("lastused-key", []string{ScopeAll}, 1, 0, "test")
	if err != nil {
		t.Fatalf("CreateInternalApiKey() failed: %v", err)
	}

	// Validate to trigger last_used_at update (happens in goroutine)
	_, err = ValidateInternalApiKey(rawKey)
	if err != nil {
		t.Fatalf("ValidateInternalApiKey() failed: %v", err)
	}

	// Sleep briefly to let the async goroutine complete
	time.Sleep(200 * time.Millisecond)

	var updated InternalApiKey
	DB.First(&updated, "id = ?", apiKey.Id)
	if updated.LastUsedAt == 0 {
		t.Error("LastUsedAt should be > 0 after validation")
	}
}

func TestApiKey_HasScope_Wildcard(t *testing.T) {
	scopesJSON, _ := json.Marshal([]string{ScopeAll})
	key := &InternalApiKey{Scopes: string(scopesJSON)}

	testScopes := []string{ScopeUserRead, ScopeUserWrite, ScopeQuotaRead, "anything:custom"}
	for _, s := range testScopes {
		if !key.HasScope(s) {
			t.Errorf("HasScope(%q) = false for wildcard key, want true", s)
		}
	}
}

func TestApiKey_HasScope_Specific(t *testing.T) {
	scopesJSON, _ := json.Marshal([]string{ScopeUserRead})
	key := &InternalApiKey{Scopes: string(scopesJSON)}

	if !key.HasScope(ScopeUserRead) {
		t.Error("HasScope(user:read) = false, want true")
	}
}

func TestApiKey_HasScope_Missing(t *testing.T) {
	scopesJSON, _ := json.Marshal([]string{ScopeUserRead})
	key := &InternalApiKey{Scopes: string(scopesJSON)}

	if key.HasScope(ScopeUserWrite) {
		t.Error("HasScope(user:write) = true for user:read-only key, want false")
	}
}

// Ensure common import is used
var _ = common.GetTimestamp
