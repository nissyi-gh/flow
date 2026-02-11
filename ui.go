package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	stateList appState = iota
	stateAdd
	stateConfirm
)

var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type model struct {
	state   appState
	list    list.Model
	input   textinput.Model
	store   *TaskStore
	err     error
	width   int
	height  int
}

type tasksLoadedMsg []Task
type errMsg struct{ error }

func newModel(store *TaskStore) model {
	// Text input for adding tasks
	ti := textinput.New()
	ti.Placeholder = "Task title..."
	ti.CharLimit = 256

	// List delegate
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "flow"
	l.Styles.Title = titleStyle
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)

	return model{
		state: stateList,
		list:  l,
		input: ti,
		store: store,
	}
}

func (m model) Init() tea.Cmd {
	return m.loadTasks
}

func (m model) loadTasks() tea.Msg {
	tasks, err := m.store.List()
	if err != nil {
		return errMsg{err}
	}
	return tasksLoadedMsg(tasks)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tasksLoadedMsg:
		items := make([]list.Item, len(msg))
		for i, t := range msg {
			items[i] = taskItem{task: t}
		}
		m.list.SetItems(items)
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
	}

	return m, nil
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.list.SettingFilter() {
		switch keyMsg.String() {
		case "a", "n":
			m.state = stateAdd
			m.input.Reset()
			cmd := m.input.Focus()
			return m, cmd
		case "enter", "x":
			if item, ok := m.list.SelectedItem().(taskItem); ok {
				if err := m.store.ToggleComplete(item.task.ID); err != nil {
					m.err = err
					return m, nil
				}
				return m, m.loadTasks
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

func (m model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			title := m.input.Value()
			if title != "" {
				if _, err := m.store.Add(title); err != nil {
					m.err = err
				}
			}
			m.state = stateList
			return m, m.loadTasks
		case "esc":
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y":
			if item, ok := m.list.SelectedItem().(taskItem); ok {
				if err := m.store.Delete(item.task.ID); err != nil {
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

func (m model) View() string {
	switch m.state {
	case stateAdd:
		return appStyle.Render(
			titleStyle.Render("New Task") + "\n\n" +
				m.input.View() + "\n\n" +
				statusStyle.Render("enter: save • esc: cancel"),
		)
	case stateConfirm:
		item, _ := m.list.SelectedItem().(taskItem)
		return appStyle.Render(
			titleStyle.Render("Delete Task?") + "\n\n" +
				"  " + item.task.Title + "\n\n" +
				statusStyle.Render("y: delete • n/esc: cancel"),
		)
	default:
		return appStyle.Render(m.list.View())
	}
}
