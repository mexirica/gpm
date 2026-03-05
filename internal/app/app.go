// Package app provides the main application logic for the GPM package manager TUI.
package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/gpm/internal/apt"
	"github.com/mexirica/gpm/internal/fetch"
	"github.com/mexirica/gpm/internal/fuzzy"
	"github.com/mexirica/gpm/internal/history"
	"github.com/mexirica/gpm/internal/model"
	"github.com/mexirica/gpm/internal/ui"
	"github.com/mexirica/gpm/internal/ui/components"
)

type tabKind int

const (
	tabAll tabKind = iota
	tabInstalled
	tabUpgradable
)

type App struct {
	// Data
	allPackages   []model.Package
	filtered      []model.Package
	upgradableMap map[string]model.Package

	// Tab filter
	activeTab tabKind

	// Selection state
	selectedIdx  int
	scrollOffset int

	// Inline detail of the selected package
	detailInfo string
	detailName string

	// Search / filter
	searchInput textinput.Model
	searching   bool
	filterQuery string

	// Multi-selection for bulk actions (by package name)
	selected map[string]bool

	// History
	historyStore      *history.Store
	historyView       bool
	historyItems      []history.Transaction
	historyIdx        int
	historyOffset     int
	pendingExecOp     string   // operation in progress (for recording)
	pendingExecPkgs   []string // packages in progress
	pendingExecCount  int      // how many exec commands still pending
	pendingExecFailed bool     // whether any exec in the batch failed

	// Fetch mirrors
	fetchView     bool
	fetchDistro   fetch.Distro
	fetchMirrors  []fetch.Mirror // scored/sorted results
	fetchIdx      int
	fetchOffset   int
	fetchSelected map[int]bool            // user-selected mirrors by index
	fetchTesting  bool                    // true while latency tests are running
	fetchTested   int                     // how many mirrors have been tested so far
	fetchTotal    int                     // total mirrors to test
	fetchResultCh <-chan fetch.TestResult // channel for incremental results

	// Lazy info loading (version + size)
	infoCache map[string]apt.PackageInfo // cached info by package name

	// UI
	spinner spinner.Model
	help    help.Model
	keys    model.KeyMap
	status  string
	loading bool
	width   int
	height  int
}

func New() App {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.CharLimit = 100
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#5B3FC4"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#5B3FC4"))

	return App{
		upgradableMap: make(map[string]model.Package),
		selected:      make(map[string]bool),
		infoCache:     make(map[string]apt.PackageInfo),
		searchInput:   ti,
		spinner:       s,
		help:          h,
		keys:          model.Keys,
		status:        "Loading packages...",
		loading:       true,
		historyStore:  history.Load(),
	}
}

func loadAllCmd() tea.Msg {
	// Load all available package names (fast: apt-cache pkgnames)
	allNames, err := apt.ListAllNames()
	if err != nil {
		allNames = nil
	}
	installed, err := apt.ListInstalled()
	if err != nil {
		return allPackagesMsg{nil, nil, nil, err}
	}
	upgradable, _ := apt.ListUpgradable()
	return allPackagesMsg{allNames, installed, upgradable, nil}
}

func searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := apt.SearchPackages(query)
		return searchResultMsg{pkgs, err}
	}
}

func showDetailCmd(name string) tea.Cmd {
	return func() tea.Msg {
		info, err := apt.ShowPackage(name)
		return detailLoadedMsg{name, info, err}
	}
}

func installCmd(name string) tea.Cmd {
	cmd := apt.ParallelInstallCmd(name)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "install", name: name, err: err}
	})
}

func removeCmd(name string) tea.Cmd {
	return tea.ExecProcess(apt.RemoveCmd(name), func(err error) tea.Msg {
		return execFinishedMsg{op: "remove", name: name, err: err}
	})
}

func upgradeCmd(name string) tea.Cmd {
	cmd := apt.ParallelUpgradeCmd(name)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade", name: name, err: err}
	})
}

func upgradeAllCmd() tea.Cmd {
	cmd := apt.ParallelUpgradeAllCmd()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade-all", name: "todos", err: err}
	})
}

// fetchMirrorsCmd detects distro and fetches the mirror list.
func fetchMirrorsCmd() tea.Cmd {
	return func() tea.Msg {
		distro, err := fetch.DetectDistro()
		if err != nil {
			return fetchMirrorsMsg{err: err}
		}
		mirrors, err := fetch.FetchMirrorList(distro)
		if err != nil {
			return fetchMirrorsMsg{err: err}
		}
		return fetchMirrorsMsg{mirrors: mirrors, distro: distro}
	}
}

