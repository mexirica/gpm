// Package datadir centralizes data directory resolution and file persistence for aptui.
// It handles the SUDO_USER case so files always go to the real user's home,
// and fixes ownership after writing when running under sudo.
package datadir

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

// Dir returns the aptui data directory (~/.local/share/aptui) for the real user.
func Dir() string {
	return filepath.Join(RealUserHome(), ".local", "share", "aptui")
}

// RealUserHome returns the home directory of the real user,
// even when running under sudo.
func RealUserHome() string {
	if u := os.Getenv("SUDO_USER"); u != "" {
		if lu, err := user.Lookup(u); err == nil {
			return lu.HomeDir
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// SaveJSON marshals v as indented JSON and writes it to path,
// creating parent directories as needed. When running under sudo,
// it chowns the directory and file to the real user.
func SaveJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fixOwnership(dir, path)
	return nil
}

func fixOwnership(paths ...string) {
	u := os.Getenv("SUDO_USER")
	if u == "" {
		return
	}
	lu, err := user.Lookup(u)
	if err != nil {
		return
	}
	uid, uidErr := strconv.Atoi(lu.Uid)
	gid, gidErr := strconv.Atoi(lu.Gid)
	if uidErr != nil || gidErr != nil {
		return
	}
	for _, p := range paths {
		_ = os.Chown(p, uid, gid)
	}
}
