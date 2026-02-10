package publish

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterPost_NotConfigured(t *testing.T) {
	// Reset config
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	DiscoveryURL = ""
	DiscoveryKey = ""
	BaseURL = ""

	result := &PublishResult{
		Success: true,
		Path:    "posts/20260201/test.md",
		Title:   "Test Post",
		Version: "sha256:abc123",
	}

	// Should silently return nil when not configured
	err := RegisterPost(t.TempDir(), result, nil)
	if err != nil {
		t.Errorf("expected nil error when not configured, got: %v", err)
	}
}

func TestRegisterPost_NoEmail(t *testing.T) {
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	DiscoveryURL = "https://discovery.example.com"
	DiscoveryKey = "test-key"
	BaseURL = "https://test.polis.pub"

	dataDir := t.TempDir()

	// Create .well-known/polis WITHOUT email
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)
	wk := map[string]interface{}{
		"public_key": "ssh-ed25519 AAAA...",
	}
	data, _ := json.MarshalIndent(wk, "", "  ")
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), data, 0644)

	result := &PublishResult{
		Success: true,
		Path:    "posts/20260201/test.md",
		Title:   "Test Post",
		Version: "sha256:abc123",
	}

	err := RegisterPost(dataDir, result, nil)
	if err == nil {
		t.Error("expected error when email is missing")
	}
}

func TestRegisterPost_PartialConfig(t *testing.T) {
	oldURL, oldKey, oldBase := DiscoveryURL, DiscoveryKey, BaseURL
	defer func() { DiscoveryURL, DiscoveryKey, BaseURL = oldURL, oldKey, oldBase }()

	tests := []struct {
		name string
		url  string
		key  string
		base string
	}{
		{"no URL", "", "key", "https://test.polis.pub"},
		{"no key", "https://discovery.example.com", "", "https://test.polis.pub"},
		{"no base URL", "https://discovery.example.com", "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DiscoveryURL = tt.url
			DiscoveryKey = tt.key
			BaseURL = tt.base

			err := RegisterPost(t.TempDir(), &PublishResult{}, nil)
			if err != nil {
				t.Errorf("expected nil when partially configured, got: %v", err)
			}
		})
	}
}
