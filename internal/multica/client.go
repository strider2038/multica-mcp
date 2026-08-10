package multica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/strider2038/multica-mcp/internal/domain"
)

type Client struct {
	baseURL       string
	token         string
	clientVersion string
	httpClient    *http.Client
	workspaceID   string
	workspaceSlug string
}

func NewClient(baseURL, token, clientVersion string) *Client {
	return &Client{
		baseURL:       baseURL,
		token:         token,
		clientVersion: strings.TrimSpace(clientVersion),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetWorkspaceScope sets how workspace-scoped API routes identify the workspace.
// Use either id (UUID) or slug; when slug is non-empty it is sent as X-Workspace-Slug
// (Multica resolves it first). When slug is empty, id is sent as X-Workspace-ID.
func (c *Client) SetWorkspaceScope(id, slug string) {
	c.workspaceID = id
	c.workspaceSlug = slug
}

func (c *Client) workspaceAttrs() []any {
	if c.workspaceSlug != "" {
		return []any{"workspace_slug", c.workspaceSlug}
	}
	return []any{"workspace_id", c.workspaceID}
}

func (c *Client) attachWorkspaceHeaders(req *http.Request) error {
	if c.workspaceSlug != "" {
		req.Header.Set("X-Workspace-Slug", c.workspaceSlug)
		return nil
	}
	if c.workspaceID != "" {
		req.Header.Set("X-Workspace-ID", c.workspaceID)
		return nil
	}
	return fmt.Errorf("multica client: workspace scope not configured (call SetWorkspaceScope with id or slug)")
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
	if err := c.doGet(ctx, "/api/workspaces", &resp, false); err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return resp.Workspaces, nil
}

func (c *Client) GetWorkspace(ctx context.Context, id string) (*domain.Workspace, error) {
	var resp domain.Workspace
	if err := c.doGet(ctx, "/api/workspaces/"+id, &resp, false); err != nil {
		return nil, fmt.Errorf("get workspace %s: %w", id, err)
	}
	return &resp, nil
}

func (c *Client) ListProjects(ctx context.Context) ([]domain.Project, error) {
	path := "/api/projects"
	var resp listProjectsResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return resp.Projects, nil
}

func (c *Client) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	path := "/api/projects/" + projectID
	var resp domain.Project
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("get project %s: %w", projectID, err)
	}
	return &resp, nil
}

func (c *Client) ListTasks(ctx context.Context, opts domain.ListTasksInput) ([]domain.Task, error) {
	q := url.Values{}
	if opts.ProjectID != "" {
		q.Set("project_id", opts.ProjectID)
	}
	if opts.Status != nil && *opts.Status != "" {
		q.Set("status", *opts.Status)
	}
	if opts.Assignee != nil && *opts.Assignee != "" {
		q.Set("assignee_id", *opts.Assignee)
	}
	if opts.Limit != nil && *opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(*opts.Limit))
	}
	path := "/api/issues"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	var resp listIssuesResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return resp.Issues, nil
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	path := "/api/issues/" + taskID
	var resp domain.Task
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("get task %s: %w", taskID, err)
	}
	return &resp, nil
}

func (c *Client) ListChildIssues(ctx context.Context, parentID string) ([]domain.Task, error) {
	path := "/api/issues/" + parentID + "/children"
	var resp listIssuesResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("list child issues for %s: %w", parentID, err)
	}
	return resp.Issues, nil
}

