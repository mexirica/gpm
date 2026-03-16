package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/model"
)

var (
	selectedLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Bold(true)

	normalLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B0B0C0"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8888AA"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C6C8A"))

	selCheckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	selUncheckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A4A5A"))
)

func RenderPackageList(packages []model.Package, selected int, offset int, maxVisible int, width int, selectedSet map[string]bool, sortCol ...filter.SortInfo) string {
	if len(packages) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  No packages found.\n")
	}

	// prefix takes: cursor(3) + space(1) + selMarker(3) + space(1) + badge(26) + space(1) = ~11
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
	sortIndicatorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFC107"))

	// Determine sort indicator
	var si filter.SortInfo
	if len(sortCol) > 0 {
		si = sortCol[0]
	}
	sortArrow := func(col filter.SortColumn) string {
		if si.Column == col && col != filter.SortNone {
			if si.Desc {
				return " " + sortIndicatorStyle.Render("▼")
			}
			return " " + sortIndicatorStyle.Render("▲")
		}
		return ""
	}

	var b strings.Builder

	nameArrow := sortArrow(filter.SortName)
	versionArrow := sortArrow(filter.SortVersion)
	sizeArrow := sortArrow(filter.SortSize)

	padName := colName - 4
	if nameArrow != "" {
		padName -= 2 // arrow + space
	}
	if padName < 0 {
		padName = 0
	}
	padVer := colVersion - 7
	if versionArrow != "" {
		padVer -= 2
	}
	if padVer < 0 {
		padVer = 0
	}
	padSize := colSize - 4
	if sizeArrow != "" {
		padSize -= 2
	}
	if padSize < 0 {
		padSize = 0
	}
	header := fmt.Sprintf("%s%s%s%s  %s%s%s  %s%s%s",
		strings.Repeat(" ", prefixW),
		headerStyle.Render("Name"), nameArrow, strings.Repeat(" ", padName),
		headerStyle.Render("Version"), versionArrow, strings.Repeat(" ", padVer),
		strings.Repeat(" ", padSize), headerStyle.Render("Size"), sizeArrow)
	b.WriteString(header + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(strings.Repeat("─", width)) + "\n")

	end := offset + maxVisible
	if end > len(packages) {
		end = len(packages)
	}

	for i := offset; i < end; i++ {
		pkg := packages[i]

		selMarker := "  "
		if selectedSet != nil {
			if selectedSet[pkg.Name] {
				selMarker = selCheckStyle.Render("[x]")
			} else {
				selMarker = selUncheckStyle.Render("[ ]")
			}
		}

		badge := "○"
		badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
		if pkg.Held {
			badge = "🔒"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")).Bold(true)
		} else if pkg.Upgradable && pkg.SecurityUpdate {
			badge = "⚠"
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Bold(true)
		} else if pkg.Upgradable {
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
		if pkg.NewVersion != "" {
			version = pkg.NewVersion
		}
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

		lineNameStyle := normalLine
		lineVersionStyle := versionStyle
		lineSizeStyle := sizeStyle
		if pkg.Held {
			heldDim := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
			lineNameStyle = heldDim
			lineVersionStyle = heldDim
			lineSizeStyle = heldDim
		}

		if i == selected {
			cursor := cursorStyle.Render(" \u258c")
			selName := selectedLine
			if pkg.Held {
				selName = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Bold(true)
			}
			row := fmt.Sprintf("%s %s %s %s%s  %s%s  %s%s\n",
				cursor, selMarker, badgeStyle.Render(badge),
				selName.Render(name), strings.Repeat(" ", namePad),
				lineVersionStyle.Render(version), strings.Repeat(" ", versionPad),
				strings.Repeat(" ", sizePad), lineSizeStyle.Render(size))
			b.WriteString(row)
		} else {
			row := fmt.Sprintf("   %s %s %s%s  %s%s  %s%s\n",
				selMarker, badgeStyle.Render(badge),
				lineNameStyle.Render(name), strings.Repeat(" ", namePad),
				lineVersionStyle.Render(version), strings.Repeat(" ", versionPad),
				strings.Repeat(" ", sizePad), lineSizeStyle.Render(size))
			b.WriteString(row)
		}
	}

	return b.String()
}
