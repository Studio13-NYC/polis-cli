package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/blessing"
	"github.com/vdibart/polis-cli/cli-go/pkg/comment"
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/feed"
	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/hooks"
	"github.com/vdibart/polis-cli/cli-go/pkg/metadata"
	"github.com/vdibart/polis-cli/cli-go/pkg/publish"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
	"github.com/vdibart/polis-cli/cli-go/pkg/render"
	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
	"github.com/vdibart/polis-cli/cli-go/pkg/snippet"
	polisurl "github.com/vdibart/polis-cli/cli-go/pkg/url"
)

// draftIDSanitizer strips all characters except alphanumeric, hyphens, and underscores.
var draftIDSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// validatePostPath ensures the path is safe and within the posts directory.
// This prevents path traversal attacks that could read/write arbitrary files.
func validatePostPath(path string) error {
	// Canonicalize first to normalize encoded traversals (e.g., ./, //)
	path = filepath.Clean(path)
	// Must start with "posts/"
	if !strings.HasPrefix(path, "posts/") {
		return fmt.Errorf("invalid path: must be under posts/")
	}
	// No path traversal sequences
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: traversal not allowed")
	}
	// No null bytes (could bypass checks in some systems)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid path: null bytes not allowed")
	}
	return nil
}

// validateContentPath ensures the path is safe and within allowed directories.
// This prevents path traversal attacks.
func validateContentPath(path string) error {
	// Canonicalize first to normalize encoded traversals (e.g., ./, //)
	path = filepath.Clean(path)
	// No path traversal sequences
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: traversal not allowed")
	}
	// No null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid path: null bytes not allowed")
	}

	// Allow root-level markdown and html files (e.g., index.md, index.html, about.md)
	if (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".html")) && !strings.Contains(path, "/") {
		return nil
	}

	// Must start with an allowed prefix (including html versions)
	allowedPrefixes := []string{"posts/", "comments/", ".polis/posts/drafts/", ".polis/drafts/"}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}
	return fmt.Errorf("invalid path: must be a root .md/.html file or under posts/, comments/, or .polis/posts/drafts/")
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Run validation to get current state
	validation := site.Validate(s.DataDir)

	// For backwards compatibility, "configured" is true if site is valid
	configured := validation.Status == site.StatusValid

	response := map[string]interface{}{
		"configured": configured,
		"site_title": s.GetSiteTitle(),
		"validation": map[string]interface{}{
			"status": validation.Status,
			"errors": validation.Errors,
		},
	}

	// Include site info if valid
	if validation.SiteInfo != nil {
		response["site_info"] = validation.SiteInfo
	}

	json.NewEncoder(w).Encode(response)
}

// handleValidate returns the validation status of the site directory.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	validation := site.Validate(s.DataDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validation)
}

// handleInit initializes a new polis site in the data directory.
func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SiteTitle string `json:"site_title"`
		Author    string `json:"author"`
		Email     string `json:"email"`
		BaseURL   string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is OK - all fields are optional
		req = struct {
			SiteTitle string `json:"site_title"`
			Author    string `json:"author"`
			Email     string `json:"email"`
			BaseURL   string `json:"base_url"`
		}{}
	}

	opts := site.InitOptions{
		SiteTitle: req.SiteTitle,
	}

	s.LogDebug("Initializing new site at: %s", s.DataDir)
	result, err := site.Init(s.DataDir, opts)
	if err != nil {
		s.LogError("Failed to initialize site: %v", err)
		http.Error(w, "Failed to initialize site", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Initialized new site at: %s (title: %s)", result.SiteDir, req.SiteTitle)

	// Reload keys after successful init
	s.LoadKeys()

	// Create config file (for webapp settings like hooks, discovery)
	s.Config = &Config{}
	s.ApplyDiscoveryDefaults()
	if err := s.SaveConfig(); err != nil {
		log.Printf("[warning] Failed to save config: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    result.Success,
		"site_dir":   result.SiteDir,
		"public_key": result.PublicKey,
		"site_title": s.GetSiteTitle(),
	})
}

// handleLink creates a symlink from data/ to an existing polis site.
func (s *Server) handleLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Expand ~ to home directory
	targetPath := req.Path
	if strings.HasPrefix(targetPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			targetPath = filepath.Join(home, targetPath[2:])
		}
	}

	// Convert to absolute path
	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Validate the target is a valid polis site
	validation := site.Validate(targetPath)
	if validation.Status != site.StatusValid {
		errMsgs := []string{}
		for _, e := range validation.Errors {
			errMsgs = append(errMsgs, e.Message)
		}
		http.Error(w, "Target is not a valid polis site: "+strings.Join(errMsgs, "; "), http.StatusBadRequest)
		return
	}

	// Get the current data directory path (before it's a symlink)
	execPath, err := os.Executable()
	if err != nil {
		s.LogError("failed to get executable path: %v", err)
		http.Error(w, "Failed to get executable path", http.StatusInternalServerError)
		return
	}
	linkPath := filepath.Join(filepath.Dir(execPath), "data")

	// Safety check: refuse if data/ already has content
	entries, err := os.ReadDir(linkPath)
	if err == nil && len(entries) > 0 {
		// Check if it's already a symlink pointing somewhere
		info, err := os.Lstat(linkPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			// It's already a symlink - we can replace it
		} else {
			// It's a real directory with files
			http.Error(w, "Data directory already contains files. Remove them first or use init instead.", http.StatusConflict)
			return
		}
	}

	// Remove existing data directory/symlink
	if err := os.RemoveAll(linkPath); err != nil {
		s.LogError("failed to remove existing data directory: %v", err)
		http.Error(w, "Failed to remove existing data directory", http.StatusInternalServerError)
		return
	}

	// Create symlink
	s.LogDebug("Linking to existing site: %s", targetPath)
	if err := os.Symlink(targetPath, linkPath); err != nil {
		s.LogError("Failed to create symlink: %v", err)
		http.Error(w, "Failed to create symlink", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Linked to existing site: %s", targetPath)

	// Update server's data directory to the resolved path
	s.DataDir = targetPath

	// Reload keys and config from the linked site
	s.LoadKeys()
	s.LoadConfig()
	s.LoadEnv()
	s.ApplyDiscoveryDefaults()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"linked_to":  targetPath,
		"site_title": s.GetSiteTitle(),
		"site_info":  validation.SiteInfo,
	})
}

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Not configured - please complete setup first", http.StatusBadRequest)
		return
	}

	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Render markdown to HTML
	html, err := render.MarkdownToHTML(req.Markdown)
	if err != nil {
		s.LogError("render markdown: %v", err)
		http.Error(w, "Failed to render markdown", http.StatusInternalServerError)
		return
	}

	// Sign the content
	signature, err := signing.SignContent([]byte(req.Markdown), s.PrivateKey)
	if err != nil {
		s.LogError("sign content: %v", err)
		http.Error(w, "Failed to sign content", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"html":      html,
		"signature": signature,
	})
}

