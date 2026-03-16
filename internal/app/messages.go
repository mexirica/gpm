package app

import (
	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/model"
)

type allPackagesMsg struct {
	allNames   []string
	installed  []model.Package
	upgradable []model.Package
	err        error
}

type infoLoadedMsg struct {
	info map[string]apt.PackageInfo
}

type searchResultMsg struct {
	pkgs []model.Package
	err  error
}

type detailLoadedMsg struct {
	name string
	info string
	err  error
}

type execFinishedMsg struct {
	op   string
	name string
	err  error
}

type fetchMirrorsMsg struct {
	mirrors []fetch.Mirror
	distro  fetch.Distro
	err     error
}

type fetchTestResultMsg struct {
	result fetch.TestResult
	done   bool
}

type fetchApplyMsg struct {
	err error
}

type clearStatusMsg struct{}

type bulkInfoMsg struct {
	info map[string]apt.PackageInfo
}

type silentUpdateDoneMsg struct {
	names      []string
	upgradable []model.Package
}

type depsLoadedMsg struct {
	txIdx int
	deps  []string
}

type autoremovableMsg struct {
	names []string
	err   error
}

type ppaListMsg struct {
	ppas []apt.PPA
	err  error
}

type ppaToggleMsg struct {
	name   string
	action string
	err    error
}
