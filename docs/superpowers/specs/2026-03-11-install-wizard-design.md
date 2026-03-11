# Install Wizard — Design Spec

## Overview

Interactive CLI wizard invoked with `--install` or `-i` flag. Detects installed AI agents, collects Bitbucket credentials, validates them, and writes MCP server config into the selected agents.

## Supported Platforms

macOS and Linux only. Windows is out of scope. Config paths use `~` which resolves via `os.UserHomeDir()`.

## Configuration

No new env vars. The wizard is fully interactive. Non-interactive use (stdin redirected) is not supported — if stdin reaches EOF before all prompts are answered, the wizard prints `"Error: unexpected end of input"` and exits 1.

## Supported Agents

| Agent | Config Path | Format |
|-------|------------|--------|
| Claude Code | `~/.claude.json` | JSON (`mcpServers` key) |
| Cursor | `~/.cursor/mcp.json` | JSON (`mcpServers` key) |

Detection: agent is considered "installed" if its config file path exists on disk. Known limitation: the file may exist without the agent binary being present (e.g. after uninstall), or a freshly installed agent that has never launched may not yet have its config file. This heuristic is intentional — it's simple and correct in the common case.

## Wizard Flow

1. **Detect agents** — call `os.UserHomeDir()` to expand `~`; on error print `"Error: cannot determine home directory: <err>"` and exit 1. Scan config paths, build list of found agents.
2. **No agents found** — print static manual config template (see below) and exit 0. Do not prompt for credentials.
3. **Agent selection** — if more than one agent found, present numbered list and prompt for selection (e.g. "1", "2", or "all", case-insensitive, no comma-separated lists); default: all. Empty input uses the default. If exactly one agent found, skip the selection prompt and proceed directly. On unrecognized non-empty input, re-prompt.
4. **URL prompt** — ask for Bitbucket Server base URL. Strip trailing slash and trim whitespace. Re-prompt if empty after trimming.
5. **URL pre-validation** — if URL does not start with `http://` or `https://`, print `"Error: URL must start with http:// or https://"` and re-prompt (consistent with the empty-input re-prompt behavior; bad scheme is treated as a correctable user error, not a fatal condition).
6. **Token instructions** — print step-by-step guide to generate a Personal Access Token at `{normalizedURL}/plugins/servlet/access-tokens/manage`
7. **Token prompt** — ask for the token. Trim whitespace. Re-prompt if empty after trimming. No terminal echo suppression — the `io.Reader`/`io.Writer` abstraction used for testability does not support TTY control; this is intentional.
8. **Validation** — single pass before writing any config: `GET {normalizedURL}/rest/api/1.0/application-properties` with `Authorization: Bearer {token}`; response body is read and discarded on success (2xx). On failure, print `"Error: validation failed (HTTP <status>)"` and exit 1. No retry — user must re-run the wizard to correct credentials.
9. **Write config** — for each selected agent, read existing config, merge `bitbucket` entry, write file with mode `0600`. If a write fails partway through multiple agents, print `"Error writing <path>: <err>\nNote: config was already written to: <previously-written-paths>"` and exit 1 (no rollback).
10. **Security reminder** — print: `"Note: your token is stored in plaintext in the config file."`
11. **Done message** — list files written, prompt to restart agents

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

`"command": "bitbucket-mcp"` — bare name, assumes the binary is on `$PATH`. This is intentional. If the binary is not on PATH, the agent will fail to start the server at runtime.

If the config file already contains other `mcpServers` entries, they are preserved. If a `bitbucket` entry already exists, it is overwritten. If `mcpServers` exists in the file but is not a JSON object (e.g. array or null), it is overwritten with a new object.

`WriteConfig` rewrites the entire file with 2-space indentation regardless of the original file's formatting. This is intentional — the entire file is always round-tripped through `encoding/json`.

## Manual Config Template (no agents found)

When no agents are detected, print exactly:

```
No supported agents found (Claude Code, Cursor).

To configure manually, add the following to your agent's MCP config:

{
  "mcpServers": {
    "bitbucket": {
      "command": "bitbucket-mcp",
      "env": {
        "BITBUCKET_URL": "<YOUR_BITBUCKET_URL>",
        "BITBUCKET_TOKEN": "<YOUR_TOKEN>"
      }
    }
  }
}
```

## Architecture

### Project Structure

```
cmd/bitbucket-mcp/main.go        — add --install/-i flag check before env var validation
internal/install/
    wizard.go                    — Run(in io.Reader, out io.Writer) int entry point
    detect.go                    — DetectAgents() returns list of installed agents
    config.go                    — ReadConfig(), MergeEntry(), WriteConfig() for JSON agent configs
    validate.go                  — ValidateCredentials(url, token) → error
    wizard_test.go
    detect_test.go
    config_test.go
    validate_test.go
```