func (s *Server) handleDrafts(w http.ResponseWriter, r *http.Request) {
	draftsDir := filepath.Join(s.DataDir, ".polis", "posts", "drafts")

	switch r.Method {
	case http.MethodGet:
		// List drafts
		entries, err := os.ReadDir(draftsDir)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"drafts": []interface{}{},
			})
			return
		}

		var drafts []map[string]interface{}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			drafts = append(drafts, map[string]interface{}{
				"id":       strings.TrimSuffix(entry.Name(), ".md"),
				"name":     entry.Name(),
				"modified": info.ModTime().Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"drafts": drafts,
		})

	case http.MethodPost:
		// Save draft
		var req struct {
			ID       string `json:"id"`
			Markdown string `json:"markdown"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.ID == "" {
			req.ID = fmt.Sprintf("draft-%d", time.Now().Unix())
		}

		// Sanitize ID - whitelist only safe characters
		req.ID = draftIDSanitizer.ReplaceAllString(req.ID, "-")

		draftPath := filepath.Join(draftsDir, req.ID+".md")
		if err := os.WriteFile(draftPath, []byte(req.Markdown), 0644); err != nil {
			s.LogError("failed to save draft: %v", err)
			http.Error(w, "Failed to save draft", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      req.ID,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/drafts/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/drafts/")
	if id == "" {
		http.Error(w, "Draft ID required", http.StatusBadRequest)
		return
	}

	// Sanitize ID - whitelist only safe characters
	id = draftIDSanitizer.ReplaceAllString(id, "-")

	draftPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts", id+".md")

	switch r.Method {
	case http.MethodGet:
		content, err := os.ReadFile(draftPath)
		if err != nil {
			http.Error(w, "Draft not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       id,
			"markdown": string(content),
		})

	case http.MethodDelete:
		if err := os.Remove(draftPath); err != nil {
			s.LogError("failed to delete draft: %v", err)
			http.Error(w, "Failed to delete draft", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Not configured - please complete setup first", http.StatusBadRequest)
		return
	}

	var req struct {
		Markdown string `json:"markdown"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Markdown) == "" {
		http.Error(w, "Markdown content required", http.StatusBadRequest)
		return
	}

	// Strip existing frontmatter if present
	markdown := req.Markdown
	if publish.HasFrontmatter(markdown) {
		markdown = publish.StripFrontmatter(markdown)
	}

	s.LogDebug("Publishing post with filename: %s", req.Filename)
	result, err := publish.PublishPost(s.DataDir, markdown, req.Filename, s.PrivateKey)
	if err != nil {
		s.LogError("Failed to publish: %v", err)
		http.Error(w, "Failed to publish", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Published post: %s (title: %s)", result.Path, result.Title)

	// Render site to generate HTML files
	if err := s.RenderSite(); err != nil {
		// Log but don't fail - the post was published successfully
		log.Printf("[warning] post-publish render failed: %v", err)
	}

	// Run post-publish hook (checks explicit config, then auto-discovers .polis/hooks/)
	{
		var hc *hooks.HookConfig
		if s.Config != nil {
			hc = s.Config.Hooks
		}
		payload := &hooks.HookPayload{
			Event:         hooks.EventPostPublish,
			Path:          result.Path,
			Title:         result.Title,
			Version:       result.Version,
			Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			CommitMessage: hooks.GenerateCommitMessage(hooks.EventPostPublish, result.Title),
		}
		hookResult, err := hooks.RunHook(s.DataDir, hc, payload)
		if err != nil {
			// Log hook error but don't fail the publish
			log.Printf("[warning] post-publish hook failed: %v", err)
			s.LogWarn("Post-publish hook failed: %v", err)
		}
		if hookResult != nil && hookResult.Executed {
			log.Printf("[info] post-publish hook executed: %s", hookResult.Output)
			s.LogInfo("Post-publish hook executed: %s", hookResult.Output)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handlePosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read posts from public.jsonl
	indexPath := filepath.Join(s.DataDir, "metadata", "public.jsonl")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		// No posts yet
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"posts": []interface{}{},
		})
		return
	}

	var posts []map[string]interface{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		// Filter out comments - only include posts
		if path, ok := entry["path"].(string); ok {
			if strings.HasPrefix(path, "comments/") {
				continue
			}
		}
		posts = append(posts, entry)
	}

	// Reverse order (newest first)
	for i, j := 0, len(posts)-1; i < j; i, j = i+1, j-1 {
		posts[i], posts[j] = posts[j], posts[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": posts,
	})
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL: /api/posts/posts/20260125/my-post.md
	postPath := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	if postPath == "" {
		http.Error(w, "Post path required", http.StatusBadRequest)
		return
	}

	// Validate path to prevent directory traversal
	if err := validatePostPath(postPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Read the post file
	fullPath := filepath.Join(s.DataDir, postPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	rawMarkdown := string(content)

	// Strip frontmatter to get just the body markdown
	markdown := publish.StripFrontmatter(rawMarkdown)

	// Parse frontmatter for metadata
	frontmatter := publish.ParseFrontmatter(rawMarkdown)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":         postPath,
		"markdown":     markdown,
		"raw_markdown": rawMarkdown,
		"title":        frontmatter["title"],
		"published":    frontmatter["published"],
		"updated":      frontmatter["updated"],
	})
}

