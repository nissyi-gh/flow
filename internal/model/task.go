package model

import "time"

// Tag represents a reusable label that can be attached to tasks.
type Tag struct {
	ID    int
	Name  string
	Color string // lipgloss 256-color code, e.g. "205", "39", "148"
}

// Task represents a single task stored in the database.
type Task struct {
	ID          int
	Title       string
	Description *string
	Completed   bool
	ParentID    *int
	CreatedAt   time.Time
	ScheduledOn *string
	DueDate     *string
	Tags        []Tag
}

// IsToday returns true if the task is scheduled for today.
func (t Task) IsToday() bool {
	if t.ScheduledOn == nil {
		return false
	}
	return *t.ScheduledOn == time.Now().Format("2006-01-02")
}

// IsDueToday returns true if the task's due date is today.
func (t Task) IsDueToday() bool {
	if t.DueDate == nil {
		return false
	}
	return *t.DueDate == time.Now().Format("2006-01-02")
}

// IsOverdue returns true if the task is past its due date and not completed.
func (t Task) IsOverdue() bool {
	if t.DueDate == nil || t.Completed {
		return false
	}
	return *t.DueDate < time.Now().Format("2006-01-02")
}
