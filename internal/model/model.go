package model

type Package struct {
	Name           string
	Version        string
	Size           string
	Description    string
	Installed      bool
	Upgradable     bool
	NewVersion     string
	Section        string
	Architecture   string
	SecurityUpdate bool
	Held           bool
	Pinned         bool
	Essential      bool
}
