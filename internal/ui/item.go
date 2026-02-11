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
	dueMark := ""
	if i.Task.IsOverdue() {
		dueMark = "⚠️ "
	} else if i.Task.IsDueToday() {
		dueMark = "📅 "
	}
	return fmt.Sprintf("%s%s %s%s%s", i.Prefix, check, dueMark, todayMark, i.Task.Title)
}

func (i TaskItem) Description() string {
	return ""
}

func (i TaskItem) FilterValue() string {
	return i.Task.Title
}
