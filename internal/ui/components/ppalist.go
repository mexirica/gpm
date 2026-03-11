package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mexirica/aptui/internal/apt"
)

var (
	ppaNameStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BCD4")).Bold(true)
	ppaURLStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	ppaEnabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	ppaDisabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Bold(true)
	ppaHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	ppaDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
)

func RenderPPAList(ppas []apt.PPA, selected int, offset int, maxVisible int, width int) string {
	if len(ppas) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  No PPA repositories found.\n  Press 'a' to add one.\n")
	}

	colStatus := 10
	colName := 30
	prefixW := 4
	colURL := width - prefixW - colStatus - colName - 6
	if colURL < 15 {
		colURL = 15
	}

	var b strings.Builder

	header := fmt.Sprintf("%s%s  %s%s  %s",
		strings.Repeat(" ", prefixW),
		ppaHeaderStyle.Render("Status"), strings.Repeat(" ", colStatus-6),
		ppaHeaderStyle.Render("Name"), strings.Repeat(" ", colName-4)+ppaHeaderStyle.Render("URL"))
	b.WriteString(header + "\n")

	end := offset + maxVisible
	if end > len(ppas) {
		end = len(ppas)
	}

	for i := offset; i < end; i++ {
		p := ppas[i]

		statusStr := "✔ enabled"
		stStyle := ppaEnabledStyle
		if !p.Enabled {
			statusStr = "✘ disabled"
			stStyle = ppaDisabledStyle
		}

		nameStr := p.Name
		if len(nameStr) > colName {
			nameStr = nameStr[:colName-1] + "…"
		}

		urlStr := p.URL
		if len(urlStr) > colURL {
			urlStr = urlStr[:colURL-1] + "…"
		}

		statusPad := colStatus - len(statusStr) + 4
		if statusPad < 0 {
			statusPad = 0
		}
		namePad := colName - len(nameStr)
		if namePad < 0 {
			namePad = 0
		}

		if i == selected {
			cursor := cursorStyle.Render(" ▌")
			row := fmt.Sprintf("%s %s%s  %s%s  %s\n",
				cursor,
				stStyle.Render(statusStr), strings.Repeat(" ", statusPad),
				ppaNameStyle.Render(nameStr), strings.Repeat(" ", namePad),
				ppaURLStyle.Render(urlStr))
			b.WriteString(row)
		} else {
			row := fmt.Sprintf("    %s%s  %s%s  %s\n",
				stStyle.Render(statusStr), strings.Repeat(" ", statusPad),
				ppaDimStyle.Render(nameStr), strings.Repeat(" ", namePad),
				ppaDimStyle.Render(urlStr))
			b.WriteString(row)
		}
	}

	return b.String()
}

func RenderPPAFooterHelp() string {
	return "a: add • r: remove • esc: back • q: quit"
}
