package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
	taskTitle := fmt.Sprintf("%s%s%s", dueMark, todayMark, i.Task.Title)
	if i.Task.Completed {
		taskTitle = lipgloss.NewStyle().Strikethrough(true).Render(taskTitle)
	}
	title := fmt.Sprintf("%s%s %s", i.Prefix, check, taskTitle)

	for _, tag := range i.Task.Tags {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color(tag.Color)).
			Render("[" + tag.Name + "]")
		title += " " + badge
	}

	return title
}

func (i TaskItem) Description() string {
	return ""
}

func (i TaskItem) FilterValue() string {
	return i.Task.Title
}
