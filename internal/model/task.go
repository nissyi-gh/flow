package model

import "time"

// Task represents a single task stored in the database.
type Task struct {
	ID        int
	Title     string
	Completed bool
	ParentID  *int
	CreatedAt time.Time
}
