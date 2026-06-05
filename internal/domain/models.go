package domain

type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusInReview   TaskStatus = "in_review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

var ValidTaskStatuses = []TaskStatus{
	TaskStatusBacklog,
	TaskStatusTodo,
	TaskStatusInProgress,
	TaskStatusInReview,
	TaskStatusDone,
	TaskStatusBlocked,
	TaskStatusCancelled,
}

func (s TaskStatus) IsValid() bool {
	for _, v := range ValidTaskStatuses {
		if s == v {
			return true
		}
	}
	return false
}

type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

var ValidPriorities = []Priority{
	PriorityNone,
	PriorityUrgent,
	PriorityHigh,
	PriorityMedium,
	PriorityLow,
}

func (p Priority) IsValid() bool {
	for _, v := range ValidPriorities {
		if p == v {
			return true
		}
	}
	return false
}

type Project struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	LeadType    *string `json:"lead_type"`
	LeadID      *string `json:"lead_id"`
	IssueCount  int64   `json:"issue_count"`
	DoneCount   int64   `json:"done_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type Task struct {
	ID            string    `json:"id"`
	WorkspaceID   string    `json:"workspace_id"`
	Number        int32     `json:"number"`
	Identifier    string    `json:"identifier"`
	Title         string    `json:"title"`
	Description   *string   `json:"description"`
	Status        string    `json:"status"`
	Priority      string    `json:"priority"`
	AssigneeType  *string   `json:"assignee_type"`
	AssigneeID    *string   `json:"assignee_id"`
	CreatorType   string    `json:"creator_type"`
	CreatorID     string    `json:"creator_id"`
	ParentIssueID *string          `json:"parent_issue_id"`
	ProjectID     *string          `json:"project_id"`
	Position      float64          `json:"position"`
	StartDate     *string          `json:"start_date"`
	DueDate       *string          `json:"due_date"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	Reactions     []any     `json:"reactions,omitempty"`
	Attachments   []any     `json:"attachments,omitempty"`
	Comments      []Comment `json:"comments,omitempty"`
	Subtasks      []Task    `json:"subtasks,omitempty"`
}

type Comment struct {
	ID         string  `json:"id"`
	IssueID    string  `json:"issue_id"`
	AuthorType string  `json:"author_type"`
	AuthorID   string  `json:"author_id"`
	Content    string  `json:"content"`
	Type       string  `json:"type"`
	ParentID   *string `json:"parent_id"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type Agent struct {
	ID                 string  `json:"id"`
	WorkspaceID        string  `json:"workspace_id"`
	RuntimeID          string  `json:"runtime_id"`
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Instructions       string  `json:"instructions"`
	AvatarURL          *string `json:"avatar_url"`
	RuntimeMode        string  `json:"runtime_mode"`
	Visibility         string  `json:"visibility"`
	Status             string  `json:"status"`
	MaxConcurrentTasks int32   `json:"max_concurrent_tasks"`
	Model              string  `json:"model"`
	OwnerID            *string `json:"owner_id"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	ArchivedAt         *string `json:"archived_at"`
}

type Workspace struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IssuePrefix string  `json:"issue_prefix"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type ListProjectsInput struct {
	Query string
}

type GetProjectInput struct {
	ProjectID string
}

type ListTasksInput struct {
	ProjectID string
	Status    *string
	Assignee  *string
	Query     *string
	Limit     *int
}

type GetTaskInput struct {
	TaskID string
}

type AssigneeType string

const (
	AssigneeTypeMember AssigneeType = "member"
	AssigneeTypeAgent  AssigneeType = "agent"
	AssigneeTypeSquad  AssigneeType = "squad"
)

func (t AssigneeType) IsValid() bool {
	switch t {
	case AssigneeTypeMember, AssigneeTypeAgent, AssigneeTypeSquad:
		return true
	default:
		return false
	}
}

type CreateTaskInput struct {
	ProjectID      string
	ParentIssueID  *string
	Title          string
	Description    string
	Priority       *string
	Labels         []string
	Assignee       *string
	AssigneeType   *string
	IdempotencyKey *string
	DryRun         bool
}

type CreateSubtaskInput struct {
	ParentTaskID string
	Title        string
	Description  string
	Assignee     *string
	AssigneeType *string
	DryRun       bool
}

type UpdateTaskInput struct {
	TaskID       string
	Title        *string
	Description  *string
	Status       *string
	Priority     *string
	Labels       []string
	Assignee     *string
	AssigneeType *string
	DryRun       bool
}

type AddCommentInput struct {
	TaskID  string
	Comment string
}

type ListAgentsInput struct {
	ProjectID *string
}

type AssignTaskInput struct {
	TaskID       string
	AssigneeID   string
	AssigneeType *string
}

type SearchTasksInput struct {
	Query     string
	ProjectID *string
	Status    *string
	Limit     *int
}

type PlanTaskBreakdownInput struct {
	Title          string
	Description    string
	ProjectContext *string
}

type PlannedSubtask struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Dependencies       []string `json:"dependencies,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type PlanTaskBreakdownOutput struct {
	Subtasks []PlannedSubtask `json:"subtasks"`
}

type CreateTaskWithSubtasksInput struct {
	ProjectID      string
	Title          string
	Description    string
	Subtasks       []SubtaskDef
	Assignee       *string
	AssigneeType   *string
	IdempotencyKey *string
	DryRun         bool
}

type SubtaskDef struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateTaskResult struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`
}

type CreateTaskWithSubtasksResult struct {
	Parent   CreateTaskResult   `json:"parent"`
	Subtasks []CreateTaskResult `json:"subtasks"`
}
