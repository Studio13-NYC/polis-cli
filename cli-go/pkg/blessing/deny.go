package blessing

import (
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
)

// DenyResult contains the result of denying a blessing.
type DenyResult struct {
	Success        bool   `json:"success"`
	CommentVersion string `json:"comment_version"`
}

// Deny rejects a blessing request.
// This calls the discovery service to deny the blessing (with signed payload).
// No local state changes are needed for denials.
func Deny(commentVersion string, client *discovery.Client, privateKey []byte) (*DenyResult, error) {
	if err := client.DenyBlessing(commentVersion, privateKey); err != nil {
		return nil, fmt.Errorf("failed to deny blessing: %w", err)
	}

	return &DenyResult{
		Success:        true,
		CommentVersion: commentVersion,
	}, nil
}

// DenyRequest denies a blessing request using the full request object.
func DenyRequest(request *IncomingRequest, client *discovery.Client, privateKey []byte) (*DenyResult, error) {
	return Deny(request.CommentVersion, client, privateKey)
}
