package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strider2038/multica-mcp/internal/app"
	"github.com/strider2038/multica-mcp/internal/domain"
	"github.com/strider2038/multica-mcp/internal/version"
)

type Server struct {
	mcpServer *mcp.Server
	useCase   *app.UseCase
}

func NewServer(useCase *app.UseCase, readOnly bool) *Server {
	s := &Server{
		useCase: useCase,
		mcpServer: mcp.NewServer(&mcp.Implementation{
			Name:    "multica-mcp",
			Version: version.Version,
		}, nil),
	}

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
	s.addTool(previewCommentTriggersTool(), s.handlePreviewCommentTriggers)
	s.addTool(previewIssueTriggersTool(), s.handlePreviewIssueTriggers)

	if !readOnly {
		s.addTool(createTaskTool(), s.handleCreateTask)
		s.addTool(createSubtaskTool(), s.handleCreateSubtask)
		s.addTool(updateTaskTool(), s.handleUpdateTask)
		s.addTool(addCommentTool(), s.handleAddComment)
		s.addTool(assignTaskTool(), s.handleAssignTask)
		s.addTool(createTaskWithSubtasksTool(), s.handleCreateTaskWithSubtasks)
	}
}

func (s *Server) addTool(tool *mcp.Tool, handler mcp.ToolHandler) {
	s.mcpServer.AddTool(tool, handler)
}

func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcpServer
}

func listProjectsTool() *mcp.Tool {
	return newTool("multica_list_projects", "List projects in the Multica workspace. Optionally filter by name query.", properties(
		stringProp("query", "Optional search query to filter projects by name"),
	), nil)
}

func (s *Server) handleListProjects(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.ListProjectsInput{Query: argsGetString(req, "query")}

	projects, err := s.useCase.ListProjects(ctx, input)
	if err != nil {
		return errorResult("list projects", err), nil
	}

	return jsonResult(projects), nil
}

func getProjectTool() *mcp.Tool {
	return newTool("multica_get_project", "Get detailed information about a specific project by ID.", properties(
		stringProp("project_id", "Project ID"),
	), []string{"project_id"})
}

func (s *Server) handleGetProject(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.GetProjectInput{ProjectID: argsGetString(req, "project_id")}

	project, err := s.useCase.GetProject(ctx, input)
	if err != nil {
		return errorResult("get project", err), nil
	}

	return jsonResult(project), nil
}

func listTasksTool() *mcp.Tool {
	return newTool("multica_list_tasks", "List tasks in a project with optional filters for status, assignee, and text query.", properties(
		stringProp("project_id", "Project ID to filter tasks"),
		stringProp("status", "Filter by status: backlog, todo, in_progress, in_review, done, blocked, cancelled"),
		stringProp("assignee", "Filter by assignee ID"),
		stringProp("query", "Optional text search query"),
		numberProp("limit", "Maximum number of tasks to return (default 100)"),
	), nil)
}

func (s *Server) handleListTasks(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.ListTasksInput{
		ProjectID: argsGetString(req, "project_id"),
		Status:    argsGetStringPtr(req, "status"),
		Assignee:  argsGetStringPtr(req, "assignee"),
		Query:     argsGetStringPtr(req, "query"),
		Limit:     argsGetIntPtr(req, "limit"),
	}

	tasks, err := s.useCase.ListTasks(ctx, input)
	if err != nil {
		return errorResult("list tasks", err), nil
	}

	return jsonResult(tasks), nil
}

func getTaskTool() *mcp.Tool {
	return newTool("multica_get_task", "Get a task with its description, status, assignee, comments, and subtasks.", properties(
		stringProp("task_id", "Task ID or identifier (e.g. MUL-123)"),
	), []string{"task_id"})
}

func (s *Server) handleGetTask(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.GetTaskInput{TaskID: argsGetString(req, "task_id")}

	task, err := s.useCase.GetTask(ctx, input)
	if err != nil {
		return errorResult("get task", err), nil
	}

	return jsonResult(task), nil
}

func createTaskTool() *mcp.Tool {
	return newTool("multica_create_task", "Create a new task in a project.", properties(
		stringProp("project_id", "Project ID to create the task in"),
		stringProp("title", "Task title"),
		stringProp("description", "Task description (Markdown supported)"),
		stringProp("priority", "Task priority: none, urgent, high, medium, low"),
		stringProp("assignee", "Assignee ID (member, agent, or squad)"),
		stringProp("assignee_type", "Assignee type: member, agent, or squad (inferred from agents list when omitted)"),
		numberProp("stage", "Optional ordered stage (>= 1) for sub-issue barrier grouping under a parent"),
		arrayProp("labels", "Issue label IDs to attach on create"),
		booleanProp("dry_run", "If true, validate without creating"),
	), []string{"project_id", "title", "description"})
}

