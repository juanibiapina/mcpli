package mcp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
