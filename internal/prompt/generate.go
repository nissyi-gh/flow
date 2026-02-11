package prompt

import (
	"fmt"
	"strings"

	"github.com/nissyi-gh/flow/internal/model"
)

const yamlFormat = `以下のYAMLフォーマットで出力してください。YAMLのコードブロックのみを出力し、それ以外の文章は含めないでください。

` + "```yaml" + `
tasks:
  - title: "タスク名"
    description: "タスクの詳細説明"
    due_date: "YYYY-MM-DD"
    tags:
      - "タグ名"
    children:
      - title: "子タスク名"
        description: "子タスクの説明"
` + "```" + `

フィールドの説明:
- title: (必須) タスクのタイトル
- description: (任意) タスクの詳細な説明
- due_date: (任意) 期限日 (YYYY-MM-DD形式)
- tags: (任意) タグのリスト
- children: (任意) 子タスクのリスト (再帰的にネスト可能)`

// GenerateNew returns a prompt for creating new tasks from scratch.
func GenerateNew() string {
	return fmt.Sprintf(`あなたはタスク管理のアシスタントです。
ユーザーの要求に基づいて、タスクを適切な粒度に分解してください。

%s
`, yamlFormat)
}

// GenerateFromTask returns a prompt for breaking down an existing task.
func GenerateFromTask(task model.Task, children []model.Task) string {
	var sb strings.Builder

	sb.WriteString("あなたはタスク管理のアシスタントです。\n")
	sb.WriteString("以下の既存タスクをより具体的な子タスクに分解してください。\n\n")

	sb.WriteString("## 対象タスク\n")
	sb.WriteString(fmt.Sprintf("- タイトル: %s\n", task.Title))

	if task.Description != nil && *task.Description != "" {
		sb.WriteString(fmt.Sprintf("- 説明: %s\n", *task.Description))
	}
	if task.DueDate != nil {
		sb.WriteString(fmt.Sprintf("- 期限: %s\n", *task.DueDate))
	}
	if len(task.Tags) > 0 {
		var tagNames []string
		for _, t := range task.Tags {
			tagNames = append(tagNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf("- タグ: %s\n", strings.Join(tagNames, ", ")))
	}

	if len(children) > 0 {
		sb.WriteString("\n## 既存の子タスク\n")
		for _, c := range children {
			status := "未完了"
			if c.Completed {
				status = "完了"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", c.Title, status))
		}
		sb.WriteString("\n上記の既存子タスクを考慮した上で、不足している子タスクを追加してください。\n")
	}

	sb.WriteString("\n")
	sb.WriteString(yamlFormat)
	sb.WriteString("\n")

	return sb.String()
}
