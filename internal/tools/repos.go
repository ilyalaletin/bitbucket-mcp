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
	args := req.Params.Arguments.(map[string]any)
	resp, err := t.client.Browse(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		stringArg(args, "path"),
		stringArg(args, "ref"),
		"",
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
	args := req.Params.Arguments.(map[string]any)
	path := stringArg(args, "path")
	resp, err := t.client.Browse(ctx,
		stringArg(args, "project"),
		stringArg(args, "repo"),
		path,
		stringArg(args, "ref"),
		"",
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
	args := req.Params.Arguments.(map[string]any)
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
