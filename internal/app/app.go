// Package app provides the main Bubbletea application model and logic for the aptui TUI.
package app

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/errlog"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/history"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/pin"
	"github.com/mexirica/aptui/internal/portpkg"
	"github.com/mexirica/aptui/internal/ui"
)

type tabKind int

const (
	tabAll tabKind = iota
	tabInstalled
	tabUpgradable
	tabCleanup
	tabErrorLog
)

type tabDef struct {
	label string
	kind  tabKind
	name  string
}

var tabDefs = []tabDef{
	{" ◉ All ", tabAll, "All"},
	{" ● Installed ", tabInstalled, "Installed"},
	{" ↑ Upgradable ", tabUpgradable, "Upgradable"},
	{" 🧹 Cleanup ", tabCleanup, "Cleanup"},
	{" ❌ Errors ", tabErrorLog, "Errors"},
}

// App is the main Bubbletea model. It manages three views:
// the package list (default), the transaction history, and the mirror selector.
type App struct {
	allPackages   []model.Package
	filtered      []model.Package
	upgradableMap map[string]model.Package

	activeTab tabKind

	selectedIdx  int
	scrollOffset int

	detailInfo string
	detailName string

	// Search state
	searchInput           textinput.Model
	searching             bool
	filterQuery           string
	filterQueryBeforeEdit string

	selected map[string]bool

	sortColumn filter.SortColumn
	sortDesc   bool

	transactionStore  *history.Store
	transactionView   bool
	transactionItems  []history.Transaction
	transactionIdx    int
	transactionOffset int
	transactionDeps   []string
	pendingExecOp     string
	pendingExecPkgs   []string
	pendingExecCount  int
	pendingExecFailed bool

	fetchView     bool
	fetchDistro   fetch.Distro
	fetchMirrors  []fetch.Mirror
	fetchIdx      int
	fetchOffset   int
	fetchSelected map[int]bool
	fetchTesting  bool
	fetchTested   int
	fetchTotal    int
	fetchResultCh <-chan fetch.TestResult

	ppaView   bool
	ppaItems  []apt.PPA
	ppaIdx    int
	ppaOffset int
	ppaAdding bool
	ppaInput  textinput.Model

	infoCache map[string]apt.PackageInfo
	pkgIndex  map[string]int

	autoremovable    []string
	autoremovableSet map[string]bool

	heldSet     map[string]bool
	holdPending int
	holdFailed  bool

	essentialSet map[string]bool

	pinStore  *pin.Store
	pinnedSet map[string]bool

	allNamesLoaded bool
	installedCount int

	importingPath      bool
	importInput        textinput.Model
	importConfirm      bool
	importDetails      bool
	importDetailOffset int
	importToInstall    []string
	importFromPath     string

	errlogStore  *errlog.Store
	errlogItems  []errlog.Entry
	errlogIdx    int
	errlogOffset int

	spinner       spinner.Model
	help          help.Model
	keys          model.KeyMap
	status        string
	statusLock    time.Time
	pendingStatus string
	loading       bool
	width         int
	height        int
}

func New() App {
	ti := textinput.New()
	ti.Placeholder = "Search or filter: section: arch: size> installed ..."
	ti.CharLimit = 200
	ti.Width = 80

	pi := textinput.New()
	pi.Placeholder = "ppa:user/repository"
	pi.CharLimit = 100
	pi.Width = 50

	ii := textinput.New()
	ii.Placeholder = portpkg.DefaultPath()
	ii.CharLimit = 300
	ii.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0C0"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0C0"))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	ps := pin.Load()

	return App{
		upgradableMap:    make(map[string]model.Package),
		selected:         make(map[string]bool),
		infoCache:        make(map[string]apt.PackageInfo),
		pkgIndex:         make(map[string]int),
		autoremovableSet: make(map[string]bool),
		heldSet:          make(map[string]bool),
		essentialSet:     make(map[string]bool),
		pinStore:         ps,
		pinnedSet:        ps.Set(),
		searchInput:      ti,
		ppaInput:         pi,
		importInput:      ii,
		spinner:          s,
		help:             h,
		keys:             model.Keys,
		status:           "Loading packages...",
		loading:          true,
		transactionStore: history.Load(),
		errlogStore:      errlog.Load(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, reloadAllPackages, loadAutoremovableCmd(), loadHeldCmd())
}
