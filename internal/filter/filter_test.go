package filter

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	f := Parse("")
	if !f.IsEmpty() {
		t.Error("empty query should produce empty filter")
	}
}

func TestParseSection(t *testing.T) {
	f := Parse("section:utils")
	if f.Section != "utils" {
		t.Errorf("expected section 'utils', got '%s'", f.Section)
	}
}

func TestParseSectionAlias(t *testing.T) {
	f := Parse("sec:libs")
	if f.Section != "libs" {
		t.Errorf("expected section 'libs', got '%s'", f.Section)
	}
}

func TestParseArch(t *testing.T) {
	f := Parse("arch:amd64")
	if f.Architecture != "amd64" {
		t.Errorf("expected arch 'amd64', got '%s'", f.Architecture)
	}
}

func TestParseSizeGt(t *testing.T) {
	f := Parse("size>10MB")
	if f.Size == nil {
		t.Fatal("expected size filter")
	}
	if f.Size.Op != SizeGt {
		t.Errorf("expected SizeGt, got %d", f.Size.Op)
	}
	if f.Size.KB != 10*1024 {
		t.Errorf("expected %d kB, got %d", 10*1024, f.Size.KB)
	}
}

func TestParseSizeLt(t *testing.T) {
	f := Parse("size<5MB")
	if f.Size == nil {
		t.Fatal("expected size filter")
	}
	if f.Size.Op != SizeLt {
		t.Errorf("expected SizeLt, got %d", f.Size.Op)
	}
	if f.Size.KB != 5*1024 {
		t.Errorf("expected %d kB, got %d", 5*1024, f.Size.KB)
	}
}

func TestParseSizeGe(t *testing.T) {
	f := Parse("size>=100kB")
	if f.Size == nil {
		t.Fatal("expected size filter")
	}
	if f.Size.Op != SizeGe {
		t.Errorf("expected SizeGe, got %d", f.Size.Op)
	}
	if f.Size.KB != 100 {
		t.Errorf("expected 100 kB, got %d", f.Size.KB)
	}
}

func TestParseSizeColonVariant(t *testing.T) {
	f := Parse("size:>2GB")
	if f.Size == nil {
		t.Fatal("expected size filter")
	}
	if f.Size.Op != SizeGt {
		t.Errorf("expected SizeGt, got %d", f.Size.Op)
	}
	if f.Size.KB != 2*1024*1024 {
		t.Errorf("expected %d kB, got %d", 2*1024*1024, f.Size.KB)
	}
}

func TestParseInstalled(t *testing.T) {
	f := Parse("installed")
	if f.Installed == nil || !*f.Installed {
		t.Error("expected installed=true")
	}
}

func TestParseNotInstalled(t *testing.T) {
	f := Parse("!installed")
	if f.Installed == nil || *f.Installed {
		t.Error("expected installed=false")
	}
}

func TestParseUpgradable(t *testing.T) {
	f := Parse("upgradable")
	if f.Upgradable == nil || !*f.Upgradable {
		t.Error("expected upgradable=true")
	}
}

func TestParseName(t *testing.T) {
	f := Parse("name:vim")
	if f.Name != "vim" {
		t.Errorf("expected name 'vim', got '%s'", f.Name)
	}
}

func TestParseVersion(t *testing.T) {
	f := Parse("ver:2.0")
	if f.Version != "2.0" {
		t.Errorf("expected version '2.0', got '%s'", f.Version)
	}
}

func TestParseDescription(t *testing.T) {
	f := Parse("desc:editor")
	if f.Description != "editor" {
		t.Errorf("expected description 'editor', got '%s'", f.Description)
	}
}

func TestParseMultiple(t *testing.T) {
	f := Parse("section:utils arch:amd64 size>10MB installed")
	if f.Section != "utils" {
		t.Errorf("section: expected 'utils', got '%s'", f.Section)
	}
	if f.Architecture != "amd64" {
		t.Errorf("arch: expected 'amd64', got '%s'", f.Architecture)
	}
	if f.Size == nil || f.Size.Op != SizeGt || f.Size.KB != 10*1024 {
		t.Error("size filter mismatch")
	}
	if f.Installed == nil || !*f.Installed {
		t.Error("expected installed=true")
	}
}

func TestMatchSection(t *testing.T) {
	f := Parse("section:utils")
	p := PackageData{Section: "utils", Name: "test"}
	if !f.Match(p) {
		t.Error("should match package in utils section")
	}
	p2 := PackageData{Section: "libs", Name: "test"}
	if f.Match(p2) {
		t.Error("should not match package in libs section")
	}
}

