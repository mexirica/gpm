package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var queryPromptStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7D56F4")).
	Bold(true)

var queryTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#D0D0E0"))

// RenderQueryPrompt renders the unified search/filter prompt.
func RenderQueryPrompt(query string, focused bool) string {
	cursor := ""
	if focused {
		cursor = "█"
	}
	prompt := queryPromptStyle.Render("❯ ")
	q := queryTextStyle.Render(query + cursor)
	return fmt.Sprintf("  %s%s", prompt, q)
}
