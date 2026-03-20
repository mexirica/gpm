package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mexirica/aptui/internal/errlog"
)

var (
	errIDStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC107")).Bold(true)
	errSrcStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BCD4")).Bold(true)
	errDateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	errMsgStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	errDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	errHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
)

func RenderErrorLogList(entries []errlog.Entry, selected int, offset int, maxVisible int, width int) string {
	if len(entries) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C")).
			Render("\n  No errors logged.\n")
	}

	colID := 6
	colSrc := 16
	colDate := 21
	prefixW := 4
	colMsg := width - prefixW - colID - colSrc - colDate - 8
	if colMsg < 15 {
		colMsg = 15
	}

	var b strings.Builder

	header := fmt.Sprintf("%s%s  %s%s  %s%s  %s",
		strings.Repeat(" ", prefixW),
		errHeaderStyle.Render("ID"), strings.Repeat(" ", colID-2),
		errHeaderStyle.Render("Source"), strings.Repeat(" ", colSrc-6),
		errHeaderStyle.Render("Date"),
		strings.Repeat(" ", colDate-4)+errHeaderStyle.Render("Message"))
	b.WriteString(header + "\n")

	end := offset + maxVisible
	if end > len(entries) {
		end = len(entries)
	}

	for i := offset; i < end; i++ {
		e := entries[i]

		idStr := fmt.Sprintf("#%-4d", e.ID)

		srcStr := e.Source
		if len(srcStr) > colSrc {
			srcStr = srcStr[:colSrc-1] + "…"
		}

		dateStr := errlog.FormatTimestamp(e.Timestamp)

		msgStr := e.Message
		if len(msgStr) > colMsg {
			msgStr = msgStr[:colMsg-1] + "…"
		}

		srcPad := colSrc - len(srcStr)
		if srcPad < 0 {
			srcPad = 0
		}
		datePad := colDate - len(dateStr)
		if datePad < 0 {
			datePad = 0
		}

		if i == selected {
			cursor := cursorStyle.Render(" ▌")
			row := fmt.Sprintf("%s %s %s%s  %s%s  %s\n",
				cursor,
				errIDStyle.Render(idStr),
				errSrcStyle.Render(srcStr), strings.Repeat(" ", srcPad),
				errMsgStyle.Render(dateStr), strings.Repeat(" ", datePad),
				errMsgStyle.Render(msgStr))
			b.WriteString(row)
		} else {
			row := fmt.Sprintf("    %s %s%s  %s%s  %s\n",
				errDimStyle.Render(idStr),
				errSrcStyle.Render(srcStr), strings.Repeat(" ", srcPad),
				errDateStyle.Render(dateStr), strings.Repeat(" ", datePad),
				errDimStyle.Render(msgStr))
			b.WriteString(row)
		}
	}

	return b.String()
}

func RenderErrorLogDetail(entry errlog.Entry, width int) string {
	lbl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).Bold(true)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A4A4A"))
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))

	var b strings.Builder
	fmt.Fprintf(&b, " %s %s %s\n", lbl.Render("ID"), sep.Render(":"), val.Render(fmt.Sprintf("#%d", entry.ID)))
	fmt.Fprintf(&b, " %s %s %s\n", lbl.Render("Source"), sep.Render(":"), errSrcStyle.Render(entry.Source))
	fmt.Fprintf(&b, " %s %s %s\n", lbl.Render("Date"), sep.Render(":"), val.Render(errlog.FormatTimestamp(entry.Timestamp)))

	// Wrap message
	msgLabel := "Message"
	prefix := fmt.Sprintf(" %s %s ", lbl.Render(msgLabel), sep.Render(":"))
	indent := " " + strings.Repeat(" ", len(msgLabel)+3)
	avail := width - len(msgLabel) - 5
	if avail < 20 {
		avail = 20
	}

	msgLines := wrapText(entry.Message, avail)
	for idx, line := range msgLines {
		if idx == 0 {
			b.WriteString(prefix + val.Render(line) + "\n")
		} else {
			b.WriteString(indent + val.Render(line) + "\n")
		}
	}

	return b.String()
}

func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	var lines []string
	for len(runes) > maxWidth {
		lines = append(lines, string(runes[:maxWidth]))
		runes = runes[maxWidth:]
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}
