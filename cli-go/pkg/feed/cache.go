package feed

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Version is set at init time by cmd package.
var Version = "dev"

// GetGenerator returns the generator identifier for metadata files.
func GetGenerator() string {
	return "polis-cli-go/" + Version
}

// CachedFeedItem represents a single item in the feed cache.
type CachedFeedItem struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Published    string `json:"published"`
	Hash         string `json:"hash,omitempty"`
	AuthorURL    string `json:"author_url"`
	AuthorDomain string `json:"author_domain"`
	CachedAt     string `json:"cached_at"`
	ReadAt       string `json:"read_at,omitempty"`
}

// CacheManifest stores metadata about the feed cache.
type CacheManifest struct {
	Version          string `json:"version"`
	LastRefresh      string `json:"last_refresh"`
	StalenessMinutes int    `json:"staleness_minutes"`
	MaxItems         int    `json:"max_items"`
	MaxAgeDays       int    `json:"max_age_days"`
}

// CacheManager handles feed cache operations.
type CacheManager struct {
	cacheFile    string
	manifestFile string
}

// DefaultCacheFile returns the default path to feed-cache.jsonl.
func DefaultCacheFile(dataDir string) string {
	return filepath.Join(dataDir, ".polis", "social", "feed-cache.jsonl")
}

// DefaultManifestFile returns the default path to feed-manifest.json.
func DefaultCacheManifestFile(dataDir string) string {
	return filepath.Join(dataDir, ".polis", "social", "feed-manifest.json")
}

// NewCacheManager creates a new feed cache manager.
func NewCacheManager(dataDir string) *CacheManager {
	return &CacheManager{
		cacheFile:    DefaultCacheFile(dataDir),
		manifestFile: DefaultCacheManifestFile(dataDir),
	}
}

// ComputeItemID generates a deterministic ID for a feed item.
// ID = first 16 hex chars of sha256(author_url + "|" + relative_path).
func ComputeItemID(authorURL, path string) string {
	h := sha256.Sum256([]byte(authorURL + "|" + path))
	return fmt.Sprintf("%x", h[:8])
}

// List returns all cached feed items, sorted by published descending.
func (cm *CacheManager) List() ([]CachedFeedItem, error) {
	file, err := os.Open(cm.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []CachedFeedItem{}, nil
		}
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	var items []CachedFeedItem
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item CachedFeedItem
		if err := json.Unmarshal(line, &item); err != nil {
			continue // Skip malformed lines
		}
		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	return items, nil
}

// ListByType returns cached feed items filtered by type ("post" or "comment").
func (cm *CacheManager) ListByType(itemType string) ([]CachedFeedItem, error) {
	all, err := cm.List()
	if err != nil {
		return nil, err
	}

	if itemType == "" {
		return all, nil
	}

	var filtered []CachedFeedItem
	for _, item := range all {
		if item.Type == itemType {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

// UnreadCount returns the number of unread items in the cache.
func (cm *CacheManager) UnreadCount() (int, error) {
	items, err := cm.List()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, item := range items {
		if item.ReadAt == "" {
			count++
		}
	}
	return count, nil
}

// IsStale returns true if the cache needs refreshing based on staleness_minutes.
func (cm *CacheManager) IsStale() (bool, error) {
	manifest, err := cm.LoadManifest()
	if err != nil {
		return true, nil // If we can't read manifest, consider it stale
	}

	if manifest.LastRefresh == "" {
		return true, nil
	}

	lastRefresh, err := time.Parse(time.RFC3339, manifest.LastRefresh)
	if err != nil {
		return true, nil
	}

	staleness := manifest.StalenessMinutes
	if staleness <= 0 {
		staleness = 15
	}

	return time.Since(lastRefresh) > time.Duration(staleness)*time.Minute, nil
}

// Merge integrates an AggregateResult into the cache. Returns the number of new items added.
func (cm *CacheManager) Merge(result *AggregateResult) (int, error) {
	existing, err := cm.List()
	if err != nil {
		return 0, err
	}

	// Build ID map of existing items
	idMap := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		idMap[item.ID] = struct{}{}
	}

	// Add new items
	now := time.Now().UTC().Format(time.RFC3339)
	newCount := 0
	for _, item := range result.Items {
		id := ComputeItemID(item.AuthorURL, item.URL)
		if _, exists := idMap[id]; exists {
			continue
		}
		existing = append(existing, CachedFeedItem{
			ID:           id,
			Type:         item.Type,
			Title:        item.Title,
			URL:          item.URL,
			Published:    item.Published,
			Hash:         item.Hash,
			AuthorURL:    item.AuthorURL,
			AuthorDomain: item.AuthorDomain,
			CachedAt:     now,
		})
		idMap[id] = struct{}{}
		newCount++
	}

	// Sort by published descending
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Published > existing[j].Published
	})

	if err := cm.writeAll(existing); err != nil {
		return 0, err
	}

	// Update manifest
	manifest, _ := cm.LoadManifest()
	manifest.LastRefresh = now
	if err := cm.saveManifest(manifest); err != nil {
		return newCount, err
	}

	// Prune after merge
	cm.Prune()

	return newCount, nil
}

