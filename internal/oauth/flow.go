package oauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// tokenResponse is the OAuth token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Authenticate runs the full OAuth authorization code flow with PKCE.
// It performs discovery, client registration (if needed), opens the browser,
// waits for the callback, and exchanges the code for tokens.
func Authenticate(serverURL string) error {
	fmt.Println("OAuth authentication required. Starting authorization flow...")

	// 1. Discovery
	meta, err := Discover(serverURL)
	if err != nil {
		return fmt.Errorf("OAuth discovery failed: %w", err)
	}

	redirectURI := RedirectURI()

	// 2. Load store and check for existing client registration
	store, err := LoadStore()
	if err != nil {
		return fmt.Errorf("failed to load auth store: %w", err)
	}

	entry := store.Entries[serverURL]
	var clientID, clientSecret string

	if entry != nil && entry.ClientID != "" {
		clientID = entry.ClientID
		clientSecret = entry.ClientSecret
	} else {
		// 3. Dynamic client registration
		if meta.RegistrationEndpoint == "" {
			return fmt.Errorf("server does not support dynamic client registration and no client is registered")
		}

		fmt.Println("Registering client...")
		clientID, clientSecret, err = RegisterClient(meta.RegistrationEndpoint, redirectURI)
		if err != nil {
			return fmt.Errorf("client registration failed: %w", err)
		}
	}

	// 4. Generate PKCE challenge
	pkce, err := GeneratePKCE()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE challenge: %w", err)
	}

	// 5. Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// 6. Build authorization URL
	authURL, err := url.Parse(meta.AuthorizationEndpoint)
	if err != nil {
		return fmt.Errorf("invalid authorization endpoint: %w", err)
	}

	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("code_challenge", pkce.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	authURL.RawQuery = q.Encode()

	// 7. Start callback server BEFORE opening browser to avoid race condition
	callbackServer, err := StartCallbackServer()
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Opening browser for authentication...")
	OpenBrowser(authURL.String())

	code, returnedState, err := callbackServer.Wait(ctx)
	if err != nil {
		return fmt.Errorf("authorization callback failed: %w", err)
	}

	// 8. Verify state
	if returnedState != state {
		return fmt.Errorf("state mismatch: possible CSRF attack")
	}

	// 9. Exchange code for tokens
	tokens, err := exchangeCode(meta.TokenEndpoint, clientID, clientSecret, code, redirectURI, pkce.CodeVerifier)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	// 10. Store credentials
	store.Entries[serverURL] = &AuthEntry{
		ClientID:            clientID,
		ClientSecret:        clientSecret,
		AccessToken:         tokens.AccessToken,
		RefreshToken:        tokens.RefreshToken,
		ExpiresAt:           time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		TokenType:           tokens.TokenType,
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save auth credentials: %w", err)
	}

	fmt.Println("Authentication successful.")
	return nil
}

// GetValidToken returns a valid access token for the given server URL.
// It refreshes the token if it's expired.
func GetValidToken(serverURL string) (string, error) {
	store, err := LoadStore()
	if err != nil {
		return "", fmt.Errorf("failed to load auth store: %w", err)
	}

	entry, ok := store.Entries[serverURL]
	if !ok {
		return "", fmt.Errorf("no OAuth credentials found for %s", serverURL)
	}

	if !entry.IsExpired() {
		return entry.AccessToken, nil
	}

	// Token is expired, try to refresh
	if entry.RefreshToken == "" {
		return "", fmt.Errorf("access token expired and no refresh token available")
	}

	// Discover token endpoint
	meta, err := Discover(serverURL)
	if err != nil {
		return "", fmt.Errorf("OAuth discovery failed during token refresh: %w", err)
	}

	tokens, err := refreshToken(meta.TokenEndpoint, entry.ClientID, entry.ClientSecret, entry.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("token refresh failed: %w", err)
	}

	// Update stored tokens
	entry.AccessToken = tokens.AccessToken
	entry.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	if tokens.RefreshToken != "" {
		entry.RefreshToken = tokens.RefreshToken
	}

	if err := store.Save(); err != nil {
		return "", fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return entry.AccessToken, nil
}

func exchangeCode(tokenEndpoint, clientID, clientSecret, code, redirectURI, codeVerifier string) (*tokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
		"client_id":     {clientID},
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	return doTokenRequest(tokenEndpoint, data)
}

func refreshToken(tokenEndpoint, clientID, clientSecret, refreshTok string) (*tokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshTok},
		"client_id":     {clientID},
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	return doTokenRequest(tokenEndpoint, data)
}

func doTokenRequest(tokenEndpoint string, data url.Values) (*tokenResponse, error) {
	resp, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &tokens, nil
}
