package cmd

import (
	"flag"
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/clone"
)

func handleClone(args []string) {
	fs := flag.NewFlagSet("clone", flag.ExitOnError)
	fullClone := fs.Bool("full", false, "Re-download all content (ignore cached state)")
	diffClone := fs.Bool("diff", false, "Only download new/changed content (default if previously cloned)")
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 1 {
		exitError("Usage: polis clone <url> [target-dir] [--full|--diff]")
	}

	serverURL := remaining[0]
	targetDir := ""
	if len(remaining) > 1 {
		targetDir = remaining[1]
	}

	// Derive target directory from URL if not specified
	if targetDir == "" {
		targetDir = clone.ExtractDomainForDir(serverURL)
	}

	// Determine mode
	opts := clone.CloneOptions{
		FullClone: *fullClone,
	}

	// If neither specified, let the clone package decide based on state file
	if !*fullClone && !*diffClone {
		// Mode will be determined by clone package
	} else if *fullClone {
		opts.FullClone = true
	}

	if !jsonOutput {
		mode := "auto"
		if *fullClone {
			mode = "full"
		} else if *diffClone {
			mode = "diff"
		}
		fmt.Printf("[i] Cloning %s to %s (mode: %s)...\n", serverURL, targetDir, mode)
	}

	result, err := clone.Clone(serverURL, targetDir, opts)
	if err != nil {
		exitError("Failed to clone: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "clone",
			"data": map[string]interface{}{
				"target_dir":              result.TargetDir,
				"posts_downloaded":        result.PostsDownloaded,
				"comments_downloaded":     result.CommentsDownloaded,
				"blessed_comments_synced": result.BlessedCommentsSynced,
				"errors":                  result.Errors,
			},
		})
	} else {
		fmt.Println()
		fmt.Println("[✓] Clone complete!")
		fmt.Printf("  Posts downloaded: %d\n", result.PostsDownloaded)
		fmt.Printf("  Comments downloaded: %d\n", result.CommentsDownloaded)
		fmt.Printf("  Blessed comments synced: %d\n", result.BlessedCommentsSynced)
		if result.Errors > 0 {
			fmt.Printf("  Errors: %d\n", result.Errors)
		}
		fmt.Printf("  Target directory: %s\n", result.TargetDir)
	}
}
