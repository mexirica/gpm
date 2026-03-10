package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mexirica/aptui/internal/model"
	"github.com/mexirica/aptui/internal/ui"
)

func newTestApp() App {
	a := New()
	a.width = 120
	a.height = 40
	a.loading = false
	return a
}

func TestNewApp(t *testing.T) {
	a := New()
	if a.upgradableMap == nil {
		t.Error("upgradableMap should be initialized")
	}
	if a.selected == nil {
		t.Error("selected should be initialized")
	}
	if a.infoCache == nil {
		t.Error("infoCache should be initialized")
	}
	if !a.loading {
		t.Error("app should start in loading state")
	}
	if a.status != "Loading packages..." {
		t.Errorf("unexpected initial status: %s", a.status)
	}
	if a.transactionStore == nil {
		t.Error("transactionStore should be initialized")
	}
}

func TestApplyFilterAll(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true, Upgradable: true},
		{Name: "curl", Installed: false},
	}
	a.activeTab = tabAll
	a.filterQuery = ""
	a.applyFilter()

	if len(a.filtered) != 3 {
		t.Errorf("expected 3 packages on All tab, got %d", len(a.filtered))
	}
}

func TestApplyFilterInstalledTab(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true, Upgradable: true},
		{Name: "curl", Installed: false},
	}
	a.activeTab = tabInstalled
	a.filterQuery = ""
	a.applyFilter()

	if len(a.filtered) != 2 {
		t.Errorf("expected 2 installed packages, got %d", len(a.filtered))
	}
	for _, p := range a.filtered {
		if !p.Installed {
			t.Errorf("non-installed package in Installed tab: %s", p.Name)
		}
	}
}

func TestApplyFilterUpgradableTab(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true, Upgradable: true},
		{Name: "curl", Installed: false},
	}
	a.activeTab = tabUpgradable
	a.filterQuery = ""
	a.applyFilter()

	if len(a.filtered) != 1 {
		t.Errorf("expected 1 upgradable package, got %d", len(a.filtered))
	}
	if a.filtered[0].Name != "git" {
		t.Errorf("expected 'git', got '%s'", a.filtered[0].Name)
	}
}

func TestApplyFilterFuzzySearch(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true},
		{Name: "curl", Installed: false},
		{Name: "htop", Installed: true},
	}
	a.activeTab = tabAll
	a.filterQuery = "vim"
	a.applyFilter()

	if len(a.filtered) == 0 {
		t.Error("expected at least 1 result for 'vim'")
	}
	if a.filtered[0].Name != "vim" {
		t.Errorf("expected 'vim' as top result, got '%s'", a.filtered[0].Name)
	}
}

func TestApplyFilterResetsSelection(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.selectedIdx = 2
	a.scrollOffset = 1
	a.applyFilter()

	if a.selectedIdx != 0 {
		t.Errorf("expected selectedIdx reset to 0, got %d", a.selectedIdx)
	}
	if a.scrollOffset != 0 {
		t.Errorf("expected scrollOffset reset to 0, got %d", a.scrollOffset)
	}
}

func TestListHeight(t *testing.T) {
	a := newTestApp()
	h := a.packageListHeight()
	if h < 5 {
		t.Errorf("listHeight should be at least 5, got %d", h)
	}
}

func TestDetailHeight(t *testing.T) {
	a := newTestApp()
	if a.packageDetailHeight() != 10 {
		t.Errorf("expected detailHeight=10, got %d", a.packageDetailHeight())
	}
}

