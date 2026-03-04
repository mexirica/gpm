package model

type Package struct {
	Name        string
	Version     string
	Size        string
	Description string
	Installed   bool
	Upgradable  bool
	NewVersion  string
}

type ViewState int

const (
	ViewBrowseInstalled ViewState = iota
	ViewBrowseAll
	ViewSearch
	ViewPackageDetail
	ViewUpgradable
)
