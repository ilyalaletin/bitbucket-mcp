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
	args := req.Params.Arguments.(map[string]any)
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
	args := req.Params.Arguments.(map[string]any)
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
	args := req.Params.Arguments.(map[string]any)
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
	args := req.Params.Arguments.(map[string]any)
	commits, err := t.client.GetPRCommits(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		intArg(args, "pr_id"),
		intArg(args, "limit"),
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
