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
	cmds = append(cmds, a.preloadVisiblePackageInfo())
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
	cmds = append(cmds, a.preloadVisiblePackageInfo())
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
	}
	return a, nil, false
}

func (a App) installSelectedPackages() (tea.Model, tea.Cmd) {
	if len(a.selected) > 0 {
		var cmds []tea.Cmd
		var names []string
		for name := range a.selected {
			cmds = append(cmds, installPackageCmd(name))
			names = append(names, name)
		}
		a.pendingExecOp = "install"
		a.pendingExecPkgs = names
		a.pendingExecCount = len(cmds)
		a.loading = true
		a.status = fmt.Sprintf("Installing %d packages...", len(cmds))
		a.selected = make(map[string]bool)
		return a, tea.Batch(cmds...)
	}
	if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if pkg.Installed {
			a.status = fmt.Sprintf("'%s' is already installed.", pkg.Name)
			return a, nil
		}
		a.pendingExecOp = "install"
		a.pendingExecPkgs = []string{pkg.Name}
		a.pendingExecCount = 1
		a.loading = true
		a.status = fmt.Sprintf("Installing %s...", pkg.Name)
		return a, installPackageCmd(pkg.Name)
	}
	return a, nil
}

func (a App) removeSelectedPackages() (tea.Model, tea.Cmd) {
	if len(a.selected) > 0 {
		var cmds []tea.Cmd
		var names []string
		for name := range a.selected {
			cmds = append(cmds, removePackageCmd(name))
			names = append(names, name)
		}
		a.pendingExecOp = "remove"
		a.pendingExecPkgs = names
		a.pendingExecCount = len(cmds)
		a.loading = true
		a.status = fmt.Sprintf("Removing %d packages...", len(cmds))
		a.selected = make(map[string]bool)
		return a, tea.Batch(cmds...)
	}
	if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Installed {
			a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
			return a, nil
		}
		a.pendingExecOp = "remove"
		a.pendingExecPkgs = []string{pkg.Name}
		a.pendingExecCount = 1
		a.loading = true
		a.status = fmt.Sprintf("Removing %s...", pkg.Name)
		return a, removePackageCmd(pkg.Name)
	}
	return a, nil
}

func (a App) purgeSelectedPackages() (tea.Model, tea.Cmd) {
	if len(a.selected) > 0 {
		var cmds []tea.Cmd
		var names []string
		for name := range a.selected {
			cmds = append(cmds, purgePackageCmd(name))
			names = append(names, name)
		}
		a.pendingExecOp = "purge"
		a.pendingExecPkgs = names
		a.pendingExecCount = len(cmds)
		a.loading = true
		a.status = fmt.Sprintf("Purging %d packages...", len(cmds))
		a.selected = make(map[string]bool)
		return a, tea.Batch(cmds...)
	}
	if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Installed {
			a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
			return a, nil
		}
		a.pendingExecOp = "purge"
		a.pendingExecPkgs = []string{pkg.Name}
		a.pendingExecCount = 1
		a.loading = true
		a.status = fmt.Sprintf("Purging %s...", pkg.Name)
		return a, purgePackageCmd(pkg.Name)
	}
	return a, nil
}

func (a App) upgradeSelectedPackages() (tea.Model, tea.Cmd) {
	if len(a.selected) > 0 {
		var cmds []tea.Cmd
		var names []string
		for name := range a.selected {
			cmds = append(cmds, upgradePackageCmd(name))
			names = append(names, name)
		}
		a.pendingExecOp = "upgrade"
		a.pendingExecPkgs = names
		a.pendingExecCount = len(cmds)
		a.loading = true
		a.status = fmt.Sprintf("Upgrading %d packages...", len(cmds))
		a.selected = make(map[string]bool)
		return a, tea.Batch(cmds...)
	}
	if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
		pkg := a.filtered[a.selectedIdx]
		if !pkg.Upgradable {
			a.status = fmt.Sprintf("'%s' is already at the latest version.", pkg.Name)
			return a, nil
		}
		a.pendingExecOp = "upgrade"
		a.pendingExecPkgs = []string{pkg.Name}
		a.pendingExecCount = 1
		a.loading = true
		a.status = fmt.Sprintf("Upgrading %s...", pkg.Name)
		return a, upgradePackageCmd(pkg.Name)
	}
	return a, nil
}

func (a App) upgradeAllPackages() (tea.Model, tea.Cmd) {
	var names []string
	for name := range a.upgradableMap {
		names = append(names, name)
	}
	if len(names) == 0 {
		a.status = "No upgradable packages found."
		return a, nil
	}
	a.pendingExecOp = "upgrade-all"
	a.pendingExecPkgs = names
	a.pendingExecCount = 1
	a.loading = true
	a.status = fmt.Sprintf("Upgrading %d packages (sudo apt-get dist-upgrade)...", len(names))
	return a, upgradeAllPackagesCmd()
}

func (a App) switchTab(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "tab":
		a.activeTab = (a.activeTab + 1) % 3
	case "shift+tab":
		a.activeTab = (a.activeTab + 2) % 3
	default:
		return a, nil, false
	}

	a.applyFilter()
	var cmds []tea.Cmd
	if len(a.filtered) > 0 {
		cmds = append(cmds, showPackageDetailCmd(a.filtered[0].Name))
	}
	cmds = append(cmds, a.preloadVisiblePackageInfo())
	tabNames := []string{"All", "Installed", "Upgradable"}
	a.status = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabNames[a.activeTab])
	return a, tea.Batch(cmds...), true
}
