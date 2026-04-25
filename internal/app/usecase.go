package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/strider2038/multica-mcp/internal/domain"
	"github.com/strider2038/multica-mcp/internal/multica"
)

type UseCase struct {
	client      *multica.Client
	workspaceID string
	readOnly    bool
}

func NewUseCase(client *multica.Client, workspaceID string, readOnly bool) *UseCase {
	return &UseCase{
		client:      client,
		workspaceID: workspaceID,
		readOnly:    readOnly,
	}
}

func (u *UseCase) checkReadOnly() error {
	if u.readOnly {
		return fmt.Errorf("server is in read-only mode: write operations are disabled")
	}
	return nil
}

func (u *UseCase) ListProjects(ctx context.Context, input domain.ListProjectsInput) ([]domain.Project, error) {
	if input.Query != "" {
		return u.client.SearchProjects(ctx, u.workspaceID, input.Query)
	}
	return u.client.ListProjects(ctx, u.workspaceID, "")
}

func (u *UseCase) GetProject(ctx context.Context, input domain.GetProjectInput) (*domain.Project, error) {
	return u.client.GetProject(ctx, u.workspaceID, input.ProjectID)
}

func (u *UseCase) ListTasks(ctx context.Context, input domain.ListTasksInput) ([]domain.Task, error) {
	if input.Query != nil && *input.Query != "" {
		return u.client.SearchIssues(ctx, u.workspaceID, *input.Query, &input.ProjectID, input.Status, input.Limit)
	}
	return u.client.ListTasks(ctx, u.workspaceID, input)
}

func (u *UseCase) GetTask(ctx context.Context, input domain.GetTaskInput) (*domain.Task, error) {
	task, err := u.client.GetTask(ctx, u.workspaceID, input.TaskID)
	if err != nil {
		return nil, err
	}

	comments, err := u.client.ListComments(ctx, u.workspaceID, input.TaskID)
	if err != nil {
		slog.Warn("failed to load comments for task", "task_id", input.TaskID, "error", err)
	} else {
		task.Comments = comments
	}

	children, err := u.client.ListChildIssues(ctx, u.workspaceID, input.TaskID)
	if err != nil {
		slog.Warn("failed to load subtasks for task", "task_id", input.TaskID, "error", err)
	} else {
		task.Subtasks = children
	}

	return task, nil
}

func (u *UseCase) CreateTask(ctx context.Context, input domain.CreateTaskInput) (*domain.CreateTaskResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	if input.DryRun {
		return &domain.CreateTaskResult{
			Title:      input.Title,
			Status:     "todo",
			Identifier: "(dry run)",
		}, nil
	}

	task, err := u.client.CreateTask(ctx, u.workspaceID, input)
	if err != nil {
		return nil, err
	}

	return &domain.CreateTaskResult{
		ID:         task.ID,
		Identifier: task.Identifier,
		Title:      task.Title,
		Status:     task.Status,
	}, nil
}

func (u *UseCase) CreateSubtask(ctx context.Context, input domain.CreateSubtaskInput) (*domain.CreateTaskResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	if input.DryRun {
		return &domain.CreateTaskResult{
			Title:      input.Title,
			Status:     "todo",
			Identifier: "(dry run)",
		}, nil
	}

	task, err := u.client.GetTask(ctx, u.workspaceID, input.ParentTaskID)
	if err != nil {
		return nil, fmt.Errorf("parent task not found: %w", err)
	}

	createInput := domain.CreateTaskInput{
		Title:       input.Title,
		Description: input.Description,
		Assignee:    input.Assignee,
	}
	if task.ProjectID != nil && *task.ProjectID != "" {
		createInput.ProjectID = *task.ProjectID
	}

	result, err := u.client.CreateTask(ctx, u.workspaceID, createInput)
	if err != nil {
		return nil, err
	}

	return &domain.CreateTaskResult{
		ID:         result.ID,
		Identifier: result.Identifier,
		Title:      result.Title,
		Status:     result.Status,
	}, nil
}

func (u *UseCase) UpdateTask(ctx context.Context, input domain.UpdateTaskInput) (*domain.CreateTaskResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	if input.Status != nil && !domain.TaskStatus(*input.Status).IsValid() {
		return nil, fmt.Errorf("invalid status %q; valid values: %s", *input.Status, validStatusList())
	}

	if input.DryRun {
		return &domain.CreateTaskResult{
			ID:     input.TaskID,
			Status: stringPtrValue(input.Status, "(unchanged)"),
			Title:  stringPtrValue(input.Title, "(unchanged)"),
		}, nil
	}

	task, err := u.client.UpdateTask(ctx, u.workspaceID, input.TaskID, input)
	if err != nil {
		return nil, err
	}

	return &domain.CreateTaskResult{
		ID:         task.ID,
		Identifier: task.Identifier,
		Title:      task.Title,
		Status:     task.Status,
	}, nil
}

func (u *UseCase) AddComment(ctx context.Context, input domain.AddCommentInput) (*domain.Comment, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}
	return u.client.CreateComment(ctx, u.workspaceID, input.TaskID, input.Comment)
}

func (u *UseCase) ListAgents(ctx context.Context, input domain.ListAgentsInput) ([]domain.Agent, error) {
	return u.client.ListAgents(ctx, u.workspaceID)
}

