package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/filter"
)

func (a App) onFilterKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return a.submitFilter()
	case "esc":
		return a.cancelFilter()
	default:
		return a.updateFilterInput(msg)
	}
}

func (a App) submitFilter() (tea.Model, tea.Cmd) {
	query := a.filterInput.Value()
	a.filtering = false
	a.filterInput.Blur()
	a.advancedFilter = query
	a.applyFilter()

	var cmds []tea.Cmd

	// If filter uses metadata fields, load info only for narrowed candidates
	af := filter.Parse(query)
	if af.NeedsMetadata() {
		if cmd := a.loadFilterCandidateInfo(); cmd != nil {
			cmds = append(cmds, cmd)
			a.status = fmt.Sprintf("%d packages matching filter (loading metadata...)", len(a.filtered))
		} else if len(a.filtered) == 0 && query != "" {
			a.status = fmt.Sprintf("No packages match filter: %s", query)
		} else if query != "" {
			a.status = fmt.Sprintf("%d packages matching filter", len(a.filtered))
		} else {
			a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		}
	} else {
		if len(a.filtered) == 0 && query != "" {
			a.status = fmt.Sprintf("No packages match filter: %s", query)
		} else if query != "" {
			a.status = fmt.Sprintf("%d packages matching filter", len(a.filtered))
		} else {
			a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		}
	}

	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	return a, tea.Batch(cmds...)
}

func (a App) cancelFilter() (tea.Model, tea.Cmd) {
	a.filtering = false
	a.filterInput.Blur()
	a.filterInput.SetValue(a.advancedFilter)
	return a, nil
}

func (a App) updateFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.filterInput, cmd = a.filterInput.Update(msg)
	return a, cmd
}
