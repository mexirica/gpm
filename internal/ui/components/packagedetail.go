// Package components provides UI components for the package manager.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	detailLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f2edff")).
			Bold(true).
			Width(18).
			Align(lipgloss.Right)

	detailSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5B3FC4"))

	detailValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D0D0E0"))

	detailMuted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3A3A4A"))
)

var displayFields = []string{
	"Package",
	"Status",
	"Version",
	"Section",
	"Installed-Size",
	"Maintainer",
	"Architecture",
	"Depends",
	"Description",
	"Homepage",
}

func extractFirstEntry(info string) string {
	lines := strings.Split(info, "\n")
	var result []string
	for _, line := range lines {
		if line == "" && len(result) > 0 {
			break
		}
		if line != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func parseFields(info string) map[string]string {
	first := extractFirstEntry(info)
	fields := make(map[string]string)
	lines := strings.Split(first, "\n")
	var lastKey string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if lastKey != "" {
				fields[lastKey] += " " + strings.TrimSpace(line)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			fields[key] = val
			lastKey = key
		}
	}
	return fields
}

func RenderPackageDetail(info string, width int, maxLines int, pageNum int) string {
	if info == "" {
		return detailMuted.Render("  No package selected.")
	}

	fields := parseFields(info)

	maxValW := width - 26
	if maxValW < 20 {
		maxValW = 20
	}

	var rendered []string

	for _, key := range displayFields {
		val, ok := fields[key]
		if !ok || val == "" {
			val = "N/A"
		}

		display := val
		if len(display) > maxValW {
			display = display[:maxValW-3] + "..."
		}

		var line string
		switch key {
		case "Homepage":
			if val == "N/A" {
				line = fmt.Sprintf("  %s %s %s",
					detailLabel.Render(key),
					detailSep.Render(":"),
					detailMuted.Render(display))
			} else {
				line = fmt.Sprintf("  %s %s %s",
					detailLabel.Render(key),
					detailSep.Render(":"),
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00BCD4")).Render(display))
			}
		case "Status":
			statusColor := lipgloss.Color("#6C6C6C")
			if strings.Contains(val, "Upgrade") {
				statusColor = lipgloss.Color("#FFC107")
			} else if strings.Contains(val, "Installed") {
				statusColor = lipgloss.Color("#04B575")
			}
			line = fmt.Sprintf("  %s %s %s",
				detailLabel.Render(key),
				detailSep.Render(":"),
				lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(display))
		default:
			line = fmt.Sprintf("  %s %s %s",
				detailLabel.Render(key),
				detailSep.Render(":"),
				detailValue.Render(display))
		}
		rendered = append(rendered, line)
	}

	if len(rendered) == 0 {
		return detailMuted.Render("  Sem informações disponíveis.") + "\n"
	}

	if maxLines > 0 && len(rendered) > maxLines {
		rendered = rendered[:maxLines]
	}

	var b strings.Builder
	for _, l := range rendered {
		b.WriteString(l + "\n")
	}

	return b.String()
}
