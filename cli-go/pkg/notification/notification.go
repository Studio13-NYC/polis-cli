// Package notification manages local notifications and preferences.
package notification

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version is set at init time by cmd package.
var Version = "dev"

// Notification represents a single notification entry.
type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"created_at"`
	ReadAt    string          `json:"read_at,omitempty"`
}

// Manifest represents the notifications manifest file.
type Manifest struct {
	Version     string      `json:"version"`
	LastSync    string      `json:"last_sync"`
	Preferences Preferences `json:"preferences"`
}

// Preferences holds notification configuration.
type Preferences struct {
	PollIntervalMinutes int      `json:"poll_interval_minutes"`
	EnabledTypes        []string `json:"enabled_types"`
	MutedDomains        []string `json:"muted_domains"`
}

// DefaultNotificationsFile returns the default path to notifications.jsonl.
func DefaultNotificationsFile(dataDir string) string {
	return filepath.Join(dataDir, ".polis", "notifications.jsonl")
}

// DefaultManifestFile returns the default path to notifications-manifest.json.
func DefaultManifestFile(dataDir string) string {
	return filepath.Join(dataDir, ".polis", "notifications-manifest.json")
}

// Manager handles notification operations.
type Manager struct {
	notificationsFile string
	manifestFile      string
}

// NewManager creates a new notification manager.
func NewManager(dataDir string) *Manager {
	return &Manager{
		notificationsFile: DefaultNotificationsFile(dataDir),
		manifestFile:      DefaultManifestFile(dataDir),
	}
}

// InitManifest ensures the manifest file exists with defaults.
func (m *Manager) InitManifest() error {
	if _, err := os.Stat(m.manifestFile); err == nil {
		return nil // Already exists
	}

	manifest := Manifest{
		Version:  Version,
		LastSync: "1970-01-01T00:00:00Z",
		Preferences: Preferences{
			PollIntervalMinutes: 60,
			EnabledTypes:        []string{"version_available", "blessing_request", "comment_blessed", "domain_migration"},
			MutedDomains:        []string{},
		},
	}

	return m.saveManifest(&manifest)
}

// LoadManifest loads the notification manifest.
func (m *Manager) LoadManifest() (*Manifest, error) {
	data, err := os.ReadFile(m.manifestFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults
			return &Manifest{
				Version:  Version,
				LastSync: "1970-01-01T00:00:00Z",
				Preferences: Preferences{
					PollIntervalMinutes: 60,
					EnabledTypes:        []string{"version_available", "blessing_request", "comment_blessed", "domain_migration"},
					MutedDomains:        []string{},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// saveManifest saves the manifest to disk.
func (m *Manager) saveManifest(manifest *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(m.manifestFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(m.manifestFile, append(data, '\n'), 0644)
}

// GetPreferences returns the notification preferences.
func (m *Manager) GetPreferences() (*Preferences, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}
	return &manifest.Preferences, nil
}

// SetPollInterval sets the poll interval in minutes.
func (m *Manager) SetPollInterval(minutes int) error {
	if minutes < 15 {
		minutes = 15
	}

	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	manifest.Preferences.PollIntervalMinutes = minutes
	return m.saveManifest(manifest)
}

// EnableType enables a notification type.
func (m *Manager) EnableType(notifType string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	// Check if already enabled
	for _, t := range manifest.Preferences.EnabledTypes {
		if t == notifType {
			return nil
		}
	}

	manifest.Preferences.EnabledTypes = append(manifest.Preferences.EnabledTypes, notifType)
	return m.saveManifest(manifest)
}

// DisableType disables a notification type.
func (m *Manager) DisableType(notifType string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	var newTypes []string
	for _, t := range manifest.Preferences.EnabledTypes {
		if t != notifType {
			newTypes = append(newTypes, t)
		}
	}

	manifest.Preferences.EnabledTypes = newTypes
	return m.saveManifest(manifest)
}

// IsTypeEnabled checks if a notification type is enabled.
func (m *Manager) IsTypeEnabled(notifType string) (bool, error) {
	prefs, err := m.GetPreferences()
	if err != nil {
		return false, err
	}

	for _, t := range prefs.EnabledTypes {
		if t == notifType {
			return true, nil
		}
	}
	return false, nil
}

// MuteDomain mutes notifications from a domain.
func (m *Manager) MuteDomain(domain string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	// Check if already muted
	for _, d := range manifest.Preferences.MutedDomains {
		if d == domain {
			return nil
		}
	}

	manifest.Preferences.MutedDomains = append(manifest.Preferences.MutedDomains, domain)
	return m.saveManifest(manifest)
}

// UnmuteDomain unmutes a domain.
func (m *Manager) UnmuteDomain(domain string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	var newDomains []string
	for _, d := range manifest.Preferences.MutedDomains {
		if d != domain {
			newDomains = append(newDomains, d)
		}
	}

	manifest.Preferences.MutedDomains = newDomains
	return m.saveManifest(manifest)
}

// GetWatermark returns the last sync timestamp.
func (m *Manager) GetWatermark() (string, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return "1970-01-01T00:00:00Z", nil
	}
	return manifest.LastSync, nil
}

// UpdateWatermark updates the last sync timestamp.
func (m *Manager) UpdateWatermark(timestamp string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	manifest.LastSync = timestamp
	return m.saveManifest(manifest)
}

// List returns all notifications.
func (m *Manager) List() ([]Notification, error) {
	file, err := os.Open(m.notificationsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Notification{}, nil
		}
		return nil, fmt.Errorf("failed to open notifications file: %w", err)
	}
	defer file.Close()

	var notifications []Notification
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var n Notification
		if err := json.Unmarshal(scanner.Bytes(), &n); err != nil {
			continue // Skip malformed lines
		}
		notifications = append(notifications, n)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read notifications: %w", err)
	}

	return notifications, nil
}

// ListByType returns notifications filtered by type.
func (m *Manager) ListByType(types []string) ([]Notification, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}

	if len(types) == 0 {
		return all, nil
	}

	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	var filtered []Notification
	for _, n := range all {
		if typeSet[n.Type] {
			filtered = append(filtered, n)
		}
	}

	return filtered, nil
}

// Count returns the number of notifications.
func (m *Manager) Count() (int, error) {
	notifications, err := m.List()
	if err != nil {
		return 0, err
	}
	return len(notifications), nil
}

// Add adds a new notification.
func (m *Manager) Add(notifType, source string, payload json.RawMessage, dedupeKey string) (string, error) {
	// Check for duplicate if dedupeKey provided
	if dedupeKey != "" {
		existing, _ := m.List()
		for _, n := range existing {
			if n.ID == dedupeKey {
				return "", nil // Already exists
			}
		}
	}

	// Generate ID
	id := dedupeKey
	if id == "" {
		id = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	notification := Notification{
		ID:        id,
		Type:      notifType,
		Source:    source,
		Payload:   payload,
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.notificationsFile), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Append to file
	file, err := os.OpenFile(m.notificationsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open notifications file: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(notification)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notification: %w", err)
	}

	if _, err := file.WriteString(string(data) + "\n"); err != nil {
		return "", fmt.Errorf("failed to write notification: %w", err)
	}

	return id, nil
}

// Remove removes a notification by ID.
func (m *Manager) Remove(id string) error {
	notifications, err := m.List()
	if err != nil {
		return err
	}

	var remaining []Notification
	for _, n := range notifications {
		if n.ID != id {
			remaining = append(remaining, n)
		}
	}

	return m.writeAll(remaining)
}

// RemoveAll removes all notifications.
func (m *Manager) RemoveAll() error {
	return m.writeAll([]Notification{})
}

// RemoveOlderThan removes notifications older than the given number of days.
func (m *Manager) RemoveOlderThan(days int) (int, error) {
	notifications, err := m.List()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var remaining []Notification
	removed := 0

	for _, n := range notifications {
		t, err := time.Parse("2006-01-02T15:04:05Z", n.CreatedAt)
		if err != nil || t.After(cutoff) {
			remaining = append(remaining, n)
		} else {
			removed++
		}
	}

	if err := m.writeAll(remaining); err != nil {
		return 0, err
	}

	return removed, nil
}

// writeAll rewrites all notifications to the file.
func (m *Manager) writeAll(notifications []Notification) error {
	if err := os.MkdirAll(filepath.Dir(m.notificationsFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(m.notificationsFile)
	if err != nil {
		return fmt.Errorf("failed to create notifications file: %w", err)
	}
	defer file.Close()

	for _, n := range notifications {
		data, err := json.Marshal(n)
		if err != nil {
			continue
		}
		file.WriteString(string(data) + "\n")
	}

	return nil
}
