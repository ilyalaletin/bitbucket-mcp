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
