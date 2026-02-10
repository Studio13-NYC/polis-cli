package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestRebuildCommentsIndex_UsesPackageVersion(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)

	opts := RebuildOptions{Comments: true}
	_, err := rebuildCommentsIndex(dataDir, opts)
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "metadata", "blessed-comments.json"))
	if err != nil {
		t.Fatalf("failed to read blessed-comments.json: %v", err)
	}

	if !strings.Contains(string(data), `"version": "`+GetGenerator()+`"`) {
		t.Errorf("blessed-comments.json should contain version %q, got: %s", GetGenerator(), string(data))
	}
}

func TestRegenerateManifest_UsesPackageVersion(t *testing.T) {
	dataDir := t.TempDir()
	os.MkdirAll(filepath.Join(dataDir, "posts"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "metadata"), 0755)

	if err := regenerateManifest(dataDir); err != nil {
		t.Fatalf("regenerateManifest failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "metadata", "manifest.json"))
	if err != nil {
		t.Fatalf("failed to read manifest.json: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to parse manifest.json: %v", err)
	}

	if manifest["version"] != GetGenerator() {
		t.Errorf("manifest.json version = %q, want %q", manifest["version"], GetGenerator())
	}
}
