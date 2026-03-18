package apt

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mexirica/aptui/internal/model"
)

// LoadAllAvailableInfo parses /var/lib/apt/lists/*_Packages files to bulk-load
// metadata for all available packages. This is much faster than spawning
// apt-cache show processes because it's pure file I/O with no process overhead.
func LoadAllAvailableInfo() map[string]PackageInfo {
	files, err := filepath.Glob("/var/lib/apt/lists/*_Packages")
	if err != nil || len(files) == 0 {
		return nil
	}

	info := make(map[string]PackageInfo, 100000)

	for _, f := range files {
		parsePackageFile(f, info)
	}

	return info
}

// parsePackageFile parses a single *_Packages file and merges entries into info.
// Later files overwrite earlier ones; note that filepath.Glob returns files in
// lexicographic order, which may not exactly match apt pin priorities —
// this is a known simplification that works for typical setups.
func parsePackageFile(path string, info map[string]PackageInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var curPkg, curVer, curSize, curSection, curArch string

	var curDesc string

	flush := func() {
		if curPkg != "" {
			info[curPkg] = PackageInfo{
				Version:      curVer,
				Size:         formatSize(curSize),
				Section:      curSection,
				Architecture: curArch,
				Description:  curDesc,
			}
		}
		curPkg, curVer, curSize, curSection, curArch, curDesc = "", "", "", "", "", ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			flush()
			continue
		}
		switch line[0] {
		case 'P':
			if strings.HasPrefix(line, "Package: ") {
				curPkg = line[9:]
			}
		case 'V':
			if strings.HasPrefix(line, "Version: ") {
				curVer = line[9:]
			}
		case 'I':
			if strings.HasPrefix(line, "Installed-Size: ") {
				curSize = line[16:]
			}
		case 'S':
			if strings.HasPrefix(line, "Section: ") {
				curSection = line[9:]
			}
		case 'A':
			if strings.HasPrefix(line, "Architecture: ") {
				curArch = line[14:]
			}
		case 'D':
			if strings.HasPrefix(line, "Description") && !strings.HasPrefix(line, "Description-md5") {
				if curDesc == "" {
					if idx := strings.Index(line, ": "); idx != -1 {
						curDesc = line[idx+2:]
					}
				}
			}
		}
	}
	flush()
	// Ignore scanner errors (e.g. token too long); entries parsed so far
	// are still usable, and the background reload will recover.
	_ = scanner.Err()
}

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
		"-f=${Package}\t${Version}\t${Installed-Size}\t${binary:Summary}\t${Section}\t${Architecture}\n")
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

// ListHeld returns the names of packages currently held back via apt-mark.
func ListHeld() ([]string, error) {
	cmd := exec.Command("apt-mark", "showhold")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("apt-mark showhold: %s", stderr.String())
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

// Hold holds packages via apt-mark hold.
func Hold(names []string) error {
	args := append([]string{"-n", "apt-mark", "hold"}, names...)
	cmd := exec.Command("sudo", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-mark hold: %s", stderr.String())
	}
	return nil
}

// Unhold unholds packages via apt-mark unhold.
func Unhold(names []string) error {
	args := append([]string{"-n", "apt-mark", "unhold"}, names...)
	cmd := exec.Command("sudo", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-mark unhold: %s", stderr.String())
	}
	return nil
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

// SetPPAEnabled enables or disables a PPA by modifying its source file.
func SetPPAEnabled(ppa PPA, enabled bool) error {
	data, err := os.ReadFile(ppa.File)
	if err != nil {
		return fmt.Errorf("read %s: %w", ppa.File, err)
	}
	content := string(data)
	var newContent string

	if strings.HasSuffix(ppa.File, ".list") {
		newContent = toggleListFile(content, ppa, enabled)
	} else if strings.HasSuffix(ppa.File, ".sources") {
		newContent = toggleSourcesFile(content, enabled)
	} else {
		return fmt.Errorf("unsupported source file format: %s", ppa.File)
	}

	cmd := exec.Command("sudo", "tee", ppa.File)
	cmd.Stdin = strings.NewReader(newContent)
	cmd.Stdout = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write %s: %s", ppa.File, stderr.String())
	}
	return nil
}

func toggleListFile(content string, ppa PPA, enabled bool) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		raw := trimmed
		if strings.HasPrefix(raw, "#") {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "#"))
		}
		if !strings.HasPrefix(raw, "deb") {
			continue
		}
		if !strings.Contains(raw, "ppa.launchpad.net") && !strings.Contains(raw, "ppa.launchpadcontent.net") {
			continue
		}
		if extractPPAName(raw) != ppa.Name {
			continue
		}
		if enabled {
			// Remove leading "# " to enable
			lines[i] = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		} else {
			// Add "# " to disable (only if not already commented)
			if !strings.HasPrefix(trimmed, "#") {
				lines[i] = "# " + trimmed
			}
		}
	}
	return strings.Join(lines, "\n")
}

func toggleSourcesFile(content string, enabled bool) string {
	lines := strings.Split(content, "\n")
	foundEnabled := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Enabled:") {
			foundEnabled = true
			if enabled {
				lines[i] = "Enabled: yes"
			} else {
				lines[i] = "Enabled: no"
			}
		}
	}
	if !foundEnabled && !enabled {
		// Insert "Enabled: no" after the "Types:" line
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "Types:") {
				rest := make([]string, len(lines)-i-1)
				copy(rest, lines[i+1:])
				lines = append(lines[:i+1], "Enabled: no")
				lines = append(lines, rest...)
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

type PackageInfo struct {
	Version      string
	Size         string
	Section      string
	Architecture string
	Description  string
}

// ParseShowEntry parses a single apt-cache show output and returns PackageInfo.
func ParseShowEntry(info string) PackageInfo {
	var ver, size, section, arch, desc string
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
		} else if strings.HasPrefix(line, "Description") && !strings.HasPrefix(line, "Description-md5") {
			if desc == "" {
				if idx := strings.Index(line, ": "); idx != -1 {
					desc = line[idx+2:]
				}
			}
		}
	}
	return PackageInfo{
		Version:      ver,
		Size:         formatSize(size),
		Section:      section,
		Architecture: arch,
		Description:  desc,
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
