package site

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// InitOptions contains options for initializing a new polis site.
type InitOptions struct {
	SiteTitle string // Optional site title
	Author    string // Optional author name
	Email     string // Optional author email
	BaseURL   string // Optional base URL (e.g., https://alice.polis.pub)
}

// InitResult contains the result of site initialization.
type InitResult struct {
	Success   bool   `json:"success"`
	SiteDir   string `json:"site_dir"`
	PublicKey string `json:"public_key"`
	Error     string `json:"error,omitempty"`
}

// Init creates a new polis site in the given directory.
// This function has no webapp dependencies and can be used by CLI tools.
// It will NOT overwrite existing keys or .well-known/polis - this is a safety feature.
func Init(siteDir string, opts InitOptions) (*InitResult, error) {
	result := &InitResult{
		Success: false,
		SiteDir: siteDir,
	}

	// SAFETY: Check if keys already exist - refuse to overwrite
	privKeyPath := filepath.Join(siteDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(siteDir, ".polis", "keys", "id_ed25519.pub")
	wellKnownPath := filepath.Join(siteDir, ".well-known", "polis")

	if _, err := os.Stat(privKeyPath); err == nil {
		return nil, fmt.Errorf("private key already exists at %s - refusing to overwrite", privKeyPath)
	}
	if _, err := os.Stat(pubKeyPath); err == nil {
		return nil, fmt.Errorf("public key already exists at %s - refusing to overwrite", pubKeyPath)
	}
	if _, err := os.Stat(wellKnownPath); err == nil {
		return nil, fmt.Errorf(".well-known/polis already exists at %s - refusing to overwrite", wellKnownPath)
	}

	// Create directory structure (matches CLI's init behavior)
	dirs := []string{
		siteDir,
		// Private directories (matches CLI)
		filepath.Join(siteDir, ".polis"),
		filepath.Join(siteDir, ".polis", "keys"),
		filepath.Join(siteDir, ".polis", "themes"),
		filepath.Join(siteDir, ".polis", "drafts"),
		filepath.Join(siteDir, ".polis", "comments", "drafts"),
		filepath.Join(siteDir, ".polis", "comments", "pending"),
		filepath.Join(siteDir, ".polis", "comments", "denied"),
		// Public directories (matches CLI)
		filepath.Join(siteDir, ".well-known"),
		filepath.Join(siteDir, "posts"),
		filepath.Join(siteDir, "comments"),
		filepath.Join(siteDir, "snippets"),
		filepath.Join(siteDir, "metadata"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate keypair
	privKey, pubKey, err := signing.GenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Save keys with appropriate permissions
	if err := os.WriteFile(privKeyPath, privKey, 0600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}
	if err := os.WriteFile(pubKeyPath, pubKey, 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %w", err)
	}

	// Create .well-known/polis
	setupTime := time.Now().UTC()

	// Get author/email from options or fall back to git config
	author := opts.Author
	if author == "" {
		author = getGitConfig("user.name")
	}
	email := opts.Email
	if email == "" {
		email = getGitConfig("user.email")
	}

	wk := &WellKnown{
		Version:   "0.1.0",
		Author:    author,
		Email:     email,
		PublicKey: strings.TrimSpace(string(pubKey)),
		SiteTitle: opts.SiteTitle,
		Created:   setupTime.Format(time.RFC3339),
		Config: &WellKnownConfig{
			Directories: WellKnownDirectories{
				Keys:     ".polis/keys",
				Posts:    "posts",
				Comments: "comments",
				Snippets: "snippets",
				Themes:   ".polis/themes",
				Versions: ".versions", // Directory name only; path constructed at runtime
			},
			Files: WellKnownFiles{
				PublicIndex:     "metadata/public.jsonl",
				BlessedComments: "metadata/blessed-comments.json",
				FollowingIndex:  "metadata/following.json",
			},
		},
		// Note: base_url is NOT stored in .well-known/polis
		// Use POLIS_BASE_URL env var instead (matches bash CLI behavior)
	}

	if err := SaveWellKnown(siteDir, wk); err != nil {
		return nil, fmt.Errorf("failed to create .well-known/polis: %w", err)
	}

	// Create metadata files (matches CLI initialization)
	if err := initMetadataFiles(siteDir); err != nil {
		return nil, fmt.Errorf("failed to create metadata files: %w", err)
	}

	// Create .gitignore (matches CLI format)
	if err := initGitignore(siteDir); err != nil {
		// Non-fatal, just log
		fmt.Printf("warning: failed to create .gitignore: %v\n", err)
	}

	result.Success = true
	result.PublicKey = strings.TrimSpace(string(pubKey))
	return result, nil
}

// initMetadataFiles creates the initial metadata files.
func initMetadataFiles(siteDir string) error {
	metadataDir := filepath.Join(siteDir, "metadata")

	// Create manifest.json if it doesn't exist
	manifestPath := filepath.Join(metadataDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifest := map[string]interface{}{
			"version":        "1.0",
			"last_published": "",
			"post_count":     0,
			"comment_count":  0,
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(manifestPath, data, 0644); err != nil {
			return err
		}
	}

	// Create following.json if it doesn't exist
	followingPath := filepath.Join(metadataDir, "following.json")
	if _, err := os.Stat(followingPath); os.IsNotExist(err) {
		following := map[string]interface{}{
			"version":   "1.0",
			"following": []interface{}{},
		}
		data, _ := json.MarshalIndent(following, "", "  ")
		if err := os.WriteFile(followingPath, data, 0644); err != nil {
			return err
		}
	}

	// Create blessed-comments.json if it doesn't exist
	blessedPath := filepath.Join(metadataDir, "blessed-comments.json")
	if _, err := os.Stat(blessedPath); os.IsNotExist(err) {
		blessed := map[string]interface{}{
			"version":  "1.0",
			"comments": []interface{}{},
		}
		data, _ := json.MarshalIndent(blessed, "", "  ")
		if err := os.WriteFile(blessedPath, data, 0644); err != nil {
			return err
		}
	}

	// Create empty public.jsonl if it doesn't exist
	publicPath := filepath.Join(metadataDir, "public.jsonl")
	if _, err := os.Stat(publicPath); os.IsNotExist(err) {
		if err := os.WriteFile(publicPath, []byte{}, 0644); err != nil {
			return err
		}
	}

	return nil
}

// initGitignore creates the .gitignore file.
func initGitignore(siteDir string) error {
	gitignorePath := filepath.Join(siteDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := `.polis/
/themes
/polis
/polis-tui
.env*
`
		return os.WriteFile(gitignorePath, []byte(content), 0644)
	}
	return nil
}

// getGitConfig retrieves a git config value.
// Returns empty string if git is not available or the config key is not set.
func getGitConfig(key string) string {
	cmd := exec.Command("git", "config", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
