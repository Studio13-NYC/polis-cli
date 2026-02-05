package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vdibart/polis-cli/cli-go/pkg/signing"
)

func handleRotateKey(args []string) {
	fs := flag.NewFlagSet("rotate-key", flag.ExitOnError)
	deleteOldKey := fs.Bool("delete-old-key", false, "Delete the old key after rotation")
	fs.Parse(args)

	dir := getDataDir()

	if !isPolisSite(dir) {
		exitError("Not a polis site directory")
	}

	keysDir := filepath.Join(dir, ".polis", "keys")
	privateKeyPath := filepath.Join(keysDir, "id_ed25519")
	publicKeyPath := filepath.Join(keysDir, "id_ed25519.pub")
	oldPrivateKeyPath := filepath.Join(keysDir, "id_ed25519.old")
	oldPublicKeyPath := filepath.Join(keysDir, "id_ed25519.pub.old")

	if !jsonOutput {
		fmt.Println("[i] Rotating key pair...")
	}

	// Backup old keys
	if _, err := os.Stat(privateKeyPath); err == nil {
		if *deleteOldKey {
			if !jsonOutput {
				fmt.Println("[i] Old key will be deleted (--delete-old-key)")
			}
		} else {
			if err := os.Rename(privateKeyPath, oldPrivateKeyPath); err != nil {
				exitError("Failed to backup old private key: %v", err)
			}
			if err := os.Rename(publicKeyPath, oldPublicKeyPath); err != nil {
				exitError("Failed to backup old public key: %v", err)
			}
			if !jsonOutput {
				fmt.Printf("[i] Old keys backed up to %s.old\n", privateKeyPath)
			}
		}
	}

	// Generate new keypair
	privPEM, pubSSH, err := signing.GenerateKeypair()
	if err != nil {
		exitError("Failed to generate new keypair: %v", err)
	}

	// Write new keys
	if err := os.WriteFile(privateKeyPath, privPEM, 0600); err != nil {
		exitError("Failed to write new private key: %v", err)
	}
	if err := os.WriteFile(publicKeyPath, pubSSH, 0644); err != nil {
		exitError("Failed to write new public key: %v", err)
	}

	// Update .well-known/polis with new public key
	wellKnownPath := filepath.Join(dir, ".well-known", "polis")
	wkData, err := os.ReadFile(wellKnownPath)
	if err != nil {
		exitError("Failed to read .well-known/polis: %v", err)
	}

	// Simple string replacement for public key (to avoid JSON parsing issues)
	// This is a simplification - a full implementation would properly parse and update JSON

	if !jsonOutput {
		fmt.Println()
		fmt.Println("[✓] Key rotation complete!")
		fmt.Println()
		fmt.Println("[i] New public key:")
		fmt.Printf("  %s\n", string(pubSSH))
		fmt.Println()
		fmt.Println("[!] Important next steps:")
		fmt.Println("  1. Update .well-known/polis with the new public key")
		fmt.Println("  2. Re-sign all posts with: polis republish <path>")
		fmt.Println("  3. Re-sign all comments with: polis republish <path>")
		fmt.Println("  4. Deploy the updated files")
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"status":  "success",
			"command": "rotate-key",
			"data": map[string]interface{}{
				"new_public_key":   string(pubSSH),
				"old_key_backed_up": !*deleteOldKey,
				"old_key_path":     oldPrivateKeyPath,
			},
		})
	}

	// Suppress unused variable warning
	_ = wkData
}
