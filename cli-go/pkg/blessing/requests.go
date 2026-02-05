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
// These are comments from other users awaiting our blessing decision.
func FetchPendingRequests(client *discovery.Client, domain string) ([]IncomingRequest, error) {
	requests, err := client.GetPendingRequests(domain)
	if err != nil {
		return nil, err
	}

	// Convert discovery.BlessingRequest to IncomingRequest
	result := make([]IncomingRequest, len(requests))
	for i, r := range requests {
		result[i] = IncomingRequest{
			ID:             r.ID,
			CommentURL:     r.CommentURL,
			CommentVersion: r.CommentVersion,
			InReplyTo:      r.InReplyTo,
			RootPost:       r.RootPost,
			Author:         r.Author,
			Timestamp:      r.Timestamp,
			CreatedAt:      r.CreatedAt,
		}
	}

	return result, nil
}
