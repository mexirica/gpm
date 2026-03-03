package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mexirica/gpm/internal/model"
)

var (
	selectedLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Bold(true)

	normalLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFC107")).
			Bold(true)

	counterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C6C6C"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A4A4A"))
)

func RenderPackageList(packages []model.Package, selected int, offset int, maxVisible int, width int) string {
	if len(packages) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  Nenhum pacote encontrado.\n")
	}

	var b strings.Builder
	end := offset + maxVisible
	if end > len(packages) {
		end = len(packages)
	}

	for i := offset; i < end; i++ {
		pkg := packages[i]

		badge := " ○"
		badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
		if pkg.Upgradable {
			badge = " ↑"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC107")).Bold(true)
		} else if pkg.Installed {
			badge = " ●"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
		}

		if i == selected {
			cursor := cursorStyle.Render(" ▌")
			name := selectedLine.Render(pkg.Name)
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, badgeStyle.Render(badge), name))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s\n", badgeStyle.Render(badge), normalLine.Render(pkg.Name)))
		}
	}

	total := len(packages)
	counter := counterStyle.Render(fmt.Sprintf("  %d/%d", total, total))
	b.WriteString(counter + "\n")

	sep := separatorStyle.Render(strings.Repeat("─", width))
	b.WriteString(sep)

	return b.String()
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
