package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ServerMetadata holds the OAuth authorization server metadata
// from the well-known discovery endpoint.
type ServerMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
}

// Discover fetches OAuth authorization server metadata for the given server URL.
// It tries {origin}/.well-known/oauth-authorization-server first.
func Discover(serverURL string) (*ServerMetadata, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	origin := parsed.Scheme + "://" + parsed.Host

	// Try path-aware well-known URL first (RFC 9728 style)
	// e.g. https://host/.well-known/oauth-authorization-server/path
	path := strings.TrimRight(parsed.Path, "/")
	if path != "" {
		wellKnownURL := origin + "/.well-known/oauth-authorization-server" + path
		meta, err := fetchMetadata(wellKnownURL)
		if err == nil {
			return meta, nil
		}
	}

	// Fall back to origin-level well-known URL
	wellKnownURL := origin + "/.well-known/oauth-authorization-server"
	return fetchMetadata(wellKnownURL)
}

func fetchMetadata(url string) (*ServerMetadata, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata endpoint %s returned status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var meta ServerMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	if meta.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("metadata missing authorization_endpoint")
	}
	if meta.TokenEndpoint == "" {
		return nil, fmt.Errorf("metadata missing token_endpoint")
	}

	return &meta, nil
}
