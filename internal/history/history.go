// Package history provides transaction history storage for aptui.
// Each install, remove, or upgrade operation is recorded with a unique ID.
// History is persisted to ~/.local/share/aptui/history.json.
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

type Operation string

const (
	OpInstall    Operation = "install"
	OpRemove     Operation = "remove"
	OpUpgrade    Operation = "upgrade"
	OpUpgradeAll Operation = "upgrade-all"
)

type Transaction struct {
	ID        int       `json:"id"`
	Operation Operation `json:"operation"`
	Packages  []string  `json:"packages"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

type Store struct {
	mu           sync.Mutex
	Transactions []Transaction `json:"transactions"`
	NextID       int           `json:"next_id"`
	path         string
}

// historyPath returns the path to the history file.
// It is a variable so tests can override it.
var historyPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".local", "share", "aptui", "history.json")
}

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

func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

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
