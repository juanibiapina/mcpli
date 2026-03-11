package cmd

import (
	"fmt"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/oauth"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a configured server",
	Long: `Remove a server from the configuration.

Example:
  mcpli remove knuspr`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if server exists
	server, exists := cfg.Servers[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}

	// Clean up OAuth credentials if applicable
	if server.OAuth {
		store, err := oauth.LoadStore()
		if err == nil {
			store.Delete(server.URL)
			_ = store.Save()
		}
	}

	// Remove server
	delete(cfg.Servers, name)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Server %q removed\n", name)
	return nil
}
