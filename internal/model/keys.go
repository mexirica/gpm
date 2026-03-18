// Package model defines the data structures and key bindings for the application.
package model

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit        key.Binding
	Help        key.Binding
	Enter       key.Binding
	Back        key.Binding
	Search      key.Binding
	Install     key.Binding
	Remove      key.Binding
	Upgrade     key.Binding
	UpgradeAll  key.Binding
	Purge       key.Binding
	Hold        key.Binding
	Select      key.Binding
	SelectAll   key.Binding
	Refresh     key.Binding
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Transaction key.Binding
	TranUndo    key.Binding
	TranRedo    key.Binding
	Fetch       key.Binding
	AptUpdate   key.Binding
	CleanupAll  key.Binding
	ErrLogClear key.Binding
	PPA         key.Binding
	Pin         key.Binding
	Tab         key.Binding
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "help"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search/filter"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove"),
	),
	Upgrade: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "upgrade"),
	),
	UpgradeAll: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "upgrade all"),
	),
	Purge: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "purge"),
	),
	Hold: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "hold/unhold"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),

	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("pgdown", "page down"),
	),
	Transaction: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "transactions"),
	),
	TranUndo: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "undo"),
	),
	TranRedo: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "redo"),
	),
	Fetch: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "fetch"),
	),
	AptUpdate: key.NewBinding(
		key.WithKeys("U"),
		key.WithHelp("U", "apt update"),
	),
	CleanupAll: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clean up"),
	),
	ErrLogClear: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "clear errors"),
	),
	PPA: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "PPA repos"),
	),
	Pin: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "pin/unpin"),
	),

	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch tab"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.Select, k.SelectAll, k.Install, k.Remove, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Tab},
		{k.Enter, k.Select, k.SelectAll, k.Search},
		{k.Install, k.Remove, k.Upgrade, k.UpgradeAll, k.Purge, k.Hold, k.Pin},
		{k.CleanupAll, k.ErrLogClear, k.AptUpdate, k.Fetch, k.PPA, k.Refresh, k.Transaction},
		{k.TranUndo, k.TranRedo, k.Help, k.Quit},
	}
}
