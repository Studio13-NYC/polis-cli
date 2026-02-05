// Package hooks provides post-action automation for polis events.
package hooks

// TaskTemplate represents a pre-built automation task template.
type TaskTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Script      string    `json:"script"`
	Event       HookEvent `json:"event"`
}

// TaskTemplates contains pre-built templates for common automation tasks.
var TaskTemplates = map[string]TaskTemplate{
	"vercel": {
		ID:          "vercel",
		Name:        "Deploy to Vercel",
		Description: "Commit and push after publish, triggering Vercel deployment",
		Event:       EventPostPublish,
		Script: `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"
git push
`,
	},
	"github-pages": {
		ID:          "github-pages",
		Name:        "Deploy to GitHub Pages",
		Description: "Commit and push after publish for GitHub Pages",
		Event:       EventPostPublish,
		Script: `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"
git push
`,
	},
	"git-commit": {
		ID:          "git-commit",
		Name:        "Auto-commit to git",
		Description: "Commit changes without pushing",
		Event:       EventPostPublish,
		Script: `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"
`,
	},
	"custom": {
		ID:          "custom",
		Name:        "Custom script",
		Description: "Run your own script after publish",
		Event:       EventPostPublish,
		Script: `#!/bin/bash
set -e
# Available environment variables:
# POLIS_SITE_DIR - path to your site directory
# POLIS_PATH - relative path to the published file
# POLIS_TITLE - title of the post
# POLIS_COMMIT_MESSAGE - suggested commit message

echo "Published: $POLIS_TITLE"
`,
	},
}

// GetTemplate returns a task template by ID.
func GetTemplate(id string) (TaskTemplate, bool) {
	t, ok := TaskTemplates[id]
	return t, ok
}

// ListTemplates returns all available task templates.
func ListTemplates() []TaskTemplate {
	templates := make([]TaskTemplate, 0, len(TaskTemplates))
	// Return in a consistent order
	for _, id := range []string{"vercel", "github-pages", "git-commit", "custom"} {
		if t, ok := TaskTemplates[id]; ok {
			templates = append(templates, t)
		}
	}
	return templates
}
