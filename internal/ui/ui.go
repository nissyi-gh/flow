package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nissyi-gh/flow/internal/importer"
	"github.com/nissyi-gh/flow/internal/model"
	"github.com/nissyi-gh/flow/internal/prompt"
	"github.com/nissyi-gh/flow/internal/store"
)

type viewMode int

const (
	viewAll viewMode = iota
	viewToday
)

type appState int

const (
	stateList appState = iota
	stateAdd
	stateConfirm
	stateDueDate
	stateEditDesc
	stateTagSelect
	stateGenerate
	stateImportSelect
	stateImportResult
	stateQuitConfirm
)

var (
	appStyle     = lipgloss.NewStyle().Padding(1, 2)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	confirmStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	detailStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("39"))
	tagColorPalette = []string{"39", "205", "148", "214", "141", "81", "203", "227"}
)

type extraKeyMap struct {
	Add       key.Binding
	SubAdd    key.Binding
	Toggle    key.Binding
	Delete    key.Binding
	Today     key.Binding
	DueDate   key.Binding
	EditDesc  key.Binding
	TagSelect key.Binding
	Generate  key.Binding
	Import    key.Binding
}

func newExtraKeyMap() extraKeyMap {
	return extraKeyMap{
		Add: key.NewBinding(
			key.WithKeys("a", "n"),
			key.WithHelp("a/n", "add"),
		),
		SubAdd: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sub-task"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", "x"),
			key.WithHelp("enter/x", "toggle"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Today: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "today"),
		),
		DueDate: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "due date"),
		),
		EditDesc: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit desc"),
		),
		TagSelect: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "tags"),
		),
		Generate: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "AI prompt"),
		),
		Import: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "import YAML"),
		),
	}
}

// Model is the top-level BubbleTea model for the flow TUI.
type Model struct {
	state         appState
	list          list.Model
	input         textinput.Model
	dateInput     dateInput
	descInput     textarea.Model
	store         *store.TaskStore
	keys          extraKeyMap
	addParentID   *int
	dueDateTaskID int
	editTaskID    int
	tagTaskID     int
	allTags       []model.Tag
	assignedTags  map[int]bool
	tagCursor     int
	tagCreating    bool
	tagInput       textinput.Model
	genCursor       int
	importCursor    int
	importYAML      string
	importResult    string
	importIsError   bool
	viewMode       viewMode
	err            error
	width          int
	height         int
}

type tasksLoadedMsg []model.Task
type errMsg struct{ error }

// NewModel creates a new TUI model.
func NewModel(s *store.TaskStore) Model {
	ti := textinput.New()
	ti.Placeholder = "Task title..."
	ti.CharLimit = 256

	keys := newExtraKeyMap()

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true).
		Padding(0, 0, 0, 1).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("75"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle
	l := list.New(nil, delegate, 0, 0)
	l.Title = "flow"
	l.Styles.Title = titleStyle
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("task", "tasks")

	ta := textarea.New()
	ta.Placeholder = "Task description..."
	ta.CharLimit = 4096

	tagIn := textinput.New()
	tagIn.Placeholder = "New tag name..."
	tagIn.CharLimit = 32

	return Model{
		state:     stateList,
		list:      l,
		input:     ti,
		dateInput: newDateInput(),
		descInput: ta,
		tagInput:  tagIn,
		store:     s,
		keys:      keys,
	}
}

