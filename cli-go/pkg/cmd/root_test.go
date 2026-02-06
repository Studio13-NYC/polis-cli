package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify essential commands are listed
	expectedCommands := []string{
		"init",
		"render",
		"post",
		"republish",
		"comment",
		"preview",
		"extract",
		"blessing",
		"follow",
		"unfollow",
		"discover",
		"notifications",
		"register",
		"unregister",
		"migrate",
		"clone",
		"rebuild",
		"index",
		"version",
		"about",
		"rotate-key",
		"serve",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected usage to contain command %q", cmd)
		}
	}

	// Verify global flags are mentioned
	if !strings.Contains(output, "--data-dir") {
		t.Error("Expected usage to mention --data-dir flag")
	}
	if !strings.Contains(output, "--json") {
		t.Error("Expected usage to mention --json flag")
	}

	// Verify init options are listed
	initOptions := []string{
		"--site-title",
		"--register",
		"--keys-dir",
		"--posts-dir",
		"--comments-dir",
		"--snippets-dir",
		"--versions-dir",
	}
	for _, opt := range initOptions {
		if !strings.Contains(output, opt) {
			t.Errorf("Expected usage to contain init option %q", opt)
		}
	}
}

func TestVersion(t *testing.T) {
	// Save original version
	oldVersion := Version
	defer func() { Version = oldVersion }()

	Version = "test-version-1.2.3"

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleVersion([]string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test-version-1.2.3") {
		t.Errorf("Expected version output to contain %q, got %q", "test-version-1.2.3", output)
	}
}

func TestGetDataDirDefault(t *testing.T) {
	// Save original dataDir
	oldDataDir := dataDir
	defer func() { dataDir = oldDataDir }()

	dataDir = ""
	cwd, _ := os.Getwd()
	result := getDataDir()

	if result != cwd {
		t.Errorf("Expected getDataDir() to return cwd %q, got %q", cwd, result)
	}
}

func TestGetDataDirOverride(t *testing.T) {
	// Save original dataDir
	oldDataDir := dataDir
	defer func() { dataDir = oldDataDir }()

	dataDir = "/custom/path"
	result := getDataDir()

	if result != "/custom/path" {
		t.Errorf("Expected getDataDir() to return %q, got %q", "/custom/path", result)
	}
}
