package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mexirica/aptui/internal/ui"
)

func (a App) dispatchErrorLog(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if a.activeTab != tabErrorLog {
		return a, nil, false
	}
	switch msg.String() {
	case "j", "down":
		m, cmd := a.selectNextError()
		return m, cmd, true
	case "k", "up":
		m, cmd := a.selectPreviousError()
		return m, cmd, true
	case "ctrl+d", "pgdown":
		m, cmd := a.scrollErrorsDown()
		return m, cmd, true
	case "ctrl+u", "pgup":
		m, cmd := a.scrollErrorsUp()
		return m, cmd, true
	}
	return a, nil, false
}

func (a App) selectNextError() (tea.Model, tea.Cmd) {
	if a.errlogIdx < len(a.errlogItems)-1 {
		a.errlogIdx++
		a.adjustErrorLogScroll()
	}
	return a, nil
}

func (a App) selectPreviousError() (tea.Model, tea.Cmd) {
	if a.errlogIdx > 0 {
		a.errlogIdx--
		a.adjustErrorLogScroll()
	}
	return a, nil
}

func (a App) scrollErrorsDown() (tea.Model, tea.Cmd) {
	a.errlogIdx += a.errorLogListHeight()
	if a.errlogIdx >= len(a.errlogItems) {
		a.errlogIdx = len(a.errlogItems) - 1
	}
	if a.errlogIdx < 0 {
		a.errlogIdx = 0
	}
	a.adjustErrorLogScroll()
	return a, nil
}

func (a App) scrollErrorsUp() (tea.Model, tea.Cmd) {
	a.errlogIdx -= a.errorLogListHeight()
	if a.errlogIdx < 0 {
		a.errlogIdx = 0
	}
	a.adjustErrorLogScroll()
	return a, nil
}

func (a App) clearErrorLog() (tea.Model, tea.Cmd) {
	a.errlogStore.Clear()
	a.errlogItems = nil
	a.errlogIdx = 0
	a.errlogOffset = 0
	a.status = ui.SuccessStyle.Render("Error log cleared")
	return a, nil
}
