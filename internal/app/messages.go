package app

import (
	"github.com/mexirica/aptui/internal/apt"
	"github.com/mexirica/aptui/internal/fetch"
	"github.com/mexirica/aptui/internal/model"
)

type initialLoadMsg struct {
	installed  []model.Package
	upgradable []model.Package
	err        error
}

type allNamesMsg struct {
	names []string
	err   error
}

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

type depsLoadedMsg struct {
	txIdx int
	deps  []string
}
