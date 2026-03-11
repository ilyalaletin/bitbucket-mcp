# Bitbucket MCP Server Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an MCP server that provides access to Bitbucket Server/DC pull requests, build statuses, and repository content.

**Architecture:** Flat structure — MCP tool handlers call the Bitbucket HTTP client directly. stdio transport via mcp-go. Config via env vars.

**Tech Stack:** Go, mcp-go, net/http, encoding/json

**Spec:** `docs/superpowers/specs/2026-03-11-bitbucket-mcp-design.md`

---

## Chunk 1: Project Bootstrap & HTTP Client

### Task 1: Initialize Go module and dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Init Go module**

```bash
cd /Users/ilya/dev/bitbucket-mcp
go mod init github.com/ilyalaletin/bitbucket-mcp
```

- [ ] **Step 2: Add mcp-go dependency**

```bash
go get github.com/mark3labs/mcp-go
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialize Go module with mcp-go dependency"
```

---

### Task 2: Bitbucket Client — core HTTP with retry

**Files:**
- Create: `internal/bitbucket/client.go`
- Test: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for NewClient**

```go
// internal/bitbucket/client_test.go
package bitbucket

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://bitbucket.example.com", "test-token")

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "https://bitbucket.example.com" {
		t.Errorf("expected baseURL 'https://bitbucket.example.com', got %q", client.baseURL)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	client := NewClient("https://bitbucket.example.com/", "test-token")

	if client.baseURL != "https://bitbucket.example.com" {
		t.Errorf("expected trailing slash trimmed, got %q", client.baseURL)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestNewClient -v
```

Expected: FAIL — `NewClient` not defined.

- [ ] **Step 3: Implement NewClient**

```go
// internal/bitbucket/client.go
package bitbucket

import (
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/bitbucket/ -run TestNewClient -v
```

Expected: PASS

- [ ] **Step 5: Write failing test for do() with retry**

```go
// internal/bitbucket/client_test.go (append)

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer auth header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_Retry_On5xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDo_NoRetry_On4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDo_Retry_On429(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

```bash
go test ./internal/bitbucket/ -run TestDo -v
```

Expected: FAIL — `do` method not defined.

- [ ] **Step 7: Implement do() with retry**

```go
// internal/bitbucket/client.go (add to existing file)

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

