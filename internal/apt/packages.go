// Package apt provides functionality for parsing APT package manager output.
package apt

import (
	"strings"

	"github.com/mexirica/gpm/internal/model"
)

func parseDpkgOutput(output string, installed bool) []model.Package {
	var packages []model.Package
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		pkg := model.Package{
			Installed: installed,
			Name:      strings.TrimSpace(parts[0]),
			Version:   strings.TrimSpace(parts[1]),
		}
		if len(parts) >= 3 {
			pkg.Description = strings.TrimSpace(parts[2])
		}
		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}
	return packages
}

func parseSearchOutput(output string) []model.Package {
	var packages []model.Package
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
		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}
	return packages
}

func parseUpgradableOutput(output string) []model.Package {
	var packages []model.Package
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
		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}
	return packages
}
