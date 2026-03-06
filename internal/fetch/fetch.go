package fetch

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Mirror represents a single package mirror with its test results.
type Mirror struct {
	URL     string
	Country string
	Latency time.Duration
	Status  string // "ok", "slow", "error"
	Score   int    
	Active  bool 
}

type Distro struct {
	ID       string // e.g. "ubuntu", "debian", "pop"
	Codename string // e.g. "noble", "bookworm"
	Name     string // e.g. "Ubuntu 24.04"
}

// DetectDistro reads /etc/os-release to determine the distribution.
func DetectDistro() (Distro, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return Distro{}, fmt.Errorf("cannot read /etc/os-release: %w", err)
	}
	defer f.Close()

	d := Distro{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			d.ID = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			d.Codename = strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), "\"")
		}
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			d.Name = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}

	// Normalize: Pop!_OS and other Ubuntu derivatives use Ubuntu mirrors
	if d.Codename == "" {
		out, err := exec.Command("lsb_release", "-cs").Output()
		if err == nil {
			d.Codename = strings.TrimSpace(string(out))
		}
	}

	if d.ID == "" {
		return d, fmt.Errorf("could not detect distribution")
	}

	return d, nil
}

// baseDistro returns the upstream distro ID for mirror fetching.
func baseDistro(d Distro) string {
	switch d.ID {
	case "ubuntu", "pop", "linuxmint", "elementary", "zorin", "neon":
		return "ubuntu"
	case "debian", "kali", "mx", "antiX", "devuan":
		return "debian"
	default:
		f, err := os.ReadFile("/etc/os-release")
		if err == nil && strings.Contains(string(f), "ubuntu") {
			return "ubuntu"
		}
		return d.ID
	}
}

// FetchMirrorList gets mirrors for the detected distribution.
func FetchMirrorList(d Distro) ([]Mirror, error) {
	base := baseDistro(d)
	switch base {
	case "ubuntu":
		return fetchUbuntuMirrors()
	case "debian":
		return fetchDebianMirrors()
	default:
		return nil, fmt.Errorf("unsupported distro for mirror fetch: %s", d.ID)
	}
}

func fetchUbuntuMirrors() ([]Mirror, error) {
	resp, err := http.Get("https://launchpad.net/ubuntu/+archivemirrors-rss")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Ubuntu mirror list: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`<link>(https?://[^<]+)</link>`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var mirrors []Mirror
	seen := make(map[string]bool)
	for _, m := range matches {
		url := strings.TrimSpace(m[1])
		if strings.Contains(url, "launchpad.net") {
			continue
		}
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		if seen[url] {
			continue
		}
		seen[url] = true
		mirrors = append(mirrors, Mirror{
			URL:    url,
			Status: "pending",
		})
	}

	if len(mirrors) == 0 {
		mirrors = defaultUbuntuMirrors()
	}

	return mirrors, nil
}

