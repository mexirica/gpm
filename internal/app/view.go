package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	lg2 "charm.land/lipgloss/v2"

	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
	"github.com/mexirica/aptui/internal/ui/components"
)

func (a App) View() string {
	if a.width == 0 {
		return fmt.Sprintf("Updating and loading packages %s", a.spinner.View())
	}

	w := a.width

	if a.fetchView {
		return a.renderFetchView(w)
	}

	if a.ppaView {
		return a.renderPPAView(w)
	}

	if a.transactionView {
		return a.renderTransactionView(w)
	}

	tabBar := a.renderTabBar()

	if a.activeTab == tabErrorLog {
		return a.renderErrorLogTab(w, tabBar)
	}

	var listView string
	if a.loading {
		h := a.packageListHeight()
		pad := h / 2
		loadingLine := fmt.Sprintf("Updating and loading packages %s", a.spinner.View())
		centered := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(loadingLine)
		listView = strings.Repeat("\n", pad) + centered + strings.Repeat("\n", h-pad)
	} else {
		si := a.effectiveSortInfo()
		listView = components.RenderPackageList(a.filtered, a.selectedIdx, a.scrollOffset, a.packageListHeight(), w, a.selected, si)
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

	if a.importingPath {
		footer = append(footer, "  Import path: "+a.importInput.View())
	} else if a.searching {
		footer = append(footer, "  "+a.searchInput.View())
	} else {
		footer = append(footer, components.RenderQueryPrompt(a.filterQuery, false))
	}

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if !a.loading && len(a.filtered) > 0 && a.detailName != "" && a.detailInfo != "" {
		pkg := a.filtered[a.selectedIdx]
		statusLine := "Status: Not installed"
		if pkg.Held {
			statusLine = "Status: Held"
		} else if pkg.Upgradable {
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

	page := listView + strings.Repeat("\n", gap) + footerView

	if a.importConfirm {
		bg := lg2.NewLayer(page)

		yKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorSuccess).Padding(0, 1).Render("y")
		nKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorDanger).Padding(0, 1).Render("n")
		hintText := lipgloss.NewStyle().Foreground(ui.ColorSecondary)

		var box string
		if a.importDetails {
			detailTitle := lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorWhite).
				Background(ui.ColorPrimary).
				Padding(0, 2).
				Render(" Packages to Install ")

			const perPage = 15
			const maxBoxWidth = 50
			total := len(a.importToInstall)
			totalPages := (total + perPage - 1) / perPage
			currentPage := a.importDetailOffset + 1

			start := a.importDetailOffset * perPage
			end := start + perPage
			if end > total {
				end = total
			}
			visible := a.importToInstall[start:end]

			nameStyle := lipgloss.NewStyle().Foreground(ui.ColorWhite)
			var lines []string
			for _, name := range visible {
				lines = append(lines, "  "+nameStyle.Render(name))
			}

			pageInfo := lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(
				fmt.Sprintf("Page %d/%d (%d packages)", currentPage, totalPages, total),
			)

			dKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 1).Render("d")
			lKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 1).Render("←")
			rKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 1).Render("→")
			hints := yKey + hintText.Render(" confirm  ") + nKey + hintText.Render(" cancel  ") + dKey + hintText.Render(" back  ") + lKey + rKey + hintText.Render(" page")

			parts := []string{detailTitle, "", pageInfo, ""}
			parts = append(parts, lines...)
			parts = append(parts, "", hints)
			detailContent := lipgloss.JoinVertical(lipgloss.Center, parts...)

			box = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.ColorPrimary).
				Padding(1, 3).
				Align(lipgloss.Center).
				Foreground(ui.ColorWhite).
				Render(detailContent)
		} else {
			title := lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorWhite).
				Background(ui.ColorPrimary).
				Padding(0, 2).
				Render(" Import Packages ")

			countStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorInfo)
			pathStyle := lipgloss.NewStyle().Foreground(ui.ColorSubtle)
			body := fmt.Sprintf(
				"%s packages to install from\n%s",
				countStyle.Render(fmt.Sprintf("%d", len(a.importToInstall))),
				pathStyle.Render(a.importFromPath),
			)

			dKey := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 1).Render("d")
			hints := yKey + hintText.Render(" confirm  ") + nKey + hintText.Render(" cancel  ") + dKey + hintText.Render(" details")

			content := lipgloss.JoinVertical(lipgloss.Center, title, "", body, "", hints)

			box = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.ColorPrimary).
				Padding(1, 3).
				Align(lipgloss.Center).
				Foreground(ui.ColorWhite).
				Render(content)
		}

		boxW := lipgloss.Width(box)
		boxH := lipgloss.Height(box)
		fg := lg2.NewLayer(box).
			X((w - boxW) / 2).
			Y((a.height - boxH) / 2).
			Z(1)
		page = lg2.NewCompositor(bg, fg).Render()
	}

	return page
}

