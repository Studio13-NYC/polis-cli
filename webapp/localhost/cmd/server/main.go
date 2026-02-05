package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/vdibart/polis-cli/webapp/localhost/internal/server"
)

//go:embed www/*
var webUI embed.FS

func main() {
	// Find data directory relative to executable
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	execDir := filepath.Dir(execPath)
	dataDir := filepath.Join(execDir, "data")

	// Get the embedded web UI filesystem
	webFS, err := fs.Sub(webUI, "www")
	if err != nil {
		log.Fatal("Failed to create sub filesystem:", err)
	}

	// Run the server
	server.Run(webFS, dataDir)
}
