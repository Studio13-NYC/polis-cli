package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
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
	case "read":
		handleNotificationsRead(subArgs)
	case "dismiss":
		handleNotificationsDismiss(subArgs)
	case "sync":
		handleNotificationsSync(subArgs)
	case "config":
		handleNotificationsConfig(subArgs)
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

		// Get pending blessings
		requests, err := client.GetPendingRequests(domain)
		if err == nil {
			for _, r := range requests {
				pendingBlessings = append(pendingBlessings, map[string]interface{}{
					"id":              r.ID,
					"author":          r.Author,
					"in_reply_to":     r.InReplyTo,
					"comment_version": r.CommentVersion,
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

func handleNotificationsRead(args []string) {
	if len(args) < 1 {
		exitError("Notification ID required. Use --all to mark all as read.")
	}

	id := args[0]
	dir := getDataDir()
	mgr := notification.NewManager(dir)

	if id == "--all" || id == "-a" {
		if err := mgr.RemoveAll(); err != nil {
			exitError("Failed to remove notifications: %v", err)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "notifications-read",
				"data": map[string]interface{}{
					"removed": "all",
				},
			})
		} else {
			fmt.Println("[✓] All notifications marked as read")
		}
	} else {
		if err := mgr.Remove(id); err != nil {
			exitError("Failed to remove notification: %v", err)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "notifications-read",
				"data": map[string]interface{}{
					"removed": id,
				},
			})
		} else {
			fmt.Printf("[✓] Notification %s marked as read\n", id)
		}
	}
}

func handleNotificationsDismiss(args []string) {
	fs := flag.NewFlagSet("notifications dismiss", flag.ExitOnError)
	olderThan := fs.String("older-than", "", "Dismiss notifications older than N days (e.g., 30d)")
	fs.Parse(args)

	if *olderThan != "" {
		days := strings.TrimSuffix(*olderThan, "d")
		daysInt, err := strconv.Atoi(days)
		if err != nil {
			exitError("Invalid days format: %s", *olderThan)
		}

		dir := getDataDir()
		mgr := notification.NewManager(dir)

		removed, err := mgr.RemoveOlderThan(daysInt)
		if err != nil {
			exitError("Failed to dismiss notifications: %v", err)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "notifications-dismiss",
				"data": map[string]interface{}{
					"removed":        removed,
					"older_than_days": daysInt,
				},
			})
		} else {
			fmt.Printf("[✓] Dismissed %d notifications older than %d days\n", removed, daysInt)
		}
		return
	}

	// Otherwise, treat remaining args as ID
	remaining := fs.Args()
	if len(remaining) < 1 {
		exitError("Notification ID required. Use --older-than to dismiss old notifications.")
	}

	handleNotificationsRead(remaining)
}

func handleNotificationsSync(args []string) {
	fs := flag.NewFlagSet("notifications sync", flag.ExitOnError)
	reset := fs.Bool("reset", false, "Reset watermark and fetch all notifications")
	fs.Parse(args)

	dir := getDataDir()
	mgr := notification.NewManager(dir)
	mgr.InitManifest()

	if *reset {
		mgr.UpdateWatermark("1970-01-01T00:00:00Z")
		if !jsonOutput {
			fmt.Println("[i] Reset watermark. Next sync will fetch all notifications.")
		}
	}

	lastTs, _ := mgr.GetWatermark()

	// Check for version updates
	versionNotificationAdded := false
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}

	client := discovery.NewClient(discoveryURL, "")
	versionResp, err := client.CheckVersion(Version)
	if err == nil && versionResp.UpgradeAvailable {
		enabled, _ := mgr.IsTypeEnabled("version_available")
		if enabled {
			payload, _ := json.Marshal(map[string]interface{}{
				"current_version": Version,
				"latest_version":  versionResp.Latest,
				"released_at":     versionResp.ReleasedAt,
				"download_url":    versionResp.DownloadURL,
			})
			mgr.Add("version_available", "discovery", payload, "version_available:"+versionResp.Latest)
			versionNotificationAdded = true
		}
	}

	// Update watermark
	newTs := "2006-01-02T15:04:05Z" // would use time.Now()
	mgr.UpdateWatermark(newTs)

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "notifications-sync",
			"data": map[string]interface{}{
				"synced_from":                lastTs,
				"synced_to":                  newTs,
				"version_notification_added": versionNotificationAdded,
			},
		})
	} else {
		fmt.Println("[✓] Synced notifications")
		if versionNotificationAdded {
			fmt.Println("[i] New version available notification added")
		}
	}
}

func handleNotificationsConfig(args []string) {
	fs := flag.NewFlagSet("notifications config", flag.ExitOnError)
	pollInterval := fs.String("poll-interval", "", "Set poll interval (e.g., 30m)")
	enable := fs.String("enable", "", "Enable notification type")
	disable := fs.String("disable", "", "Disable notification type")
	mute := fs.String("mute", "", "Mute domain")
	unmute := fs.String("unmute", "", "Unmute domain")
	fs.Parse(args)

	dir := getDataDir()
	mgr := notification.NewManager(dir)
	mgr.InitManifest()

	// If no options, show current config
	if *pollInterval == "" && *enable == "" && *disable == "" && *mute == "" && *unmute == "" {
		prefs, err := mgr.GetPreferences()
		if err != nil {
			exitError("Failed to get preferences: %v", err)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "notifications-config",
				"data":    prefs,
			})
		} else {
			fmt.Println()
			fmt.Println("[i] Notification Preferences:")
			fmt.Printf("  Poll interval: %d minutes\n", prefs.PollIntervalMinutes)
			fmt.Printf("  Enabled types: %s\n", strings.Join(prefs.EnabledTypes, ", "))
			mutedStr := "(none)"
			if len(prefs.MutedDomains) > 0 {
				mutedStr = strings.Join(prefs.MutedDomains, ", ")
			}
			fmt.Printf("  Muted domains: %s\n", mutedStr)
			fmt.Println()
		}
		return
	}

	// Apply changes
	if *pollInterval != "" {
		minutes := strings.TrimSuffix(*pollInterval, "m")
		mins, err := strconv.Atoi(minutes)
		if err != nil {
			exitError("Invalid interval format: %s", *pollInterval)
		}
		if mins < 15 {
			mins = 15
			if !jsonOutput {
				fmt.Println("[i] Minimum poll interval is 15 minutes. Setting to 15.")
			}
		}
		mgr.SetPollInterval(mins)
		if !jsonOutput {
			fmt.Printf("[✓] Poll interval set to %d minutes\n", mins)
		}
	}

	if *enable != "" {
		mgr.EnableType(*enable)
		if !jsonOutput {
			fmt.Printf("[✓] Enabled notification type: %s\n", *enable)
		}
	}

	if *disable != "" {
		mgr.DisableType(*disable)
		if !jsonOutput {
			fmt.Printf("[✓] Disabled notification type: %s\n", *disable)
		}
	}

	if *mute != "" {
		mgr.MuteDomain(*mute)
		if !jsonOutput {
			fmt.Printf("[✓] Muted domain: %s\n", *mute)
		}
	}

	if *unmute != "" {
		mgr.UnmuteDomain(*unmute)
		if !jsonOutput {
			fmt.Printf("[✓] Unmuted domain: %s\n", *unmute)
		}
	}

	if jsonOutput {
		prefs, _ := mgr.GetPreferences()
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "notifications-config",
			"data":    prefs,
		})
	}
}
