package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type dateInput struct {
	fields [3]textinput.Model // 0:YYYY, 1:MM, 2:DD
	focus  int                // 現在フォーカス中のフィールドインデックス
}

func newDateInput() dateInput {
	placeholders := [3]string{"YYYY", "MM", "DD"}
	charLimits := [3]int{4, 2, 2}

	var fields [3]textinput.Model
	for i := 0; i < 3; i++ {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = charLimits[i]
		ti.Width = charLimits[i] + 2
		ti.Validate = func(s string) error {
			for _, r := range s {
				if !unicode.IsDigit(r) {
					return fmt.Errorf("digits only")
				}
			}
			return nil
		}
		fields[i] = ti
	}

	return dateInput{fields: fields}
}

func (d *dateInput) Focus() {
	d.focus = 0
	d.fields[0].Focus()
	d.fields[1].Blur()
	d.fields[2].Blur()
}

func (d *dateInput) Blur() {
	for i := range d.fields {
		d.fields[i].Blur()
	}
}

func (d *dateInput) SetValue(date string) {
	parts := strings.SplitN(date, "-", 3)
	for i := 0; i < 3; i++ {
		if i < len(parts) {
			d.fields[i].SetValue(parts[i])
		} else {
			d.fields[i].SetValue("")
		}
	}
}

func (d *dateInput) Value() (string, error) {
	now := time.Now()

	yyyy := strings.TrimSpace(d.fields[0].Value())
	mm := strings.TrimSpace(d.fields[1].Value())
	dd := strings.TrimSpace(d.fields[2].Value())

	if yyyy == "" {
		yyyy = fmt.Sprintf("%04d", now.Year())
	}
	if mm == "" {
		mm = fmt.Sprintf("%02d", int(now.Month()))
	}
	if dd == "" {
		return "", fmt.Errorf("day is required")
	}

	dateStr := fmt.Sprintf("%s-%s-%s", yyyy, padLeft(mm, 2), padLeft(dd, 2))

	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return "", fmt.Errorf("invalid date: %s", dateStr)
	}

	return dateStr, nil
}

func padLeft(s string, length int) string {
	for len(s) < length {
		s = "0" + s
	}
	return s
}

func (d *dateInput) IsEmpty() bool {
	return d.fields[0].Value() == "" && d.fields[1].Value() == "" && d.fields[2].Value() == ""
}

func (d *dateInput) focusField(idx int) tea.Cmd {
	d.focus = idx
	var cmds []tea.Cmd
	for i := range d.fields {
		if i == idx {
			cmds = append(cmds, d.fields[i].Focus())
		} else {
			d.fields[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (d dateInput) Update(msg tea.Msg) (dateInput, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "right":
			if d.focus < 2 {
				cmd := d.focusField(d.focus + 1)
				return d, cmd
			}
			return d, nil
		case "shift+tab", "left":
			if d.focus > 0 {
				cmd := d.focusField(d.focus - 1)
				return d, cmd
			}
			return d, nil
		}
	}

	var cmd tea.Cmd
	d.fields[d.focus], cmd = d.fields[d.focus].Update(msg)
	return d, cmd
}

func (d dateInput) View() string {
	return d.fields[0].View() + " - " + d.fields[1].View() + " - " + d.fields[2].View()
}
