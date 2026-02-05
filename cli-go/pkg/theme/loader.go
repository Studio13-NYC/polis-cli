// Package theme provides theme loading and management for polis.
package theme

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Templates holds the loaded theme templates.
type Templates struct {
	Post          string // post.html - required
	Comment       string // comment.html - required
	CommentInline string // comment-inline.html - required
	Index         string // index.html - required
}

// Manifest represents the site manifest (metadata/manifest.json).
type Manifest struct {
	Version      string `json:"version"`
	ActiveTheme  string `json:"active_theme"`
	PostCount    int    `json:"post_count"`
	CommentCount int    `json:"comment_count"`
}

// Load loads templates from the active theme.
// It tries the local theme first (.polis/themes/{name}/), then falls back to CLI themes.
func Load(dataDir, cliThemesDir, themeName string) (*Templates, error) {
	if themeName == "" {
		return nil, fmt.Errorf("theme name is required")
	}

	// Try local theme first
	localThemeDir := filepath.Join(dataDir, ".polis", "themes", themeName)
	templates, err := loadFromDir(localThemeDir)
	if err == nil {
		return templates, nil
	}

	// Fall back to CLI themes
	if cliThemesDir != "" {
		cliThemeDir := filepath.Join(cliThemesDir, themeName)
		templates, err = loadFromDir(cliThemeDir)
		if err == nil {
			return templates, nil
		}
	}

	return nil, fmt.Errorf("theme %q not found in %s or %s", themeName, localThemeDir, cliThemesDir)
}

// loadFromDir loads all required templates from a theme directory.
func loadFromDir(themeDir string) (*Templates, error) {
	if _, err := os.Stat(themeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("theme directory not found: %s", themeDir)
	}

	templates := &Templates{}

	// Load required templates
	required := map[string]*string{
		"post.html":           &templates.Post,
		"comment.html":        &templates.Comment,
		"comment-inline.html": &templates.CommentInline,
		"index.html":          &templates.Index,
	}

	for filename, dest := range required {
		content, err := os.ReadFile(filepath.Join(themeDir, filename))
		if err != nil {
			return nil, fmt.Errorf("required template %q not found: %w", filename, err)
		}
		*dest = string(content)
	}

	return templates, nil
}

// GetActiveTheme returns the active theme name from the manifest.
// Returns empty string if no theme is set.
func GetActiveTheme(dataDir string) (string, error) {
	manifest, err := LoadManifest(dataDir)
	if err != nil {
		return "", err
	}
	return manifest.ActiveTheme, nil
}

// LoadManifest loads the site manifest from metadata/manifest.json.
func LoadManifest(dataDir string) (*Manifest, error) {
	manifestPath := filepath.Join(dataDir, "metadata", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// SaveManifest saves the site manifest to metadata/manifest.json.
func SaveManifest(dataDir string, manifest *Manifest) error {
	manifestPath := filepath.Join(dataDir, "metadata", "manifest.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// SetActiveTheme updates the active theme in the manifest.
func SetActiveTheme(dataDir, themeName string) error {
	manifest, err := LoadManifest(dataDir)
	if err != nil {
		// Create new manifest if it doesn't exist
		manifest = &Manifest{
			Version: "0.1.0",
		}
	}

	manifest.ActiveTheme = themeName

	return SaveManifest(dataDir, manifest)
}

// CopyCSS copies the theme's CSS file to styles.css at the site root.
// The CSS filename should match the theme name ({themename}.css).
func CopyCSS(dataDir, cliThemesDir, themeName string) error {
	cssFilename := themeName + ".css"

	// Try local theme first
	localCSSPath := filepath.Join(dataDir, ".polis", "themes", themeName, cssFilename)
	destPath := filepath.Join(dataDir, "styles.css")

	if _, err := os.Stat(localCSSPath); err == nil {
		return copyFile(localCSSPath, destPath)
	}

	// Fall back to CLI themes
	if cliThemesDir != "" {
		cliCSSPath := filepath.Join(cliThemesDir, themeName, cssFilename)
		if _, err := os.Stat(cliCSSPath); err == nil {
			return copyFile(cliCSSPath, destPath)
		}
	}

	return fmt.Errorf("CSS file not found: %s", cssFilename)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// ListThemes returns the names of all available themes.
// It combines themes from both local and CLI directories.
func ListThemes(dataDir, cliThemesDir string) ([]string, error) {
	themeSet := make(map[string]bool)

	// List local themes
	localThemesDir := filepath.Join(dataDir, ".polis", "themes")
	if entries, err := os.ReadDir(localThemesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && isValidTheme(filepath.Join(localThemesDir, entry.Name())) {
				themeSet[entry.Name()] = true
			}
		}
	}

	// List CLI themes
	if cliThemesDir != "" {
		if entries, err := os.ReadDir(cliThemesDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && isValidTheme(filepath.Join(cliThemesDir, entry.Name())) {
					themeSet[entry.Name()] = true
				}
			}
		}
	}

	// Convert to slice
	var themes []string
	for name := range themeSet {
		themes = append(themes, name)
	}

	return themes, nil
}

// isValidTheme checks if a directory contains a valid theme (has required templates).
func isValidTheme(themeDir string) bool {
	required := []string{"post.html", "comment.html", "comment-inline.html", "index.html"}
	for _, file := range required {
		if _, err := os.Stat(filepath.Join(themeDir, file)); err != nil {
			return false
		}
	}
	return true
}

// GetThemeDir returns the path to a theme's directory.
// Returns the local theme path if it exists, otherwise the CLI theme path.
func GetThemeDir(dataDir, cliThemesDir, themeName string) string {
	localDir := filepath.Join(dataDir, ".polis", "themes", themeName)
	if _, err := os.Stat(localDir); err == nil {
		return localDir
	}

	if cliThemesDir != "" {
		cliDir := filepath.Join(cliThemesDir, themeName)
		if _, err := os.Stat(cliDir); err == nil {
			return cliDir
		}
	}

	return ""
}

// CalculateCSSPath returns the relative path to styles.css from a given file path.
// For example:
// - posts/2026/01/post.html -> ../../../styles.css
// - comments/2026/01/comment.html -> ../../../styles.css
// - index.html -> styles.css
func CalculateCSSPath(filePath string) string {
	// Clean and normalize the path
	filePath = filepath.Clean(filePath)
	filePath = filepath.ToSlash(filePath) // Use forward slashes

	// Count directory depth
	parts := strings.Split(filePath, "/")
	depth := len(parts) - 1 // Subtract 1 for the filename

	if depth <= 0 {
		return "styles.css"
	}

	// Build relative path
	var prefix string
	for i := 0; i < depth; i++ {
		prefix += "../"
	}

	return prefix + "styles.css"
}

// CalculateHomePath returns the relative path to index.html from a given file path.
func CalculateHomePath(filePath string) string {
	// Clean and normalize the path
	filePath = filepath.Clean(filePath)
	filePath = filepath.ToSlash(filePath) // Use forward slashes

	// Count directory depth
	parts := strings.Split(filePath, "/")
	depth := len(parts) - 1 // Subtract 1 for the filename

	if depth <= 0 {
		return "index.html"
	}

	// Build relative path
	var prefix string
	for i := 0; i < depth; i++ {
		prefix += "../"
	}

	return prefix + "index.html"
}
