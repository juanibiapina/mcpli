package oauth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// AuthEntry holds OAuth credentials for a single server.
type AuthEntry struct {
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret,omitempty"`
	AccessToken         string `json:"access_token"`
	RefreshToken        string `json:"refresh_token,omitempty"`
	ExpiresAt           time.Time `json:"expires_at"`
	TokenType           string `json:"token_type"`
}

// IsExpired returns true if the access token has expired (with a 30-second buffer).
func (e *AuthEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt.Add(-30 * time.Second))
}

// AuthStore holds OAuth state for all servers.
type AuthStore struct {
	Entries map[string]*AuthEntry `json:"entries"`
}

// storePath returns the path to the auth store file.
func storePath() string {
	return filepath.Join(xdg.StateHome, "mcpli", "auth.json")
}

// LoadStore reads the auth store from disk.
// Returns an empty store if the file doesn't exist.
func LoadStore() (*AuthStore, error) {
	path := storePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthStore{Entries: make(map[string]*AuthEntry)}, nil
		}
		return nil, err
	}

	var store AuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	if store.Entries == nil {
		store.Entries = make(map[string]*AuthEntry)
	}

	return &store, nil
}

// Save writes the auth store to disk with 0600 permissions.
func (s *AuthStore) Save() error {
	path := storePath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Delete removes the auth entry for the given server URL.
func (s *AuthStore) Delete(serverURL string) {
	delete(s.Entries, serverURL)
}
