package comment

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/hooks"
)

// SyncResult contains the results of syncing pending comments.
type SyncResult struct {
	Blessed      []string `json:"blessed"`
	Denied       []string `json:"denied"`
	StillPending []string `json:"still_pending"`
	Errors       []string `json:"errors"`
}

// SyncPendingComments checks all pending comments with the discovery service
// and moves them to blessed/denied based on their status.
// It also runs the post-comment hook when a comment is blessed.
// Pending/denied comments are in .polis/comments/, blessed go to public comments/YYYY/MM/.
func SyncPendingComments(dataDir string, discoveryClient *discovery.Client, hookConfig *hooks.HookConfig) (*SyncResult, error) {
	result := &SyncResult{
		Blessed:      []string{},
		Denied:       []string{},
		StillPending: []string{},
		Errors:       []string{},
	}

	pendingDir := filepath.Join(dataDir, ".polis", "comments", StatusPending)
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // No pending directory, nothing to sync
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		commentID := strings.TrimSuffix(entry.Name(), ".md")

		// Read comment to get version and URL
		commentPath := filepath.Join(pendingDir, entry.Name())
		data, err := os.ReadFile(commentPath)
		if err != nil {
			result.Errors = append(result.Errors, "failed to read "+commentID+": "+err.Error())
			continue
		}

		fm := ParseFrontmatter(string(data))
		commentURL := fm["comment_url"]
		inReplyTo := fm["in_reply_to"]
		if commentURL == "" || inReplyTo == "" {
			result.Errors = append(result.Errors, commentID+": missing comment_url or in_reply_to")
			continue
		}

		// Check blessing status via relationship-query
		resp, err := discoveryClient.QueryRelationships("polis.blessing", map[string]string{
			"source_url": commentURL,
			"target_url": inReplyTo,
		})
		if err != nil {
			result.Errors = append(result.Errors, "failed to check "+commentID+": "+err.Error())
			continue
		}

		// Determine status from the relationship record
		status := "pending"
		if resp != nil && len(resp.Records) > 0 {
			status = resp.Records[0].Status
		}

		switch status {
		case "granted":
			// Move to blessed directory (public comments/YYYY/MM/)
			if err := MoveComment(dataDir, commentID, StatusPending, StatusBlessed); err != nil {
				result.Errors = append(result.Errors, "failed to move "+commentID+" to blessed: "+err.Error())
				continue
			}
			result.Blessed = append(result.Blessed, commentID)

			// Run post-comment hook
			if hookConfig != nil {
				// Determine the blessed comment path based on timestamp
				timestamp := time.Now().UTC()
				if ts := fm["timestamp"]; ts != "" {
					if parsed, err := time.Parse("2006-01-02T15:04:05Z", ts); err == nil {
						timestamp = parsed
					}
				}
				dateDir := timestamp.Format("20060102")
				blessedPath := filepath.Join("comments", dateDir, commentID+".md")

				payload := &hooks.HookPayload{
					Event:         hooks.EventPostComment,
					Path:          blessedPath,
					Title:         fm["in_reply_to"],
					Version:       fm["comment_version"],
					Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					CommitMessage: hooks.GenerateCommitMessage(hooks.EventPostComment, fm["in_reply_to"]),
				}
				if _, err := hooks.RunHook(dataDir, hookConfig, payload); err != nil {
					// Log error but don't fail sync
					result.Errors = append(result.Errors, "hook failed for "+commentID+": "+err.Error())
				}
			}

		case "denied":
			// Move to denied directory
			if err := MoveComment(dataDir, commentID, StatusPending, StatusDenied); err != nil {
				result.Errors = append(result.Errors, "failed to move "+commentID+" to denied: "+err.Error())
				continue
			}
			result.Denied = append(result.Denied, commentID)

		case "pending":
			result.StillPending = append(result.StillPending, commentID)

		default:
			result.Errors = append(result.Errors, commentID+": unknown status "+status)
		}
	}

	return result, nil
}

// SyncSingleComment syncs a single comment by ID.
func SyncSingleComment(dataDir, commentID string, discoveryClient *discovery.Client, hookConfig *hooks.HookConfig) (string, error) {
	// Read comment to get URL (from .polis/comments/pending/)
	commentPath := filepath.Join(dataDir, ".polis", "comments", StatusPending, commentID+".md")
	data, err := os.ReadFile(commentPath)
	if err != nil {
		return "", err
	}

	fm := ParseFrontmatter(string(data))
	commentURL := fm["comment_url"]
	inReplyTo := fm["in_reply_to"]
	if commentURL == "" || inReplyTo == "" {
		return "", nil
	}

	// Check status via relationship-query
	resp, err := discoveryClient.QueryRelationships("polis.blessing", map[string]string{
		"source_url": commentURL,
		"target_url": inReplyTo,
	})
	if err != nil {
		return "", err
	}

	status := "pending"
	if resp != nil && len(resp.Records) > 0 {
		status = resp.Records[0].Status
	}

	switch status {
	case "granted":
		if err := MoveComment(dataDir, commentID, StatusPending, StatusBlessed); err != nil {
			return "", err
		}

		// Run hook
		if hookConfig != nil {
			// Determine the blessed comment path based on timestamp
			timestamp := time.Now().UTC()
			if ts := fm["timestamp"]; ts != "" {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z", ts); err == nil {
					timestamp = parsed
				}
			}
			dateDir := timestamp.Format("20060102")
			blessedPath := filepath.Join("comments", dateDir, commentID+".md")

			payload := &hooks.HookPayload{
				Event:         hooks.EventPostComment,
				Path:          blessedPath,
				Title:         fm["in_reply_to"],
				Version:       fm["comment_version"],
				Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				CommitMessage: hooks.GenerateCommitMessage(hooks.EventPostComment, fm["in_reply_to"]),
			}
			hooks.RunHook(dataDir, hookConfig, payload)
		}

		return "granted", nil

	case "denied":
		if err := MoveComment(dataDir, commentID, StatusPending, StatusDenied); err != nil {
			return "", err
		}
		return "denied", nil

	default:
		return "pending", nil
	}
}
