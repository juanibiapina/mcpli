package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// registrationRequest is the dynamic client registration request body.
type registrationRequest struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	GrantTypes   []string `json:"grant_types"`
	ResponseTypes []string `json:"response_types"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
}

// registrationResponse is the dynamic client registration response.
type registrationResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// RegisterClient performs OAuth 2.0 Dynamic Client Registration.
// Returns the client_id and optionally client_secret.
func RegisterClient(registrationEndpoint string, redirectURI string) (clientID, clientSecret string, err error) {
	reqBody := registrationRequest{
		ClientName:   "mcpli",
		RedirectURIs: []string{redirectURI},
		GrantTypes:   []string{"authorization_code", "refresh_token"},
		ResponseTypes: []string{"code"},
		TokenEndpointAuthMethod: "none",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal registration request: %w", err)
	}

	resp, err := http.Post(registrationEndpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read registration response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("registration failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var regResp registrationResponse
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return "", "", fmt.Errorf("failed to parse registration response: %w", err)
	}

	if regResp.ClientID == "" {
		return "", "", fmt.Errorf("registration response missing client_id")
	}

	return regResp.ClientID, regResp.ClientSecret, nil
}
