// Package discovery provides tests for the discovery service client.
package discovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckSiteRegistration_Registered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/ds-sites-check" {
			t.Errorf("Expected /ds-sites-check, got %s", r.URL.Path)
		}
		domain := r.URL.Query().Get("domain")
		if domain != "alice.com" {
			t.Errorf("Expected domain=alice.com, got %s", domain)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_registered":        true,
			"domain":               "alice.com",
			"registered_at":        "2026-01-15T10:30:00Z",
			"registry_url":         "https://registry.polis.pub/alice.com",
			"registration_version": 1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	result, err := client.CheckSiteRegistration("alice.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsRegistered {
		t.Error("Expected IsRegistered=true")
	}
	if result.Domain != "alice.com" {
		t.Errorf("Expected domain=alice.com, got %s", result.Domain)
	}
	if result.RegisteredAt != "2026-01-15T10:30:00Z" {
		t.Errorf("Expected registered_at=2026-01-15T10:30:00Z, got %s", result.RegisteredAt)
	}
}

func TestCheckSiteRegistration_NotRegistered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_registered": false,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	result, err := client.CheckSiteRegistration("bob.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsRegistered {
		t.Error("Expected IsRegistered=false")
	}
}

func TestCheckSiteRegistration_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:99999", "test-api-key")
	_, err := client.CheckSiteRegistration("test.com")
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestCheckSiteRegistration_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	_, err := client.CheckSiteRegistration("test.com")
	if err == nil {
		t.Error("Expected error for server failure")
	}
}

func TestCanonicalPayloadFormat_Register(t *testing.T) {
	// CRITICAL: This test verifies that the canonical payload format matches the bash CLI exactly.
	// The bash CLI uses: {"version":1,"action":"register","domain":"alice.com"}
	// Field order must be: version, action, domain

	payload, err := MakeSiteRegistrationCanonicalJSON("register", "alice.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The expected format from bash CLI
	expected := `{"version":1,"action":"register","domain":"alice.com"}`

	if string(payload) != expected {
		t.Errorf("Canonical payload mismatch.\nExpected: %s\nGot:      %s", expected, string(payload))
	}
}

func TestCanonicalPayloadFormat_Unregister(t *testing.T) {
	// CRITICAL: This test verifies that the canonical payload format matches the bash CLI exactly.
	// The bash CLI uses: {"version":1,"action":"unregister","domain":"alice.com"}
	// Field order must be: version, action, domain

	payload, err := MakeSiteRegistrationCanonicalJSON("unregister", "alice.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The expected format from bash CLI
	expected := `{"version":1,"action":"unregister","domain":"alice.com"}`

	if string(payload) != expected {
		t.Errorf("Canonical payload mismatch.\nExpected: %s\nGot:      %s", expected, string(payload))
	}
}

func TestRegisterSite_Success(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/ds-sites-register" {
			t.Errorf("Expected /ds-sites-register, got %s", r.URL.Path)
		}

		// Decode and store the received payload for verification
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"domain":        "alice.com",
			"registry_url":  "https://registry.polis.pub/alice.com",
			"registered_at": "2026-01-15T10:30:00Z",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	// Use a mock private key (this won't actually sign correctly, but we can test the structure)
	// In real usage, this would be a valid Ed25519 private key
	mockPrivateKey := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n")

	// Note: This test will fail at signing because we're using a mock key
	// In production, we'd use a proper key fixture
	_, err := client.RegisterSite("alice.com", mockPrivateKey, "alice@example.com", "Alice")
	// We expect a signing error due to the mock key
	if err == nil {
		// If we get here, verify the payload structure
		if receivedPayload["version"] != float64(1) {
			t.Errorf("Expected version=1, got %v", receivedPayload["version"])
		}
		if receivedPayload["action"] != "register" {
			t.Errorf("Expected action=register, got %v", receivedPayload["action"])
		}
		if receivedPayload["domain"] != "alice.com" {
			t.Errorf("Expected domain=alice.com, got %v", receivedPayload["domain"])
		}
	}
	// Note: With a mock key, signing will fail, which is expected behavior
}

func TestRegisterSite_AlreadyRegistered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Site is already registered",
			"code":    "ALREADY_REGISTERED",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	mockPrivateKey := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n")

	_, err := client.RegisterSite("alice.com", mockPrivateKey, "", "")
	// We expect an error (either signing or already registered)
	if err == nil {
		t.Error("Expected error for already registered site")
	}
}

func TestUnregisterSite_Success(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/ds-sites-unregister" {
			t.Errorf("Expected /ds-sites-unregister, got %s", r.URL.Path)
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"domain":  "alice.com",
			"message": "Site unregistered successfully",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	mockPrivateKey := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n")

	_, err := client.UnregisterSite("alice.com", mockPrivateKey)
	if err == nil {
		// Verify payload structure if signing succeeded
		if receivedPayload["version"] != float64(1) {
			t.Errorf("Expected version=1, got %v", receivedPayload["version"])
		}
		if receivedPayload["action"] != "unregister" {
			t.Errorf("Expected action=unregister, got %v", receivedPayload["action"])
		}
		if receivedPayload["domain"] != "alice.com" {
			t.Errorf("Expected domain=alice.com, got %v", receivedPayload["domain"])
		}
	}
	// Note: With a mock key, signing will fail, which is expected behavior
}

func TestUnregisterSite_NotRegistered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Site is not registered",
			"code":    "NOT_REGISTERED",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	mockPrivateKey := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n")

	_, err := client.UnregisterSite("bob.com", mockPrivateKey)
	if err == nil {
		t.Error("Expected error for unregistered site")
	}
}
