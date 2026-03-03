package components

import (
	"github.com/charmbracelet/lipgloss"
)

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6C6C6C")).
	Padding(0, 1)

func RenderStatusBar(status string, width int) string {
	return statusBarStyle.Width(width).Render(status)
}
