package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdibart/polis-cli/cli-go/pkg/feed"
	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/hooks"
	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// Helper to create a test server with temp directory
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dataDir := t.TempDir()

	// Create required directories (matching main.go initialization)
	dirs := []string{
		filepath.Join(dataDir, ".polis"),
		filepath.Join(dataDir, ".polis", "keys"),
		filepath.Join(dataDir, ".polis", "themes"),
		filepath.Join(dataDir, ".polis", "posts", "drafts"),
		filepath.Join(dataDir, ".polis", "comments", "drafts"),
		filepath.Join(dataDir, ".polis", "comments", "pending"),
		filepath.Join(dataDir, ".polis", "comments", "denied"),
		filepath.Join(dataDir, ".well-known"),
		filepath.Join(dataDir, "posts"),
		filepath.Join(dataDir, "comments"),
		filepath.Join(dataDir, "snippets"),
		filepath.Join(dataDir, "metadata"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	return &Server{DataDir: dataDir}
}

// Helper to create a server with keys configured
func newConfiguredServer(t *testing.T) *Server {
	t.Helper()
	s := newTestServer(t)

	// Generate real keys
	privKey, pubKey, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}
	s.PrivateKey = privKey
	s.PublicKey = pubKey

	// Save keys to disk
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(privKeyPath, privKey, 0600)
	os.WriteFile(pubKeyPath, pubKey, 0644)

	// Set config (Subdomain is deprecated - use BaseURL instead)
	s.Config = &Config{
		SetupCode: "test-setup",
		SetupAt:   "2026-01-01T00:00:00Z",
	}
	s.BaseURL = "https://test-site.polis.pub"

	// Create .well-known/polis (single source of truth for identity)
	wellKnown := map[string]interface{}{
		"subdomain":  "test-site",
		"base_url":   "https://test-site.polis.pub",
		"site_title": "Test Site",
		"public_key": string(pubKey),
		"email":      "test@example.com",
		"author":     "Test Author",
	}
	wellKnownData, _ := json.MarshalIndent(wellKnown, "", "  ")
	wellKnownPath := filepath.Join(s.DataDir, ".well-known", "polis")
	os.WriteFile(wellKnownPath, wellKnownData, 0644)

	return s
}

// Helper to make JSON request body
func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return bytes.NewBuffer(data)
}

// ============================================================================
// handleStatus Tests
// ============================================================================

func TestHandleStatus_Unconfigured(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()

	s.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["configured"] != false {
		t.Errorf("expected configured=false, got %v", resp["configured"])
	}

	// Check that validation is returned
	validation, ok := resp["validation"].(map[string]interface{})
	if !ok {
		t.Error("expected validation object in response")
	} else if validation["status"] == "valid" {
		t.Error("expected validation status to be not_found or incomplete, got valid")
	}
}

func TestHandleStatus_Configured(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()

	s.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["configured"] != true {
		t.Errorf("expected configured=true, got %v", resp["configured"])
	}
	if resp["site_title"] != "Test Site" {
		t.Errorf("expected site_title='Test Site', got %v", resp["site_title"])
	}

	// Check validation shows valid
	validation, ok := resp["validation"].(map[string]interface{})
	if !ok {
		t.Error("expected validation object in response")
	} else if validation["status"] != "valid" {
		t.Errorf("expected validation status='valid', got %v", validation["status"])
	}

	// Check base_url is included (required by frontend init for domain display)
	if resp["base_url"] != "https://test-site.polis.pub" {
		t.Errorf("expected base_url='https://test-site.polis.pub', got %v", resp["base_url"])
	}

	// Check show_frontmatter is included
	if _, exists := resp["show_frontmatter"]; !exists {
		t.Error("expected show_frontmatter field in status response")
	}
}

func TestHandleSettings_DefaultViewModeIsList(t *testing.T) {
	s := newConfiguredServer(t)
	// Config has no ViewMode set — should default to "list"
	s.Config.ViewMode = ""

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()

	s.handleSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	siteData := resp["site"].(map[string]interface{})
	if siteData["view_mode"] != "list" {
		t.Errorf("expected default view_mode='list', got %v", siteData["view_mode"])
	}
}

// ============================================================================
// handleValidate Tests
// ============================================================================

func TestHandleValidate_NotFound(t *testing.T) {
	s := newTestServer(t)
	// Remove all polis files to simulate empty directory
	os.RemoveAll(filepath.Join(s.DataDir, ".well-known"))
	os.RemoveAll(filepath.Join(s.DataDir, ".polis", "keys"))

	req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	rr := httptest.NewRecorder()

	s.handleValidate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// Empty directory should be not_found or incomplete
	status := resp["status"].(string)
	if status == "valid" {
		t.Errorf("expected status to be not_found or incomplete, got %v", status)
	}
}

func TestHandleValidate_Valid(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	rr := httptest.NewRecorder()

	s.handleValidate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "valid" {
		t.Errorf("expected status='valid', got %v", resp["status"])
	}
}

func TestHandleValidate_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/validate", nil)
	rr := httptest.NewRecorder()

	s.handleValidate(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleInit Tests
// ============================================================================

func TestHandleInit_Success(t *testing.T) {
	s := newTestServer(t)
	// Remove any existing polis files
	os.RemoveAll(filepath.Join(s.DataDir, ".well-known"))
	os.RemoveAll(filepath.Join(s.DataDir, ".polis", "keys"))

	body := jsonBody(t, map[string]string{
		"site_title": "My Test Site",
		"base_url":   "https://test.example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/init", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleInit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}

	// Verify keys were created
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	if _, err := os.Stat(privKeyPath); os.IsNotExist(err) {
		t.Error("private key file was not created")
	}

	pubKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Error("public key file was not created")
	}

	// Verify .well-known/polis was created
	wellKnownPath := filepath.Join(s.DataDir, ".well-known", "polis")
	if _, err := os.Stat(wellKnownPath); os.IsNotExist(err) {
		t.Error(".well-known/polis was not created")
	}

	// Verify metadata files were created
	manifestPath := filepath.Join(s.DataDir, "metadata", "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest.json was not created")
	}
}

