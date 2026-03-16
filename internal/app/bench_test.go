package app

import (
	"testing"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/filter"
	"github.com/mexirica/aptui/internal/model"
)

// buildBenchApp creates an App with realistic data for benchmarking.
// Uses LoadAllAvailableInfo to populate packages with full metadata.
func buildBenchApp(b *testing.B) App {
	b.Helper()

	bulkInfo := apt.LoadAllAvailableInfo()
	if len(bulkInfo) == 0 {
		b.Fatal("LoadAllAvailableInfo returned empty - are apt lists available?")
	}

	installed, err := apt.ListInstalled()
	if err != nil {
		b.Fatal(err)
	}

	a := New()
	a.width = 120
	a.height = 40
	a.loading = false

	seen := make(map[string]bool, len(installed)+len(bulkInfo))
	a.infoCache = make(map[string]apt.PackageInfo, len(bulkInfo))
	for name, info := range bulkInfo {
		a.infoCache[name] = info
	}

	all := make([]model.Package, 0, len(installed)+len(bulkInfo))
	for _, p := range installed {
		if info, ok := bulkInfo[p.Name]; ok {
			if p.Size == "" || p.Size == "-" {
				p.Size = info.Size
			}
			if p.Section == "" {
				p.Section = info.Section
			}
			if p.Architecture == "" {
				p.Architecture = info.Architecture
			}
		}
		all = append(all, p)
		seen[p.Name] = true
	}
	for name, info := range bulkInfo {
		if !seen[name] {
			all = append(all, model.Package{
				Name:         name,
				NewVersion:   info.Version,
				Size:         info.Size,
				Section:      info.Section,
				Architecture: info.Architecture,
			})
			seen[name] = true
		}
	}
	a.allPackages = all
	a.rebuildIndex()
	a.installedCount = len(installed)
	a.allNamesLoaded = true

	return a
}

// BenchmarkApplyFilterNoFilter benchmarks applyFilter with no filter active.
func BenchmarkApplyFilterNoFilter(b *testing.B) {
	a := buildBenchApp(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterName benchmarks a non-metadata filter (name contains).
func BenchmarkApplyFilterName(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "name:vim"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterSection benchmarks a metadata filter (section contains).
// This is the filter type that was slow before (required apt-cache show).
func BenchmarkApplyFilterSection(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "section:utils"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterArch benchmarks architecture filter.
func BenchmarkApplyFilterArch(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "arch:amd64"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterSize benchmarks size comparison filter.
func BenchmarkApplyFilterSize(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "size>10MB"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterCombined benchmarks a combined metadata + non-metadata filter.
func BenchmarkApplyFilterCombined(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "section:utils arch:amd64 size>1MB"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterInstalled benchmarks filtering installed packages.
func BenchmarkApplyFilterInstalled(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "installed"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkApplyFilterSortBySize benchmarks filter + sorting by size.
func BenchmarkApplyFilterSortBySize(b *testing.B) {
	a := buildBenchApp(b)
	a.filterQuery = "installed order:size:desc"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.applyFilter()
	}
}

// BenchmarkFilterParse benchmarks just the filter parsing.
func BenchmarkFilterParse(b *testing.B) {
	queries := []string{
		"section:utils",
		"arch:amd64 installed",
		"name:vim section:editors size>1MB order:name:asc",
		"installed !upgradable desc:editor",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Parse(queries[i%len(queries)])
	}
}

// BenchmarkSubmitSearchSection simulates the full submit search path
// for a metadata filter. Now uses the unified search/filter bar.
func BenchmarkSubmitSearchSection(b *testing.B) {
	a := buildBenchApp(b)
	a.searchInput.SetValue("section:utils")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.filterQuery = ""
		a.searchInput.SetValue("section:utils")
		a.submitSearch()
	}
}
