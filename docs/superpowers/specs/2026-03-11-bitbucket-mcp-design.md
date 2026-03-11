# Bitbucket MCP Server — Design Spec

## Overview

MCP server providing access to Bitbucket Server / Data Center via stdio transport. Built in Go using the mcp-go library. Authenticated via Personal Access Token.

## Configuration

Environment variables:
- `BITBUCKET_URL` — base URL of Bitbucket Server instance (required)
- `BITBUCKET_TOKEN` — HTTP Access Token (required)

## MCP Tools

### Pull Requests

| Tool | Description | Parameters |
|------|-------------|------------|
| `list_prs` | List PRs with filters. Uses `/rest/api/1.0/dashboard/pull-requests` for role-based filtering | `project`, `repo` (optional when using role filter), `state` (OPEN/MERGED/DECLINED/ALL), `role` (AUTHOR/REVIEWER, optional) |
| `get_pr` | PR details: description, author, reviewers, status | `project`, `repo`, `pr_id` |
| `get_pr_diff` | Full PR diff | `project`, `repo`, `pr_id` |
| `get_pr_commits` | List commits in a PR | `project`, `repo`, `pr_id` |

### CI/CD (Build Status)

| Tool | Description | Parameters |
|------|-------------|------------|
| `get_build_status` | Build statuses for a commit via `/rest/build-status/1.0/commits/{commitId}` | `commit_id` |

### Browse Code & Diff

| Tool | Description | Parameters |
|------|-------------|------------|
| `list_files` | List files/directories in a repo | `project`, `repo`, `path` (optional), `ref` (branch/commit, optional) |
| `get_file_content` | Read file content | `project`, `repo`, `path`, `ref` (optional) |
| `get_diff` | Diff between two refs | `project`, `repo`, `from`, `to`, `path` (optional, scope to single file) |

## Architecture

### Project Structure

```
cmd/bitbucket-mcp/main.go
internal/bitbucket/client.go    — HTTP client with retry
internal/bitbucket/types.go     — Bitbucket API response structs
internal/tools/pr.go            — PR tool handlers
internal/tools/builds.go        — Build status tool handler
internal/tools/repos.go         — Browse/diff tool handlers
```

### HTTP Client (`internal/bitbucket`)

Single `Client` struct:

```go
type Client struct {
    baseURL    string
    token      string
    httpClient *http.Client
}
```

- All methods return `(result, error)`
- Pagination handled internally (Bitbucket Server uses `isLastPage` + `nextPageStart`) — methods return complete results
- API errors wrapped with status code and response body
- Retry: up to 3 attempts with exponential backoff (1s, 2s, 4s) on recoverable errors (429, 5xx, network timeouts). No retry on 4xx (except 429).

### Entry Point (`cmd/bitbucket-mcp/main.go`)

1. Read `BITBUCKET_URL` and `BITBUCKET_TOKEN` from env (exit on missing)
2. Create `bitbucket.Client`
3. Create mcp-go server with stdio transport
4. Register all 8 tools with descriptions and input schemas
5. Start server

### Dependencies

- `github.com/mark3labs/mcp-go` — MCP protocol implementation
- Go standard library for everything else

## Bitbucket Server API Endpoints

- `GET /rest/api/1.0/projects/{project}/repos/{repo}/pull-requests` — list PRs by repo
- `GET /rest/api/1.0/dashboard/pull-requests` — list PRs by role (assigned/authored)
- `GET /rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{prId}` — PR details
- `GET /rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{prId}/diff` — PR diff
- `GET /rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{prId}/commits` — PR commits
- `GET /rest/build-status/1.0/commits/{commitId}` — build statuses
- `GET /rest/api/1.0/projects/{project}/repos/{repo}/browse/{path}?at={ref}` — file listing / content
- `GET /rest/api/1.0/projects/{project}/repos/{repo}/diff/{path}?from={from}&to={to}` — diff between refs
