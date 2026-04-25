package multica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/strider2038/multica-mcp/internal/domain"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type apiError struct {
	StatusCode int
	Message    string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("multica api error: status=%d message=%s", e.StatusCode, e.Message)
}

type listWorkspacesResponse struct {
	Workspaces []domain.Workspace `json:"workspaces"`
}

type listProjectsResponse struct {
	Projects []domain.Project `json:"projects"`
	Total    int              `json:"total"`
}

type listIssuesResponse struct {
	Issues []domain.Task `json:"issues"`
	Total  int           `json:"total"`
}

type listCommentsResponse []domain.Comment

type listAgentsResponse []domain.Agent

type searchIssuesResponse struct {
	Issues []searchIssueItem `json:"issues"`
	Total  int               `json:"total"`
}

type searchIssueItem struct {
	domain.Task
	MatchSource    string  `json:"match_source"`
	MatchedSnippet *string `json:"matched_snippet,omitempty"`
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	var resp listWorkspacesResponse
	if err := c.doGet(ctx, "/api/workspaces", &resp); err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return resp.Workspaces, nil
}

func (c *Client) GetWorkspace(ctx context.Context, id string) (*domain.Workspace, error) {
	var resp domain.Workspace
	if err := c.doGet(ctx, "/api/workspaces/"+id, &resp); err != nil {
		return nil, fmt.Errorf("get workspace %s: %w", id, err)
	}
	return &resp, nil
}

func (c *Client) ListProjects(ctx context.Context, workspaceID string, query string) ([]domain.Project, error) {
	path := fmt.Sprintf("/api/workspaces/%s/projects", workspaceID)
	if query != "" {
		path += "?q=" + query
	}
	var resp listProjectsResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return resp.Projects, nil
}

func (c *Client) GetProject(ctx context.Context, workspaceID, projectID string) (*domain.Project, error) {
	path := fmt.Sprintf("/api/workspaces/%s/projects/%s", workspaceID, projectID)
	var resp domain.Project
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get project %s: %w", projectID, err)
	}
	return &resp, nil
}

func (c *Client) ListTasks(ctx context.Context, workspaceID string, opts domain.ListTasksInput) ([]domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues?", workspaceID)
	params := []string{}
	if opts.ProjectID != "" {
		params = append(params, "project_id="+opts.ProjectID)
	}
	if opts.Status != nil && *opts.Status != "" {
		params = append(params, "status="+*opts.Status)
	}
	if opts.Assignee != nil && *opts.Assignee != "" {
		params = append(params, "assignee_id="+*opts.Assignee)
	}
	if opts.Limit != nil && *opts.Limit > 0 {
		params = append(params, "limit="+strconv.Itoa(*opts.Limit))
	}
	for i, p := range params {
		if i == 0 {
			path += p
		} else {
			path += "&" + p
		}
	}

	var resp listIssuesResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return resp.Issues, nil
}

func (c *Client) GetTask(ctx context.Context, workspaceID, taskID string) (*domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/%s", workspaceID, taskID)
	var resp domain.Task
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get task %s: %w", taskID, err)
	}
	return &resp, nil
}

func (c *Client) ListChildIssues(ctx context.Context, workspaceID, parentID string) ([]domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/%s/children", workspaceID, parentID)
	var resp listIssuesResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list child issues for %s: %w", parentID, err)
	}
	return resp.Issues, nil
}

func (c *Client) CreateTask(ctx context.Context, workspaceID string, input domain.CreateTaskInput) (*domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues", workspaceID)
	body := map[string]any{
		"title":       input.Title,
		"description": input.Description,
	}
	if input.Priority != nil && *input.Priority != "" {
		body["priority"] = *input.Priority
	}
	if input.Assignee != nil && *input.Assignee != "" {
		body["assignee_id"] = *input.Assignee
	}
	if input.ProjectID != "" {
		body["project_id"] = input.ProjectID
	}

	var resp domain.Task
	if err := c.doPost(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	slog.Info("task created", "task_id", resp.ID, "title", resp.Title, "workspace_id", workspaceID)
	return &resp, nil
}

func (c *Client) UpdateTask(ctx context.Context, workspaceID, taskID string, input domain.UpdateTaskInput) (*domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/%s", workspaceID, taskID)
	body := map[string]any{}
	if input.Title != nil {
		body["title"] = *input.Title
	}
	if input.Description != nil {
		body["description"] = *input.Description
	}
	if input.Status != nil {
		body["status"] = *input.Status
	}
	if input.Priority != nil {
		body["priority"] = *input.Priority
	}
	if input.Assignee != nil {
		if *input.Assignee == "" {
			body["assignee_id"] = nil
			body["assignee_type"] = nil
		} else {
			body["assignee_id"] = *input.Assignee
		}
	}

	var resp domain.Task
	if err := c.doPut(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("update task %s: %w", taskID, err)
	}
	slog.Info("task updated", "task_id", taskID, "workspace_id", workspaceID)
	return &resp, nil
}

func (c *Client) ListComments(ctx context.Context, workspaceID, issueID string) ([]domain.Comment, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/%s/comments", workspaceID, issueID)
	var resp listCommentsResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list comments for issue %s: %w", issueID, err)
	}
	return resp, nil
}

func (c *Client) CreateComment(ctx context.Context, workspaceID, issueID, content string) (*domain.Comment, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/%s/comments", workspaceID, issueID)
	body := map[string]any{
		"content": content,
	}
	var resp domain.Comment
	if err := c.doPost(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("create comment on issue %s: %w", issueID, err)
	}
	slog.Info("comment created", "comment_id", resp.ID, "issue_id", issueID, "workspace_id", workspaceID)
	return &resp, nil
}

func (c *Client) ListAgents(ctx context.Context, workspaceID string) ([]domain.Agent, error) {
	path := fmt.Sprintf("/api/workspaces/%s/agents", workspaceID)
	var resp listAgentsResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return resp, nil
}

func (c *Client) SearchIssues(ctx context.Context, workspaceID, query string, projectID *string, status *string, limit *int) ([]domain.Task, error) {
	path := fmt.Sprintf("/api/workspaces/%s/issues/search?q=%s", workspaceID, query)
	if projectID != nil && *projectID != "" {
		path += "&project_id=" + *projectID
	}
	if status != nil && *status != "" {
		path += "&status=" + *status
	}
	if limit != nil && *limit > 0 {
		path += "&limit=" + strconv.Itoa(*limit)
	}

	var resp searchIssuesResponse
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}
	result := make([]domain.Task, len(resp.Issues))
	for i, item := range resp.Issues {
		result[i] = item.Task
	}
	return result, nil
}

func (c *Client) SearchProjects(ctx context.Context, workspaceID, query string) ([]domain.Project, error) {
	path := fmt.Sprintf("/api/workspaces/%s/projects/search?q=%s", workspaceID, query)
	var resp struct {
		Projects []domain.Project `json:"projects"`
		Total    int              `json:"total"`
	}
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("search projects: %w", err)
	}
	return resp.Projects, nil
}

func (c *Client) doGet(ctx context.Context, path string, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) doPost(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

func (c *Client) doPut(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		slog.Debug("api error response", "method", method, "path", path, "status", resp.StatusCode, "body", truncate(string(respBody), 500))
		return &apiError{
			StatusCode: resp.StatusCode,
			Message:    truncate(string(respBody), 200),
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
