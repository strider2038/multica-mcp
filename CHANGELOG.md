# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with major version `0` while the project is in active development.

## [Unreleased]

## [0.2.0] - 2026-06-29

### Added

- Multica REST API **v0.3.31** compatibility.
- MCP tool `multica_preview_issue_triggers` — preview agent runs that would start from issue create/update.
- MCP tool `multica_preview_comment_triggers` — preview which agents a comment would wake before posting.
- `multica_add_comment` supports `parent_id`, `suppress_agent_ids`, and documents the `/note` prefix for human-only comments.
- `multica_create_task`, `multica_create_subtask`, and `multica_update_task` support sub-issue `stage` barrier grouping.
- `multica_update_task` supports `suppress_run`, `handoff_note`, and `clear_stage`.
- `go install github.com/strider2038/multica-mcp@latest` — entry point moved to repository root.
- `CHANGELOG.md`, `AGENTS.md`, and automated GitHub releases on push to `main`.

### Changed

- Comment model includes thread resolution metadata (`resolved_at`, `reply_count`, `last_activity_at`, `source_task_id`).
- Task model includes `stage` for ordered sub-issue barrier groups.
- Multica REST API target bumped from **v0.3.16** to **v0.3.31**.
- Binary name: `multica-mcp` (was `multica-mcp-server`).
- MCP server reports build version from `VERSION` / link-time `-ldflags`.

## [0.1.0] - 2026-06-05

### Added

- Initial MCP server with stdio/HTTP transports and 13 Multica tools.
- Workspace auto-detection and slug/ID scoping via `X-Workspace-*` headers.

[Unreleased]: https://github.com/strider2038/multica-mcp/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/strider2038/multica-mcp/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/strider2038/multica-mcp/releases/tag/v0.1.0
