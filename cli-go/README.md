# Polis Go CLI

Go implementation of the Polis CLI with full feature parity with the bash CLI.
This is the **recommended** implementation.

## Version

Current version: **0.45.0** (full feature parity with Bash CLI)

## Building

**Prerequisites:** Go 1.21+

```bash
# Development build
go build ./cmd/polis

# Run
./polis help
./polis version
```

## Distribution Builds

```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o polis-linux-amd64 ./cmd/polis

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o polis-darwin-arm64 ./cmd/polis

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o polis-darwin-amd64 ./cmd/polis

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o polis-windows-amd64.exe ./cmd/polis
```

## Package Structure

Core packages in `pkg/` are designed to be imported by other Go applications (23 packages):

| Package | Description |
|---------|-------------|
| `pkg/blessing` | Blessing workflow (requests, grant, deny, beseech, sync) |
| `pkg/clone` | Remote site cloning |
| `pkg/cmd` | CLI command handlers |
| `pkg/comment` | Comment management |
| `pkg/discovery` | Discovery service HTTP client |
| `pkg/following` | following.json management |
| `pkg/hooks` | Post-action automation |
| `pkg/index` | Index rebuilding |
| `pkg/metadata` | Public index (JSONL) management |
| `pkg/migrate` | Domain migration |
| `pkg/notification` | Local notification CRUD |
| `pkg/publish` | Post publishing logic |
| `pkg/remote` | HTTP fetching for remote polis sites |
| `pkg/render` | Markdown to HTML conversion and page rendering |
| `pkg/signing` | Ed25519 cryptographic signing |
| `pkg/site` | Site validation and initialization |
| `pkg/snippet` | Snippet file management |
| `pkg/template` | Mustache-like template engine |
| `pkg/theme` | Theme template loading |
| `pkg/url` | URL normalization utilities |
| `pkg/verify` | Remote content signature/hash verification |
| `pkg/version` | Version history parsing/reconstruction |

## Commands

All 27 commands from the bash CLI are implemented:

| Command | Description |
|---------|-------------|
| `about` | Show site configuration |
| `beseech` | Send blessing request to discovery service |
| `blessing deny` | Deny a pending blessing request |
| `blessing grant` | Grant a pending blessing request |
| `blessing requests` | List pending blessing requests |
| `blessing sync` | Sync blessings from discovery service |
| `clone` | Clone a remote polis site |
| `comment` | Add a comment to a post |
| `discover` | Discover content from followed sites |
| `extract` | Extract post content |
| `follow` | Follow a polis site |
| `help` | Show help message |
| `index` | Manage public index |
| `init` | Initialize a new polis site |
| `migrate` | Migrate site to new domain |
| `notifications` | Manage local notifications |
| `post` | Publish a new post |
| `preview` | Preview content in browser |
| `publish` | Alias for post |
| `rebuild` | Rebuild site from history |
| `render` | Render markdown to HTML |
| `rotate-key` | Rotate signing key |
| `serve` | Start local HTTP server (bundled binary only) |
| `unfollow` | Unfollow a polis site |
| `version` | Show version information |

## Usage as Library

```go
import (
    "github.com/vdibart/polis-cli/cli-go/pkg/render"
    "github.com/vdibart/polis-cli/cli-go/pkg/publish"
)

// Render all pages
renderer, _ := render.NewPageRenderer(render.PageConfig{
    DataDir:       "/path/to/site",
    CLIThemesDir:  "/path/to/themes",
    BaseURL:       "https://example.com",
    RenderMarkers: false,
})
stats, _ := renderer.RenderAll(true)

// Publish a post
result, _ := publish.PublishPost(dataDir, content, filename, privateKey)
```

## Template System

The Go CLI implements the same Mustache-like template syntax as the bash CLI:

- `{{variable}}` - Variable substitution
- `{{> snippet}}` - Partial includes (global-first by default)
- `{{> theme:snippet}}` - Theme-first lookup
- `{{#section}}...{{/section}}` - Loop blocks

Resolution order for snippets without explicit extension: `.md` → `.html` → exact

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./pkg/render/...
```
