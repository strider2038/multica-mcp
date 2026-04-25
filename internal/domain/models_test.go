package domain

import "testing"

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		valid  bool
	}{
		{TaskStatusBacklog, true},
		{TaskStatusTodo, true},
		{TaskStatusInProgress, true},
		{TaskStatusInReview, true},
		{TaskStatusDone, true},
		{TaskStatusBlocked, true},
		{TaskStatusCancelled, true},
		{TaskStatus("unknown"), false},
		{TaskStatus(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("TaskStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		priority Priority
		valid    bool
	}{
		{PriorityNone, true},
		{PriorityUrgent, true},
		{PriorityHigh, true},
		{PriorityMedium, true},
		{PriorityLow, true},
		{Priority("unknown"), false},
		{Priority(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.valid {
				t.Errorf("Priority(%q).IsValid() = %v, want %v", tt.priority, got, tt.valid)
			}
		})
	}
}
