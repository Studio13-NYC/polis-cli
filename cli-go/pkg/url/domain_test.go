package url

import "testing"

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://alice.polis.pub/posts/2026/01/hello.md", "alice.polis.pub"},
		{"http://example.com/path", "example.com"},
		{"https://example.com:8080/path", "example.com"},
		{"https://example.com/", "example.com"},
		{"https://example.com", "example.com"},
		{"example.com/path", "example.com"},
		{"", ""},
		{"  https://spaced.com  ", "spaced.com"},
	}

	for _, tt := range tests {
		result := ExtractDomain(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractDomain(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
