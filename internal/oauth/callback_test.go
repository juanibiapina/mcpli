package oauth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCallbackServer_CodeReceived(t *testing.T) {
	cs, err := StartCallbackServer()
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}

	// Send the callback in a goroutine
	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s?code=test-code&state=test-state", CallbackPort, CallbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	code, state, err := cs.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait() error: %v", err)
	}
	if code != "test-code" {
		t.Errorf("code = %q, want %q", code, "test-code")
	}
	if state != "test-state" {
		t.Errorf("state = %q, want %q", state, "test-state")
	}
}

func TestCallbackServer_ErrorParam(t *testing.T) {
	cs, err := StartCallbackServer()
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s?error=access_denied&error_description=nope", CallbackPort, CallbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err = cs.Wait(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCallbackServer_MissingCode(t *testing.T) {
	cs, err := StartCallbackServer()
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s", CallbackPort, CallbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err = cs.Wait(ctx)
	if err == nil {
		t.Fatal("expected error for missing code, got nil")
	}
}