// fetchTestAllCmd is no longer used — replaced by incremental channel approach.
// waitForFetchResult reads one result from the channel and returns it as a message.
func waitForFetchResult(ch <-chan fetch.TestResult) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-ch
		if !ok {
			return fetchTestResultMsg{done: true}
		}
		return fetchTestResultMsg{result: r, done: false}
	}
}

type allPackagesMsg struct {
	allNames   []string
	installed  []model.Package
	upgradable []model.Package
	err        error
}

type infoLoadedMsg struct {
	info map[string]apt.PackageInfo
}

type searchResultMsg struct {
	pkgs []model.Package
	err  error
}

type detailLoadedMsg struct {
	name string
	info string
	err  error
}

type execFinishedMsg struct {
	op   string
	name string
	err  error
}

// Fetch mirror messages
type fetchMirrorsMsg struct {
	mirrors []fetch.Mirror
	distro  fetch.Distro
	err     error
}

type fetchTestResultMsg struct {
	result fetch.TestResult
	done   bool // true when channel is closed (all tests done)
}

type fetchApplyMsg struct {
	err error
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, loadAllCmd)
}

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
		a.loading = false
		if msg.err != nil {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
			return a, nil
		}
		// Build upgradable map
		a.upgradableMap = make(map[string]model.Package)
		for _, p := range msg.upgradable {
			a.upgradableMap[p.Name] = p
		}
		// Build installed map for fast lookup
		installedMap := make(map[string]model.Package, len(msg.installed))
		for _, p := range msg.installed {
			if up, ok := a.upgradableMap[p.Name]; ok {
				p.Upgradable = true
				p.NewVersion = up.NewVersion
			}
			installedMap[p.Name] = p
		}
		// Merge: start with installed packages, then add all remaining available names
		seen := make(map[string]bool, len(msg.installed)+len(msg.allNames))
		var all []model.Package
		// Installed first (they have full info)
		for _, p := range msg.installed {
			if up, ok := a.upgradableMap[p.Name]; ok {
				p.Upgradable = true
				p.NewVersion = up.NewVersion
			}
			all = append(all, p)
			seen[p.Name] = true
		}
		// Then all available names not already in installed
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
		a.applyFilter()
		upgCount := len(msg.upgradable)
		a.status = fmt.Sprintf("%d packages (%d installed, %d upgradable) ",
			len(a.allPackages), len(msg.installed), upgCount)
		// Load detail + versions for visible packages
		var cmds []tea.Cmd
		if len(a.filtered) > 0 {
			cmds = append(cmds, showDetailCmd(a.filtered[0].Name))
		}
		cmds = append(cmds, a.loadVisibleVersionsCmd())
		return a, tea.Batch(cmds...)

	case infoLoadedMsg:
		// Merge fetched info into cache and update packages
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

	case searchResultMsg:
		a.loading = false
		if msg.err != nil {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error in search: %v", msg.err))
			return a, nil
		}
		// Build lookup from allPackages for installed info
		installedMap := make(map[string]model.Package, len(a.allPackages))
		for _, p := range a.allPackages {
			if p.Installed {
				installedMap[p.Name] = p
			}
		}
		// Enrich search results with version/size from installed or cache
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
			return a, tea.Batch(showDetailCmd(a.filtered[0].Name), a.loadVisibleVersionsCmd())
		}
		a.detailInfo = ""
		a.detailName = ""
		return a, nil

	case detailLoadedMsg:
		if msg.err != nil {
			a.detailInfo = fmt.Sprintf("Error: %v", msg.err)
		} else {
			a.detailInfo = msg.info
			// Parse version/size from the detail info and update the list
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

	case execFinishedMsg:
		if msg.err != nil {
			a.pendingExecFailed = true
		}
		a.pendingExecCount--
		if a.pendingExecCount > 0 {
			return a, nil
		}

		a.loading = false
		// Record in history (once for the whole batch)
		success := !a.pendingExecFailed
		op := history.Operation(a.pendingExecOp)
		pkgs := a.pendingExecPkgs
		if len(pkgs) == 0 {
			pkgs = []string{msg.name}
		}
		a.historyStore.Record(op, pkgs, success)
		a.pendingExecPkgs = nil
		a.pendingExecOp = ""
		a.pendingExecFailed = false

		if !success {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error (%s %s): %v", msg.op, msg.name, msg.err))
		} else {
			a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ %s %s completed!", msg.op, msg.name))
		}
		return a, loadAllCmd

	case fetchMirrorsMsg:
		if msg.err != nil {
			a.fetchView = false
			a.loading = false
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Fetch error: %v", msg.err))
			return a, nil
		}
		a.fetchDistro = msg.distro
		// Limit mirrors to 50 for faster testing
		a.fetchMirrors = fetch.LimitMirrors(msg.mirrors, 50)
		a.fetchTesting = true
		a.fetchTested = 0
		a.fetchTotal = len(a.fetchMirrors)
		a.status = fmt.Sprintf("Testing %d mirrors for %s...", a.fetchTotal, msg.distro.Name)
		// Start channel-based incremental testing
		a.fetchResultCh = fetch.TestMirrorsChan(a.fetchMirrors)
		return a, tea.Batch(a.spinner.Tick, waitForFetchResult(a.fetchResultCh))

	case fetchTestResultMsg:
		if msg.done {
			// All tests complete — score and sort
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
		// Process individual result
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
		// Request next result
		return a, waitForFetchResult(a.fetchResultCh)

	case fetchApplyMsg:
		if msg.err != nil {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error writing mirrors: %v", msg.err))
		} else {
			a.status = ui.SuccessStyle.Render("✔ Mirrors saved! Run apt update to apply.")
		}
		a.fetchView = false
		return a, nil

	case tea.KeyMsg:
		if a.fetchView {
			return a.handleFetchKeypress(msg)
		}
		if a.historyView {
			return a.handleHistoryKeypress(msg)
		}
		if a.searching {
			return a.handleSearchInput(msg)
		}
		return a.handleKeypress(msg)
	}

	return a, nil
}

