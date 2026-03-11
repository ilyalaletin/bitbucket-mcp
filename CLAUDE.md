# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCP server for Bitbucket Server / Data Center. Provides access to PRs, build statuses, and repository content via stdio transport. Uses mcp-go library, authenticated via Personal Access Token.

## Language & Build

- **Language**: Go
- **Build**: `go build ./cmd/bitbucket-mcp/`
- **Test**: `go test ./...` (single test: `go test -run TestName ./path/to/package`)
- **Lint**: `go vet ./...`
- **Run**: `BITBUCKET_URL=... BITBUCKET_TOKEN=... go run ./cmd/bitbucket-mcp/`

## Architecture

```
cmd/bitbucket-mcp/main.go          — entry point, tool registration, stdio server
internal/bitbucket/client.go       — HTTP client with retry (3 attempts, exp backoff)
internal/bitbucket/types.go        — Bitbucket API response structs
internal/tools/pr.go               — list_prs, get_pr, get_pr_diff, get_pr_commits
internal/tools/builds.go           — get_build_status
internal/tools/repos.go            — list_files, get_file_content, get_diff
```

**Flat architecture** — tools call the Bitbucket client directly, no service layer.

## MCP Tools (8 total)

- **PR:** `list_prs` (with role-based routing), `get_pr`, `get_pr_diff`, `get_pr_commits`
- **CI/CD:** `get_build_status`
- **Code:** `list_files`, `get_file_content`, `get_diff`

## Key Design Decisions

- `list_prs` uses dashboard endpoint when `role` is set, repo-scoped endpoint otherwise
- Pagination handled inside client, capped at max 100 results
- Diffs truncated at 1MB
- Binary files return a message instead of content
- Retry on 429, 5xx, network timeouts only
- Config via env vars: `BITBUCKET_URL`, `BITBUCKET_TOKEN`

## Development Rules

- **Branching:** All changes go in feature branches, never commit directly to main.
- **TDD:** Write tests first, see them fail, then implement. No code without a failing test.
- **Green tests required:** All tests must pass before merging. Run `go test ./...` to verify.

## Specs

- Design spec: `docs/superpowers/specs/2026-03-11-bitbucket-mcp-design.md`
