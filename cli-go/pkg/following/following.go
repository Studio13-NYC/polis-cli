// Package following manages the following.json file for tracking followed authors.
package following

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FollowingFile represents the following.json structure.
type FollowingFile struct {
	Version   string           `json:"version"`
	Following []FollowingEntry `json:"following"`
}

// FollowingEntry represents a single followed author.
type FollowingEntry struct {
	URL         string `json:"url"`
	AddedAt     string `json:"added_at"`
	LastChecked string `json:"last_checked,omitempty"`
}

// DefaultPath returns the default path to following.json.
func DefaultPath(dataDir string) string {
	return filepath.Join(dataDir, "metadata", "following.json")
}

// Load loads the following.json file from the given path.
func Load(path string) (*FollowingFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FollowingFile{
				Version:   "0.45.0",
				Following: []FollowingEntry{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read following.json: %w", err)
	}

	var f FollowingFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("failed to parse following.json: %w", err)
	}

	return &f, nil
}

// Save saves the following.json file to the given path.
func Save(path string, f *FollowingFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal following.json: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write following.json: %w", err)
	}

	return nil
}

// Add adds an author to the following list.
func (f *FollowingFile) Add(authorURL string) bool {
	// Check if already following
	for _, entry := range f.Following {
		if entry.URL == authorURL {
			return false // Already following
		}
	}

	f.Following = append(f.Following, FollowingEntry{
		URL:     authorURL,
		AddedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	})

	return true
}

// Remove removes an author from the following list.
func (f *FollowingFile) Remove(authorURL string) bool {
	for i, entry := range f.Following {
		if entry.URL == authorURL {
			f.Following = append(f.Following[:i], f.Following[i+1:]...)
			return true
		}
	}
	return false // Not found
}

// IsFollowing checks if an author is in the following list.
func (f *FollowingFile) IsFollowing(authorURL string) bool {
	for _, entry := range f.Following {
		if entry.URL == authorURL {
			return true
		}
	}
	return false
}

// Get retrieves a following entry by URL.
func (f *FollowingFile) Get(authorURL string) *FollowingEntry {
	for i := range f.Following {
		if f.Following[i].URL == authorURL {
			return &f.Following[i]
		}
	}
	return nil
}

// UpdateLastChecked updates the last_checked timestamp for an author.
func (f *FollowingFile) UpdateLastChecked(authorURL string) bool {
	for i := range f.Following {
		if f.Following[i].URL == authorURL {
			f.Following[i].LastChecked = time.Now().UTC().Format("2006-01-02T15:04:05Z")
			return true
		}
	}
	return false
}

// Count returns the number of followed authors.
func (f *FollowingFile) Count() int {
	return len(f.Following)
}

// All returns all following entries.
func (f *FollowingFile) All() []FollowingEntry {
	return f.Following
}