func (s *Server) handleCreateTask(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.CreateTaskInput{
		ProjectID:    argsGetString(req, "project_id"),
		Title:        argsGetString(req, "title"),
		Description:  argsGetString(req, "description"),
		Priority:     argsGetStringPtr(req, "priority"),
		Assignee:     argsGetStringPtr(req, "assignee"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
		Stage:        argsGetIntPtr(req, "stage"),
		Labels:       argsGetStringSlice(req, "labels"),
		DryRun:       argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateTask(ctx, input)
	if err != nil {
		return errorResult("create task", err), nil
	}

	return jsonResult(result), nil
}

func createSubtaskTool() *mcp.Tool {
	return newTool("multica_create_subtask", "Create a subtask under an existing task.", properties(
		stringProp("parent_task_id", "Parent task ID"),
		stringProp("title", "Subtask title"),
		stringProp("description", "Subtask description"),
		stringProp("assignee", "Assignee ID"),
		stringProp("assignee_type", "Assignee type: member, agent, or squad"),
		numberProp("stage", "Optional ordered stage (>= 1) for barrier grouping among sibling subtasks"),
		booleanProp("dry_run", "If true, validate without creating"),
	), []string{"parent_task_id", "title", "description"})
}

func (s *Server) handleCreateSubtask(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.CreateSubtaskInput{
		ParentTaskID: argsGetString(req, "parent_task_id"),
		Title:        argsGetString(req, "title"),
		Description:  argsGetString(req, "description"),
		Assignee:     argsGetStringPtr(req, "assignee"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
		Stage:        argsGetIntPtr(req, "stage"),
		DryRun:       argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateSubtask(ctx, input)
	if err != nil {
		return errorResult("create subtask", err), nil
	}

	return jsonResult(result), nil
}

func updateTaskTool() *mcp.Tool {
	return newTool("multica_update_task", "Update a task's title, description, status, priority, assignee, or stage. Use suppress_run to apply assignee/status changes without starting an agent run, and handoff_note to inject context when a run starts.", properties(
		stringProp("task_id", "Task ID to update"),
		stringProp("title", "New title"),
		stringProp("description", "New description"),
		stringProp("status", "New status: backlog, todo, in_progress, in_review, done, blocked, cancelled"),
		stringProp("priority", "New priority: none, urgent, high, medium, low"),
		stringProp("assignee", "New assignee ID. Pass an empty string to unassign."),
		stringProp("assignee_type", "Assignee type: member, agent, or squad"),
		numberProp("stage", "Ordered stage (>= 1) for sub-issue barrier grouping"),
		booleanProp("clear_stage", "If true, remove the task from its stage (unstage)"),
		booleanProp("suppress_run", "If true, apply changes without enqueueing an agent run"),
		stringProp("handoff_note", "Optional handoff instruction injected when an agent run starts"),
		booleanProp("dry_run", "If true, validate without updating"),
	), []string{"task_id"})
}

func (s *Server) handleUpdateTask(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.UpdateTaskInput{
		TaskID:      argsGetString(req, "task_id"),
		Title:       argsGetStringPtr(req, "title"),
		Description: argsGetStringPtr(req, "description"),
		Status:      argsGetStringPtr(req, "status"),
		Priority:    argsGetStringPtr(req, "priority"),
		Assignee:     argsGetOptionalStringPtr(req, "assignee"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
		Stage:        argsGetIntPtr(req, "stage"),
		ClearStage:   argsGetBool(req, "clear_stage"),
		SuppressRun:  argsGetBool(req, "suppress_run"),
		HandoffNote:  argsGetString(req, "handoff_note"),
		DryRun:       argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.UpdateTask(ctx, input)
	if err != nil {
		return errorResult("update task", err), nil
	}

	return jsonResult(result), nil
}

func addCommentTool() *mcp.Tool {
	return newTool("multica_add_comment", "Add a comment to a task. Prefix the comment with /note to leave a human-only note that does not trigger agents. Use suppress_agent_ids to skip specific agents while still triggering others.", properties(
		stringProp("task_id", "Task ID"),
		stringProp("comment", "Comment text (Markdown supported). Prefix with /note to skip all agent triggers."),
		stringProp("parent_id", "Optional parent comment ID for thread replies"),
		arrayProp("suppress_agent_ids", "Optional agent IDs to exclude from comment-triggered runs"),
	), []string{"task_id", "comment"})
}

func (s *Server) handleAddComment(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.AddCommentInput{
		TaskID:           argsGetString(req, "task_id"),
		Comment:          argsGetString(req, "comment"),
		ParentID:         argsGetOptionalStringPtr(req, "parent_id"),
		SuppressAgentIDs: argsGetStringSlice(req, "suppress_agent_ids"),
	}

	result, err := s.useCase.AddComment(ctx, input)
	if err != nil {
		return errorResult("add comment", err), nil
	}

	return jsonResult(result), nil
}

func previewCommentTriggersTool() *mcp.Tool {
	return newTool("multica_preview_comment_triggers", "Preview which agents a comment would trigger before posting it.", properties(
		stringProp("task_id", "Task ID"),
		stringProp("content", "Comment text to evaluate"),
		stringProp("parent_id", "Optional parent comment ID for thread replies"),
	), []string{"task_id", "content"})
}

func (s *Server) handlePreviewCommentTriggers(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.PreviewCommentTriggersInput{
		TaskID:   argsGetString(req, "task_id"),
		Content:  argsGetString(req, "content"),
		ParentID: argsGetOptionalStringPtr(req, "parent_id"),
	}

	result, err := s.useCase.PreviewCommentTriggers(ctx, input)
	if err != nil {
		return errorResult("preview comment triggers", err), nil
	}

	return jsonResult(result), nil
}

func previewIssueTriggersTool() *mcp.Tool {
	return newTool("multica_preview_issue_triggers", "Preview which agent runs would start for a prospective issue create or update (assignee/status changes).", properties(
		arrayProp("issue_ids", "Issue IDs to evaluate for updates"),
		booleanProp("is_create", "If true, preview a not-yet-created issue using assignee/status fields"),
		stringProp("assignee_type", "Prospective assignee type: member, agent, or squad"),
		stringProp("assignee_id", "Prospective assignee ID"),
		stringProp("status", "Prospective status for create or update preview"),
	), nil)
}

func (s *Server) handlePreviewIssueTriggers(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.PreviewIssueTriggersInput{
		IssueIDs:     argsGetStringSlice(req, "issue_ids"),
		IsCreate:     argsGetBool(req, "is_create"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
		AssigneeID:   argsGetStringPtr(req, "assignee_id"),
		Status:       argsGetStringPtr(req, "status"),
	}

	result, err := s.useCase.PreviewIssueTriggers(ctx, input)
	if err != nil {
		return errorResult("preview issue triggers", err), nil
	}

	return jsonResult(result), nil
}

func listAgentsTool() *mcp.Tool {
	return newTool("multica_list_agents", "List available agents in the workspace.", properties(
		stringProp("project_id", "Optional project ID to filter agents"),
	), nil)
}

func (s *Server) handleListAgents(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.ListAgentsInput{ProjectID: argsGetStringPtr(req, "project_id")}

	agents, err := s.useCase.ListAgents(ctx, input)
	if err != nil {
		return errorResult("list agents", err), nil
	}

	return jsonResult(agents), nil
}

func assignTaskTool() *mcp.Tool {
	return newTool("multica_assign_task", "Assign a task to a person or agent.", properties(
		stringProp("task_id", "Task ID"),
		stringProp("assignee_id", "ID of the member, agent, or squad to assign"),
		stringProp("assignee_type", "Assignee type: member, agent, or squad"),
	), []string{"task_id", "assignee_id"})
}

func (s *Server) handleAssignTask(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.AssignTaskInput{
		TaskID:       argsGetString(req, "task_id"),
		AssigneeID:   argsGetString(req, "assignee_id"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
	}

	result, err := s.useCase.AssignTask(ctx, input)
	if err != nil {
		return errorResult("assign task", err), nil
	}

	return jsonResult(result), nil
}

func searchTasksTool() *mcp.Tool {
	return newTool("multica_search_tasks", "Search tasks by text across titles, descriptions, and comments.", properties(
		stringProp("query", "Search query text"),
		stringProp("project_id", "Optional project ID to scope the search"),
		stringProp("status", "Optional status filter"),
		numberProp("limit", "Maximum results to return"),
	), []string{"query"})
}

func (s *Server) handleSearchTasks(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.SearchTasksInput{
		Query:     argsGetString(req, "query"),
		ProjectID: argsGetStringPtr(req, "project_id"),
		Status:    argsGetStringPtr(req, "status"),
		Limit:     argsGetIntPtr(req, "limit"),
	}

	tasks, err := s.useCase.SearchTasks(ctx, input)
	if err != nil {
		return errorResult("search tasks", err), nil
	}

	return jsonResult(tasks), nil
}

func planTaskBreakdownTool() *mcp.Tool {
	return newTool("multica_plan_task_breakdown", "Generate a structured plan of subtasks based on a task description. Does NOT create any tasks.", properties(
		stringProp("title", "Task title"),
		stringProp("description", "Task description"),
		stringProp("project_context", "Optional project context for more relevant breakdown"),
	), []string{"title", "description"})
}

func (s *Server) handlePlanTaskBreakdown(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := domain.PlanTaskBreakdownInput{
		Title:          argsGetString(req, "title"),
		Description:    argsGetString(req, "description"),
		ProjectContext: argsGetStringPtr(req, "project_context"),
	}

	result, err := s.useCase.PlanTaskBreakdown(ctx, input)
	if err != nil {
		return errorResult("plan task breakdown", err), nil
	}

	return jsonResult(result), nil
}

func createTaskWithSubtasksTool() *mcp.Tool {
	subtaskSchema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":       map[string]any{"type": "string", "description": "Subtask title"},
				"description": map[string]any{"type": "string", "description": "Subtask description"},
			},
			"required": []string{"title", "description"},
		},
	}

	props := properties(
		stringProp("project_id", "Project ID"),
		stringProp("title", "Parent task title"),
		stringProp("description", "Parent task description"),
		property{Name: "subtasks", Schema: subtaskSchema},
		stringProp("assignee", "Assignee ID for parent and subtasks"),
		stringProp("assignee_type", "Assignee type: member, agent, or squad"),
		booleanProp("dry_run", "If true, validate without creating"),
	)
	return newTool("multica_create_task_with_subtasks", "Create a parent task with subtasks in a single operation.", props, []string{"project_id", "title", "description", "subtasks"})
}

func (s *Server) handleCreateTaskWithSubtasks(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(req)
	subtasksRaw, ok := args["subtasks"]
	if !ok {
		return errorResult("create task with subtasks", fmt.Errorf("subtasks is required")), nil
	}

	subtaskDefs, err := parseSubtaskDefs(subtasksRaw)
	if err != nil {
		return errorResult("create task with subtasks", err), nil
	}

	input := domain.CreateTaskWithSubtasksInput{
		ProjectID:   argsGetString(req, "project_id"),
		Title:       argsGetString(req, "title"),
		Description: argsGetString(req, "description"),
		Subtasks:    subtaskDefs,
		Assignee:     argsGetStringPtr(req, "assignee"),
		AssigneeType: argsGetStringPtr(req, "assignee_type"),
		DryRun:       argsGetBool(req, "dry_run"),
	}

	result, err := s.useCase.CreateTaskWithSubtasks(ctx, input)
	if err != nil {
		return errorResult("create task with subtasks", err), nil
	}

	return jsonResult(result), nil
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

type property struct {
	Name   string
	Schema map[string]any
}

func newTool(name, description string, props map[string]any, required []string) *mcp.Tool {
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return &mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}
}

func properties(props ...property) map[string]any {
	result := make(map[string]any, len(props))
	for _, prop := range props {
		result[prop.Name] = prop.Schema
	}
	return result
}

func stringProp(name, description string) property {
	return property{Name: name, Schema: map[string]any{"type": "string", "description": description}}
}

func numberProp(name, description string) property {
	return property{Name: name, Schema: map[string]any{"type": "number", "description": description}}
}

func booleanProp(name, description string) property {
	return property{Name: name, Schema: map[string]any{"type": "boolean", "description": description}}
}

func arrayProp(name, description string) property {
	return property{Name: name, Schema: map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": description,
	}}
}

func requestArgs(req *mcp.CallToolRequest) map[string]any {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return nil
	}

	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		slog.Warn("failed to decode tool arguments", "error", err)
		return nil
	}
	return args
}

func argsGetString(req *mcp.CallToolRequest, key string) string {
	args := requestArgs(req)
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

func argsGetStringPtr(req *mcp.CallToolRequest, key string) *string {
	s := argsGetOptionalStringPtr(req, key)
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func argsGetOptionalStringPtr(req *mcp.CallToolRequest, key string) *string {
	args := requestArgs(req)
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

func argsGetIntPtr(req *mcp.CallToolRequest, key string) *int {
	args := requestArgs(req)
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

func argsGetBool(req *mcp.CallToolRequest, key string) bool {
	args := requestArgs(req)
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

func argsGetStringSlice(req *mcp.CallToolRequest, key string) []string {
	args := requestArgs(req)
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch items := v.(type) {
	case []string:
		return items
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			s, ok := item.(string)
			if ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func jsonResult(data any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.Error("failed to marshal result", "error", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "internal error: failed to format result"}},
			IsError: true,
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func errorResult(op string, err error) *mcp.CallToolResult {
	slog.Warn("tool error", "operation", op, "error", err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("%s: %v", op, err)}},
		IsError: true,
	}
}
