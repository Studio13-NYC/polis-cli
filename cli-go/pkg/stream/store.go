// Package stream manages per-projection cursors, state, and manifest on disk.
//
// Root: .polis/projections/<discovery-service-domain>/
// Files:
//
//	manifest.json                   — active projections registry
//	polis.follow.json               — self-contained: {cursor, last_updated, state}
//	polis.blessing.json             — self-contained: {cursor, last_updated, state}
//	polis.notification.json         — self-contained: {cursor, last_updated, state}
package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store manages per-projection cursors, state, and manifest on disk.
type Store struct {
	dir string // .polis/projections/<domain>/
}

// NewStore creates a new Store rooted at dataDir/.polis/projections/discoveryDomain/.
func NewStore(dataDir string, discoveryDomain string) *Store {
	return &Store{
		dir: filepath.Join(dataDir, ".polis", "projections", discoveryDomain),
	}
}

// Dir returns the store's root directory.
func (s *Store) Dir() string {
	return s.dir
}

// ensureDir creates the store directory if it doesn't exist.
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir, 0755)
}

// --- Self-contained projection state files ---

// ProjectionState is a self-contained projection file that embeds its cursor alongside state.
type ProjectionState struct {
	Cursor      string      `json:"cursor"`
	LastUpdated string      `json:"last_updated"`
	State       interface{} `json:"state"`
}

// projectionFileName returns the file name for a projection's self-contained state.
func projectionFileName(projection string) string {
	return projection + ".json"
}

// GetCursor returns the cursor position for a projection, or "0" if not set.
func (s *Store) GetCursor(projection string) (string, error) {
	ps, err := s.loadProjectionState(projection)
	if err != nil {
		return "0", nil
	}
	if ps.Cursor == "" {
		return "0", nil
	}
	return ps.Cursor, nil
}

// SetCursor stores the cursor position for a projection.
func (s *Store) SetCursor(projection string, cursor string) error {
	ps, _ := s.loadProjectionState(projection)
	if ps == nil {
		ps = &ProjectionState{}
	}
	ps.Cursor = cursor
	ps.LastUpdated = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	return s.saveProjectionState(projection, ps)
}

func (s *Store) loadProjectionState(projection string) (*ProjectionState, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, projectionFileName(projection)))
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectionState{Cursor: "0"}, nil
		}
		return nil, fmt.Errorf("read projection state for %s: %w", projection, err)
	}
	var ps ProjectionState
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("parse projection state for %s: %w", projection, err)
	}
	return &ps, nil
}

func (s *Store) saveProjectionState(projection string, ps *ProjectionState) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal projection state for %s: %w", projection, err)
	}
	return os.WriteFile(filepath.Join(s.dir, projectionFileName(projection)), data, 0644)
}

// --- Projection state operations ---

// LoadState loads the materialized state for a projection into target.
// Reads from the self-contained projection file's .state field.
func (s *Store) LoadState(projection string, target interface{}) error {
	ps, err := s.loadProjectionState(projection)
	if err != nil {
		return err
	}
	if ps.State == nil {
		return nil // no state yet
	}

	// Re-marshal the state field and unmarshal into target
	stateJSON, err := json.Marshal(ps.State)
	if err != nil {
		return fmt.Errorf("re-marshal state for %s: %w", projection, err)
	}
	if err := json.Unmarshal(stateJSON, target); err != nil {
		return fmt.Errorf("parse state for %s: %w", projection, err)
	}
	return nil
}

// SaveState writes the materialized state for a projection.
// Updates the state field in the self-contained projection file.
func (s *Store) SaveState(projection string, state interface{}) error {
	ps, _ := s.loadProjectionState(projection)
	if ps == nil {
		ps = &ProjectionState{Cursor: "0"}
	}
	ps.State = state
	ps.LastUpdated = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	return s.saveProjectionState(projection, ps)
}

// --- Manifest operations ---

const manifestFile = "manifest.json"

// ProjectionManifest declares what projections are active.
type ProjectionManifest struct {
	Projections map[string]ProjectionEntry `json:"projections"`
}

// ProjectionEntry describes a single active projection.
type ProjectionEntry struct {
	Handler     string   `json:"handler"`
	EventTypes  []string `json:"event_types"`
	StateFile   string   `json:"state_file"`
	LastUpdated string   `json:"last_updated"`
}

// LoadManifest loads the projection manifest from disk.
// Returns an empty manifest if the file doesn't exist.
func (s *Store) LoadManifest() (*ProjectionManifest, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, manifestFile))
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectionManifest{
				Projections: make(map[string]ProjectionEntry),
			}, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m ProjectionManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Projections == nil {
		m.Projections = make(map[string]ProjectionEntry)
	}
	return &m, nil
}

// SaveManifest writes the projection manifest to disk.
func (s *Store) SaveManifest(m *ProjectionManifest) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(s.dir, manifestFile), data, 0644)
}
