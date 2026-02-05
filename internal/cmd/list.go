package cmd

import (
	"fmt"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/terminal"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [server]",
	Short: "List servers or tools",
	Long: `List configured servers, or list tools for a specific server.

Examples:
  mcpli list           # List all servers
  mcpli list knuspr    # List tools for knuspr server`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(args) == 0 {
		// List servers
		if len(cfg.Servers) == 0 {
			fmt.Println("No servers configured. Use 'mcpli add' to add one.")
			return nil
		}

		for name, server := range cfg.Servers {
			fmt.Printf("%s - %s (%d tools)\n", name, server.URL, len(server.Tools))
		}
		return nil
	}

	// List tools for a specific server
	name := args[0]
	server, exists := cfg.Servers[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}

	if len(server.Tools) == 0 {
		fmt.Println("No tools available")
		return nil
	}

	fmt.Printf("Server: %s\n", server.ServerInfo.Name)
	fmt.Printf("URL: %s\n", server.URL)
	fmt.Println()
	fmt.Println("Tools:")

	termWidth := terminal.GetWidth()
	descIndent := "      " // 6 spaces for description indent

	for _, tool := range server.Tools {
		fmt.Printf("  %s\n", tool.Name)
		if tool.Description != "" {
			wrapped := terminal.WrapText(tool.Description, termWidth-len(descIndent), descIndent)
			fmt.Printf("%s%s\n", descIndent, wrapped)
		}
		fmt.Println()
	}

	fmt.Printf("Use \"mcpli %s <tool> --help\" for more information about a tool.\n", name)
	return nil
}
