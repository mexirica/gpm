package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/ui"
)

func (a App) exportInstalledPackages() (tea.Model, tea.Cmd) {
	if a.loading {
		return a, nil
	}
	a.status = "Exporting installed packages..."
	return a, exportPackagesCmd(a.allPackages)
}

func (a App) importPackages() (tea.Model, tea.Cmd) {
	a.importingPath = true
	a.importInput.SetValue("")
	a.importInput.Focus()
	a.status = "Enter file path (empty for default) • enter: confirm • esc: cancel"
	return a, textinput.Blink
}

func (a App) onImportInputKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		a.importingPath = false
		a.importInput.Blur()
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		return a, nil
	case tea.KeyEnter:
		path := a.importInput.Value()
		a.importingPath = false
		a.importInput.Blur()
		a.status = "Reading package list..."
		return a, importPackagesCmd(path)
	}

	var cmd tea.Cmd
	a.importInput, cmd = a.importInput.Update(msg)
	return a, cmd
}

func (a App) onExportFinished(msg exportFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.errlogStore.Log("export", msg.err.Error())
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Export failed: %v", msg.err))
		return a, nil
	}
	a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ Exported to %s", msg.path))
	return a, clearStatusAfter(5 * time.Second)
}

func (a App) onImportFinished(msg importFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.errlogStore.Log("import", msg.err.Error())
		a.status = ui.ErrorStyle.Render(fmt.Sprintf("Import failed: %v", msg.err))
		return a, nil
	}
	// Filter out packages that are already installed
	var toInstall []string
	for _, name := range msg.names {
		if idx, ok := a.pkgIndex[name]; ok && a.allPackages[idx].Installed {
			continue
		}
		toInstall = append(toInstall, name)
	}
	if len(toInstall) == 0 {
		a.status = ui.SuccessStyle.Render("✔ All packages from the list are already installed.")
		return a, clearStatusAfter(5 * time.Second)
	}
	a.importConfirm = true
	a.importToInstall = toInstall
	a.importFromPath = msg.path
	return a, nil
}

func (a App) onImportConfirmKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.importDetails {
		const perPage = 15
		totalPages := (len(a.importToInstall) + perPage - 1) / perPage
		switch msg.String() {
		case "d":
			a.importDetails = false
			a.importDetailOffset = 0
			return a, nil
		case "right", "l":
			if a.importDetailOffset < totalPages-1 {
				a.importDetailOffset++
			}
			return a, nil
		case "left", "h":
			if a.importDetailOffset > 0 {
				a.importDetailOffset--
			}
			return a, nil
		}
	}
	switch msg.String() {
	case "y":
		a.importConfirm = false
		a.importDetails = false
		a.pendingExecOp = "install"
		a.pendingExecPkgs = a.importToInstall
		a.pendingExecCount = 1
		a.loading = true
		a.status = fmt.Sprintf("Installing %d packages from %s...", len(a.importToInstall), a.importFromPath)
		return a, installBatchCmd(a.importToInstall)
	case "n", "esc":
		a.importConfirm = false
		a.importDetails = false
		a.importToInstall = nil
		a.importFromPath = ""
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		return a, nil
	case "d":
		a.importDetails = true
		a.importDetailOffset = 0
		return a, nil
	}
	return a, nil
}
