package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (a App) dispatchNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "j", "down":
		model, cmd := a.selectNextPackage()
		return model, cmd, true
	case "k", "up":
		model, cmd := a.selectPreviousPackage()
		return model, cmd, true
	case "ctrl+d", "pgdown":
		model, cmd := a.scrollPackagesDown()
		return model, cmd, true
	case "ctrl+u", "pgup":
		model, cmd := a.scrollPackagesUp()
		return model, cmd, true
	}
	return a, nil, false
}

func (a App) selectNextPackage() (tea.Model, tea.Cmd) {
	if a.selectedIdx < len(a.filtered)-1 {
		a.selectedIdx++
		a.adjustPackageScroll()
		return a, showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
	}
	return a, nil
}

func (a App) selectPreviousPackage() (tea.Model, tea.Cmd) {
	if a.selectedIdx > 0 {
		a.selectedIdx--
		a.adjustPackageScroll()
		return a, showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
	}
	return a, nil
}

func (a App) scrollPackagesDown() (tea.Model, tea.Cmd) {
	a.selectedIdx += a.packageListHeight()
	if a.selectedIdx >= len(a.filtered) {
		a.selectedIdx = len(a.filtered) - 1
	}
	if a.selectedIdx < 0 {
		a.selectedIdx = 0
	}
	a.adjustPackageScroll()
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[a.selectedIdx].Name))
	}
	return a, tea.Batch(cmds...)
}

func (a App) scrollPackagesUp() (tea.Model, tea.Cmd) {
	a.selectedIdx -= a.packageListHeight()
	if a.selectedIdx < 0 {
		a.selectedIdx = 0
	}
	a.adjustPackageScroll()
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[a.selectedIdx].Name))
	}
	return a, tea.Batch(cmds...)
}

func (a App) dispatchSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case " ":
		model, cmd := a.togglePackageSelection()
		return model, cmd, true
	case "a":
		model, cmd := a.toggleSelectAll()
		return model, cmd, true
	}
	return a, nil, false
}

func (a App) togglePackageSelection() (tea.Model, tea.Cmd) {
	if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if a.selected == nil {
			a.selected = make(map[string]bool)
		}
		if a.selected[pkg.Name] {
			delete(a.selected, pkg.Name)
		} else {
			a.selected[pkg.Name] = true
		}
		a.status = fmt.Sprintf("%d selected ", len(a.selected))
	}
	return a, nil
}

func (a App) toggleSelectAll() (tea.Model, tea.Cmd) {
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}
	allSelected := true
	for _, p := range a.filtered {
		if !a.selected[p.Name] {
			allSelected = false
			break
		}
	}
	if allSelected {
		a.selected = make(map[string]bool)
		a.status = "0 selected "
		return a, nil
	}
	for _, p := range a.filtered {
		a.selected[p.Name] = true
	}
	a.status = fmt.Sprintf("%d selected ", len(a.selected))
	return a, nil
}

func (a App) dispatchPackageAction(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "i":
		model, cmd := a.installSelectedPackages()
		return model, cmd, true
	case "r":
		model, cmd := a.removeSelectedPackages()
		return model, cmd, true
	case "u":
		model, cmd := a.upgradeSelectedPackages()
		return model, cmd, true
	case "G":
		model, cmd := a.upgradeAllPackages()
		return model, cmd, true
	case "p":
		model, cmd := a.purgeSelectedPackages()
		return model, cmd, true
	case "H":
		model, cmd := a.holdSelectedPackages()
		return model, cmd, true
	case "F":
		model, cmd := a.togglePinPackages()
		return model, cmd, true
	case "c":
		model, cmd := a.cleanupAllPackages()
		return model, cmd, true
	}
	return a, nil, false
}

