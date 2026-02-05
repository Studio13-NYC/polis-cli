package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewPageRenderer(t *testing.T) {
	// Create temp site structure
	tempDir := t.TempDir()
	setupTestSite(t, tempDir)

	renderer, err := NewPageRenderer(PageConfig{
		DataDir:       tempDir,
		BaseURL:       "https://example.com",
		RenderMarkers: false,
	})
	if err != nil {
		t.Fatalf("NewPageRenderer failed: %v", err)
	}

	if renderer == nil {
		t.Fatal("NewPageRenderer returned nil")
	}
}

func TestRenderFile_Post(t *testing.T) {
	tempDir := t.TempDir()
	setupTestSite(t, tempDir)

	// Create a test post
	postsDir := filepath.Join(tempDir, "posts")
	os.MkdirAll(postsDir, 0755)

	postContent := `---
title: Test Post
published: 2026-01-15T12:00:00Z
---
This is the **test** content.
`
	postPath := filepath.Join(postsDir, "test-post.md")
	os.WriteFile(postPath, []byte(postContent), 0644)

	renderer, err := NewPageRenderer(PageConfig{
		DataDir:       tempDir,
		BaseURL:       "https://example.com",
		RenderMarkers: false,
	})
	if err != nil {
		t.Fatalf("NewPageRenderer failed: %v", err)
	}

	html, rendered, err := renderer.RenderFile("posts/test-post.md", "post", true)
	if err != nil {
		t.Fatalf("RenderFile failed: %v", err)
	}

	if !rendered {
		t.Error("Expected file to be rendered")
	}

	if !strings.Contains(html, "Test Post") {
		t.Errorf("Expected HTML to contain title, got: %s", html)
	}

	if !strings.Contains(html, "<strong>test</strong>") {
		t.Errorf("Expected HTML to contain bold text, got: %s", html)
	}

	// Verify HTML file was written
	htmlPath := filepath.Join(postsDir, "test-post.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Error("Expected HTML file to be created")
	}
}

func TestRenderFile_Skip(t *testing.T) {
	tempDir := t.TempDir()
	setupTestSite(t, tempDir)

	postsDir := filepath.Join(tempDir, "posts")
	os.MkdirAll(postsDir, 0755)

	postContent := `---
title: Skip Test
---
Content.
`
	postPath := filepath.Join(postsDir, "skip-test.md")
	os.WriteFile(postPath, []byte(postContent), 0644)

	renderer, _ := NewPageRenderer(PageConfig{
		DataDir: tempDir,
	})

	// First render with force
	_, rendered, err := renderer.RenderFile("posts/skip-test.md", "post", true)
	if err != nil {
		t.Fatalf("First render failed: %v", err)
	}
	if !rendered {
		t.Error("First render should render")
	}

	// Touch the HTML file to make it newer than MD
	htmlPath := filepath.Join(postsDir, "skip-test.html")
	futureTime := time.Now().Add(time.Second)
	os.Chtimes(htmlPath, futureTime, futureTime)

	// Second render without force should skip (HTML is newer)
	_, rendered, err = renderer.RenderFile("posts/skip-test.md", "post", false)
	if err != nil {
		t.Fatalf("Second render failed: %v", err)
	}
	if rendered {
		t.Error("Second render should skip (HTML is up to date)")
	}
}

