# Bitbucket MCP Server

[![Go Version](https://img.shields.io/github/go-mod/go-version/ilyalaletin/bitbucket-mcp?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/ilyalaletin/bitbucket-mcp?style=flat-square)](https://github.com/ilyalaletin/bitbucket-mcp/releases)
[![Tests](https://img.shields.io/github/actions/workflow/status/ilyalaletin/bitbucket-mcp/test.yml?style=flat-square&label=tests)](https://github.com/ilyalaletin/bitbucket-mcp/actions)

MCP (Model Context Protocol) server providing access to Bitbucket Server / Data Center via stdio transport. Authenticate with a Personal Access Token and query pull requests, build statuses, and repository content.

## Features

- 📋 **Pull Requests**: List, get details, view diffs, list commits
- 🔨 **CI/CD**: Access build statuses for commits
- 📁 **Code**: Browse files, get content, view diffs
- ⚙️ **Easy Setup**: Interactive `--install` wizard for Claude Code and Cursor

## Installation

### Option 1: Using go install (recommended)

```bash
go install github.com/ilyalaletin/bitbucket-mcp/cmd/bitbucket-mcp@latest
```

Or a specific version:
```bash
go install github.com/ilyalaletin/bitbucket-mcp/cmd/bitbucket-mcp@v0.1
```

### Option 2: Download binary

Download from [releases](https://github.com/ilyalaletin/bitbucket-mcp/releases):
- Linux: `bitbucket-mcp-linux-amd64` or `bitbucket-mcp-linux-arm64`
- macOS: `bitbucket-mcp-darwin-amd64` or `bitbucket-mcp-darwin-arm64`

```bash
# Extract and move to PATH
chmod +x bitbucket-mcp-*
mv bitbucket-mcp-* /usr/local/bin/bitbucket-mcp
```

### Option 3: Build from source

```bash
git clone https://github.com/ilyalaletin/bitbucket-mcp.git
cd bitbucket-mcp
go build ./cmd/bitbucket-mcp/
mv bitbucket-mcp /usr/local/bin/  # optional
```

### Configure with install wizard

```bash
./bitbucket-mcp --install
# or
./bitbucket-mcp -i
```

The wizard will:
1. Detect installed agents (Claude Code, Cursor)
2. Prompt for your Bitbucket Server URL
3. Guide you to generate a Personal Access Token
4. Validate credentials and save config

### Manual setup (if wizard doesn't detect your agent)

Add to your agent's MCP config:

**Claude Code** (`~/.claude.json`):
```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "bitbucket-mcp",
      "env": {
        "BITBUCKET_URL": "https://bitbucket.example.com",
        "BITBUCKET_TOKEN": "your-personal-access-token"
      }
    }
  }
}
```

**Cursor** (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "bitbucket-mcp",
      "env": {
        "BITBUCKET_URL": "https://bitbucket.example.com",
        "BITBUCKET_TOKEN": "your-personal-access-token"
      }
    }
  }
}
```

### Restart your agent

Restart Claude Code or Cursor to activate the MCP server.

## Usage

Once installed, ask Claude / Cursor to:
- "List open PRs in project KEY"
- "Show the diff for PR #123 in project KEY/repo"
- "Get build status for commit abc123def"
- "List files in the repo root"

## MCP Tools

| Tool | Description |
|------|-------------|
| `list_prs` | List pull requests (by repo or by role) |
| `get_pr` | Get PR details, description, reviewers |
| `get_pr_diff` | Full diff for a PR |
| `get_pr_commits` | List commits in a PR |
| `get_build_status` | Build statuses for a commit |
| `list_files` | List files/directories in a repo path |
| `get_file_content` | Read file content |
| `get_diff` | Diff between two refs |

## Development

### Build
```bash
go build ./cmd/bitbucket-mcp/
```

### Test
```bash
go test ./...
```

### Lint
```bash
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE)

## Support

- **Issues**: [GitHub Issues](https://github.com/ilyalaletin/bitbucket-mcp/issues)
- **Spec**: See [design spec](docs/superpowers/specs/2026-03-11-bitbucket-mcp-design.md)
