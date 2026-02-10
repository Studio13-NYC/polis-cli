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
	typeFilter := fs.String("type", "", "Filter by type (comma-separated)")
	fs.String("t", "", "Filter by type (alias)")
	fs.Parse(args)

	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	mgr := notification.NewManager(dir)
	mgr.InitManifest()

	// Get local notifications
	var notifications []notification.Notification
	var err error

	if *typeFilter != "" {
		types := strings.Split(*typeFilter, ",")
		notifications, err = mgr.ListByType(types)
	} else {
		notifications, err = mgr.List()
	}

	if err != nil {
		exitError("Failed to list notifications: %v", err)
	}

	// Also fetch pending blessings from discovery service
	var pendingBlessings []map[string]interface{}
	var migrations []map[string]interface{}

	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}
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
				"notifications":     notifications,
				"pending_blessings": pendingBlessings,
				"domain_migrations": migrations,
			},
		})
	} else {
		totalCount := len(notifications) + len(pendingBlessings) + len(migrations)

		if totalCount == 0 {
			fmt.Println("[i] No notifications")
			return
		}

		fmt.Println()

		if len(notifications) > 0 {
			fmt.Printf("[i] === Notifications (%d) ===\n", len(notifications))
			for _, n := range notifications {
				payloadStr := string(n.Payload)
				if len(payloadStr) > 60 {
					payloadStr = payloadStr[:60] + "..."
				}
				fmt.Printf("  [%s] %s (%s)\n", n.Type, payloadStr, n.CreatedAt[:10])
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

	// Suppress unused variable warning
	_ = showAll
}

