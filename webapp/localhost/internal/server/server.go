// Package server provides the HTTP server implementation for the Polis webapp.
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/comment"
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/feed"
	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/hooks"
	"github.com/vdibart/polis-cli/cli-go/pkg/metadata"
	"github.com/vdibart/polis-cli/cli-go/pkg/notification"
	"github.com/vdibart/polis-cli/cli-go/pkg/publish"
	"github.com/vdibart/polis-cli/cli-go/pkg/render"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
	"github.com/vdibart/polis-cli/cli-go/pkg/stream"
)

// DefaultDiscoveryServiceURL is the default discovery service URL matching the CLI
const DefaultDiscoveryServiceURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"

// DefaultDiscoveryServiceKey is the public anon key for the default discovery service.
// This is intentionally public — it's the Supabase anon key used for unauthenticated
// access to edge functions. All authorization is handled via Ed25519 signatures.
const DefaultDiscoveryServiceKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Imx0ZnBlenJpaWFxdmp1cHhidHR3Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3NjcxNDQwODMsImV4cCI6MjA4MjcyMDA4M30.N9ScKbdcswutM6i__W9sPWWcBONIcxdAqIbsljqMKMI"

// Log levels
const (
	LogLevelOff     = 0 // No logging
	LogLevelBasic   = 1 // Basic logging (errors, warnings, important info)
	LogLevelVerbose = 2 // Verbose logging (all operations)
)

// Config holds the application configuration
// Note: SetupCode and Subdomain are deprecated but still parsed for backwards compatibility
type Config struct {
	SetupCode string `json:"setup_code,omitempty"` // Deprecated: ignored
	Subdomain string `json:"subdomain,omitempty"`  // Deprecated: derive from .well-known/polis
	SetupAt   string `json:"setup_at,omitempty"`   // Deprecated: derive from .well-known/polis

	// Hooks configuration
	Hooks *hooks.HookConfig `json:"hooks,omitempty"`

	// View mode: "list" or "browser"
	ViewMode string `json:"view_mode,omitempty"`

	// Show frontmatter in markdown pane (default true)
	ShowFrontmatter *bool `json:"show_frontmatter,omitempty"`

	// Logging level: 0=off, 1=basic, 2=verbose
	LogLevel int `json:"log_level,omitempty"`

	// Setup wizard dismissed state (false = show wizard after init)
	SetupWizardDismissed bool `json:"setup_wizard_dismissed,omitempty"`
}

// Server holds the application state
type Server struct {
	DataDir      string
	CLIThemesDir string // Path to CLI themes directory (fallback for theme snippets)
	CLIVersion   string // CLI version for metadata files (set by bundled binary or from version.txt)
	Config       *Config
	PrivateKey   []byte
	PublicKey    []byte
	Logger       *Logger
	BaseURL      string // From POLIS_BASE_URL env var (runtime config, not stored in .well-known/polis)
	DiscoveryURL string // From .env / env var DISCOVERY_SERVICE_URL (not stored in webapp-config.json)
	DiscoveryKey string // From .env / env var DISCOVERY_SERVICE_KEY (not stored in webapp-config.json)
}

// Logger handles logging to files organized by date
type Logger struct {
	level   int
	logsDir string
	file    *os.File
	mu      sync.Mutex
}

// NewLogger creates a new logger with the given level and logs directory
func NewLogger(level int, logsDir string) *Logger {
	return &Logger{
		level:   level,
		logsDir: logsDir,
	}
}

// getLogFile returns the log file for today, creating it if necessary
func (l *Logger) getLogFile() (*os.File, error) {
	if l.level == LogLevelOff {
		return nil, nil
	}

	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(l.logsDir, today+".log")

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(l.logsDir, 0755); err != nil {
		return nil, err
	}

	// Check if we need to open a new file (date changed)
	if l.file != nil {
		// Check if the current file is for today
		info, err := l.file.Stat()
		if err == nil && strings.HasPrefix(info.Name(), today) {
			return l.file, nil
		}
		// Close old file
		l.file.Close()
	}

	// Open new log file
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	l.file = file
	return file, nil
}

// log writes a message to the log file
func (l *Logger) log(level int, prefix string, format string, args ...interface{}) {
	if l == nil || l.level < level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := l.getLogFile()
	if err != nil || file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(file, "%s [%s] %s\n", timestamp, prefix, message)
}