func TestAdjustScroll(t *testing.T) {
	a := newTestApp()
	a.allPackages = make([]model.Package, 100)
	a.filtered = a.allPackages

	// Scroll down past viewport
	a.selectedIdx = 50
	a.scrollOffset = 0
	a.adjustPackageScroll()
	if a.scrollOffset == 0 {
		t.Error("scrollOffset should have been adjusted for selectedIdx=50")
	}

	// Scroll back up
	a.selectedIdx = 0
	a.adjustPackageScroll()
	if a.scrollOffset != 0 {
		t.Errorf("scrollOffset should be 0 when selectedIdx=0, got %d", a.scrollOffset)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	a := newTestApp()
	m, _ := a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	app := m.(App)
	if app.width != 200 || app.height != 50 {
		t.Errorf("expected 200x50, got %dx%d", app.width, app.height)
	}
}

func TestToggleSelection(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.selectedIdx = 0

	// Toggle select
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app := m.(App)
	if !app.selected["vim"] {
		t.Error("vim should be selected after space")
	}

	// Toggle deselect
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app = m.(App)
	if app.selected["vim"] {
		t.Error("vim should be deselected after second space")
	}
}

func TestSelectAll(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}

	// Select all
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app := m.(App)
	if len(app.selected) != 3 {
		t.Errorf("expected 3 selected, got %d", len(app.selected))
	}

	// Toggle again to deselect all
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = m.(App)
	if len(app.selected) != 0 {
		t.Errorf("expected 0 selected after toggle, got %d", len(app.selected))
	}
}

func TestNavigationDown(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.selectedIdx = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	app := m.(App)
	if app.selectedIdx != 1 {
		t.Errorf("expected selectedIdx=1 after j, got %d", app.selectedIdx)
	}
}

func TestNavigationUp(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.selectedIdx = 2

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	app := m.(App)
	if app.selectedIdx != 1 {
		t.Errorf("expected selectedIdx=1 after k, got %d", app.selectedIdx)
	}
}

func TestNavigationBounds(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim"}, {Name: "git"},
	}

	// Can't go above 0
	a.selectedIdx = 0
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	app := m.(App)
	if app.selectedIdx != 0 {
		t.Errorf("should stay at 0, got %d", app.selectedIdx)
	}

	// Can't go below len-1
	a.selectedIdx = 1
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	app = m.(App)
	if app.selectedIdx != 1 {
		t.Errorf("should stay at 1, got %d", app.selectedIdx)
	}
}

func TestTabSwitching(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true, Upgradable: true},
		{Name: "curl", Installed: false},
	}
	a.applyFilter()

	if a.activeTab != tabAll {
		t.Errorf("expected tabAll initially, got %d", a.activeTab)
	}

	// Press tab -> tabInstalled
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	app := m.(App)
	if app.activeTab != tabInstalled {
		t.Errorf("expected tabInstalled, got %d", app.activeTab)
	}

	// Press tab again -> tabUpgradable
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if app.activeTab != tabUpgradable {
		t.Errorf("expected tabUpgradable, got %d", app.activeTab)
	}

	// Press tab again -> back to tabAll
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if app.activeTab != tabAll {
		t.Errorf("expected tabAll, got %d", app.activeTab)
	}
}

func TestTransactionViewToggle(t *testing.T) {
	a := newTestApp()

	// Open transaction view
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	app := m.(App)
	if !app.transactionView {
		t.Error("expected transactionView=true after 't'")
	}

	// Close transaction view with esc
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.transactionView {
		t.Error("expected transactionView=false after esc")
	}
}

func TestSearchMode(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.applyFilter()

	// Enter search mode
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app := m.(App)
	if !app.searching {
		t.Error("expected searching=true after '/'")
	}

	// Cancel search with esc
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.searching {
		t.Error("expected searching=false after esc")
	}
}

func TestHelpToggle(t *testing.T) {
	a := newTestApp()
	if a.help.ShowAll {
		t.Error("help should start collapsed")
	}

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app := m.(App)
	if !app.help.ShowAll {
		t.Error("expected help.ShowAll=true after 'h'")
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app = m.(App)
	if app.help.ShowAll {
		t.Error("expected help.ShowAll=false after second 'h'")
	}
}

func TestViewNotEmpty(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true, Version: "8.2"},
	}
	a.applyFilter()

	v := a.View()
	if v == "" {
		t.Error("View should not be empty")
	}
}

