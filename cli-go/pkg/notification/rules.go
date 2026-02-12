package notification

import (
	"crypto/sha256"
	"fmt"
	"path"
	"strings"
)

// Rule defines how a stream event type maps to a notification.
type Rule struct {
	ID        string       `json:"id"`
	EventType string       `json:"event_type"`
	Enabled   bool         `json:"enabled"`
	Filter    RuleFilter   `json:"filter"`
	Template  RuleTemplate `json:"template"`
	Batch     bool         `json:"batch"`
	// BatchWindow is a duration string like "24h". Only used when Batch is true.
	BatchWindow string `json:"batch_window,omitempty"`
}

// RuleFilter specifies how to determine relevance of an event.
type RuleFilter struct {
	// Relevance is one of: "target_domain", "source_domain", "followed_author"
	Relevance string `json:"relevance"`
}

// RuleTemplate defines the display format for a notification.
type RuleTemplate struct {
	Icon    string `json:"icon"`
	Message string `json:"message"`
}

// DefaultRules returns the built-in rule set seeded on first sync.
func DefaultRules() []Rule {
	return []Rule{
		{
			ID:        "new-follower",
			EventType: "polis.follow.announced",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "target_domain"},
			Template:  RuleTemplate{Icon: "\U0001F464", Message: "{{actor}} started following you"},
			Batch:     true,
			BatchWindow: "24h",
		},
		{
			ID:        "lost-follower",
			EventType: "polis.follow.removed",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "target_domain"},
			Template:  RuleTemplate{Icon: "\U0001F464", Message: "{{actor}} unfollowed you"},
		},
		{
			ID:        "blessing-requested",
			EventType: "polis.blessing.requested",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "target_domain"},
			Template:  RuleTemplate{Icon: "\U0001F514", Message: "{{actor}} requested a blessing on {{post_name}}"},
		},
		{
			ID:        "blessing-granted",
			EventType: "polis.blessing.granted",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "source_domain"},
			Template:  RuleTemplate{Icon: "\u2713", Message: "{{actor}} blessed your comment"},
		},
		{
			ID:        "blessing-denied",
			EventType: "polis.blessing.denied",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "source_domain"},
			Template:  RuleTemplate{Icon: "\u2717", Message: "{{actor}} denied your comment"},
		},
		{
			ID:        "new-comment",
			EventType: "polis.comment.published",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "target_domain"},
			Template:  RuleTemplate{Icon: "\U0001F4AC", Message: "{{actor}} commented on {{post_name}}"},
		},
		{
			ID:        "updated-comment",
			EventType: "polis.comment.republished",
			Enabled:   false,
			Filter:    RuleFilter{Relevance: "target_domain"},
			Template:  RuleTemplate{Icon: "\U0001F4AC", Message: "{{actor}} updated their comment on {{post_name}}"},
		},
		{
			ID:        "new-post",
			EventType: "polis.post.published",
			Enabled:   true,
			Filter:    RuleFilter{Relevance: "followed_author"},
			Template:  RuleTemplate{Icon: "\U0001F4DD", Message: "{{actor}} published a new post"},
		},
		{
			ID:        "updated-post",
			EventType: "polis.post.republished",
			Enabled:   false,
			Filter:    RuleFilter{Relevance: "followed_author"},
			Template:  RuleTemplate{Icon: "\U0001F4DD", Message: "{{actor}} updated a post"},
		},
	}
}

// ResolveTemplate performs simple {{var}} substitution on a template string.
// Available variables: actor, timestamp, source_url, target_url, target_domain, source_domain, post_name.
func ResolveTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// TemplateVarsFromEvent builds template variables from a stream event's fields.
func TemplateVarsFromEvent(actor, timestamp string, payload map[string]interface{}) map[string]string {
	vars := map[string]string{
		"actor":     actor,
		"timestamp": timestamp,
	}

	for _, key := range []string{"source_url", "target_url", "target_domain", "source_domain"} {
		if v, ok := payload[key].(string); ok {
			vars[key] = v
		}
	}

	// Derive post_name from target_url or url (last path segment, without extension)
	postURL := ""
	if targetURL, ok := payload["target_url"].(string); ok && targetURL != "" {
		postURL = targetURL
	} else if u, ok := payload["url"].(string); ok && u != "" {
		postURL = u
	}
	if postURL != "" {
		base := path.Base(postURL)
		if strings.HasSuffix(base, ".md") {
			base = strings.TrimSuffix(base, ".md")
		}
		vars["post_name"] = base
	}

	// Extract title from metadata if available (post events include it)
	if meta, ok := payload["metadata"].(map[string]interface{}); ok {
		if title, ok := meta["title"].(string); ok && title != "" {
			vars["title"] = title
		}
	}

	return vars
}

// DedupeKey computes a deterministic dedupe key for a notification from a rule and event.
// Format: "<rule_id>:<content_identifier>"
func DedupeKey(ruleID string, payload map[string]interface{}) string {
	// Use source_url as the primary content identifier (unique per comment/action)
	if sourceURL, ok := payload["source_url"].(string); ok && sourceURL != "" {
		return ruleID + ":" + sourceURL
	}
	// Post events use "url" instead of "source_url"
	if postURL, ok := payload["url"].(string); ok && postURL != "" {
		return ruleID + ":" + postURL
	}
	// For follow events, use actor + target_domain
	if targetDomain, ok := payload["target_domain"].(string); ok && targetDomain != "" {
		return ruleID + ":" + targetDomain
	}
	// Fallback: hash the payload
	h := sha256.Sum256([]byte(fmt.Sprintf("%v", payload)))
	return ruleID + ":" + fmt.Sprintf("%x", h[:8])
}