func (m Model) viewTitle() string {
	switch m.viewMode {
	case viewToday:
		return "flow [📌 today]"
	default:
		return "flow"
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadTasks
}

func (m Model) loadTasks() tea.Msg {
	tasks, err := m.store.List()
	if err != nil {
		return errMsg{err}
	}
	return tasksLoadedMsg(tasks)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		contentWidth := msg.Width - h
		leftWidth := contentWidth / 2
		rightWidth := contentWidth - leftWidth
		helpHeight := 2
		m.list.SetSize(leftWidth, msg.Height-v-helpHeight)
		m.descInput.SetWidth(rightWidth - 6)
		m.descInput.SetHeight(msg.Height - v - 10)
		return m, nil

	case tasksLoadedMsg:
		tasks := []model.Task(msg)
		if m.viewMode == viewToday {
			taskByID := make(map[int]model.Task)
			for _, t := range tasks {
				taskByID[t.ID] = t
			}
			include := make(map[int]bool)
			for _, t := range tasks {
				if t.IsToday() {
					// Include the task and all its ancestors
					for cur := &t; cur != nil; {
						if include[cur.ID] {
							break
						}
						include[cur.ID] = true
						if cur.ParentID != nil {
							p := taskByID[*cur.ParentID]
							cur = &p
						} else {
							cur = nil
						}
					}
				}
			}
			var filtered []model.Task
			for _, t := range tasks {
				if include[t.ID] {
					filtered = append(filtered, t)
				}
			}
			tasks = filtered
		}
		treeItems := BuildTree(tasks)
		items := make([]list.Item, len(treeItems))
		for i, ti := range treeItems {
			items[i] = ti
		}
		m.list.SetItems(items)
		m.list.Title = m.viewTitle()
		m.err = nil
		return m, nil

	case errMsg:
		m.err = msg.error
		return m, nil
	}

	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateAdd:
		return m.updateAdd(msg)
	case stateConfirm:
		return m.updateConfirm(msg)
	case stateDueDate:
		return m.updateDueDate(msg)
	case stateEditDesc:
		return m.updateEditDesc(msg)
	case stateTagSelect:
		return m.updateTagSelect(msg)
	case stateGenerate:
		return m.updateGenerate(msg)
	case stateImportSelect:
		return m.updateImportSelect(msg)
	case stateImportResult:
		return m.updateImportResult(msg)
	case stateQuitConfirm:
		return m.updateQuitConfirm(msg)
	}

	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.list.SettingFilter() {
		switch keyMsg.String() {
		case "esc", "q":
			m.state = stateQuitConfirm
			return m, nil
		case "v":
			if m.viewMode == viewAll {
				m.viewMode = viewToday
			} else {
				m.viewMode = viewAll
			}
			return m, m.loadTasks
		case "a", "n":
			m.state = stateAdd
			m.addParentID = nil
			m.input.Reset()
			cmd := m.input.Focus()
			return m, cmd
		case "s":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.state = stateAdd
				id := item.Task.ID
				m.addParentID = &id
				m.input.Reset()
				cmd := m.input.Focus()
				return m, cmd
			}
		case "p":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				if err := m.store.SetStatus(item.Task.ID, model.StatusInProgress); err != nil {
					m.err = err
					return m, nil
				}
				return m, m.loadTasks
			}
		case "enter", "x":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				if err := m.store.SetStatus(item.Task.ID, model.StatusCompleted); err != nil {
					m.err = err
					return m, nil
				}
				return m, m.loadTasks
			}
		case "t":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				if err := m.store.ToggleToday(item.Task.ID); err != nil {
					m.err = err
					return m, nil
				}
				return m, m.loadTasks
			}
		case "D":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.state = stateDueDate
				m.dueDateTaskID = item.Task.ID
				m.dateInput = newDateInput()
				if item.Task.DueDate != nil {
					m.dateInput.SetValue(*item.Task.DueDate)
				}
				m.dateInput.Focus()
				return m, nil
			}
		case "T":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.tagTaskID = item.Task.ID
				m.state = stateTagSelect
				m.tagCursor = 0
				m.tagCreating = false
				allTags, err := m.store.ListTags()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.allTags = allTags
				assigned, err := m.store.TagsForTask(item.Task.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.assignedTags = make(map[int]bool)
				for _, t := range assigned {
					m.assignedTags[t.ID] = true
				}
				return m, nil
			}
		case "e":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.state = stateEditDesc
				m.editTaskID = item.Task.ID
				m.descInput.Reset()
				if item.Task.Description != nil {
					m.descInput.SetValue(*item.Task.Description)
				}
				cmd := m.descInput.Focus()
				return m, cmd
			}
		case "g":
			m.state = stateGenerate
			m.genCursor = 0
			return m, nil
		case "G":
			content, err := clipboard.ReadAll()
			if err != nil {
				m.importResult = fmt.Sprintf("クリップボードの読み取りに失敗しました: %v", err)
				m.importIsError = true
				m.state = stateImportResult
				return m, nil
			}
			m.importYAML = stripCodeBlock(content)
			m.importCursor = 0
			m.state = stateImportSelect
			return m, nil
		case "c":
			md := m.tasksToMarkdown()
			if err := clipboard.WriteAll(md); err != nil {
				m.importResult = fmt.Sprintf("クリップボードへのコピーに失敗しました: %v", err)
				m.importIsError = true
			} else {
				m.importResult = "タスク一覧をクリップボードにコピーしました"
				m.importIsError = false
			}
			m.state = stateImportResult
			return m, nil
		case "d":
			if m.list.SelectedItem() != nil {
				m.state = stateConfirm
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			title := m.input.Value()
			if title != "" {
				if _, err := m.store.Add(title, m.addParentID); err != nil {
					m.err = err
				}
			}
			m.state = stateList
			m.addParentID = nil
			return m, m.loadTasks
		case "esc":
			m.state = stateList
			m.addParentID = nil
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateEditDesc(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			val := m.descInput.Value()
			var desc *string
			if val != "" {
				desc = &val
			}
			if err := m.store.UpdateDescription(m.editTaskID, desc); err != nil {
				m.err = err
			}
			m.state = stateList
			return m, m.loadTasks
		case "ctrl+c":
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func nextTagColor(existingCount int) string {
	return tagColorPalette[existingCount%len(tagColorPalette)]
}

func (m Model) updateTagSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.tagCreating {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter":
				name := strings.TrimSpace(m.tagInput.Value())
				if name != "" {
					tag, err := m.store.CreateTag(name, nextTagColor(len(m.allTags)))
					if err != nil {
						m.err = err
					} else {
						m.allTags = append(m.allTags, tag)
						if err := m.store.AssignTag(m.tagTaskID, tag.ID); err != nil {
							m.err = err
						} else {
							m.assignedTags[tag.ID] = true
						}
					}
				}
				m.tagCreating = false
				m.tagInput.Reset()
				return m, nil
			case "esc":
				m.tagCreating = false
				m.tagInput.Reset()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.tagInput, cmd = m.tagInput.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "j", "down":
			if m.tagCursor < len(m.allTags) {
				m.tagCursor++
			}
		case "k", "up":
			if m.tagCursor > 0 {
				m.tagCursor--
			}
		case "enter", " ", "x":
			if m.tagCursor < len(m.allTags) {
				tag := m.allTags[m.tagCursor]
				if m.assignedTags[tag.ID] {
					if err := m.store.UnassignTag(m.tagTaskID, tag.ID); err != nil {
						m.err = err
					} else {
						delete(m.assignedTags, tag.ID)
					}
				} else {
					if err := m.store.AssignTag(m.tagTaskID, tag.ID); err != nil {
						m.err = err
					} else {
						m.assignedTags[tag.ID] = true
					}
				}
			} else {
				m.tagCreating = true
				m.tagInput.Reset()
				cmd := m.tagInput.Focus()
				return m, cmd
			}
		case "esc":
			m.state = stateList
			return m, m.loadTasks
		}
	}
	return m, nil
}

