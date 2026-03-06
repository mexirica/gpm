package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (a App) onSearchKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return a.submitSearch()
	case "esc":
		return a.cancelSearch()
	default:
		return a.updateSearchFilter(msg)
	}
}

func (a App) submitSearch() (tea.Model, tea.Cmd) {
	query := a.searchInput.Value()
	a.searching = false
	a.searchInput.Blur()
	a.filterQuery = query
	if query == "" {
		a.applyFilter()
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		if len(a.filtered) > 0 {
			return a, showPackageDetailCmd(a.filtered[0].Name)
		}
		return a, nil
	}
	if len(a.filtered) == 0 {
		a.loading = true
		a.status = fmt.Sprintf("Searching '%s' via apt-cache...", query)
		return a, searchPackagesCmd(query)
	}
	a.status = fmt.Sprintf("%d packages matching '%s'", len(a.filtered), query)
	return a, tea.Batch(showPackageDetailCmd(a.filtered[0].Name), a.preloadVisiblePackageInfo())
}

func (a App) cancelSearch() (tea.Model, tea.Cmd) {
	a.searching = false
	a.searchInput.Blur()
	a.filterQuery = ""
	a.applyFilter()
	a.status = fmt.Sprintf("%d packages ", len(a.filtered))
	return a, nil
}

func (a App) updateSearchFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.searchInput, cmd = a.searchInput.Update(msg)
	a.filterQuery = a.searchInput.Value()
	a.applyFilter()
	a.status = fmt.Sprintf("%d matching ", len(a.filtered))
	var detailCmd tea.Cmd
	if len(a.filtered) > 0 {
		detailCmd = showPackageDetailCmd(a.filtered[0].Name)
	}
	return a, tea.Batch(cmd, detailCmd, a.preloadVisiblePackageInfo())
}
