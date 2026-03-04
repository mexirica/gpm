package apt

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mexirica/gpm/internal/model"
)

func ListInstalled() ([]model.Package, error) {
	cmd := exec.Command("dpkg-query", "-W",
		"-f=${Package}\t${Version}\t${Installed-Size}\t${Description}\n")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dpkg-query: %s", stderr.String())
	}
	return parseDpkgOutput(out.String(), true), nil
}

func SearchPackages(query string) ([]model.Package, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	cmd := exec.Command("apt-cache", "search", query)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt-cache search: %s", stderr.String())
	}
	return parseSearchOutput(out.String()), nil
}

func ShowPackage(name string) (string, error) {
	cmd := exec.Command("apt-cache", "show", name)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("apt-cache show: %s", stderr.String())
	}
	return out.String(), nil
}

func ListUpgradable() ([]model.Package, error) {
	cmd := exec.Command("apt", "list", "--upgradable")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt list --upgradable: %s", stderr.String())
	}
	return parseUpgradableOutput(out.String()), nil
}

func InstallCmd(name string) *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "install", "-y", name)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func RemoveCmd(name string) *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "remove", "-y", name)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func UpgradeCmd(name string) *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "install", "--only-upgrade", "-y", name)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func UpgradeAllCmd() *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "upgrade", "-y")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func UpdateIndexCmd() *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "update")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// ListAllNames returns all available package names from the apt cache.
// This uses 'apt-cache pkgnames' which is fast (~70k names in <1s).
func ListAllNames() ([]string, error) {
	cmd := exec.Command("apt-cache", "pkgnames")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt-cache pkgnames: %s", stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	names := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			names = append(names, l)
		}
	}
	return names, nil
}

// ListAllWithVersions is deprecated — use BatchGetVersions for lazy loading.
// Kept for reference but no longer called.

// BatchGetVersions uses 'apt-cache policy' to get candidate versions for
// a batch of package names. It splits work into chunks and runs them in
// parallel for speed. Returns a map of name → candidate version.
func BatchGetVersions(names []string) map[string]string {
	if len(names) == 0 {
		return nil
	}

	const chunkSize = 100 // packages per apt-cache policy call
	const maxWorkers = 8  // parallel workers

	type result struct {
		versions map[string]string
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
			v := getPolicyVersions(pkgs)
			results <- result{versions: v}
		}(chunk)
	}

	// Collect
	merged := make(map[string]string, len(names))
	for range chunks {
		r := <-results
		for k, v := range r.versions {
			merged[k] = v
		}
	}

	return merged
}

// getPolicyVersions runs 'apt-cache policy pkg1 pkg2 ...' and parses
// the Candidate: line for each package.
func getPolicyVersions(names []string) map[string]string {
	args := append([]string{"policy"}, names...)
	cmd := exec.Command("apt-cache", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil
	}

	versions := make(map[string]string, len(names))
	var curPkg string
	for _, line := range strings.Split(out.String(), "\n") {
		trimmed := strings.TrimSpace(line)
		// Package header line: "pkgname:" (no leading whitespace in original line)
		if len(line) > 0 && line[0] != ' ' && strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed[:len(trimmed)-1], " ") {
			curPkg = strings.TrimSuffix(trimmed, ":")
		}
		if strings.HasPrefix(trimmed, "Candidate:") && curPkg != "" {
			ver := strings.TrimSpace(strings.TrimPrefix(trimmed, "Candidate:"))
			if ver != "" && ver != "(none)" {
				versions[curPkg] = ver
			}
		}
	}
	return versions
}

func IsInstalled(name string) bool {
	cmd := exec.Command("dpkg-query", "-W", "-f=${Status}", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.Contains(out.String(), "install ok installed")
}

// PackageInfo holds version and formatted size for a package.
type PackageInfo struct {
	Version string
	Size    string
}

// BatchGetInfo uses 'apt-cache show --no-all-versions' to get version and
// installed-size for a batch of package names. It splits work into chunks
// and runs them in parallel. Returns a map of name → PackageInfo.
func BatchGetInfo(names []string) map[string]PackageInfo {
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
			v := getShowInfo(pkgs)
			results <- result{info: v}
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

// getShowInfo runs 'apt-cache show pkg1 pkg2 ...' and
// parses Package, Version, and Installed-Size for each entry.
// Only the first entry per package is kept (the candidate version).
func getShowInfo(names []string) map[string]PackageInfo {
	args := append([]string{"show"}, names...)
	cmd := exec.Command("apt-cache", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	// Ignore error: apt-cache show returns non-zero if any package is
	// not found, but still outputs data for the ones that exist.
	_ = cmd.Run()

	info := make(map[string]PackageInfo, len(names))
	var curPkg string
	var curVer string
	var curSize string

	flush := func() {
		if curPkg != "" {
			// Keep only the first entry per package (candidate version)
			if _, exists := info[curPkg]; !exists {
				info[curPkg] = PackageInfo{
					Version: curVer,
					Size:    formatSize(curSize),
				}
			}
		}
		curPkg = ""
		curVer = ""
		curSize = ""
	}

	for _, line := range strings.Split(out.String(), "\n") {
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "Package: ") {
			curPkg = strings.TrimPrefix(line, "Package: ")
		} else if strings.HasPrefix(line, "Version: ") {
			curVer = strings.TrimPrefix(line, "Version: ")
		} else if strings.HasPrefix(line, "Installed-Size: ") {
			curSize = strings.TrimPrefix(line, "Installed-Size: ")
		}
	}
	flush() // last entry

	return info
}

// ParseShowEntry parses a single apt-cache show output and returns PackageInfo.
func ParseShowEntry(info string) PackageInfo {
	var ver, size string
	for _, line := range strings.Split(info, "\n") {
		if line == "" && ver != "" {
			break // only first entry
		}
		if strings.HasPrefix(line, "Version: ") {
			ver = strings.TrimPrefix(line, "Version: ")
		} else if strings.HasPrefix(line, "Installed-Size: ") {
			size = strings.TrimPrefix(line, "Installed-Size: ")
		}
	}
	return PackageInfo{
		Version: ver,
		Size:    formatSize(size),
	}
}
