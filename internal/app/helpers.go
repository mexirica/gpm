package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/fuzzy"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
)

type scoredPackage struct {
	pkg   model.Package
	score int
}

// tabStyle returns the appropriate style for a tab given the current state.
func (a App) tabStyle(t tabDef) lipgloss.Style {
	if t.kind == a.activeTab {
		return ui.TabActiveStyle
	}
	if t.kind == tabUpgradable && len(a.upgradableMap) > 0 {
		return ui.TabNotifyStyle
	}
	return ui.TabInactiveStyle
}

// activateTab switches to the given tab and returns the commands to refresh the view.
func (a *App) activateTab() tea.Cmd {
	a.applyFilter()
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	a.status = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabDefs[a.activeTab].name)
	return tea.Batch(cmds...)
}

// applyFilter rebuilds the filtered list from allPackages based on active tab,
// advanced filter, and search query. Uses fuzzy scoring when a search query is active.
func (a *App) applyFilter() {
	var source []model.Package
	switch a.activeTab {
	case tabInstalled:
		for _, p := range a.allPackages {
			if p.Installed {
				source = append(source, p)
			}
		}
	case tabUpgradable:
		for _, p := range a.allPackages {
			if p.Upgradable {
				source = append(source, p)
			}
		}
	default:
		source = a.allPackages
	}

	// Apply advanced filter if set
	af := filter.Parse(a.advancedFilter)
	if !af.IsEmpty() {
		var filtered []model.Package
		for _, p := range source {
			pd := filter.PackageData{
				Name:         p.Name,
				Version:      p.Version,
				NewVersion:   p.NewVersion,
				Size:         p.Size,
				Description:  p.Description,
				Installed:    p.Installed,
				Upgradable:   p.Upgradable,
				Section:      p.Section,
				Architecture: p.Architecture,
			}
			if af.Match(pd) {
				filtered = append(filtered, p)
			}
		}
		source = filtered
	}

	if a.filterQuery == "" {
		a.filtered = source
	} else {
		minScore := fuzzy.MinQuality(len(a.filterQuery))
		var scored []scoredPackage
		for _, p := range source {
			nameRes := fuzzy.Score(a.filterQuery, p.Name)
			descRes := fuzzy.Score(a.filterQuery, p.Description)

			s := 0
			matched := false
			if nameRes.Matched {
				matched = true
				s = nameRes.Score + 50
			}
			if descRes.Matched && descRes.Score > s {
				matched = true
				s = descRes.Score
			}

			if matched && s >= minScore {
				scored = append(scored, scoredPackage{pkg: p, score: s})
			}
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		a.filtered = make([]model.Package, len(scored))
		for i, sp := range scored {
			a.filtered[i] = sp.pkg
		}
	}

	// Apply sorting: click-based sort takes priority over filter-based sort
	sortCol := af.OrderBy
	sortDesc := af.OrderDesc
	if a.sortColumn != filter.SortNone {
		sortCol = a.sortColumn
		sortDesc = a.sortDesc
	}
	if sortCol != filter.SortNone {
		sort.SliceStable(a.filtered, func(i, j int) bool {
			pi, pj := a.filtered[i], a.filtered[j]

			// Push packages with unknown data to the end
			iEmpty, jEmpty := sortFieldEmpty(pi, sortCol), sortFieldEmpty(pj, sortCol)
			if iEmpty != jEmpty {
				return !iEmpty // non-empty comes first
			}
			if iEmpty && jEmpty {
				return false // both empty, keep original order
			}

			var less bool
			switch sortCol {
			case filter.SortName:
				less = strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
			case filter.SortVersion:
				less = pi.Version < pj.Version
			case filter.SortSize:
				less = filter.ParseSizeToKB(pi.Size) < filter.ParseSizeToKB(pj.Size)
			case filter.SortSection:
				less = strings.ToLower(pi.Section) < strings.ToLower(pj.Section)
			case filter.SortArchitecture:
				less = strings.ToLower(pi.Architecture) < strings.ToLower(pj.Architecture)
			default:
				return false
			}
			if sortDesc {
				return !less
			}
			return less
		})
	}

	a.selectedIdx = 0
	a.scrollOffset = 0
}

// effectiveSortInfo returns the active sort state, preferring click-based sort
// over filter-based sort.
func (a App) effectiveSortInfo() filter.SortInfo {
	if a.sortColumn != filter.SortNone {
		return filter.SortInfo{Column: a.sortColumn, Desc: a.sortDesc}
	}
	af := filter.Parse(a.advancedFilter)
	return filter.SortInfo{Column: af.OrderBy, Desc: af.OrderDesc}
}

func sortFieldEmpty(p model.Package, col filter.SortColumn) bool {
	switch col {
	case filter.SortName:
		return p.Name == ""
	case filter.SortVersion:
		return p.Version == "" && p.NewVersion == ""
	case filter.SortSize:
		return p.Size == "" || p.Size == "-"
	case filter.SortSection:
		return p.Section == ""
	case filter.SortArchitecture:
		return p.Architecture == ""
	default:
		return false
	}
}

// loadFilterCandidateInfo fetches metadata only for packages that pass all
// non-metadata filters but are missing metadata needed by the active filter.
// This is much faster than loading ALL packages since non-metadata filters
// (name, version, installed, etc.) narrow the set first.
func (a *App) loadFilterCandidateInfo() tea.Cmd {
	af := filter.Parse(a.advancedFilter)
	if !af.NeedsMetadata() {
		return nil
	}

	var names []string
	for _, p := range a.allPackages {
		// Skip packages already cached
		if _, ok := a.infoCache[p.Name]; ok {
			continue
		}
		// Skip packages that already have metadata populated
		if p.Section != "" || p.Architecture != "" || p.Size != "" {
			continue
		}
		// Only load metadata for packages that pass the non-metadata filters
		pd := filter.PackageData{
			Name:        p.Name,
			Version:     p.Version,
			NewVersion:  p.NewVersion,
			Description: p.Description,
			Installed:   p.Installed,
			Upgradable:  p.Upgradable,
		}
		if af.MatchWithoutMetadata(pd) {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return func() tea.Msg {
		info := apt.BatchGetInfo(names)
		return infoLoadedMsg{info: info}
	}
}

// preloadVisiblePackageInfo fetches version/size info for packages near the visible
// viewport (±20/+50 rows) that aren't already cached.
func (a *App) preloadVisiblePackageInfo() tea.Cmd {
	if len(a.filtered) == 0 {
		return nil
	}
	h := a.packageListHeight()
	start := a.scrollOffset
	end := start + h + 50
	if start > 20 {
		start -= 20
	} else {
		start = 0
	}
	if end > len(a.filtered) {
		end = len(a.filtered)
	}
	var names []string
	for i := start; i < end; i++ {
		name := a.filtered[i].Name
		if _, ok := a.infoCache[name]; !ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return func() tea.Msg {
		info := apt.BatchGetInfo(names)
		return infoLoadedMsg{info: info}
	}
}

func (a *App) adjustPackageScroll() {
	h := a.packageListHeight()
	if a.selectedIdx < a.scrollOffset {
		a.scrollOffset = a.selectedIdx
	}
	if a.selectedIdx >= a.scrollOffset+h {
		a.scrollOffset = a.selectedIdx - h + 1
	}
}

func (a *App) adjustMirrorScroll() {
	h := a.packageListHeight()
	if a.fetchIdx < a.fetchOffset {
		a.fetchOffset = a.fetchIdx
	}
	if a.fetchIdx >= a.fetchOffset+h {
		a.fetchOffset = a.fetchIdx - h + 1
	}
}

func (a *App) adjustTransactionScroll() {
	h := a.transactionListHeight()
	if a.transactionIdx < a.transactionOffset {
		a.transactionOffset = a.transactionIdx
	}
	if a.transactionIdx >= a.transactionOffset+h {
		a.transactionOffset = a.transactionIdx - h + 1
	}
}

// searchBarY returns the Y coordinate of the search bar row.
func (a App) searchBarY() int {
	helpLines := strings.Count(a.help.View(a.keys), "\n") + 1
	filterLines := 0
	if a.filtering || a.advancedFilter != "" {
		filterLines = 1
	}
	if !a.loading && len(a.filtered) > 0 {
		detailLines := a.packageDetailHeight()
		if a.detailName == "" || a.detailInfo == "" {
			pkg := a.filtered[a.selectedIdx]
			detailLines = strings.Count(a.renderBasicDetail(pkg), "\n")
		}
		return a.height - 4 - filterLines - detailLines - helpLines
	}
	return a.height - 3 - filterLines - helpLines
}

func (a App) packageListHeight() int {
	helpLines := strings.Count(a.help.View(a.keys), "\n") + 1
	h := a.height - a.packageDetailHeight() - 9 - helpLines
	if h < 5 {
		h = 5
	}
	return h
}

func (a App) packageDetailHeight() int {
	return 10
}

func (a App) transactionListHeight() int {
	helpLines := strings.Count(a.help.View(a.keys), "\n") + 1
	footerLines := 2 + helpLines
	innerH := a.height - 3 - footerLines
	if innerH < 5 {
		innerH = 5
	}
	mv := innerH - 1
	if mv < 3 {
		mv = 3
	}
	return mv
}
