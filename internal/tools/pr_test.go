package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetPRDiffTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("diff --git a/file.go b/file.go\n+new line\n"))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewPRTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"pr_id":   float64(1),
	}

	result, err := handler.GetPRDiff(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "diff --git") {
		t.Errorf("expected diff content, got %q", text)
	}
}

func TestGetPRCommitsTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"size":1,"limit":25,"isLastPage":true,"start":0,"values":[{"id":"abc123","displayId":"abc","message":"fix"}]}`))
	}))
	defer server.Close()

	client := bitbucket.NewClient(server.URL, "token")
	handler := NewPRTools(client)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"project": "PRJ",
		"repo":    "my-repo",
		"pr_id":   float64(1),
	}

	result, err := handler.GetPRCommits(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := result.Content[0].(mcp.TextContent).Text
	var commits []bitbucket.Commit
	if err := json.Unmarshal([]byte(text), &commits); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(commits))
	}
}
