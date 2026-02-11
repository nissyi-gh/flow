package ui

import (
	"fmt"

	"github.com/nissyi-gh/flow/internal/model"
)

// TaskItem wraps model.Task to satisfy the list.DefaultItem interface.
type TaskItem struct {
	Task model.Task
	// Prefix holds the tree-drawing characters, e.g. "│  └─ "
	Prefix string
	// DescPrefix holds the continuation lines for the description row
	DescPrefix string
}

func (i TaskItem) Title() string {
	check := "[ ]"
	if i.Task.Completed {
		check = "[x]"
	}
	todayMark := ""
	if i.Task.IsToday() {
		todayMark = "📌 "
	}
	return fmt.Sprintf("%s%s %s%s  created_at: %s", i.Prefix, check, todayMark, i.Task.Title, i.Task.CreatedAt.Format("2006-01-02 15:04"))
}

func (i TaskItem) Description() string {
	return ""
}

func (i TaskItem) FilterValue() string {
	return i.Task.Title
}
