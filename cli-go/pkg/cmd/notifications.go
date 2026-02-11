package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	"github.com/vdibart/polis-cli/cli-go/pkg/notification"
)

func handleNotifications(args []string) {
	if len(args) < 1 {
		handleNotificationsList([]string{})
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		handleNotificationsList(subArgs)
	default:
		// Treat as list with options
		handleNotificationsList(args)
	}
}

func handleNotificationsList(args []string) {
	fs := flag.NewFlagSet("notifications list", flag.ExitOnError)
	showAll := fs.Bool("all", false, "Show all notifications")
	fs.Bool("a", false, "Show all notifications (alias)")
	fs.Parse(args)

	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Determine discovery domain
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}
	discoveryDomain := extractDomain(discoveryURL)
	if discoveryDomain == "" {
		discoveryDomain = "default"
	}

	// Read notifications from new state file
	mgr := notification.NewManager(dir, discoveryDomain)

	var entries []notification.StateEntry
	var err error

	if *showAll {
		entries, _, err = mgr.ListPaginated(0, 0, true)
	} else {
		entries, _, err = mgr.ListPaginated(0, 0, false)
	}
	if err != nil {
		exitError("Failed to list notifications: %v", err)
	}

	// Also fetch pending blessings from discovery service
	var pendingBlessings []map[string]interface{}
	var migrations []map[string]interface{}

	apiKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	baseURL := os.Getenv("POLIS_BASE_URL")

	if apiKey != "" && baseURL != "" {
		client := discovery.NewClient(discoveryURL, apiKey)
		domain := extractDomain(baseURL)

		// Get pending blessings via relationship-query
		resp, err := client.QueryRelationships("polis.blessing", map[string]string{
			"actor":  domain,
			"status": "pending",
		})
		if err == nil {
			for _, r := range resp.Records {
				author, _ := r.Metadata["author"].(string)
				pendingBlessings = append(pendingBlessings, map[string]interface{}{
					"author":      author,
					"in_reply_to": r.TargetURL,
					"comment_url": r.SourceURL,
				})
			}
		}

		// Get pending migrations
		domains, _ := collectRelevantDomains(dir)
		if len(domains) > 0 {
			migrationsResp, err := client.QueryMigrations(domains)
			if err == nil {
				for _, m := range migrationsResp.Migrations {
					migrations = append(migrations, map[string]interface{}{
						"old_domain":  m.OldDomain,
						"new_domain":  m.NewDomain,
						"migrated_at": m.MigratedAt,
					})
				}
			}
		}
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "notifications",
			"data": map[string]interface{}{
				"notifications":     entries,
				"pending_blessings": pendingBlessings,
				"domain_migrations": migrations,
			},
		})
	} else {
		totalCount := len(entries) + len(pendingBlessings) + len(migrations)

		if totalCount == 0 {
			fmt.Println("[i] No notifications")
			return
		}

		fmt.Println()

		if len(entries) > 0 {
			fmt.Printf("[i] === Notifications (%d) ===\n", len(entries))
			for _, e := range entries {
				readMark := " "
				if e.ReadAt != "" {
					readMark = "."
				}
				fmt.Printf("  %s %s %s (%s)\n", readMark, e.Icon, e.Message, e.CreatedAt[:10])
			}
			fmt.Println()
		}

		if len(pendingBlessings) > 0 {
			fmt.Printf("[i] === Pending Blessing Requests (%d) ===\n", len(pendingBlessings))
			for _, b := range pendingBlessings {
				inReplyTo := b["in_reply_to"].(string)
				parts := strings.Split(inReplyTo, "/")
				shortPath := parts[len(parts)-1]
				fmt.Printf("  #%v: %s on %s\n", b["id"], b["author"], shortPath)
			}
			fmt.Println()
			fmt.Println("[i] Run 'polis blessing requests' for details, or 'polis blessing grant <id>' to approve.")
			fmt.Println()
		}

		if len(migrations) > 0 {
			fmt.Printf("[i] === Domain Migrations (%d) ===\n", len(migrations))
			for _, m := range migrations {
				migratedAt := m["migrated_at"].(string)
				if len(migratedAt) > 10 {
					migratedAt = migratedAt[:10]
				}
				fmt.Printf("  %s -> %s (%s)\n", m["old_domain"], m["new_domain"], migratedAt)
			}
			fmt.Println()
			fmt.Println("[i] Run 'polis migrations apply' to update local references.")
		}
	}
}
