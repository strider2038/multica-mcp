package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/strider2038/multica-mcp/internal/app"
	"github.com/strider2038/multica-mcp/internal/domain"
)

type Server struct {
	mcpServer *server.MCPServer
	useCase   *app.UseCase
}

func NewServer(useCase *app.UseCase, readOnly bool) *Server {
	s := &Server{
		useCase: useCase,
	}

	s.mcpServer = server.NewMCPServer(
		"multica-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	s.registerTools(readOnly)
	return s
}

func (s *Server) registerTools(readOnly bool) {
	s.addTool(listProjectsTool(), s.handleListProjects)
	s.addTool(getProjectTool(), s.handleGetProject)
	s.addTool(listTasksTool(), s.handleListTasks)
	s.addTool(getTaskTool(), s.handleGetTask)
	s.addTool(searchTasksTool(), s.handleSearchTasks)
	s.addTool(listAgentsTool(), s.handleListAgents)
	s.addTool(planTaskBreakdownTool(), s.handlePlanTaskBreakdown)

	if !readOnly {
		s.addTool(createTaskTool(), s.handleCreateTask)
		s.addTool(createSubtaskTool(), s.handleCreateSubtask)
		s.addTool(updateTaskTool(), s.handleUpdateTask)
		s.addTool(addCommentTool(), s.handleAddComment)
		s.addTool(assignTaskTool(), s.handleAssignTask)
		s.addTool(createTaskWithSubtasksTool(), s.handleCreateTaskWithSubtasks)
	}
}

func (s *Server) addTool(tool mcplib.Tool, handler func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error)) {
	s.mcpServer.AddTool(tool, handler)
}

func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

func listProjectsTool() mcplib.Tool {
	return mcplib.NewTool("multica_list_projects",
		mcplib.WithDescription("List projects in the Multica workspace. Optionally filter by name query."),
		mcplib.WithString("query",
			mcplib.Description("Optional search query to filter projects by name"),
		),
	)
}

func (s *Server) handleListProjects(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.ListProjectsInput{
		Query: argsGetString(req, "query"),
	}

	projects, err := s.useCase.ListProjects(ctx, input)
	if err != nil {
		return errorResult("list projects", err)
	}

	return jsonResult(projects)
}

func getProjectTool() mcplib.Tool {
	return mcplib.NewTool("multica_get_project",
		mcplib.WithDescription("Get detailed information about a specific project by ID."),
		mcplib.WithString("project_id",
			mcplib.Description("Project ID"),
			mcplib.Required(),
		),
	)
}

func (s *Server) handleGetProject(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.GetProjectInput{
		ProjectID: argsGetString(req, "project_id"),
	}

	project, err := s.useCase.GetProject(ctx, input)
	if err != nil {
		return errorResult("get project", err)
	}

	return jsonResult(project)
}

func listTasksTool() mcplib.Tool {
	return mcplib.NewTool("multica_list_tasks",
		mcplib.WithDescription("List tasks in a project with optional filters for status, assignee, and text query."),
		mcplib.WithString("project_id",
			mcplib.Description("Project ID to filter tasks"),
		),
		mcplib.WithString("status",
			mcplib.Description("Filter by status: backlog, todo, in_progress, in_review, done, blocked, cancelled"),
		),
		mcplib.WithString("assignee",
			mcplib.Description("Filter by assignee ID"),
		),
		mcplib.WithString("query",
			mcplib.Description("Optional text search query"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum number of tasks to return (default 100)"),
		),
	)
}

func (s *Server) handleListTasks(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.ListTasksInput{
		ProjectID: argsGetString(req, "project_id"),
		Status:    argsGetStringPtr(req, "status"),
		Assignee:  argsGetStringPtr(req, "assignee"),
		Query:     argsGetStringPtr(req, "query"),
		Limit:     argsGetIntPtr(req, "limit"),
	}

	tasks, err := s.useCase.ListTasks(ctx, input)
	if err != nil {
		return errorResult("list tasks", err)
	}

	return jsonResult(tasks)
}

func getTaskTool() mcplib.Tool {
	return mcplib.NewTool("multica_get_task",
		mcplib.WithDescription("Get a task with its description, status, assignee, comments, and subtasks."),
		mcplib.WithString("task_id",
			mcplib.Description("Task ID or identifier (e.g. MUL-123)"),
			mcplib.Required(),
		),
	)
}

func (s *Server) handleGetTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.GetTaskInput{
		TaskID: argsGetString(req, "task_id"),
	}

	task, err := s.useCase.GetTask(ctx, input)
	if err != nil {
		return errorResult("get task", err)
	}

	return jsonResult(task)
}

