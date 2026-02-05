package following

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddAndRemove(t *testing.T) {
	f := &FollowingFile{
		Version:   "0.45.0",
		Following: []FollowingEntry{},
	}

	// Test Add
	added := f.Add("https://example.com")
	if !added {
		t.Error("Expected Add to return true for new entry")
	}

	if f.Count() != 1 {
		t.Errorf("Expected count 1, got %d", f.Count())
	}

	// Test duplicate Add
	added = f.Add("https://example.com")
	if added {
		t.Error("Expected Add to return false for duplicate entry")
	}

	// Test IsFollowing
	if !f.IsFollowing("https://example.com") {
		t.Error("Expected IsFollowing to return true")
	}

	if f.IsFollowing("https://other.com") {
		t.Error("Expected IsFollowing to return false for non-existent entry")
	}

	// Test Remove
	removed := f.Remove("https://example.com")
	if !removed {
		t.Error("Expected Remove to return true")
	}

	if f.Count() != 0 {
		t.Errorf("Expected count 0 after remove, got %d", f.Count())
	}

	// Test Remove non-existent
	removed = f.Remove("https://other.com")
	if removed {
		t.Error("Expected Remove to return false for non-existent entry")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "following.json")

	// Create and save
	f := &FollowingFile{
		Version:   "0.45.0",
		Following: []FollowingEntry{},
	}
	f.Add("https://example.com")

	if err := Save(path, f); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Load and verify
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Count() != 1 {
		t.Errorf("Expected count 1, got %d", loaded.Count())
	}

	if !loaded.IsFollowing("https://example.com") {
		t.Error("Expected loaded file to contain entry")
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Loading non-existent file should return empty following
	f, err := Load("/non/existent/path.json")
	if err != nil {
		t.Fatalf("Load of non-existent file failed: %v", err)
	}

	if f.Count() != 0 {
		t.Errorf("Expected count 0 for non-existent file, got %d", f.Count())
	}
}

func TestUpdateLastChecked(t *testing.T) {
	f := &FollowingFile{
		Version:   "0.45.0",
		Following: []FollowingEntry{},
	}
	f.Add("https://example.com")

	// Update last checked
	updated := f.UpdateLastChecked("https://example.com")
	if !updated {
		t.Error("Expected UpdateLastChecked to return true")
	}

	entry := f.Get("https://example.com")
	if entry == nil {
		t.Fatal("Expected to get entry")
	}

	if entry.LastChecked == "" {
		t.Error("Expected LastChecked to be set")
	}

	// Try updating non-existent
	updated = f.UpdateLastChecked("https://other.com")
	if updated {
		t.Error("Expected UpdateLastChecked to return false for non-existent entry")
	}
}
