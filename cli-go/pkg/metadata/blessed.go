// Package metadata provides management for polis public metadata files.
package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// BlessedCommentsFilename is the name of the blessed comments index file.
	BlessedCommentsFilename = "blessed-comments.json"
)

// BlessedComments represents the blessed-comments.json file structure.
// This file is the public index of comments that the site owner has blessed,
// grouped by the post they're replying to.
type BlessedComments struct {
	Version  string         `json:"version"`
	Comments []PostComments `json:"comments"`
}

// PostComments groups blessed comments for a single post.
type PostComments struct {
	Post    string           `json:"post"`
	Blessed []BlessedComment `json:"blessed"`
}

// BlessedComment represents a single blessed comment entry.
type BlessedComment struct {
	URL       string `json:"url"`
	Version   string `json:"version"`
	BlessedAt string `json:"blessed_at"`
}

// LoadBlessedComments reads the blessed-comments.json file from the metadata directory.
// Returns an error if the file doesn't exist.
func LoadBlessedComments(siteDir string) (*BlessedComments, error) {
	filePath := filepath.Join(siteDir, "metadata", BlessedCommentsFilename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read blessed-comments.json: %w", err)
	}

	var bc BlessedComments
	if err := json.Unmarshal(data, &bc); err != nil {
		return nil, fmt.Errorf("failed to parse blessed-comments.json: %w", err)
	}

	return &bc, nil
}

// SaveBlessedComments writes the blessed-comments.json file atomically.
// It writes to a temporary file first, then renames to ensure atomic update.
func SaveBlessedComments(siteDir string, bc *BlessedComments) error {
	metadataDir := filepath.Join(siteDir, "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blessed comments: %w", err)
	}

	// Write atomically via temp file
	filePath := filepath.Join(metadataDir, BlessedCommentsFilename)
	tmpPath := filePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// InitBlessedComments creates an empty blessed-comments.json if it doesn't exist.
// Returns nil if the file already exists (does not overwrite).
func InitBlessedComments(siteDir string, version string) error {
	filePath := filepath.Join(siteDir, "metadata", BlessedCommentsFilename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return nil // Already exists, don't overwrite
	}

	// Create metadata directory if needed
	metadataDir := filepath.Join(siteDir, "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Create empty structure
	bc := &BlessedComments{
		Version:  version,
		Comments: []PostComments{},
	}

	return SaveBlessedComments(siteDir, bc)
}

// AddBlessedComment adds a comment to the blessed comments index.
// Creates the post entry if it doesn't exist.
// This is an atomic read-modify-write operation.
func AddBlessedComment(siteDir string, postPath string, comment BlessedComment) error {
	// Load current state
	bc, err := LoadBlessedComments(siteDir)
	if err != nil {
		// If file doesn't exist, create new structure
		if errors.Is(err, os.ErrNotExist) {
			bc = &BlessedComments{
				Version:  "0.42.0",
				Comments: []PostComments{},
			}
		} else {
			return err
		}
	}

	// Set blessed_at if not provided
	if comment.BlessedAt == "" {
		comment.BlessedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}

	// Find or create post entry
	found := false
	for i, pc := range bc.Comments {
		if pc.Post == postPath {
			// Check if comment already exists (by URL or version)
			for _, existing := range pc.Blessed {
				if existing.URL == comment.URL || existing.Version == comment.Version {
					// Already blessed, nothing to do
					return nil
				}
			}
			// Add to existing post entry
			bc.Comments[i].Blessed = append(bc.Comments[i].Blessed, comment)
			found = true
			break
		}
	}

	if !found {
		// Create new post entry
		bc.Comments = append(bc.Comments, PostComments{
			Post:    postPath,
			Blessed: []BlessedComment{comment},
		})
	}

	return SaveBlessedComments(siteDir, bc)
}

// RemoveBlessedComment removes a comment from the blessed comments index.
// Matches by URL.
func RemoveBlessedComment(siteDir string, commentURL string) error {
	bc, err := LoadBlessedComments(siteDir)
	if err != nil {
		return err
	}

	// Find and remove the comment
	for i, pc := range bc.Comments {
		for j, c := range pc.Blessed {
			if c.URL == commentURL {
				// Remove this comment
				bc.Comments[i].Blessed = append(pc.Blessed[:j], pc.Blessed[j+1:]...)

				// If post has no more blessed comments, remove the post entry
				if len(bc.Comments[i].Blessed) == 0 {
					bc.Comments = append(bc.Comments[:i], bc.Comments[i+1:]...)
				}

				return SaveBlessedComments(siteDir, bc)
			}
		}
	}

	// Comment not found, nothing to do
	return nil
}

// GetBlessedCommentsForPost returns all blessed comments for a specific post.
func GetBlessedCommentsForPost(siteDir string, postPath string) ([]BlessedComment, error) {
	bc, err := LoadBlessedComments(siteDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BlessedComment{}, nil
		}
		return nil, err
	}

	for _, pc := range bc.Comments {
		if pc.Post == postPath {
			return pc.Blessed, nil
		}
	}

	return []BlessedComment{}, nil
}

// IsBlessedComment checks if a comment URL is in the blessed index.
func IsBlessedComment(siteDir string, commentURL string) (bool, error) {
	bc, err := LoadBlessedComments(siteDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	for _, pc := range bc.Comments {
		for _, c := range pc.Blessed {
			if c.URL == commentURL {
				return true, nil
			}
		}
	}

	return false, nil
}
