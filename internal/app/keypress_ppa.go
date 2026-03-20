package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/ui"
)

func (a App) onPPAKeypress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.ppaAdding {
		return a.onPPAInputKeypress(msg)
	}

	switch msg.String() {
	case "esc":
		return a.closePPAView()
	case "q", "ctrl+c":
		return a, tea.Quit
	case "h":
		a.help.ShowAll = !a.help.ShowAll
		return a, nil
	case "j", "down":
		return a.selectNextPPA()
	case "k", "up":
		return a.selectPreviousPPA()
	case "ctrl+d", "pgdown":
		return a.scrollPPAsDown()
	case "ctrl+u", "pgup":
		return a.scrollPPAsUp()
	case "a":
		return a.startAddPPA()
	case "r":
		return a.removeSelectedPPA()
	case "e":
		return a.toggleSelectedPPA()
	}

	return a, nil
}

func (a App) onPPAInputKeypress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Key().Code {
	case tea.KeyEsc:
		a.ppaAdding = false
		a.ppaInput.Blur()
		a.status = fmt.Sprintf("%d PPAs | a: add • r: remove • e: enable/disable • esc: back", len(a.ppaItems))
		return a, nil
	case tea.KeyEnter:
		return a.submitAddPPA()
	}

	var cmd tea.Cmd
	a.ppaInput, cmd = a.ppaInput.Update(msg)
	return a, cmd
}

func (a App) closePPAView() (tea.Model, tea.Cmd) {
	a.ppaView = false
	a.ppaAdding = false
	a.ppaInput.Blur()
	a.status = fmt.Sprintf("%d packages ", len(a.filtered))
	return a, nil
}

func (a App) selectNextPPA() (tea.Model, tea.Cmd) {
	if a.ppaIdx < len(a.ppaItems)-1 {
		a.ppaIdx++
		a.adjustPPAScroll()
	}
	return a, nil
}

func (a App) selectPreviousPPA() (tea.Model, tea.Cmd) {
	if a.ppaIdx > 0 {
		a.ppaIdx--
		a.adjustPPAScroll()
	}
	return a, nil
}

func (a App) scrollPPAsDown() (tea.Model, tea.Cmd) {
	a.ppaIdx += a.packageListHeight()
	if a.ppaIdx >= len(a.ppaItems) {
		a.ppaIdx = len(a.ppaItems) - 1
	}
	if a.ppaIdx < 0 {
		a.ppaIdx = 0
	}
	a.adjustPPAScroll()
	return a, nil
}

func (a App) scrollPPAsUp() (tea.Model, tea.Cmd) {
	a.ppaIdx -= a.packageListHeight()
	if a.ppaIdx < 0 {
		a.ppaIdx = 0
	}
	a.adjustPPAScroll()
	return a, nil
}

func (a App) startAddPPA() (tea.Model, tea.Cmd) {
	a.ppaAdding = true
	a.ppaInput.SetValue("")
	cmd := a.ppaInput.Focus()
	a.status = "Enter PPA (e.g. ppa:mozillateam/ppa) | enter: confirm • esc: cancel"
	return a, cmd
}

func (a App) submitAddPPA() (tea.Model, tea.Cmd) {
	val := a.ppaInput.Value()
	if err := apt.ValidatePPA(val); err != nil {
		a.status = ui.ErrorStyle.Render(err.Error())
		return a, nil
	}
	a.ppaAdding = false
	a.ppaInput.Blur()
	a.loading = true
	a.pendingExecOp = "ppa-add"
	a.pendingExecPkgs = []string{val}
	a.pendingExecCount = 1
	a.status = fmt.Sprintf("Adding %s...", val)
	return a, addPPACmd(val)
}

func (a App) removeSelectedPPA() (tea.Model, tea.Cmd) {
	if len(a.ppaItems) == 0 || a.ppaIdx >= len(a.ppaItems) {
		return a, nil
	}
	ppa := a.ppaItems[a.ppaIdx]
	a.loading = true
	a.pendingExecOp = "ppa-remove"
	a.pendingExecPkgs = []string{ppa.Name}
	a.pendingExecCount = 1
	a.status = fmt.Sprintf("Removing %s...", ppa.Name)
	return a, removePPACmd(ppa.Name)
}

func (a App) toggleSelectedPPA() (tea.Model, tea.Cmd) {
	if len(a.ppaItems) == 0 || a.ppaIdx >= len(a.ppaItems) {
		return a, nil
	}
	ppa := a.ppaItems[a.ppaIdx]
	action := "Enabling"
	if ppa.Enabled {
		action = "Disabling"
	}
	a.loading = true
	a.status = fmt.Sprintf("%s %s...", action, ppa.Name)
	return a, togglePPACmd(ppa)
}
