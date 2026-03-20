package portpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportAndImport(t *testing.T) {
	tmpDir := t.TempDir()
	orig := DefaultPath
	DefaultPath = func() string { return filepath.Join(tmpDir, "test-packages.json") }
	defer func() { DefaultPath = orig }()

	packages := []PackageEntry{
		{Name: "zsh"},
		{Name: "curl"},
		{Name: "git"},
	}

	path, err := Export(packages)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	entries, gotPath, err := Import("")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if gotPath != path {
		t.Errorf("path = %q, want %q", gotPath, path)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(entries))
	}
	want := []string{"curl", "git", "zsh"}
	for i, e := range entries {
		if e.Name != want[i] {
			t.Errorf("entries[%d].Name = %q, want %q", i, e.Name, want[i])
		}
	}
}

func TestExportSortsPackages(t *testing.T) {
	tmpDir := t.TempDir()
	orig := DefaultPath
	DefaultPath = func() string { return filepath.Join(tmpDir, "sorted.json") }
	defer func() { DefaultPath = orig }()

	packages := []PackageEntry{
		{Name: "zsh"},
		{Name: "apt"},
		{Name: "curl"},
	}
	path, err := Export(packages)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ef ExportFile
	if err := json.Unmarshal(data, &ef); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ef.Packages[0].Name != "apt" || ef.Packages[1].Name != "curl" || ef.Packages[2].Name != "zsh" {
		t.Errorf("packages not sorted: %+v", ef.Packages)
	}
}

func TestImportMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	orig := DefaultPath
	DefaultPath = func() string { return filepath.Join(tmpDir, "nonexistent.json") }
	defer func() { DefaultPath = orig }()

	_, _, err := Import("")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestImportInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := DefaultPath
	DefaultPath = func() string { return path }
	defer func() { DefaultPath = orig }()

	_, _, err := Import("")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestImportValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "valid.json")
	ef := ExportFile{Packages: []PackageEntry{
		{Name: "vim"},
		{Name: "tmux"},
	}}
	data, _ := json.MarshalIndent(ef, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	orig := DefaultPath
	DefaultPath = func() string { return path }
	defer func() { DefaultPath = orig }()

	entries, gotPath, err := Import("")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if gotPath != path {
		t.Errorf("path = %q, want %q", gotPath, path)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "vim" || entries[1].Name != "tmux" {
		t.Errorf("unexpected entries: %+v", entries)
	}
}
