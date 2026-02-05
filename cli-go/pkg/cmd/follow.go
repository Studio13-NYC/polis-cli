package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vdibart/polis-cli/cli-go/pkg/blessing"
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

func handleFollow(args []string) {
	if len(args) < 1 {
		exitError("Usage: polis follow <author-url>")
	}

	authorURL := args[0]
	dir := getDataDir()

	// Validate URL format
	if len(authorURL) < 8 || authorURL[:8] != "https://" {
		exitError("Author URL must use HTTPS (e.g., https://example.com)")
	}

	if !isPolisSite(dir) {
		exitError("Polis not initialized. Run 'polis init' first.")
	}

	// Get our author email for blessing operations
	wk, err := site.LoadWellKnown(dir)
	if err != nil {
		exitError("Failed to load .well-known/polis: %v", err)
	}
	blessedBy := wk.Email

	// Fetch author info from their site
	remoteClient := remote.NewClient()
	remoteWK, err := remoteClient.FetchWellKnown(authorURL)
	if err != nil {
		exitError("Failed to fetch author information: %v", err)
	}
	authorEmail := remoteWK.Email

	// Load discovery client
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}
	apiKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	client := discovery.NewClient(discoveryURL, apiKey)

	// Fetch unblessed comments from this author on our posts
	pendingComments, _ := client.GetCommentsByAuthor(authorEmail, "pending")
	deniedComments, _ := client.GetCommentsByAuthor(authorEmail, "denied")

	var allUnblessed []discovery.Comment
	allUnblessed = append(allUnblessed, pendingComments...)
	allUnblessed = append(allUnblessed, deniedComments...)

	// Bless all unblessed comments
	blessedCount := 0
	failedCount := 0

	privKey, err := loadPrivateKey(dir)
	if err != nil {
		exitError("Failed to load private key: %v", err)
	}

	for _, comment := range allUnblessed {
		// Grant blessing via discovery service
		if err := client.GrantBlessing(comment.CommentVersion, privKey); err != nil {
			failedCount++
			continue
		}
		blessedCount++
	}

	// Add to following.json
	followingPath := filepath.Join(dir, "metadata", "following.json")
	f, err := following.Load(followingPath)
	if err != nil {
		exitError("Failed to load following.json: %v", err)
	}

	f.Add(authorURL)

	if err := following.Save(followingPath, f); err != nil {
		exitError("Failed to save following.json: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "follow",
			"data": map[string]interface{}{
				"author_url":         authorURL,
				"author_email":       authorEmail,
				"blessed_by":         blessedBy,
				"comments_found":     len(allUnblessed),
				"comments_blessed":   blessedCount,
				"added_to_following": true,
			},
		})
	} else {
		fmt.Println()
		fmt.Printf("[✓] Successfully followed %s\n", authorURL)
		fmt.Printf("  - Added to following.json\n")
		if len(allUnblessed) > 0 {
			if failedCount == 0 {
				fmt.Printf("  - Blessed %d comment(s)\n", blessedCount)
			} else {
				fmt.Printf("  - Blessed %d/%d comment(s) (%d failed)\n", blessedCount, len(allUnblessed), failedCount)
			}
		}
	}

	// Suppress unused variable warning
	_ = blessedBy
	_ = blessing.SyncBlessedComments
}
