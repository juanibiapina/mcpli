package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/mcp"
	"github.com/juanibiapina/mcpli/internal/terminal"
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
		Run: func(cmd *cobra.Command, args []string) {
			// When called without subcommand, show the tool list
			printServerHelp(name, server)
		},
	}

	// Add tool subcommands
	for _, tool := range server.Tools {
		cmd.AddCommand(createToolCommand(server, tool))
	}

	// Set custom help template for better tool listing
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		printServerHelp(name, server)
	})

	return cmd
}

// printServerHelp prints formatted help for a server command
func printServerHelp(name string, server *config.Server) {
	termWidth := terminal.GetWidth()
	descIndent := "      " // 6 spaces for description indent

	fmt.Printf("Server: %s\n", server.ServerInfo.Name)
	fmt.Printf("URL: %s\n", server.URL)
	fmt.Println()
	fmt.Println("Tools:")

	for _, tool := range server.Tools {
		fmt.Printf("  %s\n", tool.Name)
		if tool.Description != "" {
			wrapped := terminal.WrapText(tool.Description, termWidth-len(descIndent), descIndent)
			fmt.Printf("%s%s\n", descIndent, wrapped)
		}
		fmt.Println()
	}

	fmt.Printf("Use \"mcpli %s <tool> --help\" for more information about a tool.\n", name)
}

// createToolCommand creates a command for a specific tool
func createToolCommand(server *config.Server, tool config.Tool) *cobra.Command {
	cmd := &cobra.Command{
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

	// Set explicit help function to avoid inheriting parent's custom help
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		// Print Long description with word wrapping, then usage
		if c.Long != "" {
			termWidth := terminal.GetWidth()
			wrapped := terminal.WrapText(c.Long, termWidth, "")
			fmt.Println(wrapped)
			fmt.Println()
		}
		fmt.Print(c.UsageString())
	})

	return cmd
}

// truncateDescription shortens a description for display
func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
