package notification

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Test InitManifest
	if err := mgr.InitManifest(); err != nil {
		t.Fatalf("InitManifest failed: %v", err)
	}

	// Test GetPreferences
	prefs, err := mgr.GetPreferences()
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	if prefs.PollIntervalMinutes != 60 {
		t.Errorf("Expected default poll interval 60, got %d", prefs.PollIntervalMinutes)
	}

	// Test SetPollInterval
	if err := mgr.SetPollInterval(30); err != nil {
		t.Fatalf("SetPollInterval failed: %v", err)
	}

	prefs, _ = mgr.GetPreferences()
	if prefs.PollIntervalMinutes != 30 {
		t.Errorf("Expected poll interval 30, got %d", prefs.PollIntervalMinutes)
	}

	// Test minimum poll interval
	if err := mgr.SetPollInterval(5); err != nil {
		t.Fatalf("SetPollInterval failed: %v", err)
	}

	prefs, _ = mgr.GetPreferences()
	if prefs.PollIntervalMinutes != 15 {
		t.Errorf("Expected minimum poll interval 15, got %d", prefs.PollIntervalMinutes)
	}
}

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

	// Test Remove
	if err := mgr.Remove(id); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	count, _ = mgr.Count()
	if count != 0 {
		t.Errorf("Expected count 0 after remove, got %d", count)
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

func TestMuteDomain(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	mgr.InitManifest()

	// Mute a domain
	if err := mgr.MuteDomain("example.com"); err != nil {
		t.Fatalf("MuteDomain failed: %v", err)
	}

	prefs, _ := mgr.GetPreferences()
	found := false
	for _, d := range prefs.MutedDomains {
		if d == "example.com" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected domain to be muted")
	}

	// Unmute the domain
	if err := mgr.UnmuteDomain("example.com"); err != nil {
		t.Fatalf("UnmuteDomain failed: %v", err)
	}

	prefs, _ = mgr.GetPreferences()
	for _, d := range prefs.MutedDomains {
		if d == "example.com" {
			t.Error("Expected domain to be unmuted")
		}
	}
}

func TestInitManifest_UsesPackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if err := mgr.InitManifest(); err != nil {
		t.Fatalf("InitManifest failed: %v", err)
	}

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if manifest.Version != Version {
		t.Errorf("InitManifest version = %q, want %q", manifest.Version, Version)
	}
}

func TestLoadManifest_DefaultUsesPackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load without init — should return defaults with package version
	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if manifest.Version != Version {
		t.Errorf("default manifest version = %q, want %q", manifest.Version, Version)
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
