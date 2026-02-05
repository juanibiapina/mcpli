# mcpli

A command-line interface for interacting with MCP (Model Context Protocol) servers.

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

## Usage

### Add a server

```bash
mcpli add <name> <url> [--header "key: value"]...
```

Headers can include environment variable references using `${VAR_NAME}` syntax:

```bash
mcpli add knuspr https://mcp.knuspr.de/mcp/ \
  --header 'rhl-email: ${ROHLIK_USERNAME}' \
  --header 'rhl-pass: ${ROHLIK_PASSWORD}'
```

This connects to the server, fetches all available tools, and caches them locally for instant invocations.

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
mcpli knuspr get_cart

# Tool with arguments
mcpli knuspr search_products '{"keyword": "milk"}'
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