func TestMatchSectionContains(t *testing.T) {
	f := Parse("section:util")
	p := PackageData{Section: "utils", Name: "test"}
	if !f.Match(p) {
		t.Error("section filter should use contains matching")
	}
}

func TestMatchArch(t *testing.T) {
	f := Parse("arch:amd64")
	p := PackageData{Architecture: "amd64", Name: "test"}
	if !f.Match(p) {
		t.Error("should match amd64")
	}
	p2 := PackageData{Architecture: "arm64", Name: "test"}
	if f.Match(p2) {
		t.Error("should not match arm64")
	}
}

func TestMatchSize(t *testing.T) {
	f := Parse("size>5MB")
	p := PackageData{Size: "10.0 MB", Name: "test"}
	if !f.Match(p) {
		t.Error("10MB should be > 5MB")
	}
	p2 := PackageData{Size: "3.0 MB", Name: "test"}
	if f.Match(p2) {
		t.Error("3MB should not be > 5MB")
	}
}

func TestMatchSizeUnknown(t *testing.T) {
	f := Parse("size>5MB")
	p := PackageData{Size: "-", Name: "test"}
	if f.Match(p) {
		t.Error("unknown size should not match size filter")
	}
}

func TestMatchInstalled(t *testing.T) {
	f := Parse("installed")
	if !f.Match(PackageData{Installed: true, Name: "a"}) {
		t.Error("should match installed package")
	}
	if f.Match(PackageData{Installed: false, Name: "b"}) {
		t.Error("should not match non-installed package")
	}
}

func TestMatchCombined(t *testing.T) {
	f := Parse("section:editors arch:amd64 installed")
	p := PackageData{
		Name:         "vim",
		Section:      "editors",
		Architecture: "amd64",
		Installed:    true,
	}
	if !f.Match(p) {
		t.Error("should match all criteria")
	}
	p2 := PackageData{
		Name:         "vim",
		Section:      "editors",
		Architecture: "arm64",
		Installed:    true,
	}
	if f.Match(p2) {
		t.Error("should not match wrong architecture")
	}
}