// Info logs informational messages (level 1)
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelBasic, "INFO", format, args...)
}

// Warn logs warning messages (level 1)
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelBasic, "WARN", format, args...)
}

// Error logs error messages (level 1)
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelBasic, "ERROR", format, args...)
}

// Debug logs debug messages (level 2)
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelVerbose, "DEBUG", format, args...)
}

// Close closes the log file
func (l *Logger) Close() {
	if l != nil && l.file != nil {
		l.file.Close()
	}
}

// Server logging helpers
func (s *Server) LogInfo(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Info(format, args...)
	}
}

func (s *Server) LogWarn(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Warn(format, args...)
	}
}

func (s *Server) LogError(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Error(format, args...)
	}
}

func (s *Server) LogDebug(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Debug(format, args...)
	}
}

// GetBaseURL returns the site's base URL from POLIS_BASE_URL environment variable.
// This matches the bash CLI behavior - base_url is runtime config, not stored in .well-known/polis.
func (s *Server) GetBaseURL() string {
	// Return cached value from LoadEnv()
	return s.BaseURL
}

// RenderSite renders all pages after publish/republish operations.
// This ensures HTML files are updated and hooks can act on the complete output.
func (s *Server) RenderSite() error {
	// Get site base URL from POLIS_BASE_URL env var (matches bash CLI behavior)
	baseURL := s.GetBaseURL()

	// Create page renderer
	renderer, err := render.NewPageRenderer(render.PageConfig{
		DataDir:       s.DataDir,
		CLIThemesDir:  s.CLIThemesDir,
		BaseURL:       baseURL,
		RenderMarkers: false, // No markers needed for publish flow
	})
	if err != nil {
		s.LogError("Failed to create renderer: %v", err)
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	// Render all pages
	stats, err := renderer.RenderAll(true)
	if err != nil {
		s.LogError("Render failed: %v", err)
		return fmt.Errorf("render failed: %w", err)
	}

	s.LogInfo("Rendered site: %d posts, %d comments", stats.PostsRendered, stats.CommentsRendered)
	return nil
}

// LoadConfig loads the webapp configuration from webapp-config.json
func (s *Server) LoadConfig() {
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return // Config doesn't exist yet
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}
	s.Config = &config
}

