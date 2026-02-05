// Package discovery provides a client for the polis discovery service.
package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// Client is an HTTP client for the discovery service.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new discovery service client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BeseechRequest is the request body for the beseech blessing endpoint.
type BeseechRequest struct {
	CommentURL       string `json:"comment_url"`
	CommentVersion   string `json:"comment_version"`
	InReplyTo        string `json:"in_reply_to"`
	InReplyToVersion string `json:"in_reply_to_version,omitempty"`
	RootPost         string `json:"root_post"`
	Author           string `json:"author"`
	Timestamp        string `json:"timestamp"`
	Signature        string `json:"signature"`
}

// BeseechResponse is the response from the beseech blessing endpoint.
type BeseechResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status"` // "pending" or "blessed" (auto-blessed)
	Error   string `json:"error,omitempty"`
}

// BlessingRequest represents a pending blessing request.
type BlessingRequest struct {
	ID               string `json:"id"`
	CommentURL       string `json:"comment_url"`
	CommentVersion   string `json:"comment_version"`
	InReplyTo        string `json:"in_reply_to"`
	InReplyToVersion string `json:"in_reply_to_version,omitempty"`
	RootPost         string `json:"root_post"`
	Author           string `json:"author"`
	Timestamp        string `json:"timestamp"`
	BlessingStatus   string `json:"blessing_status"` // "pending", "blessed", "denied"
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// BlessingStatusResponse is the response when checking blessing status.
type BlessingStatusResponse struct {
	Success        bool   `json:"success"`
	BlessingStatus string `json:"blessing_status"` // "pending", "blessed", "denied"
	Error          string `json:"error,omitempty"`
}

// BeseechBlessing sends a blessing request to the discovery service.
func (c *Client) BeseechBlessing(req *BeseechRequest) (*BeseechResponse, error) {
	endpoint := c.BaseURL + "/comments-blessing-beseech"

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

	var result BeseechResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Error != "" {
			return &result, fmt.Errorf("request failed: %s", result.Error)
		}
		return &result, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return &result, nil
}

// CheckBlessingStatus checks the status of a blessing request.
func (c *Client) CheckBlessingStatus(commentVersion string) (*BlessingStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/comments-blessing-requests?version=%s", c.BaseURL, commentVersion)

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

	var result BlessingStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Error != "" {
			return &result, fmt.Errorf("request failed: %s", result.Error)
		}
		return &result, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return &result, nil
}

// GetPendingRequests returns all pending blessing requests for a domain.
func (c *Client) GetPendingRequests(domain string) ([]BlessingRequest, error) {
	endpoint := fmt.Sprintf("%s/comments-blessing-requests?domain=%s&status=pending", c.BaseURL, domain)

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

	var result struct {
		Success  bool              `json:"success"`
		Requests []BlessingRequest `json:"requests"`
		Error    string            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("request failed: %s", result.Error)
	}

	return result.Requests, nil
}

// blessingPayload is the canonical payload structure for grant/deny requests.
// Field order matches bash CLI: {action, comment_version, timestamp}
// Using a struct guarantees consistent JSON field ordering for signature verification.
type blessingPayload struct {
	Action         string `json:"action"`
	CommentVersion string `json:"comment_version"`
	Timestamp      string `json:"timestamp"`
}

// grantDenyRequest is the full request payload for grant/deny endpoints.
// CRITICAL: Field order must match bash CLI for consistency.
// Bash CLI order: comment_version, action, timestamp, signature
type grantDenyRequest struct {
	CommentVersion string `json:"comment_version"`
	Action         string `json:"action"`
	Timestamp      string `json:"timestamp"`
	Signature      string `json:"signature"`
}

// GrantBlessing grants a blessing for a comment.
// The privateKey is used to sign the request payload for authentication.
// This matches the bash CLI's behavior which signs the grant request.
func (c *Client) GrantBlessing(commentVersion string, privateKey []byte) error {
	endpoint := c.BaseURL + "/comments-blessing-grant"

	// Create timestamp for the request
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Create canonical payload for signing (matches bash CLI format)
	// The bash CLI creates: {action: "grant", comment_version: ..., timestamp: ...}
	// Using struct to guarantee field order for signature verification
	canonicalPayload := blessingPayload{
		Action:         "grant",
		CommentVersion: commentVersion,
		Timestamp:      timestamp,
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

	// Create the full request payload with signature
	// Using struct to guarantee field order for consistency with bash CLI
	body, err := json.Marshal(grantDenyRequest{
		CommentVersion: commentVersion,
		Action:         "grant",
		Timestamp:      timestamp,
		Signature:      signature,
	})
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
		return fmt.Errorf("grant blessing failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DenyBlessing denies a blessing for a comment.
// The privateKey is used to sign the request payload for authentication.
// This matches the bash CLI's behavior which signs the deny request.
func (c *Client) DenyBlessing(commentVersion string, privateKey []byte) error {
	endpoint := c.BaseURL + "/comments-blessing-deny"

	// Create timestamp for the request
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Create canonical payload for signing (matches bash CLI format)
	// The bash CLI creates: {action: "deny", comment_version: ..., timestamp: ...}
	// Using struct to guarantee field order for signature verification
	canonicalPayload := blessingPayload{
		Action:         "deny",
		CommentVersion: commentVersion,
		Timestamp:      timestamp,
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

	// Create the full request payload with signature
	// Using struct to guarantee field order for consistency with bash CLI
	body, err := json.Marshal(grantDenyRequest{
		CommentVersion: commentVersion,
		Action:         "deny",
		Timestamp:      timestamp,
		Signature:      signature,
	})
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
		return fmt.Errorf("deny blessing failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// beseechCanonicalPayload is the canonical payload for beseech signing.
// CRITICAL: Field order must match bash CLI exactly.
// Bash CLI order: comment_url, comment_version, in_reply_to, root_post, author, timestamp
type beseechCanonicalPayload struct {
	CommentURL     string `json:"comment_url"`
	CommentVersion string `json:"comment_version"`
	InReplyTo      string `json:"in_reply_to"`
	RootPost       string `json:"root_post"`
	Author         string `json:"author"`
	Timestamp      string `json:"timestamp"`
}

// beseechCanonicalPayloadWithVersion includes optional in_reply_to_version.
// Bash CLI order: comment_url, comment_version, in_reply_to, in_reply_to_version, root_post, author, timestamp
type beseechCanonicalPayloadWithVersion struct {
	CommentURL       string `json:"comment_url"`
	CommentVersion   string `json:"comment_version"`
	InReplyTo        string `json:"in_reply_to"`
	InReplyToVersion string `json:"in_reply_to_version"`
	RootPost         string `json:"root_post"`
	Author           string `json:"author"`
	Timestamp        string `json:"timestamp"`
}

// MakeBeseechCanonicalJSON creates canonical JSON for beseech signing.
// Returns JSON bytes that should be signed (no trailing newline).
func MakeBeseechCanonicalJSON(commentURL, commentVersion, inReplyTo, inReplyToVersion, rootPost, author, timestamp string) ([]byte, error) {
	if inReplyToVersion != "" {
		return json.Marshal(beseechCanonicalPayloadWithVersion{
			CommentURL:       commentURL,
			CommentVersion:   commentVersion,
			InReplyTo:        inReplyTo,
			InReplyToVersion: inReplyToVersion,
			RootPost:         rootPost,
			Author:           author,
			Timestamp:        timestamp,
		})
	}
	return json.Marshal(beseechCanonicalPayload{
		CommentURL:     commentURL,
		CommentVersion: commentVersion,
		InReplyTo:      inReplyTo,
		RootPost:       rootPost,
		Author:         author,
		Timestamp:      timestamp,
	})
}

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
// CRITICAL: Field order must match bash CLI exactly for signature verification.
// Bash CLI order: version, action, domain
type siteRegistrationPayload struct {
	Version int    `json:"version"`
	Action  string `json:"action"`
	Domain  string `json:"domain"`
}

// siteRegisterRequest is the full request payload for the sites-register endpoint.
// CRITICAL: Field order must match bash CLI for consistency.
type siteRegisterRequest struct {
	Version        int    `json:"version"`
	Action         string `json:"action"`
	Domain         string `json:"domain"`
	OwnerSignature string `json:"owner_signature"`
	Email          string `json:"email,omitempty"`
	AuthorName     string `json:"author_name,omitempty"`
}

// siteUnregisterRequest is the full request payload for the sites-unregister endpoint.
// CRITICAL: Field order must match bash CLI for consistency.
type siteUnregisterRequest struct {
	Version        int    `json:"version"`
	Action         string `json:"action"`
	Domain         string `json:"domain"`
	OwnerSignature string `json:"owner_signature"`
}

// CheckSiteRegistration checks if a domain is registered with the discovery service.
func (c *Client) CheckSiteRegistration(domain string) (*SiteCheckResponse, error) {
	endpoint := fmt.Sprintf("%s/sites-check?domain=%s", c.BaseURL, domain)

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
// The privateKey is used to sign the registration payload for authentication.
func (c *Client) RegisterSite(domain string, privateKey []byte, email, authorName string) (*SiteRegisterResponse, error) {
	endpoint := c.BaseURL + "/sites-register"

	// Create canonical payload for signing (matches bash CLI format exactly)
	// CRITICAL: Field order must be: version, action, domain
	canonicalPayload := siteRegistrationPayload{
		Version: 1,
		Action:  "register",
		Domain:  domain,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	// Sign the canonical payload
	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payload: %w", err)
	}

	// Create the full request payload with signature
	req := siteRegisterRequest{
		Version:        1,
		Action:         "register",
		Domain:         domain,
		OwnerSignature: signature,
		Email:          email,
		AuthorName:     authorName,
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
// The privateKey is used to sign the unregistration payload for authentication.
func (c *Client) UnregisterSite(domain string, privateKey []byte) (*SiteUnregisterResponse, error) {
	endpoint := c.BaseURL + "/sites-unregister"

	// Create canonical payload for signing (matches bash CLI format exactly)
	// CRITICAL: Field order must be: version, action, domain
	canonicalPayload := siteRegistrationPayload{
		Version: 1,
		Action:  "unregister",
		Domain:  domain,
	}
	canonicalJSON, err := json.Marshal(canonicalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical payload: %w", err)
	}

	// Sign the canonical payload
	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payload: %w", err)
	}

	// Create the full request payload with signature
	req := siteUnregisterRequest{
		Version:        1,
		Action:         "unregister",
		Domain:         domain,
		OwnerSignature: signature,
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
// Returns JSON bytes that should be signed (no trailing newline).
// CRITICAL: Field order must match bash CLI exactly: {"version":1,"action":"...","domain":"..."}
func MakeSiteRegistrationCanonicalJSON(action, domain string) ([]byte, error) {
	return json.Marshal(siteRegistrationPayload{
		Version: 1,
		Action:  action,
		Domain:  domain,
	})
}

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
	endpoint := fmt.Sprintf("%s/migrations?domains=%s", c.BaseURL, joinDomains(domains))

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

// joinDomains joins domain strings with commas for URL query.
func joinDomains(domains []string) string {
	result := ""
	for i, d := range domains {
		if i > 0 {
			result += ","
		}
		result += d
	}
	return result
}

// Comment represents a comment from the discovery service.
type Comment struct {
	ID             int    `json:"id"`
	CommentURL     string `json:"comment_url"`
	CommentVersion string `json:"comment_version"`
	InReplyTo      string `json:"in_reply_to"`
	RootPost       string `json:"root_post"`
	Author         string `json:"author"`
	Timestamp      string `json:"timestamp"`
	BlessingStatus string `json:"blessing_status"`
	BlessedAt      string `json:"blessed_at,omitempty"`
	BlessedBy      string `json:"blessed_by,omitempty"`
}

// GetCommentsByAuthor fetches comments by author email with optional status filter.
func (c *Client) GetCommentsByAuthor(authorEmail, status string) ([]Comment, error) {
	endpoint := fmt.Sprintf("%s/comments?author=%s", c.BaseURL, authorEmail)
	if status != "" {
		endpoint += "&status=" + status
	}

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

	var result struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Comments, nil
}

// GetCommentByURL fetches a comment by its URL.
func (c *Client) GetCommentByURL(commentURL string) (*Comment, error) {
	endpoint := fmt.Sprintf("%s/comments?url=%s", c.BaseURL, commentURL)

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

	var result struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Comments) == 0 {
		return nil, fmt.Errorf("comment not found")
	}

	return &result.Comments[0], nil
}

// GetBlessedComments fetches blessed comments for a domain.
func (c *Client) GetBlessedComments(domain string) ([]Comment, error) {
	endpoint := fmt.Sprintf("%s/comments?in_reply_to_domain=%s&status=blessed", c.BaseURL, domain)

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

	var result struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Comments, nil
}

// VersionCheckResponse is the response from the polis-version endpoint.
type VersionCheckResponse struct {
	Latest           string `json:"latest"`
	UpgradeAvailable bool   `json:"upgrade_available"`
	ReleasedAt       string `json:"released_at,omitempty"`
	DownloadURL      string `json:"download_url,omitempty"`
}

// CheckVersion checks for CLI updates.
func (c *Client) CheckVersion(currentVersion string) (*VersionCheckResponse, error) {
	endpoint := fmt.Sprintf("%s/polis-version?current=%s", c.BaseURL, currentVersion)

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
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
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var result VersionCheckResponse
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
	Version        int    `json:"version"`
	Action         string `json:"action"`
	OldDomain      string `json:"old_domain"`
	NewDomain      string `json:"new_domain"`
	OwnerSignature string `json:"owner_signature"`
}

// RegisterMigration registers a domain migration with the discovery service.
func (c *Client) RegisterMigration(oldDomain, newDomain string, privateKey []byte) error {
	endpoint := c.BaseURL + "/migrations-register"

	// Create canonical payload for signing
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

	// Sign the canonical payload
	signature, err := signing.SignContent(canonicalJSON, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	// Create the full request
	req := MigrationRegisterRequest{
		Version:        1,
		Action:         "migrate",
		OldDomain:      oldDomain,
		NewDomain:      newDomain,
		OwnerSignature: signature,
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
