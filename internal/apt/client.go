package apt

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mexirica/aptui/internal/model"
)

func SilentUpdate() error {
	cmd := exec.Command("sudo", "-n", "apt-get", "update", "-qq")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func UpdateCmd() *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "update")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func AutoRemoveCmd() *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "autoremove", "-y")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// ListAutoremovable returns the names of packages that can be autoremoved.
func ListAutoremovable() ([]string, error) {
	cmd := exec.Command("apt-get", "autoremove", "-s")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt autoremove -s: %s", stderr.String())
	}
	var names []string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "Remv") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				names = append(names, fields[1])
			}
		}
	}
	return names, nil
}

func ListInstalled() ([]model.Package, error) {
	cmd := exec.Command("dpkg-query", "-W",
		"-f=${Package}\t${Version}\t${Installed-Size}\t${Description}\t${Section}\t${Architecture}\n")
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

// InstallBatchCmd returns an install command for multiple packages at once.
func InstallBatchCmd(names []string) *exec.Cmd {
	args := []string{
		"apt-get", "install", "-y",
		"-o", "Acquire::Queue-Mode=access",
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Pipeline-Depth=5",
		"-o", "Acquire::Languages=none",
	}
	args = append(args, names...)
	c := exec.Command("sudo", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// UpgradeBatchCmd returns an upgrade command for multiple packages at once.
func UpgradeBatchCmd(names []string) *exec.Cmd {
	args := []string{
		"apt-get", "install", "--only-upgrade", "-y",
		"-o", "Acquire::Queue-Mode=access",
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Pipeline-Depth=5",
		"-o", "Acquire::Languages=none",
	}
	args = append(args, names...)
	c := exec.Command("sudo", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// RemoveBatchCmd returns a remove command for multiple packages at once.
func RemoveBatchCmd(names []string) *exec.Cmd {
	args := append([]string{"apt-get", "remove", "-y"}, names...)
	c := exec.Command("sudo", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// PurgeBatchCmd returns a purge command for multiple packages at once.
func PurgeBatchCmd(names []string) *exec.Cmd {
	args := append([]string{"apt-get", "purge", "-y"}, names...)
	c := exec.Command("sudo", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

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

func IsInstalled(name string) bool {
	cmd := exec.Command("dpkg-query", "-W", "-f=${Status}", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.Contains(out.String(), "install ok installed")
}

// PPA represents a PPA repository configured on the system.
type PPA struct {
	Name    string // e.g. "ppa:deadsnakes/ppa"
	URL     string // e.g. "https://ppa.launchpad.net/deadsnakes/ppa/ubuntu"
	File    string // source file path
	Enabled bool
}

// ListPPAs scans /etc/apt/sources.list.d/ for PPA entries.
func ListPPAs() ([]PPA, error) {
	dir := "/etc/apt/sources.list.d"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read sources.list.d: %w", err)
	}

	var ppas []PPA
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := dir + "/" + entry.Name()

		if strings.HasSuffix(entry.Name(), ".list") {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				enabled := true
				if strings.HasPrefix(line, "#") {
					enabled = false
					line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
				}
				if !strings.HasPrefix(line, "deb") {
					continue
				}
				if !strings.Contains(line, "ppa.launchpad.net") && !strings.Contains(line, "ppa.launchpadcontent.net") {
					continue
				}
				ppaName := extractPPAName(line)
				if ppaName != "" && !seen[ppaName] {
					seen[ppaName] = true
					ppas = append(ppas, PPA{
						Name:    ppaName,
						URL:     extractPPAURL(line),
						File:    path,
						Enabled: enabled,
					})
				}
			}
		}

		if strings.HasSuffix(entry.Name(), ".sources") {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			content := string(data)
			if !strings.Contains(content, "ppa.launchpad.net") && !strings.Contains(content, "ppa.launchpadcontent.net") {
				continue
			}
			for _, line := range strings.Split(content, "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "URIs:") {
					continue
				}
				uri := strings.TrimSpace(strings.TrimPrefix(line, "URIs:"))
				ppaName := extractPPAName(uri)
				if ppaName != "" && !seen[ppaName] {
					seen[ppaName] = true
					enabled := !strings.Contains(content, "Enabled: no")
					ppas = append(ppas, PPA{
						Name:    ppaName,
						URL:     uri,
						File:    path,
						Enabled: enabled,
					})
				}
			}
		}
	}

	return ppas, nil
}

func extractPPAName(line string) string {
	patterns := []string{"ppa.launchpad.net/", "ppa.launchpadcontent.net/"}
	for _, pat := range patterns {
		idx := strings.Index(line, pat)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(pat):]
		parts := strings.SplitN(rest, "/", 3)
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return "ppa:" + parts[0] + "/" + parts[1]
		}
	}
	return ""
}

func extractPPAURL(line string) string {
	for _, field := range strings.Fields(line) {
		if strings.Contains(field, "ppa.launchpad.net") || strings.Contains(field, "ppa.launchpadcontent.net") {
			return field
		}
	}
	return ""
}

// ValidatePPA checks that a PPA string has the correct format.
func ValidatePPA(input string) error {
	if !strings.HasPrefix(input, "ppa:") {
		return fmt.Errorf("PPA must start with 'ppa:' (e.g. ppa:user/repo)")
	}
	rest := strings.TrimPrefix(input, "ppa:")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("PPA format must be 'ppa:user/repository'")
	}
	return nil
}

// AddPPACmd returns a command to add a PPA repository.
func AddPPACmd(ppa string) *exec.Cmd {
	c := exec.Command("sudo", "add-apt-repository", "-y", ppa)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// RemovePPACmd returns a command to remove a PPA repository.
func RemovePPACmd(ppa string) *exec.Cmd {
	c := exec.Command("sudo", "add-apt-repository", "-y", "--remove", ppa)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

type PackageInfo struct {
	Version      string
	Size         string
	Section      string
	Architecture string
}

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
	var curSection string
	var curArch string

	flush := func() {
		if curPkg != "" {
			// Keep only the first entry per package (candidate version)
			if _, exists := info[curPkg]; !exists {
				info[curPkg] = PackageInfo{
					Version:      curVer,
					Size:         formatSize(curSize),
					Section:      curSection,
					Architecture: curArch,
				}
			}
		}
		curPkg = ""
		curVer = ""
		curSize = ""
		curSection = ""
		curArch = ""
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
		} else if strings.HasPrefix(line, "Section: ") {
			curSection = strings.TrimPrefix(line, "Section: ")
		} else if strings.HasPrefix(line, "Architecture: ") {
			curArch = strings.TrimPrefix(line, "Architecture: ")
		}
	}
	flush() // last entry

	return info
}

// ParseShowEntry parses a single apt-cache show output and returns PackageInfo.
func ParseShowEntry(info string) PackageInfo {
	var ver, size, section, arch string
	for _, line := range strings.Split(info, "\n") {
		if line == "" && ver != "" {
			break // only first entry
		}
		if strings.HasPrefix(line, "Version: ") {
			ver = strings.TrimPrefix(line, "Version: ")
		} else if strings.HasPrefix(line, "Installed-Size: ") {
			size = strings.TrimPrefix(line, "Installed-Size: ")
		} else if strings.HasPrefix(line, "Section: ") {
			section = strings.TrimPrefix(line, "Section: ")
		} else if strings.HasPrefix(line, "Architecture: ") {
			arch = strings.TrimPrefix(line, "Architecture: ")
		}
	}
	return PackageInfo{
		Version:      ver,
		Size:         formatSize(size),
		Section:      section,
		Architecture: arch,
	}
}

// GetDependencies returns the direct dependency package names for a given package.
func GetDependencies(name string) ([]string, error) {
	cmd := exec.Command("apt-cache", "depends", "--no-recommends", "--no-suggests",
		"--no-conflicts", "--no-breaks", "--no-replaces", "--no-enhances", name)
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt-cache depends: %s", stderr.String())
	}

	seen := make(map[string]bool)
	var deps []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Depends:") {
			dep := strings.TrimSpace(strings.TrimPrefix(line, "Depends:"))
			// skip virtual packages (lines starting with <)
			if dep != "" && !strings.HasPrefix(dep, "<") && !seen[dep] {
				seen[dep] = true
				deps = append(deps, dep)
			}
		}
	}
	return deps, nil
}