func (c *Client) do(ctx context.Context, method, path string, query url.Values) (*http.Response, error) {
	fullURL := c.baseURL + path
	if query != nil {
		fullURL += "?" + query.Encode()
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastResp = resp
			continue
		}

		return resp, nil
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
go test ./internal/bitbucket/ -v
```

Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add Bitbucket HTTP client with retry logic"
```

---

### Task 3: Bitbucket Client — API error handling and JSON decoding

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for getJSON helper**

```go
// internal/bitbucket/client_test.go (append — add "errors" to imports)

func TestGetJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name": "test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var result struct {
		Name string `json:"name"`
	}
	err := client.getJSON(context.Background(), "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected name 'test', got %q", result.Name)
	}
}

func TestGetJSON_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors": [{"message": "not found"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var result map[string]any
	err := client.getJSON(context.Background(), "/test", nil, &result)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetJSON -v
```

Expected: FAIL

- [ ] **Step 3: Implement APIError and getJSON**

```go
// internal/bitbucket/client.go (add)

import "encoding/json"

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bitbucket API error (status %d): %s", e.StatusCode, e.Body)
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, result any) error {
	resp, err := c.do(ctx, "GET", path, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bitbucket/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add API error handling and JSON decoding"
```

---

### Task 3b: Bitbucket Client — getRaw for diff endpoints

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for getRaw**

```go
// internal/bitbucket/client_test.go (append)

func TestGetRaw_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("raw content here"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	data, err := client.getRaw(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "raw content here" {
		t.Errorf("expected 'raw content here', got %q", string(data))
	}
}

func TestGetRaw_Truncation(t *testing.T) {
	// Create content larger than 1MB
	bigContent := make([]byte, 1024*1024+100)
	for i := range bigContent {
		bigContent[i] = 'x'
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bigContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	data, err := client.getRaw(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) > 1024*1024+200 {
		t.Errorf("expected truncated content, got %d bytes", len(data))
	}
	if !strings.Contains(string(data), "[truncated") {
		t.Error("expected truncation notice")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetRaw -v
```

Expected: FAIL — `getRaw` not defined.

- [ ] **Step 3: Implement getRaw with 1MB truncation**

```go
// internal/bitbucket/client.go (add)

const maxDiffSize = 1024 * 1024 // 1MB

func (c *Client) getRaw(ctx context.Context, path string, query url.Values) ([]byte, error) {
	resp, err := c.do(ctx, "GET", path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDiffSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	if len(body) > maxDiffSize {
		body = body[:maxDiffSize]
		body = append(body, []byte("\n\n[truncated — diff exceeds 1MB]")...)
	}
	return body, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bitbucket/ -run TestGetRaw -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add getRaw method with 1MB diff truncation"
```

---

### Task 4: Bitbucket types

**Files:**
- Create: `internal/bitbucket/types.go`

- [ ] **Step 1: Create types file**

```go
// internal/bitbucket/types.go
package bitbucket

// PagedResponse is the generic wrapper for paginated Bitbucket responses.
type PagedResponse[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

type PullRequest struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedDate int64      `json:"createdDate"`
	UpdatedDate int64      `json:"updatedDate"`
	Author      PRUser     `json:"author"`
	Reviewers   []PRUser   `json:"reviewers"`
	FromRef     PRRef      `json:"fromRef"`
	ToRef       PRRef      `json:"toRef"`
	Links       Links      `json:"links"`
}

type PRUser struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	Status   string `json:"status"`
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
}

type PRRef struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

type Repository struct {
	Slug    string  `json:"slug"`
	Project Project `json:"project"`
}

type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Links struct {
	Self []Link `json:"self"`
}

type Link struct {
	Href string `json:"href"`
}

type Commit struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"`
	Message   string `json:"message"`
	Author    CommitAuthor `json:"author"`
	AuthorTimestamp int64 `json:"authorTimestamp"`
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type BuildStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	DateAdded   int64  `json:"dateAdded"`
}

type BrowseResponse struct {
	Path     BrowsePath      `json:"path"`
	Children *PagedResponse[BrowseEntry] `json:"children,omitempty"`
	Lines    []BrowseLine    `json:"lines,omitempty"`
	Binary   bool            `json:"binary,omitempty"`
}

type BrowsePath struct {
	Components []string `json:"components"`
	Name       string   `json:"name"`
}

type BrowseEntry struct {
	Path      BrowsePath `json:"path"`
	ContentID string     `json:"contentId"`
	Type      string     `json:"type"` // "FILE" or "DIRECTORY"
	Size      int64      `json:"size,omitempty"`
}

type BrowseLine struct {
	Text string `json:"text"`
}

// Note: Diff endpoints return raw text (not JSON), handled by getRaw with 1MB truncation.
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/bitbucket/
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/bitbucket/types.go
git commit -m "feat: add Bitbucket API response types"
```

---

## Chunk 2: Bitbucket Client API Methods

### Task 5: PR client methods

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for ListPRs (repo-scoped)**

```go
// internal/bitbucket/client_test.go (append)

func TestListPRs_RepoScoped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/pull-requests" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "OPEN" {
			t.Errorf("expected state=OPEN, got %s", r.URL.Query().Get("state"))
		}
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"id":1,"title":"Test PR","state":"OPEN"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	prs, err := client.ListPRs(context.Background(), ListPRsOptions{
		Project: "PRJ",
		Repo:    "my-repo",
		State:   "OPEN",
		Limit:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", prs[0].Title)
	}
}

