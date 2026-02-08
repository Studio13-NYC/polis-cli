package feed

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/remote"
)

// mockServer creates an httptest server that serves manifest and public index for a mock polis site.
func mockServer(t *testing.T, lastPublished string, entries []remote.PublicIndexEntry) *httptest.Server {
	t.Helper()
	manifest := remote.Manifest{
		Version:       "0.48.0",
		LastPublished: lastPublished,
		PostCount:     len(entries),
	}
	manifestJSON, _ := json.Marshal(manifest)

	var indexLines []string
	for _, e := range entries {
		line, _ := json.Marshal(e)
		indexLines = append(indexLines, string(line))
	}
	indexContent := strings.Join(indexLines, "\n")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/metadata/manifest.json"):
			w.Header().Set("Content-Type", "application/json")
			w.Write(manifestJSON)
		case strings.HasSuffix(r.URL.Path, "/metadata/public.jsonl"):
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, indexContent)
		default:
			http.NotFound(w, r)
		}
	}))
}

func setupFollowing(t *testing.T, authors ...string) string {
	t.Helper()
	tmpDir := t.TempDir()
	followingPath := filepath.Join(tmpDir, "following.json")
	f := &following.FollowingFile{
		Version:   "test",
		Following: []following.FollowingEntry{},
	}
	for _, url := range authors {
		f.Add(url)
	}
	if err := following.Save(followingPath, f); err != nil {
		t.Fatalf("failed to save following.json: %v", err)
	}
	return followingPath
}

func TestAggregate_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	followingPath := filepath.Join(tmpDir, "following.json")
	// No following.json file — Load returns empty

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.AuthorsChecked != 0 {
		t.Errorf("expected 0 authors checked, got %d", result.AuthorsChecked)
	}
}

func TestAggregate_SingleAuthor(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "Hello World", Path: "posts/hello.md", Published: "2026-01-15T12:00:00Z"},
		{Type: "post", Title: "Second Post", Path: "posts/second.md", Published: "2026-01-10T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	followingPath := setupFollowing(t, server.URL)

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AuthorsChecked != 1 {
		t.Errorf("expected 1 author checked, got %d", result.AuthorsChecked)
	}
	if result.AuthorsWithNew != 1 {
		t.Errorf("expected 1 author with new, got %d", result.AuthorsWithNew)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}

	// Check items are sorted by published desc
	if len(result.Items) >= 2 && result.Items[0].Published < result.Items[1].Published {
		t.Error("expected items sorted by published descending")
	}
}

func TestAggregate_WithSinceOverride(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "New Post", Path: "posts/new.md", Published: "2026-01-15T12:00:00Z"},
		{Type: "post", Title: "Old Post", Path: "posts/old.md", Published: "2026-01-05T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	followingPath := setupFollowing(t, server.URL)

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{
		SinceOverride: "2026-01-10T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the new post should be included
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].Title != "New Post" {
		t.Errorf("expected 'New Post', got %q", result.Items[0].Title)
	}
}

func TestAggregate_SpecificAuthor_NotFollowing(t *testing.T) {
	followingPath := setupFollowing(t, "https://example.com")

	client := remote.NewClient()
	_, err := Aggregate(followingPath, client, AggregateOptions{
		SpecificAuthor: "https://other.com",
	})
	if err == nil {
		t.Fatal("expected error for non-followed author")
	}
	if _, ok := err.(*NotFollowingError); !ok {
		t.Errorf("expected NotFollowingError, got %T", err)
	}
}

func TestAggregate_UnreachableAuthor(t *testing.T) {
	// Use a URL that won't resolve
	followingPath := setupFollowing(t, "https://127.0.0.1:1")

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error for unreachable author")
	}
	if result.AuthorsChecked != 1 {
		t.Errorf("expected 1 author checked, got %d", result.AuthorsChecked)
	}
}

