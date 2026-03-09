package app

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/history"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
)

type tabKind int

const (
	tabAll tabKind = iota
	tabInstalled
	tabUpgradable
)

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

	searchInput textinput.Model
	searching   bool
	filterQuery string

	filterInput    textinput.Model
	filtering      bool
	advancedFilter string

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

	infoCache map[string]apt.PackageInfo

	allNamesLoaded    bool
	loadingFilterMeta bool
	installedCount    int

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

	fi := textinput.New()
	fi.Placeholder = "section: arch: size>|<|= installed upgradable name: ver: desc:"
	fi.CharLimit = 200
	fi.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0C0"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0C0"))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	return App{
		upgradableMap:    make(map[string]model.Package),
		selected:         make(map[string]bool),
		infoCache:        make(map[string]apt.PackageInfo),
		searchInput:      ti,
		filterInput:      fi,
		spinner:          s,
		help:             h,
		keys:             model.Keys,
		status:           "Loading packages...",
		loading:          true,
		transactionStore: history.Load(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, reloadAllPackages)
}