func (u *UseCase) AssignTask(ctx context.Context, input domain.AssignTaskInput) (*domain.CreateTaskResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	updateInput := domain.UpdateTaskInput{
		TaskID:   input.TaskID,
		Assignee: &input.AssigneeID,
	}

	task, err := u.client.UpdateTask(ctx, u.workspaceID, input.TaskID, updateInput)
	if err != nil {
		return nil, err
	}

	return &domain.CreateTaskResult{
		ID:         task.ID,
		Identifier: task.Identifier,
		Title:      task.Title,
		Status:     task.Status,
	}, nil
}

func (u *UseCase) SearchTasks(ctx context.Context, input domain.SearchTasksInput) ([]domain.Task, error) {
	return u.client.SearchIssues(ctx, u.workspaceID, input.Query, input.ProjectID, input.Status, input.Limit)
}

func (u *UseCase) PlanTaskBreakdown(ctx context.Context, input domain.PlanTaskBreakdownInput) (*domain.PlanTaskBreakdownOutput, error) {
	subtasks := planSubtasks(input.Title, input.Description)
	return &domain.PlanTaskBreakdownOutput{
		Subtasks: subtasks,
	}, nil
}

func (u *UseCase) CreateTaskWithSubtasks(ctx context.Context, input domain.CreateTaskWithSubtasksInput) (*domain.CreateTaskWithSubtasksResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	if input.DryRun {
		parent := domain.CreateTaskResult{
			Title:      input.Title,
			Status:     "todo",
			Identifier: "(dry run)",
		}
		subs := make([]domain.CreateTaskResult, len(input.Subtasks))
		for i, s := range input.Subtasks {
			subs[i] = domain.CreateTaskResult{
				Title:      s.Title,
				Status:     "todo",
				Identifier: "(dry run)",
			}
		}
		return &domain.CreateTaskWithSubtasksResult{
			Parent:   parent,
			Subtasks: subs,
		}, nil
	}

	parentInput := domain.CreateTaskInput{
		ProjectID:   input.ProjectID,
		Title:       input.Title,
		Description: input.Description,
		Assignee:    input.Assignee,
	}

	parent, err := u.client.CreateTask(ctx, u.workspaceID, parentInput)
	if err != nil {
		return nil, fmt.Errorf("create parent task: %w", err)
	}

	result := &domain.CreateTaskWithSubtasksResult{
		Parent: domain.CreateTaskResult{
			ID:         parent.ID,
			Identifier: parent.Identifier,
			Title:      parent.Title,
			Status:     parent.Status,
		},
		Subtasks: make([]domain.CreateTaskResult, 0, len(input.Subtasks)),
	}

	for _, sub := range input.Subtasks {
		subInput := domain.CreateTaskInput{
			ProjectID:   input.ProjectID,
			Title:       sub.Title,
			Description: sub.Description,
			Assignee:    input.Assignee,
		}

		subTask, err := u.client.CreateTask(ctx, u.workspaceID, subInput)
		if err != nil {
			slog.Warn("failed to create subtask", "title", sub.Title, "error", err)
			continue
		}

		updateInput := domain.UpdateTaskInput{
			TaskID: subTask.ID,
		}

		_, updateErr := u.client.UpdateTask(ctx, u.workspaceID, subTask.ID, updateInput)
		if updateErr != nil {
			slog.Warn("failed to set parent for subtask", "subtask_id", subTask.ID, "parent_id", parent.ID, "error", updateErr)
		}

		result.Subtasks = append(result.Subtasks, domain.CreateTaskResult{
			ID:         subTask.ID,
			Identifier: subTask.Identifier,
			Title:      subTask.Title,
			Status:     subTask.Status,
		})
	}

	return result, nil
}

func (u *UseCase) ResolveProjectReference(ctx context.Context, ref string) (string, error) {
	projects, err := u.client.ListProjects(ctx, u.workspaceID, "")
	if err != nil {
		return "", err
	}

	refLower := strings.ToLower(ref)
	for _, p := range projects {
		if strings.EqualFold(p.ID, ref) {
			return p.ID, nil
		}
		if strings.EqualFold(p.Title, ref) {
			return p.ID, nil
		}
		if strings.Contains(strings.ToLower(p.Title), refLower) {
			return p.ID, nil
		}
	}

	return "", fmt.Errorf("project %q not found", ref)
}

func planSubtasks(title, description string) []domain.PlannedSubtask {
	return []domain.PlannedSubtask{
		{
			Title:              "Analysis and requirements",
			Description:        fmt.Sprintf("Analyze requirements and define acceptance criteria for: %s", title),
			Dependencies:       []string{},
			AcceptanceCriteria: []string{"Requirements documented", "Acceptance criteria defined"},
		},
		{
			Title:              "Implementation",
			Description:        fmt.Sprintf("Implement the core functionality for: %s", title),
			Dependencies:       []string{"Analysis and requirements"},
			AcceptanceCriteria: []string{"Core functionality implemented", "Code follows project conventions"},
		},
		{
			Title:              "Testing",
			Description:        fmt.Sprintf("Write and run tests for: %s", title),
			Dependencies:       []string{"Implementation"},
			AcceptanceCriteria: []string{"Unit tests pass", "Integration tests pass"},
		},
		{
			Title:              "Documentation and cleanup",
			Description:        fmt.Sprintf("Document changes and clean up for: %s", title),
			Dependencies:       []string{"Testing"},
			AcceptanceCriteria: []string{"Documentation updated", "Code reviewed and cleaned"},
		},
	}
}

func validStatusList() string {
	statuses := make([]string, len(domain.ValidTaskStatuses))
	for i, s := range domain.ValidTaskStatuses {
		statuses[i] = string(s)
	}
	return strings.Join(statuses, ", ")
}

func stringPtrValue(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}
