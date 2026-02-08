package feed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeItemID(t *testing.T) {
	// Same inputs produce same ID
	id1 := ComputeItemID("https://alice.polis.pub", "posts/hello.md")
	id2 := ComputeItemID("https://alice.polis.pub", "posts/hello.md")
	if id1 != id2 {
		t.Errorf("same inputs should produce same ID: %s vs %s", id1, id2)
	}

	// Different inputs produce different IDs
	id3 := ComputeItemID("https://bob.polis.pub", "posts/hello.md")
	if id1 == id3 {
		t.Errorf("different authors should produce different IDs")
	}

	// ID is 16 hex chars
	if len(id1) != 16 {
		t.Errorf("expected 16-char ID, got %d: %s", len(id1), id1)
	}
}

func TestCacheManager_EmptyCache(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	items, err := cm.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}

	count, err := cm.UnreadCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 unread, got %d", count)
	}

	stale, err := cm.IsStale()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stale {
		t.Error("empty cache should be stale")
	}
}

func TestCacheManager_Merge(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "First Post", URL: "posts/first.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "post", Title: "Second Post", URL: "posts/second.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "comment", Title: "A Comment", URL: "comments/reply.md", Published: "2026-02-03T10:00:00Z", AuthorURL: "https://bob.polis.pub", AuthorDomain: "bob.polis.pub"},
		},
	}

	newCount, err := cm.Merge(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCount != 3 {
		t.Errorf("expected 3 new items, got %d", newCount)
	}

	items, err := cm.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Should be sorted by published descending
	if items[0].Title != "A Comment" {
		t.Errorf("expected newest first, got %s", items[0].Title)
	}
	if items[2].Title != "First Post" {
		t.Errorf("expected oldest last, got %s", items[2].Title)
	}

	// All should be unread
	for _, item := range items {
		if item.ReadAt != "" {
			t.Errorf("new items should be unread, got ReadAt=%s", item.ReadAt)
		}
		if item.CachedAt == "" {
			t.Error("CachedAt should be set")
		}
		if item.ID == "" {
			t.Error("ID should be set")
		}
	}
}

func TestCacheManager_MergeDedup(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
		},
	}

	// First merge
	newCount, err := cm.Merge(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCount != 1 {
		t.Errorf("expected 1 new, got %d", newCount)
	}

	// Mark it read
	items, _ := cm.List()
	cm.MarkRead(items[0].ID)

	// Second merge with same item (different title shouldn't matter, same author+path)
	result.Items[0].Title = "Post A Updated"
	newCount, err = cm.Merge(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCount != 0 {
		t.Errorf("expected 0 new (dedup), got %d", newCount)
	}

	// Read state should be preserved
	items, _ = cm.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ReadAt == "" {
		t.Error("read state should be preserved after dedup merge")
	}
}

func TestCacheManager_MarkRead(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "post", Title: "Post B", URL: "posts/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
		},
	}
	cm.Merge(result)

	items, _ := cm.List()
	if err := cm.MarkRead(items[0].ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items, _ = cm.List()
	if items[0].ReadAt == "" {
		t.Error("item should be marked as read")
	}
	if items[1].ReadAt != "" {
		t.Error("other item should still be unread")
	}

	unread, _ := cm.UnreadCount()
	if unread != 1 {
		t.Errorf("expected 1 unread, got %d", unread)
	}
}

func TestCacheManager_MarkUnread(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
		},
	}
	cm.Merge(result)

	items, _ := cm.List()
	cm.MarkRead(items[0].ID)

	// Verify it's read
	items, _ = cm.List()
	if items[0].ReadAt == "" {
		t.Fatal("should be read")
	}

	// Mark unread
	if err := cm.MarkUnread(items[0].ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items, _ = cm.List()
	if items[0].ReadAt != "" {
		t.Error("should be unread again")
	}
}

func TestCacheManager_MarkAllRead(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "post", Title: "Post B", URL: "posts/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "comment", Title: "Comment C", URL: "comments/c.md", Published: "2026-02-03T10:00:00Z", AuthorURL: "https://bob.polis.pub", AuthorDomain: "bob.polis.pub"},
		},
	}
	cm.Merge(result)

	if err := cm.MarkAllRead(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unread, _ := cm.UnreadCount()
	if unread != 0 {
		t.Errorf("expected 0 unread, got %d", unread)
	}
}

func TestCacheManager_MarkUnreadFrom(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Old", URL: "posts/old.md", Published: "2026-01-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "post", Title: "Mid", URL: "posts/mid.md", Published: "2026-01-15T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "post", Title: "New", URL: "posts/new.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
		},
	}
	cm.Merge(result)

	// Mark all read first
	cm.MarkAllRead()

	// Get the middle item's ID
	items, _ := cm.List()
	// Items sorted desc: New, Mid, Old
	midID := items[1].ID

	if err := cm.MarkUnreadFrom(midID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items, _ = cm.List()
	// New (2026-02-01) should be unread (more recent than mid)
	if items[0].ReadAt != "" {
		t.Error("New should be unread")
	}
	// Mid (2026-01-15) should be unread (the target)
	if items[1].ReadAt != "" {
		t.Error("Mid should be unread")
	}
	// Old (2026-01-01) should still be read (older than mid)
	if items[2].ReadAt == "" {
		t.Error("Old should still be read")
	}
}

