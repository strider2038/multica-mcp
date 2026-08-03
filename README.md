# multica-mcp

MCP server for [Multica](https://multica.ai) — the open-source managed agents platform. Allows local coding agents (Cursor, Claude Code, OpenCode, Codex, etc.) to interact with Multica projects, tasks, comments, and agents via the Model Context Protocol.

Targets **Multica REST API v0.4.16**.

## Features

- **15 MCP tools** for full Multica integration: list projects, create/update tasks, add comments, search, plan breakdowns, etc.
- **stdio and HTTP transports** — connect via CLI pipe or HTTP endpoint
- **Read-only mode** — safe deployment where write tools are disabled
- **Dry-run support** — validate create/update operations without side effects
- **Status validation** — rejects invalid task status transitions
- **Auto workspace detection** — uses the only workspace if you have one

## Quick Start

### Install with Go

Requires Go 1.25+ and a `GOBIN` on your `PATH` (often `~/go/bin`):

```bash
go install github.com/strider2038/multica-mcp@latest
```

The binary is installed as **`multica-mcp`**. Use `$(go env GOPATH)/bin/multica-mcp` or ensure `GOBIN` is on your `PATH`.

### Install a release binary

Download the archive for your OS and CPU from the [GitHub releases](https://github.com/strider2038/multica-mcp/releases) page:

- Linux x86_64: `multica-mcp-linux-amd64.tar.gz`
- Linux ARM64: `multica-mcp-linux-arm64.tar.gz`
- macOS Intel: `multica-mcp-darwin-amd64.tar.gz`
- macOS Apple Silicon: `multica-mcp-darwin-arm64.tar.gz`
- Windows x86_64: `multica-mcp-windows-amd64.zip`
- Windows ARM64: `multica-mcp-windows-arm64.zip`

Linux/macOS example:

```bash
tar -xzf multica-mcp-linux-amd64.tar.gz
chmod +x multica-mcp-linux-amd64
sudo mv multica-mcp-linux-amd64 /usr/local/bin/multica-mcp
```

Windows PowerShell example:

```powershell
Expand-Archive .\multica-mcp-windows-amd64.zip .
Move-Item .\multica-mcp-windows-amd64.exe $env:USERPROFILE\bin\multica-mcp.exe
```

Use the installed path in your MCP client configuration, for example `/usr/local/bin/multica-mcp` on Linux/macOS.

### Build from source

```bash
make build
```

### Configure

```bash
export MULTICA_BASE_URL=https://multica.ai   # or your self-hosted Multica URL
export MULTICA_TOKEN=mul_your_token_here       # required: PAT from Multica settings

# Workspace (pick one strategy — see Configuration below)
# export MULTICA_WORKSPACE_ID=550e8400-e29b-41d4-a716-446655440000
# export MULTICA_WORKSPACE_SLUG=my-team

export MCP_TRANSPORT=stdio                    # stdio (default) or http
export LOG_LEVEL=info                          # debug, info, warn, error
```

### Run

```bash
# stdio mode (for agent integration)
./bin/multica-mcp

# HTTP mode
MCP_TRANSPORT=http ./bin/multica-mcp
```

## Configuration

Environment variables are read at process startup (`internal/config`). The Multica HTTP client sends `X-Workspace-ID` or `X-Workspace-Slug` on workspace-scoped routes (see `internal/multica/client.go`).

### Environment variables

| Variable | Required | Default | Description |
| -------- | -------- | ------- | ----------- |
| `MULTICA_BASE_URL` | **Yes** | — | Base URL of your Multica instance (e.g. `https://multica.ai` or self-hosted origin). |
| `MULTICA_TOKEN` | **Yes** | — | Personal access token (PAT), usually prefixed with `mul_`. |
| `MULTICA_WORKSPACE_ID` | No† | auto | Workspace UUID. If unset and your account has **exactly one** workspace, its ID is detected automatically. Ignored for API headers when `MULTICA_WORKSPACE_SLUG` is set. |
| `MULTICA_WORKSPACE_SLUG` | No† | — | Workspace slug (human-readable id, e.g. `acme-backend`). Sent as `X-Workspace-Slug`. If set (non-empty after trim), it **takes precedence** over `MULTICA_WORKSPACE_ID`. |
| `MCP_TRANSPORT` | No | `stdio` | `stdio` (pipe to the IDE/agent) or `http` (standalone MCP HTTP server). |
| `MCP_HTTP_PORT` | No | `8080` | Listen port when `MCP_TRANSPORT=http`. |
| `MCP_API_KEY` | No | — | When set, HTTP transport requires `Authorization: Bearer <key>` on MCP requests. Recommended for exposed HTTP deployments. |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` (structured logs via `slog`). |
| `MULTICA_READ_ONLY` | No | `false` | If `true`, write MCP tools return an error without calling the Multica API. |

† **Workspace:** You must end up with a resolvable workspace: either auto-detection (single workspace), or set **`MULTICA_WORKSPACE_ID`** or **`MULTICA_WORKSPACE_SLUG`**. If your account has **multiple** workspaces and neither variable is set, the server exits and prints the available workspaces — then set one of the two variables.

## MCP Tools

### Read Operations


| Tool                          | Description                                                |
| ----------------------------- | ---------------------------------------------------------- |
| `multica_list_projects`       | List projects (optional name filter)                       |
| `multica_get_project`         | Get project details                                        |
| `multica_list_tasks`          | List tasks with filters (project, status, assignee, query) |
| `multica_get_task`            | Get task with comments and subtasks                        |
| `multica_search_tasks`        | Full-text search across titles, descriptions, comments     |
| `multica_list_agents`         | List workspace agents                                      |
| `multica_plan_task_breakdown` | Generate a subtask plan (no tasks created)                 |


### Write Operations (disabled in read-only mode)


| Tool                                | Description                          |
| ----------------------------------- | ------------------------------------ |
| `multica_create_task`               | Create a task                        |
| `multica_create_subtask`            | Create a subtask under a parent task |
| `multica_update_task`               | Update task fields                   |
| `multica_add_comment`               | Add a comment to a task              |
| `multica_assign_task`               | Assign task to member or agent       |
| `multica_create_task_with_subtasks` | Create parent + subtasks atomically  |


### Task Statuses

`backlog`, `todo`, `in_progress`, `in_review`, `done`, `blocked`, `cancelled`

### Priorities

`none`, `urgent`, `high`, `medium`, `low`

## API Examples

### List projects

```json
{"query": "backend"}
```

Response:

```json
[
  {"id": "p1", "title": "Backend API", "status": "active", "issue_count": 12}
]
```

### Create a task

```json
{
  "project_id": "p1",
  "title": "Add pagination to list endpoint",
  "description": "Implement cursor-based pagination for the issues list endpoint.",
  "priority": "medium",
  "assignee": "agent-uuid-here",
  "assignee_type": "agent"
}
```

### Search tasks

```json
{"query": "pagination", "status": "in_progress", "limit": 10}
```

### Plan task breakdown

```json
{
  "title": "Implement user authentication",
  "description": "Add OAuth2 login with Google and GitHub providers",
  "project_context": "This is a Go backend with Chi router"
}
```

## Agent configuration

Use the same environment variables as in the table above. Prefer an **absolute path** to `multica-mcp` in `command` (from `go install` or `make build`).

### Cursor

1. **Project MCP:** add `.cursor/mcp.json` in the project root, **or** user-level config (often `~/.cursor/mcp.json` on Linux/macOS — depends on Cursor version). You can also use **Cursor Settings → MCP** to register the server in the UI.
2. Set `command` to the full path of your binary (`go install` → `$(go env GOPATH)/bin/multica-mcp`, or `make build` → `bin/multica-mcp` in this repo).
3. Reload MCP servers after editing (Command Palette: “MCP: Restart” / restart Cursor).

Example **`.cursor/mcp.json`** (workspace by **slug**; swap for `MULTICA_WORKSPACE_ID` if you prefer UUID):

```json
{
  "mcpServers": {
    "multica": {
      "command": "/absolute/path/to/multica-mcp/bin/multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_personal_access_token",
        "MULTICA_WORKSPACE_SLUG": "my-workspace-slug",
        "MCP_TRANSPORT": "stdio",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

Example with **workspace UUID** instead of slug (do not set both unless you intend slug to win):

```json
{
  "mcpServers": {
    "multica": {
      "command": "/absolute/path/to/multica-mcp/bin/multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_personal_access_token",
        "MULTICA_WORKSPACE_ID": "550e8400-e29b-41d4-a716-446655440000",
        "MULTICA_READ_ONLY": "false"
      }
    }
  }
}
```

Optional extras you can add under `env`: `MULTICA_READ_ONLY=true`, `LOG_LEVEL=debug`, or (for HTTP mode) `MCP_TRANSPORT=http`, `MCP_HTTP_PORT=8080`, `MCP_API_KEY=...`.

### Claude Code (`.claude/settings.json`)

```json
{
  "mcpServers": {
    "multica": {
      "command": "/path/to/multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_token",
        "MULTICA_WORKSPACE_ID": "550e8400-e29b-41d4-a716-446655440000"
      }
    }
  }
}
```

You can replace `MULTICA_WORKSPACE_ID` with `MULTICA_WORKSPACE_SLUG` when you configure by slug.

### OpenCode (`opencode.json`)

```json
{
  "mcp": {
    "multica": {
      "command": "/path/to/multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_token",
        "MULTICA_WORKSPACE_SLUG": "my-workspace-slug"
      }
    }
  }
}
```

## Architecture

```
main.go                    Entry point (go install / go build)
internal/
  config/                  Environment configuration
  domain/                  Domain models (Project, Task, Comment, Agent)
  multica/                 HTTP client adapter for Multica API v0.4.16
  app/                     Use case / business logic layer
  mcp/                     MCP tool handlers and registration
  version/                 Server and API version constants
  logging/                 Structured logging (slog)
```

The architecture isolates the MCP transport layer from business logic. The `internal/multica` package is the only one that knows about HTTP endpoints — if the Multica API changes, only that package needs updating.

Agent contributors: see [AGENTS.md](AGENTS.md) for versioning, changelog, and release workflow.

## Self-Hosted (VPS)

Run as an HTTP server with API key authentication:

```bash
MCP_TRANSPORT=http \
MCP_HTTP_PORT=8080 \
MCP_API_KEY=your-secret-key \
./bin/multica-mcp
```

Clients must send `Authorization: Bearer your-secret-key` with every request. Without `MCP_API_KEY`, authentication is disabled (suitable for local/stdio use only).

### With reverse proxy (Caddy)

```Caddyfile
multica-mcp.example.com {
    reverse_proxy localhost:8080
}
```

### Connecting a remote agent

Configure the agent to use the HTTP endpoint:

```json
{
  "mcpServers": {
    "multica": {
      "url": "https://multica-mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer your-secret-key"
      }
    }
  }
}
```

## Development

See [AGENTS.md](AGENTS.md) for versioning (`VERSION`, `CHANGELOG.md`) and release rules.

```bash
make test     # run tests
make lint     # run go vet
make build    # build bin/multica-mcp
```

Pushes to `main` with a new `VERSION` trigger an automated GitHub release (`v0.x.y`).

## Token Setup

1. Go to your Multica instance → Settings → Personal Access Tokens
2. Create a new token (starts with `mul_`)
3. Copy the token — it's shown only once

## License

See [LICENSE](LICENSE).
