package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/setting"
)

// ============================================================================
// Creem Signature Function Tests
// Security-critical pure/near-pure functions
// ============================================================================

func TestGenerateCreemSignature(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		secret  string
	}{
		{
			name:    "known_payload",
			payload: `{"event":"checkout.completed","id":"evt_123"}`,
			secret:  "whsec_test_secret_key",
		},
		{
			name:    "empty_payload",
			payload: "",
			secret:  "whsec_test_secret_key",
		},
		{
			name:    "different_secret",
			payload: `{"event":"checkout.completed","id":"evt_123"}`,
			secret:  "whsec_another_secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateCreemSignature(tt.payload, tt.secret)

			// Verify result is valid hex
			_, err := hex.DecodeString(got)
			if err != nil {
				t.Errorf("generateCreemSignature returned invalid hex: %q", got)
			}

			// Verify result matches manual HMAC-SHA256 computation
			h := hmac.New(sha256.New, []byte(tt.secret))
			h.Write([]byte(tt.payload))
			expected := hex.EncodeToString(h.Sum(nil))

			if got != expected {
				t.Errorf("generateCreemSignature(%q, %q) = %q, want %q", tt.payload, tt.secret, got, expected)
			}
		})
	}

	// Verify different secrets produce different hashes for the same payload
	t.Run("different_secrets_differ", func(t *testing.T) {
		payload := `{"event":"test"}`
		sig1 := generateCreemSignature(payload, "secret_a")
		sig2 := generateCreemSignature(payload, "secret_b")
		if sig1 == sig2 {
			t.Error("different secrets should produce different signatures")
		}
	})
}

func TestVerifyCreemSignature(t *testing.T) {
	secret := "whsec_test_verification_key"
	payload := `{"event":"checkout.completed","id":"evt_456"}`
	validSig := generateCreemSignature(payload, secret)

	t.Run("valid_signature", func(t *testing.T) {
		if !verifyCreemSignature(payload, validSig, secret) {
			t.Error("expected verifyCreemSignature to return true for valid signature")
		}
	})

	t.Run("invalid_signature", func(t *testing.T) {
		if verifyCreemSignature(payload, "deadbeef0000", secret) {
			t.Error("expected verifyCreemSignature to return false for invalid signature")
		}
	})

	t.Run("empty_secret_non_test_mode", func(t *testing.T) {
		prevTestMode := setting.CreemTestMode
		setting.CreemTestMode = false
		defer func() { setting.CreemTestMode = prevTestMode }()

		if verifyCreemSignature(payload, validSig, "") {
			t.Error("expected verifyCreemSignature to return false when secret is empty in non-test mode")
		}
	})

	t.Run("empty_secret_test_mode", func(t *testing.T) {
		prevTestMode := setting.CreemTestMode
		setting.CreemTestMode = true
		defer func() { setting.CreemTestMode = prevTestMode }()

		if !verifyCreemSignature(payload, "anything", "") {
			t.Error("expected verifyCreemSignature to return true when secret is empty in test mode")
		}
	})

	t.Run("tampered_payload", func(t *testing.T) {
		tampered := `{"event":"checkout.completed","id":"evt_HACKED"}`
		if verifyCreemSignature(tampered, validSig, secret) {
			t.Error("expected verifyCreemSignature to return false for tampered payload")
		}
	})
}
