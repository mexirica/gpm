package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/history"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
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

	case allPackagesMsg:
		return a.onAllPackagesLoaded(msg)

	case bulkInfoMsg:
		return a.onBulkInfoLoaded(msg)

	case silentUpdateDoneMsg:
		return a.onSilentUpdateDone(msg)

	case infoLoadedMsg:
		return a.onPackageInfoLoaded(msg)

	case searchResultMsg:
		return a.onSearchResultLoaded(msg)

	case detailLoadedMsg:
		return a.onPackageDetailLoaded(msg)

	case execFinishedMsg:
		return a.onExecFinished(msg)

	case clearStatusMsg:
		if a.pendingStatus != "" && !a.loading {
			a.status = a.pendingStatus
			a.pendingStatus = ""
		}
		return a, nil

	case depsLoadedMsg:
		return a.onDepsLoaded(msg)

	case autoremovableMsg:
		return a.onAutoremovableLoaded(msg)

	case ppaListMsg:
		return a.onPPAListLoaded(msg)

	case ppaToggleMsg:
		return a.onPPAToggled(msg)

	case fetchMirrorsMsg:
		return a.onMirrorListLoaded(msg)

	case fetchTestResultMsg:
		return a.onMirrorTestResult(msg)

	case fetchApplyMsg:
		return a.onMirrorApplyResult(msg)

	case tea.MouseMsg:
		if !a.fetchView && !a.transactionView && !a.ppaView && !a.loading {
			return a.onMouseClick(msg)
		}

	case tea.KeyMsg:
		if a.fetchView {
			return a.onFetchKeypress(msg)
		}
		if a.ppaView {
			return a.onPPAKeypress(msg)
		}
		if a.transactionView {
			return a.onTransactionKeypress(msg)
		}
		if a.filtering {
			return a.onFilterKeypress(msg)
		}
		if a.searching {
			return a.onSearchKeypress(msg)
		}
		return a.onKeypress(msg)
	}

	return a, nil
}

func (a App) onAllPackagesLoaded(msg allPackagesMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.errlogStore.Log("load-packages", msg.err.Error())
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
				pkg.Section = info.Section
				pkg.Architecture = info.Architecture
			}
			all = append(all, pkg)
			seen[name] = true
		}
	}
	a.allPackages = all
	a.rebuildIndex()
	a.installedCount = len(msg.installed)
	firstLoad := !a.allNamesLoaded
	a.allNamesLoaded = true
	a.applyFilter()
	upgCount := len(msg.upgradable)
	defaultStatus := fmt.Sprintf("%d packages (%d installed, %d upgradable) ",
		len(a.allPackages), a.installedCount, upgCount)
	if time.Since(a.statusLock) >= 2*time.Second {
		a.status = defaultStatus
	} else {
		a.pendingStatus = defaultStatus
	}
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	if firstLoad {
		cmds = append(cmds, silentUpdateCmd())
	}
	return a, tea.Batch(cmds...)
}

func (a App) onBulkInfoLoaded(msg bulkInfoMsg) (tea.Model, tea.Cmd) {
	if len(msg.info) == 0 {
		return a, nil
	}
	// Merge into infoCache (don't overwrite existing entries)
	for name, info := range msg.info {
		if _, ok := a.infoCache[name]; !ok {
			a.infoCache[name] = info
		}
	}
	// Merge into allPackages for packages missing metadata
	for name, info := range msg.info {
		if idx, ok := a.pkgIndex[name]; ok {
			p := &a.allPackages[idx]
			if p.Version == "" && p.NewVersion == "" {
				p.NewVersion = info.Version
			}
			if p.Size == "" {
				p.Size = info.Size
			}
			if p.Section == "" {
				p.Section = info.Section
			}
			if p.Architecture == "" {
				p.Architecture = info.Architecture
			}
		}
	}
	return a, nil
}

