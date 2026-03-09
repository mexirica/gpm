package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
	"github.com/mexirica/aptui/internal/ui/components"
)

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	w := a.width

	if a.fetchView {
		return a.renderFetchView(w)
	}

	if a.transactionView {
		return a.renderTransactionView(w)
	}

	tabBar := a.renderTabBar()
	var listView string
	if a.loading {
		listView = fmt.Sprintf("\n  %s Loading...\n", a.spinner.View())
	} else {
		listView = components.RenderPackageList(a.filtered, a.selectedIdx, a.scrollOffset, a.packageListHeight(), w, a.selected)
	}
	listView = tabBar + "\n" + listView

	var footer []string

	counterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888AA"))
	pos := a.selectedIdx + 1
	if len(a.filtered) == 0 {
		pos = 0
	}
	counterText := fmt.Sprintf("  %d/%d", pos, len(a.filtered))
	footer = append(footer, counterStyle.Render(counterText))

	if a.searching {
		footer = append(footer, "  "+a.searchInput.View())
	} else {
		footer = append(footer, components.RenderSearchPrompt(a.filterQuery, false))
	}

	// Advanced filter bar
	if a.filtering {
		footer = append(footer, "  "+a.filterInput.View())
	} else if a.advancedFilter != "" {
		af := filter.Parse(a.advancedFilter)
		footer = append(footer, components.RenderFilterPrompt(af.Describe(), false))
	}

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if !a.loading && len(a.filtered) > 0 && a.detailName != "" && a.detailInfo != "" {
		pkg := a.filtered[a.selectedIdx]
		statusLine := "Status: Not installed"
		if pkg.Upgradable {
			statusLine = "Status: Upgrade available (" + pkg.Version + " → " + pkg.NewVersion + ")"
		} else if pkg.Installed {
			statusLine = "Status: Installed"
		}
		enrichedInfo := statusLine + "\n" + a.detailInfo
		maxDetailLines := a.packageDetailHeight()
		detail := components.RenderPackageDetail(enrichedInfo, w, maxDetailLines, 1)
		footer = append(footer, detail)
	} else if !a.loading && len(a.filtered) > 0 {
		pkg := a.filtered[a.selectedIdx]
		basic := a.renderBasicDetail(pkg)
		footer = append(footer, basic)
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	footer = append(footer, ui.HelpStyle.Render(a.help.View(a.keys)))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)

	listLines := strings.Count(listView, "\n")
	footerLines := strings.Count(footerView, "\n") + 1
	gap := a.height - listLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return listView + strings.Repeat("\n", gap) + footerView
}

func (a App) renderTabBar() string {
	tabs := []struct {
		label string
		kind  tabKind
	}{
		{" ◉ All ", tabAll},
		{" ● Installed ", tabInstalled},
		{" ↑ Upgradable ", tabUpgradable},
	}

	var parts []string
	hasUpgradable := len(a.upgradableMap) > 0
	for _, t := range tabs {
		if t.kind == a.activeTab {
			parts = append(parts, ui.TabActiveStyle.Render(t.label))
		} else if t.kind == tabUpgradable && hasUpgradable {
			parts = append(parts, ui.TabNotifyStyle.Render(t.label))
		} else {
			parts = append(parts, ui.TabInactiveStyle.Render(t.label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (a App) renderBasicDetail(pkg model.Package) string {
	lbl := lipgloss.NewStyle().
		Foreground(ui.ColorWhite).Bold(true).Width(18).Align(lipgloss.Right)
	sepStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Name"), sepStyle.Render(":"), val.Render(pkg.Name)))
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Version"), sepStyle.Render(":"), val.Render(pkg.Version)))

	status := "Not installed"
	statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	if pkg.Upgradable {
		status = "Upgrade available"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
	} else if pkg.Installed {
		status = "Installed"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
	}
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Status"), sepStyle.Render(":"), statusStyle.Render(status)))

	if pkg.NewVersion != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("New Version"), sepStyle.Render(":"),
			lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true).Render(pkg.NewVersion)))
	}
	if pkg.Section != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Section"), sepStyle.Render(":"), val.Render(pkg.Section)))
	}
	if pkg.Architecture != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Architecture"), sepStyle.Render(":"), val.Render(pkg.Architecture)))
	}
	if pkg.Description != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Description"), sepStyle.Render(":"), val.Render(pkg.Description)))
	}

	return b.String()
}

