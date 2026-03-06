package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/gpm/internal/apt"
	"github.com/mexirica/gpm/internal/fetch"
	"github.com/mexirica/gpm/internal/history"
	"github.com/mexirica/gpm/internal/model"
	"github.com/mexirica/gpm/internal/ui"
)

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case initialLoadMsg:
		return a.onInitialLoad(msg)

	case allNamesMsg:
		return a.onAllNamesLoaded(msg)

	case allPackagesMsg:
		return a.onAllPackagesLoaded(msg)

	case infoLoadedMsg:
		return a.onPackageInfoLoaded(msg)

	case searchResultMsg:
		return a.onSearchResultLoaded(msg)

	case detailLoadedMsg:
		return a.onPackageDetailLoaded(msg)

	case execFinishedMsg:
		return a.onExecFinished(msg)

	case depsLoadedMsg:
		return a.onDepsLoaded(msg)

	case fetchMirrorsMsg:
		return a.onMirrorListLoaded(msg)

	case fetchTestResultMsg:
		return a.onMirrorTestResult(msg)

	case fetchApplyMsg:
		return a.onMirrorApplyResult(msg)

	case tea.KeyMsg:
		if a.fetchView {
			return a.onFetchKeypress(msg)
		}
		if a.transactionView {
			return a.onTransactionKeypress(msg)
		}
		if a.searching {
			return a.onSearchKeypress(msg)
		}
		return a.onKeypress(msg)
	}

	return a, nil
}

// onInitialLoad handles the first load (installed + upgradable only).
// After processing, it kicks off loadAllPackageNamesCmd to populate the full list.
func (a App) onInitialLoad(msg initialLoadMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		return a, nil
	}
	a.upgradableMap = make(map[string]model.Package)
	for _, p := range msg.upgradable {
		a.upgradableMap[p.Name] = p
	}
	var all []model.Package
	for _, p := range msg.installed {
		if up, ok := a.upgradableMap[p.Name]; ok {
			p.Upgradable = true
			p.NewVersion = up.NewVersion
		}
		all = append(all, p)
	}
	a.allPackages = all
	a.installedCount = len(msg.installed)
	a.applyFilter()
	upgCount := len(msg.upgradable)
	a.status = fmt.Sprintf("%d installed (%d upgradable) — loading all packages...",
		a.installedCount, upgCount)
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	cmds = append(cmds, loadAllPackageNamesCmd())
	return a, tea.Batch(cmds...)
}

func (a App) onAllNamesLoaded(msg allNamesMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.allNamesLoaded = true
		a.status = fmt.Sprintf("%d installed (%d upgradable) ",
			a.installedCount, len(a.upgradableMap))
		return a, nil
	}
	a.allNamesLoaded = true
	seen := make(map[string]bool, len(a.allPackages))
	for _, p := range a.allPackages {
		seen[p.Name] = true
	}
	for _, name := range msg.names {
		if !seen[name] {
			pkg := model.Package{Name: name, Installed: false}
			if info, ok := a.infoCache[name]; ok {
				pkg.NewVersion = info.Version
				pkg.Size = info.Size
			}
			a.allPackages = append(a.allPackages, pkg)
			seen[name] = true
		}
	}
	prevIdx := a.selectedIdx
	prevOffset := a.scrollOffset
	a.applyFilter()
	a.selectedIdx = prevIdx
	a.scrollOffset = prevOffset
	if a.selectedIdx >= len(a.filtered) {
		a.selectedIdx = len(a.filtered) - 1
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
	}
	a.status = fmt.Sprintf("%d packages (%d installed, %d upgradable) ",
		len(a.allPackages), a.installedCount, len(a.upgradableMap))
	return a, tea.Batch(a.preloadVisiblePackageInfo())
}

