package comment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
