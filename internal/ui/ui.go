package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
)

type extraKeyMap struct {
	Add     key.Binding
	SubAdd  key.Binding
	Toggle  key.Binding
	Delete  key.Binding
	Today   key.Binding
	DueDate key.Binding
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
	}
}

// Model is the top-level BubbleTea model for the flow TUI.
type Model struct {
	state       appState
	list        list.Model
	input       textinput.Model
	dateInput   dateInput
	store       *store.TaskStore
	keys        extraKeyMap
	addParentID   *int
	dueDateTaskID int
	err           error
	width       int
	height      int
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
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete, keys.Today, keys.DueDate}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete, keys.Today, keys.DueDate}
	}

	return Model{
		state:     stateList,
		list:      l,
		input:     ti,
		dateInput: newDateInput(),
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
		m.list.SetSize(leftWidth, msg.Height-v)
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
	return fmt.Sprintf("%s%s\n\ncreated_at: %s%s",
		todayMark,
		item.Task.Title,
		item.Task.CreatedAt.Format("2006-01-02 15:04"),
		dueLine,
	)
}

func (m Model) View() string {
	var errView string
	if m.err != nil {
		errView = "\n" + errorStyle.Render("Error: "+m.err.Error()) + "\n"
	}

	switch m.state {
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
