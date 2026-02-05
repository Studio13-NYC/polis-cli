package template

import (
	"fmt"
	"strings"
)

// WrapWithMarkers wraps content with HTML comment markers for snippet editing.
// These markers allow the webapp to identify snippet boundaries in rendered HTML.
//
// Format:
//
//	<!-- POLIS-SNIPPET-START: {source}:{path} path={path} -->
//	<span class="polis-snippet-boundary" data-snippet="{source}:{path}" data-path="{path}" data-source="{source}" hidden></span>
//	{content}
//	<!-- POLIS-SNIPPET-END: {source}:{path} -->
func WrapWithMarkers(content, path, source string) string {
	// Clean the path (remove leading/trailing whitespace)
	path = strings.TrimSpace(path)

	// Build the full identifier
	identifier := fmt.Sprintf("%s:%s", source, path)

	// Build the markers
	startMarker := fmt.Sprintf("<!-- POLIS-SNIPPET-START: %s path=%s -->", identifier, path)
	boundarySpan := fmt.Sprintf(`<span class="polis-snippet-boundary" data-snippet="%s" data-path="%s" data-source="%s" hidden></span>`,
		identifier, path, source)
	endMarker := fmt.Sprintf("<!-- POLIS-SNIPPET-END: %s -->", identifier)

	// Wrap the content
	return fmt.Sprintf("%s\n%s\n%s\n%s", startMarker, boundarySpan, content, endMarker)
}

// StripMarkers removes all POLIS-SNIPPET markers from content.
// This is useful when you need clean content without editing markers.
func StripMarkers(content string) string {
	// Remove start markers
	content = stripPattern(content, `<!-- POLIS-SNIPPET-START:[^>]*-->`)

	// Remove boundary spans
	content = stripPattern(content, `<span class="polis-snippet-boundary"[^>]*></span>`)

	// Remove end markers
	content = stripPattern(content, `<!-- POLIS-SNIPPET-END:[^>]*-->`)

	return content
}

// stripPattern removes all occurrences of a regex pattern from content.
func stripPattern(content, pattern string) string {
	// Simple string-based removal since we're dealing with specific patterns
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- POLIS-SNIPPET-START:") ||
			strings.HasPrefix(trimmed, "<!-- POLIS-SNIPPET-END:") ||
			strings.HasPrefix(trimmed, `<span class="polis-snippet-boundary"`) {
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// ExtractSnippetInfo extracts snippet information from a marker.
// Returns source, path, and whether the marker was valid.
func ExtractSnippetInfo(marker string) (source, path string, ok bool) {
	// Look for the pattern: POLIS-SNIPPET-START: {source}:{path}
	if !strings.Contains(marker, "POLIS-SNIPPET-START:") {
		return "", "", false
	}

	// Extract the identifier after "POLIS-SNIPPET-START: "
	start := strings.Index(marker, "POLIS-SNIPPET-START:") + len("POLIS-SNIPPET-START:")
	end := strings.Index(marker[start:], " ")
	if end == -1 {
		end = strings.Index(marker[start:], "-->")
	}
	if end == -1 {
		return "", "", false
	}

	identifier := strings.TrimSpace(marker[start : start+end])

	// Split by first colon
	parts := strings.SplitN(identifier, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
