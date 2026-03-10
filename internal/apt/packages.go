// Package apt provides functionality for parsing APT package manager output.
package apt

import (
	"fmt"
	"strings"

	"github.com/mexirica/aptui/internal/model"
)

const (
	gb = 1048576
	mb = 1024
)

// formatSize converts a size in kB (as reported by dpkg) to a human-friendly string.
func formatSize(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "-"
	}
	var size int64
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			size = size*10 + int64(c-'0')
		}
	}
	if size == 0 {
		return "-"
	}
	switch {
	case size >= 1*gb:
		return fmt.Sprintf("%.1f GB", float64(size)/gb)
	case size >= 1*mb:
		return fmt.Sprintf("%.1f MB", float64(size)/mb)
	default:
		return fmt.Sprintf("%d kB", size)
	}
}

func parseDpkgOutput(output string, installed bool) []model.Package {
	var packages []model.Package
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) < 2 {
			continue
		}
		pkg := model.Package{
			Installed: installed,
			Name:      strings.TrimSpace(parts[0]),
			Version:   strings.TrimSpace(parts[1]),
		}
		if len(parts) >= 3 {
			pkg.Size = formatSize(parts[2])
		}
		if len(parts) >= 4 {
			pkg.Description = strings.TrimSpace(parts[3])
		}
		if len(parts) >= 5 {
			pkg.Section = strings.TrimSpace(parts[4])
		}
		if len(parts) >= 6 {
			pkg.Architecture = strings.TrimSpace(parts[5])
		}
		if pkg.Name != "" && !seen[pkg.Name] {
			packages = append(packages, pkg)
			seen[pkg.Name] = true
		}
	}
	return packages
}

func parseSearchOutput(output string) []model.Package {
	var packages []model.Package
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		pkg := model.Package{
			Name:      strings.TrimSpace(parts[0]),
			Installed: false,
		}
		if len(parts) == 2 {
			pkg.Description = strings.TrimSpace(parts[1])
		}
		pkg.Installed = IsInstalled(pkg.Name)
		if pkg.Name != "" && !seen[pkg.Name] {
			packages = append(packages, pkg)
			seen[pkg.Name] = true
		}
	}
	return packages
}

func parseUpgradableOutput(output string) []model.Package {
	var packages []model.Package
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		nameParts := strings.SplitN(parts[0], "/", 2)
		pkg := model.Package{
			Name:       nameParts[0],
			Installed:  true,
			Upgradable: true,
		}
		if len(parts) >= 2 {
			pkg.NewVersion = parts[1]
		}
		if idx := strings.Index(line, "upgradable from:"); idx != -1 {
			rest := line[idx+len("upgradable from:"):]
			rest = strings.TrimLeft(rest, " ")
			rest = strings.TrimRight(rest, "]")
			pkg.Version = strings.TrimSpace(rest)
		}
		if pkg.Name != "" && !seen[pkg.Name] {
			packages = append(packages, pkg)
			seen[pkg.Name] = true
		}
	}
	return packages
}
