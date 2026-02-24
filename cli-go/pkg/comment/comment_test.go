package comment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

func TestGetGenerator_UsesVersion(t *testing.T) {
	old := Version
	defer func() { Version = old }()

	Version = "1.2.3"
	got := GetGenerator()
	if got != "polis-cli-go/1.2.3" {
		t.Errorf("GetGenerator() = %q, want %q", got, "polis-cli-go/1.2.3")
	}
}

func TestGetGenerator_DefaultDev(t *testing.T) {
	// Default version is "dev"
	old := Version
	defer func() { Version = old }()
	Version = "dev"

	got := GetGenerator()
	if got != "polis-cli-go/dev" {
		t.Errorf("GetGenerator() = %q, want %q", got, "polis-cli-go/dev")
	}
}

func TestEnsureUniqueCommentID_NoCollision(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "drafts"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "denied"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)

	result := ensureUniqueCommentID(dataDir, "alice-hello-20260101")
	if result != "alice-hello-20260101" {
		t.Errorf("expected original ID, got %s", result)
	}
}

func TestEnsureUniqueCommentID_DraftCollision(t *testing.T) {
	dataDir := t.TempDir()
	draftsDir := filepath.Join(dataDir, ".polis", "comments", "drafts")
	os.MkdirAll(draftsDir, 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "denied"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)

	// Create existing draft
	os.WriteFile(filepath.Join(draftsDir, "alice-hello-20260101.md"), []byte("draft"), 0644)

	result := ensureUniqueCommentID(dataDir, "alice-hello-20260101")
	if result != "alice-hello-20260101-2" {
		t.Errorf("expected suffixed ID, got %s", result)
	}
}

func TestEnsureUniqueCommentID_BlessedCollision(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "drafts"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "denied"), 0755)
	blessedDir := filepath.Join(dataDir, "comments", "20260101")
	os.MkdirAll(blessedDir, 0755)

	// Create existing blessed comment
	os.WriteFile(filepath.Join(blessedDir, "alice-hello-20260101.md"), []byte("blessed"), 0644)

	result := ensureUniqueCommentID(dataDir, "alice-hello-20260101")
	if result != "alice-hello-20260101-2" {
		t.Errorf("expected suffixed ID, got %s", result)
	}
}

func TestMoveComment_AddsToBlessedComments(t *testing.T) {
	dataDir := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)

	// Create a pending comment with CLI-compatible frontmatter
	commentContent := `---
title: Re: hello-world
type: comment
published: 2026-01-02T10:00:00Z
generator: polis-cli-go/0.1.0
in-reply-to:
  url: https://alice.polis.pub/posts/20260101/hello-world.md
  root-post: https://alice.polis.pub/posts/20260101/hello-world.md
current-version: sha256:abc123
comment_url: https://bob.polis.pub/comments/20260102/bob-hello-world-20260102.md
signature: fakesig
---

This is a test comment.`

	commentID := "bob-hello-world-20260102"
	pendingPath := filepath.Join(dataDir, ".polis", "comments", "pending", commentID+".md")
	os.WriteFile(pendingPath, []byte(commentContent), 0644)

	// Move to blessed
	err := MoveComment(dataDir, commentID, StatusPending, StatusBlessed)
	if err != nil {
		t.Fatalf("MoveComment failed: %v", err)
	}

	// Verify blessed-comments.json was created/updated
	blessedPath := filepath.Join(dataDir, "metadata", "blessed-comments.json")
	if _, err := os.Stat(blessedPath); os.IsNotExist(err) {
		t.Fatal("expected blessed-comments.json to be created")
	}

	// Verify manifest.json was created/updated with correct comment count
	manifestPath := filepath.Join(dataDir, "metadata", "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected manifest.json to be created: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", err)
	}
	commentCount := int(manifest["comment_count"].(float64))
	if commentCount != 1 {
		t.Errorf("manifest comment_count = %d, want 1", commentCount)
	}
}

func TestPublishComment_CopiesFilesAndUpdatesIndex(t *testing.T) {
	dataDir := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "posts"), 0755)

	// Create a pending comment
	commentContent := `---
title: Re: hello-world
type: comment
published: 2026-02-15T10:00:00Z
generator: polis-cli-go/0.50.0
in-reply-to:
  url: https://alice.polis.pub/posts/20260215/hello-world.md
  root-post: https://alice.polis.pub/posts/20260215/hello-world.md
current-version: sha256:abc123
author: bob.polis.pub
signature: fakesig
---

This is a **test** comment.`

	commentID := "bob-hello-world-20260215"
	pendingPath := filepath.Join(dataDir, ".polis", "comments", "pending", commentID+".md")
	os.WriteFile(pendingPath, []byte(commentContent), 0644)

	// Publish
	err := PublishComment(dataDir, commentID)
	if err != nil {
		t.Fatalf("PublishComment failed: %v", err)
	}

	// Verify .md file was copied to comments/YYYYMMDD/
	mdPath := filepath.Join(dataDir, "comments", "20260215", commentID+".md")
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("expected .md file in comments/20260215/")
	}

	// Note: HTML rendering is now handled by RenderSite(), not PublishComment().
	// The .html file is NOT expected here.

	// Verify public.jsonl was updated
	indexPath := filepath.Join(dataDir, "metadata", "public.jsonl")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("expected public.jsonl: %v", err)
	}
	if !containsString(string(indexData), commentID) {
		t.Error("public.jsonl does not contain published comment")
	}

	// Verify pending file is NOT removed (publish is a copy, not a move)
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Error("pending file should still exist after publish (removed by MoveComment later)")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// SignComment Domain Identity Tests (Phase 0)
// ============================================================================

