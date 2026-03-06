package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mexirica/gpm/internal/apt"
	"github.com/mexirica/gpm/internal/fetch"
	"github.com/mexirica/gpm/internal/model"
)

func loadInstalledAndUpgradable() tea.Msg {
	type result struct {
		pkgs []model.Package
		err  error
	}
	installedCh := make(chan result, 1)
	upgradableCh := make(chan result, 1)

	go func() {
		p, err := apt.ListInstalled()
		installedCh <- result{p, err}
	}()
	go func() {
		p, err := apt.ListUpgradable()
		upgradableCh <- result{p, err}
	}()

	ir := <-installedCh
	ur := <-upgradableCh

	if ir.err != nil {
		return initialLoadMsg{err: ir.err}
	}
	return initialLoadMsg{installed: ir.pkgs, upgradable: ur.pkgs}
}

func loadAllPackageNamesCmd() tea.Cmd {
	return func() tea.Msg {
		names, err := apt.ListAllNames()
		return allNamesMsg{names: names, err: err}
	}
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

func installPackageCmd(name string) tea.Cmd {
	cmd := apt.ParallelInstallCmd(name)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "install", name: name, err: err}
	})
}

func removePackageCmd(name string) tea.Cmd {
	return tea.ExecProcess(apt.RemoveCmd(name), func(err error) tea.Msg {
		return execFinishedMsg{op: "remove", name: name, err: err}
	})
}

func upgradePackageCmd(name string) tea.Cmd {
	cmd := apt.ParallelUpgradeCmd(name)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade", name: name, err: err}
	})
}

func upgradeAllPackagesCmd() tea.Cmd {
	cmd := apt.ParallelUpgradeAllCmd()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{op: "upgrade-all", name: "all", err: err}
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
