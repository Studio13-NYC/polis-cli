package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRebuildPostsIndex_SkipsVersionsDir(t *testing.T) {
	dataDir := t.TempDir()
	postsDir := filepath.Join(dataDir, "posts", "20260101")
	versionsDir := filepath.Join(postsDir, ".versions")
	metadataDir := filepath.Join(dataDir, "metadata")
	os.MkdirAll(postsDir, 0755)
	os.MkdirAll(versionsDir, 0755)
	os.MkdirAll(metadataDir, 0755)

	// Create a real post
	postContent := `---
title: Hello World
published: 2026-01-01T00:00:00Z
current-version: sha256:abc
---

Hello World content.
`
	os.WriteFile(filepath.Join(postsDir, "hello.md"), []byte(postContent), 0644)

	// Create a version file that should NOT be indexed
	os.WriteFile(filepath.Join(versionsDir, "hello.md"), []byte("version data"), 0644)

	count, err := rebuildPostsIndex(dataDir, "https://test.polis.pub")
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 post (skipping .versions), got %d", count)
	}
}

func TestRebuildCommentsIndex_EmptyWhenNoDiscovery(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)

	opts := RebuildOptions{Comments: true}
	count, err := rebuildCommentsIndex(dataDir, opts)
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 comments without discovery, got %d", count)
	}

	// Verify file was created
	blessedPath := filepath.Join(dataDir, "metadata", "blessed-comments.json")
	if _, err := os.Stat(blessedPath); os.IsNotExist(err) {
		t.Error("expected blessed-comments.json to be created")
	}
}