func TestListPRs_Dashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/dashboard/pull-requests" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("role") != "REVIEWER" {
			t.Errorf("expected role=REVIEWER")
		}
		w.Write([]byte(`{"size":0,"limit":25,"isLastPage":true,"start":0,"values":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	prs, err := client.ListPRs(context.Background(), ListPRsOptions{
		Role:  "REVIEWER",
		State: "OPEN",
		Limit: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(prs))
	}
}

func TestListPRs_ValidationError(t *testing.T) {
	client := NewClient("http://localhost", "token")
	_, err := client.ListPRs(context.Background(), ListPRsOptions{
		State: "OPEN",
		Limit: 25,
	})
	if err == nil {
		t.Fatal("expected validation error when no role and no project/repo")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestListPRs -v
```

Expected: FAIL

- [ ] **Step 3: Implement ListPRs**

```go
// internal/bitbucket/client.go (add)

type ListPRsOptions struct {
	Project string
	Repo    string
	State   string
	Role    string
	Limit   int
}

func (c *Client) ListPRs(ctx context.Context, opts ListPRsOptions) ([]PullRequest, error) {
	if opts.Limit <= 0 {
		opts.Limit = 25
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	query := url.Values{}
	if opts.State != "" {
		query.Set("state", opts.State)
	}

	var path string
	if opts.Role != "" {
		path = "/rest/api/1.0/dashboard/pull-requests"
		query.Set("role", opts.Role)
	} else {
		if opts.Project == "" || opts.Repo == "" {
			return nil, fmt.Errorf("project and repo are required when role is not specified")
		}
		path = fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", opts.Project, opts.Repo)
	}

	return paginate[PullRequest](ctx, c, path, query, opts.Limit)
}

// paginate fetches all pages up to the specified limit.
func paginate[T any](ctx context.Context, c *Client, path string, query url.Values, limit int) ([]T, error) {
	var all []T
	start := 0
	for {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("start", fmt.Sprintf("%d", start))
		remaining := limit - len(all)
		q.Set("limit", fmt.Sprintf("%d", remaining))

		var page PagedResponse[T]
		if err := c.getJSON(ctx, path, q, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Values...)
		if page.IsLastPage || len(all) >= limit {
			break
		}
		start = page.NextPageStart
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bitbucket/ -run TestListPRs -v
```

Expected: all PASS

- [ ] **Step 5: Write failing test for GetPR**

```go
// internal/bitbucket/client_test.go (append)
func TestGetPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/pull-requests/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":42,"title":"My PR","state":"OPEN","description":"desc"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	pr, err := client.GetPR(context.Background(), "PRJ", "my-repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.ID != 42 {
		t.Errorf("expected PR ID 42, got %d", pr.ID)
	}
	if pr.Description != "desc" {
		t.Errorf("expected description 'desc', got %q", pr.Description)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetPR -v
```

Expected: FAIL — `GetPR` not defined.

- [ ] **Step 7: Implement GetPR**

```go
func (c *Client) GetPR(ctx context.Context, project, repo string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", project, repo, prID)
	var pr PullRequest
	if err := c.getJSON(ctx, path, nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
```

- [ ] **Step 8: Run test to verify it passes**

```bash
go test ./internal/bitbucket/ -run TestGetPR -v
```

Expected: PASS

- [ ] **Step 9: Write failing test for GetPRDiff**

```go
func TestGetPRDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/pull-requests/1/diff" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte("diff --git a/file.go b/file.go\n+added line\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	diff, err := client.GetPRDiff(context.Background(), "PRJ", "my-repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "diff --git") {
		t.Errorf("expected diff content, got %q", diff)
	}
}
```

- [ ] **Step 10: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetPRDiff -v
```

Expected: FAIL — `GetPRDiff` not defined.

- [ ] **Step 11: Implement GetPRDiff**

```go
func (c *Client) GetPRDiff(ctx context.Context, project, repo string, prID int) (string, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff", project, repo, prID)
	data, err := c.getRaw(ctx, path, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 12: Run test to verify it passes**

```bash
go test ./internal/bitbucket/ -run TestGetPRDiff -v
```

Expected: PASS

- [ ] **Step 13: Write failing test for GetPRCommits**

```go
func TestGetPRCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/pull-requests/1/commits" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"id":"abc123","displayId":"abc","message":"fix bug"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	commits, err := client.GetPRCommits(context.Background(), "PRJ", "my-repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].ID != "abc123" {
		t.Errorf("expected commit ID 'abc123', got %q", commits[0].ID)
	}
}
```

- [ ] **Step 14: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetPRCommits -v
```

Expected: FAIL — `GetPRCommits` not defined.

- [ ] **Step 15: Implement GetPRCommits**

```go
func (c *Client) GetPRCommits(ctx context.Context, project, repo string, prID int) ([]Commit, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/commits", project, repo, prID)
	return paginate[Commit](ctx, c, path, nil, 100)
}
```

- [ ] **Step 16: Run all tests**

```bash
go test ./internal/bitbucket/ -v
```

Expected: all PASS

- [ ] **Step 17: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add PR client methods (list, get, diff, commits)"
```

---

### Task 6: Build status client method

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for GetBuildStatus**

```go
func TestGetBuildStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/build-status/1.0/commits/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"state":"SUCCESSFUL","key":"build-1","name":"CI","url":"https://ci.example.com"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	statuses, err := client.GetBuildStatus(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "SUCCESSFUL" {
		t.Errorf("expected state SUCCESSFUL, got %q", statuses[0].State)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bitbucket/ -run TestGetBuildStatus -v
```

- [ ] **Step 3: Implement GetBuildStatus**

```go
func (c *Client) GetBuildStatus(ctx context.Context, commitID string) ([]BuildStatus, error) {
	path := fmt.Sprintf("/rest/build-status/1.0/commits/%s", commitID)
	return paginate[BuildStatus](ctx, c, path, nil, 100)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/bitbucket/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add build status client method"
```

---

### Task 7: Browse and diff client methods

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [ ] **Step 1: Write failing test for ListFiles**

```go
func TestListFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/browse/src" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("at") != "main" {
			t.Errorf("expected at=main")
		}
		w.Write([]byte(`{"path":{"name":"src"},"children":{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"path":{"name":"main.go"},"type":"FILE","size":123}]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PRJ", "my-repo", "src", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Children == nil {
		t.Fatal("expected children for directory listing")
	}
	if len(resp.Children.Values) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Children.Values))
	}
	if resp.Children.Values[0].Path.Name != "main.go" {
		t.Errorf("expected file 'main.go', got %q", resp.Children.Values[0].Path.Name)
	}
}

func TestGetFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/browse/main.go" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"path":{"name":"main.go"},"lines":[{"text":"package main"},{"text":""}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PRJ", "my-repo", "main.go", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(resp.Lines))
	}
	if resp.Lines[0].Text != "package main" {
		t.Errorf("expected 'package main', got %q", resp.Lines[0].Text)
	}
}

func TestGetFileContent_Binary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"path":{"name":"image.png"},"binary":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PRJ", "my-repo", "image.png", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Binary {
		t.Error("expected binary=true")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/bitbucket/ -run "TestListFiles|TestGetFileContent" -v
```

- [ ] **Step 3: Implement Browse**

```go
func (c *Client) Browse(ctx context.Context, project, repo, path, ref string) (*BrowseResponse, error) {
	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/browse/%s", project, repo, path)
	query := url.Values{}
	if ref != "" {
		query.Set("at", ref)
	}

	var resp BrowseResponse
	if err := c.getJSON(ctx, apiPath, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

- [ ] **Step 4: Write failing test for GetDiff (compare)**

```go
func TestGetDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PRJ/repos/my-repo/compare/diff" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("from") != "feature" {
			t.Errorf("expected from=feature")
		}
		if r.URL.Query().Get("to") != "main" {
			t.Errorf("expected to=main")
		}
		w.Write([]byte("diff --git a/file.go b/file.go\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	diff, err := client.GetDiff(context.Background(), "PRJ", "my-repo", "feature", "main", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "diff --git") {
		t.Errorf("expected diff content, got %q", diff)
	}
}

func TestGetDiff_WithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("path") != "src/main.go" {
			t.Errorf("expected path=src/main.go, got %s", r.URL.Query().Get("path"))
		}
		w.Write([]byte("diff content"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.GetDiff(context.Background(), "PRJ", "my-repo", "feature", "main", "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 5: Implement GetDiff**

```go
func (c *Client) GetDiff(ctx context.Context, project, repo, from, to, path string) (string, error) {
	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/compare/diff", project, repo)
	query := url.Values{}
	query.Set("from", from)
	query.Set("to", to)
	if path != "" {
		query.Set("path", path)
	}

	data, err := c.getRaw(ctx, apiPath, query)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/bitbucket/ -v
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/bitbucket/
git commit -m "feat: add browse and diff client methods"
```

---

## Chunk 3: MCP Tool Handlers

### Task 8: PR tool handlers

**Files:**
- Create: `internal/tools/pr.go`
- Test: `internal/tools/pr_test.go`

- [ ] **Step 1: Write failing test for list_prs tool handler**

```go
// internal/tools/pr_test.go
package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestListPRsTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"id":1,"title":"Test PR","state":"OPEN"}]}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewPRTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"state":   "OPEN",
	}

	result, err := handler.ListPRs(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}
	// Verify response contains PR data
	text := result.Content[0].(mcp.TextContent).Text
	if text == "" {
		t.Error("expected non-empty response")
	}
	var prs []bitbucket.PullRequest
	if err := json.Unmarshal([]byte(text), &prs); err != nil {
		t.Fatalf("expected valid JSON array: %v", err)
	}
	if len(prs) != 1 {
		t.Errorf("expected 1 PR, got %d", len(prs))
	}
}

