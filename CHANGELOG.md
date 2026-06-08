# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with major version `0` while the project is in active development.

## [Unreleased]

### Changed

- Multica REST API target bumped to **v0.3.17** (no REST contract changes for MCP tools since v0.3.16).

### Added

- Multica REST API **v0.3.16** compatibility (`assignee_type`, client identity headers, structured API errors).
- `go install github.com/strider2038/multica-mcp@latest` — entry point moved to repository root.
- `CHANGELOG.md`, `AGENTS.md`, and automated GitHub releases on push to `main`.

### Changed

- Binary name: `multica-mcp` (was `multica-mcp-server`).
- MCP server reports build version from `VERSION` / link-time `-ldflags`.

## [0.1.0] - 2026-06-05

### Added

- Initial MCP server with stdio/HTTP transports and 13 Multica tools.
- Workspace auto-detection and slug/ID scoping via `X-Workspace-*` headers.

[Unreleased]: https://github.com/strider2038/multica-mcp/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/strider2038/multica-mcp/releases/tag/v0.1.0