func (a App) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := a.searchInput.Value()
		a.searching = false
		a.searchInput.Blur()
		a.filterQuery = query
		if query == "" {
			a.applyFilter()
			a.status = fmt.Sprintf("%d packages ", len(a.filtered))
			if len(a.filtered) > 0 {
				return a, showDetailCmd(a.filtered[0].Name)
			}
			return a, nil
		}
		// Filter is already applied live; if nothing matched locally, try apt-cache
		if len(a.filtered) == 0 {
			a.loading = true
			a.status = fmt.Sprintf("Searching '%s' via apt-cache...", query)
			return a, searchCmd(query)
		}
		a.status = fmt.Sprintf("%d packages matching '%s'", len(a.filtered), query)
		return a, tea.Batch(showDetailCmd(a.filtered[0].Name), a.loadVisibleVersionsCmd())
	case "esc":
		a.searching = false
		a.searchInput.Blur()
		a.filterQuery = ""
		a.applyFilter()
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		return a, nil
	default:
		// Update the text input first
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		// Apply fuzzy filter live as the user types
		a.filterQuery = a.searchInput.Value()
		a.applyFilter()
		a.status = fmt.Sprintf("%d matching ", len(a.filtered))
		// Load detail + versions for the top result
		var detailCmd tea.Cmd
		if len(a.filtered) > 0 {
			detailCmd = showDetailCmd(a.filtered[0].Name)
		}
		return a, tea.Batch(cmd, detailCmd, a.loadVisibleVersionsCmd())
	}
}

