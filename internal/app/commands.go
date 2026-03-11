package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/model"
)

func purgeBatchCmd(names []string) tea.Cmd {
	cmd := apt.PurgeBatchCmd(names)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "purge", name: strings.Join(names, " "), err: err}
	})
}

func reloadAllPackages() tea.Msg {
	type namesResult struct {
		names []string
		err   error
	}
	type pkgResult struct {
		pkgs []model.Package
		err  error
	}

	namesCh := make(chan namesResult, 1)
	installedCh := make(chan pkgResult, 1)
	upgradableCh := make(chan pkgResult, 1)

	go func() {
		n, err := apt.ListAllNames()
		namesCh <- namesResult{n, err}
	}()
	go func() {
		p, err := apt.ListInstalled()
		installedCh <- pkgResult{p, err}
	}()
	go func() {
		p, err := apt.ListUpgradable()
		upgradableCh <- pkgResult{p, err}
	}()

	nr := <-namesCh
	ir := <-installedCh
	ur := <-upgradableCh

	var allNames []string
	if nr.err == nil {
		allNames = nr.names
	}
	if ir.err != nil {
		return allPackagesMsg{nil, nil, nil, ir.err}
	}
	return allPackagesMsg{allNames, ir.pkgs, ur.pkgs, nil}
}

func aptUpdateCmd() tea.Cmd {
	cmd := apt.UpdateCmd()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "update", name: "apt", err: err}
	})
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func silentUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		_ = apt.SilentUpdate()
		names, _ := apt.ListAllNames()
		pkgs, _ := apt.ListUpgradable()
		return silentUpdateDoneMsg{names: names, upgradable: pkgs}
	}
}

func searchPackagesCmd(query string) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := apt.SearchPackages(query)
		return searchResultMsg{pkgs, err}
	}
}

func showPackageDetailCmd(name string) tea.Cmd {
	return func() tea.Msg {
		info, err := apt.ShowPackage(name)
		return detailLoadedMsg{name, info, err}
	}
}

func loadTransactionDepsCmd(txIdx int, packages []string) tea.Cmd {
	return func() tea.Msg {
		seen := make(map[string]bool)
		for _, pkg := range packages {
			seen[pkg] = true
		}
		allDeps := []string{}
		for _, pkg := range packages {
			deps, err := apt.GetDependencies(pkg)
			if err != nil {
				continue
			}
			for _, d := range deps {
				if !seen[d] {
					seen[d] = true
					allDeps = append(allDeps, d)
				}
			}
		}
		return depsLoadedMsg{txIdx: txIdx, deps: allDeps}
	}
}

func upgradeAllPackagesCmd(names []string) tea.Cmd {
	cmd := apt.UpgradeBatchCmd(names)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade-all", name: strings.Join(names, " "), err: err}
	})
}

func installBatchCmd(names []string) tea.Cmd {
	cmd := apt.InstallBatchCmd(names)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "install", name: strings.Join(names, " "), err: err}
	})
}

func removeBatchCmd(names []string) tea.Cmd {
	cmd := apt.RemoveBatchCmd(names)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "remove", name: strings.Join(names, " "), err: err}
	})
}

func upgradeBatchCmd(names []string) tea.Cmd {
	cmd := apt.UpgradeBatchCmd(names)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade", name: strings.Join(names, " "), err: err}
	})
}

func fetchMirrorListCmd() tea.Cmd {
	return func() tea.Msg {
		distro, err := fetch.DetectDistro()
		if err != nil {
			return fetchMirrorsMsg{err: err}
		}
		mirrors, err := fetch.FetchMirrorList(distro)
		if err != nil {
			return fetchMirrorsMsg{err: err}
		}
		return fetchMirrorsMsg{mirrors: mirrors, distro: distro}
	}
}

func awaitMirrorTestResult(ch <-chan fetch.TestResult) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-ch
		if !ok {
			return fetchTestResultMsg{done: true}
		}
		return fetchTestResultMsg{result: r, done: false}
	}
}

func loadAutoremovableCmd() tea.Cmd {
	return func() tea.Msg {
		names, err := apt.ListAutoremovable()
		return autoremovableMsg{names: names, err: err}
	}
}

func autoremoveAllCmd(names []string) tea.Cmd {
	cmd := apt.AutoRemoveCmd()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "cleanup-all", name: strings.Join(names, " "), err: err}
	})
}

func listPPAsCmd() tea.Cmd {
	return func() tea.Msg {
		ppas, err := apt.ListPPAs()
		return ppaListMsg{ppas: ppas, err: err}
	}
}

func addPPACmd(ppa string) tea.Cmd {
	cmd := apt.AddPPACmd(ppa)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "ppa-add", name: ppa, err: err}
	})
}

func removePPACmd(ppa string) tea.Cmd {
	cmd := apt.RemovePPACmd(ppa)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "ppa-remove", name: ppa, err: err}
	})
}
