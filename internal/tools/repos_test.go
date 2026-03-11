package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
		w.Write([]byte("diff --git a/file.go b/file.go\n+added line\n"))
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
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "diff --git") {
		t.Errorf("expected diff content, got %q", text)
	}
}
