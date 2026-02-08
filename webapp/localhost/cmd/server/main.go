package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/vdibart/polis-cli/webapp/localhost/internal/server"
	"github.com/vdibart/polis-cli/webapp/localhost/internal/webui"
)

func main() {
	// Find data directory relative to executable
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	execDir := filepath.Dir(execPath)
	dataDir := filepath.Join(execDir, "data")

	// Get the embedded web UI filesystem
	webFS, err := fs.Sub(webui.Assets, "www")
	if err != nil {
		log.Fatal("Failed to create sub filesystem:", err)
	}

	// Run the server
	server.Run(webFS, dataDir)
}
