# Install Wizard Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add interactive `--install` / `-i` flag to set up MCP server config in Claude Code and Cursor automatically.

**Architecture:** Five focused tasks — detect agents, validate credentials, read/write JSON configs, and orchestrate the wizard flow. All tests use dependency injection (passing `io.Reader`/`io.Writer`) to mock I/O.

**Tech Stack:** Go stdlib only. TDD with `testing.T`.

---

## Chunk 1: Agent Detection & Config Management

### Task 1: Implement agent detection

**Files:**
- Create: `internal/install/detect.go`
- Create: `internal/install/detect_test.go`

**Context:** The wizard needs to find which agents are installed by checking if their config files exist on the user's machine.

- [ ] **Step 1: Write failing test for `DetectAgents`**

```go
// internal/install/detect_test.go
package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents(t *testing.T) {
	// Create temp home dir
	tmpHome := t.TempDir()

	// Create Claude Code config
	claudePath := filepath.Join(tmpHome, ".claude.json")
	os.WriteFile(claudePath, []byte(`{"mcpServers":{}}`), 0644)

	// Mock os.UserHomeDir
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	agents, err := DetectAgents()
	if err != nil {
		t.Fatalf("DetectAgents failed: %v", err)
	}

	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "Claude Code" {
		t.Fatalf("expected 'Claude Code', got %q", agents[0].Name)
	}
	if agents[0].ConfigPath != claudePath {
		t.Fatalf("expected %q, got %q", claudePath, agents[0].ConfigPath)
	}
}

func TestDetectAgents_NoAgents(t *testing.T) {
	tmpHome := t.TempDir()
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	agents, err := DetectAgents()
	if err != nil {
		t.Fatalf("DetectAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(agents))
	}
}

func TestDetectAgents_HomeDirError(t *testing.T) {
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	_, err := DetectAgents()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -run TestDetectAgents -v`

Expected: FAIL with "undefined: userHomeDirFn" or "undefined: DetectAgents"

- [ ] **Step 3: Implement agent detection**

