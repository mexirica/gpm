package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/history"
	"github.com/mexirica/aptui/internal/ui"
)

func (a App) onTransactionKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "t":
		return a.closeTransactionView()
	case "q", "ctrl+c":
		return a, tea.Quit
	case "h":
		a.help.ShowAll = !a.help.ShowAll
		return a, nil
	case "j", "down":
		return a.selectNextTransaction()
	case "k", "up":
		return a.selectPreviousTransaction()
	case "ctrl+d", "pgdown":
		return a.scrollTransactionsDown()
	case "ctrl+u", "pgup":
		return a.scrollTransactionsUp()
	case "z":
		return a.undoTransaction()
	case "x":
		return a.redoTransaction()
	}

	return a, nil
}

func (a App) closeTransactionView() (tea.Model, tea.Cmd) {
	a.transactionView = false
	a.status = fmt.Sprintf("%d packages ", len(a.filtered))
	return a, nil
}

func (a App) selectNextTransaction() (tea.Model, tea.Cmd) {
	if a.transactionIdx < len(a.transactionItems)-1 {
		a.transactionIdx++
		a.adjustTransactionScroll()
		a.transactionDeps = nil
		return a, loadTransactionDepsCmd(a.transactionIdx, a.transactionItems[a.transactionIdx].Packages)
	}
	return a, nil
}

func (a App) selectPreviousTransaction() (tea.Model, tea.Cmd) {
	if a.transactionIdx > 0 {
		a.transactionIdx--
		a.adjustTransactionScroll()
		a.transactionDeps = nil
		return a, loadTransactionDepsCmd(a.transactionIdx, a.transactionItems[a.transactionIdx].Packages)
	}
	return a, nil
}

func (a App) scrollTransactionsDown() (tea.Model, tea.Cmd) {
	a.transactionIdx += a.transactionListHeight()
	if a.transactionIdx >= len(a.transactionItems) {
		a.transactionIdx = len(a.transactionItems) - 1
	}
	if a.transactionIdx < 0 {
		a.transactionIdx = 0
	}
	a.adjustTransactionScroll()
	a.transactionDeps = nil
	var cmd tea.Cmd
	if len(a.transactionItems) > 0 {
		cmd = loadTransactionDepsCmd(a.transactionIdx, a.transactionItems[a.transactionIdx].Packages)
	}
	return a, cmd
}

func (a App) scrollTransactionsUp() (tea.Model, tea.Cmd) {
	a.transactionIdx -= a.transactionListHeight()
	if a.transactionIdx < 0 {
		a.transactionIdx = 0
	}
	a.adjustTransactionScroll()
	a.transactionDeps = nil
	var cmd tea.Cmd
	if len(a.transactionItems) > 0 {
		cmd = loadTransactionDepsCmd(a.transactionIdx, a.transactionItems[a.transactionIdx].Packages)
	}
	return a, cmd
}

func (a App) undoTransaction() (tea.Model, tea.Cmd) {
	if len(a.transactionItems) == 0 || a.transactionIdx >= len(a.transactionItems) {
		return a, nil
	}
	tx := a.transactionItems[a.transactionIdx]
	if !tx.Success {
		a.status = ui.ErrorStyle.Render("Cannot undo a failed transaction.")
		return a, nil
	}
	if tx.Operation == history.OpUpgradeAll || tx.Operation == history.OpUpgrade {
		a.status = ui.ErrorStyle.Render("Cannot undo upgrade: downgrade is not supported.")
		return a, nil
	}
	undoOp := history.UndoOperation(tx.Operation)
	pkgs := tx.Packages
	if undoOp == history.OpRemove {
		var blocked []string
		var allowed []string
		for _, name := range pkgs {
			if a.essentialSet[name] {
				blocked = append(blocked, name)
			} else {
				allowed = append(allowed, name)
			}
		}
		if len(blocked) > 0 && len(allowed) == 0 {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Cannot undo: essential package(s): %s", strings.Join(blocked, ", ")))
			return a, nil
		}
		if len(blocked) > 0 {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Skipping essential: %s", strings.Join(blocked, ", ")))
		}
		pkgs = allowed
	}
	var cmd tea.Cmd
	switch undoOp {
	case history.OpRemove:
		cmd = removeBatchCmd(pkgs)
	case history.OpInstall:
		cmd = installBatchCmd(pkgs)
	}
	a.pendingExecOp = string(undoOp)
	a.pendingExecPkgs = pkgs
	a.pendingExecCount = 1
	a.transactionView = false
	a.loading = true
	a.status = fmt.Sprintf("Undoing #%d (%s %d packages)...", tx.ID, undoOp, len(tx.Packages))
	return a, cmd
}

func (a App) redoTransaction() (tea.Model, tea.Cmd) {
	if len(a.transactionItems) == 0 || a.transactionIdx >= len(a.transactionItems) {
		return a, nil
	}
	tx := a.transactionItems[a.transactionIdx]
	pkgs := tx.Packages
	if tx.Operation == history.OpRemove || tx.Operation == history.OpPurge {
		var blocked []string
		var allowed []string
		for _, name := range pkgs {
			if a.essentialSet[name] {
				blocked = append(blocked, name)
			} else {
				allowed = append(allowed, name)
			}
		}
		if len(blocked) > 0 && len(allowed) == 0 {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Cannot redo: essential package(s): %s", strings.Join(blocked, ", ")))
			return a, nil
		}
		if len(blocked) > 0 {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Skipping essential: %s", strings.Join(blocked, ", ")))
		}
		pkgs = allowed
	}
	var cmd tea.Cmd
	switch tx.Operation {
	case history.OpUpgradeAll:
		cmd = upgradeAllPackagesCmd(pkgs)
	case history.OpInstall:
		cmd = installBatchCmd(pkgs)
	case history.OpRemove:
		cmd = removeBatchCmd(pkgs)
	case history.OpUpgrade:
		cmd = upgradeBatchCmd(pkgs)
	case history.OpPurge:
    	cmd = purgeBatchCmd(pkgs)
	}
	a.pendingExecOp = string(tx.Operation)
	a.pendingExecPkgs = pkgs
	a.pendingExecCount = 1
	a.transactionView = false
	a.loading = true
	a.status = fmt.Sprintf("Redoing #%d (%s %d packages)...", tx.ID, tx.Operation, len(tx.Packages))
	return a, cmd
}
