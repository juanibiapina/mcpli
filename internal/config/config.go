package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/adrg/xdg"
)

// Tool represents an MCP tool definition
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ServerInfo contains MCP server metadata
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Server represents a configured MCP server
type Server struct {
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers,omitempty"`
	ProtocolVersion string            `json:"protocol_version"`
	ServerInfo      ServerInfo        `json:"server_info"`
	Tools           []Tool            `json:"tools"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Config represents the application configuration
type Config struct {
	Servers map[string]*Server `json:"servers"`
}

// configPath returns the path to the config file
func configPath() (string, error) {
	return xdg.ConfigFile("mcpli/config.json")
}

// Load reads the configuration from disk
// Returns an empty config if the file doesn't exist
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Servers: make(map[string]*Server)}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Servers == nil {
		cfg.Servers = make(map[string]*Server)
	}

	return &cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// envVarRegex matches ${VAR_NAME} patterns
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpandEnv expands ${VAR} references in a string using environment variables
func ExpandEnv(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := match[2 : len(match)-1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		// Return original if env var not set
		return match
	})
}

// ExpandHeaders returns a copy of headers with env vars expanded
func (s *Server) ExpandHeaders() map[string]string {
	expanded := make(map[string]string, len(s.Headers))
	for k, v := range s.Headers {
		expanded[k] = ExpandEnv(v)
	}
	return expanded
}
