package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vdibart/polis-cli/cli-go/pkg/metadata"
	"github.com/vdibart/polis-cli/cli-go/pkg/template"
	"github.com/vdibart/polis-cli/cli-go/pkg/theme"
)

// PageConfig holds configuration for page rendering.
type PageConfig struct {
	DataDir       string // Site data directory
	CLIThemesDir  string // CLI themes directory (fallback)
	BaseURL       string // Site base URL
	RenderMarkers bool   // Add snippet markers for editing
}

// PageRenderer renders polis pages using templates.
type PageRenderer struct {
	config    PageConfig
	engine    *template.Engine
	templates *theme.Templates
	themeName string
}

// RenderStats holds statistics from a render operation.
type RenderStats struct {
	PostsRendered    int
	PostsSkipped     int
	CommentsRendered int
	CommentsSkipped  int
	IndexGenerated   bool
}

// NewPageRenderer creates a new page renderer.
func NewPageRenderer(cfg PageConfig) (*PageRenderer, error) {
	// Load active theme
	themeName, err := theme.GetActiveTheme(cfg.DataDir)
	if err != nil {
		// Default to turbo if no theme set
		themeName = "turbo"
	}

	// Load templates
	templates, err := theme.Load(cfg.DataDir, cfg.CLIThemesDir, themeName)
	if err != nil {
		return nil, fmt.Errorf("failed to load theme: %w", err)
	}

	// Create template engine with markdown renderer
	engine := template.New(template.Config{
		DataDir:          cfg.DataDir,
		CLIThemesDir:     cfg.CLIThemesDir,
		ActiveTheme:      themeName,
		RenderMarkers:    cfg.RenderMarkers,
		BaseURL:          cfg.BaseURL,
		MarkdownRenderer: MarkdownToHTML,
	})

	return &PageRenderer{
		config:    cfg,
		engine:    engine,
		templates: templates,
		themeName: themeName,
	}, nil
}

// RenderFile renders a single file (post or comment) to HTML.
// Returns the rendered HTML, whether it was rendered (vs skipped), and any error.
func (r *PageRenderer) RenderFile(path string, fileType string, force bool) (string, bool, error) {
	// Build full paths
	mdPath := filepath.Join(r.config.DataDir, path)
	htmlPath := strings.TrimSuffix(mdPath, ".md") + ".html"

	// Check if rendering is needed (unless force)
	if !force {
		mdInfo, err := os.Stat(mdPath)
		if err != nil {
			return "", false, fmt.Errorf("source file not found: %w", err)
		}

		htmlInfo, err := os.Stat(htmlPath)
		if err == nil && htmlInfo.ModTime().After(mdInfo.ModTime()) {
			// HTML is newer than MD, skip
			return "", false, nil
		}
	}

	// Read markdown file
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter
	fm := parseFrontmatter(string(content))
	body := stripFrontmatter(string(content))

	// Convert markdown to HTML
	htmlContent, err := MarkdownToHTML(body)
	if err != nil {
		return "", false, fmt.Errorf("failed to render markdown: %w", err)
	}

	// Build render context
	ctx := template.NewRenderContext()
	ctx.Title = fm["title"]
	ctx.Content = htmlContent
	ctx.Published = fm["published"]
	ctx.PublishedHuman = template.FormatHumanDate(fm["published"])
	ctx.URL = r.buildURL(path)
	ctx.Version = fm["current-version"]
	if ctx.Version == "" {
		ctx.Version = fm["version"]
	}
	ctx.SignatureShort = template.TruncateSignature(fm["signature"], 16)

	// Site info
	ctx.SiteURL = r.config.BaseURL
	ctx.SiteTitle = r.getSiteTitle()
	ctx.CSSPath = theme.CalculateCSSPath(path)
	ctx.HomePath = theme.CalculateHomePath(path)
	ctx.AuthorName = r.getAuthorName()
	ctx.AuthorURL = r.config.BaseURL

	// Comment-specific fields
	if fileType == "comment" {
		ctx.InReplyToURL = fm["in_reply_to"]
		if ctx.InReplyToURL == "" {
			// Try nested format
			ctx.InReplyToURL = parseNestedField(string(content), "in-reply-to", "url")
		}
		ctx.RootPostURL = fm["root_post"]
		if ctx.RootPostURL == "" {
			ctx.RootPostURL = parseNestedField(string(content), "in-reply-to", "root-post")
		}
	}

	// Load blessed comments for posts
	if fileType == "post" {
		blessedComments, _ := r.loadBlessedCommentsForPost(path)
		ctx.BlessedComments = blessedComments
		ctx.BlessedCount = len(blessedComments)
	}

	// Select template
	var tmpl string
	switch fileType {
	case "post":
		tmpl = r.templates.Post
	case "comment":
		tmpl = r.templates.Comment
	default:
		tmpl = r.templates.Post
	}

	// Render template
	rendered, err := r.engine.Render(tmpl, ctx)
	if err != nil {
		return "", false, fmt.Errorf("failed to render template: %w", err)
	}

	// Write output
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0755); err != nil {
		return "", false, fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(htmlPath, []byte(rendered), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write output: %w", err)
	}

	return rendered, true, nil
}

