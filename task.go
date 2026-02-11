package main

import (
	"fmt"
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
	task Task
	// prefix holds the tree-drawing characters, e.g. "│  └─ "
	prefix string
	// descPrefix holds the continuation lines for the description row
	descPrefix string
}

func (i taskItem) Title() string {
	check := "[ ]"
	if i.task.Completed {
		check = "[x]"
	}
	return fmt.Sprintf("%s%s %s", i.prefix, check, i.task.Title)
}

func (i taskItem) Description() string {
	return i.descPrefix + i.task.CreatedAt.Format("2006-01-02 15:04")
}

func (i taskItem) FilterValue() string {
	return i.task.Title
}