func TestViewLoadingState(t *testing.T) {
	a := newTestApp()
	a.width = 0

	v := a.View()
	if v != fmt.Sprintf("Updating and loading packages %s", a.spinner.View()) {
		t.Errorf("expected 'Updating and loading packages ...' when width=0, got %q", v)
	}
}

func TestAllPackagesMsg(t *testing.T) {
	a := newTestApp()

	msg := allPackagesMsg{
		allNames:   []string{"vim", "git", "curl", "htop"},
		installed:  []model.Package{{Name: "vim", Installed: true, Version: "8.2"}},
		upgradable: []model.Package{{Name: "vim", Installed: true, Upgradable: true, NewVersion: "9.0"}},
		err:        nil,
	}

	m, _ := a.Update(msg)
	app := m.(App)

	if app.loading {
		t.Error("loading should be false after allPackagesMsg")
	}
	if len(app.allPackages) != 4 {
		t.Errorf("expected 4 packages, got %d", len(app.allPackages))
	}
	if len(app.upgradableMap) != 1 {
		t.Errorf("expected 1 upgradable, got %d", len(app.upgradableMap))
	}
	if !app.allNamesLoaded {
		t.Error("allNamesLoaded should be true after allPackagesMsg")
	}
	if app.installedCount != 1 {
		t.Errorf("expected installedCount=1, got %d", app.installedCount)
	}
}

func TestInitialLoadMsg(t *testing.T) {
	a := newTestApp()

	msg := initialLoadMsg{
		installed:  []model.Package{{Name: "vim", Installed: true, Version: "8.2"}},
		upgradable: []model.Package{{Name: "vim", Installed: true, Upgradable: true, NewVersion: "9.0"}},
		err:        nil,
	}

	m, _ := a.Update(msg)
	app := m.(App)

	if app.loading {
		t.Error("loading should be false after initialLoadMsg")
	}
	if len(app.allPackages) != 1 {
		t.Errorf("expected 1 package (installed only), got %d", len(app.allPackages))
	}
	if app.installedCount != 1 {
		t.Errorf("expected installedCount=1, got %d", app.installedCount)
	}
	if app.allNamesLoaded {
		t.Error("allNamesLoaded should be false after initialLoadMsg")
	}
	if len(app.upgradableMap) != 1 {
		t.Errorf("expected 1 upgradable, got %d", len(app.upgradableMap))
	}
}

func TestInitialLoadMsgError(t *testing.T) {
	a := newTestApp()

	msg := initialLoadMsg{err: fmt.Errorf("test error")}

	m, _ := a.Update(msg)
	app := m.(App)

	if app.loading {
		t.Error("loading should be false after error")
	}
	if app.status == "" {
		t.Error("status should contain error message")
	}
}

func TestAllPackagesMsgError(t *testing.T) {
	a := newTestApp()

	msg := allPackagesMsg{
		err: fmt.Errorf("test error"),
	}

	m, _ := a.Update(msg)
	app := m.(App)

	if app.loading {
		t.Error("loading should be false after error")
	}
	if app.status == "" {
		t.Error("status should contain error message")
	}
}

func TestExecFinishedMsg(t *testing.T) {
	a := newTestApp()
	a.pendingExecOp = "install"
	a.pendingExecPkgs = []string{"vim"}
	a.pendingExecCount = 1
	a.loading = true

	msg := execFinishedMsg{op: "install", name: "vim", err: nil}
	m, _ := a.Update(msg)
	app := m.(App)

	if app.pendingExecCount != 0 {
		t.Errorf("pendingExecCount should be 0, got %d", app.pendingExecCount)
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim", Installed: true},
	}
	a.selectedIdx = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app := m.(App)

	if app.loading {
		t.Error("should not be loading when package already installed")
	}
	if app.status == "" {
		t.Error("should show already installed message")
	}
}

