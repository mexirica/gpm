package main

import (
	"fmt"
	"os"

	"github.com/mexirica/aptui/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	a := app.New()
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
