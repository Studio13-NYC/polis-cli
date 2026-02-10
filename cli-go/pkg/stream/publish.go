package stream

import (
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

// Discovery service configuration for stream event publishing.
// Set by the calling application (CLI or webapp) during initialization.
var (
	DiscoveryURL string
	DiscoveryKey string
	BaseURL      string // POLIS_BASE_URL — actor domain is extracted from this
)

// PublishEvent publishes an event to the discovery stream.
// Silently returns nil if discovery is not configured.
func PublishEvent(eventType string, payload map[string]interface{}, privateKey []byte) error {
	if DiscoveryURL == "" || DiscoveryKey == "" || BaseURL == "" {
		return nil
	}
	if privateKey == nil {
		return nil
	}

	actor := discovery.ExtractDomainFromURL(BaseURL)
	if actor == "" {
		return fmt.Errorf("could not extract domain from base URL")
	}

	canonical, err := discovery.MakeStreamCanonicalJSON(eventType, payload)
	if err != nil {
		return fmt.Errorf("canonical JSON: %w", err)
	}

	sig, err := signing.SignContent(canonical, privateKey)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	client := discovery.NewClient(DiscoveryURL, DiscoveryKey)
	if err := client.StreamPublish(eventType, actor, payload, sig); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}
