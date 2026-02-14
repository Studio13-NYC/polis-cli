// Package discovery provides a client for the polis discovery service.
package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// Client is an HTTP client for the discovery service.
type Client struct {
	BaseURL       string
	APIKey        string
	Domain        string // optional: for signed GET requests
	PrivateKeyPEM []byte // optional: for signed GET requests
	HTTPClient    *http.Client
}

// NewClient creates a new discovery service client (unauthenticated GET requests).
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewAuthenticatedClient creates a discovery client that signs GET requests
// with domain ownership proof via X-Polis-Domain/Signature/Timestamp headers.
func NewAuthenticatedClient(baseURL, apiKey, domain string, privateKeyPEM []byte) *Client {
	return &Client{
		BaseURL:       baseURL,
		APIKey:        apiKey,
		Domain:        domain,
		PrivateKeyPEM: privateKeyPEM,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// queryAuthPayload is the canonical payload for signed GET request authentication.
// Field order is critical — must match the TS side's buildQueryAuthCanonicalJSON.
type queryAuthPayload struct {
	Action    string `json:"action"`
	Domain    string `json:"domain"`
	Timestamp string `json:"timestamp"`
}

// MakeQueryAuthCanonicalJSON creates the canonical JSON for signed GET auth.
// Must produce identical output to the TS side's buildQueryAuthCanonicalJSON.
func MakeQueryAuthCanonicalJSON(domain, timestamp string) ([]byte, error) {
	return json.Marshal(queryAuthPayload{
		Action:    "query",
		Domain:    domain,
		Timestamp: timestamp,
	})
}

// addAuthHeaders adds X-Polis-Domain, X-Polis-Signature, X-Polis-Timestamp
// headers to a request if the client has auth credentials configured.
// No-op if Domain/PrivateKeyPEM are empty (backward compatible).
func (c *Client) addAuthHeaders(req *http.Request) error {
	if c.Domain == "" || len(c.PrivateKeyPEM) == 0 {
		return nil
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	canonicalJSON, err := MakeQueryAuthCanonicalJSON(c.Domain, timestamp)
	if err != nil {
		return fmt.Errorf("failed to build auth canonical payload: %w", err)
	}

	signature, err := signing.SignContent(canonicalJSON, c.PrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to sign auth payload: %w", err)
	}

	// SSH signatures contain newlines (PEM format), which are invalid in
	// HTTP headers. Strip them — the TS parser already strips whitespace
	// via .replace(/\s/g, '') before extracting the base64 payload.
	compactSig := strings.ReplaceAll(signature, "\n", "")

	req.Header.Set("X-Polis-Domain", c.Domain)
	req.Header.Set("X-Polis-Signature", compactSig)
	req.Header.Set("X-Polis-Timestamp", timestamp)

	return nil
}

// ============================================================================
// Unified Content Types
// ============================================================================

// ContentRegisterRequest is the unified content registration request.
type ContentRegisterRequest struct {
	Type      string                 `json:"type"`
	URL       string                 `json:"url"`
	Version   string                 `json:"version"`
	Author    string                 `json:"author"`
	Metadata  map[string]interface{} `json:"metadata"`
	Signature string                 `json:"signature"`
}

// ContentRegisterResponse is the response from content-register.
type ContentRegisterResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message,omitempty"`
	Type               string `json:"type"`
	URL                string `json:"url"`
	Status             string `json:"status"` // "created" or "updated"
	RelationshipStatus string `json:"relationship_status,omitempty"`
	Error              string `json:"error,omitempty"`
}

// ContentUnregisterRequest is the request to unregister content.
type ContentUnregisterRequest struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// ContentCheckResponse is the response from content-check.
type ContentCheckResponse struct {
	Exists  bool   `json:"exists"`
	Type    string `json:"type"`
	URL     string `json:"url"`
	Version string `json:"version,omitempty"`
	Actor   string `json:"actor,omitempty"`
}

// ContentRecord represents a content item from the discovery service.
type ContentRecord struct {
	ID        int64                  `json:"id"`
	Type      string                 `json:"type"`
	URL       string                 `json:"url"`
	Version   string                 `json:"version"`
	Actor     string                 `json:"actor"`
	Author    string                 `json:"author"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// ContentQueryResponse is the response from content-query.
type ContentQueryResponse struct {
	Count   int             `json:"count"`
	Records []ContentRecord `json:"records"`
}

// ============================================================================
// Unified Relationship Types
// ============================================================================

// RelationshipUpdateRequest updates a relationship status.
type RelationshipUpdateRequest struct {
	Type      string `json:"type"`
	SourceURL string `json:"source_url"`
	TargetURL string `json:"target_url"`
	Action    string `json:"action"` // "grant" or "deny"
	Signature string `json:"signature"`
}

// RelationshipRecord represents a relationship from the discovery service.
type RelationshipRecord struct {
	ID        int64                  `json:"id"`
	Type      string                 `json:"type"`
	SourceURL string                 `json:"source_url"`
	TargetURL string                 `json:"target_url"`
	Actor     string                 `json:"actor"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// RelationshipQueryResponse is the response from relationship-query.
type RelationshipQueryResponse struct {
	Count   int                  `json:"count"`
	Records []RelationshipRecord `json:"records"`
}

// ============================================================================
// Content Methods
// ============================================================================

// RegisterContent registers or updates content with the discovery service.
func (c *Client) RegisterContent(req *ContentRegisterRequest) (*ContentRegisterResponse, error) {
	endpoint := c.BaseURL + "/ds-content-register"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result ContentRegisterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Error != "" {
			return &result, fmt.Errorf("content registration failed: %s", result.Error)
		}
		return &result, fmt.Errorf("content registration failed with status %d", resp.StatusCode)
	}

	return &result, nil
}

// UnregisterContent removes content from the discovery service.
func (c *Client) UnregisterContent(contentType, contentURL, signature string) error {
	endpoint := c.BaseURL + "/ds-content-unregister"

	req := ContentUnregisterRequest{
		Type:      contentType,
		URL:       contentURL,
		Signature: signature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("content unregistration failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CheckContent checks if content exists in the discovery service.
func (c *Client) CheckContent(contentType, contentURL string) (*ContentCheckResponse, error) {
	params := url.Values{}
	params.Set("type", contentType)
	params.Set("url", contentURL)
	endpoint := c.BaseURL + "/ds-content-check?" + params.Encode()

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("content check failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ContentCheckResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// QueryContent queries content by type and optional filters.
func (c *Client) QueryContent(contentType string, filters map[string]string) (*ContentQueryResponse, error) {
	params := url.Values{}
	params.Set("type", contentType)
	for k, v := range filters {
		params.Set(k, v)
	}
	endpoint := c.BaseURL + "/ds-content-query?" + params.Encode()

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	if err := c.addAuthHeaders(httpReq); err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("content query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ContentQueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ============================================================================
// Relationship Methods
// ============================================================================

// UpdateRelationship updates a relationship status (grant/deny blessing).
// The privateKey is used to sign the request payload.
func (c *Client) UpdateRelationship(relType, sourceURL, targetURL, action string, privateKey []byte) error {
	endpoint := c.BaseURL + "/ds-relationship-update"

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Create canonical payload for signing
	canonicalPayload := relationshipCanonicalPayload{
		Type:      relType,
		SourceURL: sourceURL,
		TargetURL: targetURL,
		Action:    action,
		Timestamp: timestamp,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	// Sign the canonical payload
	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	req := RelationshipUpdateRequest{
		Type:      relType,
		SourceURL: sourceURL,
		TargetURL: targetURL,
		Action:    action,
		Signature: signature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("relationship update failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// QueryRelationships queries relationships by type and optional filters.
func (c *Client) QueryRelationships(relType string, filters map[string]string) (*RelationshipQueryResponse, error) {
	params := url.Values{}
	params.Set("type", relType)
	for k, v := range filters {
		params.Set(k, v)
	}
	endpoint := c.BaseURL + "/ds-relationship-query?" + params.Encode()

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	if err := c.addAuthHeaders(httpReq); err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("relationship query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result RelationshipQueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ============================================================================
// Canonical Payload Types (for signing)
// ============================================================================

// contentCanonicalPayload is the unified canonical payload for content registration signing.
// CRITICAL: Field order determines signature output.
type contentCanonicalPayload struct {
	Type     string                 `json:"type"`
	URL      string                 `json:"url"`
	Version  string                 `json:"version"`
	Author   string                 `json:"author"`
	Metadata map[string]interface{} `json:"metadata"`
}

// relationshipCanonicalPayload is the canonical payload for relationship update signing.
type relationshipCanonicalPayload struct {
	Type      string `json:"type"`
	SourceURL string `json:"source_url"`
	TargetURL string `json:"target_url"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

// MakeContentCanonicalJSON creates canonical JSON for content registration signing.
func MakeContentCanonicalJSON(contentType, contentURL, version, author string, metadata map[string]interface{}) ([]byte, error) {
	return json.Marshal(contentCanonicalPayload{
		Type:     contentType,
		URL:      contentURL,
		Version:  version,
		Author:   author,
		Metadata: metadata,
	})
}

// MakeRelationshipCanonicalJSON creates canonical JSON for relationship update signing.
func MakeRelationshipCanonicalJSON(relType, sourceURL, targetURL, action, timestamp string) ([]byte, error) {
	return json.Marshal(relationshipCanonicalPayload{
		Type:      relType,
		SourceURL: sourceURL,
		TargetURL: targetURL,
		Action:    action,
		Timestamp: timestamp,
	})
}

// ============================================================================
// Site Registration (ds-sites-* endpoints, table: ds_registered_sites)
// ============================================================================

// SiteCheckResponse is the response from the sites-check endpoint.
type SiteCheckResponse struct {
	IsRegistered        bool   `json:"is_registered"`
	Domain              string `json:"domain,omitempty"`
	RegisteredAt        string `json:"registered_at,omitempty"`
	RegistryURL         string `json:"registry_url,omitempty"`
	RegistrationVersion int    `json:"registration_version,omitempty"`
	ServiceAttestation  string `json:"service_attestation,omitempty"`
}

// SiteRegisterResponse is the response from the sites-register endpoint.
type SiteRegisterResponse struct {
	Success            bool   `json:"success"`
	Domain             string `json:"domain,omitempty"`
	RegistryURL        string `json:"registry_url,omitempty"`
	RegisteredAt       string `json:"registered_at,omitempty"`
	ServiceAttestation string `json:"service_attestation,omitempty"`
	Error              string `json:"error,omitempty"`
	Code               string `json:"code,omitempty"`
}

// SiteUnregisterResponse is the response from the sites-unregister endpoint.
type SiteUnregisterResponse struct {
	Success bool   `json:"success"`
	Domain  string `json:"domain,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
}

// siteRegistrationPayload is the canonical payload structure for site registration.
type siteRegistrationPayload struct {
	Version int    `json:"version"`
	Action  string `json:"action"`
	Domain  string `json:"domain"`
}

// siteRegisterRequest is the full request payload for the sites-register endpoint.
type siteRegisterRequest struct {
	Version    int    `json:"version"`
	Action     string `json:"action"`
	Domain     string `json:"domain"`
	Signature  string `json:"signature"`
	Email      string `json:"email,omitempty"`
	AuthorName string `json:"author_name,omitempty"`
}

// siteUnregisterRequest is the full request payload for the sites-unregister endpoint.
type siteUnregisterRequest struct {
	Version   int    `json:"version"`
	Action    string `json:"action"`
	Domain    string `json:"domain"`
	Signature string `json:"signature"`
}

// CheckSiteRegistration checks if a domain is registered with the discovery service.
func (c *Client) CheckSiteRegistration(domain string) (*SiteCheckResponse, error) {
	endpoint := fmt.Sprintf("%s/ds-sites-check?domain=%s", c.BaseURL, domain)

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result SiteCheckResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RegisterSite registers a domain with the discovery service.
func (c *Client) RegisterSite(domain string, privateKey []byte, email, authorName string) (*SiteRegisterResponse, error) {
	endpoint := c.BaseURL + "/ds-sites-register"

	canonicalPayload := siteRegistrationPayload{
		Version: 1,
		Action:  "register",
		Domain:  domain,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payload: %w", err)
	}

	req := siteRegisterRequest{
		Version:    1,
		Action:     "register",
		Domain:     domain,
		Signature:  signature,
		Email:      email,
		AuthorName: authorName,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result SiteRegisterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Error != "" {
			return &result, fmt.Errorf("registration failed: %s", result.Error)
		}
		return &result, fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	result.Success = true
	return &result, nil
}

// UnregisterSite unregisters a domain from the discovery service.
func (c *Client) UnregisterSite(domain string, privateKey []byte) (*SiteUnregisterResponse, error) {
	endpoint := c.BaseURL + "/ds-sites-unregister"

	canonicalPayload := siteRegistrationPayload{
		Version: 1,
		Action:  "unregister",
		Domain:  domain,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payload: %w", err)
	}

	req := siteUnregisterRequest{
		Version:   1,
		Action:    "unregister",
		Domain:    domain,
		Signature: signature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result SiteUnregisterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Error != "" {
			return &result, fmt.Errorf("unregistration failed: %s", result.Error)
		}
		return &result, fmt.Errorf("unregistration failed with status %d", resp.StatusCode)
	}

	result.Success = true
	return &result, nil
}

// MakeSiteRegistrationCanonicalJSON creates canonical JSON for site registration signing.
func MakeSiteRegistrationCanonicalJSON(action, domain string) ([]byte, error) {
	return json.Marshal(siteRegistrationPayload{
		Version: 1,
		Action:  action,
		Domain:  domain,
	})
}

// ============================================================================
// Domain Migration (ds-migrations endpoint, table: ds_domain_migrations)
// ============================================================================

// MigrationRecord represents a domain migration record from the discovery service.
type MigrationRecord struct {
	OldDomain  string `json:"old_domain"`
	NewDomain  string `json:"new_domain"`
	MigratedAt string `json:"migrated_at"`
	PublicKey  string `json:"public_key"`
}

// MigrationResponse is the response from the migrations query endpoint.
type MigrationResponse struct {
	Count      int               `json:"count"`
	Migrations []MigrationRecord `json:"migrations"`
}

// QueryMigrations queries the discovery service for domain migrations.
func (c *Client) QueryMigrations(domains []string) (*MigrationResponse, error) {
	endpoint := fmt.Sprintf("%s/ds-migrations?domains=%s", c.BaseURL, JoinDomains(domains))

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result MigrationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// migrationPayload is the canonical payload for domain migration signing.
type migrationPayload struct {
	Version   int    `json:"version"`
	Action    string `json:"action"`
	OldDomain string `json:"old_domain"`
	NewDomain string `json:"new_domain"`
}

// MigrationRegisterRequest is the request to register a domain migration.
type MigrationRegisterRequest struct {
	Version   int    `json:"version"`
	Action    string `json:"action"`
	OldDomain string `json:"old_domain"`
	NewDomain string `json:"new_domain"`
	Signature string `json:"signature"`
}

// RegisterMigration registers a domain migration with the discovery service.
func (c *Client) RegisterMigration(oldDomain, newDomain string, privateKey []byte) error {
	endpoint := c.BaseURL + "/ds-migrations-register"

	canonicalPayload := migrationPayload{
		Version:   1,
		Action:    "migrate",
		OldDomain: oldDomain,
		NewDomain: newDomain,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	req := MigrationRegisterRequest{
		Version:   1,
		Action:    "migrate",
		OldDomain: oldDomain,
		NewDomain: newDomain,
		Signature: signature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("migration registration failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ============================================================================
// Stream (ds-stream-* endpoints, table: ds_events)
// ============================================================================

// StreamEvent represents a single event in the discovery stream.
type StreamEvent struct {
	ID        int64                  `json:"id"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Actor     string                 `json:"actor"`
	Signature string                 `json:"signature"`
	Payload   map[string]interface{} `json:"payload"`
}

// StreamQueryResponse is the response from GET /ds-stream.
type StreamQueryResponse struct {
	Events  []StreamEvent `json:"events"`
	Cursor  string        `json:"cursor"`
	HasMore bool          `json:"has_more"`
}

// StreamHealthResponse is the response from GET /ds-stream-health.
type StreamHealthResponse struct {
	Status       string `json:"status"`
	LatestCursor string `json:"latest_cursor"`
	OldestCursor string `json:"oldest_cursor"`
	EventCount   int    `json:"event_count"`
}

// StreamQuery queries the discovery stream for events.
func (c *Client) StreamQuery(since string, limit int, typeFilter string, actorFilter string, targetFilter string, sourceFilter ...string) (*StreamQueryResponse, error) {
	params := url.Values{}
	if since != "" {
		params.Set("since", since)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if typeFilter != "" {
		params.Set("type", typeFilter)
	}
	if actorFilter != "" {
		params.Set("actor", actorFilter)
	}
	if targetFilter != "" {
		params.Set("target", targetFilter)
	}
	if len(sourceFilter) > 0 && sourceFilter[0] != "" {
		params.Set("source", sourceFilter[0])
	}

	endpoint := c.BaseURL + "/ds-stream"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	if err := c.addAuthHeaders(httpReq); err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stream query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result StreamQueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// StreamPublish publishes an event to the discovery stream.
func (c *Client) StreamPublish(eventType, actor string, payload map[string]interface{}, signature string) error {
	endpoint := c.BaseURL + "/ds-stream-publish"

	reqBody := struct {
		Type      string                 `json:"type"`
		Actor     string                 `json:"actor"`
		Payload   map[string]interface{} `json:"payload"`
		Signature string                 `json:"signature"`
	}{
		Type:      eventType,
		Actor:     actor,
		Payload:   payload,
		Signature: signature,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stream publish failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// StreamHealth returns the health status of the discovery stream.
func (c *Client) StreamHealth() (*StreamHealthResponse, error) {
	endpoint := c.BaseURL + "/ds-stream-health"

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stream health failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result StreamHealthResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// MakeStreamCanonicalJSON creates the canonical JSON bytes for stream event signing.
func MakeStreamCanonicalJSON(eventType string, payload map[string]interface{}) ([]byte, error) {
	canonical := struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}{
		Type:    eventType,
		Payload: payload,
	}
	return json.Marshal(canonical)
}

// ============================================================================
// Utility Functions
// ============================================================================

// ExtractDomainFromURL extracts the hostname from a URL string.
func ExtractDomainFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// JoinDomains joins domain strings with commas for the stream actor filter.
func JoinDomains(domains []string) string {
	return strings.Join(domains, ",")
}
