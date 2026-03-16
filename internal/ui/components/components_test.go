package components

import (
	"strings"
	"testing"

	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/history"
	"github.com/mexirica/aptui/internal/model"
)

func TestRenderPackageListEmpty(t *testing.T) {
	result := RenderPackageList(nil, 0, 0, 10, 120, nil)
	if !strings.Contains(result, "No packages found") {
		t.Error("empty list should show 'no packages' message")
	}
}

func TestRenderPackageListWithPackages(t *testing.T) {
	pkgs := []model.Package{
		{Name: "vim", Version: "8.2", Installed: true, Size: "9.8 MB"},
		{Name: "git", Version: "2.34", Installed: true, Upgradable: true, NewVersion: "2.40", Size: "3.2 MB"},
		{Name: "curl", Installed: false},
	}

	result := RenderPackageList(pkgs, 0, 0, 10, 120, nil)
	if result == "" {
		t.Error("rendered list should not be empty")
	}
	if !strings.Contains(result, "Name") {
		t.Error("should contain Name header")
	}
	if !strings.Contains(result, "Version") {
		t.Error("should contain Version header")
	}
}

func TestRenderPackageListSelectedIndex(t *testing.T) {
	pkgs := []model.Package{
		{Name: "vim", Version: "8.2", Installed: true},
		{Name: "git", Version: "2.34", Installed: true},
	}

	result := RenderPackageList(pkgs, 1, 0, 10, 120, nil)
	if !strings.Contains(result, "\u258c") {
		t.Error("selected item should show cursor")
	}
}

func TestRenderPackageListWithSelection(t *testing.T) {
	pkgs := []model.Package{
		{Name: "vim", Installed: true},
		{Name: "git", Installed: true},
	}

	selected := map[string]bool{"vim": true}
	result := RenderPackageList(pkgs, 0, 0, 10, 120, selected)
	if !strings.Contains(result, "[x]") {
		t.Error("selected package should show [x]")
	}
	if !strings.Contains(result, "[ ]") {
		t.Error("unselected package should show [ ]")
	}
}

func TestRenderPackageListOffset(t *testing.T) {
	pkgs := make([]model.Package, 50)
	for i := range pkgs {
		pkgs[i] = model.Package{Name: "pkg-" + string(rune('a'+i%26))}
	}

	result := RenderPackageList(pkgs, 10, 5, 10, 120, nil)
	lines := strings.Split(result, "\n")
	if len(lines) < 10 {
		t.Errorf("expected at least 10 lines, got %d", len(lines))
	}
}

func TestRenderPackageDetailEmpty(t *testing.T) {
	result := RenderPackageDetail("", 120, 10, 1)
	if !strings.Contains(result, "No package selected") {
		t.Error("empty detail should show placeholder message")
	}
}

func TestRenderPackageDetailWithInfo(t *testing.T) {
	info := "Package: vim\nVersion: 2:8.2.4919-1ubuntu1\nStatus: Installed\nSection: editors\nInstalled-Size: 3984\nMaintainer: Debian Vim Maintainers\nArchitecture: amd64\nDepends: vim-common\nDescription: Vi IMproved\nHomepage: https://www.vim.org"

	result := RenderPackageDetail(info, 120, 10, 1)
	if result == "" {
		t.Error("detail should not be empty")
	}
	if !strings.Contains(result, "vim") {
		t.Error("should contain package name")
	}
}

func TestRenderPackageDetailMaxLines(t *testing.T) {
	info := "Package: vim\nVersion: 1.0\nSection: editors\nInstalled-Size: 100\nMaintainer: Test\nArchitecture: amd64\nDepends: libc6\nDescription: Test package\nHomepage: https://example.com\nStatus: Installed"

	result := RenderPackageDetail(info, 120, 3, 1)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) > 3 {
		t.Errorf("expected at most 3 lines, got %d", len(lines))
	}
}

func TestRenderQueryPrompt(t *testing.T) {
	result := RenderQueryPrompt("vim", false)
	if !strings.Contains(result, "vim") {
		t.Error("query prompt should contain query")
	}
	if !strings.Contains(result, "\u276f") {
		t.Error("query prompt should contain prompt char")
	}
}

