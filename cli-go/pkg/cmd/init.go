package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	siteTitle := fs.String("site-title", "", "Site display name")
	register := fs.Bool("register", false, "Auto-register with discovery service after init")
	keysDir := fs.String("keys-dir", "", "Custom keys directory (default: .polis/keys)")
	postsDir := fs.String("posts-dir", "", "Custom posts directory (default: posts)")
	commentsDir := fs.String("comments-dir", "", "Custom comments directory (default: comments)")
	snippetsDir := fs.String("snippets-dir", "", "Custom snippets directory (default: snippets)")
	themesDir := fs.String("themes-dir", "", "Custom themes directory (default: .polis/themes)")
	versionsDir := fs.String("versions-dir", "", "Custom versions directory (default: .versions)")
	publicIndex := fs.String("public-index", "", "Custom public index path (default: metadata/public.jsonl)")
	blessedComments := fs.String("blessed-comments", "", "Custom blessed comments path (default: metadata/blessed-comments.json)")
	followingIndex := fs.String("following-index", "", "Custom following index path (default: metadata/following.json)")
	fs.Parse(args)

	dir := getDataDir()

	opts := site.InitOptions{
		SiteTitle:       *siteTitle,
		Version:         Version,
		KeysDir:         *keysDir,
		PostsDir:        *postsDir,
		CommentsDir:     *commentsDir,
		SnippetsDir:     *snippetsDir,
		ThemesDir:       *themesDir,
		VersionsDir:     *versionsDir,
		PublicIndex:     *publicIndex,
		BlessedComments: *blessedComments,
		FollowingIndex:  *followingIndex,
	}

	result, err := site.Init(dir, opts)
	if err != nil {
		exitError("Failed to initialize site: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "init",
			"data": map[string]interface{}{
				"directories_created": result.DirsCreated,
				"files_created":       result.FilesCreated,
				"key_paths": map[string]interface{}{
					"private": result.KeyPaths.Private,
					"public":  result.KeyPaths.Public,
				},
			},
		})
	} else {
		fmt.Printf("[✓] Initialized polis site at: %s\n", result.SiteDir)
		fmt.Printf("[i] Public key: %s\n", result.PublicKey[:50]+"...")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Set POLIS_BASE_URL in .env file")
		fmt.Println("  2. Create your first post: polis post my-post.md")
		fmt.Println("  3. Render the site: polis render")
	}

	// Auto-register if requested
	if *register {
		if os.Getenv("POLIS_BASE_URL") == "" {
			if !jsonOutput {
				fmt.Println("[!] Skipping registration: POLIS_BASE_URL not set")
			}
			return
		}
		if os.Getenv("DISCOVERY_SERVICE_KEY") == "" {
			if !jsonOutput {
				fmt.Println("[!] Skipping registration: DISCOVERY_SERVICE_KEY not set")
			}
			return
		}
		handleRegister(nil)
	}
}
