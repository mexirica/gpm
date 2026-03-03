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
	"github.com/mexirica/gpm/internal/fuzzy"
	"github.com/mexirica/gpm/internal/model"
	"github.com/mexirica/gpm/internal/ui"
	"github.com/mexirica/gpm/internal/ui/components"
)

type App struct {
	// Data
	allPackages   []model.Package
	filtered      []model.Package
	upgradableMap map[string]model.Package

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

	return App{
		upgradableMap: make(map[string]model.Package),
		searchInput:   ti,
		spinner:       s,
		help:          help.New(),
		keys:          model.Keys,
		status:        "Loading packages...",
		loading:       true,
	}
}

func loadAllCmd() tea.Msg {
	// Load all available package names (fast: apt-cache pkgnames)
	allNames, err := apt.ListAllNames()
	if err != nil {
		// Fallback: if apt-cache pkgnames fails, just use installed
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
	return tea.ExecProcess(apt.InstallCmd(name), func(err error) tea.Msg {
		return execFinishedMsg{op: "install", name: name, err: err}
	})
}

func removeCmd(name string) tea.Cmd {
	return tea.ExecProcess(apt.RemoveCmd(name), func(err error) tea.Msg {
		return execFinishedMsg{op: "remove", name: name, err: err}
	})
}

func upgradeCmd(name string) tea.Cmd {
	return tea.ExecProcess(apt.UpgradeCmd(name), func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade", name: name, err: err}
	})
}

func upgradeAllCmd() tea.Cmd {
	return tea.ExecProcess(apt.UpgradeAllCmd(), func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade-all", name: "todos", err: err}
	})
}

type allPackagesMsg struct {
	allNames   []string
	installed  []model.Package
	upgradable []model.Package
	err        error
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
				all = append(all, model.Package{Name: name, Installed: false})
				seen[name] = true
			}
		}
		a.allPackages = all
		a.applyFilter()
		upgCount := len(msg.upgradable)
		a.status = fmt.Sprintf("%d packages (%d installed, %d upgradable) | ? help",
			len(a.allPackages), len(msg.installed), upgCount)
		// Load detail of the first package
		if len(a.filtered) > 0 {
			return a, showDetailCmd(a.filtered[0].Name)
		}
		return a, nil

	case searchResultMsg:
		a.loading = false
		if msg.err != nil {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error in search: %v", msg.err))
			return a, nil
		}
		// Mark installed and upgradable in the results
		for i := range msg.pkgs {
			if up, ok := a.upgradableMap[msg.pkgs[i].Name]; ok {
				msg.pkgs[i].Upgradable = true
				msg.pkgs[i].NewVersion = up.NewVersion
			}
		}
		a.filtered = msg.pkgs
		a.selectedIdx = 0
		a.scrollOffset = 0
		a.status = fmt.Sprintf("%d results for '%s'", len(msg.pkgs), a.filterQuery)
		if len(a.filtered) > 0 {
			return a, showDetailCmd(a.filtered[0].Name)
		}
		a.detailInfo = ""
		a.detailName = ""
		return a, nil

	case detailLoadedMsg:
		if msg.err != nil {
			a.detailInfo = fmt.Sprintf("Error: %v", msg.err)
		} else {
			a.detailInfo = msg.info
		}
		a.detailName = msg.name
		return a, nil

	case execFinishedMsg:
		a.loading = false
		if msg.err != nil {
			a.status = ui.ErrorStyle.Render(fmt.Sprintf("Error (%s %s): %v", msg.op, msg.name, msg.err))
		} else {
			a.status = ui.SuccessStyle.Render(fmt.Sprintf("✔ %s %s completed!", msg.op, msg.name))
		}
		// Reload everything
		return a, loadAllCmd

	case tea.KeyMsg:
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
			a.status = fmt.Sprintf("%d packages | ? help", len(a.filtered))
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
		return a, showDetailCmd(a.filtered[0].Name)
	case "esc":
		a.searching = false
		a.searchInput.Blur()
		a.filterQuery = ""
		a.applyFilter()
		a.status = fmt.Sprintf("%d packages | ? help", len(a.filtered))
		return a, nil
	default:
		// Update the text input first
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		// Apply fuzzy filter live as the user types
		a.filterQuery = a.searchInput.Value()
		a.applyFilter()
		a.status = fmt.Sprintf("%d matching | ? help", len(a.filtered))
		// Load detail for the top result
		var detailCmd tea.Cmd
		if len(a.filtered) > 0 {
			detailCmd = showDetailCmd(a.filtered[0].Name)
		}
		return a, tea.Batch(cmd, detailCmd)
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
			a.status = fmt.Sprintf("%d packages | ? help", len(a.filtered))
			if len(a.filtered) > 0 {
				return a, showDetailCmd(a.filtered[0].Name)
			}
			return a, nil
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
		if len(a.filtered) > 0 {
			return a, showDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil

	case msg.String() == "ctrl+u" || msg.String() == "pgup":
		a.selectedIdx -= a.listHeight()
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.adjustScroll()
		if len(a.filtered) > 0 {
			return a, showDetailCmd(a.filtered[a.selectedIdx].Name)
		}
		return a, nil

	case msg.String() == "i":
		if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
			pkg := a.filtered[a.selectedIdx]
			if pkg.Installed {
				a.status = fmt.Sprintf("'%s' is already installed.", pkg.Name)
				return a, nil
			}
			a.loading = true
			a.status = fmt.Sprintf("Installing %s...", pkg.Name)
			return a, installCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "r":
		if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
			pkg := a.filtered[a.selectedIdx]
			if !pkg.Installed {
				a.status = fmt.Sprintf("'%s' is not installed.", pkg.Name)
				return a, nil
			}
			a.loading = true
			a.status = fmt.Sprintf("Removing %s...", pkg.Name)
			return a, removeCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "u":
		if len(a.filtered) > 0 && a.selectedIdx < len(a.filtered) {
			pkg := a.filtered[a.selectedIdx]
			if !pkg.Upgradable {
				a.status = fmt.Sprintf("'%s' is already at the latest version.", pkg.Name)
				return a, nil
			}
			a.loading = true
			a.status = fmt.Sprintf("Upgrading %s...", pkg.Name)
			return a, upgradeCmd(pkg.Name)
		}
		return a, nil

	case msg.String() == "U":
		a.loading = true
		a.status = "Upgrading ALL packages (sudo apt-get upgrade)..."
		return a, upgradeAllCmd()

	case msg.String() == "ctrl+r":
		a.loading = true
		a.filterQuery = ""
		a.status = "Reloading..."
		return a, loadAllCmd
	}

	return a, nil
}

