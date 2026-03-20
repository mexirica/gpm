package pin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToggleAndSet(t *testing.T) {
	dir := t.TempDir()
	orig := pinPath
	pinPath = func() string { return filepath.Join(dir, "pins.json") }
	defer func() { pinPath = orig }()

	s := Load()

	// Initially empty
	set := s.Set()
	if len(set) != 0 {
		t.Fatalf("expected empty set, got %d", len(set))
	}

	// Pin a package
	if !s.Toggle("vim") {
		t.Fatal("expected Toggle to return true (pinned)")
	}
	if !s.IsPinned("vim") {
		t.Fatal("expected vim to be pinned")
	}

	// Pin another
	s.Toggle("git")
	set = s.Set()
	if len(set) != 2 {
		t.Fatalf("expected 2 pinned, got %d", len(set))
	}

	// Unpin
	if s.Toggle("vim") {
		t.Fatal("expected Toggle to return false (unpinned)")
	}
	if s.IsPinned("vim") {
		t.Fatal("expected vim to not be pinned")
	}

	set = s.Set()
	if len(set) != 1 || !set["git"] {
		t.Fatal("expected only git to be pinned")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	orig := pinPath
	pinPath = func() string { return filepath.Join(dir, "pins.json") }
	defer func() { pinPath = orig }()

	s := Load()
	s.Toggle("curl")
	s.Toggle("wget")

	// Reload from disk
	s2 := Load()
	set := s2.Set()
	if !set["curl"] || !set["wget"] {
		t.Fatal("expected pinned packages to persist")
	}
}

func TestLoadMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	orig := pinPath
	pinPath = func() string { return filepath.Join(dir, "pins.json") }
	defer func() { pinPath = orig }()

	// Write malformed JSON
	if err := os.WriteFile(filepath.Join(dir, "pins.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := Load()
	if len(s.Packages) != 0 {
		t.Fatalf("expected empty packages for malformed file, got %d", len(s.Packages))
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	orig := pinPath
	pinPath = func() string { return filepath.Join(dir, "nonexistent", "pins.json") }
	defer func() { pinPath = orig }()

	s := Load()
	if len(s.Packages) != 0 {
		t.Fatal("expected empty packages for missing file")
	}

	// Ensure directory is created on save
	s.Toggle("test")
	if _, err := os.Stat(filepath.Join(dir, "nonexistent", "pins.json")); os.IsNotExist(err) {
		t.Fatal("expected pins.json to be created")
	}
}
