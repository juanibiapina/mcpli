package cmd

import (
	"fmt"
	"time"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/mcp"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a server's tool definitions",
	Long: `Refresh the cached tool definitions for a configured server.

Use this when the server has added new tools or updated existing ones.

Example:
  mcpli update knuspr`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Find server
	server, exists := cfg.Servers[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}

	// Create client with expanded headers
	client := mcp.NewClient(server.URL, server.ExpandHeaders())

	// Initialize connection
	fmt.Printf("Connecting to %s...\n", server.URL)
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

	// Update server config
	server.ProtocolVersion = initResult.ProtocolVersion
	server.ServerInfo = config.ServerInfo{
		Name:    initResult.ServerInfo.Name,
		Version: initResult.ServerInfo.Version,
	}
	server.Tools = tools
	server.UpdatedAt = time.Now()

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Server %q updated successfully\n", name)
	return nil
}