func TestParseSizeToKB(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1.5 MB", 1536},
		{"324 kB", 324},
		{"2.1 GB", 2202009},
		{"-", 0},
		{"", 0},
		{"10.0 MB", 10240},
	}
	for _, tt := range tests {
		got := ParseSizeToKB(tt.input)
		if got != tt.expected {
			t.Errorf("ParseSizeToKB(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestDescribe(t *testing.T) {
	f := Parse("section:utils arch:amd64 installed")
	desc := f.Describe()
	if desc == "" {
		t.Error("describe should not be empty")
	}
}

func TestFilterIsEmpty(t *testing.T) {
	f := Filter{}
	if !f.IsEmpty() {
		t.Error("zero-value filter should be empty")
	}
	f2 := Parse("installed")
	if f2.IsEmpty() {
		t.Error("filter with installed flag should not be empty")
	}
}

func TestParseOrderByNameAsc(t *testing.T) {
	f := Parse("order:name")
	if f.OrderBy != SortName {
		t.Errorf("expected SortName, got %d", f.OrderBy)
	}
	if f.OrderDesc {
		t.Error("expected ascending order by default")
	}
}

func TestParseOrderByNameDesc(t *testing.T) {
	f := Parse("order:name:desc")
	if f.OrderBy != SortName {
		t.Errorf("expected SortName, got %d", f.OrderBy)
	}
	if !f.OrderDesc {
		t.Error("expected descending order")
	}
}

func TestParseOrderBySizeAsc(t *testing.T) {
	f := Parse("order:size:asc")
	if f.OrderBy != SortSize {
		t.Errorf("expected SortSize, got %d", f.OrderBy)
	}
	if f.OrderDesc {
		t.Error("expected ascending order")
	}
}

func TestParseOrderByVersionDesc(t *testing.T) {
	f := Parse("order:ver:desc")
	if f.OrderBy != SortVersion {
		t.Errorf("expected SortVersion, got %d", f.OrderBy)
	}
	if !f.OrderDesc {
		t.Error("expected descending order")
	}
}

func TestParseOrderCombinedWithFilter(t *testing.T) {
	f := Parse("installed order:size:desc")
	if f.Installed == nil || !*f.Installed {
		t.Error("expected installed=true")
	}
	if f.OrderBy != SortSize {
		t.Errorf("expected SortSize, got %d", f.OrderBy)
	}
	if !f.OrderDesc {
		t.Error("expected descending order")
	}
}

func TestParseOrderIsNotEmpty(t *testing.T) {
	f := Parse("order:name")
	if f.IsEmpty() {
		t.Error("filter with order should not be empty")
	}
}

func TestDescribeWithOrder(t *testing.T) {
	f := Parse("order:name:desc")
	desc := f.Describe()
	if desc != "order:name:desc" {
		t.Errorf("expected 'order:name:desc', got '%s'", desc)
	}
}

func TestSortByNameAsc(t *testing.T) {
	pkgs := []PackageData{
		{Name: "zsh"},
		{Name: "apt"},
		{Name: "nano"},
	}
	f := Filter{OrderBy: SortName}
	Sort(pkgs, f)
	if pkgs[0].Name != "apt" || pkgs[1].Name != "nano" || pkgs[2].Name != "zsh" {
		t.Errorf("unexpected order: %s, %s, %s", pkgs[0].Name, pkgs[1].Name, pkgs[2].Name)
	}
}

func TestSortByNameDesc(t *testing.T) {
	pkgs := []PackageData{
		{Name: "apt"},
		{Name: "zsh"},
		{Name: "nano"},
	}
	f := Filter{OrderBy: SortName, OrderDesc: true}
	Sort(pkgs, f)
	if pkgs[0].Name != "zsh" || pkgs[1].Name != "nano" || pkgs[2].Name != "apt" {
		t.Errorf("unexpected order: %s, %s, %s", pkgs[0].Name, pkgs[1].Name, pkgs[2].Name)
	}
}

func TestSortBySizeAsc(t *testing.T) {
	pkgs := []PackageData{
		{Name: "big", Size: "10.0 MB"},
		{Name: "small", Size: "100 kB"},
		{Name: "med", Size: "1.0 MB"},
	}
	f := Filter{OrderBy: SortSize}
	Sort(pkgs, f)
	if pkgs[0].Name != "small" || pkgs[1].Name != "med" || pkgs[2].Name != "big" {
		t.Errorf("unexpected order: %s, %s, %s", pkgs[0].Name, pkgs[1].Name, pkgs[2].Name)
	}
}

func TestSortBySizeDesc(t *testing.T) {
	pkgs := []PackageData{
		{Name: "big", Size: "10.0 MB"},
		{Name: "small", Size: "100 kB"},
		{Name: "med", Size: "1.0 MB"},
	}
	f := Filter{OrderBy: SortSize, OrderDesc: true}
	Sort(pkgs, f)
	if pkgs[0].Name != "big" || pkgs[1].Name != "med" || pkgs[2].Name != "small" {
		t.Errorf("unexpected order: %s, %s, %s", pkgs[0].Name, pkgs[1].Name, pkgs[2].Name)
	}
}

func TestSortNoneDoesNothing(t *testing.T) {
	pkgs := []PackageData{
		{Name: "zsh"},
		{Name: "apt"},
		{Name: "nano"},
	}
	f := Filter{OrderBy: SortNone}
	Sort(pkgs, f)
	if pkgs[0].Name != "zsh" || pkgs[1].Name != "apt" || pkgs[2].Name != "nano" {
		t.Errorf("SortNone should not change order: %s, %s, %s", pkgs[0].Name, pkgs[1].Name, pkgs[2].Name)
	}
}

func TestParseFreeText(t *testing.T) {
	f := Parse("vim")
	if f.FreeText != "vim" {
		t.Errorf("expected FreeText 'vim', got '%s'", f.FreeText)
	}
}

func TestParseFreeTextWithFilter(t *testing.T) {
	f := Parse("section:utils vim editor")
	if f.Section != "utils" {
		t.Errorf("expected section 'utils', got '%s'", f.Section)
	}
	if f.FreeText != "vim editor" {
		t.Errorf("expected FreeText 'vim editor', got '%s'", f.FreeText)
	}
}

func TestParseFreeTextEmpty(t *testing.T) {
	f := Parse("section:utils installed")
	if f.FreeText != "" {
		t.Errorf("expected empty FreeText, got '%s'", f.FreeText)
	}
}

func TestDescribeIncludesFreeText(t *testing.T) {
	f := Parse("section:utils vim")
	desc := f.Describe()
	if desc != "sec:utils vim" {
		t.Errorf("expected 'sec:utils vim', got '%s'", desc)
	}
}

func TestIsEmptyWithFreeText(t *testing.T) {
	f := Parse("vim")
	if f.IsEmpty() {
		t.Error("filter with free text should not be empty")
	}
}
