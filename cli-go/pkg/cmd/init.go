package cmd

import (
	"flag"
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	siteTitle := fs.String("title", "", "Site title")
	author := fs.String("author", "", "Author name")
	email := fs.String("email", "", "Author email")
	baseURL := fs.String("base-url", "", "Site base URL (e.g., https://alice.polis.pub)")
	fs.Parse(args)

	dir := getDataDir()

	opts := site.InitOptions{
		SiteTitle: *siteTitle,
		Author:    *author,
		Email:     *email,
		BaseURL:   *baseURL,
	}

	result, err := site.Init(dir, opts)
	if err != nil {
		exitError("Failed to initialize site: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"success":    result.Success,
			"site_dir":   result.SiteDir,
			"public_key": result.PublicKey,
		})
	} else {
		fmt.Printf("Initialized polis site at: %s\n", result.SiteDir)
		fmt.Printf("Public key: %s\n", result.PublicKey[:50]+"...")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Set POLIS_BASE_URL in .env file")
		fmt.Println("  2. Create your first post: polis publish post.md")
		fmt.Println("  3. Render the site: polis render")
	}
}
