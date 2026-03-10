package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/aptui/internal/filter"
)

const (
	packageListHeaderY = 1
	packageListStartY  = 3
)

func (a App) onMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		a.selectedIdx -= 3
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.adjustPackageScroll()
		if len(a.filtered) > 0 {
			return a, showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		a.selectedIdx += 3
		if a.selectedIdx >= len(a.filtered) {
			a.selectedIdx = len(a.filtered) - 1
		}
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.adjustPackageScroll()
		if len(a.filtered) > 0 {
			return a, showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return a, nil
	}

	y := msg.Y

	// Click on tab bar (row 0) → switch tab
	if y == 0 {
		return a.onTabClick(msg.X)
	}

	// Click on column header/separator area → toggle sort
	if y >= packageListHeaderY && y < packageListStartY {
		return a.onHeaderClick(msg.X)
	}

	if y == a.searchBarY() && !a.searching {
		return a.openSearch()
	}

	row := y - packageListStartY
	if row < 0 || row >= a.packageListHeight() {
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

func (a App) onTabClick(x int) (tea.Model, tea.Cmd) {
	pos := 0
	for _, tab := range tabDefs {
		w := lipgloss.Width(a.tabStyle(tab).Render(tab.label))
		if x >= pos && x < pos+w {
			if tab.kind == a.activeTab {
				return a, nil
			}
			a.activeTab = tab.kind
			return a, a.activateTab()
		}
		pos += w
	}
	return a, nil
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