// scoredPackage pairs a package with its fuzzy match score for sorting.
type scoredPackage struct {
	pkg   model.Package
	score int
}

func (a *App) applyFilter() {
	if a.filterQuery == "" {
		a.filtered = a.allPackages
	} else {
		minScore := fuzzy.MinQuality(len(a.filterQuery))
		var scored []scoredPackage
		for _, p := range a.allPackages {
			// Score against name (primary) and description (secondary)
			nameRes := fuzzy.Score(a.filterQuery, p.Name)
			descRes := fuzzy.Score(a.filterQuery, p.Description)

			// Pick the best scoring match
			s := 0
			matched := false
			if nameRes.Matched {
				matched = true
				s = nameRes.Score + 50 // name matches always boosted
			}
			if descRes.Matched && descRes.Score > s {
				matched = true
				s = descRes.Score
			}

			// Only include if above minimum quality threshold
			if matched && s >= minScore {
				scored = append(scored, scoredPackage{pkg: p, score: s})
			}
		}
		// Sort by score descending (best matches first)
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})
		// Cap results to keep UI responsive and relevant
		const maxResults = 100
		if len(scored) > maxResults {
			scored = scored[:maxResults]
		}
		a.filtered = make([]model.Package, len(scored))
		for i, sp := range scored {
			a.filtered[i] = sp.pkg
		}
	}
	a.selectedIdx = 0
	a.scrollOffset = 0
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
	// Reserve space: header(1) + prompt(1) + separator(1) + details(detailHeight) + status(1) + help(2) + margins(2)
	detailH := a.detailHeight()
	h := a.height - detailH - 8
	if h < 5 {
		h = 5
	}
	return h
}

// detailHeight returns how many detail lines to show.
func (a App) detailHeight() int {
	if a.height <= 20 {
		return 5
	}
	if a.height <= 30 {
		return 7
	}
	if a.height <= 40 {
		return 9
	}
	return 10
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	w := a.width

	var sections []string

	// ── Header
	logo := ui.LogoStyle.Render(" GPM ") + ui.HeaderBarStyle.Width(w-6).Render()
	sections = append(sections, logo)

	// ── Package list (upper part)
	if a.loading {
		sections = append(sections, fmt.Sprintf("\n  %s Loading...\n", a.spinner.View()))
	} else {
		listView := components.RenderPackageList(a.filtered, a.selectedIdx, a.scrollOffset, a.listHeight(), w)
		sections = append(sections, listView)
	}

	// ── Search prompt (middle, fzf style "> query")
	if a.searching {
		sections = append(sections, "  "+a.searchInput.View())
	} else {
		sections = append(sections, components.RenderSearchPrompt(a.filterQuery, false))
	}

	// ── Separator
	sep := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(strings.Repeat("─", w))
	sections = append(sections, sep)

	// ── Detail panel (lower part, pacman -Qi style)
	if !a.loading && len(a.filtered) > 0 && a.detailName != "" && a.detailInfo != "" {
		pkg := a.filtered[a.selectedIdx]
		// Inject Status field that apt-cache show doesn't have
		statusLine := "Status: Not installed"
		if pkg.Upgradable {
			statusLine = "Status: Upgrade available (" + pkg.Version + " → " + pkg.NewVersion + ")"
		} else if pkg.Installed {
			statusLine = "Status: Installed"
		}
		enrichedInfo := statusLine + "\n" + a.detailInfo
		maxDetailLines := a.detailHeight()
		detail := components.RenderPackageDetail(enrichedInfo, w, maxDetailLines, 1)
		sections = append(sections, detail)
	} else if !a.loading && len(a.filtered) > 0 {
		// Show basic info while loading details
		pkg := a.filtered[a.selectedIdx]
		basic := a.renderBasicDetail(pkg)
		sections = append(sections, basic)
	}

	// ── Status bar
	sections = append(sections, components.RenderStatusBar(a.status, w))

	// ── Compact help
	sections = append(sections, ui.HelpStyle.Render(a.help.View(a.keys)))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
