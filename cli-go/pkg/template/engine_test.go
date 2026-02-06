package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVariableSubstitution(t *testing.T) {
	engine := New(Config{})
	ctx := NewRenderContext()
	ctx.Title = "My Post"
	ctx.SiteTitle = "My Site"
	ctx.Year = "2026"

	template := `<title>{{title}} - {{site_title}}</title>
<footer>&copy; {{year}}</footer>`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "My Post - My Site") {
		t.Errorf("Expected title substitution, got: %s", result)
	}

	if !strings.Contains(result, "&copy; 2026") {
		t.Errorf("Expected year substitution, got: %s", result)
	}
}

func TestUnknownVariablePassthrough(t *testing.T) {
	engine := New(Config{})
	ctx := NewRenderContext()

	template := `Hello {{unknown_var}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if result != "Hello {{unknown_var}}" {
		t.Errorf("Expected unknown variable to pass through, got: %s", result)
	}
}

func TestPostsSection(t *testing.T) {
	engine := New(Config{})
	ctx := NewRenderContext()
	ctx.Posts = []PostData{
		{URL: "/posts/1.html", Title: "First Post", PublishedHuman: "January 1, 2026"},
		{URL: "/posts/2.html", Title: "Second Post", PublishedHuman: "January 2, 2026"},
	}

	template := `{{#posts}}<a href="{{url}}">{{title}}</a>{{/posts}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, `<a href="/posts/1.html">First Post</a>`) {
		t.Errorf("Expected first post, got: %s", result)
	}

	if !strings.Contains(result, `<a href="/posts/2.html">Second Post</a>`) {
		t.Errorf("Expected second post, got: %s", result)
	}
}

func TestBlessedCommentsSection(t *testing.T) {
	engine := New(Config{})
	ctx := NewRenderContext()
	ctx.BlessedComments = []BlessedCommentData{
		{URL: "/comment1", AuthorName: "Alice", Content: "<p>Great post!</p>"},
		{URL: "/comment2", AuthorName: "Bob", Content: "<p>Thanks for sharing</p>"},
	}

	template := `{{#blessed_comments}}<div class="comment">{{author_name}}: {{content}}</div>{{/blessed_comments}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Alice: <p>Great post!</p>") {
		t.Errorf("Expected Alice's comment, got: %s", result)
	}

	if !strings.Contains(result, "Bob: <p>Thanks for sharing</p>") {
		t.Errorf("Expected Bob's comment, got: %s", result)
	}
}

func TestCommentsSection(t *testing.T) {
	engine := New(Config{})
	ctx := NewRenderContext()
	ctx.Comments = []CommentData{
		{URL: "/comments/1.html", TargetAuthor: "alice.com", PublishedHuman: "January 1, 2026", Preview: "Great post!"},
	}

	template := `{{#comments}}<span>{{target_author}}</span> {{preview}}{{/comments}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<span>alice.com</span> Great post!") {
		t.Errorf("Expected comment variables substituted, got: %s", result)
	}
}

func TestCommentsSectionViaPartial(t *testing.T) {
	// Regression test: comment loop variables must resolve through partials.
	// The index template uses {{> theme:comment-item}} inside {{#comments}},
	// so target_author and preview must be available after partial expansion.
	tempDir := t.TempDir()
	themeDir := filepath.Join(tempDir, ".polis", "themes", "turbo", "snippets")
	os.MkdirAll(themeDir, 0755)
	os.WriteFile(filepath.Join(themeDir, "comment-item.html"), []byte(
		`<span class="author">{{target_author}}</span> <span class="preview">{{preview}}</span>`), 0644)

	engine := New(Config{
		DataDir:     tempDir,
		ActiveTheme: "turbo",
	})
	ctx := NewRenderContext()
	ctx.Comments = []CommentData{
		{URL: "/c/1.html", TargetAuthor: "bob.example.com", PublishedHuman: "Feb 6, 2026", Preview: "Nice work"},
	}

	template := `{{#comments}}{{> theme:comment-item}}{{/comments}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(result, "{{target_author}}") {
		t.Errorf("target_author was not substituted: %s", result)
	}
	if strings.Contains(result, "{{preview}}") {
		t.Errorf("preview was not substituted: %s", result)
	}
	if !strings.Contains(result, "bob.example.com") {
		t.Errorf("Expected target_author value, got: %s", result)
	}
	if !strings.Contains(result, "Nice work") {
		t.Errorf("Expected preview value, got: %s", result)
	}
}

func TestPartialLoading(t *testing.T) {
	// Create temp directory with snippets
	tempDir := t.TempDir()
	snippetsDir := filepath.Join(tempDir, "snippets")
	os.MkdirAll(snippetsDir, 0755)

	// Create a test snippet
	os.WriteFile(filepath.Join(snippetsDir, "about.html"), []byte("<p>About content</p>"), 0644)

	engine := New(Config{
		DataDir: tempDir,
	})
	ctx := NewRenderContext()

	template := `<div>{{> about}}</div>`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<p>About content</p>") {
		t.Errorf("Expected snippet content, got: %s", result)
	}
}

// mockMarkdownRenderer is a simple markdown renderer for testing.
func mockMarkdownRenderer(md string) (string, error) {
	// Very simple markdown simulation for testing
	result := md
	result = strings.ReplaceAll(result, "# ", "<h1 id=\"test\">")
	result = strings.ReplaceAll(result, "\n\n", "</h1>\n<p>")
	result = strings.ReplaceAll(result, "*", "<em>")
	if strings.Contains(result, "<p>") {
		result += "</em></p>"
	}
	return result, nil
}

func TestPartialWithMarkdown(t *testing.T) {
	// Create temp directory with snippets
	tempDir := t.TempDir()
	snippetsDir := filepath.Join(tempDir, "snippets")
	os.MkdirAll(snippetsDir, 0755)

	// Create a markdown snippet
	os.WriteFile(filepath.Join(snippetsDir, "intro.md"), []byte("# Hello\n\nThis is *intro*"), 0644)

	engine := New(Config{
		DataDir:          tempDir,
		MarkdownRenderer: mockMarkdownRenderer,
	})
	ctx := NewRenderContext()

	template := `{{> intro}}`

	result, err := engine.Render(template, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Markdown should be rendered to HTML
	if !strings.Contains(result, "<h1") || !strings.Contains(result, ">Hello") {
		t.Errorf("Expected h1 heading, got: %s", result)
	}

	if !strings.Contains(result, "<em>") {
		t.Errorf("Expected em text, got: %s", result)
	}
}

func TestPartialResolutionOrder(t *testing.T) {
	// Create temp directory with both global and theme snippets
	tempDir := t.TempDir()

	// Global snippets
	globalDir := filepath.Join(tempDir, "snippets")
	os.MkdirAll(globalDir, 0755)
	os.WriteFile(filepath.Join(globalDir, "about.html"), []byte("GLOBAL ABOUT"), 0644)

	// Theme snippets
	themeDir := filepath.Join(tempDir, ".polis", "themes", "turbo", "snippets")
	os.MkdirAll(themeDir, 0755)
	os.WriteFile(filepath.Join(themeDir, "about.html"), []byte("THEME ABOUT"), 0644)

	engine := New(Config{
		DataDir:     tempDir,
		ActiveTheme: "turbo",
	})
	ctx := NewRenderContext()

	// Global-first (default)
	result, _ := engine.Render(`{{> about}}`, ctx)
	if !strings.Contains(result, "GLOBAL ABOUT") {
		t.Errorf("Expected global snippet (default), got: %s", result)
	}

	// Explicit global
	result, _ = engine.Render(`{{> global:about}}`, ctx)
	if !strings.Contains(result, "GLOBAL ABOUT") {
		t.Errorf("Expected global snippet (explicit), got: %s", result)
	}

	// Theme-first
	result, _ = engine.Render(`{{> theme:about}}`, ctx)
	if !strings.Contains(result, "THEME ABOUT") {
		t.Errorf("Expected theme snippet, got: %s", result)
	}
}

func TestMarkers(t *testing.T) {
	content := WrapWithMarkers("<p>Hello</p>", "about.html", "global")

	if !strings.Contains(content, "POLIS-SNIPPET-START: global:about.html") {
		t.Errorf("Expected start marker, got: %s", content)
	}

	if !strings.Contains(content, `data-source="global"`) {
		t.Errorf("Expected data-source attribute, got: %s", content)
	}

	if !strings.Contains(content, "POLIS-SNIPPET-END: global:about.html") {
		t.Errorf("Expected end marker, got: %s", content)
	}
}

func TestStripMarkers(t *testing.T) {
	marked := `<!-- POLIS-SNIPPET-START: global:about.html path=about.html -->
<span class="polis-snippet-boundary" data-snippet="global:about.html" data-path="about.html" data-source="global" hidden></span>
<p>Hello</p>
<!-- POLIS-SNIPPET-END: global:about.html -->`

	stripped := StripMarkers(marked)

	if strings.Contains(stripped, "POLIS-SNIPPET") {
		t.Errorf("Expected markers to be stripped, got: %s", stripped)
	}

	if !strings.Contains(stripped, "<p>Hello</p>") {
		t.Errorf("Expected content to remain, got: %s", stripped)
	}
}

func TestMaxRecursionDepth(t *testing.T) {
	tempDir := t.TempDir()
	snippetsDir := filepath.Join(tempDir, "snippets")
	os.MkdirAll(snippetsDir, 0755)

	// Create recursive snippet
	os.WriteFile(filepath.Join(snippetsDir, "recursive.html"), []byte("{{> recursive}}"), 0644)

	engine := New(Config{
		DataDir: tempDir,
	})
	ctx := NewRenderContext()

	_, err := engine.Render(`{{> recursive}}`, ctx)
	if err == nil {
		t.Error("Expected error for infinite recursion")
	}

	if !strings.Contains(err.Error(), "maximum") {
		t.Errorf("Expected max depth error, got: %v", err)
	}
}

func TestFormatHumanDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-01-08T12:00:00Z", "January 8, 2026"},
		{"2026-12-25", "December 25, 2026"},
		{"invalid", "invalid"},
	}

	for _, tc := range tests {
		result := FormatHumanDate(tc.input)
		if result != tc.expected {
			t.Errorf("FormatHumanDate(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestTruncateSignature(t *testing.T) {
	sig := "AAAAC3NzaC1lZDI1NTE5AAAAIKs8y..."
	result := TruncateSignature(sig, 16)

	if len(result) != 16+3 { // 16 chars + "..."
		t.Errorf("Expected truncated signature of length 19, got: %d", len(result))
	}

	if !strings.HasSuffix(result, "...") {
		t.Errorf("Expected truncated signature to end with ..., got: %s", result)
	}
}
