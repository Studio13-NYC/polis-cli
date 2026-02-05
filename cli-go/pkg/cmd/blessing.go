package cmd

import (
	"fmt"
	"os"

	"github.com/vdibart/polis-cli/cli-go/pkg/blessing"
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
	polisurl "github.com/vdibart/polis-cli/cli-go/pkg/url"
)

func handleBlessing(args []string) {
	if len(args) < 1 {
		printBlessingUsage()
		os.Exit(1)
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "requests", "list":
		handleBlessingRequests(subArgs)
	case "grant":
		handleBlessingGrant(subArgs)
	case "deny":
		handleBlessingDeny(subArgs)
	case "help", "--help", "-h":
		printBlessingUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown blessing subcommand: %s\n", subcommand)
		printBlessingUsage()
		os.Exit(1)
	}
}

func printBlessingUsage() {
	fmt.Print(`Usage: polis blessing <subcommand> [options]

Subcommands:
  requests              List pending blessing requests on your posts
  grant <version>       Grant a blessing to a comment
  deny <version>        Deny a blessing request

Examples:
  polis blessing requests
  polis blessing grant sha256:abc123...
  polis blessing deny sha256:abc123...
`)
}

func handleBlessingRequests(args []string) {
	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Load discovery config from env
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	discoveryKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}

	// Get domain from POLIS_BASE_URL
	baseURL := os.Getenv("POLIS_BASE_URL")
	if baseURL == "" {
		exitError("POLIS_BASE_URL not set")
	}
	domain := polisurl.ExtractDomain(baseURL)
	if domain == "" {
		exitError("Could not extract domain from POLIS_BASE_URL")
	}

	client := discovery.NewClient(discoveryURL, discoveryKey)

	requests, err := blessing.FetchPendingRequests(client, domain)
	if err != nil {
		exitError("Failed to fetch requests: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"requests": requests,
		})
	} else {
		if len(requests) == 0 {
			fmt.Println("No pending blessing requests.")
		} else {
			fmt.Printf("Pending blessing requests (%d):\n", len(requests))
			for _, req := range requests {
				fmt.Printf("\n  Version: %s\n", req.CommentVersion)
				fmt.Printf("  Author: %s\n", req.Author)
				fmt.Printf("  Reply to: %s\n", req.InReplyTo)
				fmt.Printf("  Comment URL: %s\n", req.CommentURL)
			}
		}
	}
}

func handleBlessingGrant(args []string) {
	if len(args) < 1 {
		exitError("Usage: polis blessing grant <comment-version>")
	}

	commentVersion := args[0]
	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Load private key
	privKey, err := loadPrivateKey(dir)
	if err != nil {
		exitError("Failed to load private key: %v", err)
	}

	// Load discovery config from env
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	discoveryKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}

	client := discovery.NewClient(discoveryURL, discoveryKey)

	// Grant the blessing
	result, err := blessing.GrantByVersion(dir, commentVersion, "", "", client, nil, privKey)
	if err != nil {
		exitError("Failed to grant blessing: %v", err)
	}

	if jsonOutput {
		outputJSON(result)
	} else {
		fmt.Printf("Blessed comment: %s\n", commentVersion)
		if result.CommentURL != "" {
			fmt.Printf("Comment URL: %s\n", result.CommentURL)
		}
	}
}

func handleBlessingDeny(args []string) {
	if len(args) < 1 {
		exitError("Usage: polis blessing deny <comment-version>")
	}

	commentVersion := args[0]
	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	// Load private key
	privKey, err := loadPrivateKey(dir)
	if err != nil {
		exitError("Failed to load private key: %v", err)
	}

	// Load discovery config from env
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	discoveryKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	if discoveryURL == "" {
		discoveryURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"
	}

	client := discovery.NewClient(discoveryURL, discoveryKey)

	// Deny the blessing
	result, err := blessing.Deny(commentVersion, client, privKey)
	if err != nil {
		exitError("Failed to deny blessing: %v", err)
	}

	if jsonOutput {
		outputJSON(result)
	} else {
		fmt.Printf("Denied blessing for: %s\n", commentVersion)
	}
}
