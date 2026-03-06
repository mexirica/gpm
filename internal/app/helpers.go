package app

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fuzzy"
	"github.com/mexirica/aptui/internal/model"
)

type scoredPackage struct {
	pkg   model.Package
	score int
}

// applyFilter rebuilds the filtered list from allPackages based on active tab and search query.
// Uses fuzzy scoring when a search query is active.
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
	a.selectedIdx = 0
	a.scrollOffset = 0
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