func (s *Server) handleRepublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Not configured - please complete setup first", http.StatusBadRequest)
		return
	}

	var req struct {
		Path     string `json:"path"`
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Post path required", http.StatusBadRequest)
		return
	}

	// Validate path to prevent directory traversal
	if err := validatePostPath(req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Markdown) == "" {
		http.Error(w, "Markdown content required", http.StatusBadRequest)
		return
	}

	// Strip existing frontmatter if present
	markdown := req.Markdown
	if publish.HasFrontmatter(markdown) {
		markdown = publish.StripFrontmatter(markdown)
	}

	s.LogDebug("Republishing post: %s", req.Path)
	result, err := publish.RepublishPost(s.DataDir, req.Path, markdown, s.PrivateKey)
	if err != nil {
		s.LogError("Failed to republish %s: %v", req.Path, err)
		http.Error(w, "Failed to republish", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Republished post: %s (title: %s)", result.Path, result.Title)

	// Render site to generate HTML files
	if err := s.RenderSite(); err != nil {
		// Log but don't fail - the post was republished successfully
		log.Printf("[warning] post-republish render failed: %v", err)
	}

	// Run post-republish hook (checks explicit config, then auto-discovers .polis/hooks/)
	{
		var hc *hooks.HookConfig
		if s.Config != nil {
			hc = s.Config.Hooks
		}
		payload := &hooks.HookPayload{
			Event:         hooks.EventPostRepublish,
			Path:          result.Path,
			Title:         result.Title,
			Version:       result.Version,
			Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			CommitMessage: hooks.GenerateCommitMessage(hooks.EventPostRepublish, result.Title),
		}
		hookResult, err := hooks.RunHook(s.DataDir, hc, payload)
		if err != nil {
			// Log hook error but don't fail the republish
			log.Printf("[warning] post-republish hook failed: %v", err)
		}
		if hookResult != nil && hookResult.Executed {
			log.Printf("[info] post-republish hook executed: %s", hookResult.Output)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Comment API handlers

func (s *Server) handleCommentDrafts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List comment drafts
		drafts, err := comment.ListDrafts(s.DataDir)
		if err != nil {
			s.LogError("failed to list drafts: %v", err)
			http.Error(w, "Failed to list drafts", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"drafts": drafts,
		})

	case http.MethodPost:
		// Save comment draft
		var req struct {
			ID        string `json:"id"`
			InReplyTo string `json:"in_reply_to"`
			RootPost  string `json:"root_post"`
			Content   string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.InReplyTo == "" {
			http.Error(w, "in_reply_to is required", http.StatusBadRequest)
			return
		}

		draft := &comment.CommentDraft{
			ID:        req.ID,
			InReplyTo: polisurl.NormalizeToMD(req.InReplyTo),
			RootPost:  polisurl.NormalizeToMD(req.RootPost),
			Content:   req.Content,
		}

		if err := comment.SaveDraft(s.DataDir, draft); err != nil {
			s.LogError("failed to save draft: %v", err)
			http.Error(w, "Failed to save draft", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      draft.ID,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCommentDraft(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/comments/drafts/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/comments/drafts/")
	if id == "" {
		http.Error(w, "Draft ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		draft, err := comment.LoadDraft(s.DataDir, id)
		if err != nil {
			http.Error(w, "Draft not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(draft)

	case http.MethodDelete:
		if err := comment.DeleteDraft(s.DataDir, id); err != nil {
			s.LogError("failed to delete draft: %v", err)
			http.Error(w, "Failed to delete draft", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCommentSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Not configured - please complete setup first", http.StatusBadRequest)
		return
	}

	var req struct {
		DraftID   string `json:"draft_id"`
		InReplyTo string `json:"in_reply_to"`
		RootPost  string `json:"root_post"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Load draft if ID provided, otherwise use inline content
	var draft *comment.CommentDraft
	if req.DraftID != "" {
		var err error
		draft, err = comment.LoadDraft(s.DataDir, req.DraftID)
		if err != nil {
			http.Error(w, "Draft not found", http.StatusNotFound)
			return
		}
	} else {
		if req.InReplyTo == "" {
			http.Error(w, "in_reply_to is required", http.StatusBadRequest)
			return
		}
		draft = &comment.CommentDraft{
			InReplyTo: polisurl.NormalizeToMD(req.InReplyTo),
			RootPost:  polisurl.NormalizeToMD(req.RootPost),
			Content:   req.Content,
		}
	}

	// Get author email from config or .well-known/polis
	authorEmail := ""
	if s.Config != nil {
		authorEmail = s.Config.AuthorEmail
	}
	if authorEmail == "" {
		// Fall back to email from .well-known/polis (matches bash CLI behavior)
		if wk, err := site.LoadWellKnown(s.DataDir); err == nil && wk.Email != "" {
			authorEmail = wk.Email
		}
	}
	if authorEmail == "" {
		http.Error(w, "Author email not configured - set email in .well-known/polis or .env file", http.StatusBadRequest)
		return
	}

	// Get site URL from POLIS_BASE_URL env var (authoritative source, matches bash CLI)
	siteURL := s.GetBaseURL()
	if siteURL == "" {
		http.Error(w, "POLIS_BASE_URL not configured - set it in .env file", http.StatusBadRequest)
		return
	}

	signed, err := comment.SignComment(s.DataDir, draft, authorEmail, siteURL, s.PrivateKey)
	if err != nil {
		s.LogError("failed to sign comment: %v", err)
		http.Error(w, "Failed to sign comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"comment":   signed.Meta,
		"signature": signed.Signature,
	})
}

func (s *Server) handleCommentBeseech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config == nil || s.Config.DiscoveryURL == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	var req struct {
		CommentID string `json:"comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.CommentID == "" {
		http.Error(w, "comment_id is required", http.StatusBadRequest)
		return
	}

	// Get the pending comment
	signed, err := comment.GetComment(s.DataDir, req.CommentID, comment.StatusPending)
	if err != nil {
		http.Error(w, "Comment not found in pending", http.StatusNotFound)
		return
	}

	// Compute comment_url dynamically (like bash CLI does)
	// Get base_url from POLIS_BASE_URL env var (matches bash CLI behavior)
	baseURL := s.GetBaseURL()
	if baseURL == "" {
		http.Error(w, "POLIS_BASE_URL not configured - set it in .env file", http.StatusBadRequest)
		return
	}

	// Parse timestamp to get date directory (YYYYMMDD format)
	ts, err := time.Parse("2006-01-02T15:04:05Z", signed.Meta.Timestamp)
	if err != nil {
		s.LogError("invalid timestamp in comment: %v", err)
		http.Error(w, "Invalid timestamp in comment", http.StatusInternalServerError)
		return
	}
	dateDir := ts.Format("20060102")

	// Construct comment URL: base_url/comments/YYYYMMDD/comment_id.md
	commentURL := fmt.Sprintf("%s/comments/%s/%s.md", baseURL, dateDir, req.CommentID)

	// Build canonical JSON payload for signing (must match bash CLI field order exactly)
	// Order: comment_url, comment_version, in_reply_to, [in_reply_to_version], root_post, author, timestamp
	var canonicalPayload string
	if signed.Meta.InReplyToVersion != "" {
		canonicalPayload = fmt.Sprintf(`{"comment_url":"%s","comment_version":"%s","in_reply_to":"%s","in_reply_to_version":"%s","root_post":"%s","author":"%s","timestamp":"%s"}`,
			commentURL, signed.Meta.CommentVersion, signed.Meta.InReplyTo,
			signed.Meta.InReplyToVersion, signed.Meta.RootPost, signed.Meta.Author, signed.Meta.Timestamp)
	} else {
		canonicalPayload = fmt.Sprintf(`{"comment_url":"%s","comment_version":"%s","in_reply_to":"%s","root_post":"%s","author":"%s","timestamp":"%s"}`,
			commentURL, signed.Meta.CommentVersion, signed.Meta.InReplyTo,
			signed.Meta.RootPost, signed.Meta.Author, signed.Meta.Timestamp)
	}

	// Sign the canonical payload (NOT the file signature)
	beseechSig, err := signing.SignContent([]byte(canonicalPayload), s.PrivateKey)
	if err != nil {
		s.LogError("failed to sign beseech payload: %v", err)
		http.Error(w, "Failed to sign beseech payload", http.StatusInternalServerError)
		return
	}

	// Create discovery client
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)

	// Send blessing request with the beseech payload signature
	beseechReq := &discovery.BeseechRequest{
		CommentURL:     commentURL,
		CommentVersion: signed.Meta.CommentVersion,
		InReplyTo:      signed.Meta.InReplyTo,
		RootPost:       signed.Meta.RootPost,
		Author:         signed.Meta.Author,
		Timestamp:      signed.Meta.Timestamp,
		Signature:      beseechSig,
	}

	resp, err := client.BeseechBlessing(beseechReq)
	if err != nil {
		s.LogError("failed to send blessing request: %v", err)
		http.Error(w, "Failed to send blessing request", http.StatusInternalServerError)
		return
	}

	// If auto-blessed, move to blessed directory and re-render
	if resp.Status == "blessed" {
		if err := comment.MoveComment(s.DataDir, req.CommentID, comment.StatusPending, comment.StatusBlessed); err != nil {
			log.Printf("[warning] Failed to move auto-blessed comment: %v", err)
		}

		// Re-render site so HTML includes the new blessed comment
		if err := s.RenderSite(); err != nil {
			log.Printf("[warning] post-beseech render failed: %v", err)
		}

		// Run post-comment hook (checks explicit config, then auto-discovers .polis/hooks/)
		{
			var hc *hooks.HookConfig
			if s.Config != nil {
				hc = s.Config.Hooks
			}
			payload := &hooks.HookPayload{
				Event:         hooks.EventPostComment,
				Path:          fmt.Sprintf("comments/blessed/%s.md", req.CommentID),
				Title:         signed.Meta.InReplyTo,
				Version:       signed.Meta.CommentVersion,
				Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				CommitMessage: hooks.GenerateCommitMessage(hooks.EventPostComment, signed.Meta.InReplyTo),
			}
			hooks.RunHook(s.DataDir, hc, payload)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.Success,
		"status":  resp.Status,
		"message": resp.Message,
	})
}

func (s *Server) handleCommentsPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	comments, err := comment.ListComments(s.DataDir, comment.StatusPending)
	if err != nil {
		s.LogError("failed to list pending comments: %v", err)
		http.Error(w, "Failed to list pending comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"comments": comments,
	})
}

func (s *Server) handleCommentsBlessed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	comments, err := comment.ListComments(s.DataDir, comment.StatusBlessed)
	if err != nil {
		s.LogError("failed to list blessed comments: %v", err)
		http.Error(w, "Failed to list blessed comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"comments": comments,
	})
}

func (s *Server) handleCommentsDenied(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	comments, err := comment.ListComments(s.DataDir, comment.StatusDenied)
	if err != nil {
		s.LogError("failed to list denied comments: %v", err)
		http.Error(w, "Failed to list denied comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"comments": comments,
	})
}

// handleCommentByStatus handles GET /api/comments/{status}/{id}
func (s *Server) handleCommentByStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract status and ID from URL: /api/comments/{status}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/comments/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		http.Error(w, "Comment ID required", http.StatusBadRequest)
		return
	}

	status := parts[0]
	commentID := parts[1]

	// Validate status
	if status != comment.StatusPending && status != comment.StatusBlessed && status != comment.StatusDenied {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	result, err := comment.GetComment(s.DataDir, commentID, status)
	if err != nil {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"comment": map[string]interface{}{
			"id":          result.Meta.ID,
			"title":       result.Meta.Title,
			"in_reply_to": result.Meta.InReplyTo,
			"root_post":   result.Meta.RootPost,
			"comment_url": result.Meta.CommentURL,
			"timestamp":   result.Meta.Timestamp,
			"status":      result.Meta.Status,
			"content":     result.Content,
		},
	})
}

func (s *Server) handleCommentsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config == nil || s.Config.DiscoveryURL == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	// Create discovery client
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)

	// Sync pending comments
	result, err := comment.SyncPendingComments(s.DataDir, client, s.Config.Hooks)
	if err != nil {
		s.LogError("failed to sync comments: %v", err)
		http.Error(w, "Failed to sync comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Blessing API handlers (ON MY POSTS - incoming blessing requests)

func (s *Server) handleBlessingRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config == nil || s.Config.DiscoveryURL == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	// Create discovery client
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)

	// Get domain from base URL
	domain := s.GetSubdomain()

	// Fetch pending blessing requests
	requests, err := blessing.FetchPendingRequests(client, domain)
	if err != nil {
		s.LogError("failed to fetch requests: %v", err)
		http.Error(w, "Failed to fetch requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": requests,
	})
}

func (s *Server) handleBlessingGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config == nil || s.Config.DiscoveryURL == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Private key not configured", http.StatusBadRequest)
		return
	}

	var req struct {
		CommentVersion string `json:"comment_version"`
		CommentURL     string `json:"comment_url"`
		InReplyTo      string `json:"in_reply_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.CommentVersion == "" {
		http.Error(w, "comment_version is required", http.StatusBadRequest)
		return
	}

	// Create discovery client
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)

	// Grant the blessing (with signed request)
	// Normalize URLs to .md format for consistent storage
	s.LogDebug("Granting blessing for comment version: %s", req.CommentVersion)
	result, err := blessing.GrantByVersion(
		s.DataDir,
		req.CommentVersion,
		polisurl.NormalizeToMD(req.CommentURL),
		polisurl.NormalizeToMD(req.InReplyTo),
		client,
		s.Config.Hooks,
		s.PrivateKey,
	)
	if err != nil {
		s.LogError("Failed to grant blessing: %v", err)
		http.Error(w, "Failed to grant blessing", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Granted blessing for comment: %s", req.CommentURL)

	// Render site to include the newly blessed comment
	if err := s.RenderSite(); err != nil {
		// Log but don't fail - the blessing was granted successfully
		log.Printf("[warning] post-blessing render failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleBlessingDeny(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config == nil || s.Config.DiscoveryURL == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	if s.PrivateKey == nil {
		http.Error(w, "Private key not configured", http.StatusBadRequest)
		return
	}

	var req struct {
		CommentVersion string `json:"comment_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.CommentVersion == "" {
		http.Error(w, "comment_version is required", http.StatusBadRequest)
		return
	}

	// Create discovery client
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)

	// Deny the blessing (with signed request)
	s.LogDebug("Denying blessing for comment version: %s", req.CommentVersion)
	result, err := blessing.Deny(req.CommentVersion, client, s.PrivateKey)
	if err != nil {
		s.LogError("Failed to deny blessing: %v", err)
		http.Error(w, "Failed to deny blessing", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Denied blessing for comment version: %s", req.CommentVersion)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleBlessedComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load blessed comments from local metadata
	bc, err := metadata.LoadBlessedComments(s.DataDir)
	if err != nil {
		// Return empty list if file doesn't exist
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":  "",
			"comments": []interface{}{},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bc)
}

func (s *Server) handleBlessingRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CommentURL string `json:"comment_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.CommentURL == "" {
		http.Error(w, "comment_url is required", http.StatusBadRequest)
		return
	}

	// Normalize URL to .md format for consistent lookup
	normalizedURL := polisurl.NormalizeToMD(req.CommentURL)

	// Remove from blessed-comments.json
	if err := metadata.RemoveBlessedComment(s.DataDir, normalizedURL); err != nil {
		s.LogError("failed to revoke blessing: %v", err)
		http.Error(w, "Failed to revoke blessing", http.StatusInternalServerError)
		return
	}
	s.LogInfo("Revoked blessing for comment: %s", normalizedURL)

	// Render site to remove the comment from pages
	if err := s.RenderSite(); err != nil {
		// Log but don't fail - the revoke was successful
		log.Printf("[warning] post-revoke render failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"comment_url": normalizedURL,
	})
}

// Settings and Automation API handlers

// Automation represents a configured automation (hook)
type Automation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Event       string `json:"event"`
	ScriptPath  string `json:"script_path"`
	Enabled     bool   `json:"enabled"`
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build site info from .well-known/polis and config
	subdomain := ""
	publicKey := ""
	discoveryURL := ""
	discoveryConfigured := false
	siteTitle := s.GetSiteTitle() // From .well-known/polis with fallback to base_url
	viewMode := "list"            // Default to list mode
	showFrontmatter := true       // Default to showing frontmatter
	baseURL := ""

	if s.Config != nil {
		subdomain = s.GetSubdomain()
		discoveryURL = s.Config.DiscoveryURL
		discoveryConfigured = s.Config.DiscoveryURL != "" && s.Config.DiscoveryKey != ""
		if s.Config.ViewMode != "" {
			viewMode = s.Config.ViewMode
		}
		if s.Config.ShowFrontmatter != nil {
			showFrontmatter = *s.Config.ShowFrontmatter
		}
	}
	if s.PublicKey != nil {
		publicKey = strings.TrimSpace(string(s.PublicKey))
	}

	// Get base URL from POLIS_BASE_URL env var (matches bash CLI behavior)
	baseURL = s.GetBaseURL()

	// Build automations list from hooks config
	automations := s.getAutomations()

	// Check which hook files exist
	existingHooks := s.getExistingHooks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"site": map[string]interface{}{
			"subdomain":            subdomain,
			"site_title":           siteTitle,
			"public_key":           publicKey,
			"data_dir":             s.DataDir,
			"discovery_url":        discoveryURL,
			"discovery_configured": discoveryConfigured,
			"view_mode":            viewMode,
			"show_frontmatter":     showFrontmatter,
			"base_url":             baseURL,
		},
		"automations":    automations,
		"existing_hooks": existingHooks,
	})
}

func (s *Server) getAutomations() []Automation {
	var automations []Automation
	var hc *hooks.HookConfig
	if s.Config != nil {
		hc = s.Config.Hooks
	}

	type hookInfo struct {
		event       hooks.HookEvent
		id          string
		name        string
		description string
	}
	allHooks := []hookInfo{
		{hooks.EventPostPublish, "post-publish", "Post-publish hook", "Runs after each publish"},
		{hooks.EventPostRepublish, "post-republish", "Post-republish hook", "Runs after each republish"},
		{hooks.EventPostComment, "post-comment", "Post-comment hook", "Runs after you bless a comment on your site"},
	}

	for _, h := range allHooks {
		path := hooks.GetHookPathWithDiscovery(hc, h.event, s.DataDir)
		if path != "" {
			automations = append(automations, Automation{
				ID:          h.id,
				Name:        h.name,
				Description: h.description,
				Event:       string(h.event),
				ScriptPath:  path,
				Enabled:     true,
			})
		}
	}

	return automations
}

func (s *Server) handleAutomations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		automations := s.getAutomations()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"automations": automations,
		})

	case http.MethodPost:
		// Create a new automation
		var req struct {
			TemplateID string `json:"template_id"`
			HookType   string `json:"hook_type"`
			Script     string `json:"script"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Default to post-publish if not specified
		hookType := req.HookType
		if hookType == "" {
			hookType = "post-publish"
		}

		// Validate hook type
		validTypes := map[string]bool{
			"post-publish":   true,
			"post-republish": true,
			"post-comment":   true,
		}
		if !validTypes[hookType] {
			http.Error(w, "Invalid hook type", http.StatusBadRequest)
			return
		}

		// Get script from template or use provided script
		script := req.Script
		if req.TemplateID != "" {
			template, ok := hooks.GetTemplate(req.TemplateID)
			if !ok {
				http.Error(w, "Unknown template ID", http.StatusBadRequest)
				return
			}
			script = template.Script
		}

		if script == "" {
			http.Error(w, "Script is required", http.StatusBadRequest)
			return
		}

		// Create the hook script
		scriptPath, err := s.createHookScript(script, hookType)
		if err != nil {
			s.LogError("failed to create hook: %v", err)
			http.Error(w, "Failed to create hook", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"script_path": scriptPath,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAutomationsQuick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use the vercel template (default for quick create)
	template, _ := hooks.GetTemplate("vercel")

	scriptPath, err := s.createHookScript(template.Script, "post-publish")
	if err != nil {
		s.LogError("failed to create hook: %v", err)
		http.Error(w, "Failed to create hook", http.StatusInternalServerError)
		return
	}
	_ = scriptPath // suppress unused variable warning

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"script_path": ".polis/hooks/post-publish.sh",
		"message":     "Created post-publish hook at .polis/hooks/post-publish.sh",
	})
}

func (s *Server) createHookScript(script string, hookType string) (string, error) {
	// Create hooks directory
	hooksDir := filepath.Join(s.DataDir, ".polis", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write the script file
	relativePath := ".polis/hooks/" + hookType + ".sh"
	scriptPath := filepath.Join(hooksDir, hookType+".sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return "", fmt.Errorf("failed to write hook script: %w", err)
	}

	// Update config to use this hook
	if s.Config == nil {
		s.Config = &Config{}
	}
	if s.Config.Hooks == nil {
		s.Config.Hooks = &hooks.HookConfig{}
	}

	switch hookType {
	case "post-publish":
		s.Config.Hooks.PostPublish = relativePath
	case "post-republish":
		s.Config.Hooks.PostRepublish = relativePath
	case "post-comment":
		s.Config.Hooks.PostComment = relativePath
	}

	if err := s.SaveConfig(); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	return relativePath, nil
}

func (s *Server) handleAutomation(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/automations/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/automations/")
	if id == "" {
		http.Error(w, "Automation ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		// Remove the automation
		if s.Config == nil || s.Config.Hooks == nil {
			http.Error(w, "No automations configured", http.StatusNotFound)
			return
		}

		var scriptPath string
		switch id {
		case "post-publish":
			scriptPath = s.Config.Hooks.PostPublish
			s.Config.Hooks.PostPublish = ""
		case "post-republish":
			scriptPath = s.Config.Hooks.PostRepublish
			s.Config.Hooks.PostRepublish = ""
		case "post-comment":
			scriptPath = s.Config.Hooks.PostComment
			s.Config.Hooks.PostComment = ""
		default:
			http.Error(w, "Unknown automation ID", http.StatusNotFound)
			return
		}

		// Save the updated config
		if err := s.SaveConfig(); err != nil {
			s.LogError("failed to save config: %v", err)
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Optionally delete the script file (only if it's in our hooks directory)
		if scriptPath != "" && strings.HasPrefix(scriptPath, ".polis/hooks/") {
			fullPath := filepath.Join(s.DataDir, scriptPath)
			os.Remove(fullPath) // Ignore error - file might not exist
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templates := hooks.ListTemplates()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

// getExistingHooks returns a list of hook types that have existing hook files
func (s *Server) getExistingHooks() []string {
	var existing []string
	hooksDir := filepath.Join(s.DataDir, ".polis", "hooks")

	hookTypes := []string{"post-publish", "post-republish", "post-comment"}
	for _, hookType := range hookTypes {
		scriptPath := filepath.Join(hooksDir, hookType+".sh")
		if _, err := os.Stat(scriptPath); err == nil {
			existing = append(existing, hookType)
		}
	}
	return existing
}

// handleHooksGenerate handles POST /api/hooks/generate to create an empty hook script
func (s *Server) handleHooksGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		HookType string `json:"hook_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate hook type
	validTypes := map[string]bool{
		"post-publish":   true,
		"post-republish": true,
		"post-comment":   true,
	}
	if !validTypes[req.HookType] {
		http.Error(w, "Invalid hook type. Must be one of: post-publish, post-republish, post-comment", http.StatusBadRequest)
		return
	}

	// Create hooks directory
	hooksDir := filepath.Join(s.DataDir, ".polis", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		s.LogError("failed to create hooks directory: %v", err)
		http.Error(w, "Failed to create hooks directory", http.StatusInternalServerError)
		return
	}

	// Check if file already exists
	scriptPath := filepath.Join(hooksDir, req.HookType+".sh")
	if _, err := os.Stat(scriptPath); err == nil {
		http.Error(w, "Hook file already exists: .polis/hooks/"+req.HookType+".sh", http.StatusConflict)
		return
	}

	// Create empty hook script with boilerplate
	script := fmt.Sprintf(`#!/bin/bash
set -e
# %s hook
# Available environment variables:
# POLIS_SITE_DIR - path to your site directory
# POLIS_PATH - relative path to the published file
# POLIS_TITLE - title of the post
# POLIS_COMMIT_MESSAGE - suggested commit message
# POLIS_EVENT - event type (%s)
# POLIS_VERSION - content hash
# POLIS_TIMESTAMP - ISO timestamp

# Add your custom logic below:
echo "Hook triggered: %s"
`, req.HookType, req.HookType, req.HookType)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		s.LogError("failed to write hook script: %v", err)
		http.Error(w, "Failed to write hook script", http.StatusInternalServerError)
		return
	}

	// Update config to use this hook
	if s.Config == nil {
		s.Config = &Config{}
	}
	if s.Config.Hooks == nil {
		s.Config.Hooks = &hooks.HookConfig{}
	}

	relativePath := ".polis/hooks/" + req.HookType + ".sh"
	switch req.HookType {
	case "post-publish":
		s.Config.Hooks.PostPublish = relativePath
	case "post-republish":
		s.Config.Hooks.PostRepublish = relativePath
	case "post-comment":
		s.Config.Hooks.PostComment = relativePath
	}

	if err := s.SaveConfig(); err != nil {
		s.LogError("failed to save config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"hook_type":   req.HookType,
		"script_path": relativePath,
		"message":     "Created " + req.HookType + " hook at " + relativePath,
	})
}

// handleViewMode handles POST /api/settings/view-mode to switch between list and browser modes
func (s *Server) handleViewMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ViewMode string `json:"view_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate view mode
	if req.ViewMode != "list" && req.ViewMode != "browser" {
		http.Error(w, "Invalid view mode: must be 'list' or 'browser'", http.StatusBadRequest)
		return
	}

	// Ensure config exists
	if s.Config == nil {
		s.Config = &Config{}
	}

	// Update and save
	s.Config.ViewMode = req.ViewMode
	if err := s.SaveConfig(); err != nil {
		s.LogError("failed to save config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"view_mode": req.ViewMode,
	})
}

// handleShowFrontmatter handles POST /api/settings/show-frontmatter to toggle frontmatter visibility
func (s *Server) handleShowFrontmatter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ShowFrontmatter bool `json:"show_frontmatter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ensure config exists
	if s.Config == nil {
		s.Config = &Config{}
	}

	// Update and save
	s.Config.ShowFrontmatter = &req.ShowFrontmatter
	if err := s.SaveConfig(); err != nil {
		s.LogError("failed to save config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"show_frontmatter": req.ShowFrontmatter,
	})
}

// handleContent handles GET /api/content/{path} for browser mode navigation
func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL: /api/content/{path}
	contentPath := strings.TrimPrefix(r.URL.Path, "/api/content/")
	if contentPath == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Validate path to prevent directory traversal
	if err := validateContentPath(contentPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if this is an HTML file request
	if strings.HasSuffix(contentPath, ".html") {
		s.handleHTMLContent(w, contentPath)
		return
	}

	// Read the content file (markdown)
	fullPath := filepath.Join(s.DataDir, contentPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Determine content type and editability
	contentType := "page"
	editable := true // Default: own content is editable
	rawMarkdown := string(content)
	markdown := rawMarkdown // Start with raw, strip frontmatter for rendering

	if strings.HasPrefix(contentPath, "posts/") {
		contentType = "post"
		editable = true // Own posts are editable
		// Strip frontmatter for rendering
		markdown = publish.StripFrontmatter(rawMarkdown)
	} else if strings.HasPrefix(contentPath, "comments/blessed/") {
		contentType = "blessed_comment"
		editable = false // Blessed comments from others are not editable
	} else if strings.HasPrefix(contentPath, ".polis/posts/drafts/") || strings.HasPrefix(contentPath, ".polis/drafts/") {
		contentType = "draft"
		editable = true // Own drafts are editable
	} else if strings.HasSuffix(contentPath, ".md") && !strings.Contains(contentPath, "/") {
		// Root-level markdown files (index.md, about.md, etc.)
		contentType = "page"
		editable = true // Own pages are editable
		// Strip frontmatter for rendering if present
		if publish.HasFrontmatter(rawMarkdown) {
			markdown = publish.StripFrontmatter(rawMarkdown)
		}
	}

	// Render markdown to HTML (without frontmatter)
	html, err := render.MarkdownToHTML(markdown)
	if err != nil {
		s.LogError("failed to render markdown: %v", err)
		http.Error(w, "Failed to render markdown", http.StatusInternalServerError)
		return
	}

	// Parse frontmatter for metadata
	frontmatter := publish.ParseFrontmatter(rawMarkdown)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":     contentPath,
		"markdown": rawMarkdown, // Return full content including frontmatter
		"html":     html,
		"editable": editable,
		"type":     contentType,
		"metadata": frontmatter,
	})
}

// handleHTMLContent serves pre-rendered HTML files for browser mode
func (s *Server) handleHTMLContent(w http.ResponseWriter, contentPath string) {
	// First check if the HTML file exists (to validate the path)
	fullPath := filepath.Join(s.DataDir, contentPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Try to find the corresponding .md source file
	mdPath := strings.TrimSuffix(contentPath, ".html") + ".md"
	fullMdPath := filepath.Join(s.DataDir, mdPath)
	markdown := ""
	editable := false
	var metadata map[string]string
	var html string

	mdContent, err := os.ReadFile(fullMdPath)
	if err == nil {
		// Found the source markdown - render it fresh for consistent preview styling
		markdown = string(mdContent)
		editable = true // Can edit if we have the source
		metadata = publish.ParseFrontmatter(markdown)
		// Strip frontmatter for HTML rendering only
		markdownForRender := markdown
		if publish.HasFrontmatter(markdown) {
			markdownForRender = publish.StripFrontmatter(markdown)
		}
		// Render markdown to HTML (same as editor preview)
		renderedHTML, renderErr := render.MarkdownToHTML(markdownForRender)
		if renderErr == nil {
			html = renderedHTML
		}
	}

	// If we couldn't render from markdown, fall back to the pre-rendered HTML
	if html == "" {
		htmlContent, err := os.ReadFile(fullPath)
		if err != nil {
			http.Error(w, "Content not found", http.StatusNotFound)
			return
		}
		html = string(htmlContent)
	}

	// Determine content type
	contentType := "page"
	if strings.HasPrefix(contentPath, "posts/") {
		contentType = "post"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":       contentPath,
		"markdown":   markdown,
		"html":       html,
		"editable":   editable,
		"type":       contentType,
		"metadata":   metadata,
		"source_md":  mdPath,
		"has_source": markdown != "",
	})
}

// handleSiteRegistrationStatus returns the site's registration status with the discovery service.
func (s *Server) handleSiteRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if discovery service is configured
	if s.Config == nil || s.Config.DiscoveryURL == "" || s.Config.DiscoveryKey == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"configured": false,
			"error":      "Discovery service not configured",
		})
		return
	}

	// Extract domain from POLIS_BASE_URL
	baseURL := s.GetBaseURL()
	if baseURL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"configured": true,
			"error":      "POLIS_BASE_URL not set",
		})
		return
	}

	domain := polisurl.ExtractDomain(baseURL)
	if domain == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"configured": true,
			"error":      "Could not extract domain from POLIS_BASE_URL",
		})
		return
	}

	// Query discovery service for registration status
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)
	result, err := client.CheckSiteRegistration(domain)
	if err != nil {
		s.LogWarn("Failed to check registration status: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"configured": true,
			"domain":     domain,
			"error":      "Unable to check registration status",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"configured":    true,
		"domain":        domain,
		"is_registered": result.IsRegistered,
		"registered_at": result.RegisteredAt,
		"registry_url":  result.RegistryURL,
	})
}

// handleSiteRegister registers the site with the discovery service.
func (s *Server) handleSiteRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate discovery service is configured
	if s.Config == nil || s.Config.DiscoveryURL == "" || s.Config.DiscoveryKey == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	// Validate private key is available
	if s.PrivateKey == nil {
		http.Error(w, "Private key not available", http.StatusBadRequest)
		return
	}

	// Extract domain from POLIS_BASE_URL
	baseURL := s.GetBaseURL()
	if baseURL == "" {
		http.Error(w, "POLIS_BASE_URL not set", http.StatusBadRequest)
		return
	}

	domain := polisurl.ExtractDomain(baseURL)
	if domain == "" {
		http.Error(w, "Could not extract domain from POLIS_BASE_URL", http.StatusBadRequest)
		return
	}

	// Get email and author_name from .well-known/polis
	var email, authorName string
	wkp, err := s.LoadWellKnownPolis()
	if err == nil {
		// Try to get email and author from .well-known/polis if available
		// The WellKnownPolis struct currently doesn't have these fields,
		// so we'll read the raw JSON
		wellKnownPath := filepath.Join(s.DataDir, ".well-known", "polis")
		data, readErr := os.ReadFile(wellKnownPath)
		if readErr == nil {
			var rawWKP map[string]interface{}
			if json.Unmarshal(data, &rawWKP) == nil {
				if e, ok := rawWKP["email"].(string); ok {
					email = e
				}
				if a, ok := rawWKP["author_name"].(string); ok {
					authorName = a
				}
			}
		}
		_ = wkp // silence unused warning
	}

	// Register with discovery service
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)
	result, err := client.RegisterSite(domain, s.PrivateKey, email, authorName)
	if err != nil {
		s.LogError("Failed to register site: %v", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	s.LogInfo("Site registered successfully: %s", domain)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       result.Success,
		"domain":        domain,
		"registered_at": result.RegisteredAt,
		"registry_url":  result.RegistryURL,
	})
}

// handleSiteUnregister unregisters the site from the discovery service.
func (s *Server) handleSiteUnregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate discovery service is configured
	if s.Config == nil || s.Config.DiscoveryURL == "" || s.Config.DiscoveryKey == "" {
		http.Error(w, "Discovery service not configured", http.StatusBadRequest)
		return
	}

	// Validate private key is available
	if s.PrivateKey == nil {
		http.Error(w, "Private key not available", http.StatusBadRequest)
		return
	}

	// Extract domain from POLIS_BASE_URL
	baseURL := s.GetBaseURL()
	if baseURL == "" {
		http.Error(w, "POLIS_BASE_URL not set", http.StatusBadRequest)
		return
	}

	domain := polisurl.ExtractDomain(baseURL)
	if domain == "" {
		http.Error(w, "Could not extract domain from POLIS_BASE_URL", http.StatusBadRequest)
		return
	}

	// Unregister from discovery service
	client := discovery.NewClient(s.Config.DiscoveryURL, s.Config.DiscoveryKey)
	result, err := client.UnregisterSite(domain, s.PrivateKey)
	if err != nil {
		s.LogError("Failed to unregister site: %v", err)
		http.Error(w, "Unregistration failed", http.StatusInternalServerError)
		return
	}

	s.LogInfo("Site unregistered successfully: %s", domain)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": result.Success,
		"domain":  domain,
		"message": result.Message,
	})
}

// Snippets API handlers

// handleSnippets handles GET /api/snippets?path={subdir} and POST /api/snippets
func (s *Server) handleSnippets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List snippets at path
		path := r.URL.Query().Get("path")
		filter := r.URL.Query().Get("filter") // "all", "global", or "theme"

		// List snippets from both local .polis/themes/ and CLI themes (fallback)
		tree, err := snippet.ListSnippets(s.DataDir, s.CLIThemesDir, "", path, filter)
		if err != nil {
			s.LogError("failed to list snippets: %v", err)
			http.Error(w, "Failed to list snippets", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tree)

	case http.MethodPost:
		// Create new global snippet
		var req struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Path == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}

		if err := snippet.CreateSnippet(s.DataDir, req.Path, req.Content); err != nil {
			s.LogError("failed to create snippet: %v", err)
			http.Error(w, "Failed to create snippet", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"path":    req.Path,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSnippet handles GET/PUT/DELETE /api/snippets/{path}
func (s *Server) handleSnippet(w http.ResponseWriter, r *http.Request) {
	// Extract path from URL: /api/snippets/{path}
	snippetPath := strings.TrimPrefix(r.URL.Path, "/api/snippets/")
	if snippetPath == "" {
		http.Error(w, "Snippet path required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get snippet content
		source := r.URL.Query().Get("source")
		if source == "" {
			source = "global" // Default to global
		}

		// Read from local .polis/themes/ or CLI themes (fallback)
		content, err := snippet.ReadSnippet(s.DataDir, s.CLIThemesDir, "", snippetPath, source)
		if err != nil {
			// Try the other source if not found
			if source == "global" {
				content, err = snippet.ReadSnippet(s.DataDir, s.CLIThemesDir, "", snippetPath, "theme")
			}
			if err != nil {
				http.Error(w, "Snippet not found", http.StatusNotFound)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(content)

	case http.MethodPut:
		// Update snippet
		var req struct {
			Content string `json:"content"`
			Source  string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Source == "" {
			req.Source = "global" // Default to global
		}

		// Write to local .polis/themes/ or CLI themes (fallback)
		if err := snippet.WriteSnippet(s.DataDir, s.CLIThemesDir, "", snippetPath, req.Content, req.Source); err != nil {
			s.LogError("failed to save snippet: %v", err)
			http.Error(w, "Failed to save snippet", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"path":    snippetPath,
			"source":  req.Source,
		})

	case http.MethodDelete:
		// Delete snippet (global only)
		if err := snippet.DeleteSnippet(s.DataDir, snippetPath); err != nil {
			s.LogError("failed to delete snippet: %v", err)
			http.Error(w, "Failed to delete snippet", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"path":    snippetPath,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRenderPage handles POST /api/render-page to re-render pages using Go packages.
// This is used for snippet editing workflow - after saving a snippet, re-render
// the current page to see the changes.
func (s *Server) handleRenderPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get site base URL from POLIS_BASE_URL env var (matches bash CLI behavior)
	baseURL := s.GetBaseURL()

	// Create page renderer using Go packages
	renderer, err := render.NewPageRenderer(render.PageConfig{
		DataDir:       s.DataDir,
		CLIThemesDir:  s.CLIThemesDir,
		BaseURL:       baseURL,
		RenderMarkers: true, // Enable snippet markers for editing
	})
	if err != nil {
		log.Printf("[render-page] Failed to create renderer: %v", err)
		http.Error(w, "Failed to create renderer", http.StatusInternalServerError)
		return
	}

	// Render all pages with force=true to ensure snippets are updated
	stats, err := renderer.RenderAll(true)
	if err != nil {
		log.Printf("[render-page] Render failed: %v", err)
		http.Error(w, "Render failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[render-page] Rendered %d posts, %d comments, requested path: %s",
		stats.PostsRendered, stats.CommentsRendered, req.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":           true,
		"path":              req.Path,
		"posts_rendered":    stats.PostsRendered,
		"comments_rendered": stats.CommentsRendered,
	})
}

// ============================================================================
// Social handlers (following, feed, remote post)
// ============================================================================

// handleFollowing manages the following list.
// GET: returns the list of followed authors.
// POST: follows a new author (with blessing side-effect).
// DELETE: unfollows an author (with denial side-effect).
func (s *Server) handleFollowing(w http.ResponseWriter, r *http.Request) {
	followingPath := following.DefaultPath(s.DataDir)

	switch r.Method {
	case http.MethodGet:
		f, err := following.Load(followingPath)
		if err != nil {
			s.LogError("following load failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"following": f.All(),
			"count":     f.Count(),
		})

	case http.MethodPost:
		if s.PrivateKey == nil {
			http.Error(w, "Not configured: no private key", http.StatusBadRequest)
			return
		}

		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.URL) < 8 || req.URL[:8] != "https://" {
			http.Error(w, "Author URL must use HTTPS", http.StatusBadRequest)
			return
		}

		discoveryURL := DefaultDiscoveryServiceURL
		apiKey := ""
		if s.Config != nil {
			if s.Config.DiscoveryURL != "" {
				discoveryURL = s.Config.DiscoveryURL
			}
			apiKey = s.Config.DiscoveryKey
		}
		discoveryClient := discovery.NewClient(discoveryURL, apiKey)
		remoteClient := remote.NewClient()

		result, err := following.FollowWithBlessing(followingPath, req.URL, discoveryClient, remoteClient, s.PrivateKey)
		if err != nil {
			s.LogError("follow failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.LogInfo("Followed %s (blessed %d comments)", req.URL, result.CommentsBlessed)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    result,
		})

	case http.MethodDelete:
		if s.PrivateKey == nil {
			http.Error(w, "Not configured: no private key", http.StatusBadRequest)
			return
		}

		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.URL) < 8 || req.URL[:8] != "https://" {
			http.Error(w, "Author URL must use HTTPS", http.StatusBadRequest)
			return
		}

		discoveryURL := DefaultDiscoveryServiceURL
		apiKey := ""
		if s.Config != nil {
			if s.Config.DiscoveryURL != "" {
				discoveryURL = s.Config.DiscoveryURL
			}
			apiKey = s.Config.DiscoveryKey
		}
		discoveryClient := discovery.NewClient(discoveryURL, apiKey)
		remoteClient := remote.NewClient()

		result, err := following.UnfollowWithDenial(followingPath, req.URL, discoveryClient, remoteClient, s.PrivateKey)
		if err != nil {
			s.LogError("unfollow failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.LogInfo("Unfollowed %s (denied %d comments)", req.URL, result.CommentsDenied)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    result,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFeed returns cached feed items (instant, no network).
// GET /api/feed?type=post|comment
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cm := feed.NewCacheManager(s.DataDir)
	typeFilter := r.URL.Query().Get("type")

	var items []feed.CachedFeedItem
	var err error
	if typeFilter != "" {
		items, err = cm.ListByType(typeFilter)
	} else {
		items, err = cm.List()
	}
	if err != nil {
		s.LogError("feed list failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	unread := 0
	for _, item := range items {
		if item.ReadAt == "" {
			unread++
		}
	}

	stale, _ := cm.IsStale()
	manifest, _ := cm.LoadManifest()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":        items,
		"total":        len(items),
		"unread":       unread,
		"stale":        stale,
		"last_refresh": manifest.LastRefresh,
	})
}

// handleFeedRefresh runs Aggregate + Merge and returns the updated cache.
// POST /api/feed/refresh
func (s *Server) handleFeedRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	followingPath := following.DefaultPath(s.DataDir)
	client := remote.NewClient()

	result, err := feed.Aggregate(followingPath, client, feed.AggregateOptions{})
	if err != nil {
		s.LogError("feed aggregate failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cm := feed.NewCacheManager(s.DataDir)
	newCount, err := cm.Merge(result)
	if err != nil {
		s.LogError("feed merge failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.LogInfo("Feed refresh: checked %d authors, %d new items cached", result.AuthorsChecked, newCount)

	items, _ := cm.List()
	unread := 0
	for _, item := range items {
		if item.ReadAt == "" {
			unread++
		}
	}

	manifest, _ := cm.LoadManifest()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":        items,
		"total":        len(items),
		"unread":       unread,
		"new_items":    newCount,
		"stale":        false,
		"last_refresh": manifest.LastRefresh,
		"errors":       result.Errors,
	})
}

// handleFeedRead marks feed items as read/unread.
// POST /api/feed/read
// Body: {"id":"x"} | {"id":"x","unread":true} | {"all":true} | {"from_id":"x"}
func (s *Server) handleFeedRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Unread bool   `json:"unread"`
		All    bool   `json:"all"`
		FromID string `json:"from_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cm := feed.NewCacheManager(s.DataDir)

	var err error
	if req.All {
		err = cm.MarkAllRead()
	} else if req.FromID != "" {
		err = cm.MarkUnreadFrom(req.FromID)
	} else if req.ID != "" {
		if req.Unread {
			err = cm.MarkUnread(req.ID)
		} else {
			err = cm.MarkRead(req.ID)
		}
	} else {
		http.Error(w, "Missing id, all, or from_id", http.StatusBadRequest)
		return
	}

	if err != nil {
		s.LogError("feed read failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleFeedCounts returns lightweight feed counts for sidebar badge.
// GET /api/feed/counts
func (s *Server) handleFeedCounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cm := feed.NewCacheManager(s.DataDir)

	items, err := cm.List()
	if err != nil {
		s.LogError("feed counts failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	unread := 0
	for _, item := range items {
		if item.ReadAt == "" {
			unread++
		}
	}

	stale, _ := cm.IsStale()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":  len(items),
		"unread": unread,
		"stale":  stale,
	})
}

// handleRemotePost fetches a remote post and returns it as rendered HTML.
// GET /api/remote/post?url=https://example.com/posts/hello.md
func (s *Server) handleRemotePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postURL := r.URL.Query().Get("url")
	if postURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	if len(postURL) < 8 || postURL[:8] != "https://" {
		http.Error(w, "URL must use HTTPS", http.StatusBadRequest)
		return
	}

	client := remote.NewClient()

	// Try fetching the URL as-is first
	content, err := client.FetchContent(postURL)
	fetchedURL := postURL

	if err != nil {
		s.LogError("remote post fetch failed: %v", err)
		http.Error(w, "Failed to fetch remote post: "+err.Error(), http.StatusBadGateway)
		return
	}

	// If the response looks like HTML (not markdown), the host likely served
	// the rendered page instead of the raw source. Try the alternate extension.
	if looksLikeHTML(content) {
		altContent, altURL, altErr := client.TryAlternateExtension(postURL)
		if altErr == nil && !looksLikeHTML(altContent) {
			content = altContent
			fetchedURL = altURL
		}
		// If both extensions return HTML, use the original content as-is
	}

	var body, htmlContent string
	if looksLikeHTML(content) {
		// Content is already HTML — serve it directly (strip full page shell if present)
		htmlContent = extractHTMLBody(content)
		body = content
	} else {
		// Content is markdown — strip frontmatter and render
		body = stripFrontmatter(content)
		rendered, renderErr := render.MarkdownToHTML(body)
		if renderErr != nil {
			s.LogError("remote post render failed: %v", renderErr)
			http.Error(w, "Failed to render post", http.StatusInternalServerError)
			return
		}
		htmlContent = rendered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":     fetchedURL,
		"content": htmlContent,
		"raw":     body,
	})
}

// stripFrontmatter removes YAML frontmatter (---...---) from content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return content
	}
	// Return everything after the closing ---
	after := rest[idx+4:]
	return strings.TrimLeft(after, "\n")
}

// looksLikeHTML checks if content appears to be HTML rather than markdown.
func looksLikeHTML(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "<!DOCTYPE") ||
		strings.HasPrefix(trimmed, "<!doctype") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<HTML")
}

// extractHTMLBody extracts content between <body> and </body> tags,
// or between <main> and </main> tags, falling back to the full content.
func extractHTMLBody(content string) string {
	lower := strings.ToLower(content)

	// Try <main>...</main> first (most specific)
	if mainStart := strings.Index(lower, "<main"); mainStart >= 0 {
		// Find end of opening tag
		tagEnd := strings.Index(content[mainStart:], ">")
		if tagEnd >= 0 {
			innerStart := mainStart + tagEnd + 1
			if mainEnd := strings.Index(lower[innerStart:], "</main>"); mainEnd >= 0 {
				return strings.TrimSpace(content[innerStart : innerStart+mainEnd])
			}
		}
	}

	// Try <body>...</body>
	if bodyStart := strings.Index(lower, "<body"); bodyStart >= 0 {
		tagEnd := strings.Index(content[bodyStart:], ">")
		if tagEnd >= 0 {
			innerStart := bodyStart + tagEnd + 1
			if bodyEnd := strings.Index(lower[innerStart:], "</body>"); bodyEnd >= 0 {
				return strings.TrimSpace(content[innerStart : innerStart+bodyEnd])
			}
		}
	}

	return content
}
