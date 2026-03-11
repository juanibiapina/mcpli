package cmd

import (
	"fmt"

	"github.com/juanibiapina/mcpli/internal/config"
	"github.com/juanibiapina/mcpli/internal/oauth"
)

// resolveHeaders returns the headers for a server, including an OAuth token if applicable.
func resolveHeaders(serverName string, server *config.Server) (map[string]string, error) {
	headers := server.ExpandHeaders()
	if server.OAuth {
		token, err := oauth.GetValidToken(server.URL)
		if err != nil {
			return nil, fmt.Errorf("OAuth failed: %w\nRun 'mcpli update %s' to re-authenticate", err, serverName)
		}
		headers["Authorization"] = "Bearer " + token
	}
	return headers, nil
}
