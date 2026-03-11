package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscover_ValidMetadata(t *testing.T) {
	meta := ServerMetadata{
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
		RegistrationEndpoint:  "https://auth.example.com/register",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/oauth-authorization-server" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(meta)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := Discover(server.URL)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if result.AuthorizationEndpoint != meta.AuthorizationEndpoint {
		t.Errorf("AuthorizationEndpoint = %q, want %q", result.AuthorizationEndpoint, meta.AuthorizationEndpoint)
	}
	if result.TokenEndpoint != meta.TokenEndpoint {
		t.Errorf("TokenEndpoint = %q, want %q", result.TokenEndpoint, meta.TokenEndpoint)
	}
	if result.RegistrationEndpoint != meta.RegistrationEndpoint {
		t.Errorf("RegistrationEndpoint = %q, want %q", result.RegistrationEndpoint, meta.RegistrationEndpoint)
	}
}

func TestDiscover_PathAware(t *testing.T) {
	meta := ServerMetadata{
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/oauth-authorization-server/mcp/default" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(meta)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := Discover(server.URL + "/mcp/default")
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if result.AuthorizationEndpoint != meta.AuthorizationEndpoint {
		t.Errorf("AuthorizationEndpoint = %q, want %q", result.AuthorizationEndpoint, meta.AuthorizationEndpoint)
	}
}

func TestDiscover_MissingEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example.com/authorize",
			// missing token_endpoint
		})
	}))
	defer server.Close()

	_, err := Discover(server.URL)
	if err == nil {
		t.Fatal("Discover() should fail when token_endpoint is missing")
	}
	if !strings.Contains(err.Error(), "token_endpoint") {
		t.Errorf("error should mention token_endpoint, got: %v", err)
	}
}

func TestDiscover_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Discover(server.URL)
	if err == nil {
		t.Fatal("Discover() should fail when metadata endpoint returns 404")
	}
}

func TestDiscover_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	_, err := Discover(server.URL)
	if err == nil {
		t.Fatal("Discover() should fail on non-JSON response")
	}
}