func (a App) renderFetchView(w int) string {
	header := components.RenderFetchHeader(a.fetchDistro)
	var footer []string
	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	sel := len(a.fetchSelected)
	total := len(a.fetchMirrors)
	footer = append(footer, counterStyle.Render(fmt.Sprintf("  %d/%d mirrors selected", sel, total)))

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if !a.fetchTesting && len(a.fetchMirrors) > 0 && a.fetchIdx < len(a.fetchMirrors) {
		m := a.fetchMirrors[a.fetchIdx]
		lbl := lipgloss.NewStyle().Foreground(ui.ColorWhite).Bold(true).Width(14).Align(lipgloss.Right)
		sepChar := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

		var detail strings.Builder
		detail.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("URL"), sepChar.Render(":"), val.Render(m.URL)))
		detail.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Latency"), sepChar.Render(":"), val.Render(fetch.FormatLatency(m.Latency))))
		detail.WriteString(fmt.Sprintf("  %s %s %d\n", lbl.Render("Score"), sepChar.Render(":"), m.Score))
		footer = append(footer, detail.String())
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	helpLine := components.RenderFetchFooterHelp()
	footer = append(footer, lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(helpLine))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)
	footerLines := strings.Count(footerView, "\n") + 1

	var upperView string
	if a.fetchTesting {
		progress := components.RenderFetchProgress(a.fetchTested, a.fetchTotal)
		progLine := fmt.Sprintf("%s %s", a.spinner.View(), progress)

		centeredProg := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(progLine)

		headerLines := strings.Count(header, "\n") + 1
		availLines := a.height - headerLines - footerLines
		if availLines < 1 {
			availLines = 1
		}
		topPad := (availLines - 1) / 2
		if topPad < 0 {
			topPad = 0
		}

		upperView = header + "\n"
		upperView += strings.Repeat("\n", topPad)
		upperView += centeredProg + "\n"
		rem := availLines - topPad - 1
		if rem > 0 {
			upperView += strings.Repeat("\n", rem)
		}
	} else {
		listView := components.RenderMirrorList(a.fetchMirrors, a.fetchIdx, a.fetchOffset, a.packageListHeight(), w, a.fetchSelected)
		upperView = header + "\n" + listView
	}

	listLines := strings.Count(upperView, "\n")
	gap := a.height - listLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return upperView + strings.Repeat("\n", gap) + footerView
}

func (a App) renderTransactionView(w int) string {
	var footerParts []string
	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	footerParts = append(footerParts, counterStyle.Render(fmt.Sprintf("  %d transactions", len(a.transactionItems))))
	footerParts = append(footerParts, components.RenderStatusBar(a.status, w))
	footerParts = append(footerParts, ui.HelpStyle.Render(a.help.View(a.keys)))
	footerView := lipgloss.JoinVertical(lipgloss.Left, footerParts...)
	footerLines := strings.Count(footerView, "\n") + 1

	panelH := a.height - 1 - footerLines
	if panelH < 7 {
		panelH = 7
	}
	leftW := w / 2
	rightW := w - leftW

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary)

	innerH := panelH - 2
	innerLW := leftW - 2
	innerRW := rightW - 2

	maxVisible := innerH - 1
	if maxVisible < 3 {
		maxVisible = 3
	}
	listContent := components.RenderTransactionList(a.transactionItems, a.transactionIdx, a.transactionOffset, maxVisible, innerLW)
	leftPanel := borderStyle.Width(innerLW).Height(innerH).Render(listContent)

	detailTitleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(ui.ColorWhite).Background(ui.ColorPrimary).
		Width(innerRW).Padding(0, 1)
	detailTitle := detailTitleStyle.Render("Transaction Details")

	detailContent := ""
	if len(a.transactionItems) > 0 && a.transactionIdx < len(a.transactionItems) {
		tx := a.transactionItems[a.transactionIdx]
		detailContent = "\n" + components.RenderTransactionDetail(tx, a.transactionDeps, innerRW, innerH-2)
	}
	rightContent := detailTitle + detailContent
	rightPanel := borderStyle.Width(innerRW).Height(innerH).Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	panelLines := strings.Count(panels, "\n") + 1
	gap := a.height - 1 - panelLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return panels + strings.Repeat("\n", gap) + footerView
}
