// Package server provides the HTTP server implementation for the Polis webapp.
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/hooks"
	"github.com/vdibart/polis-cli/cli-go/pkg/render"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

// DefaultDiscoveryServiceURL is the default discovery service URL matching the CLI
const DefaultDiscoveryServiceURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"

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

	// Discovery service integration
	DiscoveryURL string `json:"discovery_url,omitempty"`
	DiscoveryKey string `json:"discovery_key,omitempty"`
	AuthorEmail  string `json:"author_email,omitempty"`

	// Hooks configuration
	Hooks *hooks.HookConfig `json:"hooks,omitempty"`

	// View mode: "list" or "browser"
	ViewMode string `json:"view_mode,omitempty"`

	// Show frontmatter in markdown pane (default true)
	ShowFrontmatter *bool `json:"show_frontmatter,omitempty"`

	// Logging level: 0=off, 1=basic, 2=verbose
	LogLevel int `json:"log_level,omitempty"`
}

// Server holds the application state
type Server struct {
	DataDir      string
	CLIThemesDir string // Path to CLI themes directory (fallback for theme snippets)
	Config       *Config
	PrivateKey   []byte
	PublicKey    []byte
	Logger       *Logger
	BaseURL      string // From POLIS_BASE_URL env var (runtime config, not stored in .well-known/polis)
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
	data, err := json.MarshalIndent(s.Config, "", "  ")
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

	// Apply discovery service settings from .env
	// These override any values in webapp-config.json (env takes precedence)
	if url := env["DISCOVERY_SERVICE_URL"]; url != "" {
		if s.Config == nil {
			s.Config = &Config{}
		}
		s.Config.DiscoveryURL = url
	}
	if key := env["DISCOVERY_SERVICE_KEY"]; key != "" {
		if s.Config == nil {
			s.Config = &Config{}
		}
		s.Config.DiscoveryKey = key
	}

	// Store POLIS_BASE_URL for runtime use (matches bash CLI behavior)
	// This is the authoritative source for base_url - not stored in .well-known/polis
	if baseURL := env["POLIS_BASE_URL"]; baseURL != "" {
		s.BaseURL = strings.TrimSuffix(baseURL, "/")

		// Also derive subdomain if not set (for backwards compatibility)
		if s.Config != nil && s.Config.Subdomain == "" {
			host := strings.TrimPrefix(baseURL, "https://")
			host = strings.TrimPrefix(host, "http://")
			if idx := strings.Index(host, "."); idx > 0 {
				s.Config.Subdomain = host[:idx]
			}
		}
	}
}

// ApplyDiscoveryDefaults sets default discovery service URL if not configured.
// This ensures the webapp works out of the box without requiring .env configuration.
func (s *Server) ApplyDiscoveryDefaults() {
	if s.Config == nil {
		s.Config = &Config{}
	}
	if s.Config.DiscoveryURL == "" {
		s.Config.DiscoveryURL = DefaultDiscoveryServiceURL
	}
}

// WellKnownPolis represents the .well-known/polis file structure
type WellKnownPolis struct {
	Subdomain string `json:"subdomain"`
	BaseURL   string `json:"base_url"`
	SiteTitle string `json:"site_title,omitempty"`
	PublicKey string `json:"public_key"`
	Generator string `json:"generator"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
}

// LoadWellKnownPolis reads and parses .well-known/polis
func (s *Server) LoadWellKnownPolis() (*WellKnownPolis, error) {
	wellKnownPath := filepath.Join(s.DataDir, ".well-known", "polis")
	data, err := os.ReadFile(wellKnownPath)
	if err != nil {
		return nil, err
	}
	var wkp WellKnownPolis
	if err := json.Unmarshal(data, &wkp); err != nil {
		return nil, err
	}
	return &wkp, nil
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
	wkp, err := s.LoadWellKnownPolis()
	if err != nil {
		// No .well-known/polis file - try config subdomain
		if s.Config != nil && s.Config.Subdomain != "" {
			return "https://" + s.Config.Subdomain + ".polis.pub"
		}
		return ""
	}
	// 1. Try site_title from file
	if wkp.SiteTitle != "" {
		return wkp.SiteTitle
	}
	// 2. Try base_url from file (for backwards compat with existing files)
	if wkp.BaseURL != "" {
		return wkp.BaseURL
	}
	// 3. Construct from subdomain
	if wkp.Subdomain != "" {
		return "https://" + wkp.Subdomain + ".polis.pub"
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

// Close cleans up server resources.
func (s *Server) Close() {
	if s.Logger != nil {
		s.Logger.Close()
	}
}

// Run starts the HTTP server with the given embedded filesystem.
func Run(webFS fs.FS, dataDir string) {
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
	server.Initialize()
	defer server.Close()

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
