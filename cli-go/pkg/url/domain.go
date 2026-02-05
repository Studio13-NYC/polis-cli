package url

import "strings"

// ExtractDomain extracts the host/domain from a URL.
// Example: "https://alice.polis.pub/posts/..." → "alice.polis.pub"
// Matches bash CLI's extract_domain_from_url function.
func ExtractDomain(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Remove trailing slash
	rawURL = strings.TrimSuffix(rawURL, "/")

	// Remove protocol
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")

	// Remove path if present
	if idx := strings.Index(rawURL, "/"); idx > 0 {
		rawURL = rawURL[:idx]
	}

	// Remove port if present
	if idx := strings.Index(rawURL, ":"); idx > 0 {
		rawURL = rawURL[:idx]
	}

	return rawURL
}
