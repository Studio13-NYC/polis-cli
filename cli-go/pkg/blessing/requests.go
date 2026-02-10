// Package blessing provides blessing workflow management for polis.
// This handles incoming blessing requests (others' comments on my posts).
package blessing

import (
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
)

// IncomingRequest represents a blessing request from another user
// who wants their comment on our post to be blessed.
type IncomingRequest struct {
	ID             string `json:"id"`
	CommentURL     string `json:"comment_url"`
	CommentVersion string `json:"comment_version"`
	InReplyTo      string `json:"in_reply_to"`
	RootPost       string `json:"root_post"`
	Author         string `json:"author"`
	Timestamp      string `json:"timestamp"`
	CreatedAt      string `json:"created_at"`
}

// FetchPendingRequests retrieves all pending blessing requests for the given domain.
// Uses the unified relationship-query endpoint with status=pending.
func FetchPendingRequests(client *discovery.Client, domain string) ([]IncomingRequest, error) {
	resp, err := client.QueryRelationships("polis.blessing", map[string]string{
		"actor":  domain,
		"status": "pending",
	})
	if err != nil {
		return nil, err
	}

	// Convert RelationshipRecord to IncomingRequest
	result := make([]IncomingRequest, len(resp.Records))
	for i, r := range resp.Records {
		// Extract author from metadata if available
		author, _ := r.Metadata["author"].(string)

		result[i] = IncomingRequest{
			CommentURL: r.SourceURL,
			InReplyTo:  r.TargetURL,
			Author:     author,
			CreatedAt:  r.CreatedAt,
		}
	}

	return result, nil
}
