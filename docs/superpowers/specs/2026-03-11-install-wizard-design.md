# Install Wizard — Design Spec

## Overview

Interactive CLI wizard invoked with `--install` or `-i` flag. Detects installed AI agents, collects Bitbucket credentials, validates them, and writes MCP server config into the selected agents.

## Configuration

No new env vars. The wizard is fully interactive.

## Supported Agents

| Agent | Config Path | Format |
|-------|------------|--------|
| Claude Code | `~/.claude.json` | JSON (`mcpServers` key) |
| Cursor | `~/.cursor/mcp.json` | JSON (`mcpServers` key) |

Detection: agent is considered "installed" if its config file path exists on disk.

## Wizard Flow

1. **Detect agents** — scan config paths, build list of found agents
2. **No agents found** — print manual config snippet and exit 0
3. **Agent selection** — present numbered list, prompt for selection (1, 2, or "all"); default: all
4. **URL prompt** — ask for Bitbucket Server base URL
5. **Token instructions** — print step-by-step guide to generate a Personal Access Token at `{url}/plugins/servlet/access-tokens/manage`
6. **Token prompt** — ask for the token (plain stdin, no masking)
7. **Validation** — `GET {url}/rest/api/1.0/application-properties` with `Authorization: Bearer {token}`; on failure print error and exit 1
8. **Write config** — for each selected agent, merge `bitbucket` entry into `mcpServers` and write file
9. **Done message** — list files written, prompt to restart agents

## MCP Entry Written

```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "bitbucket-mcp",
      "env": {
        "BITBUCKET_URL": "<url>",
        "BITBUCKET_TOKEN": "<token>"
      }
    }
  }
}
```

If the config file already contains other `mcpServers` entries, they are preserved. If a `bitbucket` entry already exists, it is overwritten.

If the config file does not exist, it is created fresh.

## Architecture

### Project Structure

```
cmd/bitbucket-mcp/main.go        — add --install/-i flag check before env var validation
internal/install/
    wizard.go                    — Run() entry point, orchestrates the interactive flow
    detect.go                    — DetectAgents() returns list of installed agents
    config.go                    — ReadConfig(), MergeEntry(), WriteConfig() for JSON agent configs
    validate.go                  — ValidateCredentials(url, token) → error
    wizard_test.go
    detect_test.go
    config_test.go
    validate_test.go
```

### `main.go` Change

```go
if len(os.Args) > 1 && (os.Args[1] == "--install" || os.Args[1] == "-i") {
    os.Exit(install.Run())
}
```

This check happens before the env var validation block, so the wizard works without `BITBUCKET_URL`/`BITBUCKET_TOKEN` set.

### `internal/install/detect.go`

```go
type Agent struct {
    Name       string
    ConfigPath string
}

func DetectAgents() []Agent
```

Checks `~/.claude.json` and `~/.cursor/mcp.json`. Returns only those that exist.

### `internal/install/config.go`

```go
func ReadConfig(path string) (map[string]interface{}, error)
func MergeEntry(config map[string]interface{}, url, token string) map[string]interface{}
func WriteConfig(path string, config map[string]interface{}) error
```

- `ReadConfig`: reads JSON from path; if file doesn't exist returns empty map
- `MergeEntry`: sets `config["mcpServers"]["bitbucket"]` with command + env, leaves other keys intact
- `WriteConfig`: writes indented JSON (2-space), creates parent dirs if needed

### `internal/install/validate.go`

```go
func ValidateCredentials(baseURL, token string) error
```

Makes `GET {baseURL}/rest/api/1.0/application-properties` with `Authorization: Bearer {token}`. Returns nil on 2xx, error with status code otherwise. 10-second timeout.

### `internal/install/wizard.go`

```go
func Run() int
```

Orchestrates the full flow, reads from `os.Stdin`, writes to `os.Stdout`/`os.Stderr`. Returns exit code (0 = success, 1 = error).

## Edge Cases

- **No agents detected**: print manual snippet showing the JSON config, exit 0
- **Validation fails**: print HTTP status + response body, do not write any config, exit 1
- **Config file doesn't exist**: create it (and parent directory if needed)
- **Existing `bitbucket` entry**: overwrite it
- **Other existing `mcpServers` entries**: preserve them

## Dependencies

No new dependencies. Uses `encoding/json` and `net/http` from stdlib.
