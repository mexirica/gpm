package main

import (
	"github.com/mexirica/gpm/internal/app"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	a := app.New()
	p := tea.NewProgram(a, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
