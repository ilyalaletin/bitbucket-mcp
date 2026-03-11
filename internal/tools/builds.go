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
	args := req.Params.Arguments.(map[string]any)
	commitID := stringArg(args, "commit_id")
	if commitID == "" {
		return errorResult(fmt.Errorf("commit_id is required")), nil
	}

	statuses, err := t.client.GetBuildStatus(ctx, commitID, 0)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(statuses)
}