// RenderIndex generates the index.html page.
func (r *PageRenderer) RenderIndex() error {
	// Load posts and comments from public.jsonl
	posts, comments, err := r.loadPublicIndex()
	if err != nil {
		return fmt.Errorf("failed to load public index: %w", err)
	}

	// Build render context
	ctx := template.NewRenderContext()
	ctx.SiteURL = r.config.BaseURL
	ctx.SiteTitle = r.getSiteTitle()
	ctx.CSSPath = "styles.css"
	ctx.HomePath = "index.html"
	ctx.AuthorName = r.getAuthorName()
	ctx.AuthorURL = r.config.BaseURL
	ctx.PostCount = len(posts)
	ctx.CommentCount = len(comments)
	ctx.Posts = posts
	ctx.Comments = comments

	// Set recent posts (first 3)
	if len(posts) > 3 {
		ctx.RecentPosts = posts[:3]
	} else {
		ctx.RecentPosts = posts
	}

	// Render template
	rendered, err := r.engine.Render(r.templates.Index, ctx)
	if err != nil {
		return fmt.Errorf("failed to render index template: %w", err)
	}

	// Write output
	indexPath := filepath.Join(r.config.DataDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("failed to write index.html: %w", err)
	}

	return nil
}

// RenderAll renders all posts and comments, and generates the index.
func (r *PageRenderer) RenderAll(force bool) (*RenderStats, error) {
	stats := &RenderStats{}

	// Copy CSS first
	if err := theme.CopyCSS(r.config.DataDir, r.config.CLIThemesDir, r.themeName); err != nil {
		return nil, fmt.Errorf("failed to copy CSS: %w", err)
	}

	// Find all posts
	postsDir := filepath.Join(r.config.DataDir, "posts")
	if err := filepath.Walk(postsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}

		relPath, _ := filepath.Rel(r.config.DataDir, path)
		_, rendered, err := r.RenderFile(relPath, "post", force)
		if err != nil {
			return fmt.Errorf("failed to render %s: %w", relPath, err)
		}

		if rendered {
			stats.PostsRendered++
		} else {
			stats.PostsSkipped++
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Find all comments
	commentsDir := filepath.Join(r.config.DataDir, "comments")
	if err := filepath.Walk(commentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}

		relPath, _ := filepath.Rel(r.config.DataDir, path)
		_, rendered, err := r.RenderFile(relPath, "comment", force)
		if err != nil {
			return fmt.Errorf("failed to render %s: %w", relPath, err)
		}

		if rendered {
			stats.CommentsRendered++
		} else {
			stats.CommentsSkipped++
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Generate index
	if err := r.RenderIndex(); err != nil {
		return nil, fmt.Errorf("failed to render index: %w", err)
	}
	stats.IndexGenerated = true

	return stats, nil
}

// loadPublicIndex loads posts and comments from public.jsonl.
func (r *PageRenderer) loadPublicIndex() ([]template.PostData, []template.CommentData, error) {
	entries, err := metadata.LoadPublicIndex(r.config.DataDir)
	if err != nil {
		return nil, nil, err
	}

	var posts []template.PostData
	var comments []template.CommentData

	for _, entry := range entries {
		if strings.HasPrefix(entry.Path, "posts/") || entry.Type == "post" {
			// Convert .md to .html for URL
			htmlPath := strings.TrimSuffix(entry.Path, ".md") + ".html"
			posts = append(posts, template.PostData{
				URL:            htmlPath,
				Title:          entry.Title,
				Published:      entry.Published,
				PublishedHuman: template.FormatHumanDate(entry.Published),
				CommentCount:   0, // Comment count not stored in public.jsonl
			})
		} else if strings.HasPrefix(entry.Path, "comments/") || entry.Type == "comment" {
			htmlPath := strings.TrimSuffix(entry.Path, ".md") + ".html"
			inReplyToURL := ""
			if entry.InReplyTo != nil {
				inReplyToURL = entry.InReplyTo.URL
			}
			comments = append(comments, template.CommentData{
				URL:            htmlPath,
				TargetAuthor:   extractDomain(inReplyToURL),
				Published:      entry.Published,
				PublishedHuman: template.FormatHumanDate(entry.Published),
				Preview:        truncateText(entry.Title, 100), // Use title as preview
			})
		}
	}

	// Reverse order (newest first)
	for i, j := 0, len(posts)-1; i < j; i, j = i+1, j-1 {
		posts[i], posts[j] = posts[j], posts[i]
	}
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}

	return posts, comments, nil
}

// loadBlessedCommentsForPost loads blessed comments for a specific post.
func (r *PageRenderer) loadBlessedCommentsForPost(postPath string) ([]template.BlessedCommentData, error) {
	// Load blessed comments for this specific post
	comments, err := metadata.GetBlessedCommentsForPost(r.config.DataDir, postPath)
	if err != nil {
		return nil, err
	}

	var results []template.BlessedCommentData

	for _, comment := range comments {
		// TODO: Actually load comment content from URL
		// For now, create placeholder with available info
		results = append(results, template.BlessedCommentData{
			URL:            comment.URL,
			AuthorName:     extractDomain(comment.URL),
			Published:      comment.BlessedAt,
			PublishedHuman: template.FormatHumanDate(comment.BlessedAt),
			Content:        "", // Would need to fetch from URL
		})
	}

	return results, nil
}

// getSiteTitle returns the site title from .well-known/polis.
func (r *PageRenderer) getSiteTitle() string {
	wkPath := filepath.Join(r.config.DataDir, ".well-known", "polis")
	data, err := os.ReadFile(wkPath)
	if err != nil {
		return r.config.BaseURL
	}

	var wk struct {
		SiteTitle string `json:"site_title"`
		BaseURL   string `json:"base_url"`
	}
	if err := json.Unmarshal(data, &wk); err != nil {
		return r.config.BaseURL
	}

	if wk.SiteTitle != "" {
		return wk.SiteTitle
	}
	return wk.BaseURL
}

// getAuthorName returns the author name from .well-known/polis.
func (r *PageRenderer) getAuthorName() string {
	wkPath := filepath.Join(r.config.DataDir, ".well-known", "polis")
	data, err := os.ReadFile(wkPath)
	if err != nil {
		return ""
	}

	var wk struct {
		AuthorName string `json:"author_name"`
	}
	if err := json.Unmarshal(data, &wk); err != nil {
		return ""
	}

	return wk.AuthorName
}

// buildURL builds a full URL for a file path.
func (r *PageRenderer) buildURL(path string) string {
	if r.config.BaseURL == "" {
		return path
	}
	return strings.TrimSuffix(r.config.BaseURL, "/") + "/" + path
}

// parseFrontmatter extracts frontmatter fields from content.
func parseFrontmatter(content string) map[string]string {
	result := make(map[string]string)
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return result
	}

	// Find end of frontmatter
	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		return result
	}

	fm := content[4 : end+3]
	lines := strings.Split(fm, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			continue // Skip nested items
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

// stripFrontmatter removes frontmatter from content.
func stripFrontmatter(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return content
	}

	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		return content
	}

	return strings.TrimSpace(content[end+7:])
}

// parseNestedField extracts a nested field value from frontmatter.
// For example: parseNestedField(content, "in-reply-to", "url")
func parseNestedField(content, section, field string) string {
	lines := strings.Split(content, "\n")
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section start
		if strings.HasPrefix(line, section+":") && !strings.HasPrefix(line, "  ") {
			inSection = true
			continue
		}

		// Check for section end (new non-indented field)
		if inSection && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
			break
		}

		// Look for field in section
		if inSection && strings.HasPrefix(trimmed, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, field+":"))
		}
	}

	return ""
}

// extractDomain extracts the domain from a URL.
func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	if idx := strings.Index(url, "/"); idx > 0 {
		return url[:idx]
	}
	return url
}

// truncateText truncates text to a maximum length, adding "..." if truncated.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
