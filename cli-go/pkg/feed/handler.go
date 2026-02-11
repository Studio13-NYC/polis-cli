package feed

import (
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
)

// FeedHandler transforms discovery stream events into FeedItems.
// It filters out self-authored content and maps event payloads
// to the common FeedItem structure.
type FeedHandler struct {
	// MyDomain is the local site's domain (used to skip self-authored events).
	MyDomain string
	// FollowedDomains is the set of domains we follow (for validation).
	FollowedDomains map[string]bool
}

// Process converts stream events into FeedItems.
// It skips self-authored events (actor == MyDomain) and unknown event types.
func (h *FeedHandler) Process(events []discovery.StreamEvent) []FeedItem {
	var items []FeedItem

	for _, evt := range events {
		// Skip self-authored events
		if evt.Actor == h.MyDomain {
			continue
		}

		item, ok := h.eventToItem(evt)
		if !ok {
			continue
		}
		items = append(items, item)
	}

	return items
}

// eventToItem maps a single stream event to a FeedItem.
// Returns false if the event type is not a feed-relevant type.
func (h *FeedHandler) eventToItem(evt discovery.StreamEvent) (FeedItem, bool) {
	switch evt.Type {
	case "polis.post.published", "polis.post.republished":
		return h.postEventToItem(evt), true
	case "polis.comment.published", "polis.comment.republished":
		return h.commentEventToItem(evt), true
	default:
		return FeedItem{}, false
	}
}

// postEventToItem extracts FeedItem fields from a post event.
func (h *FeedHandler) postEventToItem(evt discovery.StreamEvent) FeedItem {
	url, _ := evt.Payload["url"].(string)
	version, _ := evt.Payload["version"].(string)
	author, _ := evt.Payload["author"].(string)
	_ = author // available but not stored in FeedItem

	title := ""
	published := ""
	if md, ok := evt.Payload["metadata"].(map[string]interface{}); ok {
		title, _ = md["title"].(string)
		published, _ = md["published_at"].(string)
	}

	if published == "" {
		published = evt.Timestamp
	}

	return FeedItem{
		Type:         "post",
		Title:        title,
		URL:          url,
		Published:    published,
		Hash:         version,
		AuthorURL:    "https://" + evt.Actor,
		AuthorDomain: evt.Actor,
	}
}

// commentEventToItem extracts FeedItem fields from a comment event.
func (h *FeedHandler) commentEventToItem(evt discovery.StreamEvent) FeedItem {
	commentURL, _ := evt.Payload["comment_url"].(string)
	version, _ := evt.Payload["version"].(string)

	title := ""
	published := ""
	if md, ok := evt.Payload["metadata"].(map[string]interface{}); ok {
		title, _ = md["title"].(string)
		published, _ = md["published_at"].(string)
	}

	if published == "" {
		published = evt.Timestamp
	}

	return FeedItem{
		Type:         "comment",
		Title:        title,
		URL:          commentURL,
		Published:    published,
		Hash:         version,
		AuthorURL:    "https://" + evt.Actor,
		AuthorDomain: evt.Actor,
	}
}
