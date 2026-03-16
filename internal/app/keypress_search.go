package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/filter"
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
		// Fallback to apt-cache search using free text portion
		af := filter.Parse(query)
		searchTerm := af.FreeText
		if searchTerm == "" {
			searchTerm = query
		}
		a.loading = true
		a.status = fmt.Sprintf("Searching '%s' via apt-cache...", searchTerm)
		return a, searchPackagesCmd(searchTerm)
	}
	a.status = fmt.Sprintf("%d packages matching '%s'", len(a.filtered), query)
	return a, showPackageDetailCmd(a.filtered[0].Name)
}

func (a App) cancelSearch() (tea.Model, tea.Cmd) {
	a.searching = false
	a.searchInput.Blur()
	// Restore filterQuery to what it was before opening search
	// (searchInput was set to filterQuery on open, so if user edited
	// and then cancelled, we keep the original)
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
	return a, tea.Batch(cmd, detailCmd)
}
