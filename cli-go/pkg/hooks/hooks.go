// Package hooks provides post-action automation for polis events.
package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// HookEvent represents the type of event that triggers a hook.
type HookEvent string

const (
	// EventPostPublish is triggered when a new post is published.
	EventPostPublish HookEvent = "post-publish"
	// EventPostRepublish is triggered when an existing post is updated.
	EventPostRepublish HookEvent = "post-republish"
	// EventPostComment is triggered when a comment is blessed.
	EventPostComment HookEvent = "post-comment"
)

// HookConfig contains paths to hook scripts.
type HookConfig struct {
	PostPublish   string `json:"post-publish,omitempty"`
	PostRepublish string `json:"post-republish,omitempty"`
	PostComment   string `json:"post-comment,omitempty"`
}

// HookPayload contains data passed to hook scripts.
type HookPayload struct {
	Event         HookEvent `json:"event"`
	Path          string    `json:"path"`
	Title         string    `json:"title"`
	Version       string    `json:"version"`
	Timestamp     string    `json:"timestamp"`
	CommitMessage string    `json:"commit_message"`
}

// HookResult contains the result of running a hook.
type HookResult struct {
	Executed bool   `json:"executed"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// RunHook executes a hook script if configured.
// Returns nil error if no hook is configured (not an error condition).
func RunHook(siteDir string, config *HookConfig, payload *HookPayload) (*HookResult, error) {
	if config == nil {
		return &HookResult{Executed: false}, nil
	}

	// Get hook path from config based on event type
	var hookPath string
	switch payload.Event {
	case EventPostPublish:
		hookPath = config.PostPublish
	case EventPostRepublish:
		hookPath = config.PostRepublish
	case EventPostComment:
		hookPath = config.PostComment
	default:
		return &HookResult{Executed: false}, nil
	}

	if hookPath == "" {
		return &HookResult{Executed: false}, nil
	}

	// Resolve relative paths from site root
	if !filepath.IsAbs(hookPath) {
		hookPath = filepath.Join(siteDir, hookPath)
	}

	// Check if hook exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("hook not found: %s", hookPath)
	}

	// Build environment variables
	configDir := filepath.Join(siteDir, ".polis")
	env := os.Environ()
	env = append(env,
		"POLIS_EVENT="+string(payload.Event),
		"POLIS_PATH="+payload.Path,
		"POLIS_TITLE="+payload.Title,
		"POLIS_VERSION="+payload.Version,
		"POLIS_TIMESTAMP="+payload.Timestamp,
		"POLIS_SITE_DIR="+siteDir,
		"POLIS_CONFIG_DIR="+configDir,
		"POLIS_COMMIT_MESSAGE="+payload.CommitMessage,
	)

	// Execute hook
	cmd := exec.Command(hookPath)
	cmd.Env = env
	cmd.Dir = siteDir // Run in site directory

	// Pass JSON payload to stdin
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook payload: %w", err)
	}
	cmd.Stdin = bytes.NewReader(jsonPayload)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &HookResult{
			Executed: true,
			Output:   string(output),
			Error:    err.Error(),
		}, fmt.Errorf("hook failed: %w\nOutput: %s", err, output)
	}

	return &HookResult{
		Executed: true,
		Output:   string(output),
	}, nil
}

// GenerateCommitMessage generates a git commit message for the given event.
func GenerateCommitMessage(event HookEvent, title string) string {
	switch event {
	case EventPostPublish:
		return fmt.Sprintf("Publish: %s", title)
	case EventPostRepublish:
		return fmt.Sprintf("Update: %s", title)
	case EventPostComment:
		return fmt.Sprintf("Comment blessed: %s", title)
	default:
		return fmt.Sprintf("Polis: %s", title)
	}
}

// GetHookPath returns the configured hook path for a given event.
// Returns empty string if no hook is configured.
func GetHookPath(config *HookConfig, event HookEvent) string {
	if config == nil {
		return ""
	}

	switch event {
	case EventPostPublish:
		return config.PostPublish
	case EventPostRepublish:
		return config.PostRepublish
	case EventPostComment:
		return config.PostComment
	default:
		return ""
	}
}
