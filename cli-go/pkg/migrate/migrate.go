// Package migrate provides domain migration functionality.
package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Migration represents a domain migration record.
type Migration struct {
	OldDomain   string `json:"old_domain"`
	NewDomain   string `json:"new_domain"`
	MigratedAt  string `json:"migrated_at"`
	PublicKey   string `json:"public_key"`
}

// MigrationResult contains the results of applying a migration.
type MigrationResult struct {
	OldDomain    string   `json:"old_domain"`
	NewDomain    string   `json:"new_domain"`
	FilesUpdated []string `json:"files_updated"`
}

// ApplyMigrationOptions configures migration application.
type ApplyMigrationOptions struct {
	FollowingPath       string
	BlessedCommentsPath string
	CommentsDir         string
}

// ApplyMigration applies a domain migration to local files.
func ApplyMigration(migration Migration, opts ApplyMigrationOptions) (*MigrationResult, error) {
	result := &MigrationResult{
		OldDomain:    migration.OldDomain,
		NewDomain:    migration.NewDomain,
		FilesUpdated: []string{},
	}

	// Update following.json
	if opts.FollowingPath != "" {
		updated, err := updateFollowingFile(opts.FollowingPath, migration.OldDomain, migration.NewDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to update following.json: %w", err)
		}
		if updated {
			result.FilesUpdated = append(result.FilesUpdated, opts.FollowingPath)
		}
	}

	// Update blessed-comments.json
	if opts.BlessedCommentsPath != "" {
		updated, err := updateBlessedCommentsFile(opts.BlessedCommentsPath, migration.OldDomain, migration.NewDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to update blessed-comments.json: %w", err)
		}
		if updated {
			result.FilesUpdated = append(result.FilesUpdated, opts.BlessedCommentsPath)
		}
	}

	// Update comment files
	if opts.CommentsDir != "" {
		files, err := updateCommentFiles(opts.CommentsDir, migration.OldDomain, migration.NewDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to update comment files: %w", err)
		}
		result.FilesUpdated = append(result.FilesUpdated, files...)
	}

	return result, nil
}

// updateFollowingFile updates domain references in following.json.
func updateFollowingFile(path, oldDomain, newDomain string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	oldURL := "https://" + oldDomain
	newURL := "https://" + newDomain

	if !strings.Contains(string(data), oldURL) {
		return false, nil
	}

	updated := strings.ReplaceAll(string(data), oldURL, newURL)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return false, err
	}

	return true, nil
}

// updateBlessedCommentsFile updates domain references in blessed-comments.json.
func updateBlessedCommentsFile(path, oldDomain, newDomain string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	oldURL := "https://" + oldDomain
	newURL := "https://" + newDomain

	if !strings.Contains(string(data), oldURL) {
		return false, nil
	}

	updated := strings.ReplaceAll(string(data), oldURL, newURL)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return false, err
	}

	return true, nil
}

// updateCommentFiles updates domain references in comment frontmatter.
func updateCommentFiles(commentsDir, oldDomain, newDomain string) ([]string, error) {
	var updatedFiles []string

	err := filepath.Walk(commentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		oldURL := "https://" + oldDomain
		newURL := "https://" + newDomain

		if !strings.Contains(string(data), oldURL) {
			return nil
		}

		updated := strings.ReplaceAll(string(data), oldURL, newURL)
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			return nil
		}

		updatedFiles = append(updatedFiles, path)
		return nil
	})

	return updatedFiles, err
}

// CollectRelevantDomains collects all domains referenced in local files.
func CollectRelevantDomains(dataDir string) ([]string, error) {
	domainSet := make(map[string]bool)

	// Check following.json
	followingPath := filepath.Join(dataDir, "metadata", "following.json")
	if data, err := os.ReadFile(followingPath); err == nil {
		var f struct {
			Following []struct {
				URL string `json:"url"`
			} `json:"following"`
		}
		if json.Unmarshal(data, &f) == nil {
			for _, entry := range f.Following {
				if domain := extractDomain(entry.URL); domain != "" {
					domainSet[domain] = true
				}
			}
		}
	}

	// Check blessed-comments.json
	blessedPath := filepath.Join(dataDir, "metadata", "blessed-comments.json")
	if data, err := os.ReadFile(blessedPath); err == nil {
		var bc struct {
			Comments []struct {
				Post    string `json:"post"`
				Blessed []struct {
					URL string `json:"url"`
				} `json:"blessed"`
			} `json:"comments"`
		}
		if json.Unmarshal(data, &bc) == nil {
			for _, post := range bc.Comments {
				if domain := extractDomain(post.Post); domain != "" {
					domainSet[domain] = true
				}
				for _, comment := range post.Blessed {
					if domain := extractDomain(comment.URL); domain != "" {
						domainSet[domain] = true
					}
				}
			}
		}
	}

	// Check comment files for in-reply-to URLs
	commentsDir := filepath.Join(dataDir, "comments")
	filepath.Walk(commentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Look for https:// URLs in frontmatter
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "https://") {
				start := strings.Index(line, "https://")
				rest := line[start+8:]
				if idx := strings.IndexAny(rest, "/ \t\n"); idx > 0 {
					domainSet[rest[:idx]] = true
				}
			}
		}
		return nil
	})

	var domains []string
	for domain := range domainSet {
		domains = append(domains, domain)
	}

	return domains, nil
}

// extractDomain extracts domain from a URL.
func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if idx := strings.Index(url, "/"); idx > 0 {
		return url[:idx]
	}
	return url
}

// FindAffectedFiles finds files that reference a specific domain.
func FindAffectedFiles(dataDir, domain string) ([]string, error) {
	var affected []string
	oldURL := "https://" + domain

	// Check following.json
	followingPath := filepath.Join(dataDir, "metadata", "following.json")
	if data, err := os.ReadFile(followingPath); err == nil {
		if strings.Contains(string(data), oldURL) {
			affected = append(affected, followingPath)
		}
	}

	// Check blessed-comments.json
	blessedPath := filepath.Join(dataDir, "metadata", "blessed-comments.json")
	if data, err := os.ReadFile(blessedPath); err == nil {
		if strings.Contains(string(data), oldURL) {
			affected = append(affected, blessedPath)
		}
	}

	// Check comment files
	commentsDir := filepath.Join(dataDir, "comments")
	filepath.Walk(commentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), oldURL) {
			affected = append(affected, path)
		}
		return nil
	})

	return affected, nil
}
