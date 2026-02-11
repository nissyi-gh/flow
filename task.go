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
	CreatedAt time.Time
}

// taskItem wraps Task to satisfy the list.DefaultItem interface.
type taskItem struct {
	task Task
}

func (i taskItem) Title() string {
	check := "[ ]"
	if i.task.Completed {
		check = "[x]"
	}
	return fmt.Sprintf("%s %s", check, i.task.Title)
}

func (i taskItem) Description() string {
	return i.task.CreatedAt.Format("2006-01-02 15:04")
}

func (i taskItem) FilterValue() string {
	return i.task.Title
}