func TestSignComment_DomainInFrontmatter(t *testing.T) {
	dataDir := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "drafts"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "denied"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)

	// Generate a test key
	privKey := generateTestKey(t)

	draft := &CommentDraft{
		InReplyTo: "https://alice.polis.pub/posts/20260101/hello.md",
		Content:   "Great post!",
	}

	signed, err := SignComment(dataDir, draft, "bob.polis.pub", "https://bob.polis.pub", privKey)
	if err != nil {
		t.Fatalf("SignComment failed: %v", err)
	}

	// Verify meta uses domain, not email
	if signed.Meta.Author != "bob.polis.pub" {
		t.Errorf("Meta.Author = %q, want %q", signed.Meta.Author, "bob.polis.pub")
	}

	// Verify the written file has domain in frontmatter
	pendingDir := filepath.Join(dataDir, ".polis", "comments", "pending")
	entries, _ := os.ReadDir(pendingDir)
	if len(entries) == 0 {
		t.Fatal("Expected a pending comment file")
	}
	data, _ := os.ReadFile(filepath.Join(pendingDir, entries[0].Name()))
	fm := ParseFrontmatter(string(data))
	if fm["author"] != "bob.polis.pub" {
		t.Errorf("Frontmatter author = %q, want %q", fm["author"], "bob.polis.pub")
	}
}

func TestSignComment_BackwardCompatWithEmail(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "drafts"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "denied"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)

	privKey := generateTestKey(t)

	draft := &CommentDraft{
		InReplyTo: "https://alice.polis.pub/posts/20260101/hello.md",
		Content:   "Legacy comment",
	}

	// Passing email as author identity should still work (backward compat)
	signed, err := SignComment(dataDir, draft, "bob@example.com", "https://bob.example.com", privKey)
	if err != nil {
		t.Fatalf("SignComment with email should still work: %v", err)
	}
	if signed.Meta.Author != "bob@example.com" {
		t.Errorf("Meta.Author = %q, want %q", signed.Meta.Author, "bob@example.com")
	}
}

func generateTestKey(t *testing.T) []byte {
	t.Helper()
	// Use the signing package to generate a real key
	privKey, _, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	return privKey
}

func TestMoveComment_UpdatesManifestCommentCount(t *testing.T) {
	dataDir := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(dataDir, ".polis", "comments", "pending"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "comments"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "posts"), 0755)

	// Create initial manifest with comment_count: 0
	initialManifest := map[string]interface{}{
		"version":        "0.47.0",
		"last_published": "",
		"post_count":     0,
		"comment_count":  0,
	}
	manifestData, _ := json.MarshalIndent(initialManifest, "", "  ")
	os.WriteFile(filepath.Join(dataDir, "metadata", "manifest.json"), manifestData, 0644)

	// Create two pending comments
	for i, id := range []string{"comment-a", "comment-b"} {
		content := `---
title: Re: hello-world
type: comment
published: 2026-01-02T10:00:00Z
generator: polis-cli-go/0.47.0
in-reply-to:
  url: https://alice.polis.pub/posts/20260101/hello-world.md
  root-post: https://alice.polis.pub/posts/20260101/hello-world.md
current-version: sha256:abc` + string(rune('0'+i)) + `
signature: fakesig
---

Test comment ` + id
		pendingPath := filepath.Join(dataDir, ".polis", "comments", "pending", id+".md")
		os.WriteFile(pendingPath, []byte(content), 0644)
	}

	// Bless first comment
	if err := MoveComment(dataDir, "comment-a", StatusPending, StatusBlessed); err != nil {
		t.Fatalf("MoveComment failed: %v", err)
	}

	// Check manifest shows 1 comment
	data, _ := os.ReadFile(filepath.Join(dataDir, "metadata", "manifest.json"))
	var m1 map[string]interface{}
	json.Unmarshal(data, &m1)
	if int(m1["comment_count"].(float64)) != 1 {
		t.Errorf("after first blessing: comment_count = %v, want 1", m1["comment_count"])
	}

	// Bless second comment
	if err := MoveComment(dataDir, "comment-b", StatusPending, StatusBlessed); err != nil {
		t.Fatalf("MoveComment failed: %v", err)
	}

	// Check manifest shows 2 comments
	data, _ = os.ReadFile(filepath.Join(dataDir, "metadata", "manifest.json"))
	var m2 map[string]interface{}
	json.Unmarshal(data, &m2)
	if int(m2["comment_count"].(float64)) != 2 {
		t.Errorf("after second blessing: comment_count = %v, want 2", m2["comment_count"])
	}
}
