package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/ui"
)

// RenderFetchHeader renders the fetch view title.
func RenderFetchHeader(distro fetch.Distro) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 2)
	return titleStyle.Render(fmt.Sprintf(" Fetch Mirrors — %s (%s) ", distro.Name, distro.Codename))
}

// RenderFetchProgress renders the mirror testing progress bar.
func RenderFetchProgress(tested, total int) string {
	pct := 0
	if total > 0 {
		pct = tested * 100 / total
	}
	barW := 30
	filled := barW * pct / 100
	empty := barW - filled
	bar := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(strings.Repeat("░", empty))
	return fmt.Sprintf("  Testing mirrors... %s %d%% (%d/%d)", bar, pct, tested, total)
}

// RenderMirrorList renders the list of tested mirrors.
func RenderMirrorList(mirrors []fetch.Mirror, selectedIdx, offset, maxLines, width int, selected map[int]bool) string {
	if len(mirrors) == 0 {
		return "  No mirrors found."
	}

	prefixW := 5
	rankW := 5
	statusW := 8
	latencyW := 12
	urlW := width - prefixW - rankW - statusW - latencyW - 2
	if urlW < 20 {
		urlW = 20
	}

	// Header
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)
	header := strings.Repeat(" ", prefixW) +
		hdrStyle.Render(padRight("Rank", rankW)) +
		hdrStyle.Render(padRight("Mirror URL", urlW)) +
		hdrStyle.Render(padRight("Latency", latencyW)) +
		hdrStyle.Render(padRight("Status", statusW))

	var b strings.Builder
	b.WriteString(header + "\n")

	end := offset + maxLines
	if end > len(mirrors) {
		end = len(mirrors)
	}

	for i := offset; i < end; i++ {
		m := mirrors[i]
		isSelected := i == selectedIdx
		isChecked := selected[i]

		prefix := "  "
		if isChecked {
			prefix += lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true).Render("✓") + " "
		} else if isSelected {
			prefix += lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("►") + " "
		} else {
			prefix += "  "
		}

		rank := fmt.Sprintf("#%02d ", i+1)

		url := m.URL
		if len(url) > urlW-1 {
			url = url[:urlW-4] + "..."
		}
		url = padRight(url, urlW)

		var latency string
		switch m.Status {
		case "ok":
			latency = fetch.FormatLatency(m.Latency)
		case "error":
			latency = "—"
		default:
			latency = "..."
		}
		latency = padRight(latency, latencyW)

		var status string
		switch m.Status {
		case "ok":
			status = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("● ok")
		case "error":
			status = lipgloss.NewStyle().Foreground(ui.ColorDanger).Render("✗ err")
		case "slow":
			status = lipgloss.NewStyle().Foreground(ui.ColorWarning).Render("◑ slow")
		default:
			status = lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render("… test")
		}

		line := prefix + rank + url + latency + status

		if isSelected {
			line = lipgloss.NewStyle().Background(lipgloss.Color("#2A2A5E")).Foreground(ui.ColorWhite).Render(line)
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func RenderFetchFooterHelp() string {
	return "  space: toggle • enter: apply selected • esc: cancel • j/k: navigate"
}
