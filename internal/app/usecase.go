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
	client   *multica.Client
	readOnly bool
}

func NewUseCase(client *multica.Client, readOnly bool) *UseCase {
	return &UseCase{
		client:   client,
		readOnly: readOnly,
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
		return u.client.SearchProjects(ctx, input.Query)
	}
	return u.client.ListProjects(ctx)
}

func (u *UseCase) GetProject(ctx context.Context, input domain.GetProjectInput) (*domain.Project, error) {
	return u.client.GetProject(ctx, input.ProjectID)
}

func (u *UseCase) ListTasks(ctx context.Context, input domain.ListTasksInput) ([]domain.Task, error) {
	if input.Query != nil && *input.Query != "" {
		return u.client.SearchIssues(ctx, *input.Query, &input.ProjectID, input.Status, input.Limit)
	}
	return u.client.ListTasks(ctx, input)
}

func (u *UseCase) GetTask(ctx context.Context, input domain.GetTaskInput) (*domain.Task, error) {
	task, err := u.client.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	comments, err := u.client.ListComments(ctx, input.TaskID)
	if err != nil {
		slog.Warn("failed to load comments for task", "task_id", input.TaskID, "error", err)
	} else {
		task.Comments = comments
	}

	children, err := u.client.ListChildIssues(ctx, input.TaskID)
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

	if input.Assignee != nil && *input.Assignee != "" {
		assigneeType, err := u.resolveAssigneeType(ctx, *input.Assignee, input.AssigneeType)
		if err != nil {
			return nil, err
		}
		input.AssigneeType = assigneeType
	}

	task, err := u.client.CreateTask(ctx, input)
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

	task, err := u.client.GetTask(ctx, input.ParentTaskID)
	if err != nil {
		return nil, fmt.Errorf("parent task not found: %w", err)
	}

	createInput := domain.CreateTaskInput{
		ParentIssueID: &task.ID,
		Title:         input.Title,
		Description:   input.Description,
		Assignee:      input.Assignee,
		AssigneeType:  input.AssigneeType,
		Stage:         input.Stage,
	}
	if task.ProjectID != nil && *task.ProjectID != "" {
		createInput.ProjectID = *task.ProjectID
	}

	if createInput.Assignee != nil && *createInput.Assignee != "" {
		assigneeType, err := u.resolveAssigneeType(ctx, *createInput.Assignee, createInput.AssigneeType)
		if err != nil {
			return nil, err
		}
		createInput.AssigneeType = assigneeType
	}

	result, err := u.client.CreateTask(ctx, createInput)
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

	if input.Assignee != nil && *input.Assignee != "" {
		assigneeType, err := u.resolveAssigneeType(ctx, *input.Assignee, input.AssigneeType)
		if err != nil {
			return nil, err
		}
		input.AssigneeType = assigneeType
	}

	task, err := u.client.UpdateTask(ctx, input.TaskID, input)
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
	return u.client.CreateComment(ctx, input)
}

func (u *UseCase) PreviewCommentTriggers(ctx context.Context, input domain.PreviewCommentTriggersInput) (*domain.CommentTriggerPreview, error) {
	return u.client.PreviewCommentTriggers(ctx, input)
}

func (u *UseCase) PreviewIssueTriggers(ctx context.Context, input domain.PreviewIssueTriggersInput) (*domain.IssueTriggerPreview, error) {
	return u.client.PreviewIssueTriggers(ctx, input)
}

func (u *UseCase) ListAgents(ctx context.Context, input domain.ListAgentsInput) ([]domain.Agent, error) {
	return u.client.ListAgents(ctx)
}

func (u *UseCase) AssignTask(ctx context.Context, input domain.AssignTaskInput) (*domain.CreateTaskResult, error) {
	if err := u.checkReadOnly(); err != nil {
		return nil, err
	}

	assigneeType, err := u.resolveAssigneeType(ctx, input.AssigneeID, input.AssigneeType)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateTaskInput{
		TaskID:       input.TaskID,
		Assignee:     &input.AssigneeID,
		AssigneeType: assigneeType,
	}

	task, err := u.client.UpdateTask(ctx, input.TaskID, updateInput)
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
	return u.client.SearchIssues(ctx, input.Query, input.ProjectID, input.Status, input.Limit)
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
		ProjectID:    input.ProjectID,
		Title:        input.Title,
		Description:  input.Description,
		Assignee:     input.Assignee,
		AssigneeType: input.AssigneeType,
	}

	if parentInput.Assignee != nil && *parentInput.Assignee != "" {
		assigneeType, err := u.resolveAssigneeType(ctx, *parentInput.Assignee, parentInput.AssigneeType)
		if err != nil {
			return nil, err
		}
		parentInput.AssigneeType = assigneeType
	}

	parent, err := u.client.CreateTask(ctx, parentInput)
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
			ProjectID:     input.ProjectID,
			ParentIssueID: &parent.ID,
			Title:         sub.Title,
			Description:   sub.Description,
			Assignee:      input.Assignee,
			AssigneeType:  parentInput.AssigneeType,
		}

		subTask, err := u.client.CreateTask(ctx, subInput)
		if err != nil {
			slog.Warn("failed to create subtask", "title", sub.Title, "error", err)
			continue
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
	projects, err := u.client.ListProjects(ctx)
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
