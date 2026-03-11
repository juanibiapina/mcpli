package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	// code_verifier must be 43-128 characters of unreserved characters
	if len(pkce.CodeVerifier) < 43 || len(pkce.CodeVerifier) > 128 {
		t.Errorf("CodeVerifier length = %d, want 43-128", len(pkce.CodeVerifier))
	}

	// code_challenge must be valid base64url-encoded SHA256 of code_verifier
	hash := sha256.Sum256([]byte(pkce.CodeVerifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])
	if pkce.CodeChallenge != expected {
		t.Errorf("CodeChallenge = %q, want SHA256 of verifier = %q", pkce.CodeChallenge, expected)
	}
}

func TestGeneratePKCE_Unique(t *testing.T) {
	pkce1, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	pkce2, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	if pkce1.CodeVerifier == pkce2.CodeVerifier {
		t.Error("two calls to GeneratePKCE produced identical verifiers")
	}
}