func TestRemoveNotInstalled(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim", Installed: false},
	}
	a.selectedIdx = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	app := m.(App)

	if app.loading {
		t.Error("should not be loading when package not installed")
	}
}

func TestUpgradeNotUpgradable(t *testing.T) {
	a := newTestApp()
	a.filtered = []model.Package{
		{Name: "vim", Installed: true, Upgradable: false},
	}
	a.selectedIdx = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	app := m.(App)

	if app.loading {
		t.Error("should not be loading when package not upgradable")
	}
}

func TestFetchViewToggle(t *testing.T) {
	a := newTestApp()

	// Open fetch view
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app := m.(App)
	if !app.fetchView {
		t.Error("expected fetchView=true after 'f'")
	}
}

func TestAdjustFetchScroll(t *testing.T) {
	a := newTestApp()
	a.fetchIdx = 50
	a.fetchOffset = 0
	a.adjustMirrorScroll()
	if a.fetchOffset == 0 {
		t.Error("fetchOffset should adjust when fetchIdx is past viewport")
	}
}

func TestAdjustTransactionScroll(t *testing.T) {
	a := newTestApp()
	a.transactionIdx = 50
	a.transactionOffset = 0
	a.adjustTransactionScroll()
	if a.transactionOffset == 0 {
		t.Error("transactionOffset should adjust when transactionIdx is past viewport")
	}
}

func TestTabDefsOrder(t *testing.T) {
	if len(tabDefs) != 3 {
		t.Fatalf("expected 3 tab definitions, got %d", len(tabDefs))
	}
	expected := []struct {
		kind tabKind
		name string
	}{
		{tabAll, "All"},
		{tabInstalled, "Installed"},
		{tabUpgradable, "Upgradable"},
	}
	for i, e := range expected {
		if tabDefs[i].kind != e.kind {
			t.Errorf("tabDefs[%d].kind = %d, want %d", i, tabDefs[i].kind, e.kind)
		}
		if tabDefs[i].name != e.name {
			t.Errorf("tabDefs[%d].name = %q, want %q", i, tabDefs[i].name, e.name)
		}
		if tabDefs[i].label == "" {
			t.Errorf("tabDefs[%d].label should not be empty", i)
		}
	}
}

func TestTabStyleActive(t *testing.T) {
	a := newTestApp()
	a.activeTab = tabAll

	got := a.tabStyle(tabDefs[0]).Render("X")
	want := ui.TabActiveStyle.Render("X")
	if got != want {
		t.Errorf("expected TabActiveStyle for the active tab, got %q vs %q", got, want)
	}
}

func TestTabStyleInactive(t *testing.T) {
	a := newTestApp()
	a.activeTab = tabAll

	got := a.tabStyle(tabDefs[1]).Render("X")
	want := ui.TabInactiveStyle.Render("X")
	if got != want {
		t.Errorf("expected TabInactiveStyle for an inactive tab, got %q vs %q", got, want)
	}
}

func TestTabStyleNotify(t *testing.T) {
	a := newTestApp()
	a.activeTab = tabAll
	a.upgradableMap = map[string]model.Package{"vim": {Name: "vim"}}

	got := a.tabStyle(tabDefs[2]).Render("X")
	want := ui.TabNotifyStyle.Render("X")
	if got != want {
		t.Errorf("expected TabNotifyStyle for upgradable tab when upgradable packages exist, got %q vs %q", got, want)
	}
}

func TestTabStyleUpgradableActiveNoNotify(t *testing.T) {
	a := newTestApp()
	a.activeTab = tabUpgradable
	a.upgradableMap = map[string]model.Package{"vim": {Name: "vim"}}

	got := a.tabStyle(tabDefs[2]).Render("X")
	want := ui.TabActiveStyle.Render("X")
	if got != want {
		t.Errorf("expected TabActiveStyle for active upgradable tab, got %q vs %q", got, want)
	}
}

