package cmd

import (
	"fmt"
	"os"
)

// ServeHandler is the function that handles the serve command.
// In the CLI-only binary, this prints a message directing users to the bundled binary.
// In the bundled binary, this is overridden by the webapp's serve implementation.
var ServeHandler func(args []string) = defaultServeHandler

func defaultServeHandler(args []string) {
	fmt.Fprintln(os.Stderr, "The serve command requires the bundled binary (polis-full).")
	fmt.Fprintln(os.Stderr, "Download from: https://github.com/vdibart/polis-cli/releases")
	os.Exit(1)
}

func handleServe(args []string) {
	ServeHandler(args)
}
