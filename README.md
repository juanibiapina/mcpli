# mcpli

Turn MCP servers into native CLI applications with shell completion.

## Features

- 🔍 **Discoverable** - Tab completion for servers AND tools in your shell
- 📖 **Self-documenting** - `--help` shows full tool descriptions at every level
- ⚡ **Instant** - Tools cached locally, no server roundtrip for discovery
- 🔧 **Familiar** - Works like any CLI you already use (git, kubectl, etc.)

## Quick Start

```bash
# 1. Add a server (fetches and caches all tools)
mcpli add myserver https://example.com/mcp/

# 2. Explore available tools
mcpli myserver --help

# 3. Invoke a tool
mcpli myserver search '{"query": "hello"}'
```

## Installation

```bash
go install github.com/juanibiapina/mcpli/cmd/mcpli@latest
```

Or build from source:

```bash
git clone https://github.com/juanibiapina/mcpli
cd mcpli
go build -o mcpli ./cmd/mcpli
```

## Discovering Tools

Every server and tool is a native subcommand with built-in help:

```bash
# See all configured servers
mcpli --help

# See all tools on a server
mcpli myserver --help

# See a tool's full description and usage
mcpli myserver search_products --help
```

Example output of `mcpli myserver --help`:

```
Usage:
  mcpli myserver [command]

Available Commands:
  get_cart              View everything currently in the shopping cart...
  search_products       Search products by keyword, filters, or recomm...
  add_items_to_cart     Put products into the cart for purchase...

Use "mcpli myserver [command] --help" for more information about a command.
```

## Shell Completion

Enable tab completion for servers and tools:

```bash
# Bash
echo 'source <(mcpli completion bash)' >> ~/.bashrc

# Zsh
echo 'source <(mcpli completion zsh)' >> ~/.zshrc

# Fish
mcpli completion fish | source
```

After setup, tab completion works for everything:

```bash
mcpli <TAB>              # Complete server names
mcpli myserver <TAB>     # Complete tool names
```

## Commands

### Add a server

```bash
mcpli add <name> <url> [--header "key: value"]...
```

Headers can include environment variable references using `${VAR_NAME}` syntax:

```bash
mcpli add knuspr https://mcp.knuspr.de/mcp/ \
  --header 'auth-email: ${MY_EMAIL}' \
  --header 'auth-pass: ${MY_PASSWORD}'
```

This connects to the server, fetches all available tools, and caches them locally.

### List servers

```bash
mcpli list
```

### List tools for a server

```bash
mcpli list <server>
```

### Invoke a tool

```bash
mcpli <server> <tool> [json-arguments]
```

Examples:

```bash
# Tool with no arguments
mcpli myserver get_cart

# Tool with arguments
mcpli myserver search_products '{"keyword": "milk"}'
```

### Update a server

Refresh the cached tool definitions:

```bash
mcpli update <server>
```

### Remove a server

```bash
mcpli remove <server>
```

## Configuration

Configuration is stored in `~/.config/mcpli/config.json` (following XDG conventions).

The config file contains server URLs, headers (with unexpanded env var references), and cached tool definitions.

## License

MIT