func TestActivateTabSetsStatus(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true},
	}
	a.activeTab = tabInstalled
	a.activateTab()

	if !strings.Contains(a.status, "2 packages") {
		t.Errorf("expected status to mention package count, got %q", a.status)
	}
	if !strings.Contains(a.status, "Installed") {
		t.Errorf("expected status to mention tab name, got %q", a.status)
	}
}

func TestActivateTabResetsSelection(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim"}, {Name: "git"}, {Name: "curl"},
	}
	a.selectedIdx = 2
	a.scrollOffset = 1
	a.activeTab = tabAll
	a.activateTab()

	if a.selectedIdx != 0 {
		t.Errorf("expected selectedIdx=0 after activateTab, got %d", a.selectedIdx)
	}
	if a.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0 after activateTab, got %d", a.scrollOffset)
	}
}

func TestRenderTabBarContainsAllLabels(t *testing.T) {
	a := newTestApp()
	bar := a.renderTabBar()

	for _, td := range tabDefs {
		if !strings.Contains(bar, strings.TrimSpace(td.label)) {
			t.Errorf("renderTabBar missing label %q", td.label)
		}
	}
}

func TestSwitchTabBackward(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{{Name: "vim"}}

	if a.activeTab != tabAll {
		t.Fatal("expected initial tab to be tabAll")
	}

	m, _, handled := a.switchTab(tea.KeyMsg{Type: tea.KeyShiftTab})
	if !handled {
		t.Fatal("expected switchTab to handle shift+tab")
	}
	app := m.(App)
	if app.activeTab != tabUpgradable {
		t.Errorf("expected tabUpgradable after shift+tab from tabAll, got %d", app.activeTab)
	}

	m, _, _ = app.switchTab(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = m.(App)
	if app.activeTab != tabInstalled {
		t.Errorf("expected tabInstalled after shift+tab from tabUpgradable, got %d", app.activeTab)
	}
}

func TestSearchBarYPositive(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{
		{Name: "vim", Installed: true},
	}
	a.applyFilter()
	y := a.searchBarY()
	if y <= 0 || y >= a.height {
		t.Errorf("searchBarY=%d should be between 1 and %d", y, a.height-1)
	}
}

func TestUpgradeAllNoUpgradable(t *testing.T) {
	a := newTestApp()
	a.upgradableMap = map[string]model.Package{}

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	app := m.(App)

	if app.loading {
		t.Error("should not be loading when no upgradable packages")
	}
	if !strings.Contains(app.status, "No upgradable") {
		t.Errorf("expected 'No upgradable' in status, got %q", app.status)
	}
}

func TestUpgradeAllSetsState(t *testing.T) {
	a := newTestApp()
	a.upgradableMap = map[string]model.Package{
		"vim": {Name: "vim", Upgradable: true},
		"git": {Name: "git", Upgradable: true},
	}

	m, _ := a.upgradeAllPackages()
	app := m.(App)

	if !app.loading {
		t.Error("should be loading after upgradeAll")
	}
	if app.pendingExecOp != "upgrade-all" {
		t.Errorf("expected pendingExecOp='upgrade-all', got %q", app.pendingExecOp)
	}
	if len(app.pendingExecPkgs) != 2 {
		t.Errorf("expected 2 pending packages, got %d", len(app.pendingExecPkgs))
	}
	if !strings.Contains(app.status, "2 packages") {
		t.Errorf("expected status to mention 2 packages, got %q", app.status)
	}
}

func TestOnTabClickSameTab(t *testing.T) {
	a := newTestApp()
	a.allPackages = []model.Package{{Name: "vim"}}
	a.activeTab = tabAll

	m, cmd := a.onTabClick(0)
	app := m.(App)

	if app.activeTab != tabAll {
		t.Error("expected tab to stay on tabAll when clicking active tab")
	}
	if cmd != nil {
		t.Error("expected nil cmd when clicking already-active tab")
	}
}
