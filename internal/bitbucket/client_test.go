package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

// Task 5 Tests: PR client methods

func TestListPRs_RepoScoped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/repo/pull-requests") {
			t.Errorf("expected repo-scoped path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"size": 1,
			"limit": 25,
			"isLastPage": true,
			"start": 0,
			"values": [
				{
					"id": 1,
					"title": "Test PR",
					"state": "OPEN",
					"author": {"user": {"name": "user1", "displayName": "User 1"}},
					"reviewers": [],
					"fromRef": {"id": "refs/heads/feature", "repository": {"slug": "repo", "project": {"key": "PROJ"}}},
					"toRef": {"id": "refs/heads/main", "repository": {"slug": "repo", "project": {"key": "PROJ"}}}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	prs, err := client.ListPRs(context.Background(), ListPRsOptions{
		Project: "PROJ",
		Repo:    "repo",
		State:   "OPEN",
		Limit:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Errorf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", prs[0].Title)
	}
}

func TestListPRs_Dashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/dashboard/pull-requests") {
			t.Errorf("expected dashboard path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"size": 2,
			"limit": 25,
			"isLastPage": true,
			"start": 0,
			"values": [
				{
					"id": 1,
					"title": "PR 1",
					"state": "OPEN",
					"author": {"user": {"name": "user1", "displayName": "User 1"}},
					"reviewers": [],
					"fromRef": {"id": "refs/heads/feature1", "repository": {"slug": "repo1", "project": {"key": "PROJ1"}}},
					"toRef": {"id": "refs/heads/main", "repository": {"slug": "repo1", "project": {"key": "PROJ1"}}}
				},
				{
					"id": 2,
					"title": "PR 2",
					"state": "OPEN",
					"author": {"user": {"name": "user2", "displayName": "User 2"}},
					"reviewers": [],
					"fromRef": {"id": "refs/heads/feature2", "repository": {"slug": "repo2", "project": {"key": "PROJ2"}}},
					"toRef": {"id": "refs/heads/main", "repository": {"slug": "repo2", "project": {"key": "PROJ2"}}}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	prs, err := client.ListPRs(context.Background(), ListPRsOptions{
		State: "OPEN",
		Role:  "AUTHOR",
		Limit: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListPRs_ValidationError(t *testing.T) {
	client := NewClient("https://example.com", "token")
	_, err := client.ListPRs(context.Background(), ListPRsOptions{
		State: "OPEN",
		// Missing project/repo and role
	})
	if err == nil {
		t.Fatal("expected validation error for missing project/repo and role")
	}
	if !strings.Contains(err.Error(), "project and repo required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/repo/pull-requests/1") {
			t.Errorf("expected PR detail path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 1,
			"title": "Test PR",
			"description": "This is a test PR",
			"state": "OPEN",
			"createdDate": 1234567890,
			"updatedDate": 1234567899,
			"author": {"user": {"name": "user1", "displayName": "User 1", "emailAddress": "user1@example.com"}, "role": "AUTHOR"},
			"reviewers": [{"user": {"name": "user2", "displayName": "User 2"}, "role": "REVIEWER", "approved": false}],
			"fromRef": {"id": "refs/heads/feature", "displayId": "feature", "latestCommit": "abc123", "repository": {"slug": "repo", "project": {"key": "PROJ"}}},
			"toRef": {"id": "refs/heads/main", "displayId": "main", "latestCommit": "def456", "repository": {"slug": "repo", "project": {"key": "PROJ"}}}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	pr, err := client.GetPR(context.Background(), "PROJ", "repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", pr.Title)
	}
	if pr.Description != "This is a test PR" {
		t.Errorf("expected description 'This is a test PR', got %q", pr.Description)
	}
}

func TestGetPRDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/repo/pull-requests/1/diff") {
			t.Errorf("expected PR diff path, got %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "text/plain" {
			t.Errorf("expected Accept: text/plain, got %s", r.Header.Get("Accept"))
		}
		w.Write([]byte("--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	diff, err := client.GetPRDiff(context.Background(), "PROJ", "repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "--- a/file.txt") {
		t.Errorf("expected diff content, got %q", diff)
	}
}

func TestGetPRCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/repo/pull-requests/1/commits") {
			t.Errorf("expected PR commits path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"size": 1,
			"limit": 25,
			"isLastPage": true,
			"start": 0,
			"values": [
				{
					"id": "abc123def456",
					"displayId": "abc123d",
					"message": "Fix bug",
					"author": {"name": "user1", "emailAddress": "user1@example.com"},
					"authorTimestamp": 1234567890
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	commits, err := client.GetPRCommits(context.Background(), "PROJ", "repo", 1, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != "Fix bug" {
		t.Errorf("expected message 'Fix bug', got %q", commits[0].Message)
	}
}

// Task 6 Tests: Build status

func TestGetBuildStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/build-status/1.0/commits/abc123def456") {
			t.Errorf("expected build status path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"size": 2,
			"limit": 25,
			"isLastPage": true,
			"start": 0,
			"values": [
				{
					"state": "SUCCESSFUL",
					"key": "build-1",
					"name": "Build 1",
					"url": "https://ci.example.com/build/1",
					"description": "Build completed successfully",
					"dateAdded": 1234567890
				},
				{
					"state": "INPROGRESS",
					"key": "test-1",
					"name": "Test 1",
					"url": "https://ci.example.com/test/1",
					"description": "Tests running",
					"dateAdded": 1234567891
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	statuses, err := client.GetBuildStatus(context.Background(), "abc123def456", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses[0].State != "SUCCESSFUL" {
		t.Errorf("expected state 'SUCCESSFUL', got %q", statuses[0].State)
	}
	if statuses[1].Name != "Test 1" {
		t.Errorf("expected name 'Test 1', got %q", statuses[1].Name)
	}
}

// Task 7 Tests: Browse and Diff

func TestListFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects/PROJ/repos/repo/browse/src") {
			t.Errorf("expected browse path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"path": {
				"components": ["src"],
				"name": "src"
			},
			"children": {
				"size": 2,
				"limit": 25,
				"isLastPage": true,
				"start": 0,
				"values": [
					{
						"path": {"components": ["src", "main.go"], "name": "main.go"},
						"contentId": "abc123",
						"type": "FILE",
						"size": 1234
					},
					{
						"path": {"components": ["src", "util"], "name": "util"},
						"contentId": "def456",
						"type": "DIRECTORY"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PROJ", "repo", "src", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Children == nil {
		t.Fatal("expected children in response")
	}
	if len(resp.Children.Values) != 2 {
		t.Errorf("expected 2 entries, got %d", len(resp.Children.Values))
	}
}

func TestGetFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects/PROJ/repos/repo/browse/main.go") {
			t.Errorf("expected browse path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"path": {
				"components": ["main.go"],
				"name": "main.go"
			},
			"lines": [
				{"text": "package main"},
				{"text": ""},
				{"text": "func main() {"},
				{"text": "  fmt.Println(\"Hello\")"},
				{"text": "}"}
			],
			"binary": false
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PROJ", "repo", "main.go", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Lines == nil {
		t.Fatal("expected lines in response")
	}
	if len(resp.Lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(resp.Lines))
	}
	if resp.Lines[0].Text != "package main" {
		t.Errorf("expected 'package main', got %q", resp.Lines[0].Text)
	}
}

func TestGetFileContent_Binary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"path": {
				"components": ["image.png"],
				"name": "image.png"
			},
			"binary": true
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.Browse(context.Background(), "PROJ", "repo", "image.png", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Binary {
		t.Error("expected binary flag set to true")
	}
}

func TestGetDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects/PROJ/repos/repo/compare/diff") {
			t.Errorf("expected compare diff path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("from") != "main" {
			t.Errorf("expected from=main, got %s", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "feature" {
			t.Errorf("expected to=feature, got %s", r.URL.Query().Get("to"))
		}
		if r.Header.Get("Accept") != "text/plain" {
			t.Errorf("expected Accept: text/plain, got %s", r.Header.Get("Accept"))
		}
		w.Write([]byte("--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	diff, err := client.GetDiff(context.Background(), "PROJ", "repo", "main", "feature", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "--- a/file.txt") {
		t.Errorf("expected diff content, got %q", diff)
	}
}

func TestGetDiff_WithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("path") != "src/main.go" {
			t.Errorf("expected path=src/main.go, got %s", r.URL.Query().Get("path"))
		}
		w.Write([]byte("--- a/src/main.go\n+++ b/src/main.go\n@@ -1 +1 @@\n-old\n+new\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	diff, err := client.GetDiff(context.Background(), "PROJ", "repo", "main", "feature", "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "src/main.go") {
		t.Errorf("expected file path in diff, got %q", diff)
	}
}
