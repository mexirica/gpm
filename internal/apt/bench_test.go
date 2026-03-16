package apt

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// BenchmarkLoadAllAvailableInfo benchmarks the new approach: parsing
// /var/lib/apt/lists/*_Packages files to bulk-load all package metadata.
// This replaces both ListAllNames + BatchGetInfo from the old flow.
func BenchmarkLoadAllAvailableInfo(b *testing.B) {
	// Warm up to ensure files are in OS page cache (fair comparison)
	_ = LoadAllAvailableInfo()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := LoadAllAvailableInfo()
		if len(info) == 0 {
			b.Fatal("LoadAllAvailableInfo returned empty map")
		}
	}
}

// BenchmarkListAllNames benchmarks the old approach to get package names:
// spawning 'apt-cache pkgnames'.
func BenchmarkListAllNames(b *testing.B) {
	// Warm up
	_, _ = ListAllNames()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		names, err := ListAllNames()
		if err != nil {
			b.Fatal(err)
		}
		if len(names) == 0 {
			b.Fatal("no names returned")
		}
	}
}

// oldBatchGetInfo is a copy of the removed BatchGetInfo function,
// kept here purely for benchmarking the old approach.
func oldBatchGetInfo(names []string) map[string]PackageInfo {
	if len(names) == 0 {
		return nil
	}
	const chunkSize = 50
	const maxWorkers = 8

	type result struct {
		info map[string]PackageInfo
	}
	chunks := make([][]string, 0, len(names)/chunkSize+1)
	for i := 0; i < len(names); i += chunkSize {
		end := i + chunkSize
		if end > len(names) {
			end = len(names)
		}
		chunks = append(chunks, names[i:end])
	}
	results := make(chan result, len(chunks))
	sem := make(chan struct{}, maxWorkers)
	for _, chunk := range chunks {
		sem <- struct{}{}
		go func(pkgs []string) {
			defer func() { <-sem }()
			args := append([]string{"show"}, pkgs...)
			cmd := exec.Command("apt-cache", args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &bytes.Buffer{}
			_ = cmd.Run()
			info := make(map[string]PackageInfo, len(pkgs))
			var curPkg, curVer, curSize, curSection, curArch string
			flush := func() {
				if curPkg != "" {
					if _, exists := info[curPkg]; !exists {
						info[curPkg] = PackageInfo{
							Version:      curVer,
							Size:         formatSize(curSize),
							Section:      curSection,
							Architecture: curArch,
						}
					}
				}
				curPkg, curVer, curSize, curSection, curArch = "", "", "", "", ""
			}
			for _, line := range strings.Split(out.String(), "\n") {
				if line == "" {
					flush()
					continue
				}
				if strings.HasPrefix(line, "Package: ") {
					curPkg = line[9:]
				} else if strings.HasPrefix(line, "Version: ") {
					curVer = line[9:]
				} else if strings.HasPrefix(line, "Installed-Size: ") {
					curSize = line[16:]
				} else if strings.HasPrefix(line, "Section: ") {
					curSection = line[9:]
				} else if strings.HasPrefix(line, "Architecture: ") {
					curArch = line[14:]
				}
			}
			flush()
			results <- result{info: info}
		}(chunk)
	}
	merged := make(map[string]PackageInfo, len(names))
	for range chunks {
		r := <-results
		for k, v := range r.info {
			merged[k] = v
		}
	}
	return merged
}

// BenchmarkOldBatchGetInfo50 benchmarks the old approach: apt-cache show
// for 50 packages (a single metadata filter submission with few candidates).
func BenchmarkOldBatchGetInfo50(b *testing.B) {
	names := getTestPackageNames(b, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := oldBatchGetInfo(names)
		if len(info) == 0 {
			b.Fatal("no info returned")
		}
	}
}

// BenchmarkOldBatchGetInfo500 benchmarks the old approach: apt-cache show
// for 500 packages (a typical metadata filter with many candidates).
func BenchmarkOldBatchGetInfo500(b *testing.B) {
	names := getTestPackageNames(b, 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := oldBatchGetInfo(names)
		if len(info) == 0 {
			b.Fatal("no info returned")
		}
	}
}

// BenchmarkOldBatchGetInfo5000 benchmarks the old approach: apt-cache show
// for 5000 packages (a broad filter like "section:utils" or "installed").
func BenchmarkOldBatchGetInfo5000(b *testing.B) {
	names := getTestPackageNames(b, 5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := oldBatchGetInfo(names)
		if len(info) == 0 {
			b.Fatal("no info returned")
		}
	}
}

// BenchmarkOldFullFlow benchmarks the complete old startup flow:
// ListAllNames + ListInstalled in parallel (simulating reloadAllPackages).
func BenchmarkOldFullFlow(b *testing.B) {
	// Warm up
	_, _ = ListAllNames()
	_, _ = ListInstalled()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := make(chan struct{}, 2)
		go func() { _, _ = ListAllNames(); done <- struct{}{} }()
		go func() { _, _ = ListInstalled(); done <- struct{}{} }()
		<-done
		<-done
	}
}

// BenchmarkNewFullFlow benchmarks the new startup flow:
// LoadAllAvailableInfo + ListInstalled in parallel.
func BenchmarkNewFullFlow(b *testing.B) {
	// Warm up
	_ = LoadAllAvailableInfo()
	_, _ = ListInstalled()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := make(chan struct{}, 2)
		go func() { _ = LoadAllAvailableInfo(); done <- struct{}{} }()
		go func() { _, _ = ListInstalled(); done <- struct{}{} }()
		<-done
		<-done
	}
}

func getTestPackageNames(b *testing.B, n int) []string {
	b.Helper()
	all, err := ListAllNames()
	if err != nil {
		b.Fatal(err)
	}
	if len(all) < n {
		n = len(all)
	}
	return all[:n]
}
