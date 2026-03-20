// Package portpkg implements export and import of installed package lists.
package portpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mexirica/aptui/internal/datadir"
)

// PackageEntry represents one package in an export file.
type PackageEntry struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ExportFile is the on-disk JSON format.
type ExportFile struct {
	Packages []PackageEntry `json:"packages"`
}

// DefaultPath returns the default export file location.
var DefaultPath = func() string {
	return filepath.Join(datadir.RealUserHome(), "aptui-packages.json")
}

// Export writes the given package names and versions to the default JSON file.
func Export(packages []PackageEntry) (string, error) {
	sorted := make([]PackageEntry, len(packages))
	copy(sorted, packages)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	path := DefaultPath()
	ef := ExportFile{Packages: sorted}
	return path, datadir.SaveJSON(path, ef)
}

// Import reads the package list from the given path.
// If path is empty, the default path is used.
func Import(path string) ([]PackageEntry, string, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	var ef ExportFile
	if err := json.Unmarshal(data, &ef); err != nil {
		return nil, path, err
	}
	return ef.Packages, path, nil
}
