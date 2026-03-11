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

	// Create Bitbucket client
	client := bitbucket.NewClient(bitbucketURL, bitbucketToken)

	// Create tool handler instances
	prTools := tools.NewPRTools(client)
	buildTools := tools.NewBuildTools(client)
	repoTools := tools.NewRepoTools(client)

	// Create MCP server
	s := server.NewMCPServer(
		"bitbucket-mcp",
		"1.0.0",
	)

	// Register PR tools
	registerListPRs(s, prTools)
	registerGetPR(s, prTools)
	registerGetPRDiff(s, prTools)
	registerGetPRCommits(s, prTools)

	// Register build tools
	registerGetBuildStatus(s, buildTools)

	// Register repo tools
	registerListFiles(s, repoTools)
	registerGetFileContent(s, repoTools)
	registerGetDiff(s, repoTools)

	// Serve via stdio
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Failed to serve stdio: %v", err)
	}
}

// Tool registration functions

func registerListPRs(s *server.MCPServer, t *tools.PRTools) {
	tool := mcp.NewTool(
		"list_prs",
		mcp.WithDescription("List pull requests with filters. If role is set, uses dashboard endpoint (project/repo ignored). Otherwise project and repo are required for repo-scoped listing."),
		mcp.WithString("project", mcp.Description("Project key (optional if role is set)")),
		mcp.WithString("repo", mcp.Description("Repository slug (optional if role is set)")),
		mcp.WithString("state", mcp.Description("PR state: OPEN, MERGED, DECLINED, or ALL (default: OPEN)")),
		mcp.WithString("role", mcp.Description("Filter by role: AUTHOR or REVIEWER (optional)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default: 25, max: 100)")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.ListPRs(ctx, request)
	})
}

func registerGetPR(s *server.MCPServer, t *tools.PRTools) {
	tool := mcp.NewTool(
		"get_pr",
		mcp.WithDescription("Get details of a single pull request"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithNumber("pr_id", mcp.Description("Pull request ID"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetPR(ctx, request)
	})
}

func registerGetPRDiff(s *server.MCPServer, t *tools.PRTools) {
	tool := mcp.NewTool(
		"get_pr_diff",
		mcp.WithDescription("Get the full diff for a pull request"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithNumber("pr_id", mcp.Description("Pull request ID"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetPRDiff(ctx, request)
	})
}

func registerGetPRCommits(s *server.MCPServer, t *tools.PRTools) {
	tool := mcp.NewTool(
		"get_pr_commits",
		mcp.WithDescription("Get the list of commits in a pull request"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithNumber("pr_id", mcp.Description("Pull request ID"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetPRCommits(ctx, request)
	})
}

func registerGetBuildStatus(s *server.MCPServer, t *tools.BuildTools) {
	tool := mcp.NewTool(
		"get_build_status",
		mcp.WithDescription("Get build statuses for a commit"),
		mcp.WithString("commit_id", mcp.Description("Commit ID"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetBuildStatus(ctx, request)
	})
}

func registerListFiles(s *server.MCPServer, t *tools.RepoTools) {
	tool := mcp.NewTool(
		"list_files",
		mcp.WithDescription("List files and directories in a repository path"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithString("path", mcp.Description("Directory path (optional)")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit hash (optional)")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.ListFiles(ctx, request)
	})
}

func registerGetFileContent(s *server.MCPServer, t *tools.RepoTools) {
	tool := mcp.NewTool(
		"get_file_content",
		mcp.WithDescription("Read file content from a repository"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithString("path", mcp.Description("File path"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit hash (optional)")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetFileContent(ctx, request)
	})
}

func registerGetDiff(s *server.MCPServer, t *tools.RepoTools) {
	tool := mcp.NewTool(
		"get_diff",
		mcp.WithDescription("Get diff between two refs in a repository"),
		mcp.WithString("project", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("repo", mcp.Description("Repository slug"), mcp.Required()),
		mcp.WithString("from", mcp.Description("Source branch/tag/commit"), mcp.Required()),
		mcp.WithString("to", mcp.Description("Target branch/tag/commit"), mcp.Required()),
		mcp.WithString("path", mcp.Description("Optional path to filter diff to single file")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return t.GetDiff(ctx, request)
	})
}
