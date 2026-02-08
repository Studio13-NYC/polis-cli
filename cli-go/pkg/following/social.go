package following

import (
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
)

// FollowResult contains the result of a follow operation.
type FollowResult struct {
	AuthorURL       string `json:"author_url"`
	AuthorEmail     string `json:"author_email"`
	CommentsFound   int    `json:"comments_found"`
	CommentsBlessed int    `json:"comments_blessed"`
	CommentsFailed  int    `json:"comments_failed"`
	AlreadyFollowed bool   `json:"already_followed"`
}

// UnfollowResult contains the result of an unfollow operation.
type UnfollowResult struct {
	AuthorURL      string `json:"author_url"`
	CommentsDenied int    `json:"comments_denied"`
	CommentsFailed int    `json:"comments_failed"`
	CommentsFound  int    `json:"comments_found"`
	WasFollowing   bool   `json:"was_following"`
}

// FollowWithBlessing adds an author to the following list and blesses any
// pending or denied comments from that author. This matches the CLI behavior
// where following someone auto-blesses their comments.
func FollowWithBlessing(followingPath string, authorURL string, discoveryClient *discovery.Client, remoteClient *remote.Client, privKey []byte) (*FollowResult, error) {
	result := &FollowResult{
		AuthorURL: authorURL,
	}

	// Fetch author email from their site
	remoteWK, err := remoteClient.FetchWellKnown(authorURL)
	if err != nil {
		return nil, err
	}
	result.AuthorEmail = remoteWK.Email

	// Fetch unblessed comments from this author
	pendingComments, _ := discoveryClient.GetCommentsByAuthor(remoteWK.Email, "pending")
	deniedComments, _ := discoveryClient.GetCommentsByAuthor(remoteWK.Email, "denied")

	var allUnblessed []discovery.Comment
	allUnblessed = append(allUnblessed, pendingComments...)
	allUnblessed = append(allUnblessed, deniedComments...)
	result.CommentsFound = len(allUnblessed)

	// Bless all unblessed comments
	for _, comment := range allUnblessed {
		if err := discoveryClient.GrantBlessing(comment.CommentVersion, privKey); err != nil {
			result.CommentsFailed++
			continue
		}
		result.CommentsBlessed++
	}

	// Add to following.json
	f, err := Load(followingPath)
	if err != nil {
		return nil, err
	}

	added := f.Add(authorURL)
	if !added {
		result.AlreadyFollowed = true
	}

	if err := Save(followingPath, f); err != nil {
		return nil, err
	}

	return result, nil
}

// UnfollowWithDenial removes an author from the following list and denies
// any blessed comments from that author. This matches the CLI behavior.
func UnfollowWithDenial(followingPath string, authorURL string, discoveryClient *discovery.Client, remoteClient *remote.Client, privKey []byte) (*UnfollowResult, error) {
	result := &UnfollowResult{
		AuthorURL: authorURL,
	}

	// Fetch author email (non-fatal if unreachable)
	remoteWK, err := remoteClient.FetchWellKnown(authorURL)
	if err == nil && remoteWK != nil && remoteWK.Email != "" {
		// Deny any blessed comments from this author
		blessedComments, _ := discoveryClient.GetCommentsByAuthor(remoteWK.Email, "blessed")
		result.CommentsFound = len(blessedComments)

		for _, comment := range blessedComments {
			if err := discoveryClient.DenyBlessing(comment.CommentVersion, privKey); err != nil {
				result.CommentsFailed++
				continue
			}
			result.CommentsDenied++
		}
	}

	// Remove from following.json
	f, err := Load(followingPath)
	if err != nil {
		return nil, err
	}

	result.WasFollowing = f.Remove(authorURL)

	if err := Save(followingPath, f); err != nil {
		return nil, err
	}

	return result, nil
}
