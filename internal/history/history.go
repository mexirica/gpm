// Package history provides transaction history storage for GPM.
// Each install, remove, or upgrade operation is recorded with a unique ID.
// History is persisted to ~/.local/share/gpm/history.json.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Operation represents the type of package operation.
type Operation string

const (
	OpInstall    Operation = "install"
	OpRemove     Operation = "remove"
	OpUpgrade    Operation = "upgrade"
	OpUpgradeAll Operation = "upgrade-all"
)

// Transaction represents a single recorded operation.
type Transaction struct {
	ID        int       `json:"id"`
	Operation Operation `json:"operation"`
	Packages  []string  `json:"packages"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// Store manages the history file.
type Store struct {
	mu           sync.Mutex
	Transactions []Transaction `json:"transactions"`
	NextID       int           `json:"next_id"`
	path         string
}

func historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".local", "share", "gpm", "history.json")
}

// Load reads the history from disk (or returns an empty store).
func Load() *Store {
	p := historyPath()
	s := &Store{path: p, NextID: 1}

	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	s.path = p
	if s.NextID < 1 {
		s.NextID = 1
	}
	return s
}

// save writes the store to disk.
func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Record adds a new transaction and persists it.
func (s *Store) Record(op Operation, packages []string, success bool) Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Transaction{
		ID:        s.NextID,
		Operation: op,
		Packages:  packages,
		Timestamp: time.Now(),
		Success:   success,
	}
	s.NextID++
	s.Transactions = append(s.Transactions, t)
	_ = s.save()
	return t
}

// All returns all transactions sorted by ID descending (newest first).
func (s *Store) All() []Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Transaction, len(s.Transactions))
	copy(out, s.Transactions)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID > out[j].ID
	})
	return out
}

// Get returns a transaction by ID.
func (s *Store) Get(id int) (Transaction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.Transactions {
		if t.ID == id {
			return t, true
		}
	}
	return Transaction{}, false
}

// UndoOperation returns the inverse operation for a transaction.
// Install -> remove, Remove -> install, Upgrade cannot be truly undone but we allow reinstall.
func UndoOperation(op Operation) Operation {
	switch op {
	case OpInstall:
		return OpRemove
	case OpRemove:
		return OpInstall
	case OpUpgrade, OpUpgradeAll:
		return OpInstall
	}
	return OpInstall
}

// FormatTimestamp returns a human-friendly timestamp.
func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// Summary returns a short one-line description of the transaction.
func (t Transaction) Summary() string {
	status := "✔"
	if !t.Success {
		status = "✘"
	}
	pkgStr := ""
	if len(t.Packages) == 1 {
		pkgStr = t.Packages[0]
	} else {
		pkgStr = fmt.Sprintf("%d packages", len(t.Packages))
	}
	return fmt.Sprintf("%s  %s %s", status, t.Operation, pkgStr)
}
