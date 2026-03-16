package apt

import (
	"testing"

	"github.com/mexirica/aptui/internal/model"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "-"},
		{"  ", "-"},
		{"0", "-"},
		{"500", "500 kB"},
		{"1024", "1.0 MB"},
		{"2048", "2.0 MB"},
		{"1048576", "1.0 GB"},
		{"2097152", "2.0 GB"},
		{"512", "512 kB"},
		{"1500", "1.5 MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.input)
		if got != tt.expected {
			t.Errorf("formatSize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseDpkgOutput(t *testing.T) {
	input := `vim	8.2.4919	9876	Vi IMproved - enhanced vi editor
curl	7.88.1	456	command line tool for transferring data
`
	pkgs := parseDpkgOutput(input, true)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	vim := pkgs[0]
	if vim.Name != "vim" {
		t.Errorf("expected name 'vim', got '%s'", vim.Name)
	}
	if vim.Version != "8.2.4919" {
		t.Errorf("expected version '8.2.4919', got '%s'", vim.Version)
	}
	if !vim.Installed {
		t.Error("expected installed=true")
	}
	if vim.Description != "Vi IMproved - enhanced vi editor" {
		t.Errorf("unexpected description: %s", vim.Description)
	}

	curl := pkgs[1]
	if curl.Name != "curl" {
		t.Errorf("expected name 'curl', got '%s'", curl.Name)
	}
}

func TestParseDpkgOutputSkipsEmptyLines(t *testing.T) {
	input := `
vim	8.2	100	editor

curl	7.88	50	transfer tool
`
	pkgs := parseDpkgOutput(input, false)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Installed || pkgs[1].Installed {
		t.Error("expected installed=false")
	}
}

func TestParseDpkgOutputSkipsContinuationLines(t *testing.T) {
	input := `vim	8.2	100	editor
 this is a continuation line
curl	7.88	50	tool`
	pkgs := parseDpkgOutput(input, true)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages (skipping continuation), got %d", len(pkgs))
	}
}

func TestParseDpkgOutputMinimalFields(t *testing.T) {
	input := `vim	8.2`
	pkgs := parseDpkgOutput(input, true)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "vim" || pkgs[0].Version != "8.2" {
		t.Errorf("unexpected: %+v", pkgs[0])
	}
	if pkgs[0].Size != "" {
		t.Errorf("expected empty size, got %s", pkgs[0].Size)
	}
}

func TestParseSearchOutput(t *testing.T) {
	// parseSearchOutput calls IsInstalled which requires dpkg-query.
	// We test just the parsing logic with a simple case.
	input := `vim - Vi IMproved
git - fast version control`

	// This will call IsInstalled which may fail on CI, but the parse logic itself should work.
	pkgs := parseSearchOutput(input)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "vim" {
		t.Errorf("expected name 'vim', got '%s'", pkgs[0].Name)
	}
	if pkgs[0].Description != "Vi IMproved" {
		t.Errorf("unexpected description: %s", pkgs[0].Description)
	}
}

func TestParseUpgradableOutput(t *testing.T) {
	input := `Listing... Done
vim/noble 2:9.1.0-1 amd64 [upgradable from: 2:8.2.4919-1]
curl/noble 8.5.0-1 amd64 [upgradable from: 7.88.1-1]`

	pkgs := parseUpgradableOutput(input)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	vim := pkgs[0]
	if vim.Name != "vim" {
		t.Errorf("expected name 'vim', got '%s'", vim.Name)
	}
	if !vim.Upgradable {
		t.Error("expected upgradable=true")
	}
	if !vim.Installed {
		t.Error("expected installed=true for upgradable")
	}
	if vim.NewVersion != "2:9.1.0-1" {
		t.Errorf("expected new version '2:9.1.0-1', got '%s'", vim.NewVersion)
	}
	if vim.Version != "2:8.2.4919-1" {
		t.Errorf("expected old version '2:8.2.4919-1', got '%s'", vim.Version)
	}
	if vim.SecurityUpdate {
		t.Error("expected SecurityUpdate=false for non-security repo")
	}
}

func TestParseUpgradableOutputSecurityUpdate(t *testing.T) {
	input := `Listing... Done
vim/noble-security 2:9.1.0-1 amd64 [upgradable from: 2:8.2.4919-1]
curl/noble 8.5.0-1 amd64 [upgradable from: 7.88.1-1]
libssl3/noble-security,noble-updates 3.0.14-1 amd64 [upgradable from: 3.0.13-1]
websecurity-tools/my-cybersecurity-ppa 1.2.0-1 amd64 [upgradable from: 1.1.0-1]`

	pkgs := parseUpgradableOutput(input)
	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}

	vim := pkgs[0]
	if !vim.SecurityUpdate {
		t.Error("expected SecurityUpdate=true for security repo")
	}

	curl := pkgs[1]
	if curl.SecurityUpdate {
		t.Error("expected SecurityUpdate=false for non-security repo")
	}

	libssl := pkgs[2]
	if !libssl.SecurityUpdate {
		t.Error("expected SecurityUpdate=true for comma-separated repo with security origin")
	}

	websecTools := pkgs[3]
	if websecTools.SecurityUpdate {
		t.Error("expected SecurityUpdate=false for PPA containing 'security' substring")
	}
}

