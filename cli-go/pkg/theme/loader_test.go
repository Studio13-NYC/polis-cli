package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestTheme(t *testing.T, themeDir, themeName string) {
	t.Helper()
	dir := filepath.Join(themeDir, themeName)
	os.MkdirAll(dir, 0755)

	// Create required templates
	templates := map[string]string{
		"post.html":           "<html>{{title}}</html>",
		"comment.html":        "<html>{{title}}</html>",
		"comment-inline.html": "<div>{{content}}</div>",
		"index.html":          "<html>{{site_title}}</html>",
	}

	for name, content := range templates {
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	}

	// Create CSS file
	os.WriteFile(filepath.Join(dir, themeName+".css"), []byte("body {}"), 0644)
}

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()
	themesDir := filepath.Join(tempDir, ".polis", "themes")
	createTestTheme(t, themesDir, "turbo")

	templates, err := Load(tempDir, "", "turbo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if templates.Post == "" {
		t.Error("Expected post template to be loaded")
	}

	if templates.Index == "" {
		t.Error("Expected index template to be loaded")
	}
}

func TestLoadFallbackToCLI(t *testing.T) {
	dataDir := t.TempDir()
	cliThemesDir := t.TempDir()

	// Only create theme in CLI dir (not local)
	createTestTheme(t, cliThemesDir, "sols")

	templates, err := Load(dataDir, cliThemesDir, "sols")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if templates.Post == "" {
		t.Error("Expected post template from CLI fallback")
	}
}

func TestLoadMissingTheme(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Load(tempDir, "", "nonexistent")
	if err == nil {
		t.Error("Expected error for missing theme")
	}
}

func TestManifest(t *testing.T) {
	tempDir := t.TempDir()

	// Create metadata directory
	os.MkdirAll(filepath.Join(tempDir, "metadata"), 0755)

	// Save manifest
	manifest := &Manifest{
		Version:      "0.1.0",
		ActiveTheme:  "turbo",
		PostCount:    5,
		CommentCount: 3,
	}

	err := SaveManifest(tempDir, manifest)
	if err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	// Load manifest
	loaded, err := LoadManifest(tempDir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if loaded.ActiveTheme != "turbo" {
		t.Errorf("Expected active_theme 'turbo', got '%s'", loaded.ActiveTheme)
	}

	if loaded.PostCount != 5 {
		t.Errorf("Expected post_count 5, got %d", loaded.PostCount)
	}
}

func TestGetActiveTheme(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "metadata"), 0755)

	// Set active theme
	err := SetActiveTheme(tempDir, "zane")
	if err != nil {
		t.Fatalf("SetActiveTheme failed: %v", err)
	}

	// Get active theme
	theme, err := GetActiveTheme(tempDir)
	if err != nil {
		t.Fatalf("GetActiveTheme failed: %v", err)
	}

	if theme != "zane" {
		t.Errorf("Expected 'zane', got '%s'", theme)
	}
}

func TestCopyCSS(t *testing.T) {
	tempDir := t.TempDir()
	themesDir := filepath.Join(tempDir, ".polis", "themes")
	createTestTheme(t, themesDir, "turbo")

	err := CopyCSS(tempDir, "", "turbo")
	if err != nil {
		t.Fatalf("CopyCSS failed: %v", err)
	}

	// Verify CSS was copied
	cssPath := filepath.Join(tempDir, "styles.css")
	if _, err := os.Stat(cssPath); err != nil {
		t.Errorf("Expected styles.css to exist: %v", err)
	}
}

func TestListThemes(t *testing.T) {
	dataDir := t.TempDir()
	cliThemesDir := t.TempDir()

	// Create local theme
	createTestTheme(t, filepath.Join(dataDir, ".polis", "themes"), "local-theme")

	// Create CLI themes
	createTestTheme(t, cliThemesDir, "cli-theme1")
	createTestTheme(t, cliThemesDir, "cli-theme2")

	themes, err := ListThemes(dataDir, cliThemesDir)
	if err != nil {
		t.Fatalf("ListThemes failed: %v", err)
	}

	if len(themes) != 3 {
		t.Errorf("Expected 3 themes, got %d: %v", len(themes), themes)
	}
}

func TestCalculateCSSPath(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"index.html", "styles.css"},
		{"posts/2026/01/my-post.html", "../../../styles.css"},
		{"comments/2026/01/reply.html", "../../../styles.css"},
		{"about.html", "styles.css"},
	}

	for _, tc := range tests {
		result := CalculateCSSPath(tc.filePath)
		if result != tc.expected {
			t.Errorf("CalculateCSSPath(%q) = %q, want %q", tc.filePath, result, tc.expected)
		}
	}
}

func TestCalculateHomePath(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"index.html", "index.html"},
		{"posts/2026/01/my-post.html", "../../../index.html"},
		{"about.html", "index.html"},
	}

	for _, tc := range tests {
		result := CalculateHomePath(tc.filePath)
		if result != tc.expected {
			t.Errorf("CalculateHomePath(%q) = %q, want %q", tc.filePath, result, tc.expected)
		}
	}
}
