package cmd

import "fmt"

func handleVersion(args []string) {
	if jsonOutput {
		outputJSON(map[string]interface{}{
			"version": Version,
		})
	} else {
		fmt.Printf("polis version %s\n", Version)
	}
}
