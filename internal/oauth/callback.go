package oauth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
)

const (
	CallbackPort = 19877
	CallbackPath = "/oauth/callback"
)

// RedirectURI returns the fixed redirect URI for the OAuth callback.
func RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", CallbackPort, CallbackPath)
}

// callbackResult holds the authorization code or error from the callback.
type callbackResult struct {
	Code  string
	State string
	Err   error
}

// CallbackServer is a temporary HTTP server that receives the OAuth callback.
type CallbackServer struct {
	server   *http.Server
	resultCh chan callbackResult
}

// StartCallbackServer starts the callback server and begins listening.
// The server must be started before opening the browser to avoid a race condition
// where the OAuth redirect arrives before the server is ready.
func StartCallbackServer() (*CallbackServer, error) {
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			desc := r.URL.Query().Get("error_description")
			resultCh <- callbackResult{Err: fmt.Errorf("authorization error: %s: %s", errParam, desc)}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Authentication Failed</h2><p>%s: %s</p><p>You can close this window.</p></body></html>", html.EscapeString(errParam), html.EscapeString(desc))
			return
		}

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" {
			resultCh <- callbackResult{Err: fmt.Errorf("callback missing authorization code")}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Authentication Failed</h2><p>Missing authorization code.</p><p>You can close this window.</p></body></html>")
			return
		}

		resultCh <- callbackResult{Code: code, State: state}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Authentication Successful</h2><p>You can close this window and return to the terminal.</p></body></html>")
	})

	addr := fmt.Sprintf("127.0.0.1:%d", CallbackPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on %s: %w", addr, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	return &CallbackServer{server: server, resultCh: resultCh}, nil
}

// Wait blocks until the callback is received or the context is cancelled.
func (cs *CallbackServer) Wait(ctx context.Context) (code string, state string, err error) {
	select {
	case result := <-cs.resultCh:
		_ = cs.server.Shutdown(context.Background())
		if result.Err != nil {
			return "", "", result.Err
		}
		return result.Code, result.State, nil
	case <-ctx.Done():
		_ = cs.server.Shutdown(context.Background())
		return "", "", ctx.Err()
	}
}