func fetchDebianMirrors() ([]Mirror, error) {
	resp, err := http.Get("https://www.debian.org/mirror/list-full")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Debian mirror list: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(https?://[a-zA-Z0-9._/-]+/debian/)`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var mirrors []Mirror
	seen := make(map[string]bool)
	for _, m := range matches {
		url := strings.TrimSpace(m[1])
		if seen[url] {
			continue
		}
		seen[url] = true
		mirrors = append(mirrors, Mirror{
			URL:    url,
			Status: "pending",
		})
	}

	if len(mirrors) == 0 {
		mirrors = defaultDebianMirrors()
	}

	return mirrors, nil
}

func defaultUbuntuMirrors() []Mirror {
	urls := []string{
		"http://archive.ubuntu.com/ubuntu/",
		"http://us.archive.ubuntu.com/ubuntu/",
		"http://br.archive.ubuntu.com/ubuntu/",
		"http://de.archive.ubuntu.com/ubuntu/",
		"http://fr.archive.ubuntu.com/ubuntu/",
		"http://uk.archive.ubuntu.com/ubuntu/",
		"http://nl.archive.ubuntu.com/ubuntu/",
		"http://se.archive.ubuntu.com/ubuntu/",
		"http://jp.archive.ubuntu.com/ubuntu/",
		"http://au.archive.ubuntu.com/ubuntu/",
	}
	var mirrors []Mirror
	for _, u := range urls {
		mirrors = append(mirrors, Mirror{URL: u, Status: "pending"})
	}
	return mirrors
}

func defaultDebianMirrors() []Mirror {
	urls := []string{
		"https://deb.debian.org/debian/",
		"http://ftp.us.debian.org/debian/",
		"http://ftp.br.debian.org/debian/",
		"http://ftp.de.debian.org/debian/",
		"http://ftp.fr.debian.org/debian/",
		"http://ftp.uk.debian.org/debian/",
		"http://ftp.nl.debian.org/debian/",
		"http://ftp.se.debian.org/debian/",
		"http://ftp.jp.debian.org/debian/",
		"http://ftp.au.debian.org/debian/",
	}
	var mirrors []Mirror
	for _, u := range urls {
		mirrors = append(mirrors, Mirror{URL: u, Status: "pending"})
	}
	return mirrors
}

type TestResult struct {
	Index   int
	Latency time.Duration
	Err     error
}

// LimitMirrors returns at most max mirrors, sampled evenly.
func LimitMirrors(mirrors []Mirror, max int) []Mirror {
	if len(mirrors) <= max {
		return mirrors
	}
	step := len(mirrors) / max
	if step < 1 {
		step = 1
	}
	var limited []Mirror
	for i := 0; i < len(mirrors) && len(limited) < max; i += step {
		limited = append(limited, mirrors[i])
	}
	return limited
}

func TestMirrorsChan(mirrors []Mirror) <-chan TestResult {
	ch := make(chan TestResult, 30)
	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 25) // 25 concurrent

		for i := range mirrors {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				latency, err := testMirrorLatency(mirrors[idx].URL)
				ch <- TestResult{Index: idx, Latency: latency, Err: err}
			}(i)
		}

		wg.Wait()
		close(ch)
	}()
	return ch
}

func testMirrorLatency(url string) (time.Duration, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	testURL := url + "dists/"
	start := time.Now()
	resp, err := client.Head(testURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	latency := time.Since(start)

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return latency, nil
}

// ScoreMirrors assigns scores and sorts mirrors by latency (fastest first).
func ScoreMirrors(mirrors []Mirror) []Mirror {
	var scored []Mirror
	for i := range mirrors {
		if mirrors[i].Status == "ok" {
			scored = append(scored, mirrors[i])
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Latency < scored[j].Latency
	})

	for i := range scored {
		scored[i].Score = 100 - (i * 100 / (len(scored) + 1))
	}

	return scored
}

// WriteSourcesListCmd writes the selected mirrors to /etc/apt/sources.list.d/gpm-mirrors.list.
// Returns an exec.Cmd that must be run with sudo.
func WriteSourcesListCmd(mirrors []Mirror, d Distro) *exec.Cmd {
	var lines []string
	lines = append(lines, "# Generated by GPM - Fastest mirrors")
	lines = append(lines, fmt.Sprintf("# Distro: %s (%s)", d.Name, d.Codename))
	lines = append(lines, fmt.Sprintf("# Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	lines = append(lines, "")

	base := baseDistro(d)
	for _, m := range mirrors {
		if !m.Active {
			continue
		}
		if base == "ubuntu" {
			lines = append(lines, fmt.Sprintf("deb %s %s main restricted universe multiverse", m.URL, d.Codename))
			lines = append(lines, fmt.Sprintf("deb %s %s-updates main restricted universe multiverse", m.URL, d.Codename))
			lines = append(lines, fmt.Sprintf("deb %s %s-security main restricted universe multiverse", m.URL, d.Codename))
		} else {
			lines = append(lines, fmt.Sprintf("deb %s %s main contrib non-free non-free-firmware", m.URL, d.Codename))
			lines = append(lines, fmt.Sprintf("deb %s %s-updates main contrib non-free non-free-firmware", m.URL, d.Codename))
		}
	}
	content := strings.Join(lines, "\n") + "\n"

	c := exec.Command("sudo", "tee", "/etc/apt/sources.list.d/gpm-mirrors.list")
	c.Stdin = strings.NewReader(content)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

// FormatLatency returns a human-friendly latency string.
func FormatLatency(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%d ms", ms)
	}
	return fmt.Sprintf("%.1f s", d.Seconds())
}
