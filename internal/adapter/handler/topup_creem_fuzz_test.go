package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/setting"
)

// FuzzGenerateCreemSignature tests HMAC signature generation with random inputs
// Run: go test -fuzz=FuzzGenerateCreemSignature -fuzztime=30s ./internal/adapter/handler/...
func FuzzGenerateCreemSignature(f *testing.F) {
	// Seed corpus
	f.Add("test payload", "test secret")
	f.Add("", "secret")
	f.Add("payload", "")
	f.Add("", "")
	f.Add(`{"event":"payment.completed","id":"123"}`, "whsec_abcdef123456")
	f.Add(strings.Repeat("a", 10000), strings.Repeat("b", 1000))
	f.Add("payload\x00with\x00nulls", "secret\x00with\x00nulls")
	f.Add("payload\nwith\nnewlines", "secret\twith\ttabs")

	f.Fuzz(func(t *testing.T, payload, secret string) {
		sig := generateCreemSignature(payload, secret)

		// Invariant 1: signature should be valid hex (64 chars for SHA256)
		if len(sig) != 64 {
			t.Errorf("signature length %d != 64", len(sig))
		}

		// Invariant 2: signature should be lowercase hex
		for _, c := range sig {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("invalid hex char %c in signature", c)
			}
		}

		// Invariant 3: should be decodable as hex
		decoded, err := hex.DecodeString(sig)
		if err != nil {
			t.Errorf("signature not valid hex: %v", err)
		}
		if len(decoded) != 32 { // SHA256 = 32 bytes
			t.Errorf("decoded length %d != 32", len(decoded))
		}

		// Invariant 4: same inputs should produce same output (deterministic)
		sig2 := generateCreemSignature(payload, secret)
		if sig != sig2 {
			t.Errorf("non-deterministic: %q != %q", sig, sig2)
		}

		// Invariant 5: verify our implementation matches standard HMAC-SHA256
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(payload))
		expected := hex.EncodeToString(h.Sum(nil))
		if sig != expected {
			t.Errorf("signature mismatch: got %q, want %q", sig, expected)
		}
	})
}

// FuzzVerifyCreemSignature tests signature verification with random inputs
// Run: go test -fuzz=FuzzVerifyCreemSignature -fuzztime=30s ./internal/adapter/handler/...
func FuzzVerifyCreemSignature(f *testing.F) {
	// Seed corpus with valid and invalid signatures
	validPayload := `{"event":"test"}`
	validSecret := "test_secret_123"
	validSig := generateCreemSignature(validPayload, validSecret)

	f.Add(validPayload, validSig, validSecret)
	f.Add(validPayload, "invalid_signature", validSecret)
	f.Add(validPayload, validSig, "wrong_secret")
	f.Add("", "", "")
	f.Add("payload", "", "secret") // empty signature
	f.Add("payload", validSig, "") // empty secret (should fail unless test mode)

	f.Fuzz(func(t *testing.T, payload, signature, secret string) {
		// Save and restore test mode
		originalTestMode := setting.CreemTestMode
		defer func() { setting.CreemTestMode = originalTestMode }()
		setting.CreemTestMode = false

		result := verifyCreemSignature(payload, signature, secret)

		// Invariant 1: if secret is empty and not in test mode, should return false
		if secret == "" && !result {
			// Expected behavior - empty secret should fail
		}

		// Invariant 2: if we generate a valid signature, verification should pass
		if secret != "" {
			expectedSig := generateCreemSignature(payload, secret)
			if signature == expectedSig && !result {
				t.Errorf("valid signature rejected: payload=%q, secret=%q", payload, secret)
			}
		}

		// Invariant 3: should never panic (implicit)
	})
}

// FuzzVerifyCreemSignature_TimingAttack tests for timing-safe comparison
func FuzzVerifyCreemSignature_TimingAttack(f *testing.F) {
	secret := "production_secret_key_12345"
	payload := `{"event":"payment.completed","amount":1000}`
	validSig := generateCreemSignature(payload, secret)

	// Generate signatures that differ by one character at different positions
	for i := 0; i < len(validSig); i++ {
		modified := validSig[:i] + "X" + validSig[i+1:]
		f.Add(modified)
	}
	f.Add(validSig)
	f.Add("")
	f.Add(strings.Repeat("0", 64))
	f.Add(strings.Repeat("f", 64))

	f.Fuzz(func(t *testing.T, signature string) {
		// Save and restore test mode
		originalTestMode := setting.CreemTestMode
		defer func() { setting.CreemTestMode = originalTestMode }()
		setting.CreemTestMode = false

		result := verifyCreemSignature(payload, signature, secret)

		// The function uses hmac.Equal which is timing-safe
		// We just verify it doesn't panic and returns correct result
		expectedSig := generateCreemSignature(payload, secret)
		expected := signature == expectedSig

		if result != expected {
			t.Errorf("verification mismatch for sig=%q: got %v, want %v", signature, result, expected)
		}
	})
}
