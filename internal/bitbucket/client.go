package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

const maxDiffSize = 1024 * 1024 // 1MB

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

// ListPRsOptions specifies filters for listing pull requests
type ListPRsOptions struct {
	Project string // project key (required if Role is empty)
	Repo    string // repository slug (required if Role is empty)
	State   string // OPEN, MERGED, DECLINED, or ALL (defaults to OPEN)
	Role    string // AUTHOR or REVIEWER (optional; if set, uses dashboard endpoint)
	Limit   int    // max results per page (default 25, max 100)
}

// paginate is a generic helper that handles multi-page pagination.
// It fetches pages until isLastPage is true, accumulating results up to limit total items.
func paginate[T any](ctx context.Context, c *Client, path string, query url.Values, limit int) ([]T, error) {
	if limit == 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	var allResults []T
	start := 0

	for {
		// Set pagination params
		q := url.Values{}
		if query != nil {
			q = query
		}
		q.Set("start", fmt.Sprintf("%d", start))
		q.Set("limit", fmt.Sprintf("%d", limit))

		var resp PagedResponse[T]
		if err := c.getJSON(ctx, path, q, &resp); err != nil {
			return nil, err
		}

		allResults = append(allResults, resp.Values...)

		if resp.IsLastPage || len(allResults) >= limit {
			break
		}

		start = resp.NextPageStart
	}

	// Trim to limit if we exceeded it
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

// ListPRs lists pull requests with optional filters.
// If Role is set, uses dashboard endpoint (project/repo ignored).
// Otherwise, uses repo-scoped endpoint (project and repo required).
func (c *Client) ListPRs(ctx context.Context, opts ListPRsOptions) ([]PullRequest, error) {
	if opts.State == "" {
		opts.State = "OPEN"
	}

	query := url.Values{}
	query.Set("state", opts.State)

	var path string
	if opts.Role != "" {
		// Dashboard endpoint
		path = "/rest/api/1.0/dashboard/pull-requests"
		query.Set("role", opts.Role)
	} else {
		// Repo-scoped endpoint
		if opts.Project == "" || opts.Repo == "" {
			return nil, fmt.Errorf("project and repo required when role is not set")
		}
		path = fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", opts.Project, opts.Repo)
	}

	return paginate[PullRequest](ctx, c, path, query, opts.Limit)
}

// GetPR gets details of a single pull request.
func (c *Client) GetPR(ctx context.Context, project, repo string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", project, repo, prID)
	var pr PullRequest
	if err := c.getJSON(ctx, path, nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetPRDiff gets the full diff for a pull request as raw text.
func (c *Client) GetPRDiff(ctx context.Context, project, repo string, prID int) (string, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff", project, repo, prID)
	data, err := c.getRaw(ctx, path, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetPRCommits gets the list of commits in a pull request.
func (c *Client) GetPRCommits(ctx context.Context, project, repo string, prID int, limit int) ([]Commit, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/commits", project, repo, prID)
	return paginate[Commit](ctx, c, path, nil, limit)
}

// GetBuildStatus gets the build statuses for a commit.
func (c *Client) GetBuildStatus(ctx context.Context, commitID string, limit int) ([]BuildStatus, error) {
	path := fmt.Sprintf("/rest/build-status/1.0/commits/%s", commitID)
	return paginate[BuildStatus](ctx, c, path, nil, limit)
}

// Browse retrieves directory listing or file content for a path in a repository.
// For directories, the response includes Children.
// For files, the response includes Lines.
// For binary files, Binary flag is set to true.
func (c *Client) Browse(ctx context.Context, project, repo, path, ref string, _ string) (*BrowseResponse, error) {
	// ref is optional; omit if empty
	pathEscaped := url.PathEscape(path)
	p := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/browse/%s", project, repo, pathEscaped)

	query := url.Values{}
	if ref != "" {
		query.Set("at", ref)
	}

	var resp BrowseResponse
	if err := c.getJSON(ctx, p, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDiff returns the diff between two refs as a raw string.
// Supports optional path parameter to filter to a single file.
func (c *Client) GetDiff(ctx context.Context, project, repo, from, to, path string) (string, error) {
	p := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/compare/diff", project, repo)

	query := url.Values{}
	query.Set("from", from)
	query.Set("to", to)
	if path != "" {
		query.Set("path", path)
	}

	data, err := c.getRaw(ctx, p, query)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getRaw fetches raw text (not JSON). Used for diff endpoints.
// Sets Accept: text/plain to get raw unified diff from Bitbucket.
// Truncates response at 1MB with a notice.
func (c *Client) getRaw(ctx context.Context, path string, query url.Values) ([]byte, error) {
	fullURL := c.baseURL + path
	if query != nil {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
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