func createTaskTool() mcplib.Tool {
	return mcplib.NewTool("multica_create_task",
		mcplib.WithDescription("Create a new task in a project."),
		mcplib.WithString("project_id",
			mcplib.Description("Project ID to create the task in"),
			mcplib.Required(),
		),
		mcplib.WithString("title",
			mcplib.Description("Task title"),
			mcplib.Required(),
		),
		mcplib.WithString("description",
			mcplib.Description("Task description (Markdown supported)"),
			mcplib.Required(),
		),
		mcplib.WithString("priority",
			mcplib.Description("Task priority: none, urgent, high, medium, low"),
		),
		mcplib.WithString("assignee",
			mcplib.Description("Assignee ID (member or agent)"),
		),
		mcplib.WithBoolean("dry_run",
			mcplib.Description("If true, validate without creating"),
		),
	)
}

func (s *Server) handleCreateTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.CreateTaskInput{
		ProjectID:   argsGetString(req, "project_id"),
		Title:       argsGetString(req, "title"),
		Description: argsGetString(req, "description"),
		Priority:    argsGetStringPtr(req, "priority"),
		Assignee:    argsGetStringPtr(req, "assignee"),
		DryRun:      argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateTask(ctx, input)
	if err != nil {
		return errorResult("create task", err)
	}

	return jsonResult(result)
}

func createSubtaskTool() mcplib.Tool {
	return mcplib.NewTool("multica_create_subtask",
		mcplib.WithDescription("Create a subtask under an existing task."),
		mcplib.WithString("parent_task_id",
			mcplib.Description("Parent task ID"),
			mcplib.Required(),
		),
		mcplib.WithString("title",
			mcplib.Description("Subtask title"),
			mcplib.Required(),
		),
		mcplib.WithString("description",
			mcplib.Description("Subtask description"),
			mcplib.Required(),
		),
		mcplib.WithString("assignee",
			mcplib.Description("Assignee ID"),
		),
		mcplib.WithBoolean("dry_run",
			mcplib.Description("If true, validate without creating"),
		),
	)
}

func (s *Server) handleCreateSubtask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.CreateSubtaskInput{
		ParentTaskID: argsGetString(req, "parent_task_id"),
		Title:        argsGetString(req, "title"),
		Description:  argsGetString(req, "description"),
		Assignee:     argsGetStringPtr(req, "assignee"),
		DryRun:       argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateSubtask(ctx, input)
	if err != nil {
		return errorResult("create subtask", err)
	}

	return jsonResult(result)
}

func updateTaskTool() mcplib.Tool {
	return mcplib.NewTool("multica_update_task",
		mcplib.WithDescription("Update a task's title, description, status, priority, or assignee."),
		mcplib.WithString("task_id",
			mcplib.Description("Task ID to update"),
			mcplib.Required(),
		),
		mcplib.WithString("title",
			mcplib.Description("New title"),
		),
		mcplib.WithString("description",
			mcplib.Description("New description"),
		),
		mcplib.WithString("status",
			mcplib.Description("New status: backlog, todo, in_progress, in_review, done, blocked, cancelled"),
		),
		mcplib.WithString("priority",
			mcplib.Description("New priority: none, urgent, high, medium, low"),
		),
		mcplib.WithString("assignee",
			mcplib.Description("New assignee ID"),
		),
		mcplib.WithBoolean("dry_run",
			mcplib.Description("If true, validate without updating"),
		),
	)
}

