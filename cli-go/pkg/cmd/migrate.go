package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

func handleMigrate(args []string) {
	if len(args) < 1 {
		exitError("Usage: polis migrate <new-domain>")
	}

	newDomain := args[0]
	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Get current domain from POLIS_BASE_URL
	baseURL := os.Getenv("POLIS_BASE_URL")
	if baseURL == "" {
		exitError("POLIS_BASE_URL not set")
	}
	oldDomain := extractDomain(baseURL)

	// Load private key
	privKey, err := loadPrivateKey(dir)
	if err != nil {
		exitError("Failed to load private key: %v", err)
	}

	// Load discovery client
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}
	apiKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	client := discovery.NewClient(discoveryURL, apiKey)

	if !jsonOutput {
		fmt.Printf("[i] Migrating domain: %s -> %s\n", oldDomain, newDomain)
	}

	// Register the migration with discovery service
	if err := client.RegisterMigration(oldDomain, newDomain, privKey); err != nil {
		exitError("Failed to register migration: %v", err)
	}

	// Update .well-known/polis
	if !jsonOutput {
		fmt.Println("[i] Updating local files...")
	}

	// Re-sign all posts and comments would go here
	// For now, just update the .well-known/polis version
	wk, err := site.LoadWellKnown(dir)
	if err != nil {
		exitError("Failed to load .well-known/polis: %v", err)
	}
	wk.Version = Version

	// Note: In a full implementation, we would:
	// 1. Re-sign all posts and comments with the new domain in URLs
	// 2. Update all internal references
	// 3. Update the public index

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "migrate",
			"data": map[string]interface{}{
				"old_domain":           oldDomain,
				"new_domain":           newDomain,
				"migration_registered": true,
			},
		})
	} else {
		fmt.Println()
		fmt.Printf("[✓] Migration registered: %s -> %s\n", oldDomain, newDomain)
		fmt.Println()
		fmt.Println("[i] Next steps:")
		fmt.Println("  1. Update POLIS_BASE_URL to https://" + newDomain)
		fmt.Println("  2. Re-run 'polis republish' on all posts")
		fmt.Println("  3. Deploy to the new domain")
	}
}

func handleMigrationsApply(args []string) {
	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Load discovery client
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}
	apiKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	client := discovery.NewClient(discoveryURL, apiKey)

	// Collect relevant domains from local files
	domains, err := collectRelevantDomains(dir)
	if err != nil {
		exitError("Failed to collect domains: %v", err)
	}

	if len(domains) == 0 {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "migrations-apply",
				"data": map[string]interface{}{
					"migrations_applied": 0,
					"files_updated":      []string{},
					"message":            "No relevant domains found in local files.",
				},
			})
		} else {
			fmt.Println("[i] No relevant domains found in local files.")
		}
		return
	}

	// Query for migrations
	migrationsResp, err := client.QueryMigrations(domains)
	if err != nil {
		exitError("Failed to query migrations: %v", err)
	}

	if migrationsResp.Count == 0 {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "migrations-apply",
				"data": map[string]interface{}{
					"migrations_applied": 0,
					"files_updated":      []string{},
					"message":            "No pending migrations found.",
				},
			})
		} else {
			fmt.Println("[i] No pending migrations found.")
		}
		return
	}

	if !jsonOutput {
		fmt.Printf("[i] Found %d migration(s) to apply\n", migrationsResp.Count)
	}

	// TODO: Apply migrations to local files
	// For now, just report what was found

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "migrations-apply",
			"data": map[string]interface{}{
				"migrations_found": migrationsResp.Count,
				"migrations":       migrationsResp.Migrations,
			},
		})
	} else {
		for _, m := range migrationsResp.Migrations {
			fmt.Printf("  %s -> %s (%s)\n", m.OldDomain, m.NewDomain, m.MigratedAt[:10])
		}
		fmt.Println()
		fmt.Println("[i] Run with --apply to update local files")
	}
}

func collectRelevantDomains(dir string) ([]string, error) {
	// Use the migrate package
	domainSet := make(map[string]bool)

	// Simple domain collection from following.json
	followingPath := dir + "/metadata/following.json"
	if data, err := os.ReadFile(followingPath); err == nil {
		content := string(data)
		// Extract domains from https:// URLs
		for {
			idx := strings.Index(content, "https://")
			if idx < 0 {
				break
			}
			content = content[idx+8:]
			end := strings.IndexAny(content, "\"/,}")
			if end > 0 {
				domain := content[:end]
				if slashIdx := strings.Index(domain, "/"); slashIdx > 0 {
					domain = domain[:slashIdx]
				}
				domainSet[domain] = true
			}
		}
	}

	var domains []string
	for d := range domainSet {
		domains = append(domains, d)
	}
	return domains, nil
}
