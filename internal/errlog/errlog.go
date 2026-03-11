// Package errlog provides centralized error logging for aptui.
// Errors are persisted to ~/.local/share/aptui/errors.json and can be
// viewed interactively in the TUI.
package errlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mexirica/aptui/internal/datadir"
)

type Entry struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

type Store struct {
	mu      sync.Mutex
	Entries []Entry `json:"entries"`
	NextID  int     `json:"next_id"`
	path    string
}

var logPath = func() string {
	return filepath.Join(datadir.Dir(), "errors.json")
}

func Load() *Store {
	p := logPath()
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
	return datadir.SaveJSON(s.path, s)
}

func (s *Store) Log(source string, message string) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Entry{
		ID:        s.NextID,
		Timestamp: time.Now(),
		Source:    source,
		Message:   message,
	}
	s.NextID++
	s.Entries = append(s.Entries, e)
	_ = s.save()
	return e
}

func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Entry, len(s.Entries))
	copy(out, s.Entries)
	// Reverse: newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Entries)
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Entries = nil
	s.NextID = 1
	_ = s.save()
}

func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
