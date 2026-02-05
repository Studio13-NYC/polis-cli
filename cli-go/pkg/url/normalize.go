// Package url provides URL utilities for polis.
package url

import (
	"net/url"
	"strings"
)

// NormalizeToMD converts .html URLs to .md format.
// Polis stores all content URLs with .md extension.
// This matches the bash CLI's normalize_url_to_md function.
func NormalizeToMD(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If we can't parse, fall back to simple string replacement
		if strings.HasSuffix(rawURL, ".html") {
			return strings.TrimSuffix(rawURL, ".html") + ".md"
		}
		return rawURL
	}

	if strings.HasSuffix(parsed.Path, ".html") {
		parsed.Path = strings.TrimSuffix(parsed.Path, ".html") + ".md"
	}

	return parsed.String()
}