func TestAggregate_UpdatesLastChecked(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "Post", Path: "posts/p.md", Published: "2026-01-15T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	followingPath := setupFollowing(t, server.URL)

	client := remote.NewClient()
	_, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reload following.json and check last_checked was set
	f, err := following.Load(followingPath)
	if err != nil {
		t.Fatalf("failed to reload following.json: %v", err)
	}
	entry := f.Get(server.URL)
	if entry == nil {
		t.Fatal("expected to find entry after aggregate")
	}
	if entry.LastChecked == "" {
		t.Error("expected LastChecked to be set after aggregate")
	}
}

func TestAggregate_NoNewContent(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "Old Post", Path: "posts/old.md", Published: "2026-01-05T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-05T12:00:00Z", entries)
	defer server.Close()

	// Set up following with a last_checked in the future
	tmpDir := t.TempDir()
	followingPath := filepath.Join(tmpDir, "following.json")
	f := &following.FollowingFile{
		Version: "test",
		Following: []following.FollowingEntry{
			{
				URL:         server.URL,
				AddedAt:     "2026-01-01T00:00:00Z",
				LastChecked: "2026-02-01T00:00:00Z", // Already checked after last_published
			},
		},
	}
	if err := following.Save(followingPath, f); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AuthorsWithNew != 0 {
		t.Errorf("expected 0 authors with new, got %d", result.AuthorsWithNew)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url    string
		domain string
	}{
		{"https://example.com", "example.com"},
		{"https://example.com/path", "example.com"},
		{"https://sub.example.com/path/to/thing", "sub.example.com"},
		{"http://localhost:8080", "localhost:8080"},
	}
	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.domain {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.domain)
		}
	}
}

func TestAggregate_LimitsToTenForNeverChecked(t *testing.T) {
	var entries []remote.PublicIndexEntry
	for i := 0; i < 15; i++ {
		entries = append(entries, remote.PublicIndexEntry{
			Type:      "post",
			Title:     fmt.Sprintf("Post %d", i),
			Path:      fmt.Sprintf("posts/post-%d.md", i),
			Published: fmt.Sprintf("2026-01-%02dT12:00:00Z", i+1),
		})
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	followingPath := setupFollowing(t, server.URL)

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 10 {
		t.Errorf("expected 10 items (limit for never-checked), got %d", len(result.Items))
	}
}

func TestAggregate_SpecialCharactersInTitles(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "It's Not Beyond Our Reach", Path: "posts/its-not-beyond-our-reach.md", Published: "2026-01-15T12:00:00Z"},
		{Type: "post", Title: `She said "hello" & waved`, Path: "posts/she-said-hello.md", Published: "2026-01-14T12:00:00Z"},
		{Type: "post", Title: "2 < 3 && 5 > 4", Path: "posts/math.md", Published: "2026-01-13T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	followingPath := setupFollowing(t, server.URL)

	client := remote.NewClient()
	result, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}

	// Verify titles with special characters are preserved verbatim
	expectedTitles := []string{
		"It's Not Beyond Our Reach",
		`She said "hello" & waved`,
		"2 < 3 && 5 > 4",
	}
	for i, want := range expectedTitles {
		if result.Items[i].Title != want {
			t.Errorf("item[%d] title = %q, want %q", i, result.Items[i].Title, want)
		}
	}
}

// Ensure following.json dir is created if needed
func TestAggregate_CreatesDirectory(t *testing.T) {
	entries := []remote.PublicIndexEntry{
		{Type: "post", Title: "Post", Path: "posts/p.md", Published: "2026-01-15T12:00:00Z"},
	}
	server := mockServer(t, "2026-01-15T12:00:00Z", entries)
	defer server.Close()

	tmpDir := t.TempDir()
	followingPath := filepath.Join(tmpDir, "metadata", "following.json")

	// Pre-create the following file in a nested dir
	os.MkdirAll(filepath.Dir(followingPath), 0755)
	f := &following.FollowingFile{
		Version:   "test",
		Following: []following.FollowingEntry{},
	}
	f.Add(server.URL)
	following.Save(followingPath, f)

	client := remote.NewClient()
	_, err := Aggregate(followingPath, client, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