func (c *Client) CreateTask(ctx context.Context, input domain.CreateTaskInput) (*domain.Task, error) {
	path := "/api/issues"
	body := map[string]any{
		"title":       input.Title,
		"description": input.Description,
	}
	if input.Priority != nil && *input.Priority != "" {
		body["priority"] = *input.Priority
	}
	if input.Status != nil && *input.Status != "" {
		body["status"] = *input.Status
	}
	if input.StartDate != nil && *input.StartDate != "" {
		body["start_date"] = *input.StartDate
	}
	if input.DueDate != nil && *input.DueDate != "" {
		body["due_date"] = *input.DueDate
	}
	if len(input.Labels) > 0 {
		body["label_ids"] = input.Labels
	}
	if input.Assignee != nil && *input.Assignee != "" {
		body["assignee_id"] = *input.Assignee
		if input.AssigneeType != nil && *input.AssigneeType != "" {
			body["assignee_type"] = *input.AssigneeType
		}
	}
	if input.ProjectID != "" {
		body["project_id"] = input.ProjectID
	}
	if input.ParentIssueID != nil && *input.ParentIssueID != "" {
		body["parent_issue_id"] = *input.ParentIssueID
	}
	if input.Stage != nil && *input.Stage >= 1 {
		body["stage"] = *input.Stage
	}

	var resp domain.Task
	if err := c.doPost(ctx, path, body, &resp, true); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	slog.Info("task created", append([]any{"task_id", resp.ID, "title", resp.Title}, c.workspaceAttrs()...)...)
	return &resp, nil
}

func (c *Client) UpdateTask(ctx context.Context, taskID string, input domain.UpdateTaskInput) (*domain.Task, error) {
	path := "/api/issues/" + taskID
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
	if input.Position != nil {
		body["position"] = *input.Position
	}
	if input.ClearStartDate {
		body["start_date"] = nil
	} else if input.StartDate != nil && *input.StartDate != "" {
		body["start_date"] = *input.StartDate
	}
	if input.ClearDueDate {
		body["due_date"] = nil
	} else if input.DueDate != nil && *input.DueDate != "" {
		body["due_date"] = *input.DueDate
	}
	if input.ClearParentIssueID {
		body["parent_issue_id"] = nil
	} else if input.ParentIssueID != nil && *input.ParentIssueID != "" {
		body["parent_issue_id"] = *input.ParentIssueID
	}
	if input.ClearProjectID {
		body["project_id"] = nil
	} else if input.ProjectID != nil && *input.ProjectID != "" {
		body["project_id"] = *input.ProjectID
	}
	if input.Assignee != nil {
		if *input.Assignee == "" {
			body["assignee_id"] = nil
			body["assignee_type"] = nil
		} else {
			body["assignee_id"] = *input.Assignee
			if input.AssigneeType != nil && *input.AssigneeType != "" {
				body["assignee_type"] = *input.AssigneeType
			}
		}
	}
	if input.ClearStage {
		body["stage"] = nil
	} else if input.Stage != nil {
		body["stage"] = *input.Stage
	}
	if input.SuppressRun {
		body["suppress_run"] = true
	}
	if input.HandoffNote != "" {
		body["handoff_note"] = input.HandoffNote
	}

	var resp domain.Task
	if err := c.doPut(ctx, path, body, &resp, true); err != nil {
		return nil, fmt.Errorf("update task %s: %w", taskID, err)
	}
	slog.Info("task updated", append([]any{"task_id", taskID}, c.workspaceAttrs()...)...)
	return &resp, nil
}

func (c *Client) ListComments(ctx context.Context, issueID string) ([]domain.Comment, error) {
	path := "/api/issues/" + issueID + "/comments"
	var resp listCommentsResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("list comments for issue %s: %w", issueID, err)
	}
	return resp, nil
}

func (c *Client) CreateComment(ctx context.Context, input domain.AddCommentInput) (*domain.Comment, error) {
	path := "/api/issues/" + input.TaskID + "/comments"
	body := map[string]any{
		"content": input.Comment,
	}
	if input.ParentID != nil && *input.ParentID != "" {
		body["parent_id"] = *input.ParentID
	}
	if len(input.SuppressAgentIDs) > 0 {
		body["suppress_agent_ids"] = input.SuppressAgentIDs
	}

	var resp domain.Comment
	if err := c.doPost(ctx, path, body, &resp, true); err != nil {
		return nil, fmt.Errorf("create comment on issue %s: %w", input.TaskID, err)
	}
	slog.Info("comment created", append([]any{"comment_id", resp.ID, "issue_id", input.TaskID}, c.workspaceAttrs()...)...)
	return &resp, nil
}

