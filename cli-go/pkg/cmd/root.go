// Package cmd provides the CLI command handlers for the polis CLI.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// Version is set at build time with -ldflags
var Version = "dev"

// Global flags
var (
	dataDir    string
	jsonOutput bool
)

// Execute is the main entry point for the CLI.
func Execute(args []string) {
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
	case "publish":
		handlePublish(cmdArgs)
	case "republish":
		handleRepublish(cmdArgs)
	case "comment":
		handleComment(cmdArgs)
	case "blessing":
		handleBlessing(cmdArgs)
	case "register":
		handleRegister(cmdArgs)
	case "unregister":
		handleUnregister(cmdArgs)
	case "version", "--version", "-v":
		fmt.Printf("polis version %s\n", Version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Polis - Decentralized Social Network CLI (Go)

Usage:
  polis [global-flags] <command> [options]

Commands:
  init                Initialize a new polis site
  validate            Validate site configuration
  render [--force]    Render markdown posts and comments to HTML
  publish <file>      Publish a new post
  republish <path>    Update an existing post
  comment <subcommand> Manage comments (draft, sign, list, sync)
  blessing <subcommand> Manage blessings (requests, grant, deny)
  register            Register site/post with discovery service
  unregister          Unregister from discovery service
  version             Show version information
  help                Show this help message

Global Flags:
  --data-dir <path>   Site data directory (default: current directory)
  --json              Output results in JSON format

Comment Subcommands:
  comment draft <url>   Create a comment draft
  comment sign <id>     Sign a pending comment
  comment list [status] List comments (drafts, pending, blessed, denied)
  comment sync          Sync pending comments with discovery service

Blessing Subcommands:
  blessing requests     List pending blessing requests on your posts
  blessing grant <ver>  Grant a blessing
  blessing deny <ver>   Deny a blessing

Examples:
  polis init
  polis render --force
  polis publish post.md
  polis comment draft https://example.com/posts/hello.md
  polis blessing requests
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
