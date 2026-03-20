package main

import (
	"fmt"
	"os"

	"github.com/mexirica/aptui/internal/app"

	tea "charm.land/bubbletea/v2"
)

func main() {
	a := app.New()
	p := tea.NewProgram(a)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
