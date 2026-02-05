package clone

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"https://example.com/", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"http://example.com/", "http://example.com"},
	}

	for _, tt := range tests {
		result := normalizeURL(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractDomainForDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "example.com"},
		{"https://example.com/", "example.com"},
		{"https://example.com/path", "example.com"},
		{"http://example.com", "example.com"},
		{"example.com", "example.com"},
		{"sub.example.com", "sub.example.com"},
	}

	for _, tt := range tests {
		result := ExtractDomainForDir(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractDomainForDir(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestURLToLocalPath(t *testing.T) {
	tests := []struct {
		remoteURL string
		baseURL   string
		localDir  string
		expected  string
	}{
		{
			"https://example.com/posts/2025/01/hello.md",
			"https://example.com",
			"/local/posts",
			"/local/posts/posts/2025/01/hello.md",
		},
	}

	for _, tt := range tests {
		result := urlToLocalPath(tt.remoteURL, tt.baseURL, tt.localDir)
		if result != tt.expected {
			t.Errorf("urlToLocalPath() = %q, expected %q", result, tt.expected)
		}
	}
}
