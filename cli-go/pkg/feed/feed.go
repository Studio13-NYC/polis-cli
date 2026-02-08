// Package feed provides feed aggregation for followed authors.
package feed

import (
	"sort"
	"strings"

	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
)

// FeedItem represents a single item in the aggregated feed.
type FeedItem struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Published   string `json:"published"`
	Hash        string `json:"hash,omitempty"`
	AuthorURL   string `json:"author_url"`
	AuthorDomain string `json:"author_domain"`
}

// AggregateResult contains the result of a feed aggregation.
type AggregateResult struct {
	Items          []FeedItem `json:"items"`
	AuthorsChecked int        `json:"authors_checked"`
	AuthorsWithNew int        `json:"authors_with_new"`
	Errors         []string   `json:"errors,omitempty"`
}

// AggregateOptions configures feed aggregation.
type AggregateOptions struct {
	// SpecificAuthor limits the check to a single author URL.
	SpecificAuthor string
	// SinceOverride ignores last_checked and uses this date instead.
	SinceOverride string
}

// Aggregate fetches public indexes from followed authors, filters by last_checked,
// merges, sorts by published desc, updates last_checked timestamps, and saves.
func Aggregate(followingPath string, client *remote.Client, opts AggregateOptions) (*AggregateResult, error) {
	f, err := following.Load(followingPath)
	if err != nil {
		return nil, err
	}

	result := &AggregateResult{}

	if f.Count() == 0 {
		return result, nil
	}

	// Filter to specific author if requested
	var authors []following.FollowingEntry
	if opts.SpecificAuthor != "" {
		entry := f.Get(opts.SpecificAuthor)
		if entry == nil {
			return nil, &NotFollowingError{URL: opts.SpecificAuthor}
		}
		authors = []following.FollowingEntry{*entry}
	} else {
		authors = f.All()
	}

	result.AuthorsChecked = len(authors)

	for _, author := range authors {
		authorURL := strings.TrimSuffix(author.URL, "/")
		domain := extractDomain(authorURL)

		// Determine check-since date
		checkSince := opts.SinceOverride
		if checkSince == "" && author.LastChecked != "" {
			checkSince = author.LastChecked
		}

		// Fetch manifest
		manifest, err := client.FetchManifest(authorURL)
		if err != nil {
			result.Errors = append(result.Errors, authorURL+": "+err.Error())
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
			f.UpdateLastChecked(authorURL)
			continue
		}

		// Fetch public index
		entries, err := client.FetchPublicIndex(authorURL)
		if err != nil {
			result.Errors = append(result.Errors, authorURL+": "+err.Error())
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

		if len(newItems) > 0 {
			result.AuthorsWithNew++
			for _, item := range newItems {
				result.Items = append(result.Items, FeedItem{
					Type:         item.Type,
					Title:        item.Title,
					URL:          item.GetPath(),
					Published:    item.Published,
					Hash:         item.Hash,
					AuthorURL:    authorURL,
					AuthorDomain: domain,
				})
			}
		}

		f.UpdateLastChecked(authorURL)
	}

	// Sort by published descending
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].Published > result.Items[j].Published
	})

	// Save updated following.json
	if err := following.Save(followingPath, f); err != nil {
		result.Errors = append(result.Errors, "save following.json: "+err.Error())
	}

	return result, nil
}

// extractDomain extracts the domain from a URL like "https://example.com/path".
func extractDomain(url string) string {
	s := strings.TrimPrefix(url, "https://")
	s = strings.TrimPrefix(s, "http://")
	if idx := strings.Index(s, "/"); idx > 0 {
		return s[:idx]
	}
	return s
}

// NotFollowingError indicates the requested author is not being followed.
type NotFollowingError struct {
	URL string
}

func (e *NotFollowingError) Error() string {
	return "not following " + e.URL
}