```go
// internal/install/detect.go
package install

import (
	"os"
	"path/filepath"
)

type Agent struct {
	Name       string
	ConfigPath string
}

var userHomeDirFn = os.UserHomeDir // testable hook

func DetectAgents() ([]Agent, error) {
	homeDir, err := userHomeDirFn()
	if err != nil {
		return nil, err
	}

	var agents []Agent

	// Check Claude Code
	claudePath := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(claudePath); err == nil {
		agents = append(agents, Agent{
			Name:       "Claude Code",
			ConfigPath: claudePath,
		})
	}

	// Check Cursor
	cursorPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	if _, err := os.Stat(cursorPath); err == nil {
		agents = append(agents, Agent{
			Name:       "Cursor",
			ConfigPath: cursorPath,
		})
	}

	return agents, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -run TestDetectAgents -v`

Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/install/detect.go internal/install/detect_test.go
git commit -m "feat: implement agent detection for install wizard"
```

---

### Task 2: Implement credential validation

**Files:**
- Create: `internal/install/validate.go`
- Create: `internal/install/validate_test.go`

**Context:** The wizard needs to validate the Bitbucket URL and token before writing config.

- [ ] **Step 1: Write failing test for `ValidateCredentials`**

```go
// internal/install/validate_test.go
package install

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateCredentials_Success(t *testing.T) {
	// Mock Bitbucket API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/application-properties" {
			t.Errorf("expected /rest/api/1.0/application-properties, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer validtoken" {
			t.Errorf("expected 'Bearer validtoken', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"6.0.0"}`))
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "validtoken")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateCredentials_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "badtoken")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != fmt.Sprintf("HTTP %d", http.StatusUnauthorized) {
		t.Fatalf("expected 'HTTP 401', got %q", err.Error())
	}
}

func TestValidateCredentials_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server (will timeout)
		select {}
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -run TestValidateCredentials -v`

Expected: FAIL with "undefined: ValidateCredentials"

- [ ] **Step 3: Implement credential validation**

```go
// internal/install/validate.go
package install

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func ValidateCredentials(baseURL, token string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", baseURL+"/rest/api/1.0/application-properties", nil)
	if err != nil {
		return fmt.Errorf("HTTP %d", http.StatusBadRequest) // Simplified for bad URL
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Discard response body
	io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -run TestValidateCredentials -v`

Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/install/validate.go internal/install/validate_test.go
git commit -m "feat: implement Bitbucket credential validation"
```

---

### Task 3: Implement config file handling

**Files:**
- Create: `internal/install/config.go`
- Create: `internal/install/config_test.go`

**Context:** Read, merge, and write JSON MCP config files for Claude Code and Cursor.

- [ ] **Step 1: Write failing test for config functions**

```go
// internal/install/config_test.go
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_FileNotFound(t *testing.T) {
	config, err := ReadConfig("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(config) != 0 {
		t.Fatalf("expected empty config, got %v", config)
	}
}

func TestReadConfig_ValidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	data := `{"mcpServers":{"other":{"command":"other"}}}`
	os.WriteFile(tmpFile, []byte(data), 0644)

	config, err := ReadConfig(tmpFile)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers to be map, got %T", config["mcpServers"])
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("expected 'other' server in config")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(tmpFile, []byte("{invalid json}"), 0644)

	_, err := ReadConfig(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMergeEntry(t *testing.T) {
	config := make(map[string]interface{})
	result := MergeEntry(config, "https://bitbucket.example.com", "mytoken")

	servers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers in result")
	}

	bitbucket, ok := servers["bitbucket"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected bitbucket entry")
	}

	if bitbucket["command"] != "bitbucket-mcp" {
		t.Fatalf("expected command='bitbucket-mcp', got %v", bitbucket["command"])
	}

	env, ok := bitbucket["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected env in bitbucket entry")
	}

	if env["BITBUCKET_URL"] != "https://bitbucket.example.com" {
		t.Fatalf("expected BITBUCKET_URL to be set")
	}
	if env["BITBUCKET_TOKEN"] != "mytoken" {
		t.Fatalf("expected BITBUCKET_TOKEN to be set")
	}
}

func TestWriteConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "newconfig.json")
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"bitbucket": map[string]interface{}{
				"command": "bitbucket-mcp",
			},
		},
	}

	err := WriteConfig(tmpFile, config)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	mode := info.Mode()
	if (mode & 0077) != 0 {
		t.Fatalf("expected mode 0600, got %o", mode)
	}

	// Verify content is valid JSON
	data, _ := os.ReadFile(tmpFile)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -run TestReadConfig -v`

Expected: FAIL with "undefined: ReadConfig"

- [ ] **Step 3: Implement config file functions**

```go
// internal/install/config.go
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ReadConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("cannot read %s: %v", path, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Error: %s contains invalid JSON: %v", path, err)
	}

	return config, nil
}

func MergeEntry(config map[string]interface{}, url, token string) map[string]interface{} {
	var servers map[string]interface{}

	if s, ok := config["mcpServers"].(map[string]interface{}); ok {
		servers = s
	} else {
		servers = make(map[string]interface{})
		config["mcpServers"] = servers
	}

	servers["bitbucket"] = map[string]interface{}{
		"command": "bitbucket-mcp",
		"env": map[string]interface{}{
			"BITBUCKET_URL":   url,
			"BITBUCKET_TOKEN": token,
		},
	}

	return config
}

func WriteConfig(path string, config map[string]interface{}) error {
	// Create parent directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %v", dir, err)
	}

	// Marshal with 2-space indent
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal JSON: %v", err)
	}

	// Write with 0600 mode
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write %s: %v", path, err)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -run TestReadConfig -v && go test ./internal/install -run TestMergeEntry -v && go test ./internal/install -run TestWriteConfig -v`

Expected: PASS (8 tests total)

- [ ] **Step 5: Commit**

```bash
git add internal/install/config.go internal/install/config_test.go
git commit -m "feat: implement MCP config file handling"
```

---

## Chunk 2: Wizard Orchestration & Main Entry

### Task 4: Implement wizard orchestration

**Files:**
- Create: `internal/install/wizard.go`
- Create: `internal/install/wizard_test.go`

**Context:** Main wizard that orchestrates the full flow — detect agents, prompt user, validate, write config.

- [ ] **Step 1: Write failing test for `Run`**

```go
// internal/install/wizard_test.go
package install

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRun_NoAgents(t *testing.T) {
	// Mock no agents found
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) { return []Agent{}, nil }
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	input := bytes.NewBufferString("")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output.String(), "No supported agents found") {
		t.Fatalf("expected manual template in output, got: %s", output.String())
	}
}

func TestRun_SingleAgent(t *testing.T) {
	// Mock single agent, valid credentials, no write errors
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) {
		return []Agent{{Name: "Claude Code", ConfigPath: "/tmp/test.json"}}, nil
	}
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	oldValidate := validateCredentialsFn
	validateCredentialsFn = func(url, token string) error { return nil }
	t.Cleanup(func() { validateCredentialsFn = oldValidate })

	oldRead := readConfigFn
	readConfigFn = func(path string) (map[string]interface{}, error) {
		return make(map[string]interface{}), nil
	}
	t.Cleanup(func() { readConfigFn = oldRead })

	oldWrite := writeConfigFn
	writeConfigFn = func(path string, config map[string]interface{}) error {
		return nil
	}
	t.Cleanup(func() { writeConfigFn = oldWrite })

	input := strings.NewReader("https://bitbucket.example.com\nmytoken\n")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output.String(), "Saved to") {
		t.Fatalf("expected success message, got: %s", output.String())
	}
}

func TestRun_ValidationFails(t *testing.T) {
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) {
		return []Agent{{Name: "Claude Code", ConfigPath: "/tmp/test.json"}}, nil
	}
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	oldValidate := validateCredentialsFn
	validateCredentialsFn = func(url, token string) error {
		return &validationError{statusCode: 401}
	}
	t.Cleanup(func() { validateCredentialsFn = oldValidate })

	input := strings.NewReader("https://bitbucket.example.com\nbadtoken\n")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(output.String(), "validation failed") {
		t.Fatalf("expected validation error message, got: %s", output.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -run TestRun -v`

Expected: FAIL with "undefined: Run"

- [ ] **Step 3: Implement wizard orchestration**

```go
// internal/install/wizard.go
package install

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// testable hooks
var (
	detectAgentsFn        = DetectAgents
	validateCredentialsFn = ValidateCredentials
	readConfigFn          = ReadConfig
	writeConfigFn         = WriteConfig
)

func Run(in io.Reader, out io.Writer) int {
	printf := func(format string, args ...interface{}) {
		fmt.Fprintf(out, format, args...)
	}

	reader := bufio.NewReader(in)
	readLine := func() (string, error) {
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\n"), err
	}

	// Step 1: Detect agents
	agents, err := detectAgentsFn()
	if err != nil {
		printf("Error: cannot determine home directory: %v\n", err)
		return 1
	}

	// Step 2: No agents found
	if len(agents) == 0 {
		printf("No supported agents found (Claude Code, Cursor).\n\n")
		printf("To configure manually, add the following to your agent's MCP config:\n\n")
		printf("{\n  \"mcpServers\": {\n    \"bitbucket\": {\n      \"command\": \"bitbucket-mcp\",\n      \"env\": {\n        \"BITBUCKET_URL\": \"<YOUR_BITBUCKET_URL>\",\n        \"BITBUCKET_TOKEN\": \"<YOUR_TOKEN>\"\n      }\n    }\n  }\n}\n")
		return 0
	}

	// Step 3: Agent selection
	var selectedAgents []Agent
	if len(agents) == 1 {
		selectedAgents = agents
	} else {
		for {
			for i, agent := range agents {
				printf("[%d] %s (%s)\n", i+1, agent.Name, agent.ConfigPath)
			}
			printf("Install into (1, 2, or \"all\") [all]: ")

			input, err := readLine()
			if err != nil && err != io.EOF {
				printf("Error: %v\n", err)
				return 1
			}
			if err == io.EOF {
				printf("Error: unexpected end of input\n")
				return 1
			}

			input = strings.ToLower(strings.TrimSpace(input))
			if input == "" || input == "all" {
				selectedAgents = agents
				break
			}

			// Try parsing as number
			var idx int
			if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx > 0 && idx <= len(agents) {
				selectedAgents = []Agent{agents[idx-1]}
				break
			}

			printf("Invalid selection. Try again.\n")
		}
	}

	// Step 4: URL prompt
	var url string
	for {
		printf("Bitbucket Server URL: ")
		input, err := readLine()
		if err == io.EOF {
			printf("Error: unexpected end of input\n")
			return 1
		}
		if err != nil {
			printf("Error: %v\n", err)
			return 1
		}

		url = strings.TrimRight(strings.TrimSpace(input), "/")
		if url == "" {
			printf("URL cannot be empty. Try again.\n")
			continue
		}

		// Step 5: URL pre-validation
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			printf("Error: URL must start with http:// or https://\n")
			continue
		}

		break
	}

	// Step 6: Token instructions
	printf("\nTo generate a Personal Access Token:\n")
	printf("  1. Go to %s/plugins/servlet/access-tokens/manage\n", url)
	printf("  2. Click \"Create token\"\n")
	printf("  3. Give it a name, select \"Read\" scope\n")
	printf("  4. Copy the generated token\n\n")

	// Step 7: Token prompt
	var token string
	for {
		printf("Token: ")
		input, err := readLine()
		if err == io.EOF {
			printf("Error: unexpected end of input\n")
			return 1
		}
		if err != nil {
			printf("Error: %v\n", err)
			return 1
		}

		token = strings.TrimSpace(input)
		if token == "" {
			printf("Token cannot be empty. Try again.\n")
			continue
		}
		break
	}

	// Step 8: Validation
	if err := validateCredentialsFn(url, token); err != nil {
		printf("Error: validation failed (HTTP %v)\n", extractStatus(err))
		return 1
	}

	// Step 9: Write config
	var written []string
	for _, agent := range selectedAgents {
		config, err := readConfigFn(agent.ConfigPath)
		if err != nil {
			printf("Error writing %s: %v\nNote: config was already written to: %v\n", agent.ConfigPath, err, written)
			return 1
		}

		config = MergeEntry(config, url, token)

		if err := writeConfigFn(agent.ConfigPath, config); err != nil {
			printf("Error writing %s: %v\nNote: config was already written to: %v\n", agent.ConfigPath, err, written)
			return 1
		}

		written = append(written, agent.ConfigPath)
	}

	// Step 10: Security reminder
	printf("Note: your token is stored in plaintext in the config file.\n\n")

	// Step 11: Done message
	printf("Saved to:\n")
	for _, path := range written {
		printf("  - %s\n", path)
	}
	printf("\nRestart Claude Code / Cursor to activate the bitbucket-mcp server.\n")

	return 0
}

func extractStatus(err error) string {
	return err.Error() // Already formatted as "HTTP NNN" from ValidateCredentials
}

type validationError struct {
	statusCode int
}

func (e *validationError) Error() string {
	return fmt.Sprintf("HTTP %d", e.statusCode)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -run TestRun -v`

Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/install/wizard.go internal/install/wizard_test.go
git commit -m "feat: implement install wizard orchestration"
```

---

### Task 5: Add `--install` flag to main entry point

**Files:**
- Modify: `cmd/bitbucket-mcp/main.go:1-25` (add flag check and import)

**Context:** Wire the wizard into the binary so users can run `bitbucket-mcp --install`.

- [ ] **Step 1: Write test for main flag check**

Create a simple integration test:

```go
// cmd/bitbucket-mcp/main_test.go (if needed for verification)
// For now, manual verification: bitbucket-mcp --install should trigger wizard
```

- [ ] **Step 2: Update main.go**

Read the current main.go:

```go
// cmd/bitbucket-mcp/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/ilyalaletin/bitbucket-mcp/internal/install"
	"github.com/ilyalaletin/bitbucket-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Check for --install or -i flag
	if len(os.Args) > 1 && (os.Args[1] == "--install" || os.Args[1] == "-i") {
		os.Exit(install.Run(os.Stdin, os.Stdout))
	}

	// Read environment variables
	bitbucketURL := os.Getenv("BITBUCKET_URL")
	bitbucketToken := os.Getenv("BITBUCKET_TOKEN")

	if bitbucketURL == "" || bitbucketToken == "" {
		fmt.Fprintf(os.Stderr, "Error: BITBUCKET_URL and BITBUCKET_TOKEN environment variables are required\n")
		os.Exit(1)
	}

	// ... rest of main.go (unchanged)
}
```

- [ ] **Step 3: Verify build works**

Run: `go build ./cmd/bitbucket-mcp/`

Expected: Binary builds without errors

- [ ] **Step 4: Test the flag manually**

Run: `./bitbucket-mcp --install`

Expected: Interactive wizard prompts for URL and token (can Ctrl+C to exit)

Run: `./bitbucket-mcp` (without flag)

Expected: Error about missing env vars (correct behavior)

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`

Expected: All tests pass (including 4 test files from Tasks 1-4)

- [ ] **Step 6: Commit**

```bash
git add cmd/bitbucket-mcp/main.go
git commit -m "feat: add --install flag to main entry point"
```

---

## Summary

**Total tasks:** 5
**Total test files:** 4 (`detect_test.go`, `validate_test.go`, `config_test.go`, `wizard_test.go`)
**Total implementation files:** 4 (`detect.go`, `validate.go`, `config.go`, `wizard.go`)
**Modified files:** 1 (`main.go`)

**Exit criteria:**
- All tests pass (`go test ./...`)
- Build succeeds (`go build ./cmd/bitbucket-mcp/`)
- Manual verification: `bitbucket-mcp --install` runs wizard interactively
- Manual verification: `bitbucket-mcp` (no flag) fails as before on missing env vars