func TestCacheManager_ListByType(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
			{Type: "comment", Title: "Comment B", URL: "comments/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://bob.polis.pub", AuthorDomain: "bob.polis.pub"},
			{Type: "post", Title: "Post C", URL: "posts/c.md", Published: "2026-02-03T10:00:00Z", AuthorURL: "https://alice.polis.pub", AuthorDomain: "alice.polis.pub"},
		},
	}
	cm.Merge(result)

	posts, err := cm.ListByType("post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}

	comments, err := cm.ListByType("comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}

	all, err := cm.ListByType("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total, got %d", len(all))
	}
}

func TestCacheManager_Prune(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	// Set low limits for testing
	cm.SetStalenessMinutes(1)
	manifest, _ := cm.LoadManifest()
	manifest.MaxItems = 2
	manifest.MaxAgeDays = 90
	cm.saveManifest(manifest)

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Post 1", URL: "posts/1.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "Post 2", URL: "posts/2.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "Post 3", URL: "posts/3.md", Published: "2026-02-03T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		},
	}
	cm.Merge(result) // Merge calls Prune internally

	items, err := cm.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items after prune (MaxItems=2), got %d", len(items))
	}
	// Should keep the most recent
	if items[0].Title != "Post 3" {
		t.Errorf("expected most recent first, got %s", items[0].Title)
	}
}

func TestCacheManager_PruneByAge(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	// Set MaxAgeDays to 30
	manifest, _ := cm.LoadManifest()
	manifest.MaxAgeDays = 30
	manifest.MaxItems = 500
	cm.saveManifest(manifest)

	oldDate := time.Now().AddDate(0, 0, -60).UTC().Format(time.RFC3339)
	recentDate := time.Now().UTC().Format(time.RFC3339)

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Old Post", URL: "posts/old.md", Published: oldDate, AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "Recent Post", URL: "posts/recent.md", Published: recentDate, AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		},
	}
	cm.Merge(result)

	items, _ := cm.List()
	if len(items) != 1 {
		t.Errorf("expected 1 item after age prune, got %d", len(items))
	}
	if len(items) > 0 && items[0].Title != "Recent Post" {
		t.Errorf("expected recent post to survive, got %s", items[0].Title)
	}
}

func TestCacheManager_IsStale(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	// No manifest = stale
	stale, _ := cm.IsStale()
	if !stale {
		t.Error("should be stale with no manifest")
	}

	// Set last_refresh to now with 15 minute staleness
	manifest := &CacheManifest{
		Version:          "test",
		LastRefresh:      time.Now().UTC().Format(time.RFC3339),
		StalenessMinutes: 15,
		MaxItems:         500,
		MaxAgeDays:       90,
	}
	cm.saveManifest(manifest)

	stale, _ = cm.IsStale()
	if stale {
		t.Error("should not be stale right after refresh")
	}

	// Set last_refresh to 20 minutes ago
	manifest.LastRefresh = time.Now().Add(-20 * time.Minute).UTC().Format(time.RFC3339)
	cm.saveManifest(manifest)

	stale, _ = cm.IsStale()
	if !stale {
		t.Error("should be stale after staleness period")
	}
}

func TestCacheManager_Manifest(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	// Default manifest
	manifest, err := cm.LoadManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifest.StalenessMinutes != 15 {
		t.Errorf("expected default staleness 15, got %d", manifest.StalenessMinutes)
	}
	if manifest.MaxItems != 500 {
		t.Errorf("expected default max items 500, got %d", manifest.MaxItems)
	}

	// Update staleness
	if err := cm.SetStalenessMinutes(30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, _ = cm.LoadManifest()
	if manifest.StalenessMinutes != 30 {
		t.Errorf("expected staleness 30, got %d", manifest.StalenessMinutes)
	}
}

func TestCacheManager_VersionPropagation(t *testing.T) {
	dir := t.TempDir()
	Version = "0.49.0"
	defer func() { Version = "dev" }()

	cm := NewCacheManager(dir)

	// Merge some items to trigger manifest write
	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Test", URL: "posts/test.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		},
	}
	cm.Merge(result)

	// Check manifest version
	data, err := os.ReadFile(filepath.Join(dir, ".polis", "social", "feed-manifest.json"))
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest CacheManifest
	json.Unmarshal(data, &manifest)
	if manifest.Version != "0.49.0" {
		t.Errorf("expected version 0.49.0, got %s", manifest.Version)
	}
}

func TestCacheManager_MarkReadNotFound(t *testing.T) {
	cm := NewCacheManager(t.TempDir())

	err := cm.MarkRead("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}

	err = cm.MarkUnread("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}

	err = cm.MarkUnreadFrom("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestCacheManager_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Don't pre-create .polis/social/
	cm := NewCacheManager(dir)

	result := &AggregateResult{
		Items: []FeedItem{
			{Type: "post", Title: "Test", URL: "posts/test.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		},
	}

	_, err := cm.Merge(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(filepath.Join(dir, ".polis", "social", "feed-cache.jsonl")); err != nil {
		t.Error("cache file should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".polis", "social", "feed-manifest.json")); err != nil {
		t.Error("manifest file should exist")
	}
}
