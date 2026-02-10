// Package cmd provides the CLI command handlers for the polis CLI.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vdibart/polis-cli/cli-go/pkg/comment"
	"github.com/vdibart/polis-cli/cli-go/pkg/feed"
	"github.com/vdibart/polis-cli/cli-go/pkg/following"
	"github.com/vdibart/polis-cli/cli-go/pkg/index"
	"github.com/vdibart/polis-cli/cli-go/pkg/metadata"
	"github.com/vdibart/polis-cli/cli-go/pkg/notification"
	"github.com/vdibart/polis-cli/cli-go/pkg/publish"
	"github.com/vdibart/polis-cli/cli-go/pkg/site"
	"github.com/vdibart/polis-cli/cli-go/pkg/stream"
	"github.com/vdibart/polis-cli/cli-go/pkg/theme"
)

// Version is set at build time with -ldflags
var Version = "dev"

// Global flags
var (
	dataDir    string
	jsonOutput bool
)

// DefaultDiscoveryServiceURL is the default discovery service URL.
const DefaultDiscoveryServiceURL = "https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1"

// Execute is the main entry point for the CLI.
func Execute(args []string) {
	// Propagate CLI version to all packages that embed it in metadata
	publish.Version = Version
	comment.Version = Version
	metadata.Version = Version
	following.Version = Version
	index.Version = Version
	notification.Version = Version
	theme.Version = Version
	feed.Version = Version
	site.Version = Version

	// Propagate discovery config to packages that register with discovery
	discoveryURL := os.Getenv("DISCOVERY_SERVICE_URL")
	if discoveryURL == "" {
		discoveryURL = DefaultDiscoveryServiceURL
	}
	discoveryKey := os.Getenv("DISCOVERY_SERVICE_KEY")
	baseURL := os.Getenv("POLIS_BASE_URL")

	publish.DiscoveryURL = discoveryURL
	publish.DiscoveryKey = discoveryKey
	publish.BaseURL = baseURL

	comment.DiscoveryURL = discoveryURL
	comment.DiscoveryKey = discoveryKey
	comment.BaseURL = baseURL

	stream.DiscoveryURL = discoveryURL
	stream.DiscoveryKey = discoveryKey
	stream.BaseURL = baseURL

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// Parse global flags
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--data-dir" && i+1 < len(args):
			dataDir = args[i+1]
			i++ // Skip the next arg (the value)
		case len(arg) > 11 && arg[:11] == "--data-dir=":
			dataDir = arg[11:]
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := filteredArgs[0]
	cmdArgs := filteredArgs[1:]

	switch command {
	case "init":
		handleInit(cmdArgs)
	case "validate":
		handleValidate(cmdArgs)
	case "render":
		handleRender(cmdArgs)
	case "post":
		handlePublish(cmdArgs)
	case "republish":
		handleRepublish(cmdArgs)
	case "comment":
		handleComment(cmdArgs)
	case "preview":
		handlePreview(cmdArgs)
	case "extract":
		handleExtract(cmdArgs)
	case "index":
		handleIndex(cmdArgs)
	case "about":
		handleAbout(cmdArgs)
	case "follow":
		handleFollow(cmdArgs)
	case "unfollow":
		handleUnfollow(cmdArgs)
	case "discover":
		handleDiscover(cmdArgs)
	case "blessing":
		handleBlessing(cmdArgs)
	case "rebuild":
		handleRebuild(cmdArgs)
	case "migrate":
		handleMigrate(cmdArgs)
	case "migrations":
		if len(cmdArgs) > 0 && cmdArgs[0] == "apply" {
			handleMigrationsApply(cmdArgs[1:])
		} else {
			exitError("Unknown migrations subcommand. Use: polis migrations apply")
		}
	case "rotate-key":
		handleRotateKey(cmdArgs)
	case "notifications":
		handleNotifications(cmdArgs)
	case "clone":
		handleClone(cmdArgs)
	case "register":
		handleRegister(cmdArgs)
	case "unregister":
		handleUnregister(cmdArgs)
	case "serve":
		handleServe(cmdArgs)
	case "version", "--version", "-v":
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"command": "version",
				"data": map[string]interface{}{
					"version": Version,
				},
			})
		} else {
			fmt.Printf("polis %s\n", Version)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Polis - Decentralized Social Network CLI

Usage:
  polis [--json] <command> [options]

Global Flags:
  --json                          Output results in JSON format
  --data-dir <path>               Site data directory (default: current directory)

Commands related to creating or viewing content:
  polis post <file>               Create a new post
  polis comment <file> [url]      Create a comment on a post
  polis republish <file>          Update an already-published file
  polis preview <url>             Preview a post or comment with signature verification
  polis extract <file> <hash>     Reconstruct a specific version of a file

Commands related to requesting, reviewing, or granting blessings:
  polis blessing requests         List pending blessing requests
  polis blessing grant <hash>     Grant a blessing request by content hash
  polis blessing deny <hash>      Deny a blessing request by content hash
  polis blessing beseech <hash>   Re-request blessing by content hash
  polis blessing sync             Sync auto-blessed comments from discovery service

Commands related to following or unfollowing an author:
  polis follow <author-url>       Follow an author (auto-bless their comments)
  polis unfollow <author-url>     Unfollow an author

Commands related to content discovery:
  polis discover                  Check followed authors for new content
  polis discover --author <url>   Check a specific author
  polis discover --since <date>   Show items since date

Commands related to notifications:
  polis notifications             List unread notifications
  polis notifications list        List notifications (--type <types>)

Commands related to site administration:
  polis register                  Register site with discovery service
  polis unregister [--force]      Unregister site
  polis render [--force]          Render markdown to HTML
  polis migrate <new-domain>      Migrate content to a new domain
  polis migrations apply          Apply domain migrations to local files

Commands related to cloning remote polis sites:
  polis clone <url> [dir]         Clone a public polis site
  polis clone <url> --full        Re-download all content
  polis clone <url> --diff        Only download new/changed content

Commands related to local configuration:
  polis init [options]            Initialize Polis directory structure
    --site-title <title>          Site display name
    --register                    Auto-register with discovery service after init
    --keys-dir <path>             Custom keys directory (default: .polis/keys)
    --posts-dir <path>            Custom posts directory (default: posts)
    --comments-dir <path>         Custom comments directory (default: comments)
    --snippets-dir <path>         Custom snippets directory (default: snippets)
    --versions-dir <path>         Custom versions directory (default: .versions)
  polis rebuild --posts|--comments|--notifications|--all
                                  Rebuild indexes and reset state
  polis index                     View index
  polis version                   Print CLI version
  polis about                     Show site, versions, config info
  polis rotate-key                Generate new keypair and re-sign content
  polis serve [-d|--data-dir PATH] Start local web server (bundled binary only)

Examples:
  polis init
  polis post my-post.md
  polis comment my-comment.md https://example.com/posts/hello.md
  polis preview https://example.com/posts/hello.md
  polis blessing requests
  polis discover
`)
}

// getDataDir returns the data directory, defaulting to current working directory
func getDataDir() string {
	if dataDir != "" {
		return dataDir
	}
	cwd, err := os.Getwd()
	if err != nil {
		exitError("Failed to get working directory: %v", err)
	}
	return cwd
}

// exitError prints an error message and exits
func exitError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if jsonOutput {
		output := map[string]interface{}{
			"success": false,
			"error":   msg,
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}

// outputJSON outputs a JSON response
func outputJSON(data interface{}) {
	json.NewEncoder(os.Stdout).Encode(data)
}