// SaveConfig saves the webapp configuration to webapp-config.json
func (s *Server) SaveConfig() error {
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	// Clear deprecated fields before saving (don't persist them)
	savedSubdomain := s.Config.Subdomain
	s.Config.Subdomain = ""
	data, err := json.MarshalIndent(s.Config, "", "  ")
	s.Config.Subdomain = savedSubdomain // Restore in memory for runtime use
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// LoadKeys loads the private and public keys from the keys directory
func (s *Server) LoadKeys() {
	privPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	pubPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")

	priv, err := os.ReadFile(privPath)
	if err != nil {
		return
	}
	pub, err := os.ReadFile(pubPath)
	if err != nil {
		return
	}
	s.PrivateKey = priv
	s.PublicKey = pub
}

// LoadEnv reads the .env file and applies discovery service settings.
// This matches the bash CLI behavior where DISCOVERY_SERVICE_URL and DISCOVERY_SERVICE_KEY
// are read from .env file rather than being stored in config.json.
//
// Search order:
// 1. Data directory .env (where the polis site data lives)
// 2. Current working directory .env (user's polis site)
// 3. ~/.polis/.env (fallback for multi-site setups)
func (s *Server) LoadEnv() {
	var envPath string
	var data []byte
	var err error

	// First, try the data directory (where webapp-config.json and keys live)
	envPath = filepath.Join(s.DataDir, ".env")
	data, err = os.ReadFile(envPath)
	if err == nil {
		log.Printf("[i] Loaded .env from data directory: %s", envPath)
	}

	// If not found, try current working directory
	if err != nil {
		cwd, cwdErr := os.Getwd()
		if cwdErr == nil {
			envPath = filepath.Join(cwd, ".env")
			data, err = os.ReadFile(envPath)
			if err == nil {
				log.Printf("[i] Loaded .env from current directory: %s", envPath)
			}
		}
	}

	// If still not found, try ~/.polis/.env as fallback
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr == nil {
			envPath = filepath.Join(homeDir, ".polis", ".env")
			data, err = os.ReadFile(envPath)
			if err == nil {
				log.Printf("[i] Loaded .env from fallback: %s", envPath)
			}
		}
	}

	// If still not found, that's fine - just return
	if err != nil {
		return
	}

	// Parse .env file (simple KEY=VALUE format, one per line)
	env := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE (handle quoted values)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove surrounding quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		env[key] = value
	}

	// Apply discovery service settings from .env (single source of truth, like CLI)
	if url := env["DISCOVERY_SERVICE_URL"]; url != "" {
		s.DiscoveryURL = url
	}
	if key := env["DISCOVERY_SERVICE_KEY"]; key != "" {
		s.DiscoveryKey = key
	}

	// Store POLIS_BASE_URL for runtime use (matches bash CLI behavior)
	// This is the authoritative source for base_url - not stored in .well-known/polis
	if baseURL := env["POLIS_BASE_URL"]; baseURL != "" {
		s.BaseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// ApplyDiscoveryDefaults sets default discovery service URL and key if not configured.
// This ensures the webapp works out of the box without requiring .env configuration.
func (s *Server) ApplyDiscoveryDefaults() {
	if s.DiscoveryURL == "" {
		s.DiscoveryURL = DefaultDiscoveryServiceURL
	}
	if s.DiscoveryKey == "" {
		s.DiscoveryKey = DefaultDiscoveryServiceKey
	}
}

// GetAuthorEmail returns the author email from .well-known/polis.
// This is the single source of truth for the site owner's email address.
func (s *Server) GetAuthorEmail() string {
	wk, err := site.LoadWellKnown(s.DataDir)
	if err != nil {
		return ""
	}
	return wk.Email
}

// GetSubdomain extracts subdomain from POLIS_BASE_URL env var
func (s *Server) GetSubdomain() string {
	// Get from POLIS_BASE_URL env var (authoritative source)
	if baseURL := s.GetBaseURL(); baseURL != "" {
		// Extract subdomain from URL like https://alice.polis.pub
		host := strings.TrimPrefix(baseURL, "https://")
		host = strings.TrimPrefix(host, "http://")
		if idx := strings.Index(host, "."); idx > 0 {
			return host[:idx]
		}
	}
	// Fallback to deprecated config field
	if s.Config != nil {
		return s.Config.Subdomain
	}
	return ""
}

// GetSiteTitle returns site_title from .well-known/polis, falling back to POLIS_BASE_URL if empty
func (s *Server) GetSiteTitle() string {
	wk, err := site.LoadWellKnown(s.DataDir)
	if err != nil {
		// No .well-known/polis file - try config subdomain
		if s.Config != nil && s.Config.Subdomain != "" {
			return "https://" + s.Config.Subdomain + ".polis.pub"
		}
		return ""
	}
	// 1. Try site_title from file
	if wk.SiteTitle != "" {
		return wk.SiteTitle
	}
	// 2. Try base_url from file (for backwards compat with existing files)
	if wk.BaseURL != "" {
		return wk.BaseURL
	}
	// 3. Construct from subdomain
	if wk.Subdomain != "" {
		return "https://" + wk.Subdomain + ".polis.pub"
	}
	// 4. Fall back to POLIS_BASE_URL env var
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return ""
}

// ResolveSymlink follows symlinks to get the real path.
func ResolveSymlink(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Path doesn't exist yet or other error - return original
		return path
	}
	return resolved
}

// FindAvailablePort finds an available port on localhost.
func FindAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

// OpenBrowser opens the default browser to the given URL.
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Printf("[i] Please open %s in your browser\n", url)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("[i] Please open %s in your browser\n", url)
	}
}