func (m Model) updateGenerate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "j", "down":
			if m.genCursor < 1 {
				m.genCursor++
			}
		case "k", "up":
			if m.genCursor > 0 {
				m.genCursor--
			}
		case "enter":
			if m.genCursor == 0 {
				// New task breakdown
				p := prompt.GenerateNew()
				if err := clipboard.WriteAll(p); err != nil {
					m.importResult = fmt.Sprintf("クリップボードへのコピーに失敗しました: %v", err)
					m.importIsError = true
				} else {
					m.importResult = "新規タスク分解用のプロンプトをクリップボードにコピーしました"
					m.importIsError = false
				}
				m.state = stateImportResult
				return m, nil
			} else {
				// Improve existing task
				item, ok := m.list.SelectedItem().(TaskItem)
				if !ok {
					m.state = stateList
					return m, nil
				}
				children, err := m.store.ChildrenOf(item.Task.ID)
				if err != nil {
					m.err = err
					m.state = stateList
					return m, nil
				}
				p := prompt.GenerateFromTask(item.Task, children)
				if err := clipboard.WriteAll(p); err != nil {
					m.importResult = fmt.Sprintf("クリップボードへのコピーに失敗しました: %v", err)
					m.importIsError = true
				} else {
					m.importResult = fmt.Sprintf("「%s」の分解プロンプトをクリップボードにコピーしました", item.Task.Title)
					m.importIsError = false
				}
				m.state = stateImportResult
				return m, nil
			}
		case "esc":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateImportSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "j", "down":
			if m.importCursor < 1 {
				m.importCursor++
			}
		case "k", "up":
			if m.importCursor > 0 {
				m.importCursor--
			}
		case "enter":
			var parentID *int
			if m.importCursor == 1 {
				if item, ok := m.list.SelectedItem().(TaskItem); ok {
					id := item.Task.ID
					parentID = &id
				}
			}
			return m.doImport(parentID)
		case "esc":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) doImport(parentID *int) (tea.Model, tea.Cmd) {
	count, err := importer.Import(m.store, m.importYAML, parentID)
	if err != nil {
		m.importResult = fmt.Sprintf("YAMLのインポートに失敗しました: %v", err)
		m.importIsError = true
		m.state = stateImportResult
		return m, nil
	}

	m.importResult = fmt.Sprintf("✓ %d 件のタスクをインポートしました", count)
	m.importIsError = false
	m.state = stateImportResult
	return m, nil
}

