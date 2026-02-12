package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

// ============================================================================
// Site Validation Tests (replaces recoverConfig tests)
// ============================================================================

func TestSiteValidation_EmptyDirectory(t *testing.T) {
	dataDir := t.TempDir()

	// Create required directories but no polis files
	os.MkdirAll(filepath.Join(dataDir, ".polis", "keys"), 0755)

	result := site.Validate(dataDir)

	// Empty directory should return not_found or incomplete
	if result.Status == site.StatusValid {
		t.Error("Expected status to be not_found or incomplete, got valid")
	}
}

func TestSiteValidation_ValidSite(t *testing.T) {
	dataDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "keys"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create keys (use dummy values for testing)
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest polis-local"
	privKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(privKeyPath, []byte("fake-private-key"), 0600)
	os.WriteFile(pubKeyPath, []byte(pubKey), 0644)

	// Create .well-known/polis with matching public key
	wellKnown := map[string]string{
		"subdomain":  "testsite",
		"base_url":   "https://testsite.polis.pub",
		"public_key": pubKey,
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	result := site.Validate(dataDir)

	if result.Status != site.StatusValid {
		t.Errorf("Expected status valid, got %s", result.Status)
		for _, err := range result.Errors {
			t.Logf("Error: %s - %s", err.Code, err.Message)
		}
	}
}

func TestSiteValidation_MissingPrivateKey(t *testing.T) {
	dataDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "keys"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create only public key (no private key)
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest polis-local"
	pubKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(pubKeyPath, []byte(pubKey), 0644)

	// Create .well-known/polis
	wellKnown := map[string]string{
		"public_key": pubKey,
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	result := site.Validate(dataDir)

	if result.Status == site.StatusValid {
		t.Error("Expected incomplete status when private key missing")
	}

	// Check for specific error
	found := false
	for _, err := range result.Errors {
		if err.Code == "PRIVATE_KEY_MISSING" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected PRIVATE_KEY_MISSING error")
	}
}

func TestSiteValidation_MissingWellKnown(t *testing.T) {
	dataDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "keys"), 0755)

	// Create keys but no .well-known/polis
	privKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(privKeyPath, []byte("fake-private-key"), 0600)
	os.WriteFile(pubKeyPath, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest polis-local"), 0644)

	result := site.Validate(dataDir)

	if result.Status == site.StatusValid {
		t.Error("Expected incomplete status when .well-known/polis missing")
	}

	// Check for specific error
	found := false
	for _, err := range result.Errors {
		if err.Code == "WELLKNOWN_MISSING" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WELLKNOWN_MISSING error")
	}
}

func TestSiteValidation_PublicKeyMismatch(t *testing.T) {
	dataDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "keys"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create keys
	pubKeyFile := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKeyFile polis-local"
	pubKeyWellKnown := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDifferent polis-local"
	privKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(dataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(privKeyPath, []byte("fake-private-key"), 0600)
	os.WriteFile(pubKeyPath, []byte(pubKeyFile), 0644)

	// Create .well-known/polis with DIFFERENT public key
	wellKnown := map[string]string{
		"public_key": pubKeyWellKnown,
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	result := site.Validate(dataDir)

	if result.Status == site.StatusValid {
		t.Error("Expected incomplete status when public keys don't match")
	}

	// Check for specific error
	found := false
	for _, err := range result.Errors {
		if err.Code == "PUBLIC_KEY_MISMATCH" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected PUBLIC_KEY_MISMATCH error")
	}
}

// ============================================================================
// getSiteTitle Tests
// ============================================================================

func TestGetSiteTitle_FromWellKnownPolis(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create .well-known/polis with site_title set
	wellKnown := map[string]string{
		"subdomain":  "testsite",
		"base_url":   "https://testsite.polis.pub",
		"site_title": "My Awesome Blog",
		"public_key": "ssh-ed25519 test",
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	server := &Server{DataDir: dataDir}
	title := server.GetSiteTitle()

	if title != "My Awesome Blog" {
		t.Errorf("Expected site_title 'My Awesome Blog', got '%s'", title)
	}
}

func TestGetSiteTitle_FallbackToBaseURL(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create .well-known/polis with empty site_title (should fall back to base_url)
	wellKnown := map[string]string{
		"subdomain":  "testsite",
		"base_url":   "https://testsite.polis.pub",
		"site_title": "",
		"public_key": "ssh-ed25519 test",
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	server := &Server{DataDir: dataDir}
	title := server.GetSiteTitle()

	if title != "https://testsite.polis.pub" {
		t.Errorf("Expected fallback to base_url 'https://testsite.polis.pub', got '%s'", title)
	}
}

func TestGetSiteTitle_FallbackToSubdomain(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create .well-known/polis with no site_title and no base_url
	wellKnown := map[string]string{
		"subdomain":  "testsite",
		"public_key": "ssh-ed25519 test",
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	server := &Server{DataDir: dataDir}
	title := server.GetSiteTitle()

	if title != "https://testsite.polis.pub" {
		t.Errorf("Expected fallback to constructed URL 'https://testsite.polis.pub', got '%s'", title)
	}
}

func TestGetSiteTitle_NoWellKnownPolis_FallbackToConfig(t *testing.T) {
	dataDir := t.TempDir()
	// No .well-known/polis file

	server := &Server{
		DataDir: dataDir,
		Config: &Config{
			Subdomain: "configsite",
		},
	}
	title := server.GetSiteTitle()

	if title != "https://configsite.polis.pub" {
		t.Errorf("Expected fallback to config subdomain 'https://configsite.polis.pub', got '%s'", title)
	}
}

func TestGetSiteTitle_FallbackToPolisBaseURL(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".well-known"), 0755)

	// Create .well-known/polis with no site_title, no base_url, no subdomain
	wellKnown := map[string]string{
		"public_key": "ssh-ed25519 test",
	}
	wellKnownData, _ := json.Marshal(wellKnown)
	os.WriteFile(filepath.Join(dataDir, ".well-known", "polis"), wellKnownData, 0644)

	server := &Server{DataDir: dataDir, BaseURL: "https://mysite.example.com"}
	title := server.GetSiteTitle()

	if title != "https://mysite.example.com" {
		t.Errorf("Expected fallback to POLIS_BASE_URL 'https://mysite.example.com', got '%s'", title)
	}
}

func TestGetSiteTitle_NoWellKnownPolis_NoConfig(t *testing.T) {
	dataDir := t.TempDir()
	// No .well-known/polis file, no config

	server := &Server{DataDir: dataDir}
	title := server.GetSiteTitle()

	if title != "" {
		t.Errorf("Expected empty string when no .well-known/polis and no config, got '%s'", title)
	}
}

func TestCursorGreater(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"5", "4", true},
		{"4", "5", false},
		{"30", "4", true},  // was broken with string comparison
		{"4", "30", false},
		{"100", "9", true}, // multi-digit > single-digit
		{"9", "100", false},
		{"0", "0", false},
		{"", "", false},
		{"abc", "def", false}, // non-numeric fallback
	}
	for _, tt := range tests {
		got := cursorGreater(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("cursorGreater(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
