package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req registrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.ClientName != "mcpli" {
			t.Errorf("client_name = %q, want %q", req.ClientName, "mcpli")
		}
		if req.TokenEndpointAuthMethod != "none" {
			t.Errorf("token_endpoint_auth_method = %q, want %q", req.TokenEndpointAuthMethod, "none")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"client_id":     "test-client-id",
			"client_secret": "test-client-secret",
		})
	}))
	defer server.Close()

	clientID, clientSecret, err := RegisterClient(server.URL, "http://127.0.0.1:19877/oauth/callback")
	if err != nil {
		t.Fatalf("RegisterClient() error: %v", err)
	}
	if clientID != "test-client-id" {
		t.Errorf("clientID = %q, want %q", clientID, "test-client-id")
	}
	if clientSecret != "test-client-secret" {
		t.Errorf("clientSecret = %q, want %q", clientSecret, "test-client-secret")
	}
}

func TestRegisterClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_request"}`))
	}))
	defer server.Close()

	_, _, err := RegisterClient(server.URL, "http://127.0.0.1:19877/oauth/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegisterClient_MissingClientID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"client_secret": "some-secret",
		})
	}))
	defer server.Close()

	_, _, err := RegisterClient(server.URL, "http://127.0.0.1:19877/oauth/callback")
	if err == nil {
		t.Fatal("expected error for missing client_id, got nil")
	}
}