func (a App) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		return a, tea.Quit

	case msg.String() == "?":
		a.help.ShowAll = !a.help.ShowAll
		return a, nil

	case msg.String() == "/":
		a.searching = true
		a.searchInput.Focus()
		a.searchInput.SetValue(a.filterQuery)
		return a, textinput.Blink

	case msg.String() == "esc":
		if a.filterQuery != "" {
			a.filterQuery = ""
			a.applyFilter()
			a.selectedIdx = 0
			a.scrollOffset = 0
			a.status = fmt.Sprintf("%d packages ", len(a.filtered))
			var cmds []tea.Cmd
			if len(a.filtered) > 0 {
				cmds = append(cmds, showDetailCmd(a.filtered[0].Name))
			}
			cmds = append(cmds, a.loadVisibleVersionsCmd())
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case msg.String() == "j" || msg.String() == "down":
		if a.selectedIdx < len(a.filtered)-1 {
			a.selectedIdx++
			a.adjustScroll()
			return a, showDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil

	case msg.String() == "k" || msg.String() == "up":
		if a.selectedIdx > 0 {
			a.selectedIdx--
			a.adjustScroll()
			return a, showDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil

	case msg.String() == "ctrl+d" || msg.String() == "pgdown":
		a.selectedIdx += a.listHeight()
		if a.selectedIdx >= len(a.filtered) {
			a.selectedIdx = len(a.filtered) - 1
		}
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.adjustScroll()
		var cmds []tea.Cmd
		if len(a.filtered) > 0 {
			cmds = append(cmds, showDetailCmd(a.filtered[a.selectedIdx].Name))
		}
		cmds = append(cmds, a.loadVisibleVersionsCmd())
		return a, tea.Batch(cmds...)

	case msg.String() == "ctrl+u" || msg.String() == "pgup":
		a.selectedIdx -= a.listHeight()
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.adjustScroll()
		var cmds []tea.Cmd
		if len(a.filtered) > 0 {
			cmds = append(cmds, showDetailCmd(a.filtered[a.selectedIdx].Name))
		}
		cmds = append(cmds, a.loadVisibleVersionsCmd())
		return a, tea.Batch(cmds...)

	// Multi-selection keys
	case msg.String() == " ":
		// toggle selection of current package
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
			count := len(a.selected)
			a.status = fmt.Sprintf("%d selected ", count)
			return a, nil
		}
		return a, nil

	case msg.String() == "A":
		// toggle select all filtered
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
			// clear
			a.selected = make(map[string]bool)
			a.status = fmt.Sprintf("0 selected ")
			return a, nil
		}
		for _, p := range a.filtered {
			a.selected[p.Name] = true
		}
		a.status = fmt.Sprintf("%d selected ", len(a.selected))
		return a, nil

	case msg.String() == "i":
		if len(a.selected) > 0 {
			var cmds []tea.Cmd
			var names []string
			for name := range a.selected {
				cmds = append(cmds, installCmd(name))
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
			return a, installCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "r":
		if len(a.selected) > 0 {
			var cmds []tea.Cmd
			var names []string
			for name := range a.selected {
				cmds = append(cmds, removeCmd(name))
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
			return a, removeCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "u":
		if len(a.selected) > 0 {
			var cmds []tea.Cmd
			var names []string
			for name := range a.selected {
				cmds = append(cmds, upgradeCmd(name))
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
			return a, upgradeCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "G":
		a.pendingExecOp = "upgrade-all"
		a.pendingExecPkgs = []string{"all"}
		a.pendingExecCount = 1
		a.loading = true
		a.status = "Upgrading ALL packages (sudo apt-get upgrade)..."
		return a, upgradeAllCmd()

	case msg.String() == "ctrl+r":
		a.loading = true
		a.filterQuery = ""
		a.status = "Reloading..."
		return a, loadAllCmd

	case msg.String() == "h":
		a.historyView = true
		a.historyItems = a.historyStore.All()
		a.historyIdx = 0
		a.historyOffset = 0
		a.status = fmt.Sprintf("%d transactions | esc back | z undo | x redo ", len(a.historyItems))
		return a, nil

	case msg.String() == "f":
		a.fetchView = true
		a.fetchMirrors = nil
		a.fetchSelected = make(map[int]bool)
		a.fetchIdx = 0
		a.fetchOffset = 0
		a.fetchTesting = true
		a.loading = true
		a.status = "Detecting distro and fetching mirror list..."
		return a, tea.Batch(a.spinner.Tick, fetchMirrorsCmd())

	case msg.String() == "tab":
		a.activeTab = (a.activeTab + 1) % 3
		a.applyFilter()
		var cmds []tea.Cmd
		if len(a.filtered) > 0 {
			cmds = append(cmds, showDetailCmd(a.filtered[0].Name))
		}
		cmds = append(cmds, a.loadVisibleVersionsCmd())
		tabNames := []string{"All", "Installed", "Upgradable"}
		a.status = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabNames[a.activeTab])
		return a, tea.Batch(cmds...)

	case msg.String() == "shift+tab":
		a.activeTab = (a.activeTab + 2) % 3
		a.applyFilter()
		var cmds []tea.Cmd
		if len(a.filtered) > 0 {
			cmds = append(cmds, showDetailCmd(a.filtered[0].Name))
		}
		cmds = append(cmds, a.loadVisibleVersionsCmd())
		tabNames := []string{"All", "Installed", "Upgradable"}
		a.status = fmt.Sprintf("%d packages (%s) ", len(a.filtered), tabNames[a.activeTab])
		return a, tea.Batch(cmds...)
	}

	return a, nil
}

// handleFetchKeypress handles key events in the fetch mirrors view.
func (a App) handleFetchKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Don't allow navigation while testing
	if a.fetchTesting {
		if msg.String() == "esc" || msg.String() == "q" || msg.String() == "ctrl+c" {
			a.fetchView = false
			a.fetchTesting = false
			a.loading = false
			a.status = "Fetch cancelled."
			return a, nil
		}
		return a, nil
	}

	switch {
	case msg.String() == "esc":
		a.fetchView = false
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		return a, nil

	case msg.String() == "q" || msg.String() == "ctrl+c":
		return a, tea.Quit

	case msg.String() == "j" || msg.String() == "down":
		if a.fetchIdx < len(a.fetchMirrors)-1 {
			a.fetchIdx++
			a.adjustFetchScroll()
		}
		return a, nil

	case msg.String() == "k" || msg.String() == "up":
		if a.fetchIdx > 0 {
			a.fetchIdx--
			a.adjustFetchScroll()
		}
		return a, nil

	case msg.String() == "ctrl+d" || msg.String() == "pgdown":
		a.fetchIdx += a.listHeight()
		if a.fetchIdx >= len(a.fetchMirrors) {
			a.fetchIdx = len(a.fetchMirrors) - 1
		}
		if a.fetchIdx < 0 {
			a.fetchIdx = 0
		}
		a.adjustFetchScroll()
		return a, nil

	case msg.String() == "ctrl+u" || msg.String() == "pgup":
		a.fetchIdx -= a.listHeight()
		if a.fetchIdx < 0 {
			a.fetchIdx = 0
		}
		a.adjustFetchScroll()
		return a, nil

	case msg.String() == " ":
		if len(a.fetchMirrors) > 0 && a.fetchIdx < len(a.fetchMirrors) {
			if a.fetchSelected[a.fetchIdx] {
				delete(a.fetchSelected, a.fetchIdx)
			} else {
				a.fetchSelected[a.fetchIdx] = true
			}
			a.status = fmt.Sprintf("%d mirrors selected | enter: apply • esc: cancel", len(a.fetchSelected))
		}
		return a, nil

	case msg.String() == "enter":
		if len(a.fetchSelected) == 0 {
			a.status = ui.ErrorStyle.Render("Select at least one mirror (space to toggle).")
			return a, nil
		}
		// Mark selected mirrors as active
		for i := range a.fetchMirrors {
			a.fetchMirrors[i].Active = a.fetchSelected[i]
		}
		// Write sources list via sudo
		cmd := fetch.WriteSourcesListCmd(a.fetchMirrors, a.fetchDistro)
		return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return fetchApplyMsg{err: err}
		})
	}
	return a, nil
}

func (a *App) adjustFetchScroll() {
	h := a.listHeight()
	if a.fetchIdx < a.fetchOffset {
		a.fetchOffset = a.fetchIdx
	}
	if a.fetchIdx >= a.fetchOffset+h {
		a.fetchOffset = a.fetchIdx - h + 1
	}
}

// handleHistoryKeypress handles key events when the history view is active.
func (a App) handleHistoryKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc" || msg.String() == "h":
		a.historyView = false
		a.status = fmt.Sprintf("%d packages ", len(a.filtered))
		return a, nil

	case msg.String() == "q" || msg.String() == "ctrl+c":
		return a, tea.Quit

	case msg.String() == "?":
		a.help.ShowAll = !a.help.ShowAll
		return a, nil

	case msg.String() == "j" || msg.String() == "down":
		if a.historyIdx < len(a.historyItems)-1 {
			a.historyIdx++
			a.adjustHistoryScroll()
		}
		return a, nil

	case msg.String() == "k" || msg.String() == "up":
		if a.historyIdx > 0 {
			a.historyIdx--
			a.adjustHistoryScroll()
		}
		return a, nil

	case msg.String() == "ctrl+d" || msg.String() == "pgdown":
		a.historyIdx += a.listHeight()
		if a.historyIdx >= len(a.historyItems) {
			a.historyIdx = len(a.historyItems) - 1
		}
		if a.historyIdx < 0 {
			a.historyIdx = 0
		}
		a.adjustHistoryScroll()
		return a, nil

	case msg.String() == "ctrl+u" || msg.String() == "pgup":
		a.historyIdx -= a.listHeight()
		if a.historyIdx < 0 {
			a.historyIdx = 0
		}
		a.adjustHistoryScroll()
		return a, nil

	case msg.String() == "z":
		if len(a.historyItems) > 0 && a.historyIdx < len(a.historyItems) {
			tx := a.historyItems[a.historyIdx]
			if !tx.Success {
				a.status = ui.ErrorStyle.Render("Cannot undo a failed transaction.")
				return a, nil
			}
			undoOp := history.UndoOperation(tx.Operation)
			var cmds []tea.Cmd
			for _, pkg := range tx.Packages {
				switch undoOp {
				case history.OpRemove:
					cmds = append(cmds, removeCmd(pkg))
				case history.OpInstall:
					cmds = append(cmds, installCmd(pkg))
				}
			}
			a.pendingExecOp = string(undoOp)
			a.pendingExecPkgs = tx.Packages
			a.pendingExecCount = len(cmds)
			a.historyView = false
			a.loading = true
			a.status = fmt.Sprintf("Undoing #%d (%s %d packages)...", tx.ID, undoOp, len(tx.Packages))
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case msg.String() == "x":
		if len(a.historyItems) > 0 && a.historyIdx < len(a.historyItems) {
			tx := a.historyItems[a.historyIdx]
			var cmds []tea.Cmd
			for _, pkg := range tx.Packages {
				switch tx.Operation {
				case history.OpInstall:
					cmds = append(cmds, installCmd(pkg))
				case history.OpRemove:
					cmds = append(cmds, removeCmd(pkg))
				case history.OpUpgrade:
					cmds = append(cmds, upgradeCmd(pkg))
				case history.OpUpgradeAll:
					cmds = append(cmds, upgradeAllCmd())
				}
			}
			a.pendingExecOp = string(tx.Operation)
			a.pendingExecPkgs = tx.Packages
			a.pendingExecCount = len(cmds)
			a.historyView = false
			a.loading = true
			a.status = fmt.Sprintf("Redoing #%d (%s %d packages)...", tx.ID, tx.Operation, len(tx.Packages))
			return a, tea.Batch(cmds...)
		}
		return a, nil
	}

	return a, nil
}