func TestRenderIndex(t *testing.T) {
	tempDir := t.TempDir()
	setupTestSite(t, tempDir)

	// Create metadata/public.jsonl with some entries
	metadataDir := filepath.Join(tempDir, "metadata")
	os.MkdirAll(metadataDir, 0755)
	publicJSONL := `{"path":"posts/hello.md","title":"Hello World","published":"2026-01-15T12:00:00Z","type":"post"}
{"path":"posts/goodbye.md","title":"Goodbye World","published":"2026-01-16T12:00:00Z","type":"post"}
`
	os.WriteFile(filepath.Join(metadataDir, "public.jsonl"), []byte(publicJSONL), 0644)

	renderer, _ := NewPageRenderer(PageConfig{
		DataDir:       tempDir,
		BaseURL:       "https://example.com",
		RenderMarkers: false,
	})

	err := renderer.RenderIndex()
	if err != nil {
		t.Fatalf("RenderIndex failed: %v", err)
	}

	// Verify index.html was created
	indexPath := filepath.Join(tempDir, "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index.html: %v", err)
	}

	// The index template iterates over posts, check the site title is there
	if !strings.Contains(string(content), "Test Site") {
		t.Errorf("Expected index to contain site title, got: %s", content)
	}

	// Check that posts section was processed (URL should be converted to .html)
	if !strings.Contains(string(content), "posts/hello.html") {
		t.Errorf("Expected index to contain post URL, got: %s", content)
	}
}

func TestRenderAll(t *testing.T) {
	tempDir := t.TempDir()
	setupTestSite(t, tempDir)

	// Create posts directory with a post
	postsDir := filepath.Join(tempDir, "posts")
	os.MkdirAll(postsDir, 0755)
	os.WriteFile(filepath.Join(postsDir, "post1.md"), []byte("---\ntitle: Post 1\n---\nContent 1"), 0644)

	// Create metadata/public.jsonl
	metadataDir := filepath.Join(tempDir, "metadata")
	os.MkdirAll(metadataDir, 0755)
	os.WriteFile(filepath.Join(metadataDir, "public.jsonl"), []byte(`{"path":"posts/post1.md","title":"Post 1","type":"post"}`), 0644)

	renderer, _ := NewPageRenderer(PageConfig{
		DataDir: tempDir,
	})

	stats, err := renderer.RenderAll(true)
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	if stats.PostsRendered != 1 {
		t.Errorf("Expected 1 post rendered, got %d", stats.PostsRendered)
	}

	if !stats.IndexGenerated {
		t.Error("Expected index to be generated")
	}
}

// setupTestSite creates a minimal polis site structure for testing.
func setupTestSite(t *testing.T, dir string) {
	t.Helper()

	// Create .well-known/polis
	wellKnownDir := filepath.Join(dir, ".well-known")
	os.MkdirAll(wellKnownDir, 0755)
	os.WriteFile(filepath.Join(wellKnownDir, "polis"), []byte(`{
		"base_url": "https://example.com",
		"site_title": "Test Site",
		"author_name": "Test Author"
	}`), 0644)

	// Create .polis/themes/turbo with minimal templates
	themesDir := filepath.Join(dir, ".polis", "themes", "turbo")
	os.MkdirAll(themesDir, 0755)

	postTemplate := `<!DOCTYPE html>
<html>
<head><title>{{title}} - {{site_title}}</title></head>
<body>
<h1>{{title}}</h1>
<div class="content">{{content}}</div>
</body>
</html>`
	os.WriteFile(filepath.Join(themesDir, "post.html"), []byte(postTemplate), 0644)
	os.WriteFile(filepath.Join(themesDir, "comment.html"), []byte(postTemplate), 0644)
	os.WriteFile(filepath.Join(themesDir, "comment-inline.html"), []byte(`<div>{{content}}</div>`), 0644)

	indexTemplate := `<!DOCTYPE html>
<html>
<head><title>{{site_title}}</title></head>
<body>
<h1>{{site_title}}</h1>
{{#posts}}<div><a href="{{url}}">{{title}}</a></div>{{/posts}}
</body>
</html>`
	os.WriteFile(filepath.Join(themesDir, "index.html"), []byte(indexTemplate), 0644)

	// Create CSS file (required by RenderAll)
	os.WriteFile(filepath.Join(themesDir, "turbo.css"), []byte("/* test css */"), 0644)

	// Create empty metadata/public.jsonl
	metadataDir := filepath.Join(dir, "metadata")
	os.MkdirAll(metadataDir, 0755)
	os.WriteFile(filepath.Join(metadataDir, "public.jsonl"), []byte(""), 0644)
}
