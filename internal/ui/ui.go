package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nissyi-gh/flow/internal/model"
	"github.com/nissyi-gh/flow/internal/store"
)

type appState int

const (
	stateList appState = iota
	stateAdd
	stateConfirm
	stateDueDate
	stateEditDesc
	stateTagSelect
)

var (
	appStyle     = lipgloss.NewStyle().Padding(1, 2)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	confirmStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	detailStyle  = lipgloss.NewStyle().
			Padding(1, 2).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("241"))
	descBoxStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241"))

	tagColorPalette = []string{"39", "205", "148", "214", "141", "81", "203", "227"}
)

type extraKeyMap struct {
	Add      key.Binding
	SubAdd   key.Binding
	Toggle   key.Binding
	Delete   key.Binding
	Today    key.Binding
	DueDate   key.Binding
	EditDesc  key.Binding
	TagSelect key.Binding
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
	tagCreating   bool
	tagInput      textinput.Model
	err           error
	width         int
	height        int
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
	l := list.New(nil, delegate, 0, 0)
	l.Title = "flow"
	l.Styles.Title = titleStyle
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("task", "tasks")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete, keys.Today, keys.DueDate, keys.EditDesc, keys.TagSelect}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete, keys.Today, keys.DueDate, keys.EditDesc, keys.TagSelect}
	}

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
		leftWidth := contentWidth * 60 / 100
		rightWidth := contentWidth - leftWidth
		m.list.SetSize(leftWidth, msg.Height-v)
		m.descInput.SetWidth(rightWidth - 6)
		m.descInput.SetHeight(msg.Height - v - 10)
		return m, nil

	case tasksLoadedMsg:
		treeItems := BuildTree([]model.Task(msg))
		items := make([]list.Item, len(treeItems))
		for i, ti := range treeItems {
			items[i] = ti
		}
		m.list.SetItems(items)
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
	}

	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.list.SettingFilter() {
		switch keyMsg.String() {
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
		case "enter", "x":
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				if err := m.store.ToggleComplete(item.Task.ID); err != nil {
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
	todayMark := ""
	if item.Task.IsToday() {
		todayMark = "📌 "
	}
	descContent := statusStyle.Render("(no description)")
	if item.Task.Description != nil && *item.Task.Description != "" {
		descContent = *item.Task.Description
	}
	desc := descBoxStyle.Render(descContent)

	tagsLine := ""
	if len(item.Task.Tags) > 0 {
		var tagBadges []string
		for _, tag := range item.Task.Tags {
			badge := lipgloss.NewStyle().
				Foreground(lipgloss.Color(tag.Color)).
				Bold(true).
				Render("[" + tag.Name + "]")
			tagBadges = append(tagBadges, badge)
		}
		tagsLine = "\ntags: " + strings.Join(tagBadges, " ") + "\n"
	}

	dueLine := ""
	if item.Task.DueDate != nil {
		label := "due_date:  " + *item.Task.DueDate
		if item.Task.IsOverdue() {
			label = errorStyle.Render("⚠️ " + label)
		} else if item.Task.IsDueToday() {
			label = "📅 " + label
		}
		dueLine = "\n" + label
	}
	return fmt.Sprintf("%s%s\n\n%s%s\n\ncreated_at: %s%s\n\n%s",
		todayMark,
		item.Task.Title,
		desc,
		tagsLine,
		item.Task.CreatedAt.Format("2006-01-02 15:04"),
		dueLine,
		statusStyle.Render("e: edit description  T: tags"),
	)
}

func (m Model) View() string {
	var errView string
	if m.err != nil {
		errView = "\n" + errorStyle.Render("Error: "+m.err.Error()) + "\n"
	}

	switch m.state {
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
		contentHeight := m.height - v
		leftWidth := contentWidth * 60 / 100
		rightWidth := contentWidth - leftWidth

		leftPane := m.list.View()
		rightPane := detailStyle.
			Width(rightWidth).
			Height(contentHeight).
			Render(m.renderDetail())
		_ = leftWidth // leftWidth is controlled via SetSize
		content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		return appStyle.Render(content + errView)
	}
}