func (a App) onSilentUpdateDone(msg silentUpdateDoneMsg) (tea.Model, tea.Cmd) {
	changed := false

	// Merge new package names
	if len(msg.names) > 0 {
		for _, name := range msg.names {
			if _, ok := a.pkgIndex[name]; !ok {
				a.pkgIndex[name] = len(a.allPackages)
				a.allPackages = append(a.allPackages, model.Package{Name: name})
				changed = true
			}
		}
	}

	// Merge upgradable
	newMap := make(map[string]model.Package, len(msg.upgradable))
	for _, p := range msg.upgradable {
		newMap[p.Name] = p
	}
	if len(newMap) != len(a.upgradableMap) {
		changed = true
	} else {
		for name := range newMap {
			if _, ok := a.upgradableMap[name]; !ok {
				changed = true
				break
			}
		}
	}

	if !changed {
		return a, nil
	}

	// Clear old upgradable flags using index for O(1) access
	for name := range a.upgradableMap {
		if idx, ok := a.pkgIndex[name]; ok {
			a.allPackages[idx].Upgradable = false
			a.allPackages[idx].NewVersion = ""
		}
	}
	a.upgradableMap = newMap
	// Set new upgradable flags
	for name, up := range newMap {
		if idx, ok := a.pkgIndex[name]; ok {
			a.allPackages[idx].Upgradable = true
			a.allPackages[idx].NewVersion = up.NewVersion
		}
	}
	a.applyFilter()
	upgCount := len(msg.upgradable)
	defaultStatus := fmt.Sprintf("%d packages (%d installed, %d upgradable) ",
		len(a.allPackages), a.installedCount, upgCount)
	if time.Since(a.statusLock) >= 2*time.Second {
		a.status = defaultStatus
	} else {
		a.pendingStatus = defaultStatus
	}
	return a, a.preloadVisiblePackageInfo()
}

func (a App) onPackageInfoLoaded(msg infoLoadedMsg) (tea.Model, tea.Cmd) {
	for name, info := range msg.info {
		a.infoCache[name] = info
	}
	for name, info := range msg.info {
		if idx, ok := a.pkgIndex[name]; ok {
			if a.allPackages[idx].Version == "" {
				a.allPackages[idx].NewVersion = info.Version
			}
			if a.allPackages[idx].Size == "" {
				a.allPackages[idx].Size = info.Size
			}
			if a.allPackages[idx].Section == "" {
				a.allPackages[idx].Section = info.Section
			}
			if a.allPackages[idx].Architecture == "" {
				a.allPackages[idx].Architecture = info.Architecture
			}
		}
	}

	// If an advanced filter is active, re-apply it now that metadata has arrived.
	if a.advancedFilter != "" {
		wasLoadingMeta := a.loadingFilterMeta
		a.loadingFilterMeta = false
		if wasLoadingMeta {
			a.loading = false
		}
		prevIdx := a.selectedIdx
		prevOffset := a.scrollOffset
		a.applyFilter()
		if !wasLoadingMeta {
			a.selectedIdx = prevIdx
			a.scrollOffset = prevOffset
		}
		if a.selectedIdx >= len(a.filtered) {
			a.selectedIdx = len(a.filtered) - 1
			if a.selectedIdx < 0 {
				a.selectedIdx = 0
			}
		}
		a.status = fmt.Sprintf("%d packages matching filter", len(a.filtered))
		var cmds []tea.Cmd
		if wasLoadingMeta && len(a.filtered) > 0 {
			cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
			cmds = append(cmds, a.preloadVisiblePackageInfo())
		}
		return a, tea.Batch(cmds...)
	}

	for i := range a.filtered {
		if info, ok := msg.info[a.filtered[i].Name]; ok {
			if a.filtered[i].Version == "" {
				a.filtered[i].NewVersion = info.Version
			}
			if a.filtered[i].Size == "" {
				a.filtered[i].Size = info.Size
			}
			if a.filtered[i].Section == "" {
				a.filtered[i].Section = info.Section
			}
			if a.filtered[i].Architecture == "" {
				a.filtered[i].Architecture = info.Architecture
			}
		}
	}
	return a, nil
}

