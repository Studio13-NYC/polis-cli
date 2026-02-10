package stream

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/testsite", "example.supabase.co")
	expected := filepath.Join("/tmp/testsite", ".polis", "projections", "example.supabase.co")
	if s.Dir() != expected {
		t.Errorf("Dir() = %q, want %q", s.Dir(), expected)
	}
}

func TestCursors(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, "test.supabase.co")

	// Default cursor is "0"
	cursor, err := s.GetCursor("polis.follow")
	if err != nil {
		t.Fatalf("GetCursor: %v", err)
	}
	if cursor != "0" {
		t.Errorf("default cursor = %q, want %q", cursor, "0")
	}

	// Set and get cursor
	if err := s.SetCursor("polis.follow", "4521"); err != nil {
		t.Fatalf("SetCursor: %v", err)
	}
	cursor, err = s.GetCursor("polis.follow")
	if err != nil {
		t.Fatalf("GetCursor after set: %v", err)
	}
	if cursor != "4521" {
		t.Errorf("cursor = %q, want %q", cursor, "4521")
	}

	// Different projection has independent cursor
	cursor, err = s.GetCursor("polis.post")
	if err != nil {
		t.Fatalf("GetCursor other: %v", err)
	}
	if cursor != "0" {
		t.Errorf("other cursor = %q, want %q", cursor, "0")
	}

	// Set second cursor
	if err := s.SetCursor("polis.post", "4600"); err != nil {
		t.Fatalf("SetCursor post: %v", err)
	}

	// Verify both cursors persist independently
	cursor, _ = s.GetCursor("polis.follow")
	if cursor != "4521" {
		t.Errorf("follow cursor = %q, want %q", cursor, "4521")
	}
	cursor, _ = s.GetCursor("polis.post")
	if cursor != "4600" {
		t.Errorf("post cursor = %q, want %q", cursor, "4600")
	}
}

func TestState(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, "test.supabase.co")

	// Loading non-existent state should not error
	var state FollowerState
	if err := s.LoadState("polis.follow", &state); err != nil {
		t.Fatalf("LoadState non-existent: %v", err)
	}
	if state.Count != 0 {
		t.Errorf("empty state count = %d, want 0", state.Count)
	}

	// Save and load state
	saved := &FollowerState{
		Followers: []string{"alice.com", "bob.com"},
		Count:     2,
	}
	if err := s.SaveState("polis.follow", saved); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	var loaded FollowerState
	if err := s.LoadState("polis.follow", &loaded); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded.Count != 2 {
		t.Errorf("loaded count = %d, want 2", loaded.Count)
	}
	if len(loaded.Followers) != 2 {
		t.Errorf("loaded followers len = %d, want 2", len(loaded.Followers))
	}

	// Verify self-contained projection file exists
	projFile := filepath.Join(s.Dir(), "polis.follow.json")
	if _, err := os.Stat(projFile); os.IsNotExist(err) {
		t.Errorf("projection file %q does not exist", projFile)
	}
}

func TestManifest(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, "test.supabase.co")

	// Loading non-existent manifest returns empty
	m, err := s.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest non-existent: %v", err)
	}
	if len(m.Projections) != 0 {
		t.Errorf("empty manifest projections = %d, want 0", len(m.Projections))
	}

	// Save and load manifest
	m.Projections["polis.follow"] = ProjectionEntry{
		Handler:     "builtin:follow",
		EventTypes:  []string{"polis.follow.announced", "polis.follow.removed"},
		StateFile:   "polis.follow.json",
		LastUpdated: "2026-02-08T14:30:00Z",
	}
	if err := s.SaveManifest(m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := s.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest after save: %v", err)
	}
	if len(loaded.Projections) != 1 {
		t.Fatalf("loaded projections = %d, want 1", len(loaded.Projections))
	}
	entry := loaded.Projections["polis.follow"]
	if entry.Handler != "builtin:follow" {
		t.Errorf("handler = %q, want %q", entry.Handler, "builtin:follow")
	}
	if len(entry.EventTypes) != 2 {
		t.Errorf("event types len = %d, want 2", len(entry.EventTypes))
	}

	// Verify manifest file on disk
	mFile := filepath.Join(s.Dir(), manifestFile)
	data, err := os.ReadFile(mFile)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}
	var diskManifest ProjectionManifest
	if err := json.Unmarshal(data, &diskManifest); err != nil {
		t.Fatalf("parse manifest file: %v", err)
	}
	if len(diskManifest.Projections) != 1 {
		t.Errorf("disk manifest projections = %d, want 1", len(diskManifest.Projections))
	}
}

func TestSelfContainedProjectionFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, "test.supabase.co")

	// Set cursor first
	s.SetCursor("polis.follow", "100")

	// Save state — should preserve cursor
	saved := &FollowerState{
		Followers: []string{"alice.com"},
		Count:     1,
	}
	s.SaveState("polis.follow", saved)

	// Read raw file and verify self-contained format
	data, err := os.ReadFile(filepath.Join(s.Dir(), "polis.follow.json"))
	if err != nil {
		t.Fatalf("read projection file: %v", err)
	}

	var raw ProjectionState
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse projection file: %v", err)
	}
	if raw.Cursor != "100" {
		t.Errorf("cursor = %q, want %q", raw.Cursor, "100")
	}
	if raw.LastUpdated == "" {
		t.Error("last_updated should be set")
	}
	if raw.State == nil {
		t.Error("state should not be nil")
	}

	// Verify cursor is still readable
	cursor, _ := s.GetCursor("polis.follow")
	if cursor != "100" {
		t.Errorf("GetCursor after SaveState = %q, want %q", cursor, "100")
	}
}