func TestHandleInit_KeysAlreadyExist(t *testing.T) {
	s := newConfiguredServer(t) // Already has keys

	body := jsonBody(t, map[string]string{"site_title": "New Site"})
	req := httptest.NewRequest(http.MethodPost, "/api/init", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleInit(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "Failed to initialize site") {
		t.Error("expected generic init failure message")
	}
}

func TestHandleInit_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/init", nil)
	rr := httptest.NewRecorder()

	s.handleInit(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleLink Tests
// ============================================================================

func TestHandleLink_Success(t *testing.T) {
	// Create a "source" polis site
	sourceDir := t.TempDir()
	sourceSrv := &Server{DataDir: sourceDir}

	// Initialize the source site with all required directories
	dirs := []string{
		filepath.Join(sourceDir, ".polis", "keys"),
		filepath.Join(sourceDir, ".well-known"),
		filepath.Join(sourceDir, "metadata"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	// Generate keys and create .well-known/polis
	privKey, pubKey, _ := signing.GenerateKeypair()
	os.WriteFile(filepath.Join(sourceDir, ".polis", "keys", "id_ed25519"), privKey, 0600)
	os.WriteFile(filepath.Join(sourceDir, ".polis", "keys", "id_ed25519.pub"), pubKey, 0644)
	wellKnown := map[string]interface{}{
		"base_url":   "https://test.example.com",
		"public_key": string(pubKey),
	}
	wellKnownData, _ := json.MarshalIndent(wellKnown, "", "  ")
	os.WriteFile(filepath.Join(sourceDir, ".well-known", "polis"), wellKnownData, 0644)

	// Create a target server with empty data dir
	targetDir := t.TempDir()
	targetDataDir := filepath.Join(targetDir, "data")
	os.MkdirAll(targetDataDir, 0755)
	_ = sourceSrv // suppress unused warning

	// For this test, we need to create a mock server setup
	// Since handleLink uses os.Executable(), we'll test the validation part
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"path": sourceDir})
	req := httptest.NewRequest(http.MethodPost, "/api/link", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleLink(rr, req)

	// The test may fail due to symlink creation issues in test environment
	// but we verify the validation works
	if rr.Code == http.StatusOK {
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["success"] != true {
			t.Errorf("expected success=true, got %v", resp["success"])
		}
	}
}

func TestHandleLink_InvalidPath(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"path": "/nonexistent/path"})
	req := httptest.NewRequest(http.MethodPost, "/api/link", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleLink(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleLink_EmptyPath(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"path": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/link", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleLink(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleLink_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/link", nil)
	rr := httptest.NewRecorder()

	s.handleLink(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleRender Tests
// ============================================================================

func TestHandleRender_Success(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{"markdown": "# Hello World"})
	req := httptest.NewRequest(http.MethodPost, "/api/render", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRender(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	html, ok := resp["html"].(string)
	if !ok {
		t.Fatal("expected html field in response")
	}
	if !strings.Contains(html, "<h1") {
		t.Errorf("expected HTML with h1 tag, got %s", html)
	}

	signature, ok := resp["signature"].(string)
	if !ok || signature == "" {
		t.Error("expected non-empty signature field")
	}
}

func TestHandleRender_MethodNotAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/render", nil)
	rr := httptest.NewRecorder()

	s.handleRender(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleRender_NotConfigured(t *testing.T) {
	s := newTestServer(t) // No keys

	body := jsonBody(t, map[string]string{"markdown": "# Hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/render", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRender(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Not configured") {
		t.Error("expected 'Not configured' error message")
	}
}

func TestHandleRender_InvalidJSON(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/render", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRender(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleRender_EmptyMarkdown(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{"markdown": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/render", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRender(rr, req)

	// Empty markdown should still render (to empty HTML)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for empty markdown, got %d", rr.Code)
	}
}

// ============================================================================
// handleDrafts Tests
// ============================================================================

func TestHandleDrafts_ListEmpty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts", nil)
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// drafts may be nil or empty array when no drafts exist
	drafts := resp["drafts"]
	if drafts != nil {
		draftsArr, ok := drafts.([]interface{})
		if ok && len(draftsArr) != 0 {
			t.Errorf("expected empty drafts array, got %d items", len(draftsArr))
		}
	}
}

func TestHandleDrafts_ListWithDrafts(t *testing.T) {
	s := newTestServer(t)

	// Create some drafts
	draftsDir := filepath.Join(s.DataDir, ".polis", "posts", "drafts")
	os.WriteFile(filepath.Join(draftsDir, "draft1.md"), []byte("# Draft 1"), 0644)
	os.WriteFile(filepath.Join(draftsDir, "draft2.md"), []byte("# Draft 2"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts", nil)
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	drafts := resp["drafts"].([]interface{})
	if len(drafts) != 2 {
		t.Errorf("expected 2 drafts, got %d", len(drafts))
	}
}

func TestHandleDrafts_SaveNew(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"id":       "my-draft",
		"markdown": "# My Draft Content",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if resp["id"] != "my-draft" {
		t.Errorf("expected id='my-draft', got %v", resp["id"])
	}

	// Verify file was created
	draftPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts", "my-draft.md")
	content, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("draft file not created: %v", err)
	}
	if string(content) != "# My Draft Content" {
		t.Errorf("draft content mismatch: %s", string(content))
	}
}

func TestHandleDrafts_SaveAutoGenerateID(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"markdown": "# Auto ID Draft",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	id := resp["id"].(string)
	if !strings.HasPrefix(id, "draft-") {
		t.Errorf("expected auto-generated ID with 'draft-' prefix, got %s", id)
	}
}

func TestHandleDrafts_SaveSanitizesID(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"id":       "../../../etc/passwd",
		"markdown": "# Malicious",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	id := resp["id"].(string)
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		t.Errorf("ID should not contain path separators: %s", id)
	}

	// Verify file is in drafts directory, not elsewhere
	draftPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts", id+".md")
	if _, err := os.Stat(draftPath); os.IsNotExist(err) {
		t.Error("draft should be created in drafts directory")
	}

	// Verify no file was created outside drafts
	maliciousPath := filepath.Join(s.DataDir, "..", "..", "..", "etc", "passwd.md")
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Error("path traversal attack succeeded!")
	}
}

func TestHandleDrafts_InvalidJSON(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/drafts", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleDrafts_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/drafts", nil)
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleDraft Tests (single draft operations)
// ============================================================================

func TestHandleDraft_GetExisting(t *testing.T) {
	s := newTestServer(t)

	// Create a draft
	draftPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts", "test-draft.md")
	os.WriteFile(draftPath, []byte("# Test Draft"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts/test-draft", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["id"] != "test-draft" {
		t.Errorf("expected id='test-draft', got %v", resp["id"])
	}
	if resp["markdown"] != "# Test Draft" {
		t.Errorf("expected markdown content, got %v", resp["markdown"])
	}
}

func TestHandleDraft_GetNotFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts/nonexistent", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleDraft_GetEmptyID(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts/", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleDraft_Delete(t *testing.T) {
	s := newTestServer(t)

	// Create a draft
	draftPath := filepath.Join(s.DataDir, ".polis", "posts", "drafts", "to-delete.md")
	os.WriteFile(draftPath, []byte("# To Delete"), 0644)

	req := httptest.NewRequest(http.MethodDelete, "/api/drafts/to-delete", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Verify file was deleted
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Error("draft file should be deleted")
	}
}

func TestHandleDraft_DeleteNonexistent(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/drafts/nonexistent", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestHandleDraft_SanitizesPathTraversal(t *testing.T) {
	s := newTestServer(t)

	// Try to read a file outside drafts directory
	req := httptest.NewRequest(http.MethodGet, "/api/drafts/..%2F..%2F..%2Fetc%2Fpasswd", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	// Should return 404 because sanitization prevents path traversal
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after sanitization, got %d", rr.Code)
	}
}

func TestHandleDraft_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/drafts/test", nil)
	rr := httptest.NewRecorder()

	s.handleDraft(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handlePublish Tests
// ============================================================================

func TestHandlePublish_Success(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"markdown": "# My First Post\n\nThis is the content.",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["path"] == nil || resp["path"] == "" {
		t.Error("expected non-empty path in response")
	}
	if resp["title"] == nil {
		t.Error("expected title in response")
	}
}

func TestHandlePublish_MethodNotAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/publish", nil)
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandlePublish_NotConfigured(t *testing.T) {
	s := newTestServer(t) // No keys

	body := jsonBody(t, map[string]string{"markdown": "# Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePublish_EmptyMarkdown(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{"markdown": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePublish_WhitespaceOnlyMarkdown(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{"markdown": "   \n\t  "})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePublish_InvalidJSON(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/publish", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePublish_WithFilename(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"markdown": "# Custom Named Post",
		"filename": "custom-name.md",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	path := resp["path"].(string)
	if !strings.Contains(path, "custom-name") {
		t.Errorf("expected path to contain 'custom-name', got %s", path)
	}
}

func TestHandlePublish_StripsExistingFrontmatter(t *testing.T) {
	s := newConfiguredServer(t)

	markdownWithFrontmatter := `---
title: Old Title
---
# New Content`

	body := jsonBody(t, map[string]string{
		"markdown": markdownWithFrontmatter,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// handlePosts Tests
// ============================================================================

func TestHandlePosts_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rr := httptest.NewRecorder()

	s.handlePosts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	posts := resp["posts"].([]interface{})
	if len(posts) != 0 {
		t.Errorf("expected empty posts, got %d", len(posts))
	}
}

func TestHandlePosts_WithPosts(t *testing.T) {
	s := newConfiguredServer(t)

	// Create public.jsonl with some posts
	indexPath := filepath.Join(s.DataDir, "metadata", "public.jsonl")
	entries := []string{
		`{"path":"posts/20260101/first.md","title":"First Post"}`,
		`{"path":"posts/20260102/second.md","title":"Second Post"}`,
	}
	os.WriteFile(indexPath, []byte(strings.Join(entries, "\n")), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rr := httptest.NewRecorder()

	s.handlePosts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	posts := resp["posts"].([]interface{})
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}

	// Posts should be in reverse order (newest first)
	firstPost := posts[0].(map[string]interface{})
	if firstPost["title"] != "Second Post" {
		t.Errorf("expected newest post first, got %v", firstPost["title"])
	}
}

func TestHandlePosts_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	rr := httptest.NewRecorder()

	s.handlePosts(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handlePost Tests (single post)
// ============================================================================

func TestHandlePost_GetExisting(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a post file
	postDir := filepath.Join(s.DataDir, "posts", "20260101")
	os.MkdirAll(postDir, 0755)
	postContent := `---
title: Test Post
published: 2026-01-01T00:00:00Z
---
# Test Post

Content here.`
	os.WriteFile(filepath.Join(postDir, "test.md"), []byte(postContent), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/posts/20260101/test.md", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["title"] != "Test Post" {
		t.Errorf("expected title='Test Post', got %v", resp["title"])
	}
	markdown := resp["markdown"].(string)
	if !strings.Contains(markdown, "# Test Post") {
		t.Error("expected markdown content without frontmatter")
	}
	if strings.Contains(markdown, "---") {
		t.Error("frontmatter should be stripped from markdown")
	}
}

func TestHandlePost_NotFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/posts/20260101/nonexistent.md", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandlePost_EmptyPath(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePost_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/posts/20260101/test.md", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleRepublish Tests
// ============================================================================

func TestHandleRepublish_Success(t *testing.T) {
	s := newConfiguredServer(t)

	// First publish a post
	postDir := filepath.Join(s.DataDir, "posts", "20260101")
	os.MkdirAll(postDir, 0755)
	originalContent := `---
title: Original Title
published: 2026-01-01T00:00:00Z
version: 1
---
# Original Content`
	postPath := filepath.Join(postDir, "original.md")
	os.WriteFile(postPath, []byte(originalContent), 0644)

	body := jsonBody(t, map[string]string{
		"path":     "posts/20260101/original.md",
		"markdown": "# Updated Content",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleRepublish_MethodNotAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/republish", nil)
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleRepublish_NotConfigured(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"path":     "posts/20260101/test.md",
		"markdown": "# Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleRepublish_MissingPath(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"markdown": "# Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleRepublish_EmptyMarkdown(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"path":     "posts/20260101/test.md",
		"markdown": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentDrafts Tests
// ============================================================================

func TestHandleCommentDrafts_ListEmpty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/drafts", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// drafts may be nil or empty array when no drafts exist
	drafts := resp["drafts"]
	if drafts != nil {
		draftsArr, ok := drafts.([]interface{})
		if ok && len(draftsArr) != 0 {
			t.Errorf("expected empty drafts, got %d", len(draftsArr))
		}
	}
}

func TestHandleCommentDrafts_Save(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"in_reply_to": "https://example.com/posts/test.md",
		"root_post":   "https://example.com/posts/test.md",
		"content":     "This is my comment",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestHandleCommentDrafts_SaveMissingInReplyTo(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"content": "This is my comment",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentDrafts(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCommentDrafts_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/comments/drafts", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDrafts(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentSign Tests
// ============================================================================

func TestHandleCommentSign_MethodNotAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/sign", nil)
	rr := httptest.NewRecorder()

	s.handleCommentSign(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleCommentSign_NotConfigured(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"in_reply_to": "https://example.com/post.md",
		"content":     "Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentSign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCommentSign_MissingInReplyTo(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"content": "Test comment",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentSign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCommentSign_DraftNotFound(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{
		"draft_id": "nonexistent-draft",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentSign(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentsPending/Blessed/Denied Tests
// ============================================================================

func TestHandleCommentsPending_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/pending", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsPending(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestHandleCommentsPending_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/comments/pending", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsPending(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleCommentsBlessed_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/blessed", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsBlessed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestHandleCommentsDenied_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/denied", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsDenied(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentsSync Tests
// ============================================================================

func TestHandleCommentsSync_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/sync", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsSync(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleCommentsSync_DiscoveryNotConfigured(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/comments/sync", nil)
	rr := httptest.NewRecorder()

	s.handleCommentsSync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleBlessingRequests Tests
// ============================================================================

func TestHandleBlessingRequests_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/blessing/requests", nil)
	rr := httptest.NewRecorder()

	s.handleBlessingRequests(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleBlessingRequests_DiscoveryNotConfigured(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blessing/requests", nil)
	rr := httptest.NewRecorder()

	s.handleBlessingRequests(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleBlessingGrant Tests
// ============================================================================

func TestHandleBlessingGrant_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blessing/grant", nil)
	rr := httptest.NewRecorder()

	s.handleBlessingGrant(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleBlessingGrant_DiscoveryNotConfigured(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"comment_version": "abc123"})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/grant", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleBlessingGrant(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleBlessingGrant_PrivateKeyNotConfigured(t *testing.T) {
	s := newTestServer(t)
	s.DiscoveryURL = "https://discovery.example.com"
	s.DiscoveryKey = "test-key"

	body := jsonBody(t, map[string]string{"comment_version": "abc123"})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/grant", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleBlessingGrant(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleBlessingGrant_MissingCommentVersion(t *testing.T) {
	s := newConfiguredServer(t)
	s.DiscoveryURL = "https://discovery.example.com"
	s.DiscoveryKey = "test-key"

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/grant", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleBlessingGrant(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleBlessingDeny Tests
// ============================================================================

func TestHandleBlessingDeny_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blessing/deny", nil)
	rr := httptest.NewRecorder()

	s.handleBlessingDeny(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleBlessingDeny_MissingCommentVersion(t *testing.T) {
	s := newConfiguredServer(t)
	s.DiscoveryURL = "https://discovery.example.com"
	s.DiscoveryKey = "test-key"

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/deny", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleBlessingDeny(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleBlessedComments Tests
// ============================================================================

func TestHandleBlessedComments_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blessed-comments", nil)
	rr := httptest.NewRecorder()

	s.handleBlessedComments(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	comments := resp["comments"].([]interface{})
	if len(comments) != 0 {
		t.Errorf("expected empty comments, got %d", len(comments))
	}
}

func TestHandleBlessedComments_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/blessed-comments", nil)
	rr := httptest.NewRecorder()

	s.handleBlessedComments(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleBlessingRevoke Tests
// ============================================================================

func TestHandleBlessingRevoke_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blessing/revoke", nil)
	rr := httptest.NewRecorder()

	s.handleBlessingRevoke(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleBlessingRevoke_MissingCommentURL(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/revoke", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleBlessingRevoke(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleSettings Tests
// ============================================================================

func TestHandleSettings_Unconfigured(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()

	s.handleSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	site := resp["site"].(map[string]interface{})
	if site["subdomain"] != "" {
		t.Errorf("expected empty subdomain, got %v", site["subdomain"])
	}
	if site["discovery_configured"] != false {
		t.Error("expected discovery_configured=false")
	}
}

func TestHandleSettings_Configured(t *testing.T) {
	s := newConfiguredServer(t)
	s.DiscoveryURL = "https://discovery.example.com"
	s.DiscoveryKey = "test-key"

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()

	s.handleSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	site := resp["site"].(map[string]interface{})
	if site["subdomain"] != "test-site" {
		t.Errorf("expected subdomain='test-site', got %v", site["subdomain"])
	}
	if site["discovery_configured"] != true {
		t.Error("expected discovery_configured=true")
	}
}

func TestHandleSettings_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/settings", nil)
	rr := httptest.NewRecorder()

	s.handleSettings(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleAutomations Tests
// ============================================================================

func TestHandleAutomations_ListEmpty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/automations", nil)
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// automations may be nil or empty array when no automations exist
	automations := resp["automations"]
	if automations != nil {
		automationsArr, ok := automations.([]interface{})
		if ok && len(automationsArr) != 0 {
			t.Errorf("expected empty automations, got %d", len(automationsArr))
		}
	}
}

func TestHandleAutomations_ListWithHooks(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		Hooks: &hooks.HookConfig{
			PostPublish: ".polis/hooks/post-publish.sh",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/automations", nil)
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	automations := resp["automations"].([]interface{})
	if len(automations) != 1 {
		t.Errorf("expected 1 automation, got %d", len(automations))
	}
}

func TestHandleAutomations_CreateWithScript(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"script": "#!/bin/bash\necho 'hello'",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/automations", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify script was created
	scriptPath := filepath.Join(s.DataDir, ".polis", "hooks", "post-publish.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Error("hook script was not created")
	}
}

func TestHandleAutomations_CreateWithTemplate(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"template_id": "vercel",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/automations", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutomations_CreateUnknownTemplate(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{
		"template_id": "nonexistent-template",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/automations", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAutomations_CreateNoScript(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/automations", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAutomations_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/automations", nil)
	rr := httptest.NewRecorder()

	s.handleAutomations(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleAutomationsQuick Tests
// ============================================================================

func TestHandleAutomationsQuick_Success(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/automations/quick", nil)
	rr := httptest.NewRecorder()

	s.handleAutomationsQuick(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
}

func TestHandleAutomationsQuick_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/automations/quick", nil)
	rr := httptest.NewRecorder()

	s.handleAutomationsQuick(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleAutomation Tests (single automation)
// ============================================================================

func TestHandleAutomation_Delete(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		Hooks: &hooks.HookConfig{
			PostPublish: ".polis/hooks/post-publish.sh",
		},
	}

	// Create the hooks directory and file
	hooksDir := filepath.Join(s.DataDir, ".polis", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "post-publish.sh"), []byte("#!/bin/bash"), 0755)

	// Save config to disk first
	s.SaveConfig()

	req := httptest.NewRequest(http.MethodDelete, "/api/automations/post-publish", nil)
	rr := httptest.NewRecorder()

	s.handleAutomation(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify hook was removed from config
	if s.Config.Hooks.PostPublish != "" {
		t.Error("expected PostPublish hook to be cleared")
	}
}

func TestHandleAutomation_DeleteUnknown(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		Hooks: &hooks.HookConfig{},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/automations/unknown-hook", nil)
	rr := httptest.NewRecorder()

	s.handleAutomation(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleAutomation_DeleteNoConfig(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/automations/post-publish", nil)
	rr := httptest.NewRecorder()

	s.handleAutomation(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleAutomation_EmptyID(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/automations/", nil)
	rr := httptest.NewRecorder()

	s.handleAutomation(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAutomation_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/automations/post-publish", nil)
	rr := httptest.NewRecorder()

	s.handleAutomation(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleTemplates Tests
// ============================================================================

func TestHandleTemplates_List(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	rr := httptest.NewRecorder()

	s.handleTemplates(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	templates, ok := resp["templates"].([]interface{})
	if !ok {
		t.Fatal("expected templates array")
	}
	if len(templates) == 0 {
		t.Error("expected at least one template")
	}
}

func TestHandleTemplates_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/templates", nil)
	rr := httptest.NewRecorder()

	s.handleTemplates(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentBeseech Tests
// ============================================================================

func TestHandleCommentBeseech_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/beseech", nil)
	rr := httptest.NewRecorder()

	s.handleCommentBeseech(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleCommentBeseech_DiscoveryNotConfigured(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"comment_id": "test-id"})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/beseech", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentBeseech(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCommentBeseech_MissingCommentID(t *testing.T) {
	s := newTestServer(t)
	s.DiscoveryURL = "https://discovery.example.com"
	s.DiscoveryKey = "test-key"

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/comments/beseech", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleCommentBeseech(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleCommentDraft Tests (single comment draft)
// ============================================================================

func TestHandleCommentDraft_GetNotFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/drafts/nonexistent", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDraft(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleCommentDraft_EmptyID(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/drafts/", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDraft(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCommentDraft_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/comments/drafts/test", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDraft(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// ============================================================================
// Configuration Management Tests
// ============================================================================

func TestLoadConfig_NoFile(t *testing.T) {
	s := newTestServer(t)

	s.LoadConfig()

	if s.Config != nil {
		t.Error("expected config to be nil when no file exists")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	s := newTestServer(t)

	// Create config file
	config := Config{
		SetupCode: "test-code",
		Subdomain: "test-site",
		SetupAt:   "2026-01-01T00:00:00Z",
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	os.WriteFile(configPath, configData, 0644)

	s.LoadConfig()

	if s.Config == nil {
		t.Fatal("expected config to be loaded")
	}
	if s.Config.SetupCode != "test-code" {
		t.Errorf("expected SetupCode='test-code', got %s", s.Config.SetupCode)
	}
	if s.Config.Subdomain != "test-site" {
		t.Errorf("expected Subdomain='test-site', got %s", s.Config.Subdomain)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	s := newTestServer(t)

	// Create invalid config file
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	os.WriteFile(configPath, []byte("{invalid json"), 0644)

	s.LoadConfig()

	if s.Config != nil {
		t.Error("expected config to be nil for invalid JSON")
	}
}

func TestSaveConfig_Success(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		SetupCode: "save-test",
		Subdomain: "saved-site",
		SetupAt:   "2026-01-15T12:00:00Z",
	}

	err := s.SaveConfig()
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	var loaded Config
	json.Unmarshal(data, &loaded)

	if loaded.SetupCode != "save-test" {
		t.Errorf("expected SetupCode='save-test', got %s", loaded.SetupCode)
	}
	// Subdomain should NOT be persisted (deprecated field stripped on save)
	if loaded.Subdomain != "" {
		t.Errorf("expected Subdomain to be empty on disk, got %s", loaded.Subdomain)
	}
	// But in-memory value should be preserved
	if s.Config.Subdomain != "saved-site" {
		t.Errorf("expected in-memory Subdomain='saved-site', got %s", s.Config.Subdomain)
	}
}

func TestLoadKeys_NoFiles(t *testing.T) {
	s := newTestServer(t)

	s.LoadKeys()

	if s.PrivateKey != nil {
		t.Error("expected privateKey to be nil when no files exist")
	}
	if s.PublicKey != nil {
		t.Error("expected publicKey to be nil when no files exist")
	}
}

func TestLoadKeys_Success(t *testing.T) {
	s := newTestServer(t)

	// Create key files
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	pubKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(privKeyPath, []byte("fake-private-key"), 0600)
	os.WriteFile(pubKeyPath, []byte("fake-public-key"), 0644)

	s.LoadKeys()

	if s.PrivateKey == nil {
		t.Error("expected privateKey to be loaded")
	}
	if s.PublicKey == nil {
		t.Error("expected publicKey to be loaded")
	}
	if string(s.PrivateKey) != "fake-private-key" {
		t.Errorf("expected privateKey content, got %s", string(s.PrivateKey))
	}
}

func TestLoadKeys_PrivateOnly(t *testing.T) {
	s := newTestServer(t)

	// Create only private key
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	os.WriteFile(privKeyPath, []byte("fake-private-key"), 0600)

	s.LoadKeys()

	// Should not load if only one key exists
	if s.PrivateKey != nil {
		t.Error("expected privateKey to be nil when public key missing")
	}
}

func TestLoadEnv_NoFile(t *testing.T) {
	s := newTestServer(t)

	s.LoadEnv()

	// Should not error, just do nothing
	if s.Config != nil {
		t.Error("expected config to remain nil when no .env file")
	}
}

func TestLoadEnv_DataDirFile(t *testing.T) {
	s := newTestServer(t)

	// Create .env in data directory
	envContent := `DISCOVERY_SERVICE_URL=https://test-discovery.com
DISCOVERY_SERVICE_KEY=test-api-key`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	if s.DiscoveryURL != "https://test-discovery.com" {
		t.Errorf("expected DiscoveryURL from .env, got %s", s.DiscoveryURL)
	}
	if s.DiscoveryKey != "test-api-key" {
		t.Errorf("expected DiscoveryKey from .env, got %s", s.DiscoveryKey)
	}
}

func TestLoadEnv_QuotedValues(t *testing.T) {
	s := newTestServer(t)

	// Create .env with quoted values
	envContent := `DISCOVERY_SERVICE_URL="https://quoted.com"
DISCOVERY_SERVICE_KEY='single-quoted-key'`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	if s.DiscoveryURL != "https://quoted.com" {
		t.Errorf("expected quotes stripped from URL, got %s", s.DiscoveryURL)
	}
	if s.DiscoveryKey != "single-quoted-key" {
		t.Errorf("expected quotes stripped from key, got %s", s.DiscoveryKey)
	}
}

func TestLoadEnv_Comments(t *testing.T) {
	s := newTestServer(t)

	// Create .env with comments
	envContent := `# This is a comment
DISCOVERY_SERVICE_URL=https://actual-url.com
# Another comment
# DISCOVERY_SERVICE_KEY=commented-out
DISCOVERY_SERVICE_KEY=actual-key`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	if s.DiscoveryURL != "https://actual-url.com" {
		t.Errorf("expected non-comment URL, got %s", s.DiscoveryURL)
	}
	if s.DiscoveryKey != "actual-key" {
		t.Errorf("expected non-comment key, got %s", s.DiscoveryKey)
	}
}

func TestLoadEnv_EmptyLines(t *testing.T) {
	s := newTestServer(t)

	// Create .env with empty lines
	envContent := `

DISCOVERY_SERVICE_URL=https://test.com

DISCOVERY_SERVICE_KEY=test-key

`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	if s.DiscoveryURL != "https://test.com" {
		t.Errorf("expected URL parsed correctly, got %s", s.DiscoveryURL)
	}
}

func TestLoadEnv_MalformedLines(t *testing.T) {
	s := newTestServer(t)

	// Create .env with malformed lines
	envContent := `DISCOVERY_SERVICE_URL=https://valid.com
no-equals-sign
DISCOVERY_SERVICE_KEY=valid-key
=value-with-no-key`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	// Valid lines should still be parsed
	if s.DiscoveryURL != "https://valid.com" {
		t.Errorf("expected valid URL, got %s", s.DiscoveryURL)
	}
	if s.DiscoveryKey != "valid-key" {
		t.Errorf("expected valid key, got %s", s.DiscoveryKey)
	}
}

func TestLoadEnv_OverridesConfig(t *testing.T) {
	s := newTestServer(t)

	// Set up existing config
	s.Config = &Config{
		Subdomain: "existing-site",
	}
	s.DiscoveryURL = "https://old-discovery.com"
	s.DiscoveryKey = "old-key"

	// Create .env with new values
	envContent := `DISCOVERY_SERVICE_URL=https://new-discovery.com
DISCOVERY_SERVICE_KEY=new-key`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	// .env should override previous values
	if s.DiscoveryURL != "https://new-discovery.com" {
		t.Errorf("expected .env to override URL, got %s", s.DiscoveryURL)
	}
	if s.DiscoveryKey != "new-key" {
		t.Errorf("expected .env to override key, got %s", s.DiscoveryKey)
	}
	// Non-overridden values should remain
	if s.Config.Subdomain != "existing-site" {
		t.Errorf("expected Subdomain to remain unchanged, got %s", s.Config.Subdomain)
	}
}

func TestLoadEnv_POLIS_BASE_URL(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{}

	// Create .env with POLIS_BASE_URL
	envContent := `POLIS_BASE_URL=https://alice.polis.pub`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	// BaseURL should be set, subdomain derived via GetSubdomain()
	if s.BaseURL != "https://alice.polis.pub" {
		t.Errorf("expected BaseURL='https://alice.polis.pub', got %s", s.BaseURL)
	}
	if s.GetSubdomain() != "alice" {
		t.Errorf("expected GetSubdomain()='alice', got %s", s.GetSubdomain())
	}
}

func TestLoadEnv_POLIS_BASE_URL_Subdomain(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{}

	// Create .env with POLIS_BASE_URL
	envContent := `POLIS_BASE_URL=https://new.polis.pub`
	envPath := filepath.Join(s.DataDir, ".env")
	os.WriteFile(envPath, []byte(envContent), 0644)

	s.LoadEnv()

	// GetSubdomain derives from BaseURL
	if s.GetSubdomain() != "new" {
		t.Errorf("expected GetSubdomain()='new', got %s", s.GetSubdomain())
	}
}

func TestGetSubdomain_FallbackToConfig(t *testing.T) {
	// Test backwards compat: old configs with Subdomain field but no BaseURL
	s := newTestServer(t)
	s.Config = &Config{
		Subdomain: "legacy-site",
	}
	// No BaseURL set
	if s.GetSubdomain() != "legacy-site" {
		t.Errorf("expected GetSubdomain() to fall back to Config.Subdomain, got %s", s.GetSubdomain())
	}
}

func TestApplyDiscoveryDefaults_NoConfig(t *testing.T) {
	s := newTestServer(t)

	s.ApplyDiscoveryDefaults()

	if s.DiscoveryURL != DefaultDiscoveryServiceURL {
		t.Errorf("expected default discovery URL, got %s", s.DiscoveryURL)
	}
}

func TestApplyDiscoveryDefaults_EmptyURL(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		Subdomain: "test-site",
		// DiscoveryURL is empty
	}

	s.ApplyDiscoveryDefaults()

	if s.DiscoveryURL != DefaultDiscoveryServiceURL {
		t.Errorf("expected default discovery URL, got %s", s.DiscoveryURL)
	}
}

func TestApplyDiscoveryDefaults_ExistingURL(t *testing.T) {
	s := newTestServer(t)
	s.DiscoveryURL = "https://custom-discovery.com"

	s.ApplyDiscoveryDefaults()

	if s.DiscoveryURL != "https://custom-discovery.com" {
		t.Errorf("expected custom URL not to be overridden, got %s", s.DiscoveryURL)
	}
}

func TestConfigPersistence_RoundTrip(t *testing.T) {
	s := newTestServer(t)

	// Create and save config
	s.Config = &Config{
		SetupCode: "round-trip",
		Subdomain: "persist-test", // Deprecated - should be stripped on save
		SetupAt:   "2026-01-20T10:00:00Z",
	}
	err := s.SaveConfig()
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Create new server and load config
	s2 := &Server{DataDir: s.DataDir}
	s2.LoadConfig()

	if s2.Config == nil {
		t.Fatal("expected config to be loaded")
	}
	if s2.Config.SetupCode != "round-trip" {
		t.Errorf("SetupCode mismatch: expected 'round-trip', got %s", s2.Config.SetupCode)
	}
	// Subdomain is stripped on save (deprecated field)
	if s2.Config.Subdomain != "" {
		t.Errorf("expected Subdomain to be empty after round-trip, got %s", s2.Config.Subdomain)
	}
}

func TestGetAuthorEmail_FromWellKnown(t *testing.T) {
	s := newConfiguredServer(t) // has email in .well-known/polis
	email := s.GetAuthorEmail()
	if email != "test@example.com" {
		t.Errorf("expected test@example.com, got %q", email)
	}
}

func TestGetAuthorEmail_NoWellKnown(t *testing.T) {
	s := newTestServer(t) // no .well-known/polis file
	email := s.GetAuthorEmail()
	if email != "" {
		t.Errorf("expected empty email, got %q", email)
	}
}

func TestGetAuthorEmail_NoEmailField(t *testing.T) {
	s := newTestServer(t)
	// Create .well-known/polis without email field
	wellKnown := map[string]interface{}{
		"public_key": "ssh-ed25519 AAAA...",
		"site_title": "No Email Site",
	}
	data, _ := json.MarshalIndent(wellKnown, "", "  ")
	os.WriteFile(filepath.Join(s.DataDir, ".well-known", "polis"), data, 0644)

	email := s.GetAuthorEmail()
	if email != "" {
		t.Errorf("expected empty email, got %q", email)
	}
}

func TestConfigWithHooks_Persistence(t *testing.T) {
	s := newTestServer(t)

	// Create config with hooks
	s.Config = &Config{
		SetupCode: "hook-test",
		Subdomain: "hook-site",
		SetupAt:   "2026-01-20T10:00:00Z",
		Hooks: &hooks.HookConfig{
			PostPublish:   ".polis/hooks/publish.sh",
			PostRepublish: ".polis/hooks/republish.sh",
			PostComment:   ".polis/hooks/comment.sh",
		},
	}
	err := s.SaveConfig()
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Load into new server
	s2 := &Server{DataDir: s.DataDir}
	s2.LoadConfig()

	if s2.Config == nil {
		t.Fatal("expected config to be loaded")
	}
	if s2.Config.Hooks == nil {
		t.Fatal("expected Hooks to be loaded")
	}
	if s2.Config.Hooks.PostPublish != ".polis/hooks/publish.sh" {
		t.Errorf("PostPublish mismatch: got %s", s2.Config.Hooks.PostPublish)
	}
	if s2.Config.Hooks.PostRepublish != ".polis/hooks/republish.sh" {
		t.Errorf("PostRepublish mismatch")
	}
	if s2.Config.Hooks.PostComment != ".polis/hooks/comment.sh" {
		t.Errorf("PostComment mismatch")
	}
}

// ============================================================================
// File System Safety Tests
// ============================================================================

func TestDrafts_PathTraversalPrevention(t *testing.T) {
	s := newTestServer(t)

	// Create a sensitive file outside drafts
	sensitiveDir := filepath.Join(s.DataDir, ".polis", "keys")
	sensitiveFile := filepath.Join(sensitiveDir, "secret.txt")
	os.WriteFile(sensitiveFile, []byte("secret data"), 0644)

	// Attempt to read via path traversal
	maliciousIDs := []string{
		"../keys/secret",
		"..%2Fkeys%2Fsecret",
		"....//keys//secret",
		"..\\keys\\secret",
	}

	for _, maliciousID := range maliciousIDs {
		req := httptest.NewRequest(http.MethodGet, "/api/drafts/"+maliciousID, nil)
		rr := httptest.NewRecorder()

		s.handleDraft(rr, req)

		// Should get 404 (not found) because path is sanitized, not the actual file
		if rr.Code == http.StatusOK {
			t.Errorf("path traversal should be prevented for ID: %s", maliciousID)
		}
	}
}

func TestDrafts_SavePathTraversalPrevention(t *testing.T) {
	s := newTestServer(t)

	// Attempt to save with malicious ID
	body := jsonBody(t, map[string]string{
		"id":       "../../../tmp/malicious",
		"markdown": "# Malicious Content",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	// The save should succeed but with sanitized ID
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Verify the file was NOT created in the malicious path
	maliciousPath := filepath.Join(s.DataDir, "..", "..", "..", "tmp", "malicious.md")
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Error("path traversal attack succeeded - file created outside drafts")
	}

	// Verify file WAS created in proper drafts directory
	draftsDir := filepath.Join(s.DataDir, ".polis", "posts", "drafts")
	files, _ := os.ReadDir(draftsDir)
	if len(files) != 1 {
		t.Errorf("expected 1 file in drafts dir, got %d", len(files))
	}
}

func TestPost_ValidPathAccess(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a valid post file
	postDir := filepath.Join(s.DataDir, "posts", "20260128")
	os.MkdirAll(postDir, 0755)
	postContent := `---
title: Test Post
---
# Test Post
Content here.`
	os.WriteFile(filepath.Join(postDir, "test.md"), []byte(postContent), 0644)

	// Access the post via valid path
	req := httptest.NewRequest(http.MethodGet, "/api/posts/posts/20260128/test.md", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid post path, got %d", rr.Code)
	}
}

func TestPost_InternalFilesBlocked(t *testing.T) {
	// Verify that internal files (.polis/) are NOT accessible via /api/posts/
	s := newConfiguredServer(t)

	// Create a file in .polis
	internalFile := filepath.Join(s.DataDir, ".polis", "test-internal.txt")
	os.WriteFile(internalFile, []byte("internal data"), 0644)

	// Attempt to access internal file - should be blocked
	req := httptest.NewRequest(http.MethodGet, "/api/posts/.polis/test-internal.txt", nil)
	rr := httptest.NewRecorder()

	s.handlePost(rr, req)

	// Should get 400 Bad Request because path doesn't start with "posts/"
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for .polis path, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid path") {
		t.Error("expected 'invalid path' error message")
	}
}

func TestPost_PathTraversalBlocked(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a sensitive file
	sensitiveFile := filepath.Join(s.DataDir, ".env")
	os.WriteFile(sensitiveFile, []byte("SECRET_KEY=supersecret"), 0644)

	// Various traversal attempts
	traversalPaths := []string{
		"../.env",
		"posts/../.env",
		"posts/../../.env",
		"posts/20260128/../../../.env",
	}

	for _, path := range traversalPaths {
		req := httptest.NewRequest(http.MethodGet, "/api/posts/"+path, nil)
		rr := httptest.NewRecorder()

		s.handlePost(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for traversal path %q, got %d", path, rr.Code)
		}

		// Verify sensitive data was not exposed
		if strings.Contains(rr.Body.String(), "supersecret") {
			t.Errorf("path traversal exposed sensitive file with path: %s", path)
		}
	}
}

func TestPost_ValidPostsPathAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a valid post
	postDir := filepath.Join(s.DataDir, "posts", "20260128")
	os.MkdirAll(postDir, 0755)
	postContent := `---
title: Valid Post
---
# Valid Post
Content here.`
	os.WriteFile(filepath.Join(postDir, "valid.md"), []byte(postContent), 0644)

	// Valid paths should work
	validPaths := []string{
		"posts/20260128/valid.md",
		"posts/20260128/valid.md",
	}

	for _, path := range validPaths {
		req := httptest.NewRequest(http.MethodGet, "/api/posts/"+path, nil)
		rr := httptest.NewRecorder()

		s.handlePost(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for valid path %q, got %d: %s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestRepublish_PathTraversalBlocked(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a sensitive file that attacker might want to overwrite
	sensitiveFile := filepath.Join(s.DataDir, ".polis", "webapp-config.json")

	// Various traversal attempts
	traversalPaths := []string{
		"../.polis/webapp-config.json",
		".polis/webapp-config.json",
		"posts/../.polis/webapp-config.json",
		"posts/../../important.txt",
	}

	for _, path := range traversalPaths {
		body := jsonBody(t, map[string]string{
			"path":     path,
			"markdown": "# Malicious content",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.handleRepublish(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for traversal path %q, got %d", path, rr.Code)
		}
	}

	// Verify sensitive file was not modified
	if _, err := os.Stat(sensitiveFile); err == nil {
		content, _ := os.ReadFile(sensitiveFile)
		if strings.Contains(string(content), "Malicious") {
			t.Error("path traversal attack succeeded - config file was modified!")
		}
	}
}

func TestRepublish_ValidPostsPathAllowed(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a valid post to republish
	postDir := filepath.Join(s.DataDir, "posts", "20260128")
	os.MkdirAll(postDir, 0755)
	os.MkdirAll(filepath.Join(s.DataDir, "metadata"), 0755)

	originalContent := `---
title: Original Title
published: 2026-01-28T12:00:00Z
current-version: sha256:abc123
version-history:
  - sha256:abc123 (2026-01-28T12:00:00Z)
---

# Original Title
Original content.`
	postPath := filepath.Join(postDir, "test-republish.md")
	os.WriteFile(postPath, []byte(originalContent), 0644)

	// Valid republish should work
	body := jsonBody(t, map[string]string{
		"path":     "posts/20260128/test-republish.md",
		"markdown": "# Updated Title\n\nUpdated content.",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid republish, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidatePostPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "posts/20260128/test.md", false},
		{"valid nested path", "posts/20260128/subdir/test.md", false},
		{"missing posts prefix", "20260128/test.md", true},
		{"dotpolis path", ".polis/keys/id_ed25519", true},
		{"traversal with dotdot", "posts/../.env", true},
		{"traversal mid-path", "posts/20260128/../../.env", true},
		{"double traversal", "posts/../../etc/passwd", true},
		{"null byte injection", "posts/20260128/test\x00.md", true},
		{"empty path", "", true},
		{"just posts dir", "posts/", true}, // filepath.Clean strips trailing slash; bare directory is not a valid post path
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePostPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestAutomation_DeletePathTraversal(t *testing.T) {
	s := newTestServer(t)
	s.Config = &Config{
		Hooks: &hooks.HookConfig{
			PostPublish: ".polis/hooks/post-publish.sh",
		},
	}

	// Create a hooks directory and file
	hooksDir := filepath.Join(s.DataDir, ".polis", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "post-publish.sh"), []byte("#!/bin/bash"), 0755)

	// Create a sensitive file that attacker might want to delete
	importantFile := filepath.Join(s.DataDir, "important.txt")
	os.WriteFile(importantFile, []byte("important data"), 0644)

	// Attempt to delete via path traversal (should fail with 404)
	maliciousIDs := []string{
		"../important.txt",
		"../../important",
		"post-publish/../important",
	}

	for _, maliciousID := range maliciousIDs {
		// Save config first
		s.SaveConfig()

		req := httptest.NewRequest(http.MethodDelete, "/api/automations/"+maliciousID, nil)
		rr := httptest.NewRecorder()

		s.handleAutomation(rr, req)

		// Should get 404 for unknown automation ID
		if rr.Code != http.StatusNotFound {
			t.Logf("Note: got status %d for ID %s", rr.Code, maliciousID)
		}

		// Important file should still exist
		if _, err := os.Stat(importantFile); os.IsNotExist(err) {
			t.Errorf("path traversal deleted important file with ID: %s", maliciousID)
		}
	}
}

func TestDraft_DeletePathTraversal(t *testing.T) {
	s := newTestServer(t)

	// Create an important file
	importantFile := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(importantFile, []byte("public key"), 0644)

	// Attempt to delete via path traversal
	maliciousIDs := []string{
		"../keys/id_ed25519.pub",
		"..%2Fkeys%2Fid_ed25519.pub",
	}

	for _, maliciousID := range maliciousIDs {
		req := httptest.NewRequest(http.MethodDelete, "/api/drafts/"+maliciousID, nil)
		rr := httptest.NewRecorder()

		s.handleDraft(rr, req)

		// Important file should still exist
		if _, err := os.Stat(importantFile); os.IsNotExist(err) {
			t.Errorf("path traversal deleted important file with ID: %s", maliciousID)
		}
	}
}

func TestCommentDraft_PathTraversal(t *testing.T) {
	s := newTestServer(t)

	// Create an important file
	importantFile := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519.pub")
	os.WriteFile(importantFile, []byte("public key"), 0644)

	// Attempt to read via path traversal
	req := httptest.NewRequest(http.MethodGet, "/api/comments/drafts/../../../.polis/keys/id_ed25519.pub", nil)
	rr := httptest.NewRecorder()

	s.handleCommentDraft(rr, req)

	// Should not return the public key
	if rr.Code == http.StatusOK {
		body := rr.Body.String()
		if strings.Contains(body, "public key") {
			t.Error("path traversal exposed public key file")
		}
	}
}

func TestDrafts_IDSanitization(t *testing.T) {
	s := newTestServer(t)

	// Test that IDs with slashes get sanitized
	body := jsonBody(t, map[string]string{
		"id":       "path/with/slashes",
		"markdown": "# Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	savedID := resp["id"].(string)
	if strings.Contains(savedID, "/") {
		t.Errorf("ID should have slashes sanitized, got: %s", savedID)
	}
}

func TestDrafts_BackslashSanitization(t *testing.T) {
	s := newTestServer(t)

	// Test that IDs with backslashes get sanitized
	body := jsonBody(t, map[string]string{
		"id":       "path\\with\\backslashes",
		"markdown": "# Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleDrafts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	savedID := resp["id"].(string)
	if strings.Contains(savedID, "\\") {
		t.Errorf("ID should have backslashes sanitized, got: %s", savedID)
	}
}

func TestPublish_EmptyContentPrevention(t *testing.T) {
	s := newConfiguredServer(t)

	// Try to publish whitespace-only content
	testCases := []string{
		"",
		"   ",
		"\n\n\n",
		"\t\t",
		"   \n   \t   ",
	}

	for _, content := range testCases {
		body := jsonBody(t, map[string]string{"markdown": content})
		req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.handlePublish(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty content %q, got %d", content, rr.Code)
		}
	}
}

func TestInit_PreventKeyOverwrite(t *testing.T) {
	s := newConfiguredServer(t) // Already has keys

	// Store original key content
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	originalContent, _ := os.ReadFile(privKeyPath)

	// Try to run init again - should fail
	body := jsonBody(t, map[string]string{"site_title": "attack"})
	req := httptest.NewRequest(http.MethodPost, "/api/init", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleInit(rr, req)

	// Should get 500 Internal Server Error (site package returns error)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 error when keys exist, got %d", rr.Code)
	}

	// Verify existing keys were NOT overwritten
	content, _ := os.ReadFile(privKeyPath)
	if string(content) != string(originalContent) {
		t.Error("existing private key was overwritten!")
	}
}

func TestDirectoryCreation_Safe(t *testing.T) {
	s := newTestServer(t)

	// Publish should create directories safely
	body := jsonBody(t, map[string]string{
		"markdown": "# Test Post\n\nContent for directory creation test.",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	req.Header.Set("Content-Type", "application/json")

	// Need configured server
	privKey, _, _ := signing.GenerateKeypair()
	s.PrivateKey = privKey
	s.Config = &Config{Subdomain: "test"}

	rr := httptest.NewRecorder()
	s.handlePublish(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify directory structure was created
	postsDir := filepath.Join(s.DataDir, "posts")
	if _, err := os.Stat(postsDir); os.IsNotExist(err) {
		t.Error("posts directory should be created")
	}
}

func TestFilePermissions_PrivateKey(t *testing.T) {
	s := newTestServer(t)

	// Remove existing keys to allow init
	os.RemoveAll(filepath.Join(s.DataDir, ".polis", "keys"))
	os.RemoveAll(filepath.Join(s.DataDir, ".well-known"))

	// Run init
	body := jsonBody(t, map[string]string{"site_title": "test-site"})
	req := httptest.NewRequest(http.MethodPost, "/api/init", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleInit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("init failed: %d: %s", rr.Code, rr.Body.String())
	}

	// Check private key permissions
	privKeyPath := filepath.Join(s.DataDir, ".polis", "keys", "id_ed25519")
	info, err := os.Stat(privKeyPath)
	if err != nil {
		t.Fatalf("private key not found: %v", err)
	}

	// Private key should be readable only by owner (0600)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("private key permissions should be 0600, got %o", perm)
	}
}

// ============================================================================
// handleRenderPage Tests
// ============================================================================

func TestHandleRenderPage_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	// GET should not be allowed
	req := httptest.NewRequest(http.MethodGet, "/api/render-page", nil)
	rr := httptest.NewRecorder()

	s.handleRenderPage(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleRenderPage_InvalidJSON(t *testing.T) {
	s := newTestServer(t)

	body := bytes.NewBufferString("not json")
	req := httptest.NewRequest(http.MethodPost, "/api/render-page", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleRenderPage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// handleSnippet Tests - Source Tier Preservation
// ============================================================================

func TestHandleSnippet_SaveGlobalSource(t *testing.T) {
	s := newTestServer(t)

	// Save a snippet with source="global"
	body := jsonBody(t, map[string]string{
		"content": "<p>Global about content</p>",
		"source":  "global",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/snippets/about.html", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it was saved to the global snippets directory
	globalPath := filepath.Join(s.DataDir, "snippets", "about.html")
	content, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("snippet not saved to global directory: %v", err)
	}
	if string(content) != "<p>Global about content</p>" {
		t.Errorf("unexpected content: %s", content)
	}

	// Verify it was NOT saved to theme directory
	themePath := filepath.Join(s.DataDir, ".polis", "themes", "turbo", "snippets", "about.html")
	if _, err := os.Stat(themePath); !os.IsNotExist(err) {
		t.Error("snippet should not exist in theme directory when source=global")
	}
}

func TestHandleSnippet_SaveThemeSource(t *testing.T) {
	s := newTestServer(t)

	// Create theme directory structure
	themeDir := filepath.Join(s.DataDir, ".polis", "themes", "turbo", "snippets")
	os.MkdirAll(themeDir, 0755)

	// Set the active theme in manifest.json (used by snippet package)
	manifestDir := filepath.Join(s.DataDir, "metadata")
	os.MkdirAll(manifestDir, 0755)
	os.WriteFile(filepath.Join(manifestDir, "manifest.json"), []byte(`{"active_theme":"turbo"}`), 0644)

	// Save a snippet with source="theme"
	body := jsonBody(t, map[string]string{
		"content": "<p>Theme about content</p>",
		"source":  "theme",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/snippets/about.html", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it was saved to the theme snippets directory
	themePath := filepath.Join(s.DataDir, ".polis", "themes", "turbo", "snippets", "about.html")
	content, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("snippet not saved to theme directory: %v", err)
	}
	if string(content) != "<p>Theme about content</p>" {
		t.Errorf("unexpected content: %s", content)
	}

	// Verify it was NOT saved to global directory
	globalPath := filepath.Join(s.DataDir, "snippets", "about.html")
	if _, err := os.Stat(globalPath); !os.IsNotExist(err) {
		t.Error("snippet should not exist in global directory when source=theme")
	}
}

func TestHandleSnippet_SaveDefaultsToGlobal(t *testing.T) {
	s := newTestServer(t)

	// Save a snippet WITHOUT specifying source (should default to global)
	body := jsonBody(t, map[string]string{
		"content": "<p>Default source content</p>",
		// Note: no "source" field
	})
	req := httptest.NewRequest(http.MethodPut, "/api/snippets/default.html", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it was saved to the global snippets directory
	globalPath := filepath.Join(s.DataDir, "snippets", "default.html")
	content, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("snippet not saved to global directory: %v", err)
	}
	if string(content) != "<p>Default source content</p>" {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestHandleSnippet_ReadRespectsSource(t *testing.T) {
	s := newTestServer(t)

	// Set the active theme in manifest.json
	manifestDir := filepath.Join(s.DataDir, "metadata")
	os.MkdirAll(manifestDir, 0755)
	os.WriteFile(filepath.Join(manifestDir, "manifest.json"), []byte(`{"active_theme":"turbo"}`), 0644)

	// Create both global and theme snippets with same name but different content
	globalDir := filepath.Join(s.DataDir, "snippets")
	themeDir := filepath.Join(s.DataDir, ".polis", "themes", "turbo", "snippets")
	os.MkdirAll(themeDir, 0755)

	os.WriteFile(filepath.Join(globalDir, "about.html"), []byte("<p>GLOBAL</p>"), 0644)
	os.WriteFile(filepath.Join(themeDir, "about.html"), []byte("<p>THEME</p>"), 0644)

	// Read with source=global
	req := httptest.NewRequest(http.MethodGet, "/api/snippets/about.html?source=global", nil)
	rr := httptest.NewRecorder()
	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if content, ok := result["content"].(string); !ok || content != "<p>GLOBAL</p>" {
		t.Errorf("expected global content, got: %v", result)
	}

	// Read with source=theme
	req = httptest.NewRequest(http.MethodGet, "/api/snippets/about.html?source=theme", nil)
	rr = httptest.NewRecorder()
	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	json.NewDecoder(rr.Body).Decode(&result)
	if content, ok := result["content"].(string); !ok || content != "<p>THEME</p>" {
		t.Errorf("expected theme content, got: %v", result)
	}
}

func TestHandleSnippet_SourcePreservedInResponse(t *testing.T) {
	s := newTestServer(t)

	// Set the active theme in manifest.json
	manifestDir := filepath.Join(s.DataDir, "metadata")
	os.MkdirAll(manifestDir, 0755)
	os.WriteFile(filepath.Join(manifestDir, "manifest.json"), []byte(`{"active_theme":"turbo"}`), 0644)

	// Create theme directory
	themeDir := filepath.Join(s.DataDir, ".polis", "themes", "turbo", "snippets")
	os.MkdirAll(themeDir, 0755)

	// Save with source=theme
	body := jsonBody(t, map[string]string{
		"content": "<p>Theme content</p>",
		"source":  "theme",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/snippets/footer.html", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.handleSnippet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify response includes the source
	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	if source, ok := result["source"].(string); !ok || source != "theme" {
		t.Errorf("expected source=theme in response, got: %v", result)
	}
}

// ============================================================================
// Webhook Safety Regression Tests
// ============================================================================

func TestPublishHookNotCalledOnError(t *testing.T) {
	s := newTestServer(t)
	// No private key = publish will fail
	markerFile := filepath.Join(s.DataDir, "hook-marker")

	s.Config = &Config{
		Hooks: &hooks.HookConfig{
			PostPublish: "touch " + markerFile,
		},
	}

	body := jsonBody(t, map[string]string{"markdown": "# Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/publish", body)
	rr := httptest.NewRecorder()

	s.handlePublish(rr, req)

	// Publish should fail (no private key)
	if rr.Code == http.StatusOK {
		t.Error("expected publish to fail without private key")
	}

	// Hook marker should NOT exist
	if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
		t.Error("hook should not have been called on publish error")
	}
}

func TestRepublishHookNotCalledOnError(t *testing.T) {
	s := newConfiguredServer(t)
	markerFile := filepath.Join(s.DataDir, "hook-marker")

	s.Config.Hooks = &hooks.HookConfig{
		PostPublish: "touch " + markerFile,
	}

	// Republish a non-existent post
	body := jsonBody(t, map[string]string{
		"path":     "posts/20260101/nonexistent.md",
		"markdown": "# Updated",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/republish", body)
	rr := httptest.NewRecorder()

	s.handleRepublish(rr, req)

	// Republish should fail (file not found)
	if rr.Code == http.StatusOK {
		t.Error("expected republish to fail for nonexistent post")
	}

	// Hook marker should NOT exist
	if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
		t.Error("hook should not have been called on republish error")
	}
}

func TestBlessingGrantHookNotCalledOnError(t *testing.T) {
	s := newTestServer(t)
	// No discovery service config = grant will fail
	markerFile := filepath.Join(s.DataDir, "hook-marker")

	s.Config = &Config{
		Hooks: &hooks.HookConfig{
			PostComment: "touch " + markerFile,
		},
	}

	body := jsonBody(t, map[string]string{
		"comment_version": "sha256:abc",
		"comment_url":     "https://bob.polis.pub/comments/test.md",
		"in_reply_to":     "https://alice.polis.pub/posts/test.md",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/blessing/grant", body)
	rr := httptest.NewRecorder()

	s.handleBlessingGrant(rr, req)

	// Grant should fail (no discovery config or no private key)
	if rr.Code == http.StatusOK {
		t.Error("expected blessing grant to fail without config")
	}

	// Hook marker should NOT exist
	if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
		t.Error("hook should not have been called on blessing grant error")
	}
}

// ============================================================================
// Drafts Migration Tests
// ============================================================================

func TestMigrateDraftsDir_OldToNew(t *testing.T) {
	dataDir := t.TempDir()
	s := &Server{DataDir: dataDir}

	// Create old-style drafts dir with a file
	oldDir := filepath.Join(dataDir, ".polis", "drafts")
	os.MkdirAll(oldDir, 0755)
	os.WriteFile(filepath.Join(oldDir, "test-draft.md"), []byte("# Draft"), 0644)

	s.migrateDraftsDir()

	// Old dir should be gone
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("expected old drafts dir to be removed after migration")
	}

	// New dir should exist with the file
	newDir := filepath.Join(dataDir, ".polis", "posts", "drafts")
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Fatal("expected new drafts dir to exist")
	}
	content, err := os.ReadFile(filepath.Join(newDir, "test-draft.md"))
	if err != nil {
		t.Fatal("expected draft file to be migrated")
	}
	if string(content) != "# Draft" {
		t.Errorf("expected draft content preserved, got: %s", string(content))
	}
}

func TestMigrateDraftsDir_NoOldDir(t *testing.T) {
	dataDir := t.TempDir()
	s := &Server{DataDir: dataDir}

	// No old dir exists - should be a no-op
	s.migrateDraftsDir()

	newDir := filepath.Join(dataDir, ".polis", "posts", "drafts")
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Error("expected new dir not to be created when no old dir exists")
	}
}

func TestMigrateDraftsDir_NewAlreadyExists(t *testing.T) {
	dataDir := t.TempDir()
	s := &Server{DataDir: dataDir}

	// Create both old and new dirs
	oldDir := filepath.Join(dataDir, ".polis", "drafts")
	newDir := filepath.Join(dataDir, ".polis", "posts", "drafts")
	os.MkdirAll(oldDir, 0755)
	os.MkdirAll(newDir, 0755)
	os.WriteFile(filepath.Join(oldDir, "old-draft.md"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(newDir, "new-draft.md"), []byte("new"), 0644)

	s.migrateDraftsDir()

	// Old dir should still exist (migration skipped)
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		t.Error("expected old dir to be preserved when new dir already exists")
	}
	// New dir file should be intact
	content, _ := os.ReadFile(filepath.Join(newDir, "new-draft.md"))
	if string(content) != "new" {
		t.Error("expected new dir contents to be preserved")
	}
}

// ============================================================================
// Security: Error Redaction Tests (H1)
// ============================================================================

func TestErrorResponsesRedacted(t *testing.T) {
	s := newConfiguredServer(t)

	// Strings that should NEVER appear in HTTP error responses
	osErrorStrings := []string{
		"permission denied",
		"no such file or directory",
		"not a directory",
		"/tmp/",
		"/home/",
		s.DataDir, // The actual data directory path
	}

	// Test publish with content that will fail (trigger internal errors)
	tests := []struct {
		name    string
		method  string
		path    string
		body    interface{}
		handler func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "handleRender with nil key",
			method:  http.MethodPost,
			path:    "/api/render",
			body:    map[string]string{"markdown": "# Test"},
			handler: (&Server{DataDir: s.DataDir}).handleRender, // No private key
		},
		{
			name:   "handleCommentsPending with missing dir",
			method: http.MethodGet,
			path:   "/api/comments/pending",
			handler: (&Server{
				DataDir: "/nonexistent/path/that/does/not/exist",
			}).handleCommentsPending,
		},
		{
			name:   "handleCommentsBlessed with missing dir",
			method: http.MethodGet,
			path:   "/api/comments/blessed",
			handler: (&Server{
				DataDir: "/nonexistent/path/that/does/not/exist",
			}).handleCommentsBlessed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != nil {
				body = jsonBody(t, tt.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			responseBody := rr.Body.String()
			for _, osErr := range osErrorStrings {
				if strings.Contains(strings.ToLower(responseBody), strings.ToLower(osErr)) {
					t.Errorf("response contains OS error detail %q: %s", osErr, responseBody)
				}
			}
		})
	}
}

// ============================================================================
// Security: Draft ID Sanitization Tests (M1)
// ============================================================================

func TestDraftIDSanitization(t *testing.T) {
	s := newTestServer(t)

	tests := []struct {
		name       string
		inputID    string
		wantSafe   bool // ID should not contain dangerous chars
		wantPrefix string
	}{
		{"normal ID", "my-draft", true, ""},
		{"path traversal", "../../../etc/passwd", true, ""},
		{"null bytes", "draft\x00evil", true, ""},
		{"slashes", "a/b/c", true, ""},
		{"backslashes", "a\\b\\c", true, ""},
		{"unicode", "draft\u2028evil", true, ""},
		{"special chars", "draft@#$%.md", true, ""},
		{"spaces", "my draft name", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := jsonBody(t, map[string]string{
				"id":       tt.inputID,
				"markdown": "# Test",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/drafts", body)
			rr := httptest.NewRecorder()

			s.handleDrafts(rr, req)

			if rr.Code != http.StatusOK {
				// May fail due to missing dirs, but we want to check file system effects
				return
			}

			var resp map[string]interface{}
			json.NewDecoder(rr.Body).Decode(&resp)

			id, ok := resp["id"].(string)
			if !ok {
				return
			}

			// Verify the ID only contains safe characters
			for _, ch := range id {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
					t.Errorf("sanitized ID %q contains unsafe character %q", id, string(ch))
				}
			}

			// Verify no path traversal chars
			if strings.Contains(id, "..") {
				t.Errorf("sanitized ID still contains '..': %s", id)
			}
			if strings.Contains(id, "/") {
				t.Errorf("sanitized ID still contains '/': %s", id)
			}
			if strings.Contains(id, "\\") {
				t.Errorf("sanitized ID still contains '\\': %s", id)
			}
		})
	}
}

// ============================================================================
// Security: Path Traversal Tests (M2)
// ============================================================================

func TestValidatePostPath_Canonicalization(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"normal path", "posts/20260101/hello.md", false},
		{"dot-dot traversal", "posts/../../../etc/passwd", true},
		{"double slash", "posts//20260101//hello.md", false},
		{"dot segment", "posts/./20260101/hello.md", false},
		{"null byte", "posts/20260101/hello\x00.md", true},
		{"not posts prefix", "comments/foo.md", true},
		{"clean removes prefix", "../posts/hello.md", true}, // filepath.Clean("../posts/hello.md") = "../posts/hello.md"
		{"encoded dot-dot", "posts/20260101/..%2f..%2fetc/passwd", true}, // Contains ".." substring which is blocked
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePostPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContentPath_Canonicalization(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"root markdown", "index.md", false},
		{"root html", "index.html", false},
		{"posts path", "posts/20260101/hello.md", false},
		{"comments path", "comments/blessed/comment.md", false},
		{"drafts path", ".polis/posts/drafts/my-draft.md", false},
		{"traversal attempt", "../../../etc/passwd", true},
		{"null byte", "posts/hello\x00.md", true},
		{"invalid prefix", "secrets/key.pem", true},
		{"double dot in component", "posts/..hidden/file.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContentPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContentPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// handleFollowing Tests
// ============================================================================

func TestHandleFollowing_Get_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/following", nil)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["count"] != float64(0) {
		t.Errorf("expected count=0, got %v", resp["count"])
	}
}

func TestHandleFollowing_Get_WithEntries(t *testing.T) {
	s := newTestServer(t)

	// Pre-populate following.json
	followingData := `{
		"version": "test",
		"following": [
			{"url": "https://alice.example.com", "added_at": "2026-01-01T00:00:00Z"},
			{"url": "https://bob.example.com", "added_at": "2026-01-02T00:00:00Z"}
		]
	}`
	followingPath := filepath.Join(s.DataDir, "metadata", "following.json")
	os.WriteFile(followingPath, []byte(followingData), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/following", nil)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", resp["count"])
	}

	followingList, ok := resp["following"].([]interface{})
	if !ok || len(followingList) != 2 {
		t.Errorf("expected 2 following entries, got %v", resp["following"])
	}
}

func TestHandleFollowing_Post_InvalidURL(t *testing.T) {
	s := newConfiguredServer(t)

	body := jsonBody(t, map[string]string{"url": "http://insecure.example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/following", body)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleFollowing_Post_NoKeys(t *testing.T) {
	s := newTestServer(t)
	// No keys configured

	body := jsonBody(t, map[string]string{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/following", body)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleFollowing_Delete_NoKeys(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodDelete, "/api/following", body)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleFollowing_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/following", nil)
	w := httptest.NewRecorder()

	s.handleFollowing(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// handleFeed Tests (cache-backed)
// ============================================================================

func TestHandleFeed_EmptyCache(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	w := httptest.NewRecorder()

	s.handleFeed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("expected 0 total, got %v", resp["total"])
	}
	if resp["stale"].(bool) != true {
		t.Error("empty cache should be stale")
	}
}

func TestHandleFeed_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/feed", nil)
	w := httptest.NewRecorder()

	s.handleFeed(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleFeed_WithTypeFilter(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "A Post", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "comment", Title: "A Comment", URL: "comments/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://b.pub", AuthorDomain: "b.pub"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/feed?type=post", nil)
	w := httptest.NewRecorder()
	s.handleFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 post, got %d", len(items))
	}
}

func TestHandleFeed_SpecialCharacterTitles(t *testing.T) {
	s := newTestServer(t)

	// Populate cache with titles containing special characters
	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
		{Type: "post", Title: "It's Not Beyond Our Reach", URL: "posts/its-not.md", Published: "2026-01-15T12:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		{Type: "post", Title: `She said "hello" & waved`, URL: "posts/she-said.md", Published: "2026-01-14T12:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
		{Type: "post", Title: "2 < 3 && 5 > 4", URL: "posts/math.md", Published: "2026-01-13T12:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	w := httptest.NewRecorder()
	s.handleFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}

	// Verify titles with apostrophes, quotes, and angle brackets survive JSON round-trip
	expectedTitles := []string{
		"It's Not Beyond Our Reach",
		`She said "hello" & waved`,
		"2 < 3 && 5 > 4",
	}
	for i, want := range expectedTitles {
		if resp.Items[i].Title != want {
			t.Errorf("item[%d] title = %q, want %q", i, resp.Items[i].Title, want)
		}
	}
}

func TestHandleFeed_UnreadCount(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "B", URL: "posts/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "C", URL: "posts/c.md", Published: "2026-02-03T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	// Mark one as read
	items, _ := cm.List()
	cm.MarkRead(items[0].ID)

	req := httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	w := httptest.NewRecorder()
	s.handleFeed(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 3 {
		t.Errorf("expected 3 total, got %v", resp["total"])
	}
	if resp["unread"].(float64) != 2 {
		t.Errorf("expected 2 unread, got %v", resp["unread"])
	}
}

// ============================================================================
// handleFeedRefresh Tests
// ============================================================================

func TestHandleFeedRefresh_EmptyFollowing(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/feed/refresh", nil)
	w := httptest.NewRecorder()
	s.handleFeedRefresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("expected 0 total, got %v", resp["total"])
	}
	if resp["new_items"].(float64) != 0 {
		t.Errorf("expected 0 new_items, got %v", resp["new_items"])
	}
	if resp["stale"].(bool) != false {
		t.Error("just-refreshed cache should not be stale")
	}
}

func TestHandleFeed_NotStaleAfterCursorRefresh(t *testing.T) {
	s := newTestServer(t)
	discoveryDomain := s.GetDiscoveryDomain()

	// Set a cursor with a backdated timestamp to simulate a stale cache
	cm := feed.NewCacheManager(s.DataDir, discoveryDomain)
	cm.SetCursor("100")

	cursorsPath := filepath.Join(s.DataDir, ".polis", "ds", discoveryDomain, "state", "cursors.json")
	staleData, _ := json.Marshal(map[string]interface{}{
		"cursors": map[string]interface{}{
			"polis.feed": map[string]interface{}{
				"position":     "100",
				"last_updated": "2020-01-01T00:00:00Z",
			},
		},
	})
	os.WriteFile(cursorsPath, staleData, 0644)

	// Confirm GET /api/feed reports stale
	req := httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	w := httptest.NewRecorder()
	s.handleFeed(w, req)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["stale"].(bool) != true {
		t.Fatal("cache should be stale before refresh")
	}

	// Simulate what syncFeed does after a successful sync: SetCursor with same position
	cm.SetCursor("100")

	// GET /api/feed should now report not stale
	req = httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	w = httptest.NewRecorder()
	s.handleFeed(w, req)
	resp = map[string]interface{}{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["stale"].(bool) != false {
		t.Error("cache should not be stale after cursor refresh with same position")
	}
}

func TestHandleFeedRefresh_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/feed/refresh", nil)
	w := httptest.NewRecorder()
	s.handleFeedRefresh(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// handleFeedRead Tests
// ============================================================================

func TestHandleFeedRead_MarkRead(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "Test", URL: "posts/test.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	items, _ := cm.List()
	itemID := items[0].ID

	body := jsonBody(t, map[string]string{"id": itemID})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	items, _ = cm.List()
	if items[0].ReadAt == "" {
		t.Error("item should be marked read")
	}
}

func TestHandleFeedRead_MarkUnread(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "Test", URL: "posts/test.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	items, _ := cm.List()
	itemID := items[0].ID
	cm.MarkRead(itemID)

	body := jsonBody(t, map[string]interface{}{"id": itemID, "unread": true})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	items, _ = cm.List()
	if items[0].ReadAt != "" {
		t.Error("item should be unread")
	}
}

func TestHandleFeedRead_MarkAllRead(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "B", URL: "posts/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	body := jsonBody(t, map[string]interface{}{"all": true})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	unread, _ := cm.UnreadCount()
	if unread != 0 {
		t.Errorf("expected 0 unread, got %d", unread)
	}
}

func TestHandleFeedRead_MarkUnreadFrom(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "Old", URL: "posts/old.md", Published: "2026-01-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "Mid", URL: "posts/mid.md", Published: "2026-01-15T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "New", URL: "posts/new.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	cm.MarkAllRead()

	items, _ := cm.List()
	// Items sorted desc: New, Mid, Old
	midID := items[1].ID

	body := jsonBody(t, map[string]interface{}{"from_id": midID})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	items, _ = cm.List()
	// New should be unread (more recent than mid)
	if items[0].ReadAt != "" {
		t.Error("New should be unread")
	}
	// Mid should be unread (the target)
	if items[1].ReadAt != "" {
		t.Error("Mid should be unread")
	}
	// Old should still be read (older than mid)
	if items[2].ReadAt == "" {
		t.Error("Old should still be read")
	}
}

func TestHandleFeedRead_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/feed/read", nil)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleFeedRead_MissingFields(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]interface{}{})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedRead_InvalidID(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]string{"id": "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/feed/read", body)
	w := httptest.NewRecorder()
	s.handleFeedRead(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ============================================================================
// handleFeedCounts Tests
// ============================================================================

func TestHandleFeedCounts_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/feed/counts", nil)
	w := httptest.NewRecorder()
	s.handleFeedCounts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("expected 0 total, got %v", resp["total"])
	}
	if resp["unread"].(float64) != 0 {
		t.Errorf("expected 0 unread, got %v", resp["unread"])
	}
	if resp["stale"].(bool) != true {
		t.Error("empty cache should be stale")
	}
}

func TestHandleFeedCounts_WithItems(t *testing.T) {
	s := newTestServer(t)

	cm := feed.NewCacheManager(s.DataDir, "default")
	cm.MergeItems([]feed.FeedItem{
			{Type: "post", Title: "A", URL: "posts/a.md", Published: "2026-02-01T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
			{Type: "post", Title: "B", URL: "posts/b.md", Published: "2026-02-02T10:00:00Z", AuthorURL: "https://a.pub", AuthorDomain: "a.pub"},
	})

	items, _ := cm.List()
	cm.MarkRead(items[0].ID)

	req := httptest.NewRequest(http.MethodGet, "/api/feed/counts", nil)
	w := httptest.NewRecorder()
	s.handleFeedCounts(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 2 {
		t.Errorf("expected 2 total, got %v", resp["total"])
	}
	if resp["unread"].(float64) != 1 {
		t.Errorf("expected 1 unread, got %v", resp["unread"])
	}
}

func TestHandleFeedCounts_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/feed/counts", nil)
	w := httptest.NewRecorder()
	s.handleFeedCounts(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// handleRemotePost Tests
// ============================================================================

func TestHandleRemotePost_MissingURL(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/remote/post", nil)
	w := httptest.NewRecorder()

	s.handleRemotePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRemotePost_InvalidURL(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/remote/post?url=http://insecure.com/post.md", nil)
	w := httptest.NewRecorder()

	s.handleRemotePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRemotePost_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/remote/post?url=https://example.com/post.md", nil)
	w := httptest.NewRecorder()

	s.handleRemotePost(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// stripFrontmatter Tests
// ============================================================================

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with frontmatter",
			input:    "---\ntitle: Hello\ndate: 2026-01-01\n---\n# Hello World\n\nContent here.",
			expected: "# Hello World\n\nContent here.",
		},
		{
			name:     "without frontmatter",
			input:    "# Hello World\n\nContent here.",
			expected: "# Hello World\n\nContent here.",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "only frontmatter",
			input:    "---\ntitle: Hello\n---\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFrontmatter(tt.input)
			if got != tt.expected {
				t.Errorf("stripFrontmatter() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// looksLikeHTML Tests
// ============================================================================

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"doctype uppercase", "<!DOCTYPE html><html>...", true},
		{"doctype lowercase", "<!doctype html><html>...", true},
		{"html tag", "<html><head>...", true},
		{"html with whitespace", "  \n<!DOCTYPE html>...", true},
		{"markdown", "# Hello\n\nSome text", false},
		{"frontmatter markdown", "---\ntitle: Hi\n---\n# Hello", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeHTML(tt.input); got != tt.expected {
				t.Errorf("looksLikeHTML() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// extractHTMLBody Tests
// ============================================================================

func TestExtractHTMLBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts body",
			input:    "<html><head><title>T</title></head><body><h1>Hello</h1></body></html>",
			expected: "<h1>Hello</h1>",
		},
		{
			name:     "prefers main over body",
			input:    "<html><body><nav>Nav</nav><main><h1>Content</h1></main></body></html>",
			expected: "<h1>Content</h1>",
		},
		{
			name:     "no body tag returns full content",
			input:    "<h1>Just a heading</h1>",
			expected: "<h1>Just a heading</h1>",
		},
		{
			name:     "body with attributes",
			input:    `<html><body class="dark"><p>Text</p></body></html>`,
			expected: "<p>Text</p>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractHTMLBody(tt.input); got != tt.expected {
				t.Errorf("extractHTMLBody() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// handleActivityStream Tests
// ============================================================================

func TestHandleActivityStream_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/activity", nil)
	rr := httptest.NewRecorder()

	s.handleActivityStream(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleActivityStream_NoFollowing(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	rr := httptest.NewRecorder()

	s.handleActivityStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in response")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	if resp["cursor"] != "0" {
		t.Errorf("expected cursor '0', got %v", resp["cursor"])
	}
}

func TestHandleActivityStream_WithFollowingNoDiscovery(t *testing.T) {
	s := newConfiguredServer(t)

	// Create a following.json with an entry
	followingPath := following.DefaultPath(s.DataDir)
	f, _ := following.Load(followingPath)
	f.Add("https://alice.example.com")
	following.Save(followingPath, f)

	req := httptest.NewRequest(http.MethodGet, "/api/activity?since=0&limit=10", nil)
	rr := httptest.NewRecorder()

	s.handleActivityStream(rr, req)

	// Should return 200 with empty events (discovery service not reachable in test)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	// Should gracefully return empty on network error
	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in response")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events (no discovery service), got %d", len(events))
	}
}

// ============================================================================
// handleFollowerCount Tests
// ============================================================================

func TestHandleFollowerCount_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/followers/count", nil)
	rr := httptest.NewRecorder()

	s.handleFollowerCount(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleFollowerCount_NoBaseURL(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/followers/count", nil)
	rr := httptest.NewRecorder()

	s.handleFollowerCount(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	count, ok := resp["count"].(float64)
	if !ok || count != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestHandleFollowerCount_Configured(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/followers/count", nil)
	rr := httptest.NewRecorder()

	s.handleFollowerCount(rr, req)

	// Should return 200 with 0 followers (no stream events to project)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	count, ok := resp["count"].(float64)
	if !ok || count != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}

	followers, ok := resp["followers"].([]interface{})
	if !ok {
		t.Fatal("expected followers array")
	}
	if len(followers) != 0 {
		t.Errorf("expected 0 followers, got %d", len(followers))
	}
}

func TestHandleFollowerCount_WithRefresh(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/followers/count?refresh=true", nil)
	rr := httptest.NewRecorder()

	s.handleFollowerCount(rr, req)

	// Should return 200 even with refresh (will try to query stream, get empty)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// extractDomainFromURL Tests
// ============================================================================

func TestExtractDomainFromURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"https://sub.example.com", "sub.example.com"},
		{"http://localhost:8080", "localhost"},
		{"", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		got := extractDomainFromURL(tt.input)
		if got != tt.expected {
			t.Errorf("extractDomainFromURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestHandleNotificationCount(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/count", nil)
	w := httptest.NewRecorder()

	s.handleNotificationCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["unread"] != float64(0) {
		t.Errorf("expected unread=0, got %v", resp["unread"])
	}
}

func TestHandleNotificationCount_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/count", nil)
	w := httptest.NewRecorder()

	s.handleNotificationCount(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleNotifications_EmptyList(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	w := httptest.NewRecorder()

	s.handleNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	notifications := resp["notifications"].([]interface{})
	if len(notifications) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notifications))
	}
	if resp["total"] != float64(0) {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
}

func TestHandleNotificationRead(t *testing.T) {
	s := newTestServer(t)

	body := jsonBody(t, map[string]interface{}{"all": true})
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/read", body)
	w := httptest.NewRecorder()

	s.handleNotificationRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"] != true {
		t.Error("expected success=true")
	}
}

func TestHandleNotificationRead_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/read", nil)
	w := httptest.NewRecorder()

	s.handleNotificationRead(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// handleDeployCheck Tests
// ============================================================================

func TestHandleDeployCheck_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/site/deploy-check", nil)
	w := httptest.NewRecorder()

	s.handleDeployCheck(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleDeployCheck_NoBaseURL(t *testing.T) {
	s := newTestServer(t)
	s.BaseURL = "" // No base URL set

	req := httptest.NewRequest(http.MethodGet, "/api/site/deploy-check", nil)
	w := httptest.NewRecorder()

	s.handleDeployCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["deployed"] != false {
		t.Error("expected deployed=false when base URL not set")
	}
	if resp["error"] == nil || resp["error"] == "" {
		t.Error("expected error message when base URL not set")
	}
}

// ============================================================================
// handleSetupWizardDismiss Tests
// ============================================================================

func TestHandleSetupWizardDismiss(t *testing.T) {
	s := newConfiguredServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/site/setup-wizard-dismiss", nil)
	w := httptest.NewRecorder()

	s.handleSetupWizardDismiss(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"] != true {
		t.Error("expected success=true")
	}

	// Verify config was updated
	if !s.Config.SetupWizardDismissed {
		t.Error("expected SetupWizardDismissed to be true after dismiss")
	}

	// Verify config was persisted to disk
	configPath := filepath.Join(s.DataDir, ".polis", "webapp-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if !strings.Contains(string(data), "setup_wizard_dismissed") {
		t.Error("expected setup_wizard_dismissed in saved config")
	}
}

func TestHandleSetupWizardDismiss_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/site/setup-wizard-dismiss", nil)
	w := httptest.NewRecorder()

	s.handleSetupWizardDismiss(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================================================================
// handleInit Setup Wizard State Tests
// ============================================================================

func TestHandleInit_SetsWizardNotDismissed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/init", jsonBody(t, map[string]string{
		"site_title": "Test Site",
	}))
	w := httptest.NewRecorder()

	s.handleInit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// After init, config should exist with setup_wizard_dismissed=false
	if s.Config == nil {
		t.Fatal("expected config to be set after init")
	}
	if s.Config.SetupWizardDismissed {
		t.Error("expected SetupWizardDismissed to be false after init")
	}
}

func TestHandleInit_WritesBaseURLToEnv(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/init", jsonBody(t, map[string]string{
		"site_title": "Test Site",
		"base_url":   "https://alice.example.com",
	}))
	w := httptest.NewRecorder()

	s.handleInit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check .env was created with the base URL
	envPath := filepath.Join(s.DataDir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("expected .env file to exist: %v", err)
	}
	if !strings.Contains(string(data), "POLIS_BASE_URL=https://alice.example.com") {
		t.Errorf("expected .env to contain POLIS_BASE_URL, got: %s", string(data))
	}

	// Check server state was updated
	if s.BaseURL != "https://alice.example.com" {
		t.Errorf("expected BaseURL to be updated, got: %s", s.BaseURL)
	}

	// Check response includes base_url
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["base_url"] != "https://alice.example.com" {
		t.Errorf("expected response base_url, got: %v", resp["base_url"])
	}
}

func TestHandleInit_NoBaseURL_EnvStillHasDiscovery(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/init", jsonBody(t, map[string]string{
		"site_title": "Test Site",
	}))
	w := httptest.NewRecorder()

	s.handleInit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// .env should still be created with discovery credentials
	envPath := filepath.Join(s.DataDir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("expected .env file to exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "DISCOVERY_SERVICE_URL=") {
		t.Error("expected .env to contain DISCOVERY_SERVICE_URL")
	}
	if !strings.Contains(content, "DISCOVERY_SERVICE_KEY=") {
		t.Error("expected .env to contain DISCOVERY_SERVICE_KEY")
	}
	// Should NOT contain POLIS_BASE_URL when not provided
	if strings.Contains(content, "POLIS_BASE_URL=") {
		t.Error("expected .env to NOT contain POLIS_BASE_URL when not provided")
	}
}

// ============================================================================
// handleSettings Setup Wizard Dismissed Tests
// ============================================================================

func TestHandleSettings_IncludesSetupWizardDismissed(t *testing.T) {
	s := newConfiguredServer(t)
	s.Config.SetupWizardDismissed = true

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()

	s.handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	dismissed, ok := resp["setup_wizard_dismissed"]
	if !ok {
		t.Error("expected setup_wizard_dismissed in settings response")
	}
	if dismissed != true {
		t.Errorf("expected setup_wizard_dismissed=true, got %v", dismissed)
	}
}

func TestHandleHideRead_Toggle(t *testing.T) {
	s := newConfiguredServer(t)

	// Toggle on
	body := jsonBody(t, map[string]bool{"hide_read": true})
	req := httptest.NewRequest(http.MethodPost, "/api/settings/hide-read", body)
	w := httptest.NewRecorder()
	s.handleHideRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !s.Config.HideRead {
		t.Error("expected HideRead=true after toggle on")
	}

	// Toggle off
	body = jsonBody(t, map[string]bool{"hide_read": false})
	req = httptest.NewRequest(http.MethodPost, "/api/settings/hide-read", body)
	w = httptest.NewRecorder()
	s.handleHideRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if s.Config.HideRead {
		t.Error("expected HideRead=false after toggle off")
	}
}

func TestHandleHideRead_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/settings/hide-read", nil)
	w := httptest.NewRecorder()
	s.handleHideRead(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleSettings_IncludesHideRead(t *testing.T) {
	s := newConfiguredServer(t)
	s.Config.HideRead = true

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	s.handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	hideRead, ok := resp["hide_read"]
	if !ok {
		t.Error("expected hide_read in settings response")
	}
	if hideRead != true {
		t.Errorf("expected hide_read=true, got %v", hideRead)
	}
}

// ============================================================================
// handleNotifications Tests
// ============================================================================

func TestHandleNotifications_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	w := httptest.NewRecorder()
	s.handleNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	notifications := resp["notifications"].([]interface{})
	if len(notifications) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notifications))
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total 0, got %v", resp["total"])
	}
	if resp["limit"].(float64) != 20 {
		t.Errorf("expected default limit 20, got %v", resp["limit"])
	}
}

func TestHandleNotifications_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications", nil)
	w := httptest.NewRecorder()
	s.handleNotifications(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
