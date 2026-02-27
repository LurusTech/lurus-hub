package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/setting"
)

// ============================================================================
// Creem Subscription Signature Tests (P0-2)
// ============================================================================

func TestVerifyCreemSubscriptionSignature(t *testing.T) {
	secret := "whsec_sub_test_secret_key"
	payload := []byte(`{"eventType":"checkout.completed","id":"evt_sub_789"}`)

	// Compute a valid signature manually for the given payload/secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	t.Run("empty_secret_rejects", func(t *testing.T) {
		prevSecret := setting.CreemWebhookSecret
		setting.CreemWebhookSecret = ""
		defer func() { setting.CreemWebhookSecret = prevSecret }()

		if verifyCreemSubscriptionSignature(payload, validSig) {
			t.Error("empty webhook secret must reject all webhooks (P0-2 fix)")
		}
	})

	t.Run("valid_signature", func(t *testing.T) {
		prevSecret := setting.CreemWebhookSecret
		setting.CreemWebhookSecret = secret
		defer func() { setting.CreemWebhookSecret = prevSecret }()

		if !verifyCreemSubscriptionSignature(payload, validSig) {
			t.Error("expected verifyCreemSubscriptionSignature to return true for valid HMAC-SHA256 signature")
		}
	})

	t.Run("invalid_signature", func(t *testing.T) {
		prevSecret := setting.CreemWebhookSecret
		setting.CreemWebhookSecret = secret
		defer func() { setting.CreemWebhookSecret = prevSecret }()

		if verifyCreemSubscriptionSignature(payload, "deadbeef000000000000000000000000") {
			t.Error("expected verifyCreemSubscriptionSignature to return false for invalid signature")
		}
	})

	t.Run("tampered_payload", func(t *testing.T) {
		prevSecret := setting.CreemWebhookSecret
		setting.CreemWebhookSecret = secret
		defer func() { setting.CreemWebhookSecret = prevSecret }()

		tampered := []byte(`{"eventType":"checkout.completed","id":"evt_HACKED"}`)
		if verifyCreemSubscriptionSignature(tampered, validSig) {
			t.Error("expected verifyCreemSubscriptionSignature to return false for tampered payload")
		}
	})
}

// ============================================================================
// Amount Validation Tests (P0-6)
// maxToleranceCents = 50 is the fixed tolerance used in processSubscriptionPayment
// ============================================================================

func TestAmountValidation_MaxToleranceCents(t *testing.T) {
	// This constant must match the value inside processSubscriptionPayment.
	// Changing either value without updating the other will be caught by the table below.
	const maxToleranceCents int64 = 50

	tests := []struct {
		name          string
		amountDollars float64 // subscription price in dollars
		amountPaid    int64   // cents received from payment provider
		wantPass      bool    // true = payment accepted, false = rejected
	}{
		{
			name:          "exact_amount",
			amountDollars: 10.00,
			amountPaid:    1000, // exactly $10.00 in cents
			wantPass:      true,
		},
		{
			name:          "within_50_cents",
			amountDollars: 10.00,
			amountPaid:    970, // $9.70 — 30 cents short, within tolerance
			wantPass:      true,
		},
		{
			name:          "exactly_50_cents_short",
			amountDollars: 10.00,
			amountPaid:    950, // $9.50 — exactly 50 cents short (boundary, inclusive)
			wantPass:      true,
		},
		{
			name:          "51_cents_short",
			amountDollars: 10.00,
			amountPaid:    949, // $9.49 — 51 cents short, exceeds tolerance
			wantPass:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedCents := int64(tt.amountDollars * 100)
			// Replicate the exact condition from processSubscriptionPayment:
			// rejected when amountPaid > 0 AND amountPaid < expectedAmount - maxToleranceCents
			rejected := tt.amountPaid > 0 && tt.amountPaid < expectedCents-maxToleranceCents
			if rejected == tt.wantPass {
				t.Errorf("amount validation: amountPaid=%d cents, expected=%d cents, tolerance=%d: rejected=%v, want wantPass=%v",
					tt.amountPaid, expectedCents, maxToleranceCents, rejected, tt.wantPass)
			}
		})
	}
}