### `main.go` Change

Import: `"github.com/ilyalaletin/bitbucket-mcp/internal/install"`

```go
if len(os.Args) > 1 && (os.Args[1] == "--install" || os.Args[1] == "-i") {
    os.Exit(install.Run(os.Stdin, os.Stdout))
}
```

This check is positional — only `os.Args[1]` is examined as an exact string. `--install` in a later position, `--install=value`, or any other variation is silently ignored and the server starts normally (failing on missing env vars). This is intentional for simplicity.

The check happens before the env var validation block so the wizard works without `BITBUCKET_URL`/`BITBUCKET_TOKEN` set.

### `internal/install/detect.go`

```go
type Agent struct {
    Name       string
    ConfigPath string // absolute path, ~ already expanded
}

func DetectAgents() ([]Agent, error)
```

Calls `os.UserHomeDir()` to expand `~`. Returns error if home dir cannot be determined. Checks `<home>/.claude.json` and `<home>/.cursor/mcp.json`. Returns only those whose config file exists.

### `internal/install/config.go`

```go
func ReadConfig(path string) (map[string]interface{}, error)
func MergeEntry(config map[string]interface{}, url, token string) map[string]interface{}
func WriteConfig(path string, config map[string]interface{}) error
```

- `ReadConfig`: reads JSON from path; if file doesn't exist returns empty map and nil error (this path is only reachable by direct callers, not by the wizard — detection ensures the file exists for detected agents); if file exists but contains invalid JSON returns an error wrapping the path and parse error
- `WriteConfig`: marshals to indented JSON (2-space), writes file with mode `0600`. Creates parent directory with `os.MkdirAll` before writing (handles the TOCTOU window where the file could be deleted between detection and write, and also supports direct callers where detection hasn't run).
- `MergeEntry`: reads `config["mcpServers"]` — if it's a `map[string]interface{}`, merges into it; otherwise replaces it with a new map. Sets `bitbucket` to `{"command": "bitbucket-mcp", "env": {"BITBUCKET_URL": url, "BITBUCKET_TOKEN": token}}`.

### `internal/install/validate.go`

```go
func ValidateCredentials(baseURL, token string) error
```

Makes `GET {baseURL}/rest/api/1.0/application-properties` with `Authorization: Bearer {token}`. Returns nil on 2xx (response body discarded). Returns `fmt.Errorf("HTTP %d", statusCode)` on non-2xx. 10-second timeout.

### `internal/install/wizard.go`

```go
func Run(in io.Reader, out io.Writer) int
```

Orchestrates the full flow. Reads input line-by-line using `bufio.NewReader(in).ReadString('\n')` (avoids `bufio.Scanner`'s 64KB line limit). Writes all output (prompts, errors, results) to `out`. Stdout and stderr are intentionally merged into a single writer for simplicity and testability. Returns exit code (0 = success, 1 = error).

## Edge Cases

| Situation | Behavior |
|-----------|----------|
| No agents detected | Print manual template, exit 0 |
| Single agent detected | Skip selection prompt |
| Empty selection input | Use default (all) |
| Invalid non-empty selection input | Re-prompt |
| Empty URL input | Re-prompt |
| URL missing scheme | Print `"Error: URL must start with http:// or https://"`, re-prompt |
| URL with trailing slash | Strip before use |
| Whitespace in URL/token input | Trim before use |
| Empty token input | Re-prompt |
| EOF on stdin | Print `"Error: unexpected end of input"`, exit 1 |
| Validation fails | Print `"Error: validation failed (HTTP <status>)"`, exit 1, no config written |
| Config file doesn't exist | ReadConfig returns empty map (defensive); WriteConfig creates file with mode 0600 |
| Config file contains invalid JSON | Print `"Error: <path> contains invalid JSON: <err>"`, exit 1 |
| `mcpServers` key is not a JSON object | Overwrite with new object |
| Existing `bitbucket` entry | Overwrite it |
| Other existing `mcpServers` entries | Preserve them |
| Write fails mid-way (multiple agents) | Print error + which agents succeeded, exit 1, no rollback |
| `--install` in non-[1] arg position | Silently ignored; server starts normally |
| `--install=value` | Silently ignored (does not match exact string) |
| `os.UserHomeDir()` fails | Print error, exit 1 |

## Dependencies

No new dependencies. Uses `encoding/json`, `net/http`, `io`, and `bufio` from stdlib.
