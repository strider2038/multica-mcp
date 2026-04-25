# multica-mcp-server

MCP server for [Multica](https://multica.ai) — the open-source managed agents platform. Allows local coding agents (Claude Code, OpenCode, Codex, etc.) to interact with Multica projects, tasks, comments, and agents via the Model Context Protocol.

## Features

- **13 MCP tools** for full Multica integration: list projects, create/update tasks, add comments, search, plan breakdowns, etc.
- **stdio and HTTP transports** — connect via CLI pipe or HTTP endpoint
- **Read-only mode** — safe deployment where write tools are disabled
- **Dry-run support** — validate create/update operations without side effects
- **Status validation** — rejects invalid task status transitions
- **Auto workspace detection** — uses the only workspace if you have one

## Quick Start

### Build

```bash
make build
```

### Configure

```bash
export MULTICA_BASE_URL=https://multica.ai    # or your self-hosted URL
export MULTICA_TOKEN=mul_your_token_here       # personal access token
export MULTICA_WORKSPACE_ID=ws_abc123          # optional if you have one workspace
export MCP_TRANSPORT=stdio                     # stdio (default) or http
export LOG_LEVEL=info                          # debug, info, warn, error
```

### Run

```bash
# stdio mode (for agent integration)
./bin/multica-mcp-server

# HTTP mode
MCP_TRANSPORT=http ./bin/multica-mcp-server
```

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `MULTICA_BASE_URL` | Yes | — | Multica server URL |
| `MULTICA_TOKEN` | Yes | — | Personal access token (PAT) |
| `MULTICA_WORKSPACE_ID` | No* | auto | Workspace ID. Auto-detected if you have one workspace |
| `MCP_TRANSPORT` | No | `stdio` | Transport: `stdio` or `http` |
| `MCP_HTTP_PORT` | No | `8080` | HTTP server port (when transport=http) |
| `MCP_API_KEY` | No | — | API key for HTTP authentication. Required for self-hosted |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |
| `MULTICA_READ_ONLY` | No | `false` | Disable all write operations |

*Required when you belong to multiple workspaces.

## MCP Tools

### Read Operations

| Tool | Description |
|---|---|
| `multica_list_projects` | List projects (optional name filter) |
| `multica_get_project` | Get project details |
| `multica_list_tasks` | List tasks with filters (project, status, assignee, query) |
| `multica_get_task` | Get task with comments and subtasks |
| `multica_search_tasks` | Full-text search across titles, descriptions, comments |
| `multica_list_agents` | List workspace agents |
| `multica_plan_task_breakdown` | Generate a subtask plan (no tasks created) |

### Write Operations (disabled in read-only mode)

| Tool | Description |
|---|---|
| `multica_create_task` | Create a task |
| `multica_create_subtask` | Create a subtask under a parent task |
| `multica_update_task` | Update task fields |
| `multica_add_comment` | Add a comment to a task |
| `multica_assign_task` | Assign task to member or agent |
| `multica_create_task_with_subtasks` | Create parent + subtasks atomically |

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
  "assignee": "agent-uuid-here"
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

## Agent Configuration

### Claude Code (`.claude/settings.json`)

```json
{
  "mcpServers": {
    "multica": {
      "command": "/path/to/multica-mcp-server",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_token",
        "MULTICA_WORKSPACE_ID": "ws_abc123"
      }
    }
  }
}
```

### OpenCode (`opencode.json`)

```json
{
  "mcp": {
    "multica": {
      "command": "multica-mcp-server",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_your_token"
      }
    }
  }
}
```

## Architecture

```
cmd/multica-mcp-server/    Entry point
internal/
  config/                  Environment configuration
  domain/                  Domain models (Project, Task, Comment, Agent)
  multica/                 HTTP client adapter for Multica API
  app/                     Use case / business logic layer
  mcp/                     MCP tool handlers and registration
  logging/                 Structured logging (slog)
```

The architecture isolates the MCP transport layer from business logic. The `internal/multica` package is the only one that knows about HTTP endpoints — if the Multica API changes, only that package needs updating.

## Self-Hosted (VPS)

Run as an HTTP server with API key authentication:

```bash
MCP_TRANSPORT=http \
MCP_HTTP_PORT=8080 \
MCP_API_KEY=your-secret-key \
./bin/multica-mcp-server
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

```bash
make test     # run tests
make lint     # run go vet
make build    # build binary
```

## Token Setup

1. Go to your Multica instance → Settings → Personal Access Tokens
2. Create a new token (starts with `mul_`)
3. Copy the token — it's shown only once

## License

See [LICENSE](LICENSE).