func TestListPRsTool_ValidationError(t *testing.T) {
	client := bitbucket.NewClient("http://localhost", "token")
	handler := NewPRTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"state": "OPEN",
	}

	result, err := handler.ListPRs(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing project/repo without role")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tools/ -run TestListPRsTool -v
```

- [ ] **Step 3: Implement PR tool handlers**

```go
// internal/tools/pr.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

type PRTools struct {
	client *bitbucket.Client
}

func NewPRTools(client *bitbucket.Client) *PRTools {
	return &PRTools{client: client}
}

func (t *PRTools) ListPRs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	opts := bitbucket.ListPRsOptions{
		Project: stringArg(args, "project"),
		Repo:    stringArg(args, "repo"),
		State:   stringArg(args, "state"),
		Role:    stringArg(args, "role"),
		Limit:   intArg(args, "limit"),
	}
	if opts.State == "" {
		opts.State = "OPEN"
	}

	prs, err := t.client.ListPRs(ctx, opts)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(prs)
}

func (t *PRTools) GetPR(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	pr, err := t.client.GetPR(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		intArg(args, "pr_id"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(pr)
}

func (t *PRTools) GetPRDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	diff, err := t.client.GetPRDiff(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		intArg(args, "pr_id"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(diff), nil
}

func (t *PRTools) GetPRCommits(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	commits, err := t.client.GetPRCommits(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		intArg(args, "pr_id"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(commits)
}

// helpers

func stringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func intArg(args map[string]any, key string) int {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: text},
		},
	}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: err.Error(),
			},
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tools/ -run TestListPRsTool -v
```

Expected: PASS

- [ ] **Step 5: Write and run tests for GetPR, GetPRDiff, GetPRCommits handlers**

```go
// internal/tools/pr_test.go (append)

func TestGetPRTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":42,"title":"My PR","state":"OPEN"}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewPRTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"pr_id":   float64(42),
	}

	result, err := handler.GetPR(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := result.Content[0].(mcp.TextContent).Text
	var pr bitbucket.PullRequest
	if err := json.Unmarshal([]byte(text), &pr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if pr.ID != 42 {
		t.Errorf("expected PR ID 42, got %d", pr.ID)
	}
}
```

- [ ] **Step 6: Run all tool tests**

```bash
go test ./internal/tools/ -v
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/tools/
git commit -m "feat: add PR MCP tool handlers"
```

---

### Task 9: Build status tool handler

**Files:**
- Create: `internal/tools/builds.go`
- Test: `internal/tools/builds_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/tools/builds_test.go
package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestGetBuildStatusTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"state":"SUCCESSFUL","key":"ci","name":"CI Build","url":"https://ci.example.com"}]}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewBuildTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"commit_id": "abc123def",
	}

	result, err := handler.GetBuildStatus(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := result.Content[0].(mcp.TextContent).Text
	var statuses []bitbucket.BuildStatus
	if err := json.Unmarshal([]byte(text), &statuses); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tools/ -run TestGetBuildStatusTool -v
```

- [ ] **Step 3: Implement**

```go
// internal/tools/builds.go
package tools

import (
	"context"
	"fmt"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

type BuildTools struct {
	client *bitbucket.Client
}

func NewBuildTools(client *bitbucket.Client) *BuildTools {
	return &BuildTools{client: client}
}

func (t *BuildTools) GetBuildStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	commitID := stringArg(req.Params.Arguments, "commit_id")
	if commitID == "" {
		return errorResult(fmt.Errorf("commit_id is required")), nil
	}

	statuses, err := t.client.GetBuildStatus(ctx, commitID)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(statuses)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/tools/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat: add build status MCP tool handler"
```

---

### Task 10: Browse and diff tool handlers

**Files:**
- Create: `internal/tools/repos.go`
- Test: `internal/tools/repos_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/tools/repos_test.go
package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestListFilesTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"path":{"name":"src"},"children":{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"path":{"name":"main.go"},"type":"FILE"}]}}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewRepoTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"path":    "src",
	}

	result, err := handler.ListFiles(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestGetFileContentTool_Binary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"path":{"name":"image.png"},"binary":true}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewRepoTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"path":    "image.png",
	}

	result, err := handler.GetFileContent(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Binary file: image.png" {
		t.Errorf("expected binary file message, got %q", text)
	}
}

func TestGetDiffTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"diffs":[{"hunks":[]}]}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewRepoTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"from":    "feature",
		"to":      "main",
	}

	result, err := handler.GetDiff(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tools/ -run "TestListFilesTool|TestGetFileContentTool|TestGetDiffTool" -v
```

- [ ] **Step 3: Implement repo tool handlers**

```go
// internal/tools/repos.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/mark3labs/mcp-go/mcp"
)

type RepoTools struct {
	client *bitbucket.Client
}

func NewRepoTools(client *bitbucket.Client) *RepoTools {
	return &RepoTools{client: client}
}

func (t *RepoTools) ListFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	resp, err := t.client.Browse(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		stringArg(args, "path"),
		stringArg(args, "ref"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	if resp.Children == nil {
		return errorResult(fmt.Errorf("path is not a directory")), nil
	}
	return jsonResult(resp.Children.Values)
}

func (t *RepoTools) GetFileContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := stringArg(args, "path")
	resp, err := t.client.Browse(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		path,
		stringArg(args, "ref"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	if resp.Binary {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: fmt.Sprintf("Binary file: %s", resp.Path.Name)},
			},
		}, nil
	}
	var sb strings.Builder
	for i, line := range resp.Lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line.Text)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: sb.String()},
		},
	}, nil
}

