# Agent development guide

Instructions for AI agents and contributors working on **multica-mcp**.

## Multica API target

- Target **Multica REST API v0.3.17** (constant `internal/version.MulticaAPI`).
- HTTP client lives in `internal/multica/` — update only that package when the upstream API changes.
- Send `X-Client-Platform: mcp` and `X-Client-Version` on every request (wired in `internal/multica/client.go`).

## Versioning (0.x.y)

- **Major** stays `0` during active development.
- Before merging to `main`, bump **`VERSION`** at the repo root:
  - **Patch** (`0.1.0` → `0.1.1`): fixes, docs-only, internal refactors.
  - **Minor** (`0.1.0` → `0.2.0`): new MCP tools, behavior changes, API alignment.
- Keep **`CHANGELOG.md`** in sync under `[Unreleased]`; move entries into a dated section when you bump `VERSION`.
- Releases are created automatically when `VERSION` on `main` is newer than the latest `v*` tag.

## Changelog workflow

1. Add user-facing changes under `## [Unreleased]` in `CHANGELOG.md`.
2. On release bump, rename `[Unreleased]` to `## [x.y.z] - YYYY-MM-DD` and add a fresh `[Unreleased]` section.
3. Use categories: Added, Changed, Deprecated, Removed, Fixed, Security.

## Build and test

```bash
go install github.com/strider2038/multica-mcp@latest   # from module root
make build    # → bin/multica-mcp
make test
make lint
```

Entry point: **`main.go`** at repository root (not under `cmd/`).

## MCP tools

- Handlers: `internal/mcp/server.go`
- Business logic: `internal/app/`
- Domain types: `internal/domain/`

When adding write tools, respect `MULTICA_READ_ONLY` via `UseCase.checkReadOnly()`.

## Pull requests

- Focused commits; run `make test` before push.
- Update README / `docs/usage.ru.md` if install paths or env vars change.
- Do not bump major past `0` without explicit maintainer approval.
