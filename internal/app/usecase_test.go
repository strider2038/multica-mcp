package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strider2038/multica-mcp/internal/domain"
	"github.com/strider2038/multica-mcp/internal/multica"
)

func ptr(s string) *string { return &s }
func intptr(i int) *int    { return &i }

func setupTestServer(handler http.HandlerFunc) (*UseCase, *httptest.Server) {
	ts := httptest.NewServer(handler)
	client := multica.NewClient(ts.URL, "test-token")
	client.SetWorkspaceScope("ws-123", "")
	uc := NewUseCase(client, false)
	return uc, ts
}

func TestListProjects(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"projects": []map[string]any{
				{"id": "p1", "title": "Backend", "status": "active", "workspace_id": "ws-123"},
				{"id": "p2", "title": "Frontend", "status": "active", "workspace_id": "ws-123"},
			},
			"total": 2,
		})
	})
	defer ts.Close()

	projects, err := uc.ListProjects(context.Background(), domain.ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].Title != "Backend" {
		t.Errorf("expected first project 'Backend', got %q", projects[0].Title)
	}
}

func TestListProjectsWithQuery(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		q := r.URL.Query().Get("q")
		if q != "backend" {
			t.Errorf("expected query 'backend', got %q", q)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"projects": []map[string]any{
				{"id": "p1", "title": "Backend Search Match"},
			},
			"total": 1,
		})
	})
	defer ts.Close()

	projects, err := uc.ListProjects(context.Background(), domain.ListProjectsInput{Query: "backend"})
	if err != nil {
		t.Fatalf("ListProjects with query: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Title != "Backend Search Match" {
		t.Errorf("unexpected project title: %q", projects[0].Title)
	}
}

func TestGetTask(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/issues/task-1":
			json.NewEncoder(w).Encode(map[string]any{
				"id": "task-1", "title": "Test Task", "status": "todo",
				"identifier": "MUL-1", "workspace_id": "ws-123",
			})
		case "/api/issues/task-1/comments":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": "c1", "content": "first comment"},
			})
		case "/api/issues/task-1/children":
			json.NewEncoder(w).Encode(map[string]any{
				"issues": []map[string]any{
					{"id": "child-1", "title": "Subtask 1"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "not found: %s", r.URL.Path)
		}
	})
	defer ts.Close()

	task, err := uc.GetTask(context.Background(), domain.GetTaskInput{TaskID: "task-1"})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", task.Title)
	}
	if len(task.Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(task.Comments))
	}
}

func TestCreateTask(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "New Task" {
			t.Errorf("expected title 'New Task', got %v", body["title"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "new-1", "title": "New Task", "status": "todo",
			"identifier": "MUL-42", "workspace_id": "ws-123",
		})
	})
	defer ts.Close()

	result, err := uc.CreateTask(context.Background(), domain.CreateTaskInput{
		ProjectID:   "p1",
		Title:       "New Task",
		Description: "A task description",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if result.ID != "new-1" {
		t.Errorf("expected id 'new-1', got %q", result.ID)
	}
	if result.Identifier != "MUL-42" {
		t.Errorf("expected identifier 'MUL-42', got %q", result.Identifier)
	}
}

func TestCreateTask_DryRun(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make HTTP call in dry_run mode")
	})
	defer ts.Close()

	result, err := uc.CreateTask(context.Background(), domain.CreateTaskInput{
		Title:       "Dry Run Task",
		Description: "desc",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("CreateTask dry_run: %v", err)
	}
	if result.Identifier != "(dry run)" {
		t.Errorf("expected dry run identifier, got %q", result.Identifier)
	}
}

func TestUpdateTask_InvalidStatus(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make HTTP call for invalid status")
	})
	defer ts.Close()

	status := "invalid_status"
	_, err := uc.UpdateTask(context.Background(), domain.UpdateTaskInput{
		TaskID: "t1",
		Status: &status,
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestReadOnlyMode(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make HTTP call in read-only mode")
	})
	defer ts.Close()

	readOnlyUC := NewUseCase(uc.client, true)

	_, err := readOnlyUC.CreateTask(context.Background(), domain.CreateTaskInput{
		Title: "test", Description: "test",
	})
	if err == nil {
		t.Fatal("expected read-only error")
	}
}

func TestSearchTasks(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		if r.URL.Query().Get("q") != "refactor" {
			t.Errorf("expected q=refactor, got %q", r.URL.Query().Get("q"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"issues": []map[string]any{
				{"id": "s1", "title": "Refactor API", "status": "in_progress", "match_source": "title"},
			},
			"total": 1,
		})
	})
	defer ts.Close()

	tasks, err := uc.SearchTasks(context.Background(), domain.SearchTasksInput{
		Query: "refactor",
	})
	if err != nil {
		t.Fatalf("SearchTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestListAgents(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agents" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "a1", "name": "Claude", "status": "active"},
			{"id": "a2", "name": "Codex", "status": "active"},
		})
	})
	defer ts.Close()

	agents, err := uc.ListAgents(context.Background(), domain.ListAgentsInput{})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestPlanTaskBreakdown(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Error("plan should not make HTTP calls")
	})
	defer ts.Close()

	result, err := uc.PlanTaskBreakdown(context.Background(), domain.PlanTaskBreakdownInput{
		Title:       "Add auth",
		Description: "Implement OAuth2 authentication",
	})
	if err != nil {
		t.Fatalf("PlanTaskBreakdown: %v", err)
	}
	if len(result.Subtasks) == 0 {
		t.Fatal("expected subtasks in breakdown")
	}
	for _, s := range result.Subtasks {
		if len(s.AcceptanceCriteria) == 0 {
			t.Errorf("subtask %q missing acceptance criteria", s.Title)
		}
	}
}

func TestAddComment(t *testing.T) {
	uc, ts := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/issues/t1/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Workspace-ID") != "ws-123" {
			t.Errorf("expected X-Workspace-ID ws-123, got %q", r.Header.Get("X-Workspace-ID"))
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "c1", "content": "hello world", "issue_id": "t1",
		})
	})
	defer ts.Close()

	comment, err := uc.AddComment(context.Background(), domain.AddCommentInput{
		TaskID:  "t1",
		Comment: "hello world",
	})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if comment.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", comment.Content)
	}
}
