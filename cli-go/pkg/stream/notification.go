package stream

import (
	"fmt"
	"strconv"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/notification"
)

// NotificationHandler is a rule-driven projection handler that generates local
// notifications from stream events. Rules define which event types produce
// notifications, how to filter them, and what display template to use.
type NotificationHandler struct {
	// MyDomain is the local site's domain.
	MyDomain string
	// Rules is the active rule set (loaded from polis.notification.json or DefaultRules).
	Rules []notification.Rule
	// MutedDomains is a set of domains to suppress notifications from.
	MutedDomains map[string]bool
}

// NotificationConfig is the user configuration stored in config/notifications.json.
// It contains rules and muted_domains.
type NotificationConfig struct {
	Rules        []notification.Rule `json:"rules"`
	MutedDomains []string            `json:"muted_domains"`
	MaxItems     int                 `json:"max_items,omitempty"`
	MaxAgeDays   int                 `json:"max_age_days,omitempty"`
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
		"polis.post.republished",
		"polis.comment.published",
		"polis.comment.republished",
	}
}

func (h *NotificationHandler) NewState() interface{} {
	return &NotificationConfig{
		Rules: notification.DefaultRules(),
	}
}

// EnabledEventTypes returns the event types that have at least one enabled rule.
func (h *NotificationHandler) EnabledEventTypes() []string {
	seen := make(map[string]bool)
	var types []string
	for _, r := range h.Rules {
		if r.Enabled && !seen[r.EventType] {
			seen[r.EventType] = true
			types = append(types, r.EventType)
		}
	}
	return types
}

// RulesByRelevance groups enabled rules by their filter relevance.
func (h *NotificationHandler) RulesByRelevance() map[string][]notification.Rule {
	groups := make(map[string][]notification.Rule)
	for _, r := range h.Rules {
		if r.Enabled {
			groups[r.Filter.Relevance] = append(groups[r.Filter.Relevance], r)
		}
	}
	return groups
}

// Process applies rules to events and returns notification StateEntries.
// Unlike other projection handlers, this doesn't maintain in-memory state —
// it produces entries that get appended to state.jsonl by the caller.
func (h *NotificationHandler) Process(events []discovery.StreamEvent) []notification.StateEntry {
	// Build a rule lookup by event type
	ruleMap := make(map[string][]notification.Rule)
	for _, r := range h.Rules {
		if r.Enabled {
			ruleMap[r.EventType] = append(ruleMap[r.EventType], r)
		}
	}

	var entries []notification.StateEntry

	for _, evt := range events {
		// Skip self-events
		if evt.Actor == h.MyDomain {
			continue
		}

		// Skip muted domains
		if h.MutedDomains[evt.Actor] {
			continue
		}

		rules, ok := ruleMap[evt.Type]
		if !ok {
			continue
		}

		for _, rule := range rules {
			if !h.matchesFilter(rule, evt) {
				continue
			}

			// Resolve template
			vars := notification.TemplateVarsFromEvent(evt.Actor, evt.Timestamp, evt.Payload)
			message := notification.ResolveTemplate(rule.Template.Message, vars)

			// Compute dedupe key
			dedupeKey := notification.DedupeKey(rule.ID, evt.Payload)

			// Parse event ID
			eventID, _ := strconv.Atoi(fmt.Sprintf("%v", evt.ID))

			entries = append(entries, notification.StateEntry{
				ID:        dedupeKey,
				RuleID:    rule.ID,
				Actor:     evt.Actor,
				Icon:      rule.Template.Icon,
				Message:   message,
				EventIDs:  []int{eventID},
				CreatedAt: evt.Timestamp,
			})
		}
	}

	return entries
}

// matchesFilter checks if an event matches a rule's filter.
func (h *NotificationHandler) matchesFilter(rule notification.Rule, evt discovery.StreamEvent) bool {
	switch rule.Filter.Relevance {
	case "target_domain":
		targetDomain, _ := evt.Payload["target_domain"].(string)
		return targetDomain == h.MyDomain

	case "source_domain":
		sourceDomain, _ := evt.Payload["source_domain"].(string)
		return sourceDomain == h.MyDomain

	case "followed_author":
		// For "followed_author" relevance, the caller should have pre-filtered
		// via the actor query parameter. We just check it's not us.
		return evt.Actor != h.MyDomain

	default:
		return false
	}
}
