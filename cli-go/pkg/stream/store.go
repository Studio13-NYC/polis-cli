// Package stream manages per-projection cursors, state, and config on disk.
//
// Root: .polis/ds/<discovery-service-domain>/
// Directories:
//
//	config/    — user preferences (notification rules, feed settings); survives resets
//	state/     — computed/derived data (followers, blessings, cursors, notifications, feed cache)
package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store manages per-projection cursors, state, and config on disk.
type Store struct {
	configDir string // .polis/ds/<domain>/config/
	stateDir  string // .polis/ds/<domain>/state/
	dsDir     string // .polis/ds/<domain>/
}

// NewStore creates a new Store rooted at dataDir/.polis/ds/discoveryDomain/.
func NewStore(dataDir string, discoveryDomain string) *Store {
	dsDir := filepath.Join(dataDir, ".polis", "ds", discoveryDomain)
	return &Store{
		configDir: filepath.Join(dsDir, "config"),
		stateDir:  filepath.Join(dsDir, "state"),
		dsDir:     dsDir,
	}
}

// DSDir returns the discovery service root directory (.polis/ds/<domain>/).
func (s *Store) DSDir() string {
	return s.dsDir
}

// StateDir returns the state directory (.polis/ds/<domain>/state/).
func (s *Store) StateDir() string {
	return s.stateDir
}

// ConfigDir returns the config directory (.polis/ds/<domain>/config/).
func (s *Store) ConfigDir() string {
	return s.configDir
}

// --- Cursor operations (consolidated in state/cursors.json) ---

// CursorEntry holds a single projection's cursor position and last update time.
type CursorEntry struct {
	Position    string `json:"position"`
	LastUpdated string `json:"last_updated"`
}

// CursorsFile is the on-disk format for state/cursors.json.
type CursorsFile struct {
	Cursors map[string]CursorEntry `json:"cursors"`
}

// GetCursor returns the cursor position for a projection, or "0" if not set.
func (s *Store) GetCursor(projection string) (string, error) {
	cf, err := s.loadCursors()
	if err != nil {
		return "0", nil
	}
	entry, ok := cf.Cursors[projection]
	if !ok || entry.Position == "" {
		return "0", nil
	}
	return entry.Position, nil
}

// SetCursor stores the cursor position for a projection.
func (s *Store) SetCursor(projection string, cursor string) error {
	cf, _ := s.loadCursors()
	if cf == nil {
		cf = &CursorsFile{Cursors: make(map[string]CursorEntry)}
	}
	cf.Cursors[projection] = CursorEntry{
		Position:    cursor,
		LastUpdated: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	return s.saveCursors(cf)
}

// GetCursorEntry returns the full cursor entry for a projection, or a zero entry if not set.
func (s *Store) GetCursorEntry(projection string) (CursorEntry, error) {
	cf, err := s.loadCursors()
	if err != nil {
		return CursorEntry{}, nil
	}
	entry, ok := cf.Cursors[projection]
	if !ok {
		return CursorEntry{}, nil
	}
	return entry, nil
}

func (s *Store) loadCursors() (*CursorsFile, error) {
	data, err := os.ReadFile(filepath.Join(s.stateDir, "cursors.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &CursorsFile{Cursors: make(map[string]CursorEntry)}, nil
		}
		return nil, fmt.Errorf("read cursors: %w", err)
	}
	var cf CursorsFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parse cursors: %w", err)
	}
	if cf.Cursors == nil {
		cf.Cursors = make(map[string]CursorEntry)
	}
	return &cf, nil
}

func (s *Store) saveCursors(cf *CursorsFile) error {
	if err := os.MkdirAll(s.stateDir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cursors: %w", err)
	}
	return os.WriteFile(filepath.Join(s.stateDir, "cursors.json"), data, 0644)
}

// --- State operations (state/<name>.json) ---

// LoadState loads state from state/<name>.json directly into target.
func (s *Store) LoadState(name string, target interface{}) error {
	data, err := os.ReadFile(filepath.Join(s.stateDir, name+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no state yet
		}
		return fmt.Errorf("read state %s: %w", name, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse state %s: %w", name, err)
	}
	return nil
}

// SaveState writes state directly to state/<name>.json.
func (s *Store) SaveState(name string, state interface{}) error {
	if err := os.MkdirAll(s.stateDir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(s.stateDir, name+".json"), data, 0644)
}

// --- Config operations (config/<name>.json) ---

// LoadConfig loads config from config/<name>.json into target.
func (s *Store) LoadConfig(name string, target interface{}) error {
	data, err := os.ReadFile(filepath.Join(s.configDir, name+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no config yet
		}
		return fmt.Errorf("read config %s: %w", name, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse config %s: %w", name, err)
	}
	return nil
}

// SaveConfig writes config to config/<name>.json.
func (s *Store) SaveConfig(name string, config interface{}) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(s.configDir, name+".json"), data, 0644)
}