func (a App) renderTabBar() string {
	var parts []string
	for _, t := range tabDefs {
		parts = append(parts, a.tabStyle(t).Render(t.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (a App) renderBasicDetail(pkg model.Package) string {
	lbl := lipgloss.NewStyle().
		Foreground(ui.ColorWhite).Bold(true).Width(18).Align(lipgloss.Right)
	sepStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

	var b strings.Builder
	fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Name"), sepStyle.Render(":"), val.Render(pkg.Name))
	fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Version"), sepStyle.Render(":"), val.Render(pkg.Version))

	status := "Not installed"
	statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	if pkg.Held {
		status = "Held"
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")).Bold(true)
	} else if pkg.Upgradable {
		status = "Upgrade available"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
	} else if pkg.Installed {
		status = "Installed"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
	}

	fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Status"), sepStyle.Render(":"), statusStyle.Render(status))

	if pkg.NewVersion != "" {
		fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("New Version"), sepStyle.Render(":"),
			lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true).Render(pkg.NewVersion))
	}
	if pkg.Section != "" {
		fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Section"), sepStyle.Render(":"), val.Render(pkg.Section))
	}
	if pkg.Architecture != "" {
		fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Architecture"), sepStyle.Render(":"), val.Render(pkg.Architecture))
	}
	if pkg.Description != "" {
		fmt.Fprintf(&b, "  %s %s %s\n", lbl.Render("Description"), sepStyle.Render(":"), val.Render(pkg.Description))
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
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("URL"), sepChar.Render(":"), val.Render(m.URL))
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("Latency"), sepChar.Render(":"), val.Render(fetch.FormatLatency(m.Latency)))
		fmt.Fprintf(&detail, "  %s %s %d\n", lbl.Render("Score"), sepChar.Render(":"), m.Score)
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

func (a App) renderPPAView(w int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Width(w).Padding(0, 1)
	header := titleStyle.Render("PPA Repositories")

	var footer []string
	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	footer = append(footer, counterStyle.Render(fmt.Sprintf("  %d PPA(s)", len(a.ppaItems))))

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if !a.loading && len(a.ppaItems) > 0 && a.ppaIdx < len(a.ppaItems) {
		p := a.ppaItems[a.ppaIdx]
		lbl := lipgloss.NewStyle().Foreground(ui.ColorWhite).Bold(true).Width(14).Align(lipgloss.Right)
		sepChar := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

		var detail strings.Builder
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("Name"), sepChar.Render(":"), val.Render(p.Name))
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("URL"), sepChar.Render(":"), val.Render(p.URL))
		status := "Enabled"
		stStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
		if !p.Enabled {
			status = "Disabled"
			stStyle = lipgloss.NewStyle().Foreground(ui.ColorDanger).Bold(true)
		}
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("Status"), sepChar.Render(":"), stStyle.Render(status))
		fmt.Fprintf(&detail, "  %s %s %s\n", lbl.Render("File"), sepChar.Render(":"), val.Render(p.File))
		footer = append(footer, detail.String())
	}

	if a.ppaAdding {
		footer = append(footer, "  "+a.ppaInput.View())
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	helpLine := components.RenderPPAFooterHelp()
	footer = append(footer, lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(helpLine))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)
	footerLines := strings.Count(footerView, "\n") + 1

	var upperView string
	if a.loading {
		headerLines := strings.Count(header, "\n") + 1
		availLines := a.height - headerLines - footerLines
		if availLines < 1 {
			availLines = 1
		}
		topPad := (availLines - 1) / 2
		if topPad < 0 {
			topPad = 0
		}
		loadingLine := fmt.Sprintf("Loading PPAs %s", a.spinner.View())
		centered := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(loadingLine)
		upperView = header + "\n" + strings.Repeat("\n", topPad) + centered + "\n"
		rem := availLines - topPad - 1
		if rem > 0 {
			upperView += strings.Repeat("\n", rem)
		}
	} else {
		listView := components.RenderPPAList(a.ppaItems, a.ppaIdx, a.ppaOffset, a.packageListHeight(), w)
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

func (a App) renderErrorLogTab(w int, tabBar string) string {
	var footerParts []string
	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	footerParts = append(footerParts, counterStyle.Render(fmt.Sprintf("  %d errors", len(a.errlogItems))))
	footerParts = append(footerParts, components.RenderStatusBar(a.status, w))
	footerParts = append(footerParts, ui.HelpStyle.Render(a.help.View(a.keys)))
	footerView := lipgloss.JoinVertical(lipgloss.Left, footerParts...)
	footerLines := strings.Count(footerView, "\n") + 1

	tabBarLines := strings.Count(tabBar, "\n") + 1
	panelH := a.height - tabBarLines - 1 - footerLines
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
	listContent := components.RenderErrorLogList(a.errlogItems, a.errlogIdx, a.errlogOffset, maxVisible, innerLW)
	leftPanel := borderStyle.Width(innerLW).Height(innerH).Render(listContent)

	detailTitleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(ui.ColorWhite).Background(ui.ColorDanger).
		Width(innerRW).Padding(0, 1)
	detailTitle := detailTitleStyle.Render("Error Details")

	detailContent := ""
	if len(a.errlogItems) > 0 && a.errlogIdx < len(a.errlogItems) {
		entry := a.errlogItems[a.errlogIdx]
		detailContent = "\n" + components.RenderErrorLogDetail(entry, innerRW)
	}
	rightContent := detailTitle + detailContent
	rightPanel := borderStyle.Width(innerRW).Height(innerH).Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	upperView := tabBar + "\n" + panels
	upperLines := strings.Count(upperView, "\n")
	gap := a.height - upperLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return upperView + strings.Repeat("\n", gap) + footerView
}
