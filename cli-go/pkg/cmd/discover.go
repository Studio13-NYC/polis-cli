package cmd

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vdibart/polis-cli/cli-go/pkg/feed"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
)

func handleDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	specificAuthor := fs.String("author", "", "Check a specific author")
	sinceDate := fs.String("since", "", "Show items since date (ignores last_checked)")
	fs.Parse(args)

	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	followingPath := filepath.Join(dir, "metadata", "following.json")
	client := remote.NewClient()

	result, err := feed.Aggregate(followingPath, client, feed.AggregateOptions{
		SpecificAuthor: *specificAuthor,
		SinceOverride:  *sinceDate,
	})
	if err != nil {
		if _, ok := err.(*feed.NotFollowingError); ok {
			exitError(err.Error())
		}
		exitError("Failed to aggregate feed: %v", err)
	}

	if result.AuthorsChecked == 0 {
		if !jsonOutput {
			fmt.Println("[i] Not following any authors. Use 'polis follow <url>' to follow someone.")
		}
		return
	}

	if !jsonOutput {
		fmt.Printf("[i] Checking %d followed author(s)...\n\n", result.AuthorsChecked)

		// Print errors for unreachable authors
		for _, errMsg := range result.Errors {
			fmt.Printf("  %s - offline or error\n", errMsg)
		}

		// Print items grouped by author
		if len(result.Items) > 0 {
			currentAuthor := ""
			for _, item := range result.Items {
				if item.AuthorURL != currentAuthor {
					currentAuthor = item.AuthorURL
					fmt.Printf("\n%s\n", currentAuthor)
				}
				typeLabel := strings.ToUpper(item.Type)
				dateStr := item.Published
				if len(dateStr) > 10 {
					dateStr = dateStr[:10]
				}
				fmt.Printf("  [%s] %s - %s\n", typeLabel, item.Title, dateStr)
			}
		}

		fmt.Println()
		if len(result.Items) > 0 {
			fmt.Printf("[✓] Found %d new item(s) from %d author(s)\n", len(result.Items), result.AuthorsWithNew)
		} else {
			fmt.Println("[i] No new content from followed authors")
		}
	}

	if jsonOutput {
		// Build items array matching original format
		var items []map[string]interface{}
		for _, item := range result.Items {
			items = append(items, map[string]interface{}{
				"author":    item.AuthorURL,
				"type":      item.Type,
				"title":     item.Title,
				"url":       item.URL,
				"published": item.Published,
			})
		}

		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "discover",
			"data": map[string]interface{}{
				"authors_checked":  result.AuthorsChecked,
				"authors_with_new": result.AuthorsWithNew,
				"total_new_items":  len(result.Items),
				"items":            items,
			},
		})
	}
}
