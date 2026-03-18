// Package pin provides persistent storage for pinned (favorite) packages.
// Pinned packages are persisted to ~/.local/share/aptui/pins.json.
package pin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/mexirica/aptui/internal/datadir"
)

type Store struct {
	mu       sync.Mutex
	Packages []string `json:"packages"`
	path     string
}

var pinPath = func() string {
	return filepath.Join(datadir.Dir(), "pins.json")
}

func Load() *Store {
	p := pinPath()
	s := &Store{path: p}

	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, s); err != nil {
		return s
	}
	s.path = p
	return s
}

func (s *Store) save() error {
	return datadir.SaveJSON(s.path, s)
}

// Set returns the pinned package names as a map for O(1) lookup.
func (s *Store) Set() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := make(map[string]bool, len(s.Packages))
	for _, name := range s.Packages {
		m[name] = true
	}
	return m
}

// Toggle adds or removes a package from the pin list.
// Returns true if the package is now pinned, false if unpinned.
func (s *Store) Toggle(name string) bool {
	s.mu.Lock()
	for i, p := range s.Packages {
		if p == name {
			s.Packages = append(s.Packages[:i], s.Packages[i+1:]...)
			s.mu.Unlock()
			_ = s.save()
			return false
		}
	}
	s.Packages = append(s.Packages, name)
	s.mu.Unlock()
	_ = s.save()
	return true
}

// IsPinned returns whether a package is pinned.
func (s *Store) IsPinned(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.Packages {
		if p == name {
			return true
		}
	}
	return false
}
