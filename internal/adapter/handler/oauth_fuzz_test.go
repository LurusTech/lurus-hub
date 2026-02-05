package handler

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzParseOAuthState tests OAuth state parsing with random inputs
// Run: go test -fuzz=FuzzParseOAuthState -fuzztime=30s ./internal/adapter/handler/...
func FuzzParseOAuthState(f *testing.F) {
	// Seed corpus with valid states
	validState := OAuthStateData{
		TenantSlug:  "my-tenant",
		RedirectURL: "/dashboard",
		Nonce:       "nonce123",
		CreatedAt:   time.Now(),
	}
	validJSON, _ := json.Marshal(validState)
	validEncoded := base64.URLEncoding.EncodeToString(validJSON)

	f.Add(validEncoded)
	f.Add("")
	f.Add("!!!invalid-base64!!!")
	f.Add(base64.URLEncoding.EncodeToString([]byte("not json")))
	f.Add(base64.URLEncoding.EncodeToString([]byte("{}")))
	f.Add(base64.URLEncoding.EncodeToString([]byte(`{"tenant_slug":"a"}`)))
	f.Add(base64.URLEncoding.EncodeToString([]byte(`{"tenant_slug":"<script>alert(1)</script>"}`)))
	f.Add(strings.Repeat("A", 10000)) // large input

	// Add some edge case JSON
	edgeCases := []string{
		`{"tenant_slug":"","redirect_url":"","nonce":"","created_at":"0001-01-01T00:00:00Z"}`,
		`{"tenant_slug":null}`,
		`{"extra_field":"value","tenant_slug":"test"}`,
		`{"tenant_slug":"test","redirect_url":"javascript:alert(1)"}`,
		`{"tenant_slug":"test/../../../etc/passwd"}`,
	}
	for _, ec := range edgeCases {
		f.Add(base64.URLEncoding.EncodeToString([]byte(ec)))
	}

	f.Fuzz(func(t *testing.T, state string) {
		parsed, err := parseOAuthState(state)

		// Invariant 1: should never panic (implicit)

		// Invariant 2: if parsing succeeds, result should be non-nil
		if err == nil && parsed == nil {
			t.Error("parseOAuthState returned nil result without error")
		}

		// Invariant 3: if parsing succeeds, decoded data should be consistent
		if err == nil && parsed != nil {
			// Re-encode and decode to verify consistency
			reencoded, err2 := json.Marshal(parsed)
			if err2 != nil {
				t.Errorf("failed to re-marshal parsed state: %v", err2)
			}
			var reparsed OAuthStateData
			if err3 := json.Unmarshal(reencoded, &reparsed); err3 != nil {
				t.Errorf("failed to re-unmarshal state: %v", err3)
			}
		}

		// Invariant 4: empty input should fail
		if state == "" && err == nil {
			t.Error("empty state should fail parsing")
		}
	})
}

// FuzzGenerateOAuthState tests OAuth state generation with random inputs
// Run: go test -fuzz=FuzzGenerateOAuthState -fuzztime=30s ./internal/adapter/handler/...
func FuzzGenerateOAuthState(f *testing.F) {
	f.Add("my-tenant", "/dashboard")
	f.Add("", "")
	f.Add("tenant", "/")
	f.Add("tenant-with-dashes", "/path/with/slashes")
	f.Add(strings.Repeat("a", 1000), strings.Repeat("b", 1000))
	f.Add("tenant\x00null", "/path\x00null")
	f.Add("tenant<script>", "/path?q=<script>")
	f.Add("租户", "/中文路径")

	f.Fuzz(func(t *testing.T, tenant, redirect string) {
		// Skip invalid UTF-8 inputs as JSON encoding will transform them
		if !utf8.ValidString(tenant) || !utf8.ValidString(redirect) {
			return
		}

		state, nonce, err := generateOAuthState(tenant, redirect)

		// Invariant 1: should never panic (implicit)

		// Invariant 2: if generation succeeds, state and nonce should be non-empty
		if err == nil {
			if state == "" {
				t.Error("generated state is empty")
			}
			if nonce == "" {
				t.Error("generated nonce is empty")
			}
		}

		// Invariant 3: generated state should be parseable
		if err == nil && state != "" {
			parsed, parseErr := parseOAuthState(state)
			if parseErr != nil {
				t.Errorf("generated state not parseable: %v", parseErr)
			}
			if parsed != nil {
				// Verify round-trip
				if parsed.TenantSlug != tenant {
					t.Errorf("tenant mismatch: got %q, want %q", parsed.TenantSlug, tenant)
				}
				if parsed.RedirectURL != redirect {
					t.Errorf("redirect mismatch: got %q, want %q", parsed.RedirectURL, redirect)
				}
				if parsed.Nonce != nonce {
					t.Errorf("nonce mismatch: got %q, want %q", parsed.Nonce, nonce)
				}
			}
		}

		// Invariant 4: state should be valid base64
		if err == nil && state != "" {
			_, decodeErr := base64.URLEncoding.DecodeString(state)
			if decodeErr != nil {
				t.Errorf("state is not valid base64: %v", decodeErr)
			}
		}

		// Invariant 5: multiple calls should produce unique nonces
		if err == nil {
			state2, nonce2, _ := generateOAuthState(tenant, redirect)
			if nonce == nonce2 {
				t.Error("nonces should be unique across calls")
			}
			if state == state2 {
				t.Error("states should be unique due to unique nonces")
			}
		}
	})
}

// FuzzOAuthStateRoundTrip verifies encode/decode round-trip integrity
func FuzzOAuthStateRoundTrip(f *testing.F) {
	f.Add("tenant", "/path", "nonce123")
	f.Add("", "", "")
	f.Add("a", "/", "n")
	f.Add("multi-tenant-org", "/dashboard/settings?tab=security", "secure-nonce-abc123")

	f.Fuzz(func(t *testing.T, tenant, redirect, nonce string) {
		// Skip invalid UTF-8 inputs as JSON encoding will transform them
		if !utf8.ValidString(tenant) || !utf8.ValidString(redirect) || !utf8.ValidString(nonce) {
			return
		}

		original := OAuthStateData{
			TenantSlug:  tenant,
			RedirectURL: redirect,
			Nonce:       nonce,
			CreatedAt:   time.Now(),
		}

		// Encode
		jsonData, err := json.Marshal(original)
		if err != nil {
			return // Skip invalid inputs
		}
		encoded := base64.URLEncoding.EncodeToString(jsonData)

		// Decode
		parsed, err := parseOAuthState(encoded)
		if err != nil {
			t.Errorf("failed to parse valid state: %v", err)
			return
		}

		// Verify
		if parsed.TenantSlug != tenant {
			t.Errorf("tenant mismatch: got %q, want %q", parsed.TenantSlug, tenant)
		}
		if parsed.RedirectURL != redirect {
			t.Errorf("redirect mismatch: got %q, want %q", parsed.RedirectURL, redirect)
		}
		if parsed.Nonce != nonce {
			t.Errorf("nonce mismatch: got %q, want %q", parsed.Nonce, nonce)
		}
	})
}