func (a App) onSearchResultLoaded(msg searchResultMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.errlogStore.Log("search", msg.err.Error())
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error in search: %v", msg.err))
		return a, nil
	}
	for i := range msg.pkgs {
		if up, ok := a.upgradableMap[msg.pkgs[i].Name]; ok {
			msg.pkgs[i].Upgradable = true
			msg.pkgs[i].NewVersion = up.NewVersion
		}
		if idx, ok := a.pkgIndex[msg.pkgs[i].Name]; ok && a.allPackages[idx].Installed {
			inst := a.allPackages[idx]
			msg.pkgs[i].Installed = true
			msg.pkgs[i].Version = inst.Version
			msg.pkgs[i].Size = inst.Size
			msg.pkgs[i].Section = inst.Section
			msg.pkgs[i].Architecture = inst.Architecture
		} else if info, ok := a.infoCache[msg.pkgs[i].Name]; ok {
			msg.pkgs[i].NewVersion = info.Version
			msg.pkgs[i].Size = info.Size
			msg.pkgs[i].Section = info.Section
			msg.pkgs[i].Architecture = info.Architecture
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
		a.errlogStore.Log("package-detail", fmt.Sprintf("%s: %v", msg.name, msg.err))
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
					if a.filtered[i].Section == "" {
						a.filtered[i].Section = pi.Section
					}
					if a.filtered[i].Architecture == "" {
						a.filtered[i].Architecture = pi.Architecture
					}
					break
				}
			}
			if idx, ok := a.pkgIndex[msg.name]; ok {
				if a.allPackages[idx].Version == "" && a.allPackages[idx].NewVersion == "" {
					a.allPackages[idx].NewVersion = pi.Version
				}
				if a.allPackages[idx].Size == "" {
					a.allPackages[idx].Size = pi.Size
				}
				if a.allPackages[idx].Section == "" {
					a.allPackages[idx].Section = pi.Section
				}
				if a.allPackages[idx].Architecture == "" {
					a.allPackages[idx].Architecture = pi.Architecture
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
	if op != "update" && op != "cleanup-all" && op != "ppa-add" && op != "ppa-remove" {
		a.transactionStore.Record(op, pkgs, success)
	}
	a.pendingExecPkgs = nil
	a.pendingExecOp = ""
	a.pendingExecFailed = false

	if !success {
		a.errlogStore.Log("exec", fmt.Sprintf("%s %s: %s", msg.op, msg.name, friendlyError(msg.err)))
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error (%s %s): %s", msg.op, msg.name, friendlyError(msg.err)))
	} else if msg.op == "update" {
		a.status = ui.SuccessStyle.Render("✔ apt update completed!")
	} else if msg.op == "cleanup-all" {
		a.status = ui.SuccessStyle.Render("✔ Cleanup completed!")
	} else if msg.op == "ppa-add" {
		a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ PPA %s added!", msg.name))
	} else if msg.op == "ppa-remove" {
		a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ PPA %s removed!", msg.name))
	} else {
		a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ %s %s completed!", msg.op, msg.name))
	}
	a.statusLock = time.Now()

	if success && msg.op != "update" && msg.op != "ppa-add" && msg.op != "ppa-remove" {
		a.applyOptimisticUpdate(msg.op, pkgs)
	}

	cmds := []tea.Cmd{reloadAllPackages, loadAutoremovableCmd(), clearStatusAfter(2 * time.Second)}
	if msg.op == "ppa-add" || msg.op == "ppa-remove" {
		cmds = append(cmds, listPPAsCmd())
	}
	return a, tea.Batch(cmds...)
}

func (a App) onDepsLoaded(msg depsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.txIdx == a.transactionIdx {
		a.transactionDeps = msg.deps
	}
	return a, nil
}

func (a App) onMirrorListLoaded(msg fetchMirrorsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.errlogStore.Log("fetch-mirrors", msg.err.Error())
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
		a.errlogStore.Log("apply-mirrors", msg.err.Error())
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error writing mirrors: %v", msg.err))
	} else {
		a.status = ui.SuccessStyle.Render("✔ Mirrors saved! Run apt update to apply.")
	}
	a.fetchView = false
	return a, nil
}

func (a App) onAutoremovableLoaded(msg autoremovableMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.errlogStore.Log("autoremovable", msg.err.Error())
		a.autoremovable = nil
		a.autoremovableSet = make(map[string]bool)
		if a.activeTab == tabCleanup {
			a.applyFilter()
		}
		return a, nil
	}
	a.autoremovable = msg.names
	a.autoremovableSet = make(map[string]bool, len(msg.names))
	for _, name := range msg.names {
		a.autoremovableSet[name] = true
	}
	if a.activeTab == tabCleanup {
		a.applyFilter()
		if time.Since(a.statusLock) >= 2*time.Second {
			a.status = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabDefs[a.activeTab].name)
		} else {
			a.pendingStatus = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabDefs[a.activeTab].name)
		}
	}
	return a, nil
}

func (a App) onPPAListLoaded(msg ppaListMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.errlogStore.Log("ppa-list", msg.err.Error())
		a.ppaItems = nil
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error listing PPAs: %v", msg.err))
		return a, nil
	}
	a.ppaItems = msg.ppas
	if a.ppaIdx >= len(a.ppaItems) {
		a.ppaIdx = len(a.ppaItems) - 1
		if a.ppaIdx < 0 {
			a.ppaIdx = 0
		}
	}
	a.status = fmt.Sprintf("%d PPA(s) found", len(a.ppaItems))
	return a, nil
}

func (a App) onPPAToggled(msg ppaToggleMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.err != nil {
		a.errlogStore.Log("ppa-toggle", msg.err.Error())
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error toggling PPA %s: %v", msg.name, msg.err))
		return a, nil
	}
	a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ PPA %s %s!", msg.name, msg.action))
	return a, tea.Batch(listPPAsCmd(), silentUpdateCmd(), reloadAllPackages, clearStatusAfter(2*time.Second))
}
