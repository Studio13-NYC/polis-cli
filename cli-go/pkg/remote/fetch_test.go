package remote

import (
	"testing"
)

func TestExtractBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/posts/hello.md", "https://example.com"},
		{"https://example.com/", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"https://sub.example.com/path/to/file", "https://sub.example.com"},
		{"http://example.com/posts/hello.md", "http://example.com"},
	}

	for _, tt := range tests {
		result := ExtractBaseURL(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractBaseURL(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestClientCreation(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient() returned nil")
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
}