func (a *App) adjustHistoryScroll() {
	h := a.listHeight()
	if a.historyIdx < a.historyOffset {
		a.historyOffset = a.historyIdx
	}
	if a.historyIdx >= a.historyOffset+h {
		a.historyOffset = a.historyIdx - h + 1
	}
}

// scoredPackage pairs a package with its fuzzy match score for sorting.
type scoredPackage struct {
	pkg   model.Package
	score int
}

func (a *App) applyFilter() {
	// Start from allPackages, apply tab filter first
	var source []model.Package
	switch a.activeTab {
	case tabInstalled:
		for _, p := range a.allPackages {
			if p.Installed {
				source = append(source, p)
			}
		}
	case tabUpgradable:
		for _, p := range a.allPackages {
			if p.Upgradable {
				source = append(source, p)
			}
		}
	default:
		source = a.allPackages
	}

	if a.filterQuery == "" {
		a.filtered = source
	} else {
		minScore := fuzzy.MinQuality(len(a.filterQuery))
		var scored []scoredPackage
		for _, p := range source {
			nameRes := fuzzy.Score(a.filterQuery, p.Name)
			descRes := fuzzy.Score(a.filterQuery, p.Description)

			s := 0
			matched := false
			if nameRes.Matched {
				matched = true
				s = nameRes.Score + 50
			}
			if descRes.Matched && descRes.Score > s {
				matched = true
				s = descRes.Score
			}

			if matched && s >= minScore {
				scored = append(scored, scoredPackage{pkg: p, score: s})
			}
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		a.filtered = make([]model.Package, len(scored))
		for i, sp := range scored {
			a.filtered[i] = sp.pkg
		}
	}
	a.selectedIdx = 0
	a.scrollOffset = 0
}

// loadVisibleVersionsCmd returns a Cmd that fetches version and size info for
// visible packages that are not yet in the cache.
func (a *App) loadVisibleVersionsCmd() tea.Cmd {
	if len(a.filtered) == 0 {
		return nil
	}
	h := a.listHeight()
	start := a.scrollOffset
	// prefetch a buffer around the viewport
	end := start + h + 50
	if start > 20 {
		start -= 20
	} else {
		start = 0
	}
	if end > len(a.filtered) {
		end = len(a.filtered)
	}
	var names []string
	for i := start; i < end; i++ {
		name := a.filtered[i].Name
		if _, ok := a.infoCache[name]; !ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return func() tea.Msg {
		info := apt.BatchGetInfo(names)
		return infoLoadedMsg{info: info}
	}
}

func (a *App) adjustScroll() {
	h := a.listHeight()
	if a.selectedIdx < a.scrollOffset {
		a.scrollOffset = a.selectedIdx
	}
	if a.selectedIdx >= a.scrollOffset+h {
		a.scrollOffset = a.selectedIdx - h + 1
	}
}

// listHeight returns how many package lines fit in the upper half.
func (a App) listHeight() int {
	// Layout: tabBar(1) + colHeader(1) + colSep(1) + list(h) + counter(1) + search(1) + sep(1) + detail(10) + status(1) + help(N) = height
	helpLines := strings.Count(a.help.View(a.keys), "\n") + 1
	h := a.height - a.detailHeight() - 9 - helpLines
	if h < 5 {
		h = 5
	}
	return h
}

// detailHeight returns the fixed number of detail lines (matches displayFields count).
func (a App) detailHeight() int {
	return 10
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	w := a.width

	// ── Fetch mirrors view
	if a.fetchView {
		return a.renderFetchView(w)
	}

	// ── History view
	if a.historyView {
		return a.renderHistoryView(w)
	}

	// ── 1. Tab bar + Package list (upper region)
	tabBar := a.renderTabBar()
	var listView string
	if a.loading {
		listView = fmt.Sprintf("\n  %s Loading...\n", a.spinner.View())
	} else {
		listView = components.RenderPackageList(a.filtered, a.selectedIdx, a.scrollOffset, a.listHeight(), w, a.selected)
	}
	listView = tabBar + "\n" + listView

	// ── 2. Footer (pinned to terminal bottom)
	var footer []string

	// Package counter
	counterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888AA"))
	pos := a.selectedIdx + 1
	if len(a.filtered) == 0 {
		pos = 0
	}
	counterText := fmt.Sprintf("  %d/%d", pos, len(a.filtered))
	footer = append(footer, counterStyle.Render(counterText))

	if a.searching {
		footer = append(footer, "  "+a.searchInput.View())
	} else {
		footer = append(footer, components.RenderSearchPrompt(a.filterQuery, false))
	}

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if !a.loading && len(a.filtered) > 0 && a.detailName != "" && a.detailInfo != "" {
		pkg := a.filtered[a.selectedIdx]
		statusLine := "Status: Not installed"
		if pkg.Upgradable {
			statusLine = "Status: Upgrade available (" + pkg.Version + " → " + pkg.NewVersion + ")"
		} else if pkg.Installed {
			statusLine = "Status: Installed"
		}
		enrichedInfo := statusLine + "\n" + a.detailInfo
		maxDetailLines := a.detailHeight()
		detail := components.RenderPackageDetail(enrichedInfo, w, maxDetailLines, 1)
		footer = append(footer, detail)
	} else if !a.loading && len(a.filtered) > 0 {
		pkg := a.filtered[a.selectedIdx]
		basic := a.renderBasicDetail(pkg)
		footer = append(footer, basic)
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	footer = append(footer, ui.HelpStyle.Render(a.help.View(a.keys)))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)

	// ── 3. Spacer: push footer to the bottom
	listLines := strings.Count(listView, "\n")
	footerLines := strings.Count(footerView, "\n") + 1
	gap := a.height - listLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return listView + strings.Repeat("\n", gap) + footerView
}

// renderTabBar renders the tab bar for filtering packages.
func (a App) renderTabBar() string {
	tabs := []struct {
		label string
		kind  tabKind
	}{
		{" ◉ All ", tabAll},
		{" ● Installed ", tabInstalled},
		{" ↑ Upgradable ", tabUpgradable},
	}

	var parts []string
	hasUpgradable := len(a.upgradableMap) > 0
	for _, t := range tabs {
		if t.kind == a.activeTab {
			parts = append(parts, ui.TabActiveStyle.Render(t.label))
		} else if t.kind == tabUpgradable && hasUpgradable {
			parts = append(parts, ui.TabNotifyStyle.Render(t.label))
		} else {
			parts = append(parts, ui.TabInactiveStyle.Render(t.label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderBasicDetail shows basic package info when apt-cache show hasn't loaded yet.
func (a App) renderBasicDetail(pkg model.Package) string {
	lbl := lipgloss.NewStyle().
		Foreground(ui.ColorWhite).Bold(true).Width(18).Align(lipgloss.Right)
	sep := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Name"), sep.Render(":"), val.Render(pkg.Name)))
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Version"), sep.Render(":"), val.Render(pkg.Version)))

	status := "Not installed"
	statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	if pkg.Upgradable {
		status = "Upgrade available"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
	} else if pkg.Installed {
		status = "Installed"
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
	}
	b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Status"), sep.Render(":"), statusStyle.Render(status)))

	if pkg.NewVersion != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("New Version"), sep.Render(":"),
			lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true).Render(pkg.NewVersion)))
	}
	if pkg.Description != "" {
		b.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Description"), sep.Render(":"), val.Render(pkg.Description)))
	}

	return b.String()
}

// renderFetchView renders the fetch mirrors screen.
func (a App) renderFetchView(w int) string {
	header := components.RenderFetchHeader(a.fetchDistro)

	var upperView string
	if a.fetchTesting {
		progress := components.RenderFetchProgress(a.fetchTested, a.fetchTotal)
		upperView = header + "\n\n" + fmt.Sprintf("  %s %s\n", a.spinner.View(), progress)
	} else {
		listView := components.RenderMirrorList(a.fetchMirrors, a.fetchIdx, a.fetchOffset, a.listHeight(), w, a.fetchSelected)
		upperView = header + "\n" + listView
	}

	// Footer
	var footer []string

	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	sel := len(a.fetchSelected)
	total := len(a.fetchMirrors)
	footer = append(footer, counterStyle.Render(fmt.Sprintf("  %d/%d mirrors selected", sel, total)))

	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	// Detail of selected mirror
	if !a.fetchTesting && len(a.fetchMirrors) > 0 && a.fetchIdx < len(a.fetchMirrors) {
		m := a.fetchMirrors[a.fetchIdx]
		lbl := lipgloss.NewStyle().Foreground(ui.ColorWhite).Bold(true).Width(14).Align(lipgloss.Right)
		sepChar := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		val := lipgloss.NewStyle().Foreground(ui.ColorWhite)

		var detail strings.Builder
		detail.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("URL"), sepChar.Render(":"), val.Render(m.URL)))
		detail.WriteString(fmt.Sprintf("  %s %s %s\n", lbl.Render("Latency"), sepChar.Render(":"), val.Render(fetch.FormatLatency(m.Latency))))
		detail.WriteString(fmt.Sprintf("  %s %s %d\n", lbl.Render("Score"), sepChar.Render(":"), m.Score))
		footer = append(footer, detail.String())
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	helpLine := components.RenderFetchFooterHelp()
	footer = append(footer, lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(helpLine))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)

	listLines := strings.Count(upperView, "\n")
	footerLines := strings.Count(footerView, "\n") + 1
	gap := a.height - listLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return upperView + strings.Repeat("\n", gap) + footerView
}

// renderHistoryView renders the full history screen.
func (a App) renderHistoryView(w int) string {
	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite).Background(ui.ColorPrimary).Padding(0, 2)
	title := titleStyle.Render(" GPM Transaction History ")

	// ── 1. Upper region: title + list
	listView := components.RenderHistoryList(a.historyItems, a.historyIdx, a.historyOffset, a.listHeight(), w)
	upperView := title + "\n" + listView

	// ── 2. Footer (pinned to bottom)
	var footer []string

	// Transaction counter
	counterStyle := lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	footer = append(footer, counterStyle.Render(fmt.Sprintf("  %d transactions", len(a.historyItems))))

	// Separator + detail
	sep := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("─", w))
	footer = append(footer, sep)

	if len(a.historyItems) > 0 && a.historyIdx < len(a.historyItems) {
		tx := a.historyItems[a.historyIdx]
		detail := components.RenderHistoryDetail(tx, w, a.detailHeight())
		footer = append(footer, detail)
	}

	footer = append(footer, components.RenderStatusBar(a.status, w))
	footer = append(footer, ui.HelpStyle.Render(a.help.View(a.keys)))

	footerView := lipgloss.JoinVertical(lipgloss.Left, footer...)

	// ── 3. Spacer: push footer to the bottom
	listLines := strings.Count(upperView, "\n")
	footerLines := strings.Count(footerView, "\n") + 1
	gap := a.height - listLines - footerLines
	if gap < 0 {
		gap = 0
	}

	return upperView + strings.Repeat("\n", gap) + footerView
}
