package cmd

import (
	"fmt"

	"github.com/vdibart/polis-cli/cli-go/pkg/site"
)

func handleValidate(args []string) {
	dir := getDataDir()

	result := site.Validate(dir)

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":    result.Status,
			"errors":    result.Errors,
			"site_info": result.SiteInfo,
		})
	} else {
		fmt.Printf("Status: %s\n", result.Status)
		if len(result.Errors) > 0 {
			fmt.Println("Errors:")
			for _, err := range result.Errors {
				fmt.Printf("  - [%s] %s\n", err.Code, err.Message)
			}
		}
		if result.SiteInfo != nil {
			fmt.Printf("\nSite Info:\n")
			if result.SiteInfo.SiteTitle != "" {
				fmt.Printf("  Title: %s\n", result.SiteInfo.SiteTitle)
			}
			if result.SiteInfo.BaseURL != "" {
				fmt.Printf("  Base URL: %s\n", result.SiteInfo.BaseURL)
			}
			if result.SiteInfo.Generator != "" {
				fmt.Printf("  Generator: %s\n", result.SiteInfo.Generator)
			}
		}
	}
}