func TestParseUpgradableOutputSkipsListing(t *testing.T) {
	input := `Listing... Done`
	pkgs := parseUpgradableOutput(input)
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseShowEntry(t *testing.T) {
	info := `Package: vim
Version: 2:8.2.4919-1ubuntu1
Installed-Size: 3984
Architecture: amd64
Depends: vim-common, libc6
Description: Vi IMproved - enhanced vi editor

Package: vim-tiny
Version: 2:8.2.4919-1ubuntu1
Installed-Size: 800
`
	pi := ParseShowEntry(info)
	if pi.Version != "2:8.2.4919-1ubuntu1" {
		t.Errorf("expected version, got '%s'", pi.Version)
	}
	if pi.Size == "" || pi.Size == "-" {
		t.Errorf("expected formatted size, got '%s'", pi.Size)
	}
}

func TestParseShowEntryEmpty(t *testing.T) {
	pi := ParseShowEntry("")
	if pi.Version != "" {
		t.Errorf("expected empty version for empty input, got '%s'", pi.Version)
	}
}

// TestPackageModelFields verifies Package struct fields work correctly.
func TestPackageModelFields(t *testing.T) {
	pkg := model.Package{
		Name:        "test-pkg",
		Version:     "1.0",
		Size:        "100 kB",
		Description: "A test package",
		Installed:   true,
		Upgradable:  true,
		NewVersion:  "2.0",
	}
	if pkg.Name != "test-pkg" {
		t.Errorf("unexpected name: %s", pkg.Name)
	}
	if !pkg.Installed || !pkg.Upgradable {
		t.Error("expected installed and upgradable")
	}
	if pkg.NewVersion != "2.0" {
		t.Errorf("expected new version 2.0, got %s", pkg.NewVersion)
	}
}

func TestParseDpkgOutputDeduplicatesMultiArch(t *testing.T) {
	input := `libc6	2.39-0ubuntu8	14000	GNU C Library	libs	amd64
libc6	2.39-0ubuntu8	7000	GNU C Library	libs	i386
vim	8.2.4919	9876	Vi IMproved	editors	amd64
`
	pkgs := parseDpkgOutput(input, true)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages (deduped), got %d", len(pkgs))
	}
	if pkgs[0].Name != "libc6" {
		t.Errorf("expected first package 'libc6', got '%s'", pkgs[0].Name)
	}
	if pkgs[0].Architecture != "amd64" {
		t.Errorf("expected architecture 'amd64', got '%s'", pkgs[0].Architecture)
	}
	if pkgs[1].Name != "vim" {
		t.Errorf("expected second package 'vim', got '%s'", pkgs[1].Name)
	}
}

func TestParseUpgradableOutputDeduplicatesMultiArch(t *testing.T) {
	input := `Listing... Done
libc6/noble 2.39-1 amd64 [upgradable from: 2.39-0]
libc6/noble 2.39-1 i386 [upgradable from: 2.39-0]
vim/noble 2:9.1.0-1 amd64 [upgradable from: 2:8.2.4919-1]`

	pkgs := parseUpgradableOutput(input)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages (deduped), got %d", len(pkgs))
	}
	if pkgs[0].Name != "libc6" {
		t.Errorf("expected 'libc6', got '%s'", pkgs[0].Name)
	}
	if pkgs[1].Name != "vim" {
		t.Errorf("expected 'vim', got '%s'", pkgs[1].Name)
	}
}

func TestParseSearchOutputDeduplicates(t *testing.T) {
	input := `libc6 - GNU C Library: Shared libraries
libc6 - GNU C Library: Shared libraries
vim - Vi IMproved - enhanced vi editor`

	pkgs := parseSearchOutput(input)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages (deduped), got %d", len(pkgs))
	}
	if pkgs[0].Name != "libc6" {
		t.Errorf("expected 'libc6', got '%s'", pkgs[0].Name)
	}
	if pkgs[1].Name != "vim" {
		t.Errorf("expected 'vim', got '%s'", pkgs[1].Name)
	}
}