func (t *RepoTools) GetDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	diff, err := t.client.GetDiff(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		stringArg(args, "from"),
		stringArg(args, "to"),
		stringArg(args, "path"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(diff), nil
}
```

- [ ] **Step 4: Run all tool tests**

```bash
go test ./internal/tools/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat: add browse and diff MCP tool handlers"
```

---

## Chunk 4: Entry Point & Integration

### Task 11: MCP server entry point

**Files:**
- Create: `cmd/bitbucket-mcp/main.go`

- [ ] **Step 1: Implement main.go**

```go
// cmd/bitbucket-mcp/main.go
package main

import (
	"fmt"
	"os"

	"github.com/ilyalaletin/bitbucket-mcp/internal/bitbucket"
	"github.com/ilyalaletin/bitbucket-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("BITBUCKET_URL")
	token := os.Getenv("BITBUCKET_TOKEN")

	if baseURL == "" || token == "" {
		fmt.Fprintln(os.Stderr, "BITBUCKET_URL and BITBUCKET_TOKEN environment variables are required")
		os.Exit(1)
	}

	client := bitbucket.NewClient(baseURL, token)

	prTools := tools.NewPRTools(client)
	buildTools := tools.NewBuildTools(client)
	repoTools := tools.NewRepoTools(client)

	s := server.NewMCPServer(
		"bitbucket-mcp",
		"0.1.0",
	)

	// PR tools
	s.AddTool(mcp.NewTool("list_prs",
		mcp.WithDescription("List pull requests. When role is set, uses dashboard (project/repo ignored). Otherwise project and repo are required."),
		mcp.WithString("project", mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Description("Repository slug")),
		mcp.WithString("state", mcp.Description("PR state: OPEN, MERGED, DECLINED, ALL"), mcp.DefaultString("OPEN")),
		mcp.WithString("role", mcp.Description("Filter by role: AUTHOR or REVIEWER")),
		mcp.WithNumber("limit", mcp.Description("Max results (1-100, default 25)")),
	), prTools.ListPRs)

	s.AddTool(mcp.NewTool("get_pr",
		mcp.WithDescription("Get pull request details"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), prTools.GetPR)

	s.AddTool(mcp.NewTool("get_pr_diff",
		mcp.WithDescription("Get full diff of a pull request"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), prTools.GetPRDiff)

	s.AddTool(mcp.NewTool("get_pr_commits",
		mcp.WithDescription("List commits in a pull request"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), prTools.GetPRCommits)

	// Build tools
	s.AddTool(mcp.NewTool("get_build_status",
		mcp.WithDescription("Get build statuses for a commit"),
		mcp.WithString("commit_id", mcp.Required(), mcp.Description("Commit hash")),
	), buildTools.GetBuildStatus)

	// Repo tools
	s.AddTool(mcp.NewTool("list_files",
		mcp.WithDescription("List files and directories in a repository"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("path", mcp.Description("Path within repo (optional, defaults to root)")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit (optional, defaults to default branch)")),
	), repoTools.ListFiles)

	s.AddTool(mcp.NewTool("get_file_content",
		mcp.WithDescription("Read file content from a repository. Returns text for text files, a message for binary files."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit (optional, defaults to default branch)")),
	), repoTools.GetFileContent)

	s.AddTool(mcp.NewTool("get_diff",
		mcp.WithDescription("Get diff between two branches, tags, or commits"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Bitbucket project key")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Source ref (branch, tag, or commit)")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Target ref (branch, tag, or commit)")),
		mcp.WithString("path", mcp.Description("Scope diff to a specific file path")),
	), repoTools.GetDiff)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/bitbucket-mcp/
```

Expected: binary built successfully

- [ ] **Step 3: Commit**

```bash
git add cmd/bitbucket-mcp/
git commit -m "feat: add MCP server entry point with all 8 tools"
```

---

### Task 12: Build verification and cleanup

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS

- [ ] **Step 2: Run vet**

```bash
go vet ./...
```

Expected: no issues

- [ ] **Step 3: Build binary**

```bash
go build -o bitbucket-mcp ./cmd/bitbucket-mcp/
```

Expected: binary created

- [ ] **Step 4: Add binary to .gitignore**

Add `bitbucket-mcp` to `.gitignore`.

- [ ] **Step 5: Final commit**

```bash
git add .gitignore
git commit -m "chore: add binary to gitignore"
```
