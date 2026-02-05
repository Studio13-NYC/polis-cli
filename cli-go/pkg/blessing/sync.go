package blessing

import (
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/metadata"
)

// SyncResult contains the result of syncing blessed comments.
type SyncResult struct {
	Synced   int `json:"synced"`
	Existing int `json:"existing"`
	Total    int `json:"total"`
}

// SyncBlessedComments syncs auto-blessed comments from the discovery service to local storage.
// This ensures that comments that were auto-blessed (e.g., from followed authors)
// are reflected in the local blessed-comments.json.
func SyncBlessedComments(siteDir, domain string, client *discovery.Client) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch all blessed comments for this domain from discovery service
	blessedComments, err := client.GetBlessedComments(domain)
	if err != nil {
		return nil, err
	}

	result.Total = len(blessedComments)

	// Load current local blessed comments
	blessedFile, err := metadata.LoadBlessedComments(siteDir)
	if err != nil {
		return nil, err
	}

	// Track which comment URLs we already have locally
	existingURLs := make(map[string]bool)
	for _, post := range blessedFile.Comments {
		for _, bc := range post.Blessed {
			existingURLs[bc.URL] = true
		}
	}

	// Add any missing blessed comments
	for _, comment := range blessedComments {
		if existingURLs[comment.CommentURL] {
			result.Existing++
			continue
		}

		// Add to local index
		bc := metadata.BlessedComment{
			URL:     comment.CommentURL,
			Version: comment.CommentVersion,
		}

		postPath := extractPostPath(comment.InReplyTo)
		if err := metadata.AddBlessedComment(siteDir, postPath, bc); err != nil {
			// Log but continue
			continue
		}

		result.Synced++
	}

	return result, nil
}

// GetBlessedCommentsForDomain fetches all blessed comments for a domain.
func GetBlessedCommentsForDomain(domain string, client *discovery.Client) ([]discovery.Comment, error) {
	return client.GetBlessedComments(domain)
}

// GetCommentsByAuthor fetches comments by author with optional status filter.
func GetCommentsByAuthor(authorEmail, status string, client *discovery.Client) ([]discovery.Comment, error) {
	return client.GetCommentsByAuthor(authorEmail, status)
}
