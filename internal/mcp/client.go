package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/juanibiapina/mcpli/internal/version"
)

const (
	ProtocolVersion = "2024-11-05"
)

// Client is an MCP HTTP/SSE client
type Client struct {
	URL     string
	Headers map[string]string
	client  *http.Client
}

// NewClient creates a new MCP client
func NewClient(url string, headers map[string]string) *Client {
	return &Client{
		URL:     url,
		Headers: headers,
		client: &http.Client{
			// Custom redirect policy to preserve POST method and body
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				// Preserve original method and body
				if len(via) > 0 {
					req.Method = via[0].Method
					if via[0].GetBody != nil {
						body, err := via[0].GetBody()
						if err == nil {
							req.Body = body
						}
					}
					// Copy headers
					for key, values := range via[0].Header {
						req.Header[key] = values
					}
				}
				return nil
			},
		},
	}
}

// jsonRPCRequest represents a JSON-RPC 2.0 request
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ServerInfo contains server metadata from initialize response
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the result of an initialize call
type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ListToolsResult is the result of a tools/list call
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// doRequest sends a JSON-RPC request and parses the SSE response
func (c *Client) doRequest(method string, params interface{}, id int) (*jsonRPCResponse, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody so redirects can re-read the body
	httpReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	for k, v := range c.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE response
	return parseSSEResponse(resp.Body)
}

// parseSSEResponse extracts JSON-RPC response from SSE format
func parseSSEResponse(r io.Reader) (*jsonRPCResponse, error) {
	scanner := bufio.NewScanner(r)
	
	// Increase buffer size for large responses
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			dataLine := strings.TrimPrefix(line, "data: ")
			
			// Try to parse as JSON-RPC response
			var resp jsonRPCResponse
			if err := json.Unmarshal([]byte(dataLine), &resp); err != nil {
				// Not valid JSON, continue reading
				continue
			}
			
			// Got a valid response, return immediately
			return &resp, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return nil, fmt.Errorf("no valid JSON-RPC response in SSE stream")
}

// Initialize performs the MCP initialize handshake
func (c *Client) Initialize() (*InitializeResult, error) {
	params := map[string]interface{}{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "mcpli",
			"version": version.Version,
		},
	}

	resp, err := c.doRequest("initialize", params, 1)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("server error: %s", resp.Error.Message)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse initialize result: %w", err)
	}

	return &result, nil
}

// ListTools retrieves the list of available tools
func (c *Client) ListTools() (*ListToolsResult, error) {
	resp, err := c.doRequest("tools/list", map[string]interface{}{}, 2)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("server error: %s", resp.Error.Message)
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	return &result, nil
}

// CallTool invokes a tool and returns the raw JSON result
func (c *Client) CallTool(name string, arguments json.RawMessage) (json.RawMessage, error) {
	params := map[string]interface{}{
		"name": name,
	}

	if arguments != nil && len(arguments) > 0 {
		var args interface{}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments JSON: %w", err)
		}
		params["arguments"] = args
	}

	resp, err := c.doRequest("tools/call", params, 3)
	if err != nil {
		return nil, err
	}

	// Return raw result (including errors) as per user requirement
	return resp.Result, nil
}
