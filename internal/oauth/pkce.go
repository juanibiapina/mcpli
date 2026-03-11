package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCEChallenge holds a PKCE code verifier and its S256 challenge.
type PKCEChallenge struct {
	CodeVerifier  string
	CodeChallenge string
}

// GeneratePKCE creates a random code_verifier (43–128 chars of unreserved characters)
// and its corresponding S256 code_challenge.
func GeneratePKCE() (*PKCEChallenge, error) {
	// Generate 32 random bytes → 43 base64url chars (no padding)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	verifier := base64.RawURLEncoding.EncodeToString(b)

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}, nil
}