func (a App) onAllPackagesLoaded(msg allPackagesMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		return a, nil
	}
	a.upgradableMap = make(map[string]model.Package)
	for _, p := range msg.upgradable {
		a.upgradableMap[p.Name] = p
	}
	seen := make(map[string]bool, len(msg.installed)+len(msg.allNames))
	var all []model.Package
	for _, p := range msg.installed {
		if up, ok := a.upgradableMap[p.Name]; ok {
			p.Upgradable = true
			p.NewVersion = up.NewVersion
		}
		all = append(all, p)
		seen[p.Name] = true
	}
	for _, name := range msg.allNames {
		if !seen[name] {
			pkg := model.Package{Name: name, Installed: false}
			if info, ok := a.infoCache[name]; ok {
				pkg.NewVersion = info.Version
				pkg.Size = info.Size
			}
			all = append(all, pkg)
			seen[name] = true
		}
	}
	a.allPackages = all
	a.installedCount = len(msg.installed)
	a.allNamesLoaded = true
	a.applyFilter()
	upgCount := len(msg.upgradable)
	a.status = fmt.Sprintf("%d packages (%d installed, %d upgradable) ",
		len(a.allPackages), a.installedCount, upgCount)
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	return a, tea.Batch(cmds...)
}

func (a App) onPackageInfoLoaded(msg infoLoadedMsg) (tea.Model, tea.Cmd) {
	for name, info := range msg.info {
		a.infoCache[name] = info
	}
	for i := range a.allPackages {
		if info, ok := msg.info[a.allPackages[i].Name]; ok {
			if a.allPackages[i].Version == "" {
				a.allPackages[i].NewVersion = info.Version
			}
			if a.allPackages[i].Size == "" {
				a.allPackages[i].Size = info.Size
			}
		}
	}
	for i := range a.filtered {
		if info, ok := msg.info[a.filtered[i].Name]; ok {
			if a.filtered[i].Version == "" {
				a.filtered[i].NewVersion = info.Version
			}
			if a.filtered[i].Size == "" {
				a.filtered[i].Size = info.Size
			}
		}
	}
	return a, nil
}

func (a App) onSearchResultLoaded(msg searchResultMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error in search: %v", msg.err))
		return a, nil
	}
	installedMap := make(map[string]model.Package, len(a.allPackages))
	for _, p := range a.allPackages {
		if p.Installed {
			installedMap[p.Name] = p
		}
	}
	for i := range msg.pkgs {
		if up, ok := a.upgradableMap[msg.pkgs[i].Name]; ok {
			msg.pkgs[i].Upgradable = true
			msg.pkgs[i].NewVersion = up.NewVersion
		}
		if inst, ok := installedMap[msg.pkgs[i].Name]; ok {
			msg.pkgs[i].Version = inst.Version
			msg.pkgs[i].Size = inst.Size
		} else if info, ok := a.infoCache[msg.pkgs[i].Name]; ok {
			msg.pkgs[i].NewVersion = info.Version
			msg.pkgs[i].Size = info.Size
		}
	}
	a.filtered = msg.pkgs
	a.selectedIdx = 0
	a.scrollOffset = 0
	a.status = fmt.Sprintf("%d results for '%s'", len(msg.pkgs), a.filterQuery)
	if len(a.filtered) > 0 {
		return a, tea.Batch(showPackageDetailCmd(a.filtered[0].Name), a.preloadVisiblePackageInfo())
	}
	a.detailInfo = ""
	a.detailName = ""
	return a, nil
}