func (a App) installSelectedPackages() (tea.Model, tea.Cmd) {
	var names []string
	if len(a.selected) > 0 {
		for name := range a.selected {
			names = append(names, name)
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if pkg.Installed {
			a.status = fmt.Sprintf("'%s' is already installed.", pkg.Name)
			return a, nil
		}
		names = append(names, pkg.Name)
	}
	if len(names) == 0 {
		return a, nil
	}
	a.pendingExecOp = "install"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Installing %d packages...", len(names))
	a.selected = make(map[string]bool)
	return a, installBatchCmd(names)
}

func (a App) removeSelectedPackages() (tea.Model, tea.Cmd) {
	var names []string
	if len(a.selected) > 0 {
		for name := range a.selected {
			names = append(names, name)
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Installed {
			a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
			return a, nil
		}
		names = append(names, pkg.Name)
	}
	if len(names) == 0 {
		return a, nil
	}
	a.pendingExecOp = "remove"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Removing %d packages...", len(names))
	a.selected = make(map[string]bool)
	return a, removeBatchCmd(names)
}

func (a App) purgeSelectedPackages() (tea.Model, tea.Cmd) {
	var names []string
	if len(a.selected) > 0 {
		for name := range a.selected {
			names = append(names, name)
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Installed {
			a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
			return a, nil
		}
		names = append(names, pkg.Name)
	}
	if len(names) == 0 {
		return a, nil
	}
	a.pendingExecOp = "purge"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Purging %d packages...", len(names))
	a.selected = make(map[string]bool)
	return a, purgeBatchCmd(names)
}

func (a App) upgradeSelectedPackages() (tea.Model, tea.Cmd) {
	var names []string
	if len(a.selected) > 0 {
		for name := range a.selected {
			if a.heldSet[name] {
				continue
			}
			names = append(names, name)
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if pkg.Held {
			a.status = fmt.Sprintf("'%s' is held. Unhold it first (H).", pkg.Name)
			return a, nil
		}
		if !pkg.Upgradable {
			a.status = fmt.Sprintf("'%s' is already at the latest version.", pkg.Name)
			return a, nil
		}
		names = append(names, pkg.Name)
	}
	if len(names) == 0 {
		return a, nil
	}
	a.pendingExecOp = "upgrade"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Upgrading %d packages...", len(names))
	a.selected = make(map[string]bool)
	return a, upgradeBatchCmd(names)
}

func (a App) upgradeAllPackages() (tea.Model, tea.Cmd) {
	var names []string
	for name := range a.upgradableMap {
		if !a.heldSet[name] {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		if len(a.upgradableMap) > 0 {
			a.status = "All upgradable packages are held. Unhold them first (H)."
		} else {
			a.status = "No upgradable packages found."
		}
		return a, nil
	}
	a.pendingExecOp = "upgrade-all"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Upgrading %d packages (sudo apt-get install --only-upgrade)...", len(names))
	return a, upgradeAllPackagesCmd(names)
}

func (a App) cleanupAllPackages() (tea.Model, tea.Cmd) {
	if len(a.autoremovable) == 0 {
		a.status = "No packages to clean up."
		return a, nil
	}
	a.pendingExecOp = "cleanup-all"
	a.pendingExecPkgs = a.autoremovable
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Cleaning up all %d packages (sudo apt-get autoremove)...", len(a.autoremovable))
	return a, autoremoveAllCmd(a.autoremovable)
}

func (a App) holdSelectedPackages() (tea.Model, tea.Cmd) {
	var holdNames, unholdNames []string
	if len(a.selected) > 0 {
		for name := range a.selected {
			if a.heldSet[name] {
				unholdNames = append(unholdNames, name)
			} else {
				holdNames = append(holdNames, name)
			}
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Installed {
			a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
			return a, nil
		}
		if a.heldSet[pkg.Name] {
			unholdNames = append(unholdNames, pkg.Name)
		} else {
			holdNames = append(holdNames, pkg.Name)
		}
	}
	if len(holdNames) == 0 && len(unholdNames) == 0 {
		return a, nil
	}
	a.loading = true
	a.selected = make(map[string]bool)
	if len(holdNames) > 0 && len(unholdNames) > 0 {
		a.holdPending = 2
		a.status = fmt.Sprintf("Toggling hold on %d packages...", len(holdNames)+len(unholdNames))
		return a, tea.Batch(holdBatchCmd(holdNames), unholdBatchCmd(unholdNames))
	}
	a.holdPending = 1
	if len(holdNames) > 0 {
		a.status = fmt.Sprintf("Holding %d packages...", len(holdNames))
		return a, holdBatchCmd(holdNames)
	}
	a.status = fmt.Sprintf("Unholding %d packages...", len(unholdNames))
	return a, unholdBatchCmd(unholdNames)
}

func (a App) switchTab(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "tab":
		a.activeTab = (a.activeTab + 1) % tabKind(len(tabDefs))
	case "shift+tab":
		a.activeTab = (a.activeTab + tabKind(len(tabDefs)) - 1) % tabKind(len(tabDefs))
	default:
		return a, nil, false
	}

	cmd := a.activateTab()
	return a, cmd, true
}

func (a App) togglePinPackages() (tea.Model, tea.Cmd) {
	var names []string
	var currentName string
	if len(a.selected) > 0 {
		for name := range a.selected {
			names = append(names, name)
		}
	} else if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		currentName = a.filtered[a.selectedIdx].Name
		names = append(names, currentName)
	}
	if len(names) == 0 {
		return a, nil
	}

	var pinned, unpinned int
	for _, name := range names {
		if a.pinStore.Toggle(name) {
			a.pinnedSet[name] = true
			pinned++
		} else {
			delete(a.pinnedSet, name)
			unpinned++
		}
	}

	// Update Pinned flag on allPackages
	for _, name := range names {
		if idx, ok := a.pkgIndex[name]; ok {
			a.allPackages[idx].Pinned = a.pinnedSet[name]
		}
	}

	a.applyFilter()
	a.selected = make(map[string]bool)

	// Restore cursor to the same package after reorder
	if currentName != "" {
		for i, p := range a.filtered {
			if p.Name == currentName {
				a.selectedIdx = i
				a.adjustPackageScroll()
				break
			}
		}
	}

	if pinned > 0 && unpinned > 0 {
		a.status = fmt.Sprintf("Pinned %d, unpinned %d packages", pinned, unpinned)
	} else if pinned > 0 {
		a.status = fmt.Sprintf("Pinned %d package(s)", pinned)
	} else {
		a.status = fmt.Sprintf("Unpinned %d package(s)", unpinned)
	}

	var cmd tea.Cmd
	if len(a.filtered) > 0 {
		cmd = showPackageDetailCmd(a.filtered[a.selectedIdx].Name)
	}
	return a, cmd
}
