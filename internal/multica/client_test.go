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

	client := NewClient(ts.URL, "mul_test123")
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
		if r.URL.Path != "/api/workspaces/ws1/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"projects": []map[string]any{
				{"id": "p1", "title": "Backend", "status": "active"},
			},
			"total": 1,
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token")
	projects, err := client.ListProjects(context.Background(), "ws1", "")
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

	client := NewClient(ts.URL, "test-token")
	task, err := client.CreateTask(context.Background(), "ws1", domain.CreateTaskInput{
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

func TestClient_ErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "not found",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token")
	_, err := client.GetProject(context.Background(), "ws1", "nonexistent")
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

func TestClient_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "unauthorized")
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "bad-token")
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