// MarkRead marks a single item as read.
func (cm *CacheManager) MarkRead(id string) error {
	items, err := cm.List()
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	found := false
	for i := range items {
		if items[i].ID == id {
			items[i].ReadAt = now
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("item not found: %s", id)
	}

	return cm.writeAll(items)
}

// MarkUnread marks a single item as unread.
func (cm *CacheManager) MarkUnread(id string) error {
	items, err := cm.List()
	if err != nil {
		return err
	}

	found := false
	for i := range items {
		if items[i].ID == id {
			items[i].ReadAt = ""
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("item not found: %s", id)
	}

	return cm.writeAll(items)
}

// MarkAllRead marks all items as read.
func (cm *CacheManager) MarkAllRead() error {
	items, err := cm.List()
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for i := range items {
		if items[i].ReadAt == "" {
			items[i].ReadAt = now
		}
	}

	return cm.writeAll(items)
}

// MarkUnreadFrom marks the item with the given ID and all more recent items (by published date) as unread.
func (cm *CacheManager) MarkUnreadFrom(id string) error {
	items, err := cm.List()
	if err != nil {
		return err
	}

	// Find the target item's published date
	targetPublished := ""
	for _, item := range items {
		if item.ID == id {
			targetPublished = item.Published
			break
		}
	}

	if targetPublished == "" {
		return fmt.Errorf("item not found: %s", id)
	}

	// Mark the target and all items with same or newer published date as unread
	for i := range items {
		if items[i].Published >= targetPublished {
			items[i].ReadAt = ""
		}
	}

	return cm.writeAll(items)
}

// Prune enforces MaxItems and MaxAgeDays limits. Returns the number of items removed.
func (cm *CacheManager) Prune() (int, error) {
	items, err := cm.List()
	if err != nil {
		return 0, err
	}

	manifest, _ := cm.LoadManifest()

	maxAgeDays := manifest.MaxAgeDays
	if maxAgeDays <= 0 {
		maxAgeDays = 90
	}
	maxItems := manifest.MaxItems
	if maxItems <= 0 {
		maxItems = 500
	}

	originalLen := len(items)

	// Remove items older than MaxAgeDays
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays).UTC().Format(time.RFC3339)
	var remaining []CachedFeedItem
	for _, item := range items {
		if item.Published >= cutoff {
			remaining = append(remaining, item)
		}
	}

	// Enforce MaxItems (keep most recent)
	if len(remaining) > maxItems {
		remaining = remaining[:maxItems]
	}

	removed := originalLen - len(remaining)
	if removed > 0 {
		if err := cm.writeAll(remaining); err != nil {
			return 0, err
		}
	}

	return removed, nil
}

// LoadManifest loads the cache manifest, returning defaults if not found.
func (cm *CacheManager) LoadManifest() (*CacheManifest, error) {
	data, err := os.ReadFile(cm.manifestFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheManifest{
				Version:          Version,
				StalenessMinutes: 15,
				MaxItems:         500,
				MaxAgeDays:       90,
			}, nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// saveManifest writes the manifest to disk.
func (cm *CacheManager) saveManifest(manifest *CacheManifest) error {
	if err := os.MkdirAll(filepath.Dir(cm.manifestFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	manifest.Version = GetGenerator()

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(cm.manifestFile, append(data, '\n'), 0644)
}

// SetStalenessMinutes updates the staleness threshold.
func (cm *CacheManager) SetStalenessMinutes(minutes int) error {
	if minutes < 1 {
		minutes = 1
	}

	manifest, err := cm.LoadManifest()
	if err != nil {
		return err
	}

	manifest.StalenessMinutes = minutes
	return cm.saveManifest(manifest)
}

// writeAll rewrites all items to the cache file.
func (cm *CacheManager) writeAll(items []CachedFeedItem) error {
	if err := os.MkdirAll(filepath.Dir(cm.cacheFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(cm.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		file.WriteString(string(data) + "\n")
	}

	return nil
}

