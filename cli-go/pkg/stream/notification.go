package stream

import (
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
)

// NotificationHandler is a built-in projection handler that generates local
// notifications from stream events. It watches for follows, posts, and blessings
// targeted at the local domain and creates notification entries.
type NotificationHandler struct {
	// MyDomain is the domain to filter events for.
	MyDomain string
}

// NotificationEntry represents a single notification.
type NotificationEntry struct {
	Type      string `json:"type"`
	Actor     string `json:"actor"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Read      bool   `json:"read"`
}

// NotificationState is the materialized state for the notification projection.
type NotificationState struct {
	Notifications []NotificationEntry `json:"notifications"`
	UnreadCount   int                 `json:"unread_count"`
}

func (h *NotificationHandler) TypePrefix() string { return "polis.notification" }

func (h *NotificationHandler) EventTypes() []string {
	return []string{
		"polis.follow.announced",
		"polis.follow.removed",
		"polis.blessing.requested",
		"polis.blessing.granted",
		"polis.blessing.denied",
		"polis.post.published",
	}
}

func (h *NotificationHandler) NewState() interface{} {
	return &NotificationState{}
}

func (h *NotificationHandler) Process(events []discovery.StreamEvent, state interface{}) (interface{}, error) {
	ns, ok := state.(*NotificationState)
	if !ok {
		return nil, fmt.Errorf("notification handler: unexpected state type %T", state)
	}

	for _, evt := range events {
		var notif *NotificationEntry

		switch evt.Type {
		case "polis.follow.announced":
			targetDomain, _ := evt.Payload["target_domain"].(string)
			if targetDomain != h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "follow",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s started following you", evt.Actor),
				Timestamp: evt.Timestamp,
			}

		case "polis.follow.removed":
			targetDomain, _ := evt.Payload["target_domain"].(string)
			if targetDomain != h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "unfollow",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s unfollowed you", evt.Actor),
				Timestamp: evt.Timestamp,
			}

		case "polis.blessing.requested":
			targetURL, _ := evt.Payload["target_url"].(string)
			if targetURL == "" {
				continue
			}
			targetDomain := discovery.ExtractDomainFromURL(targetURL)
			if targetDomain != h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "blessing_request",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s requested a blessing on your post", evt.Actor),
				Timestamp: evt.Timestamp,
			}

		case "polis.blessing.granted":
			// Notify the comment author that their comment was blessed
			sourceURL, _ := evt.Payload["source_url"].(string)
			if sourceURL == "" {
				continue
			}
			sourceDomain := discovery.ExtractDomainFromURL(sourceURL)
			if sourceDomain != h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "blessing_granted",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s blessed your comment", evt.Actor),
				Timestamp: evt.Timestamp,
			}

		case "polis.blessing.denied":
			sourceURL, _ := evt.Payload["source_url"].(string)
			if sourceURL == "" {
				continue
			}
			sourceDomain := discovery.ExtractDomainFromURL(sourceURL)
			if sourceDomain != h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "blessing_denied",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s denied your comment", evt.Actor),
				Timestamp: evt.Timestamp,
			}

		case "polis.post.published":
			// Only notify if the post is from someone we follow (handled at caller level)
			// For now, skip self-posts
			if evt.Actor == h.MyDomain {
				continue
			}
			notif = &NotificationEntry{
				Type:      "new_post",
				Actor:     evt.Actor,
				Message:   fmt.Sprintf("%s published a new post", evt.Actor),
				Timestamp: evt.Timestamp,
			}
		}

		if notif != nil {
			ns.Notifications = append(ns.Notifications, *notif)
		}
	}

	// Count unread
	unread := 0
	for _, n := range ns.Notifications {
		if !n.Read {
			unread++
		}
	}

	return &NotificationState{
		Notifications: ns.Notifications,
		UnreadCount:   unread,
	}, nil
}
