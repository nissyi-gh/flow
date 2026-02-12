package ui

import "github.com/nissyi-gh/flow/internal/model"

// BuildTree converts a flat task list into a tree-ordered list of TaskItems
// with tree-drawing prefixes (├─, └─, │).
func BuildTree(tasks []model.Task) []TaskItem {
	children := make(map[int][]model.Task)
	var roots []model.Task

	for _, t := range tasks {
		if t.ParentID == nil {
			roots = append(roots, t)
		} else {
			children[*t.ParentID] = append(children[*t.ParentID], t)
		}
	}

	var items []TaskItem
	var dfs func(task model.Task, ancestors []bool)
	dfs = func(task model.Task, ancestors []bool) {
		depth := len(ancestors)
		var prefix, descPrefix string
		if depth > 0 {
			for _, hasSibling := range ancestors[:depth-1] {
				if hasSibling {
					prefix += " │  "
					descPrefix += " │  "
				} else {
					prefix += "    "
					descPrefix += "    "
				}
			}
			if ancestors[depth-1] {
				prefix += " ├─ "
				descPrefix += " │  "
			} else {
				prefix += " └─ "
				descPrefix += "    "
			}
		}

		items = append(items, TaskItem{Task: task, Prefix: prefix, DescPrefix: descPrefix})
		kids := children[task.ID]
		for idx, child := range kids {
			isLast := idx == len(kids)-1
			dfs(child, append(ancestors, !isLast))
		}
	}

	for _, root := range roots {
		dfs(root, nil)
	}
	return items
}
