package comment

import (
	"testing"
)

func TestBeseechComment_NotConfigured(t *testing.T) {
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	DiscoveryURL = ""
	DiscoveryKey = ""
	BaseURL = ""

	_, err := BeseechComment(t.TempDir(), "test-id", nil)
	if err == nil {
		t.Error("expected error when not configured")
	}
	if err.Error() != "discovery service not configured" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBeseechComment_NoBaseURL(t *testing.T) {
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	DiscoveryURL = "https://discovery.example.com"
	DiscoveryKey = "test-key"
	BaseURL = ""

	_, err := BeseechComment(t.TempDir(), "test-id", nil)
	if err == nil {
		t.Error("expected error when base URL not configured")
	}
	if err.Error() != "POLIS_BASE_URL not configured" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBeseechComment_CommentNotFound(t *testing.T) {
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	DiscoveryURL = "https://discovery.example.com"
	DiscoveryKey = "test-key"
	BaseURL = "https://test.polis.pub"

	dataDir := t.TempDir()

	_, err := BeseechComment(dataDir, "nonexistent-id", nil)
	if err == nil {
		t.Error("expected error when comment not found")
	}
}
