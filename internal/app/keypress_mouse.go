package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/filter"
)

const (
	// packageListHeaderY is the Y position of the column header row.
	// Layout: tabBar(1) + newline(1) = header at row 2
	packageListHeaderY = 2
	// packageListStartY is where the first package row begins.
	// header(1) + separator(1) after headerY = 4
	packageListStartY = 4
)

func (a App) onMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return a, nil
	}

	y := msg.Y

	// Click on column header area (rows 1-3) → toggle sort
	if y >= packageListHeaderY-1 && y <= packageListHeaderY+1 {
		return a.onHeaderClick(msg.X)
	}

	row := y - packageListStartY
	if row < 0 {
		return a, nil
	}

	idx := a.scrollOffset + row
	if idx < 0 || idx >= len(a.filtered) {
		return a, nil
	}

	// If clicking the already-selected row, toggle its selection (check/uncheck)
	if idx == a.selectedIdx {
		if a.selected == nil {
			a.selected = make(map[string]bool)
		}
		pkg := a.filtered[idx]
		if a.selected[pkg.Name] {
			delete(a.selected, pkg.Name)
		} else {
			a.selected[pkg.Name] = true
		}
		a.status = fmt.Sprintf("%d selected ", len(a.selected))
		return a, nil
	}

	// Move cursor to clicked row
	a.selectedIdx = idx
	a.adjustPackageScroll()
	return a, showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
}

// onHeaderClick maps an X coordinate to a column and toggles sorting.
func (a App) onHeaderClick(x int) (tea.Model, tea.Cmd) {
	prefixW := 11
	available := a.width - prefixW - 4
	if available < 40 {
		available = 40
	}
	colName := available * 50 / 100
	colVersion := available * 35 / 100
	if colName < 20 {
		colName = 20
	}
	if colVersion < 12 {
		colVersion = 12
	}

	// Column boundaries (accounting for prefix and 2-char gaps)
	nameStart := prefixW
	nameEnd := nameStart + colName
	versionStart := nameEnd + 2
	versionEnd := versionStart + colVersion
	sizeStart := versionEnd + 2

	var clicked filter.SortColumn
	switch {
	case x >= nameStart && x < nameEnd:
		clicked = filter.SortName
	case x >= versionStart && x < versionEnd:
		clicked = filter.SortVersion
	case x >= sizeStart:
		clicked = filter.SortSize
	default:
		return a, nil
	}

	// Toggle: same column → flip direction; different column → ascending
	if a.sortColumn == clicked {
		if a.sortDesc {
			// Already descending → clear sort
			a.sortColumn = filter.SortNone
			a.sortDesc = false
		} else {
			a.sortDesc = true
		}
	} else {
		a.sortColumn = clicked
		a.sortDesc = false
	}

	a.applyFilter()
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[a.selectedIdx].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	return a, tea.Batch(cmds...)
}
