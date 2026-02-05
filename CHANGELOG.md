# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Improved tool listing output with multi-line, word-wrapped descriptions
- `mcpli <server>` and `mcpli list <server>` now show full tool descriptions
- `mcpli <server> <tool> --help` now shows word-wrapped tool descriptions
- Tool descriptions adapt to terminal width for better readability

## [1.0.0] - 2026-02-05

### Added

- Initial release
- `mcpli add <name> <url>` - Add and initialize MCP servers
- `mcpli update <name>` - Refresh server tools
- `mcpli remove <name>` - Remove configured servers
- `mcpli list` - List all configured servers
- `mcpli <server> <tool> [json-arguments]` - Invoke server tools
- Dynamic command generation from server tool schemas
- Header support with environment variable expansion
- Configuration stored in `~/.config/mcpli/config.json`
- Version management with `--version` flag
- GitHub Actions for CI and releases
- GoReleaser configuration with Homebrew tap support
