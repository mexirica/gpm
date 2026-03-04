package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mexirica/gpm/internal/history"
)

var (
	histHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	histIDStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC107")).Bold(true)
	histOpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	histDateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	histPkgStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	histFailStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Bold(true)
	histDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
)

// RenderHistoryList renders the full-screen history view.
func RenderHistoryList(transactions []history.Transaction, selected int, offset int, maxVisible int, width int) string {
	if len(transactions) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  No transaction history yet.\n")
	}

	// Column widths
	colID := 6
	colOp := 14
	colDate := 21
	prefixW := 4 // cursor
	colPkgs := width - prefixW - colID - colOp - colDate - 8
	if colPkgs < 15 {
		colPkgs = 15
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%s%s  %s%s  %s%s  %s",
		strings.Repeat(" ", prefixW),
		histHeaderStyle.Render("ID"), strings.Repeat(" ", colID-2),
		histHeaderStyle.Render("Operation"), strings.Repeat(" ", colOp-9),
		histHeaderStyle.Render("Date"),
		strings.Repeat(" ", colDate-4)+histHeaderStyle.Render("Packages"))
	b.WriteString(header + "\n")
	b.WriteString(histDimStyle.Render(strings.Repeat("─", width)) + "\n")

	end := offset + maxVisible
	if end > len(transactions) {
		end = len(transactions)
	}

	for i := offset; i < end; i++ {
		tx := transactions[i]

		idStr := fmt.Sprintf("#%-4d", tx.ID)

		opStr := string(tx.Operation)
		opStyle := histOpStyle
		if tx.Operation == history.OpRemove {
			opStyle = histFailStyle
		}
		if len(opStr) > colOp {
			opStr = opStr[:colOp-1] + "…"
		}

		dateStr := history.FormatTimestamp(tx.Timestamp)

		pkgStr := ""
		if len(tx.Packages) == 1 {
			pkgStr = tx.Packages[0]
		} else if len(tx.Packages) <= 3 {
			pkgStr = strings.Join(tx.Packages, ", ")
		} else {
			pkgStr = fmt.Sprintf("%s, %s +%d more",
				tx.Packages[0], tx.Packages[1], len(tx.Packages)-2)
		}
		if len(pkgStr) > colPkgs {
			pkgStr = pkgStr[:colPkgs-1] + "…"
		}

		statusMark := histOpStyle.Render("✔")
		if !tx.Success {
			statusMark = histFailStyle.Render("✘")
		}

		opPad := colOp - len(opStr)
		if opPad < 0 {
			opPad = 0
		}
		datePad := colDate - len(dateStr)
		if datePad < 0 {
			datePad = 0
		}

		if i == selected {
			cursor := cursorStyle.Render(" ▌")
			row := fmt.Sprintf("%s %s %s %s%s  %s%s  %s\n",
				cursor,
				histIDStyle.Render(idStr),
				statusMark,
				opStyle.Render(opStr), strings.Repeat(" ", opPad),
				histDateStyle.Render(dateStr), strings.Repeat(" ", datePad),
				histPkgStyle.Render(pkgStr))
			b.WriteString(row)
		} else {
			row := fmt.Sprintf("    %s %s %s%s  %s%s  %s\n",
				histDimStyle.Render(idStr),
				statusMark,
				opStyle.Render(opStr), strings.Repeat(" ", opPad),
				histDateStyle.Render(dateStr), strings.Repeat(" ", datePad),
				histDimStyle.Render(pkgStr))
			b.WriteString(row)
		}
	}

	return b.String()
}

// RenderHistoryDetail renders a detailed view of a single transaction.
func RenderHistoryDetail(tx history.Transaction, width int, maxLines int) string {
	lbl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).Bold(true).Width(16).Align(lipgloss.Right)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A4A4A"))
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("ID"), sep.Render(":"), val.Render(fmt.Sprintf("#%d", tx.ID))))
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Operation"), sep.Render(":"), val.Render(string(tx.Operation))))
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Date"), sep.Render(":"), val.Render(history.FormatTimestamp(tx.Timestamp))))

	status := "Success"
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	if !tx.Success {
		status = "Failed"
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Bold(true)
	}
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Status"), sep.Render(":"), statusStyle.Render(status)))

	pkgLabel := fmt.Sprintf("Packages (%d)", len(tx.Packages))
	b.WriteString(fmt.Sprintf("  %s %s ", lbl.Render(pkgLabel), sep.Render(":")))
	remaining := maxLines - 5
	if remaining < 1 {
		remaining = 1
	}
	for idx, pkg := range tx.Packages {
		if idx >= remaining {
			b.WriteString(fmt.Sprintf("\n  %s   +%d more...", strings.Repeat(" ", 16), len(tx.Packages)-idx))
			break
		}
		if idx == 0 {
			b.WriteString(val.Render(pkg) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s   %s\n", strings.Repeat(" ", 16), val.Render(pkg)))
		}
	}

	return b.String()
}