func (a App) onPackageDetailLoaded(msg detailLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.detailInfo = fmt.Sprintf("Error: %v", msg.err)
	} else {
		a.detailInfo = msg.info
		pi := apt.ParseShowEntry(msg.info)
		if pi.Version != "" || pi.Size != "" {
			a.infoCache[msg.name] = pi
			for i := range a.filtered {
				if a.filtered[i].Name == msg.name {
					if a.filtered[i].Version == "" && a.filtered[i].NewVersion == "" {
						a.filtered[i].NewVersion = pi.Version
					}
					if a.filtered[i].Size == "" {
						a.filtered[i].Size = pi.Size
					}
					break
				}
			}
			for i := range a.allPackages {
				if a.allPackages[i].Name == msg.name {
					if a.allPackages[i].Version == "" && a.allPackages[i].NewVersion == "" {
						a.allPackages[i].NewVersion = pi.Version
					}
					if a.allPackages[i].Size == "" {
						a.allPackages[i].Size = pi.Size
					}
					break
				}
			}
		}
	}
	a.detailName = msg.name
	return a, nil
}

func (a App) onExecFinished(msg execFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.pendingExecFailed = true
	}
	a.pendingExecCount--
	if a.pendingExecCount > 0 {
		return a, nil
	}

	a.loading = false
	success := !a.pendingExecFailed
	op := history.Operation(a.pendingExecOp)
	pkgs := a.pendingExecPkgs
	if len(pkgs) == 0 {
		pkgs = []string{msg.name}
	}
	a.transactionStore.Record(op, pkgs, success)
	a.pendingExecPkgs = nil
	a.pendingExecOp = ""
	a.pendingExecFailed = false

	if !success {
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error (%s %s): %v", msg.op, msg.name, msg.err))
	} else {
		a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ %s %s completed!", msg.op, msg.name))
	}
	return a, reloadAllPackages
}

func (a App) onDepsLoaded(msg depsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.txIdx == a.transactionIdx {
		a.transactionDeps = msg.deps
	}
	return a, nil
}

func (a App) onMirrorListLoaded(msg fetchMirrorsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.fetchView = false
		a.loading = false
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Fetch error: %v", msg.err))
		return a, nil
	}
	a.fetchDistro = msg.distro
	a.fetchMirrors = fetch.LimitMirrors(msg.mirrors, 50)
	a.fetchTesting = true
	a.fetchTested = 0
	a.fetchTotal = len(a.fetchMirrors)
	a.status = fmt.Sprintf("Testing %d mirrors for %s...", a.fetchTotal, msg.distro.Name)
	a.fetchResultCh = fetch.TestMirrorsChan(a.fetchMirrors)
	return a, tea.Batch(a.spinner.Tick, awaitMirrorTestResult(a.fetchResultCh))
}

func (a App) onMirrorTestResult(msg fetchTestResultMsg) (tea.Model, tea.Cmd) {
	if msg.done {
		a.fetchTesting = false
		a.loading = false
		a.fetchMirrors = fetch.ScoreMirrors(a.fetchMirrors)
		a.fetchIdx = 0
		a.fetchOffset = 0
		a.fetchSelected = make(map[int]bool)
		for i := 0; i < 3 && i < len(a.fetchMirrors); i++ {
			a.fetchSelected[i] = true
		}
		a.status = fmt.Sprintf("%d mirrors ready | space: toggle • enter: apply • esc: cancel", len(a.fetchMirrors))
		return a, nil
	}
	r := msg.result
	if r.Index >= 0 && r.Index < len(a.fetchMirrors) {
		if r.Err != nil {
			a.fetchMirrors[r.Index].Status = "error"
		} else {
			a.fetchMirrors[r.Index].Latency = r.Latency
			if r.Latency > 3*1e9 {
				a.fetchMirrors[r.Index].Status = "slow"
			} else {
				a.fetchMirrors[r.Index].Status = "ok"
			}
		}
	}
	a.fetchTested++
	a.status = fmt.Sprintf("Testing mirrors... %d/%d", a.fetchTested, a.fetchTotal)
	return a, awaitMirrorTestResult(a.fetchResultCh)
}

func (a App) onMirrorApplyResult(msg fetchApplyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error writing mirrors: %v", msg.err))
	} else {
		a.status = ui.SuccessStyle.Render("✔ Mirrors saved! Run apt update to apply.")
	}
	a.fetchView = false
	return a, nil
}