// FindCLIThemesDir locates the cli/themes directory for fallback theme snippets.
// It searches upward from the given directory to find the repo root.
func FindCLIThemesDir(startDir string) string {
	// Start from startDir and search upward
	dir := startDir
	for i := 0; i < 10; i++ { // Max 10 levels up
		themesPath := filepath.Join(dir, "cli-bash", "themes")
		if info, err := os.Stat(themesPath); err == nil && info.IsDir() {
			return themesPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	// Fallback: try current working directory
	cwd, err := os.Getwd()
	if err == nil {
		dir = cwd
		for i := 0; i < 10; i++ {
			themesPath := filepath.Join(dir, "cli-bash", "themes")
			if info, err := os.Stat(themesPath); err == nil && info.IsDir() {
				return themesPath
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Return empty if not found (will just use local themes only)
	return ""
}

// NewServer creates and initializes a new Server instance.
func NewServer(dataDir, cliThemesDir string) *Server {
	return &Server{
		DataDir:      dataDir,
		CLIThemesDir: cliThemesDir,
	}
}

// Initialize validates the site and loads configuration.
func (s *Server) Initialize() {
	// Propagate CLI version to packages that embed it in metadata
	if s.CLIVersion != "" {
		publish.Version = s.CLIVersion
		comment.Version = s.CLIVersion
		metadata.Version = s.CLIVersion
		following.Version = s.CLIVersion
		feed.Version = s.CLIVersion
		site.Version = s.CLIVersion
	}

	// Migrate .polis/drafts -> .polis/posts/drafts if needed
	s.migrateDraftsDir()

	// Validate the site first - only load keys/config if valid
	validation := site.Validate(s.DataDir)
	if validation.Status == site.StatusValid {
		// Check for unrecognized/deprecated fields (informational logging)
		site.CheckUnrecognizedFields(s.DataDir)

		// Try to upgrade .well-known/polis if needed (adds missing canonical fields)
		if _, err := site.UpgradeWellKnown(s.DataDir); err != nil {
			log.Printf("[warning] Failed to upgrade .well-known/polis: %v", err)
		}

		// Load existing config if present
		s.LoadConfig()
		s.LoadKeys()
	}

	// Load .env file for discovery service settings (overrides webapp-config.json)
	s.LoadEnv()

	// Apply default discovery URL if not set by config or .env (matches CLI behavior)
	s.ApplyDiscoveryDefaults()

	// Propagate discovery config to packages that register with discovery.
	// This ensures publish and comment packages handle registration internally.
	publish.DiscoveryURL = s.DiscoveryURL
	publish.DiscoveryKey = s.DiscoveryKey
	publish.BaseURL = s.BaseURL
	comment.DiscoveryURL = s.DiscoveryURL
	comment.DiscoveryKey = s.DiscoveryKey
	comment.BaseURL = s.BaseURL
	stream.DiscoveryURL = s.DiscoveryURL
	stream.DiscoveryKey = s.DiscoveryKey
	stream.BaseURL = s.BaseURL

	// Initialize logger if log level is configured
	logLevel := 0
	if s.Config != nil {
		logLevel = s.Config.LogLevel
	}
	if logLevel > 0 {
		logsDir := filepath.Join(s.DataDir, "logs")
		s.Logger = NewLogger(logLevel, logsDir)
		s.Logger.Info("Server starting with log level %d", logLevel)
		s.Logger.Info("Data directory: %s", s.DataDir)
	}
}

// migrateDraftsDir migrates .polis/drafts to .polis/posts/drafts if needed.
func (s *Server) migrateDraftsDir() {
	oldPath := filepath.Join(s.DataDir, ".polis", "drafts")
	newPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts")

	// Only migrate if old path exists and new path doesn't
	oldInfo, oldErr := os.Stat(oldPath)
	_, newErr := os.Stat(newPath)

	if oldErr == nil && oldInfo.IsDir() && os.IsNotExist(newErr) {
		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
			log.Printf("[warning] Failed to create parent directory for drafts migration: %v", err)
			return
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			log.Printf("[warning] Failed to migrate drafts directory: %v", err)
		} else {
			log.Printf("[i] Migrated drafts: .polis/drafts -> .polis/posts/drafts")
		}
	}
}

// Close cleans up server resources.
func (s *Server) Close() {
	if s.Logger != nil {
		s.Logger.Close()
	}
}

// StartBackgroundSync starts background goroutines that periodically
// sync notifications and feed from the discovery service.
func (s *Server) StartBackgroundSync() {
	go func() {
		s.syncNotifications()
		s.syncFeed()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.syncNotifications()
			s.syncFeed()
		}
	}()
}

// syncNotifications runs the notification projection: queries the stream
// with separate queries per relevance group, applies rules, and appends
// new entries to state.jsonl.
func (s *Server) syncNotifications() {
	if s.DiscoveryURL == "" || s.DiscoveryKey == "" {
		return
	}
	baseURL := s.GetBaseURL()
	if baseURL == "" || s.PrivateKey == nil {
		return
	}

	myDomain := extractDomainFromURL(baseURL)
	if myDomain == "" {
		return
	}

	discoveryDomain := s.GetDiscoveryDomain()
	store := stream.NewStore(s.DataDir, discoveryDomain)

	// Load notification config (rules + muted domains)
	var config stream.NotificationConfig
	_ = store.LoadConfig("notifications", &config)

	// Seed rules from defaults if empty (first sync)
	rules := config.Rules
	if len(rules) == 0 {
		rules = notification.DefaultRules()
		config.Rules = rules
		_ = store.SaveConfig("notifications", &config)
	}

	// Build muted domains set
	mutedDomains := make(map[string]bool, len(config.MutedDomains))
	for _, d := range config.MutedDomains {
		mutedDomains[d] = true
	}

	handler := &stream.NotificationHandler{
		MyDomain:     myDomain,
		Rules:        rules,
		MutedDomains: mutedDomains,
	}

	// Get shared cursor
	cursor, _ := store.GetCursor("polis.notification")

	client := discovery.NewClient(s.DiscoveryURL, s.DiscoveryKey)

	// Group rules by relevance for targeted server-side filtering
	groups := handler.RulesByRelevance()
	var allEntries []notification.StateEntry
	newCursor := cursor

	// Query 1: target_domain rules
	if targetRules := groups["target_domain"]; len(targetRules) > 0 {
		var types []string
		for _, r := range targetRules {
			types = append(types, r.EventType)
		}
		typeFilter := discovery.JoinDomains(types)
		result, err := client.StreamQuery(cursor, 1000, typeFilter, "", myDomain)
		if err != nil {
			s.LogDebug("notification sync: target_domain query failed: %v", err)
		} else {
			entries := handler.Process(result.Events)
			allEntries = append(allEntries, entries...)
			if result.Cursor > newCursor {
				newCursor = result.Cursor
			}
		}
	}

	// Query 2: source_domain rules
	if sourceRules := groups["source_domain"]; len(sourceRules) > 0 {
		var types []string
		for _, r := range sourceRules {
			types = append(types, r.EventType)
		}
		typeFilter := discovery.JoinDomains(types)
		result, err := client.StreamQuery(cursor, 1000, typeFilter, "", "", myDomain)
		if err != nil {
			s.LogDebug("notification sync: source_domain query failed: %v", err)
		} else {
			entries := handler.Process(result.Events)
			allEntries = append(allEntries, entries...)
			if result.Cursor > newCursor {
				newCursor = result.Cursor
			}
		}
	}

	// Query 3: followed_author rules (only if enabled)
	if authorRules := groups["followed_author"]; len(authorRules) > 0 {
		// Load following list
		followingPath := following.DefaultPath(s.DataDir)
		f, err := following.Load(followingPath)
		if err == nil {
			var domains []string
			for _, entry := range f.All() {
				d := discovery.ExtractDomainFromURL(entry.URL)
				if d != "" {
					domains = append(domains, d)
				}
			}
			if len(domains) > 0 {
				var types []string
				for _, r := range authorRules {
					types = append(types, r.EventType)
				}
				typeFilter := discovery.JoinDomains(types)
				actorFilter := discovery.JoinDomains(domains)
				result, err := client.StreamQuery(cursor, 1000, typeFilter, actorFilter, "")
				if err != nil {
					s.LogDebug("notification sync: followed_author query failed: %v", err)
				} else {
					entries := handler.Process(result.Events)
					allEntries = append(allEntries, entries...)
					if result.Cursor > newCursor {
						newCursor = result.Cursor
					}
				}
			}
		}
	}

	// Append new entries to state.jsonl
	if len(allEntries) > 0 {
		mgr := notification.NewManager(s.DataDir, discoveryDomain)
		added, err := mgr.Append(allEntries)
		if err != nil {
			s.LogError("notification sync: failed to append entries: %v", err)
		} else if added > 0 {
			s.LogInfo("notification sync: added %d new notifications", added)
		}
	}

	// Update cursor
	if newCursor != cursor {
		_ = store.SetCursor("polis.notification", newCursor)
	}
}

// syncFeed queries the discovery stream for post and comment events
// from followed authors and merges them into the feed cache.
func (s *Server) syncFeed() {
	if s.DiscoveryURL == "" || s.DiscoveryKey == "" {
		return
	}
	baseURL := s.GetBaseURL()
	if baseURL == "" {
		return
	}

	myDomain := extractDomainFromURL(baseURL)
	if myDomain == "" {
		return
	}

	discoveryDomain := s.GetDiscoveryDomain()

	// Load following list to get followed domains
	followingPath := following.DefaultPath(s.DataDir)
	f, err := following.Load(followingPath)
	if err != nil || f.Count() == 0 {
		return
	}

	var domains []string
	for _, entry := range f.All() {
		d := discovery.ExtractDomainFromURL(entry.URL)
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		return
	}

	// Load feed cursor
	cm := feed.NewCacheManager(s.DataDir, discoveryDomain)
	cursor, _ := cm.GetCursor()

	// Query DS stream with actor filter for followed domains
	client := discovery.NewClient(s.DiscoveryURL, s.DiscoveryKey)
	typeFilter := "polis.post.published,polis.post.republished,polis.comment.published,polis.comment.republished"
	actorFilter := discovery.JoinDomains(domains)

	result, err := client.StreamQuery(cursor, 1000, typeFilter, actorFilter, "")
	if err != nil {
		s.LogDebug("feed sync: stream query failed: %v", err)
		return
	}

	// Transform events to feed items
	handler := &feed.FeedHandler{
		MyDomain:        myDomain,
		FollowedDomains: make(map[string]bool, len(domains)),
	}
	for _, d := range domains {
		handler.FollowedDomains[d] = true
	}

	items := handler.Process(result.Events)

	// Merge into cache
	if len(items) > 0 {
		newCount, err := cm.MergeItems(items)
		if err != nil {
			s.LogError("feed sync: merge failed: %v", err)
		} else if newCount > 0 {
			s.LogInfo("feed sync: added %d new items", newCount)
		}
	}

	// Update cursor
	if result.Cursor != "" && result.Cursor != cursor {
		_ = cm.SetCursor(result.Cursor)
	}
}

// RunOptions contains optional configuration for the server.
type RunOptions struct {
	CLIVersion string // CLI version for metadata (empty = use package default)
}

// Run starts the HTTP server with the given embedded filesystem.
func Run(webFS fs.FS, dataDir string, opts ...RunOptions) {
	// Resolve symlinks - if data/ is a symlink, follow it
	dataDir = ResolveSymlink(dataDir)

	// DON'T auto-create directories on startup - let the user choose init vs link
	// We only create the parent data dir if it doesn't exist (needed for symlink target)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		// Create just the data directory (not the full structure)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("[warning] Failed to create data directory: %v", err)
		}
	}

	// Find executable directory for CLI themes
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	execDir := filepath.Dir(execPath)

	// Find CLI themes directory (for fallback theme snippets)
	cliThemesDir := FindCLIThemesDir(execDir)

	// Initialize server
	server := NewServer(dataDir, cliThemesDir)
	if len(opts) > 0 && opts[0].CLIVersion != "" {
		server.CLIVersion = opts[0].CLIVersion
	}
	server.Initialize()
	defer server.Close()

	// Start background sync (notifications + feed)
	server.StartBackgroundSync()

	// Find available port
	port, err := FindAvailablePort()
	if err != nil {
		log.Fatal("Failed to find available port:", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	SetupRoutes(mux, server)

	// Static files from embedded filesystem
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	addr := fmt.Sprintf("localhost:%d", port)
	url := fmt.Sprintf("http://%s", addr)

	fmt.Printf("[i] Starting polis server...\n")
	fmt.Printf("[i] Listening on %s\n", url)
	fmt.Printf("[i] Data directory: %s\n", dataDir)

	// Open browser after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		OpenBrowser(url)
	}()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Server error:", err)
	}
}

// GetDiscoveryDomain returns the discovery service hostname for use as
// the namespace key in .polis/ds/<domain>/.
func (s *Server) GetDiscoveryDomain() string {
	domain := extractDomainFromURL(s.DiscoveryURL)
	if domain == "" {
		return "default"
	}
	return domain
}

// extractDomainFromURL extracts the hostname from a URL string.
func extractDomainFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
