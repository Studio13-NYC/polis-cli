// polis is the Go CLI for the Polis decentralized social network.
package main

import (
	"os"

	"github.com/vdibart/polis-cli/cli-go/pkg/cmd"
)

// Version is set at build time with -ldflags
var Version = "dev"

func main() {
	// Set version from build-time ldflags
	cmd.Version = Version

	// Execute the CLI
	cmd.Execute(os.Args[1:])
}
