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

func RenderPackageList(packages []model.Package, selected int, offset int, maxVisible int, width int, selectedSet map[string]bool) string {
	if len(packages) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  Nenhum pacote encontrado.\n")
	}

	// prefix takes: cursor(3) + space(1) + selMarker(3) + space(1) + badge(2) + space(1) = ~11
	prefixW := 11
	available := width - prefixW - 4 // 4 for column gaps (2 between each)
	if available < 40 {
		available = 40
	}
	// Proportional columns: Name ~50%, Version ~35%, Size ~15%
	colName := available * 50 / 100
	colVersion := available * 35 / 100
	colSize := available - colName - colVersion
	if colName < 20 {
		colName = 20
	}
	if colVersion < 12 {
		colVersion = 12
	}
	if colSize < 8 {
		colSize = 8
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))

	var b strings.Builder

	padName := colName - 4
	if padName < 0 {
		padName = 0
	}
	padVer := colVersion - 7
	if padVer < 0 {
		padVer = 0
	}
	padSize := colSize - 4
	if padSize < 0 {
		padSize = 0
	}
	header := fmt.Sprintf("%s%s%s  %s%s  %s%s",
		strings.Repeat(" ", prefixW),
		headerStyle.Render("Name"), strings.Repeat(" ", padName),
		headerStyle.Render("Version"), strings.Repeat(" ", padVer),
		strings.Repeat(" ", padSize), headerStyle.Render("Size"))
	b.WriteString(header + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)) + "\n")

	end := offset + maxVisible
	if end > len(packages) {
		end = len(packages)
	}

	for i := offset; i < end; i++ {
		pkg := packages[i]

		selMarker := "  "
		if selectedSet != nil {
			if selectedSet[pkg.Name] {
				selMarker = "[x]"
			} else {
				selMarker = "[ ]"
			}
		}

		badge := "○"
		badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
		if pkg.Upgradable {
			badge = "↑"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC107")).Bold(true)
		} else if pkg.Installed {
			badge = "●"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
		}

		name := pkg.Name
		if len(name) > colName {
			name = name[:colName-1] + "…"
		}

		version := pkg.Version
		if version == "" {
			version = "-"
		}
		if len(version) > colVersion {
			version = version[:colVersion-1] + "…"
		}

		size := pkg.Size
		if size == "" {
			size = "-"
		}

		namePad := colName - len(name)
		if namePad < 0 {
			namePad = 0
		}
		versionPad := colVersion - len(version)
		if versionPad < 0 {
			versionPad = 0
		}
		sizePad := colSize - len(size)
		if sizePad < 0 {
			sizePad = 0
		}

		if i == selected {
			cursor := cursorStyle.Render(" ▌")
			row := fmt.Sprintf("%s %s %s %s%s  %s%s  %s%s\n",
				cursor, selMarker, badgeStyle.Render(badge),
				selectedLine.Render(name), strings.Repeat(" ", namePad),
				dimStyle.Render(version), strings.Repeat(" ", versionPad),
				strings.Repeat(" ", sizePad), dimStyle.Render(size))
			b.WriteString(row)
		} else {
			row := fmt.Sprintf("   %s %s %s%s  %s%s  %s%s\n",
				selMarker, badgeStyle.Render(badge),
				normalLine.Render(name), strings.Repeat(" ", namePad),
				dimStyle.Render(version), strings.Repeat(" ", versionPad),
				strings.Repeat(" ", sizePad), dimStyle.Render(size))
			b.WriteString(row)
		}
	}

	return b.String()
}