func (m Model) tasksToMarkdown() string {
	items := m.list.Items()
	taskByID := make(map[int]model.Task)
	for _, it := range items {
		if t, ok := it.(TaskItem); ok {
			taskByID[t.Task.ID] = t.Task
		}
	}
	var sb strings.Builder
	for _, item := range items {
		ti, ok := item.(TaskItem)
		if !ok {
			continue
		}
		depth := 0
		pid := ti.Task.ParentID
		for pid != nil {
			depth++
			if parent, ok := taskByID[*pid]; ok {
				pid = parent.ParentID
			} else {
				break
			}
		}
		indent := strings.Repeat("  ", depth)
		var check string
		switch ti.Task.Status {
		case model.StatusInProgress:
			check = "[-]"
		case model.StatusCompleted:
			check = "[x]"
		default:
			check = "[ ]"
		}
		sb.WriteString(fmt.Sprintf("%s- %s %s\n", indent, check, ti.Task.Title))
	}
	return sb.String()
}

func stripCodeBlock(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock && (trimmed == "```yaml" || trimmed == "```yml" || trimmed == "```") {
			inBlock = true
			continue
		}
		if inBlock && trimmed == "```" {
			inBlock = false
			continue
		}
		if inBlock {
			result = append(result, line)
		}
	}
	// If no code block was found, return original
	if len(result) == 0 {
		return s
	}
	return strings.Join(result, "\n")
}

func (m Model) updateImportResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.state = stateList
		return m, m.loadTasks
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				if err := m.store.Delete(item.Task.ID); err != nil {
					m.err = err
				}
			}
			m.state = stateList
			return m, m.loadTasks
		case "n", "esc":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateQuitConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y":
			return m, tea.Quit
		case "n", "esc":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateDueDate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			if m.dateInput.IsEmpty() {
				if err := m.store.SetDueDate(m.dueDateTaskID, nil); err != nil {
					m.err = err
				}
				m.state = stateList
				return m, m.loadTasks
			}
			val, err := m.dateInput.Value()
			if err != nil {
				m.err = err
				return m, nil
			}
			if err := m.store.SetDueDate(m.dueDateTaskID, &val); err != nil {
				m.err = err
			}
			m.state = stateList
			return m, m.loadTasks
		case "esc":
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.dateInput, cmd = m.dateInput.Update(msg)
	return m, cmd
}

