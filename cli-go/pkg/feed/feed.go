// Package feed provides feed management for followed authors.
package feed

// FeedItem represents a single item in the aggregated feed.
type FeedItem struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Published    string `json:"published"`
	Hash         string `json:"hash,omitempty"`
	AuthorURL    string `json:"author_url"`
	AuthorDomain string `json:"author_domain"`
}
