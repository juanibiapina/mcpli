package oauth

import (
	"os"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

func setTestStateHome(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	original := xdg.StateHome
	xdg.StateHome = tmpDir
	t.Cleanup(func() { xdg.StateHome = original })
}

func TestAuthStore_WriteReadCycle(t *testing.T) {
	setTestStateHome(t)

	store := &AuthStore{
		Entries: map[string]*AuthEntry{
			"https://example.com/mcp": {
				ClientID:            "test-client-id",
				ClientSecret:        "test-secret",
				AccessToken:         "test-token",
				RefreshToken:        "test-refresh",
				ExpiresAt:           time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
				TokenType:           "Bearer",
			},
		},
	}

	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Check file permissions
	path := storePath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}

	// Read back
	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}

	entry, ok := loaded.Entries["https://example.com/mcp"]
	if !ok {
		t.Fatal("entry not found after reload")
	}
	if entry.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", entry.ClientID, "test-client-id")
	}
	if entry.AccessToken != "test-token" {
		t.Errorf("AccessToken = %q, want %q", entry.AccessToken, "test-token")
	}
}

func TestAuthStore_LoadEmpty(t *testing.T) {
	setTestStateHome(t)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(store.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(store.Entries))
	}
}

func TestAuthStore_Delete(t *testing.T) {
	setTestStateHome(t)

	store := &AuthStore{
		Entries: map[string]*AuthEntry{
			"https://a.com/mcp": {ClientID: "a"},
			"https://b.com/mcp": {ClientID: "b"},
		},
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	store.Delete("https://a.com/mcp")
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if _, ok := loaded.Entries["https://a.com/mcp"]; ok {
		t.Error("entry should have been deleted")
	}
	if _, ok := loaded.Entries["https://b.com/mcp"]; !ok {
		t.Error("other entry should still exist")
	}
}

func TestAuthEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"future", time.Now().Add(5 * time.Minute), false},
		{"past", time.Now().Add(-5 * time.Minute), true},
		{"within buffer", time.Now().Add(10 * time.Second), true}, // 30s buffer
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &AuthEntry{ExpiresAt: tt.expiresAt}
			if got := entry.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
