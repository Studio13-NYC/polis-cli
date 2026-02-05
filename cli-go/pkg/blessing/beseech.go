package blessing

import (
	"fmt"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// BeseechRequest contains the data needed to request a blessing.
type BeseechRequest struct {
	CommentURL       string
	CommentVersion   string
	InReplyTo        string
	InReplyToVersion string
	RootPost         string
	Author           string
}

// BeseechResult contains the result of a blessing request.
type BeseechResult struct {
	Success  bool   `json:"success"`
	Status   string `json:"status"` // "pending" or "blessed" (auto-blessed)
	Message  string `json:"message,omitempty"`
}

// Beseech sends a blessing request to the discovery service.
// This is called when publishing a comment - the author requests the post owner to bless it.
func Beseech(req *BeseechRequest, client *discovery.Client, privateKey []byte) (*BeseechResult, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Create canonical JSON for signing
	canonicalJSON, err := discovery.MakeBeseechCanonicalJSON(
		req.CommentURL,
		req.CommentVersion,
		req.InReplyTo,
		req.InReplyToVersion,
		req.RootPost,
		req.Author,
		timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical JSON: %w", err)
	}

	// Sign the canonical JSON
	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send the beseech request
	beseechReq := &discovery.BeseechRequest{
		CommentURL:       req.CommentURL,
		CommentVersion:   req.CommentVersion,
		InReplyTo:        req.InReplyTo,
		InReplyToVersion: req.InReplyToVersion,
		RootPost:         req.RootPost,
		Author:           req.Author,
		Timestamp:        timestamp,
		Signature:        signature,
	}

	resp, err := client.BeseechBlessing(beseechReq)
	if err != nil {
		return nil, fmt.Errorf("beseech request failed: %w", err)
	}

	return &BeseechResult{
		Success: resp.Success,
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

// ReBeseech re-sends a blessing request for an existing comment.
// This is useful when a previous request was denied and you want to try again.
func ReBeseech(commentVersion string, client *discovery.Client, privateKey []byte) (*BeseechResult, error) {
	// First, look up the comment details from the discovery service
	status, err := client.CheckBlessingStatus(commentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to check blessing status: %w", err)
	}

	// If already blessed, no need to re-beseech
	if status.BlessingStatus == "blessed" {
		return &BeseechResult{
			Success: true,
			Status:  "blessed",
			Message: "Comment is already blessed",
		}, nil
	}

	// For denied/pending comments, we need more details to re-beseech
	// This requires the full comment info which should be cached locally
	// For now, return an error indicating that re-beseech needs local data
	return nil, fmt.Errorf("re-beseech requires local comment data - use comment sync first")
}