func (s *Server) handleUpdateTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.UpdateTaskInput{
		TaskID:      argsGetString(req, "task_id"),
		Title:       argsGetStringPtr(req, "title"),
		Description: argsGetStringPtr(req, "description"),
		Status:      argsGetStringPtr(req, "status"),
		Priority:    argsGetStringPtr(req, "priority"),
		Assignee:    argsGetStringPtr(req, "assignee"),
		DryRun:      argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.UpdateTask(ctx, input)
	if err != nil {
		return errorResult("update task", err)
	}

	return jsonResult(result)
}

func addCommentTool() mcplib.Tool {
	return mcplib.NewTool("multica_add_comment",
		mcplib.WithDescription("Add a comment to a task."),
		mcplib.WithString("task_id",
			mcplib.Description("Task ID"),
			mcplib.Required(),
		),
		mcplib.WithString("comment",
			mcplib.Description("Comment text (Markdown supported)"),
			mcplib.Required(),
		),
	)
}

func (s *Server) handleAddComment(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.AddCommentInput{
		TaskID:  argsGetString(req, "task_id"),
		Comment: argsGetString(req, "comment"),
	}

	result, err := s.useCase.AddComment(ctx, input)
	if err != nil {
		return errorResult("add comment", err)
	}

	return jsonResult(result)
}

func listAgentsTool() mcplib.Tool {
	return mcplib.NewTool("multica_list_agents",
		mcplib.WithDescription("List available agents in the workspace."),
		mcplib.WithString("project_id",
			mcplib.Description("Optional project ID to filter agents"),
		),
	)
}

func (s *Server) handleListAgents(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.ListAgentsInput{
		ProjectID: argsGetStringPtr(req, "project_id"),
	}

	agents, err := s.useCase.ListAgents(ctx, input)
	if err != nil {
		return errorResult("list agents", err)
	}

	return jsonResult(agents)
}

func assignTaskTool() mcplib.Tool {
	return mcplib.NewTool("multica_assign_task",
		mcplib.WithDescription("Assign a task to a person or agent."),
		mcplib.WithString("task_id",
			mcplib.Description("Task ID"),
			mcplib.Required(),
		),
		mcplib.WithString("assignee_id",
			mcplib.Description("ID of the member or agent to assign"),
			mcplib.Required(),
		),
	)
}

func (s *Server) handleAssignTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.AssignTaskInput{
		TaskID:     argsGetString(req, "task_id"),
		AssigneeID: argsGetString(req, "assignee_id"),
	}

	result, err := s.useCase.AssignTask(ctx, input)
	if err != nil {
		return errorResult("assign task", err)
	}

	return jsonResult(result)
}

