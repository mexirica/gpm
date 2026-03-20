package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/ui"
)

func (a App) onFetchKeypress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.fetchTesting {
		return a.cancelFetchTest(msg)
	}

	switch msg.String() {
	case "esc":
		return a.closeMirrorView()
	case "q", "ctrl+c":
		return a, tea.Quit
	case "j", "down":
		return a.selectNextMirror()
	case "k", "up":
		return a.selectPreviousMirror()
	case "ctrl+d", "pgdown":
		return a.scrollMirrorsDown()
	case "ctrl+u", "pgup":
		return a.scrollMirrorsUp()
	case "space":
		return a.toggleMirrorSelection()
	case "enter":
		return a.applySelectedMirrors()
	}
	return a, nil
}

func (a App) cancelFetchTest(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "q" || msg.String() == "ctrl+c" {
		a.fetchView = false
		a.fetchTesting = false
		a.loading = false
		a.status = "Fetch cancelled."
		return a, nil
	}
	return a, nil
}

func (a App) closeMirrorView() (tea.Model, tea.Cmd) {
	a.fetchView = false
	a.status = fmt.Sprintf("%d packages ", len(a.filtered))
	return a, nil
}

func (a App) selectNextMirror() (tea.Model, tea.Cmd) {
	if a.fetchIdx < len(a.fetchMirrors)-1 {
		a.fetchIdx++
		a.adjustMirrorScroll()
	}
	return a, nil
}

func (a App) selectPreviousMirror() (tea.Model, tea.Cmd) {
	if a.fetchIdx > 0 {
		a.fetchIdx--
		a.adjustMirrorScroll()
	}
	return a, nil
}

func (a App) scrollMirrorsDown() (tea.Model, tea.Cmd) {
	a.fetchIdx += a.packageListHeight()
	if a.fetchIdx >= len(a.fetchMirrors) {
		a.fetchIdx = len(a.fetchMirrors) - 1
	}
	if a.fetchIdx < 0 {
		a.fetchIdx = 0
	}
	a.adjustMirrorScroll()
	return a, nil
}

func (a App) scrollMirrorsUp() (tea.Model, tea.Cmd) {
	a.fetchIdx -= a.packageListHeight()
	if a.fetchIdx < 0 {
		a.fetchIdx = 0
	}
	a.adjustMirrorScroll()
	return a, nil
}

func (a App) toggleMirrorSelection() (tea.Model, tea.Cmd) {
	if len(a.fetchMirrors) > 0 && a.fetchIdx < len(a.fetchMirrors) {
		if a.fetchSelected[a.fetchIdx] {
			delete(a.fetchSelected, a.fetchIdx)
		} else {
			a.fetchSelected[a.fetchIdx] = true
		}
		a.status = fmt.Sprintf("%d mirrors selected | enter: apply • esc: cancel", len(a.fetchSelected))
	}
	return a, nil
}

func (a App) applySelectedMirrors() (tea.Model, tea.Cmd) {
	if len(a.fetchSelected) == 0 {
		a.status = ui.ErrorStyle.Render("Select at least one mirror (space to toggle).")
		return a, nil
	}
	for i := range a.fetchMirrors {
		a.fetchMirrors[i].Active = a.fetchSelected[i]
	}
	cmd := fetch.WriteSourcesListCmd(a.fetchMirrors, a.fetchDistro)
	return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return fetchApplyMsg{err: err}
	})
}