func (m Model) renderDetail() string {
	item, ok := m.list.SelectedItem().(TaskItem)
	if !ok {
		return ""
	}

	var sb strings.Builder
	sectionHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	// # Title line with marks
	var marks []string
	if item.Task.IsOverdue() {
		marks = append(marks, "⚠️")
	}
	if item.Task.IsDueToday() {
		marks = append(marks, "📅")
	}
	if item.Task.IsToday() {
		marks = append(marks, "📌")
	}
	if len(item.Task.Tags) > 0 {
		for _, tag := range item.Task.Tags {
			badge := lipgloss.NewStyle().
				Foreground(lipgloss.Color(tag.Color)).
				Bold(true).
				Render("[" + tag.Name + "]")
			marks = append(marks, badge)
		}
	}
	titleLine := item.Task.Title
	if item.Task.Completed {
		titleLine = lipgloss.NewStyle().Strikethrough(true).Render(titleLine)
	}
	if len(marks) > 0 {
		titleLine = strings.Join(marks, " ") + " " + titleLine
	}
	sb.WriteString(titleStyle.Render(titleLine))

	// ## Description
	sb.WriteString("\n\n")
	sb.WriteString(sectionHeader.Render("Description"))
	sb.WriteString("\n")
	if item.Task.Description != nil && *item.Task.Description != "" {
		sb.WriteString(*item.Task.Description)
	} else {
		sb.WriteString(statusStyle.Render("(no description)"))
	}

	// ## Property
	sb.WriteString("\n\n")
	sb.WriteString(sectionHeader.Render("Property"))
	sb.WriteString("\n")

	dueValue := statusStyle.Render("-")
	if item.Task.DueDate != nil {
		dueValue = *item.Task.DueDate
	}
	sb.WriteString(fmt.Sprintf("due_date:    %s\n", dueValue))
	sb.WriteString(fmt.Sprintf("created_at:  %s", item.Task.CreatedAt.Format("2006-01-02 15:04")))

	// Footer
	sb.WriteString("\n\n")
	sb.WriteString(statusStyle.Render("e: edit description  T: tags"))

	return sb.String()
}