func searchTasksTool() mcplib.Tool {
	return mcplib.NewTool("multica_search_tasks",
		mcplib.WithDescription("Search tasks by text across titles, descriptions, and comments."),
		mcplib.WithString("query",
			mcplib.Description("Search query text"),
			mcplib.Required(),
		),
		mcplib.WithString("project_id",
			mcplib.Description("Optional project ID to scope the search"),
		),
		mcplib.WithString("status",
			mcplib.Description("Optional status filter"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum results to return"),
		),
	)
}

func (s *Server) handleSearchTasks(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.SearchTasksInput{
		Query:     argsGetString(req, "query"),
		ProjectID: argsGetStringPtr(req, "project_id"),
		Status:    argsGetStringPtr(req, "status"),
		Limit:     argsGetIntPtr(req, "limit"),
	}

	tasks, err := s.useCase.SearchTasks(ctx, input)
	if err != nil {
		return errorResult("search tasks", err)
	}

	return jsonResult(tasks)
}

func planTaskBreakdownTool() mcplib.Tool {
	return mcplib.NewTool("multica_plan_task_breakdown",
		mcplib.WithDescription("Generate a structured plan of subtasks based on a task description. Does NOT create any tasks."),
		mcplib.WithString("title",
			mcplib.Description("Task title"),
			mcplib.Required(),
		),
		mcplib.WithString("description",
			mcplib.Description("Task description"),
			mcplib.Required(),
		),
		mcplib.WithString("project_context",
			mcplib.Description("Optional project context for more relevant breakdown"),
		),
	)
}

func (s *Server) handlePlanTaskBreakdown(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	input := domain.PlanTaskBreakdownInput{
		Title:          argsGetString(req, "title"),
		Description:    argsGetString(req, "description"),
		ProjectContext: argsGetStringPtr(req, "project_context"),
	}

	result, err := s.useCase.PlanTaskBreakdown(ctx, input)
	if err != nil {
		return errorResult("plan task breakdown", err)
	}

	return jsonResult(result)
}

func createTaskWithSubtasksTool() mcplib.Tool {
	return mcplib.NewTool("multica_create_task_with_subtasks",
		mcplib.WithDescription("Create a parent task with subtasks in a single operation."),
		mcplib.WithString("project_id",
			mcplib.Description("Project ID"),
			mcplib.Required(),
		),
		mcplib.WithString("title",
			mcplib.Description("Parent task title"),
			mcplib.Required(),
		),
		mcplib.WithString("description",
			mcplib.Description("Parent task description"),
			mcplib.Required(),
		),
		mcplib.WithArray("subtasks",
			mcplib.Description("Array of subtask objects with title and description"),
			mcplib.Required(),
			mcplib.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string", "description": "Subtask title"},
					"description": map[string]any{"type": "string", "description": "Subtask description"},
				},
				"required": []any{"title", "description"},
			}),
		),
		mcplib.WithString("assignee",
			mcplib.Description("Assignee ID for parent and subtasks"),
		),
		mcplib.WithBoolean("dry_run",
			mcplib.Description("If true, validate without creating"),
		),
	)
}

func (s *Server) handleCreateTaskWithSubtasks(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()
	subtasksRaw, ok := args["subtasks"]
	if !ok {
		return errorResult("create task with subtasks", fmt.Errorf("subtasks is required"))
	}

	subtaskDefs, err := parseSubtaskDefs(subtasksRaw)
	if err != nil {
		return errorResult("create task with subtasks", err)
	}

	input := domain.CreateTaskWithSubtasksInput{
		ProjectID:   argsGetString(req, "project_id"),
		Title:       argsGetString(req, "title"),
		Description: argsGetString(req, "description"),
		Subtasks:    subtaskDefs,
		Assignee:    argsGetStringPtr(req, "assignee"),
		DryRun:      argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateTaskWithSubtasks(ctx, input)
	if err != nil {
		return errorResult("create task with subtasks", err)
	}

	return jsonResult(result)
}

func parseSubtaskDefs(raw any) ([]domain.SubtaskDef, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("parse subtasks: %w", err)
	}
	var defs []domain.SubtaskDef
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, fmt.Errorf("parse subtasks: %w", err)
	}
	return defs, nil
}

func argsGetString(req mcplib.CallToolRequest, key string) string {
	args := req.GetArguments()
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func argsGetStringPtr(req mcplib.CallToolRequest, key string) *string {
	args := req.GetArguments()
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return &s
}

func argsGetIntPtr(req mcplib.CallToolRequest, key string) *int {
	args := req.GetArguments()
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case json.Number:
		i, err := strconv.Atoi(string(n))
		if err != nil {
			return nil
		}
		return &i
	case int64:
		i := int(n)
		return &i
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return nil
		}
		return &i
	}
	return nil
}

func argsGetBool(req mcplib.CallToolRequest, key string) bool {
	args := req.GetArguments()
	if args == nil {
		return false
	}
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

func jsonResult(data any) (*mcplib.CallToolResult, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.Error("failed to marshal result", "error", err)
		return mcplib.NewToolResultError("internal error: failed to format result"), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func errorResult(op string, err error) (*mcplib.CallToolResult, error) {
	slog.Warn("tool error", "operation", op, "error", err)
	return mcplib.NewToolResultError(fmt.Sprintf("%s: %v", op, err)), nil
}
