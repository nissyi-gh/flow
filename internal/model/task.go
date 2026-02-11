package model

import "time"

// Task represents a single task stored in the database.
type Task struct {
	ID          int
	Title       string
	Completed   bool
	ParentID    *int
	CreatedAt   time.Time
	ScheduledOn *string
}

// IsToday returns true if the task is scheduled for today.
func (t Task) IsToday() bool {
	if t.ScheduledOn == nil {
		return false
	}
	return *t.ScheduledOn == time.Now().Format("2006-01-02")
}
