package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// TestRotateKeyUpdatesWellKnown verifies the JSON update logic used by rotate-key
// to update .well-known/polis with the new public key.
func TestRotateKeyUpdatesWellKnown(t *testing.T) {
	dir := t.TempDir()

	// Set up a minimal .well-known/polis
	wellKnownDir := filepath.Join(dir, ".well-known")
	if err := os.MkdirAll(wellKnownDir, 0755); err != nil {
		t.Fatal(err)
	}

	originalWK := map[string]interface{}{
		"base_url":    "https://example.com",
		"public_key":  "ssh-ed25519 AAAA_OLD_KEY test@example.com",
		"site_title":  "My Site",
		"author_name": "Test Author",
		"email":       "test@example.com",
	}
	data, _ := json.MarshalIndent(originalWK, "", "  ")
	wellKnownPath := filepath.Join(wellKnownDir, "polis")
	if err := os.WriteFile(wellKnownPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Generate a new keypair
	_, pubSSH, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the rotate-key update logic
	wkData, err := os.ReadFile(wellKnownPath)
	if err != nil {
		t.Fatal(err)
	}

	var wkJSON map[string]interface{}
	if err := json.Unmarshal(wkData, &wkJSON); err != nil {
		t.Fatal(err)
	}
	wkJSON["public_key"] = strings.TrimSpace(string(pubSSH))
	updatedWK, err := json.MarshalIndent(wkJSON, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wellKnownPath, append(updatedWK, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	// Read back and verify
	readBack, err := os.ReadFile(wellKnownPath)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(readBack, &result); err != nil {
		t.Fatalf("failed to parse updated .well-known/polis: %v", err)
	}

	// Verify the public key was updated
	newKey, ok := result["public_key"].(string)
	if !ok {
		t.Fatal("public_key not found in updated .well-known/polis")
	}
	if newKey == "ssh-ed25519 AAAA_OLD_KEY test@example.com" {
		t.Error("public_key was not updated - still has old value")
	}
	if !strings.HasPrefix(newKey, "ssh-ed25519 ") {
		t.Errorf("expected public_key to start with 'ssh-ed25519 ', got: %s", newKey)
	}

	// Verify other fields were preserved
	if result["base_url"] != "https://example.com" {
		t.Errorf("base_url was modified: %v", result["base_url"])
	}
	if result["site_title"] != "My Site" {
		t.Errorf("site_title was modified: %v", result["site_title"])
	}
	if result["author_name"] != "Test Author" {
		t.Errorf("author_name was modified: %v", result["author_name"])
	}
	if result["email"] != "test@example.com" {
		t.Errorf("email was modified: %v", result["email"])
	}
}
