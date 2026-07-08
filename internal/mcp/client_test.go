package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// rpcMessage is a minimal view of a JSON-RPC message for test assertions.
type rpcMessage struct {
	Method string           `json:"method"`
	HasID  bool             `json:"-"`
	ID     *json.RawMessage `json:"id"`
}

func decodeRPC(t *testing.T, r *http.Request) rpcMessage {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("failed to parse request body %q: %v", string(body), err)
	}
	msg.HasID = msg.ID != nil
	return msg
}

func TestInitialize_SendsInitializedNotificationAndReplaysSessionID(t *testing.T) {
	var sawInitialized bool
	var listSession string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := decodeRPC(t, r)
		switch msg.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "abc")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"s","version":"1"}}}`))
		case "notifications/initialized":
			if msg.HasID {
				t.Errorf("notification must not carry an id")
			}
			if got := r.Header.Get("Mcp-Session-Id"); got != "abc" {
				t.Errorf("notification Mcp-Session-Id = %q, want %q", got, "abc")
			}
			sawInitialized = true
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			listSession = r.Header.Get("Mcp-Session-Id")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"t","description":"d"}]}}`))
		default:
			t.Errorf("unexpected method %q", msg.Method)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)

	if _, err := client.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if !sawInitialized {
		t.Fatal("server never received notifications/initialized")
	}

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if listSession != "abc" {
		t.Errorf("tools/list Mcp-Session-Id = %q, want %q", listSession, "abc")
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "t" {
		t.Errorf("unexpected tools: %+v", result.Tools)
	}
}

// strictServer models a server (like incident.io) that rejects method calls
// until it has seen notifications/initialized for the session.
func strictServer(t *testing.T) *httptest.Server {
	t.Helper()
	initialized := map[string]bool{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := decodeRPC(t, r)
		session := r.Header.Get("Mcp-Session-Id")
		switch msg.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "sess-1")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"s","version":"1"}}}`))
		case "notifications/initialized":
			initialized[session] = true
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			w.Header().Set("Content-Type", "application/json")
			if !initialized[session] {
				w.Write([]byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"method \"tools/list\" is invalid during session initialization"}}`))
				return
			}
			w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"t","description":"d"}]}}`))
		default:
			t.Errorf("unexpected method %q", msg.Method)
		}
	}))
}

func TestStrictServer_HandshakeUnblocksListTools(t *testing.T) {
	server := strictServer(t)
	defer server.Close()

	client := NewClient(server.URL, nil)
	if _, err := client.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Errorf("expected 1 tool, got %+v", result.Tools)
	}
}

func TestStrictServer_WithoutNotificationRejects(t *testing.T) {
	server := strictServer(t)
	defer server.Close()

	// Bypass Initialize() (and thus the notification) by only capturing the
	// session via a raw initialize call, then listing tools directly.
	client := NewClient(server.URL, nil)
	if _, err := client.doRequest("initialize", map[string]interface{}{}, 1); err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}

	_, err := client.ListTools()
	if err == nil {
		t.Fatal("expected error when notification was skipped, got nil")
	}
}

func TestDoNotify_AcceptsStatuses(t *testing.T) {
	cases := []struct {
		status           int
		wantErr          bool
		wantUnauthorized bool
	}{
		{http.StatusOK, false, false},
		{http.StatusAccepted, false, false},
		{http.StatusUnauthorized, true, true},
		{http.StatusInternalServerError, true, false},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		client := NewClient(server.URL, nil)
		err := client.doNotify("notifications/initialized", nil)
		server.Close()

		if tc.wantErr && err == nil {
			t.Errorf("status %d: expected error, got nil", tc.status)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("status %d: unexpected error: %v", tc.status, err)
		}
		if tc.wantUnauthorized {
			var unauthorizedErr *UnauthorizedError
			if !errors.As(err, &unauthorizedErr) {
				t.Errorf("status %d: expected UnauthorizedError, got %T", tc.status, err)
			}
		}
	}
}

func TestStatelessServer_NoSessionHeader(t *testing.T) {
	var sawSessionHeader bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Mcp-Session-Id") != "" {
			sawSessionHeader = true
		}
		msg := decodeRPC(t, r)
		switch msg.Method {
		case "initialize":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"s","version":"1"}}}`))
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}`))
		default:
			t.Errorf("unexpected method %q", msg.Method)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	if _, err := client.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if _, err := client.ListTools(); err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if sawSessionHeader {
		t.Error("client sent Mcp-Session-Id header though server issued none")
	}
}

func TestDoRequest_UnauthorizedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("authentication required"))
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	_, err := client.Initialize()

	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}

	var unauthorizedErr *UnauthorizedError
	if !errors.As(err, &unauthorizedErr) {
		t.Fatalf("expected UnauthorizedError, got %T: %v", err, err)
	}

	if unauthorizedErr.Body != "authentication required" {
		t.Errorf("Body = %q, want %q", unauthorizedErr.Body, "authentication required")
	}
}

func TestDoRequest_OtherError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	_, err := client.Initialize()

	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}

	var unauthorizedErr *UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		t.Fatal("500 should not produce UnauthorizedError")
	}
}