func (m Model) renderHelp(width int) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	items := []struct{ key, desc string }{
		{"a/n", "add"}, {"s", "sub-task"}, {"p", "progress"}, {"enter/x", "done"}, {"d", "delete"},
		{"t", "today"}, {"D", "due date"}, {"e", "edit desc"}, {"T", "tags"},
		{"c", "copy"}, {"v", "view"}, {"g", "AI prompt"}, {"G", "import YAML"}, {"/", "filter"}, {"q", "quit"},
	}

	var lines []string
	line := ""
	for _, it := range items {
		entry := keyStyle.Render(it.key) + " " + it.desc
		sep := statusStyle.Render(" • ")
		if line == "" {
			line = entry
			continue
		}
		candidate := line + sep + entry
		if lipgloss.Width(candidate) > width {
			lines = append(lines, line)
			line = entry
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) View() string {
	var errView string
	if m.err != nil {
		errView = "\n" + errorStyle.Render("Error: "+m.err.Error()) + "\n"
	}

	switch m.state {
	case stateGenerate:
		options := []string{"新しいタスクを分解する", "既存タスクを改善する"}
		var taskName string
		if item, ok := m.list.SelectedItem().(TaskItem); ok {
			taskName = item.Task.Title
		}
		if taskName != "" {
			options[1] = fmt.Sprintf("既存タスクを改善する (%s)", taskName)
		}

		var lines []string
		for i, opt := range options {
			cursor := "  "
			if i == m.genCursor {
				cursor = "> "
			}
			lines = append(lines, cursor+opt)
		}
		content := titleStyle.Render("AI Task Breakdown") + "\n\n" +
			strings.Join(lines, "\n") + "\n\n" +
			statusStyle.Render("j/k: navigate  enter: select  esc: cancel")
		return appStyle.Render(content + errView)

	case stateImportSelect:
		options := []string{"ルートタスクとしてインポート"}
		if item, ok := m.list.SelectedItem().(TaskItem); ok {
			options = append(options, fmt.Sprintf("「%s」の子タスクとしてインポート", item.Task.Title))
		} else {
			options = append(options, "選択中タスクの子タスクとしてインポート (未選択)")
		}

		var lines []string
		for i, opt := range options {
			cursor := "  "
			if i == m.importCursor {
				cursor = "> "
			}
			lines = append(lines, cursor+opt)
		}
		content := titleStyle.Render("Import YAML") + "\n\n" +
			strings.Join(lines, "\n") + "\n\n" +
			statusStyle.Render("j/k: navigate  enter: select  esc: cancel")
		return appStyle.Render(content + errView)

	case stateImportResult:
		var icon string
		style := titleStyle
		if m.importIsError {
			icon = "✗ "
			style = errorStyle
		} else {
			icon = ""
		}
		content := style.Render(icon+m.importResult) + "\n\n" +
			statusStyle.Render("press any key to continue")
		return appStyle.Render(content + errView)

	case stateTagSelect:
		var lines []string
		for i, tag := range m.allTags {
			cursor := "  "
			if i == m.tagCursor {
				cursor = "> "
			}
			check := "[ ]"
			if m.assignedTags[tag.ID] {
				check = "[x]"
			}
			badge := lipgloss.NewStyle().
				Foreground(lipgloss.Color(tag.Color)).
				Render(tag.Name)
			lines = append(lines, cursor+check+" "+badge)
		}
		newCursor := "  "
		if m.tagCursor == len(m.allTags) {
			newCursor = "> "
		}
		lines = append(lines, newCursor+"+ New tag...")

		content := titleStyle.Render("Tags") + "\n\n" +
			strings.Join(lines, "\n")

		if m.tagCreating {
			content += "\n\n" + m.tagInput.View()
		}

		content += "\n\n" + statusStyle.Render("j/k: navigate  enter/space: toggle  esc: done")

		return appStyle.Render(content + errView)
	case stateEditDesc:
		return appStyle.Render(
			titleStyle.Render("Edit Description") + "\n\n" +
				m.descInput.View() + "\n\n" +
				statusStyle.Render("esc: save • ctrl+c: cancel") +
				errView,
		)
	case stateAdd:
		header := "New Task"
		if m.addParentID != nil {
			header = "New Sub-task"
		}
		return appStyle.Render(
			titleStyle.Render(header) + "\n\n" +
				m.input.View() + "\n\n" +
				statusStyle.Render("enter: save • esc: cancel") +
				errView,
		)
	case stateDueDate:
		return appStyle.Render(
			titleStyle.Render("Set Due Date") + "\n\n" +
				m.dateInput.View() + "\n\n" +
				statusStyle.Render("tab/→: next field • enter: save • esc: cancel") +
				errView,
		)
	case stateQuitConfirm:
		return appStyle.Render(
			confirmStyle.Render("Quit flow?") + "\n\n" +
				statusStyle.Render("y: quit • n/esc: cancel") +
				errView,
		)
	case stateConfirm:
		item, _ := m.list.SelectedItem().(TaskItem)
		msg := item.Task.Title
		hasChildren, _ := m.store.HasChildren(item.Task.ID)
		if hasChildren {
			msg = fmt.Sprintf("%s\n  (子タスクも削除されます)", item.Task.Title)
		}
		return appStyle.Render(
			confirmStyle.Render("Delete Task?") + "\n\n" +
				"  " + msg + "\n\n" +
				statusStyle.Render("y: delete • n/esc: cancel") +
				errView,
		)
	default:
		h, v := appStyle.GetFrameSize()
		contentWidth := m.width - h
		helpHeight := 2
		contentHeight := m.height - v - helpHeight
		leftWidth := contentWidth / 2
		rightWidth := contentWidth - leftWidth

		leftPane := lipgloss.NewStyle().Width(leftWidth).Render(m.list.View())
		rightPane := detailStyle.
			Width(rightWidth).
			Height(contentHeight).
			Render(m.renderDetail())
		_ = leftWidth
		panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := m.renderHelp(contentWidth)
		return appStyle.Render(panes + "\n" + help + errView)
	}
}
