package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
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
	appStyle     = lipgloss.NewStyle().Padding(1, 2)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	confirmStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
)

type extraKeyMap struct {
	Add    key.Binding
	SubAdd key.Binding
	Toggle key.Binding
	Delete key.Binding
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
	}
}

type model struct {
	state       appState
	list        list.Model
	input       textinput.Model
	store       *TaskStore
	keys        extraKeyMap
	addParentID *int
	err         error
	width       int
	height      int
}

type tasksLoadedMsg []Task
type errMsg struct{ error }

func newModel(store *TaskStore) model {
	ti := textinput.New()
	ti.Placeholder = "Task title..."
	ti.CharLimit = 256

	keys := newExtraKeyMap()

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	l := list.New(nil, delegate, 0, 0)
	l.Title = "flow"
	l.Styles.Title = titleStyle
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("task", "tasks")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.SubAdd, keys.Toggle, keys.Delete}
	}

	return model{
		state: stateList,
		list:  l,
		input: ti,
		store: store,
		keys:  keys,
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

// buildTree converts a flat task list into a tree-ordered list of taskItems
// with tree-drawing prefixes (├─, └─, │).
func buildTree(tasks []Task) []taskItem {
	children := make(map[int][]Task) // parentID -> children
	var roots []Task

	for _, t := range tasks {
		if t.ParentID == nil {
			roots = append(roots, t)
		} else {
			children[*t.ParentID] = append(children[*t.ParentID], t)
		}
	}

	var items []taskItem
	// ancestors tracks whether each depth level's parent still has remaining siblings.
	// true = more siblings follow (draw │), false = last child (draw space).
	var dfs func(task Task, ancestors []bool)
	dfs = func(task Task, ancestors []bool) {
		depth := len(ancestors)
		var prefix, descPrefix string
		if depth > 0 {
			// Build the leading columns from ancestor context
			for _, hasSibling := range ancestors[:depth-1] {
				if hasSibling {
					prefix += "│  "
					descPrefix += "│  "
				} else {
					prefix += "   "
					descPrefix += "   "
				}
			}
			// Current level connector
			if ancestors[depth-1] {
				prefix += "├─ "
				descPrefix += "│  "
			} else {
				prefix += "└─ "
				descPrefix += "   "
			}
		}

		items = append(items, taskItem{task: task, prefix: prefix, descPrefix: descPrefix})
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tasksLoadedMsg:
		treeItems := buildTree([]Task(msg))
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
	}

	return m, nil
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.list.SettingFilter() {
		switch keyMsg.String() {
		case "a", "n":
			m.state = stateAdd
			m.addParentID = nil
			m.input.Reset()
			cmd := m.input.Focus()
			return m, cmd
		case "s":
			if item, ok := m.list.SelectedItem().(taskItem); ok {
				m.state = stateAdd
				id := item.task.ID
				m.addParentID = &id
				m.input.Reset()
				cmd := m.input.Focus()
				return m, cmd
			}
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
	case stateConfirm:
		item, _ := m.list.SelectedItem().(taskItem)
		msg := item.task.Title
		hasChildren, _ := m.store.HasChildren(item.task.ID)
		if hasChildren {
			msg = fmt.Sprintf("%s\n  (子タスクも削除されます)", item.task.Title)
		}
		return appStyle.Render(
			confirmStyle.Render("Delete Task?") + "\n\n" +
				"  " + msg + "\n\n" +
				statusStyle.Render("y: delete • n/esc: cancel") +
				errView,
		)
	default:
		return appStyle.Render(m.list.View() + errView)
	}
}