func TestRenderQueryPromptFocused(t *testing.T) {
	result := RenderQueryPrompt("test", true)
	if !strings.Contains(result, "\u2588") {
		t.Error("focused query prompt should show cursor block")
	}
}

func TestRenderQueryPromptEmpty(t *testing.T) {
	result := RenderQueryPrompt("", false)
	if result == "" {
		t.Error("empty query prompt should still render")
	}
}

func TestRenderStatusBar(t *testing.T) {
	result := RenderStatusBar("test status", 120)
	if !strings.Contains(result, "test status") {
		t.Error("status bar should contain the status text")
	}
}

func TestRenderStatusBarEmpty(t *testing.T) {
	result := RenderStatusBar("", 80)
	// Status bar renders with style even if content is empty
	_ = result
}

func TestRenderTransactionListEmpty(t *testing.T) {
	result := RenderTransactionList(nil, 0, 0, 10, 120)
	if !strings.Contains(result, "No transaction") {
		t.Error("empty transaction list should show 'No transaction' message")
	}
}

func TestRenderTransactionListWithItems(t *testing.T) {
	items := []history.Transaction{
		{ID: 1, Operation: history.OpInstall, Packages: []string{"vim"}, Success: true},
		{ID: 2, Operation: history.OpRemove, Packages: []string{"nano"}, Success: false},
	}

	result := RenderTransactionList(items, 0, 0, 10, 120)
	if result == "" {
		t.Error("rendered transaction list should not be empty")
	}
	if !strings.Contains(result, "ID") {
		t.Error("should contain ID header")
	}
}

func TestRenderTransactionDetail(t *testing.T) {
	tx := history.Transaction{
		ID:        1,
		Operation: history.OpInstall,
		Packages:  []string{"vim", "git", "curl"},
		Success:   true,
	}

	result := RenderTransactionDetail(tx, nil, 120, 10)
	if result == "" {
		t.Error("transaction detail should not be empty")
	}
	if !strings.Contains(result, "#1") {
		t.Error("should contain transaction ID")
	}
	if !strings.Contains(result, "install") {
		t.Error("should contain operation name")
	}
}

func TestRenderFetchHeader(t *testing.T) {
	d := fetch.Distro{Name: "Ubuntu 24.04", Codename: "noble"}
	result := RenderFetchHeader(d)
	if !strings.Contains(result, "Ubuntu 24.04") {
		t.Error("should contain distro name")
	}
	if !strings.Contains(result, "noble") {
		t.Error("should contain codename")
	}
}

func TestRenderFetchProgress(t *testing.T) {
	result := RenderFetchProgress(25, 50)
	if !strings.Contains(result, "50%") {
		t.Error("should show 50% progress")
	}
	if !strings.Contains(result, "25/50") {
		t.Error("should show 25/50 count")
	}
}

func TestRenderFetchProgressZero(t *testing.T) {
	result := RenderFetchProgress(0, 0)
	if !strings.Contains(result, "0%") {
		t.Error("should show 0% for empty totals")
	}
}

func TestRenderMirrorListEmpty(t *testing.T) {
	result := RenderMirrorList(nil, 0, 0, 10, 120, nil)
	if !strings.Contains(result, "No mirrors") {
		t.Error("empty mirror list should show message")
	}
}

func TestRenderMirrorListWithMirrors(t *testing.T) {
	mirrors := []fetch.Mirror{
		{URL: "http://archive.ubuntu.com/ubuntu/", Status: "ok", Latency: 50e6},
		{URL: "http://br.archive.ubuntu.com/ubuntu/", Status: "error"},
	}
	selected := map[int]bool{0: true}

	result := RenderMirrorList(mirrors, 0, 0, 10, 120, selected)
	if result == "" {
		t.Error("mirror list should not be empty")
	}
}

func TestRenderFetchFooterHelp(t *testing.T) {
	result := RenderFetchFooterHelp()
	if !strings.Contains(result, "space") {
		t.Error("should contain space key hint")
	}
	if !strings.Contains(result, "enter") {
		t.Error("should contain enter key hint")
	}
}
