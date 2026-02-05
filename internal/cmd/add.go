package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/mcp"
	"github.com/spf13/cobra"
)

var addHeaders []string

var addCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a new MCP server",
	Long: `Add a new MCP server and fetch its available tools.

Headers can include environment variable references using ${VAR_NAME} syntax.
These will be expanded at runtime when invoking tools.

Examples:
  mcpli add knuspr https://mcp.knuspr.de/mcp/ \
    --header "rhl-email: \${ROHLIK_USERNAME}" \
    --header "rhl-pass: \${ROHLIK_PASSWORD}"`,
	Args: cobra.ExactArgs(2),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringArrayVarP(&addHeaders, "header", "H", nil, "HTTP header in 'key: value' format (can be repeated)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	url := args[1]

	// Parse headers
	headers := make(map[string]string)
	for _, h := range addHeaders {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header format: %q (expected 'key: value')", h)
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if server already exists
	if _, exists := cfg.Servers[name]; exists {
		return fmt.Errorf("server %q already exists (use 'mcpli update %s' to refresh)", name, name)
	}

	// Create client with expanded headers for the initial connection
	expandedHeaders := make(map[string]string)
	for k, v := range headers {
		expandedHeaders[k] = config.ExpandEnv(v)
	}
	client := mcp.NewClient(url, expandedHeaders)

	// Initialize connection
	fmt.Printf("Connecting to %s...\n", url)
	initResult, err := client.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Printf("Connected to %s v%s\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)

	// Fetch tools
	fmt.Println("Fetching tools...")
	toolsResult, err := client.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	fmt.Printf("Found %d tools\n", len(toolsResult.Tools))

	// Convert tools to config format
	tools := make([]config.Tool, len(toolsResult.Tools))
	for i, t := range toolsResult.Tools {
		tools[i] = config.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}

	// Save server config (with unexpanded headers)
	cfg.Servers[name] = &config.Server{
		URL:             url,
		Headers:         headers,
		ProtocolVersion: initResult.ProtocolVersion,
		ServerInfo: config.ServerInfo{
			Name:    initResult.ServerInfo.Name,
			Version: initResult.ServerInfo.Version,
		},
		Tools:     tools,
		UpdatedAt: time.Now(),
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Server %q added successfully\n", name)
	return nil
}
