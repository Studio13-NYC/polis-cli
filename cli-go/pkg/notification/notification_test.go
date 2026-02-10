package notification

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestNotifications(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	// Test empty list
	notifications, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications, got %d", len(notifications))
	}

	// Test Add
	payload, _ := json.Marshal(map[string]string{"message": "test"})
	id, err := mgr.Add("test_type", "test_source", payload, "")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty ID")
	}

	// Test list after add
	notifications, _ = mgr.List()
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	// Test Count
	count, _ := mgr.Count()
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	payload, _ := json.Marshal(map[string]string{"message": "test"})

	// Add with dedupe key
	id1, _ := mgr.Add("test_type", "test_source", payload, "dedupe_key_1")

	// Try to add again with same dedupe key
	id2, _ := mgr.Add("test_type", "test_source", payload, "dedupe_key_1")

	if id1 == "" {
		t.Error("First add should return ID")
	}

	if id2 != "" {
		t.Error("Duplicate add should return empty ID")
	}

	count, _ := mgr.Count()
	if count != 1 {
		t.Errorf("Expected count 1 (no duplicates), got %d", count)
	}
}

func TestInitManifest_UsesGeneratorVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if err := mgr.InitManifest(); err != nil {
		t.Fatalf("InitManifest failed: %v", err)
	}

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	expected := GetGenerator()
	if manifest.Version != expected {
		t.Errorf("InitManifest version = %q, want %q", manifest.Version, expected)
	}
}

func TestLoadManifest_DefaultUsesGeneratorVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load without init — should return defaults with generator version
	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	expected := GetGenerator()
	if manifest.Version != expected {
		t.Errorf("default manifest version = %q, want %q", manifest.Version, expected)
	}
}

func TestMarkRead(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	payload, _ := json.Marshal(map[string]string{"message": "test"})
	id1, _ := mgr.Add("test_type", "source1", payload, "notif1")
	id2, _ := mgr.Add("test_type", "source2", payload, "notif2")
	mgr.Add("test_type", "source3", payload, "notif3")

	// Mark specific IDs
	marked, err := mgr.MarkRead([]string{id1, id2}, false)
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}
	if marked != 2 {
		t.Errorf("Expected 2 marked, got %d", marked)
	}

	// Verify unread count
	unread, _ := mgr.CountUnread()
	if unread != 1 {
		t.Errorf("Expected 1 unread, got %d", unread)
	}

	// Mark all
	marked, err = mgr.MarkRead(nil, true)
	if err != nil {
		t.Fatalf("MarkRead all failed: %v", err)
	}
	if marked != 1 {
		t.Errorf("Expected 1 marked, got %d", marked)
	}

	unread, _ = mgr.CountUnread()
	if unread != 0 {
		t.Errorf("Expected 0 unread, got %d", unread)
	}
}

func TestCountUnread(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	// Empty = 0 unread
	count, _ := mgr.CountUnread()
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}

	payload, _ := json.Marshal(map[string]string{"message": "test"})
	mgr.Add("test_type", "source", payload, "n1")
	mgr.Add("test_type", "source", payload, "n2")

	count, _ = mgr.CountUnread()
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}

	mgr.MarkRead([]string{"n1"}, false)
	count, _ = mgr.CountUnread()
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestListPaginated(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	payload, _ := json.Marshal(map[string]string{"message": "test"})
	mgr.Add("test_type", "source", payload, "n1")
	mgr.Add("test_type", "source", payload, "n2")
	mgr.Add("test_type", "source", payload, "n3")
	mgr.Add("test_type", "source", payload, "n4")
	mgr.Add("test_type", "source", payload, "n5")

	// Mark some as read
	mgr.MarkRead([]string{"n1", "n2"}, false)

	// Unread only, first page
	items, total, err := mgr.ListPaginated(0, 2, false)
	if err != nil {
		t.Fatalf("ListPaginated failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected total 3 unread, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	// Newest first — n5 should be first
	if items[0].ID != "n5" {
		t.Errorf("Expected n5 first, got %s", items[0].ID)
	}

	// Second page
	items, _, _ = mgr.ListPaginated(2, 2, false)
	if len(items) != 1 {
		t.Errorf("Expected 1 item on second page, got %d", len(items))
	}

	// Include read
	items, total, _ = mgr.ListPaginated(0, 10, true)
	if total != 5 {
		t.Errorf("Expected total 5 with read, got %d", total)
	}
	if len(items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(items))
	}

	// Offset beyond range
	items, _, _ = mgr.ListPaginated(100, 10, true)
	if len(items) != 0 {
		t.Errorf("Expected 0 items for large offset, got %d", len(items))
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultNotificationsFile("/data")
	expected := filepath.Join("/data", ".polis", "notifications.jsonl")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}

	manifestPath := DefaultManifestFile("/data")
	expectedManifest := filepath.Join("/data", ".polis", "notifications-manifest.json")
	if manifestPath != expectedManifest {
		t.Errorf("Expected %s, got %s", expectedManifest, manifestPath)
	}
}
