package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/mcp"
	"github.com/juanibiapina/mcpli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mcpli",
	Short: "MCP CLI - invoke MCP server tools from the command line",
	Long: `mcpli is a command line interface for interacting with MCP (Model Context Protocol) servers.

Add servers with 'mcpli add', then invoke their tools directly:
  mcpli <server> <tool> [json-arguments]

Examples:
  mcpli add knuspr https://mcp.knuspr.de/mcp/ --header "rhl-email: \${ROHLIK_USERNAME}"
  mcpli knuspr search_products '{"query": "milk"}'
  mcpli knuspr get_cart`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set version for --version flag
	rootCmd.Version = version.Version

	// Add built-in commands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(listCmd)

	// Load config and add server commands dynamically
	cfg, err := config.Load()
	if err != nil {
		// Don't fail if config can't be loaded, just skip dynamic commands
		return
	}

	for name, server := range cfg.Servers {
		rootCmd.AddCommand(createServerCommand(name, server))
	}
}

// createServerCommand creates a command for a configured server
func createServerCommand(name string, server *config.Server) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Invoke tools on the %s server", name),
		Long:  fmt.Sprintf("Server: %s\nURL: %s", server.ServerInfo.Name, server.URL),
	}

	// Add tool subcommands
	for _, tool := range server.Tools {
		cmd.AddCommand(createToolCommand(server, tool))
	}

	return cmd
}

// createToolCommand creates a command for a specific tool
func createToolCommand(server *config.Server, tool config.Tool) *cobra.Command {
	return &cobra.Command{
		Use:   tool.Name + " [json-arguments]",
		Short: truncateDescription(tool.Description, 60),
		Long:  tool.Description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse arguments
			var arguments json.RawMessage
			if len(args) > 0 {
				arguments = json.RawMessage(args[0])
				// Validate it's valid JSON
				var test interface{}
				if err := json.Unmarshal(arguments, &test); err != nil {
					return fmt.Errorf("invalid JSON arguments: %w", err)
				}
			}

			// Create client with expanded headers
			client := mcp.NewClient(server.URL, server.ExpandHeaders())

			// Call the tool
			result, err := client.CallTool(tool.Name, arguments)
			if err != nil {
				return err
			}

			// Output raw JSON
			fmt.Println(string(result))
			return nil
		},
	}
}

// truncateDescription shortens a description for display
func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
