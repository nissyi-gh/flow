package importer

import (
	"fmt"

	"github.com/nissyi-gh/flow/internal/store"
	"gopkg.in/yaml.v3"
)

// YAMLTask represents a single task in the YAML input.
type YAMLTask struct {
	Title       string     `yaml:"title"`
	Description string     `yaml:"description,omitempty"`
	DueDate     string     `yaml:"due_date,omitempty"`
	Tags        []string   `yaml:"tags,omitempty"`
	Children    []YAMLTask `yaml:"children,omitempty"`
}

// YAMLInput represents the root structure of the YAML input.
type YAMLInput struct {
	Tasks []YAMLTask `yaml:"tasks"`
}

// Import parses a YAML string and creates tasks in the store.
// parentID can be nil for root-level tasks.
// Returns the number of tasks created.
func Import(s *store.TaskStore, yamlStr string, parentID *int) (int, error) {
	var input YAMLInput
	if err := yaml.Unmarshal([]byte(yamlStr), &input); err != nil {
		return 0, fmt.Errorf("YAML parse error: %w", err)
	}

	if len(input.Tasks) == 0 {
		return 0, fmt.Errorf("no tasks found in YAML")
	}

	count := 0
	for _, yt := range input.Tasks {
		n, err := importTask(s, yt, parentID)
		if err != nil {
			return count, err
		}
		count += n
	}
	return count, nil
}

func importTask(s *store.TaskStore, yt YAMLTask, parentID *int) (int, error) {
	if yt.Title == "" {
		return 0, fmt.Errorf("task title is required")
	}

	task, err := s.Add(yt.Title, parentID)
	if err != nil {
		return 0, fmt.Errorf("add task %q: %w", yt.Title, err)
	}
	count := 1

	if yt.Description != "" {
		desc := yt.Description
		if err := s.UpdateDescription(task.ID, &desc); err != nil {
			return count, fmt.Errorf("set description for %q: %w", yt.Title, err)
		}
	}

	if yt.DueDate != "" {
		dd := yt.DueDate
		if err := s.SetDueDate(task.ID, &dd); err != nil {
			return count, fmt.Errorf("set due date for %q: %w", yt.Title, err)
		}
	}

	if len(yt.Tags) > 0 {
		if err := assignTags(s, task.ID, yt.Tags); err != nil {
			return count, fmt.Errorf("assign tags for %q: %w", yt.Title, err)
		}
	}

	for _, child := range yt.Children {
		id := task.ID
		n, err := importTask(s, child, &id)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}

func assignTags(s *store.TaskStore, taskID int, tagNames []string) error {
	existingTags, err := s.ListTags()
	if err != nil {
		return err
	}

	tagMap := make(map[string]int)
	for _, t := range existingTags {
		tagMap[t.Name] = t.ID
	}

	palette := []string{"39", "205", "148", "214", "141", "81", "203", "227"}

	for _, name := range tagNames {
		tagID, exists := tagMap[name]
		if !exists {
			color := palette[len(tagMap)%len(palette)]
			tag, err := s.CreateTag(name, color)
			if err != nil {
				return fmt.Errorf("create tag %q: %w", name, err)
			}
			tagID = tag.ID
			tagMap[name] = tagID
		}
		if err := s.AssignTag(taskID, tagID); err != nil {
			return fmt.Errorf("assign tag %q: %w", name, err)
		}
	}
	return nil
}
