package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var searchPromptStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#04B575")).
	Bold(true)

var searchQueryStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA"))

func RenderSearchPrompt(query string, focused bool) string {
	cursor := ""
	if focused {
		cursor = "█"
	}
	prompt := searchPromptStyle.Render("> ")
	q := searchQueryStyle.Render(query + cursor)
	return fmt.Sprintf("  %s%s", prompt, q)
}

func RenderSearchBar(query string, focused bool) string {
	return RenderSearchPrompt(query, focused)
}
