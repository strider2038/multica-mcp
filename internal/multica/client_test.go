package multica

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strider2038/multica-mcp/internal/domain"
)

func TestClient_ListWorkspaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		assertAuth(t, r)
		json.NewEncoder(w).Encode(map[string]any{
			"workspaces": []map[string]any{
				{"id": "ws1", "name": "My Workspace", "slug": "my-workspace"},
			},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "mul_test123", "test")
	workspaces, err := client.ListWorkspaces(context.Background())
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
	if workspaces[0].Name != "My Workspace" {
		t.Errorf("expected name 'My Workspace', got %q", workspaces[0].Name)
	}
}

func TestClient_ListProjects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws1" {
			t.Errorf("expected X-Workspace-ID ws1, got %q", r.Header.Get("X-Workspace-ID"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"projects": []map[string]any{
				{"id": "p1", "title": "Backend", "status": "active"},
			},
			"total": 1,
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
}

func TestClient_CreateTask(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws1" {
			t.Errorf("expected X-Workspace-ID ws1, got %q", r.Header.Get("X-Workspace-ID"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Test" {
			t.Errorf("expected title 'Test', got %v", body["title"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "t1",
			"title":      "Test",
			"status":     "todo",
			"identifier": "MUL-1",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	task, err := client.CreateTask(context.Background(), domain.CreateTaskInput{
		Title:       "Test",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ID != "t1" {
		t.Errorf("expected id 't1', got %q", task.ID)
	}
}

func TestClient_CreateTask_WithExtendedFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "in_progress" {
			t.Errorf("expected status in_progress, got %v", body["status"])
		}
		if body["start_date"] != "2026-08-01" {
			t.Errorf("unexpected start_date: %v", body["start_date"])
		}
		if body["due_date"] != "2026-08-31" {
			t.Errorf("unexpected due_date: %v", body["due_date"])
		}
		ids, ok := body["label_ids"].([]any)
		if !ok || len(ids) != 1 || ids[0] != "label-1" {
			t.Errorf("unexpected label_ids: %v", body["label_ids"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "t1", "title": "Test", "status": "in_progress", "identifier": "MUL-1",
		})
	}))
	defer ts.Close()

	status := "in_progress"
	start := "2026-08-01"
	due := "2026-08-31"
	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.CreateTask(context.Background(), domain.CreateTaskInput{
		Title:       "Test",
		Description: "desc",
		Status:      &status,
		StartDate:   &start,
		DueDate:     &due,
		LabelIDs:    []string{"label-1"},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
}

func TestClient_UpdateTask_WithExtendedFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["position"] != 42.5 {
			t.Errorf("expected position 42.5, got %v", body["position"])
		}
		if body["start_date"] != nil {
			t.Errorf("expected start_date null, got %v", body["start_date"])
		}
		if body["parent_issue_id"] != "parent-2" {
			t.Errorf("unexpected parent_issue_id: %v", body["parent_issue_id"])
		}
		if body["project_id"] != "proj-1" {
			t.Errorf("unexpected project_id: %v", body["project_id"])
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "t1", "title": "Task"})
	}))
	defer ts.Close()

	pos := 42.5
	parent := "parent-2"
	project := "proj-1"
	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.UpdateTask(context.Background(), "t1", domain.UpdateTaskInput{
		Position:       &pos,
		ClearStartDate: true,
		ParentIssueID:  &parent,
		ProjectID:      &project,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
}

func TestClient_CreateTask_WithParentIssue(t *testing.T) {
	parentID := "parent-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["parent_issue_id"] != parentID {
			t.Errorf("expected parent_issue_id %q, got %v", parentID, body["parent_issue_id"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "subtask-1",
			"title":      "Subtask",
			"status":     "todo",
			"identifier": "MUL-2",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.CreateTask(context.Background(), domain.CreateTaskInput{
		ParentIssueID: &parentID,
		Title:         "Subtask",
		Description:   "desc",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
}

func TestClient_CreateComment_WithSuppressAgentIDs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues/t1/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		ids, ok := body["suppress_agent_ids"].([]any)
		if !ok || len(ids) != 1 || ids[0] != "agent-1" {
			t.Errorf("unexpected suppress_agent_ids: %v", body["suppress_agent_ids"])
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "c1",
			"content": "hello",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	comment, err := client.CreateComment(context.Background(), domain.AddCommentInput{
		TaskID:           "t1",
		Comment:          "hello",
		SuppressAgentIDs: []string{"agent-1"},
	})
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if comment.ID != "c1" {
		t.Errorf("expected id c1, got %q", comment.ID)
	}
}

func TestClient_PreviewCommentTriggers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues/t1/comments/trigger-preview" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "@agent please review" {
			t.Errorf("unexpected content: %v", body["content"])
		}
		json.NewEncoder(w).Encode(map[string]any{
			"agents": []map[string]any{
				{
					"id":     "agent-1",
					"name":   "Reviewer",
					"source": "mention_agent",
					"reason": "This agent was mentioned in the comment.",
				},
			},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	preview, err := client.PreviewCommentTriggers(context.Background(), domain.PreviewCommentTriggersInput{
		TaskID:  "t1",
		Content: "@agent please review",
	})
	if err != nil {
		t.Fatalf("PreviewCommentTriggers: %v", err)
	}
	if len(preview.Agents) != 1 || preview.Agents[0].ID != "agent-1" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestClient_UpdateTask_WithSuppressRunAndHandoff(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues/t1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["suppress_run"] != true {
			t.Errorf("expected suppress_run true, got %v", body["suppress_run"])
		}
		if body["handoff_note"] != "please review" {
			t.Errorf("unexpected handoff_note: %v", body["handoff_note"])
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "t1", "title": "Task"})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.UpdateTask(context.Background(), "t1", domain.UpdateTaskInput{
		Assignee:    strPtr("agent-1"),
		SuppressRun: true,
		HandoffNote: "please review",
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
}

func TestClient_UpdateTask_ClearStage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["stage"] != nil {
			t.Errorf("expected stage null, got %v", body["stage"])
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "t1", "title": "Task"})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.UpdateTask(context.Background(), "t1", domain.UpdateTaskInput{
		ClearStage: true,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
}

func TestClient_PreviewIssueTriggers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues/preview-trigger" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		ids, ok := body["issue_ids"].([]any)
		if !ok || len(ids) != 1 || ids[0] != "issue-1" {
			t.Errorf("unexpected issue_ids: %v", body["issue_ids"])
		}
		json.NewEncoder(w).Encode(map[string]any{
			"triggers": []map[string]any{
				{
					"issue_id":          "issue-1",
					"agent_id":          "agent-1",
					"source":            "assign_agent",
					"handoff_supported": true,
				},
			},
			"total_count": 1,
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	preview, err := client.PreviewIssueTriggers(context.Background(), domain.PreviewIssueTriggersInput{
		IssueIDs: []string{"issue-1"},
	})
	if err != nil {
		t.Fatalf("PreviewIssueTriggers: %v", err)
	}
	if preview.TotalCount != 1 || len(preview.Triggers) != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestClient_ErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "not found",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("ws1", "")
	_, err := client.GetProject(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr := &apiError{}
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apiError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestClient_ListProjects_WithSlug(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Workspace-Slug") != "acme" {
			t.Errorf("expected X-Workspace-Slug acme, got %q", r.Header.Get("X-Workspace-Slug"))
		}
		if r.Header.Get("X-Workspace-ID") != "" {
			t.Errorf("did not expect X-Workspace-ID when using slug, got %q", r.Header.Get("X-Workspace-ID"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"projects": []map[string]any{
				{"id": "p1", "title": "API"},
			},
			"total": 1,
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "test")
	client.SetWorkspaceScope("", "acme")
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].Title != "API" {
		t.Fatalf("unexpected projects: %+v", projects)
	}
}

func TestClient_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "unauthorized")
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "bad-token", "test")
	_, err := client.ListWorkspaces(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func assertAuth(t *testing.T, r *http.Request) {
	t.Helper()
	auth := r.Header.Get("Authorization")
	if auth != "Bearer mul_test123" {
		t.Errorf("expected Authorization 'Bearer mul_test123', got %q", auth)
	}
}