func (c *Client) PreviewCommentTriggers(ctx context.Context, input domain.PreviewCommentTriggersInput) (*domain.CommentTriggerPreview, error) {
	path := "/api/issues/" + input.TaskID + "/comments/trigger-preview"
	body := map[string]any{
		"content": input.Content,
	}
	if input.ParentID != nil && *input.ParentID != "" {
		body["parent_id"] = *input.ParentID
	}

	var resp domain.CommentTriggerPreview
	if err := c.doPost(ctx, path, body, &resp, true); err != nil {
		return nil, fmt.Errorf("preview comment triggers for issue %s: %w", input.TaskID, err)
	}
	return &resp, nil
}

func (c *Client) PreviewIssueTriggers(ctx context.Context, input domain.PreviewIssueTriggersInput) (*domain.IssueTriggerPreview, error) {
	path := "/api/issues/preview-trigger"
	body := map[string]any{}
	if len(input.IssueIDs) > 0 {
		body["issue_ids"] = input.IssueIDs
	}
	if input.IsCreate {
		body["is_create"] = true
	}
	if input.AssigneeType != nil && *input.AssigneeType != "" {
		body["assignee_type"] = *input.AssigneeType
	}
	if input.AssigneeID != nil && *input.AssigneeID != "" {
		body["assignee_id"] = *input.AssigneeID
	}
	if input.Status != nil && *input.Status != "" {
		body["status"] = *input.Status
	}

	var resp domain.IssueTriggerPreview
	if err := c.doPost(ctx, path, body, &resp, true); err != nil {
		return nil, fmt.Errorf("preview issue triggers: %w", err)
	}
	return &resp, nil
}

func (c *Client) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	path := "/api/agents"
	var resp listAgentsResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return resp, nil
}

func (c *Client) SearchIssues(ctx context.Context, query string, projectID *string, status *string, limit *int) ([]domain.Task, error) {
	q := url.Values{}
	q.Set("q", query)
	if projectID != nil && *projectID != "" {
		q.Set("project_id", *projectID)
	}
	if status != nil && *status != "" {
		q.Set("status", *status)
	}
	if limit != nil && *limit > 0 {
		q.Set("limit", strconv.Itoa(*limit))
	}
	path := "/api/issues/search?" + q.Encode()

	var resp searchIssuesResponse
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}
	result := make([]domain.Task, len(resp.Issues))
	for i, item := range resp.Issues {
		result[i] = item.Task
	}
	return result, nil
}

func (c *Client) SearchProjects(ctx context.Context, query string) ([]domain.Project, error) {
	q := url.Values{}
	q.Set("q", query)
	path := "/api/projects/search?" + q.Encode()
	var resp struct {
		Projects []domain.Project `json:"projects"`
		Total    int              `json:"total"`
	}
	if err := c.doGet(ctx, path, &resp, true); err != nil {
		return nil, fmt.Errorf("search projects: %w", err)
	}
	return resp.Projects, nil
}

func (c *Client) doGet(ctx context.Context, path string, result any, scoped bool) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result, scoped)
}

func (c *Client) doPost(ctx context.Context, path string, body any, result any, scoped bool) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result, scoped)
}

func (c *Client) doPut(ctx context.Context, path string, body any, result any, scoped bool) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result, scoped)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, result any, scoped bool) error {
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
	req.Header.Set("X-Client-Platform", "mcp")
	if c.clientVersion != "" {
		req.Header.Set("X-Client-Version", c.clientVersion)
	}
	if scoped {
		if err := c.attachWorkspaceHeaders(req); err != nil {
			return err
		}
	}

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
			Message:    parseAPIErrorMessage(respBody),
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func parseAPIErrorMessage(body []byte) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		return payload.Error
	}
	return truncate(string(body), 200)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
