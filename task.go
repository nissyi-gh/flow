package main

import (
	"fmt"
	"strings"
	"time"
)

// Task represents a single task stored in the database.
type Task struct {
	ID        int
	Title     string
	Completed bool
	ParentID  *int
	CreatedAt time.Time
}

// taskItem wraps Task to satisfy the list.DefaultItem interface.
type taskItem struct {
	task  Task
	depth int
}

func (i taskItem) Title() string {
	check := "[ ]"
	if i.task.Completed {
		check = "[x]"
	}
	indent := strings.Repeat("  ", i.depth)
	return fmt.Sprintf("%s%s %s", indent, check, i.task.Title)
}

func (i taskItem) Description() string {
	indent := strings.Repeat("  ", i.depth)
	return indent + i.task.CreatedAt.Format("2006-01-02 15:04")
}

func (i taskItem) FilterValue() string {
	return i.task.Title
}
