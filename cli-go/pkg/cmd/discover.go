package cmd

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/following"
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

	// Load following.json
	followingPath := filepath.Join(dir, "metadata", "following.json")
	f, err := following.Load(followingPath)
	if err != nil {
		exitError("Failed to load following.json: %v", err)
	}

	if f.Count() == 0 {
		if !jsonOutput {
			fmt.Println("[i] Not following any authors. Use 'polis follow <url>' to follow someone.")
		}
		return
	}

	// Filter to specific author if requested
	var authors []following.FollowingEntry
	if *specificAuthor != "" {
		entry := f.Get(*specificAuthor)
		if entry == nil {
			exitError("Not following %s", *specificAuthor)
		}
		authors = []following.FollowingEntry{*entry}
	} else {
		authors = f.All()
	}

	if !jsonOutput {
		fmt.Printf("[i] Checking %d followed author(s)...\n\n", len(authors))
	}

	client := remote.NewClient()
	var allNewItems []map[string]interface{}
	totalNewItems := 0
	authorsWithNew := 0

	for _, author := range authors {
		authorURL := strings.TrimSuffix(author.URL, "/")

		// Determine check-since date
		checkSince := *sinceDate
		if checkSince == "" && author.LastChecked != "" {
			checkSince = author.LastChecked
		}

		// Calculate human-readable last checked
		lastCheckedHuman := "never"
		if author.LastChecked != "" {
			if t, err := time.Parse("2006-01-02T15:04:05Z", author.LastChecked); err == nil {
				diff := time.Since(t)
				if diff.Hours() < 1 {
					lastCheckedHuman = "just now"
				} else if diff.Hours() < 24 {
					lastCheckedHuman = fmt.Sprintf("%dh ago", int(diff.Hours()))
				} else {
					lastCheckedHuman = fmt.Sprintf("%dd ago", int(diff.Hours()/24))
				}
			}
		}

		if !jsonOutput {
			fmt.Printf("%s (last checked: %s)", authorURL, lastCheckedHuman)
		}

		// Fetch manifest
		manifest, err := client.FetchManifest(authorURL)
		if err != nil {
			if !jsonOutput {
				fmt.Println(" - offline or no manifest")
			}
			continue
		}

		// Check if there's new content
		hasNewContent := false
		if checkSince == "" {
			hasNewContent = true
		} else if manifest.LastPublished > checkSince {
			hasNewContent = true
		}

		if !hasNewContent {
			if !jsonOutput {
				fmt.Println(" - no new content")
			}
			f.UpdateLastChecked(authorURL)
			continue
		}

		// Fetch public index
		entries, err := client.FetchPublicIndex(authorURL)
		if err != nil {
			if !jsonOutput {
				fmt.Println(" - could not fetch index")
			}
			continue
		}

		// Filter for new items
		var newItems []remote.PublicIndexEntry
		if checkSince != "" {
			for _, entry := range entries {
				if entry.Published > checkSince {
					newItems = append(newItems, entry)
				}
			}
		} else {
			// Show last 10 if never checked
			limit := 10
			if len(entries) < limit {
				limit = len(entries)
			}
			newItems = entries[:limit]
		}

		if len(newItems) == 0 {
			if !jsonOutput {
				fmt.Println(" - no new content")
			}
		} else {
			if !jsonOutput {
				fmt.Println()
				fmt.Printf("  → %d new item(s):\n", len(newItems))
				for _, item := range newItems {
					typeLabel := strings.ToUpper(item.Type)
					dateStr := item.Published
					if len(dateStr) > 10 {
						dateStr = dateStr[:10]
					}
					fmt.Printf("  [%s] %s - %s\n", typeLabel, item.Title, dateStr)
				}
			}

			totalNewItems += len(newItems)
			authorsWithNew++

			// Collect for JSON output
			for _, item := range newItems {
				allNewItems = append(allNewItems, map[string]interface{}{
					"author":    authorURL,
					"type":      item.Type,
					"title":     item.Title,
					"url":       item.URL,
					"published": item.Published,
				})
			}
		}

		f.UpdateLastChecked(authorURL)
	}

	// Save updated following.json
	if err := following.Save(followingPath, f); err != nil {
		// Non-fatal - just log
		if !jsonOutput {
			fmt.Printf("[!] Warning: Failed to update last_checked timestamps: %v\n", err)
		}
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "discover",
			"data": map[string]interface{}{
				"authors_checked":  len(authors),
				"authors_with_new": authorsWithNew,
				"total_new_items":  totalNewItems,
				"items":            allNewItems,
			},
		})
	} else {
		fmt.Println()
		if totalNewItems > 0 {
			fmt.Printf("[✓] Found %d new item(s) from %d author(s)\n", totalNewItems, authorsWithNew)
		} else {
			fmt.Println("[i] No new content from followed authors")
		}
	}
}
