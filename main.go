package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nissyi-gh/flow/internal/store"
	"github.com/nissyi-gh/flow/internal/ui"
)

func main() {
	s, err := store.NewTaskStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	p := tea.NewProgram(ui.NewModel(s), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
