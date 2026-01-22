# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.35.0] - 2026-01-22

### Added
- **Comment CTA on post pages** - Collapsible "Want to comment?" section on all default themes
  - Explains how Polis comments work (cryptographically signed, published from your own site)
  - Expandable setup instructions with code sample
  - Links to GitHub repo for full documentation
  - Uses native `<details>` element (no JavaScript required)

- **Enhanced snippet include syntax** - Full control over snippet resolution with tier prefixes and explicit extensions
  - **Explicit tier prefixes**: `{{> global:about}}` and `{{> theme:about}}` for fine-grained control
  - **Explicit extensions**: `{{> about.md}}` or `{{> about.html}}` skips fallback resolution
  - **Combined syntax**: `{{> theme:about.html}}` for complete specificity
  - Allows `about.html` (theme wrapper) to include `about.md` (author content) without naming collision

### Changed
- **Snippet lookup order reversed** - Global snippets (`./snippets/`) now take precedence over theme snippets
  - Previously: theme → global (theme won)
  - Now: global → theme (author wins)
  - Use `{{> theme:name}}` prefix to force theme-first behavior when needed

- **Theme templates updated** - All default themes now use explicit `{{> theme:...}}` prefix for snippets
  - Ensures themes work out of the box without relying on lookup precedence
  - Users can override by creating global snippets and changing templates to use `{{> name}}`

### Fixed
- **Manifest JSON validation** - `polis render` now validates manifest.json before processing
  - Shows clear error message with parse details if JSON is malformed
  - Previously failed silently when manifest had syntax errors

- **Snippet rendering error handling** - Pandoc failures in snippets now warn instead of crashing
  - Displays warning with snippet path and error details
  - Continues rendering with empty content instead of silent exit

- **Snippet debug output** - Fixed debug messages appearing in rendered HTML
  - Redirected `load_snippet` and `render_partials` debug output to stderr
  - Previously, info/warning messages were captured as part of snippet content

- **Missing link styling in about section** - Added `.about-content a` CSS rules to all themes
  - Links in the about snippet now use theme colors (teal/cyan) instead of browser defaults
  - Affects sols, turbo, and zane themes

## [0.34.0] - 2026-01-21

### Added
- **Notifications system (Phase 1)** - Local notification tracking with version checking
  - New `polis notifications` command with subcommands: `list`, `read`, `dismiss`, `sync`, `config`
  - Track notifications locally in `.polis/notifications.jsonl` (log) and `.polis/notifications-manifest.json` (preferences/state)
  - Support for notification types: `version_available`, `version_pending`, `new_follower`, `new_post`, `blessing_changed`
  - Filter notifications by type, show all (including read), mark as read, dismiss old notifications
  - Sync from discovery service with optional full reset
  - Configure notification preferences: polling intervals, enable/disable, mute/unmute specific types

- **Version checking in `polis about`** - Automatic upgrade availability detection
  - Shows "→ X.Y.Z available" when a new CLI version is released
  - Warns when metadata files need rebuild after upgrade
  - Displays unread notification count with prompt to view notifications

- **Discovery service version tracking** - Infrastructure for CLI version notifications
  - New `polis_versions` table tracks CLI releases with version, release notes, download URL, and checksum
  - `GET /polis-version` endpoint returns latest version with `upgrade_available` flag
  - Rate limited: 100 requests/hour (global, no authentication required)

- **Discovery service rate limiting** - Per-domain rate limiting infrastructure
  - New `rate_limits` table tracks requests per domain/endpoint/time window
  - Atomic rate limit checking and increment via `increment_rate_limit()` RPC
  - Automatic cleanup of old rate limit entries (older than 24 hours)

- **Documentation**
  - New `docs/NOTIFICATIONS.md` - Comprehensive notification system documentation
  - Updated `docs/SECURITY-MODEL.md` - Added notifications security section covering signed requests, attack prevention, and privacy considerations
  - Updated `docs/USAGE.md` - Added notifications commands and `--announce` flag documentation

### Changed
- **Consolidated `polis manifest` into `polis rebuild`** - Manifest generation now automatic after rebuild operations
  - Removed standalone `polis manifest` command
  - Renamed `--content` flag to `--posts` for clarity
  - Added `--notifications` flag to reset notification files
  - `--all` flag now includes posts, comments, and notifications

- **Claude Code skill updated** - Comprehensive update to `/polis` skill with full command coverage
  - Rewrote Feature 6 (Notifications) with all subcommands documented
  - Added Feature 7 (Site Registration) for `register`/`unregister` commands
  - Added Feature 8 (Clone Remote Site) for `polis clone` command
  - Added Feature 9 (About / Site Info) for `polis about` command
  - Updated `commands.md` reference with all missing commands and flags
  - Updated `json-responses.md` with new response schemas

- **Shell completions enhanced** - Full notifications support and improved JSON mode handling
  - Added `notifications` subcommands: `list`, `read`, `dismiss`, `sync`, `config`
  - Added completion options for all notification subcommands (`--type`, `--all`, `--older-than`, `--reset`, etc.)
  - `--json` flag now offered at any position and for all commands
  - Added `--announce` completion for `follow`/`unfollow` commands
  - Zsh: Added argument type hints for file paths, URLs, and durations

### Fixed
- **`polis rebuild --notifications` path error** - Fixed undefined `$POLIS_DIR` variable
  - Was causing "Permission denied" error when attempting to write to `/notifications-manifest.json`
  - Now correctly writes to `.polis/notifications-manifest.json`

## [0.33.0] - 2026-01-20

### Added
- **Post metadata tracking** - Discovery service now tracks and indexes posts from registered authors
  - Posts can be registered, updated, queried, and removed from the discovery index
  - Only posts from registered authors are accepted (same privacy model as comments)
  - Enables discovery of content across the polis network
  - Schema version bumped to 0.5.0 for discovery service compatibility

### Changed
- **Theme layout improvements** - Refined post page header layout across all built-in themes (turbo, zane, sols)
  - Date and signature information now displayed on the same line in post header
  - Date left-aligned, signature info ("Signed by ... | Version ...") right-aligned
  - Increased content width from 720px to 900px for better readability
  - Centered content container for improved visual balance
  - Signature information moved from footer to header for better visibility
  - Index page hero/title alignment now matches content edge
  - Simplified post header spacing and removed dividing rule

- **Theme visual enhancements** - Accessibility and visual polish for all default themes
  - Improved muted text contrast for better readability (WCAG compliance)
  - Added responsive breakpoints: tablet (768px) and small phone (480px)
  - Added keyboard focus states (`:focus-visible`) for accessibility
  - Turbo: Glow effects on navigation and post hover, increased border visibility, magenta accent for author names
  - Zane: Subtle shadows on content cards, gold accent for dates
  - Sols: Increased pink border visibility, added shadows for depth, sage accent for author names

### Fixed
- **Registration payload signatures** - Use compact JSON formatting to match server-side signature verification
  - Previously used pretty-printed JSON with newlines, causing signature mismatch errors
  - Affects `polis register` and `polis unregister` commands
  - Ensures cryptographic signatures verify correctly on discovery service

- **Attestation verification** - Fixed signature verification when checking registration status
  - Now uses proper allowed_signers file format for `ssh-keygen -Y verify`
  - Format requires `<principal> <key-type> <key>` instead of raw public key
  - Fixes "Attestation Verification: FAILED" error when running `polis register` on already-registered sites

## [0.32.0] - 2026-01-18

### Added
- **Shell completion support** - Tab completion for bash and zsh shells to improve command-line productivity
  - Complete all 24 polis commands (type `polis i<tab>` to expand to `init`)
  - Complete subcommands for `blessing` and `migrations` namespaces
  - Complete the global `--json` flag
  - Includes installation scripts: `completions/polis.bash` and `completions/polis.zsh`
  - Detailed setup instructions added to USAGE.md for both bash and zsh

## [0.31.0] - 2026-01-18

### Added
- **Auto-registration on init** - New `polis init --register` flag automatically registers your site with the discovery service after initialization
  - Requires `POLIS_BASE_URL` and discovery service credentials to be configured
  - Shows helpful warnings if prerequisites are missing (initialization still succeeds)
  - Streamlines the setup process for new sites joining the network

- **Improved help text** - `polis init --site-title` flag now documented in built-in help output

### Changed
- **Consolidated configuration display** - `polis config` command merged into `polis about` for unified system information
  - `polis about` now shows: site info (URL, title), version details (CLI, .well-known/polis, following.json, blessed-comments.json, manifest.json), configuration paths, key status and fingerprint, discovery service configuration, and project details
  - Supports both human-readable and `--json` output modes
  - Provides single command for complete system status overview

- **TUI about screen redesign** - TUI version bumped to 0.6.0 with enhanced about display
  - Now shows SITE, VERSIONS, KEYS, DISCOVERY, and PROJECT sections matching CLI output
  - Fixed "not configured" display bug that occurred after `polis config` command removal
  - Consistent user experience between CLI and TUI interfaces

### Removed
- **`polis config` command** - Deprecated in favor of `polis about` which provides all the same information with consistent `--json` support

## [0.30.0] - 2026-01-18

### Added
- **Glossary documentation** - New `docs/GLOSSARY.md` with definitions for 20 polis-specific terms including post, comment, blessing, snippet, theme, signature, beseech, discovery service, render, manifest, frontmatter, and more. Quick reference for understanding polis terminology.

### Fixed
- **Theme CSS specificity** - Fixed `.post-meta` styling conflicts across all three built-in themes
  - Scoped `.post-meta` styles to `.post-content` context to prevent nav bar conflicts
  - Added separate `.site-footer .post-meta` styling for footer placement
  - Affects turbo, zane, and sols themes

- **Turbo theme post layout** - Moved post date from navigation bar to content area
  - Date now appears in bottom right of post content (`.post-meta` section)
  - Simplified navigation bar to only show "← Home" link
  - Cleaned up unused flexbox styles from navigation

### Changed
- **Documentation reorganization** - Removed duplicate theme customization content from USAGE.md
  - Theme customization, template variables, and mustache syntax now consolidated in TEMPLATING.md
  - USAGE.md now links to TEMPLATING.md for detailed theming information
  - Reduces confusion from maintaining same content in multiple locations

- **Template documentation syntax** - Updated HTML comment examples in TEMPLATING.md
  - Removed mustache `{{> }}` syntax from comments (was being expanded by template engine)
  - Comments now use plain snippet names (e.g., `about` instead of `{{> about}}`)

- **Polis branding guidance** - Softened branding language in TEMPLATING.md
  - Changed from requirement to friendly suggestion for keeping cyan logo color
  - "Keep the cyan color..." → "We'd appreciate it if you keep the cyan color..."

## [0.29.0] - 2026-01-18

### Added
- **Site registration** - List your site in the public directory to make your content discoverable
  - `polis register` - Register your site in the discovery service directory (idempotent)
  - `polis unregister [--force]` - Remove your site from the directory (requires confirmation)
  - Registration status now displayed in `polis about` output
  - TUI: Register/Unregister options added to Admin menu for easy access

- **Attestation verification** - Client-side verification of discovery service signatures
  - When already registered, `polis register` verifies the server's attestation signature
  - Uses discovery service's public key to validate that registration was properly signed
  - Displays verification status: Valid, Failed, or Not checked
  - Ensures server-side attestations are cryptographically verifiable by clients

### Changed
- **BREAKING: `POLIS_ENDPOINT_BASE` renamed to `DISCOVERY_SERVICE_URL`**
  - Update your environment variables and `.env` files to use the new name
  - No backward compatibility - the old variable name is no longer recognized
  - Improves naming clarity and consistency across the codebase

- **BREAKING: Network participation now requires site registration**
  - Both comment author AND target site must be listed in the public directory
  - Unlisted sites cannot participate in cross-site conversations
  - Error codes: `AUTHOR_NOT_REGISTERED`, `TARGET_NOT_REGISTERED`
  - Register your site with `polis register` to participate in the network

- **Registration messaging** - Updated terminology to emphasize discoverability and community
  - "Enable blessing flow" → "Make your site publicly discoverable"
  - "Beseech blessings" → "Discover content and engage with posts"
  - Focus on joining the public directory for networking, not just technical feature enablement

### Discovery Service
- New Edge Functions for site registration:
  - `sites-register` - Register a site with Ed25519 signature verification
  - `sites-unregister` - Unregister a site (hard delete for privacy)
  - `sites-check` - Check if a domain is registered
  - `sites-public-key` - Get discovery service's public key for attestation verification
- New `registered_sites` database table with dual-signature scheme
- Added `isRegistered()` helper for registration checks in blessing flow

## [0.28.0] - 2026-01-18

### Changed
- **BREAKING: `polis publish` renamed to `polis post`**
  - Command syntax: `polis post <file>` (was `polis publish <file>`)
  - JSON mode: `polis --json post` or `polis post --json`
  - JSON responses now return `"command": "post"` instead of `"publish"`
  - Scripts and workflows using `polis publish` must be updated to use `polis post`
  - Test files renamed to reflect new command name

- **Post template layout improvements**
  - Post date moved to bottom right corner of content area
  - Navigation bar simplified to display only "← Home" link
  - Cleaner visual hierarchy and improved readability on post pages
  - Applied to sols and zane themes

### Fixed
- **`--json` flag positioning** - Now works as first OR last argument
  - `polis --json post file.md` (flag at start)
  - `polis post file.md --json` (flag at end)
  - Improves ergonomics for scripting and interactive use

## [0.27.0] - 2026-01-16

### Added
- **Theme system** - Self-contained styling packages with templates, CSS, and snippets
  - Ships with 3 built-in themes: turbo (retro computing), zane (neutral dark), sols (violet/peach)
  - Themes stored at `themes/` in the distribution, copied to `.polis/themes/` on init
  - Each theme contains: index.html, post.html, comment.html, comment-inline.html, theme CSS, snippets/
  - THEMES_DIR configurable via environment, .env, or .well-known/polis config

- **First-render theme selection** - Automatic theme setup on first render
  - Randomly selects from available themes if `active_theme` not set in manifest
  - Copies theme CSS to `styles.css` at site root
  - Theme selection stored in `manifest.json` for persistence

- **Snippets** - New signed content type for reusable template fragments
  - `polis snippet <file>` - Sign and publish snippets to `snippets/` directory
  - `polis snippet -` - Create snippets from stdin (supports `--filename`, `--title` flags)
  - Snippets tracked in `metadata/snippets.jsonl` (parallel to `public.jsonl`)
  - Both `.md` (pandoc processed) and `.html` (as-is) formats supported
  - Two-level lookup: theme snippets first, then global snippets directory

- **Mustache templating syntax** - Standard partial and loop syntax for composable templates
  - `{{> path/to/snippet}}` - Include snippets in templates (recursive up to 10 levels)
  - `{{#posts}}...{{/posts}}` - Loop over posts with item variables
  - `{{#comments}}...{{/comments}}` - Loop over outgoing comments
  - `{{#blessed_comments}}...{{/blessed_comments}}` - Loop over blessed comments on posts

- **Post navigation bar** - Home link and date display on post pages
  - `{{home_path}}` variable provides relative links back to homepage
  - Navigation bar styled with theme accent colors
  - Displays "← Home" link and post publication date

### Changed
- **`polis init`** - Now installs theme files alongside initialization
  - Looks for `themes/` directory alongside polis script (follows symlinks)
  - Copies themes to `.polis/themes/` (or custom location via `--themes-dir` flag)
  - Warns if themes directory not found but continues initialization
  - Adds `directories.themes` to `.well-known/polis` config

- **`polis render`** - Enhanced with theme system support
  - On first render, auto-selects theme and saves to manifest
  - Loads templates from active theme directory
  - Copies theme CSS to styles.css on each render

- **`polis republish`** - Now auto-detects content type by path
  - `posts/**` paths republish as posts
  - `comments/**` paths republish as comments
  - `snippets/**` paths republish as snippets
  - Falls back to frontmatter `type` field if path doesn't match

### Fixed
- **Theme persistence** - `active_theme` now preserved when manifest regenerates after render
- **Multiline template sections** - `{{#section}}...{{/section}}` blocks now handle multiline content correctly
- **Nested content CSS paths** - Posts/comments use relative `{{css_path}}` (e.g., `../../styles.css`) for correct styling
- **Post template duplicate h1** - Removed duplicate title from hero section; markdown `#` heading is now the only h1
- **Navigation bar layout** - Navigation items now grouped together instead of spread across full width

### Removed
- **`polis render --init-templates`** - Replaced by theme system
  - Templates now come from themes, not a separate init command
  - Custom templates should be created as a custom theme

- **`.polis/templates/` directory** - Replaced by `.polis/themes/`
  - Migration guide available in TEMPLATING.md for existing customizations

### Documentation
- **TEMPLATING.md** - Complete rewrite as comprehensive theme system guide
  - Theme structure and file organization
  - Snippet lookup order and override mechanism
  - Theme developer's guide with CSS conventions
  - Migration guide from old template system
- **USAGE.md** - Updated directory structure and `polis init` options
- **JSON-MODE.md** - Removed `--init-templates` response format documentation

## [0.26.0] - 2026-01-15

### Changed
- **Command rename** - `polis get-version` renamed to `polis extract` for clarity
  - More intuitive name for extracting specific versions from version history
  - Updated help text and usage messages to reflect new command name
- **Template title handling** - Removed `<h1>{{title}}</h1>` from default templates
  - Markdown's first `# Heading` now becomes the `<h1>` in rendered HTML
  - Frontmatter `title` field only used for `<title>` tag (browser tab/SEO)
  - Allows different display heading vs document title if desired
  - Reverses the "duplicate title fix" from v0.24.0

### Fixed
- **Version reconstruction reliability** - Improved `polis extract` command robustness
  - Now tries efficient backward reconstruction first, falls back to forward reconstruction
  - Handles edge cases where canonical file has been modified during republish
  - Better error messages when version chain is broken
- **Hash verification compatibility** - Fixed verification for content published with different hashing methods
  - Tries canonical hashing first (matches current publish behavior)
  - Falls back to non-canonical hashing for backwards compatibility
  - Ensures old content can still be verified correctly
- **Republish detection** - `polis republish` now detects when content is unchanged
  - Compares canonical hashes to determine if update is needed
  - Skips version history update when content hash is identical
  - Prevents unnecessary version entries for no-op edits
- **Blank line handling** - Fixed double blank lines between frontmatter and content
  - `extract_content_without_frontmatter` already includes separator blank line
  - Removed redundant blank line insertions in publish and republish
  - Ensures consistent spacing in generated files

## [0.25.0] - 2026-01-14

### Added
- **Site title support** - Optional branding with `--site-title` flag during initialization
  - Set during init: `polis init --site-title "My Blog"`
  - Stored in `manifest.json` and preserved across rebuilds
  - Used in rendered HTML templates via `{{site_title}}` variable
  - Displayed in `polis about`, `polis config`, and TUI About screen
  - Falls back to domain name if not configured
- **Comment author names from manifest** - Blessed comments now display author's site title
  - Fetches remote `manifest.json` to get commenter's site title
  - Falls back to domain name if title not available
  - Local comments use local manifest for author name

### Changed
- **`polis config`** - Added new "Site" section displaying URL and title
  - JSON output includes `data.site.url` and `data.site.title` fields
- **Documentation** - Updated USAGE.md and TEMPLATING.md with site title configuration details

### Fixed
- **TUI About screen** - Corrected JSON path to properly display CLI version
  - Changed from `data.cli.version` to `data.versions.cli`

## [0.24.0] - 2026-01-13

### Fixed
- **Blessed comments not re-rendering** - `polis render` without `--force` now detects when blessed comments have been updated
  - Checks if `blessed-comments.json` is newer than HTML files for posts with blessed comments
  - Re-renders posts when new comments are blessed, even if the post markdown hasn't changed
  - Ensures blessed comments appear promptly after granting blessings
- **Duplicate titles in rendered HTML** - Fixed title appearing twice in rendered output
  - Template displays `{{title}}` from frontmatter, but post body also included the `# Title` heading
  - Now strips leading blank lines and first heading from markdown body before pandoc conversion
  - Applies to both post content and inline blessed comments for consistent formatting
- **TUI stats display** - Fixed post and comment counts in TUI dashboard stats bar
  - Now properly counts posts and comments from public.jsonl instead of just posts
  - Corrected stats text to show "Blessing Requests" instead of "Blessings"
- **TUI menu display** - Fixed text parameter not being displayed before menu options in TUI

## [0.23.0] - 2026-01-13

### Added
- **`polis discover`** - Check followed authors for new content
  - Efficiently polls each author's `manifest.json` to detect updates
  - Fetches full content indexes only when new posts or comments are detected
  - Updates `last_checked` timestamp in `following.json` to track discovery state
  - Displays summary of new content since last check
- **`polis manifest`** - Generate manifest.json summary file
  - Contains `last_published`, `post_count`, `comment_count`, and CLI version
  - Auto-generated by `publish`, `comment`, and `render` commands
  - Enables efficient discovery without downloading full content indexes
- **TUI Admin menu** - New maintenance operations submenu
  - Regenerate manifest.json
  - Rebuild site (normal or force re-render)
- **TUI Discover submenu** - Enhanced content discovery interface
  - "Check for new content" - runs `polis discover` CLI command
  - "Browse followed authors" - view and manage following list

### Changed
- **`polis config`** - Now displays manifest version and groups version information
- **`polis render --force`** - Now regenerates `manifest.json` during force rebuild
- **`polis init`** - Creates initial `manifest.json` during site setup

### Fixed
- **TUI Enter key not working** - Fixed menu selection with Enter key
  - Root cause: Exit code was overwritten before second condition could check it
- **TUI submenu navigation** - Fixed q/ESC in Admin and Discover menus
  - Now properly returns to main menu instead of exiting application
- **Comment JSON output** - Fixed to output single JSON object with nested beseech data
- **TUI startup performance** - Removed duplicate stats fetch that caused ~4 second delay on startup

## [0.22.0] - 2026-01-12

### Added
- **`polis about`** - Branded info screen with tagline, versions, and project links
- **`polis config`** - Configuration viewer showing all settings, paths, and key fingerprint (supports `--json`)
- **TUI comment filename prompt** - Now prompts for filename when creating comments, matching publish workflow
- **TUI init flexibility** - Offers to initialize polis in directories with non-conflicting files
  - Previously required empty or git-only directories
  - Now only checks for actual conflicts (existing `.polis/keys` or `.well-known/polis`)
- **TUI "View in browser"** - Added option to open previewed URLs and blessing review URLs in browser

### Changed
- **BREAKING: Content canonicalization** - Signatures and hashes now use canonicalized content
  - Trims trailing whitespace from each line and ensures exactly one trailing newline
  - Eliminates verification failures caused by HTTP transfer or editor differences
  - **Existing signed content must be re-signed** with `polis rotate-key`
- **`--filename` flag** - Now works in all modes for `publish` and `comment` commands (previously stdin-only)
- **Color scheme** - Replaced BLUE with CYAN throughout the CLI for better terminal contrast
- **License correction** - Fixed to AGPL v3 in `polis about` output (was incorrectly showing MIT)

### Fixed
- **Signature verification** - Fixed false "invalid signature" errors caused by mawk incompatibility
  - Rewrote `extract_body_from_content` to use pure bash instead of awk
- **Hash verification** - Fixed false "hash mismatch" errors from trailing newline differences
- **`--json` flag positioning** - Now works as first or last argument (e.g., `polis --json config`)
- **`polis index --json`** - No longer conflicts with global `--json` flag
- **Template overwrite protection** - `polis render --init-templates` no longer overwrites existing templates
- **Index sorting** - `index.html` now lists posts in descending date order (newest first)
- **Preview URL flexibility** - `polis preview` now tries alternate extensions (.html ↔ .md) when content not found

## [0.21.0] - 2026-01-12

### Added
- **polis-tui** - New interactive terminal user interface for Polis
  - Menu-driven dashboard with keyboard navigation
  - Real-time stats display (posts, following, blessings, git status)
  - Integrated workflows: publish, comment, blessing management, discover, preview
  - Post-action options: rebuild, git commit/push, or continue
  - Built-in git integration with detailed commit messages
  - About screen with version info and configuration
- **Documentation** - Added "Implementation Security Audit" section to SECURITY-MODEL.md
  - Comprehensive verification that private keys are never printed or logged
  - Audit of file permissions, git exclusion, temp file handling
  - Complete security checklist for key management practices

### Fixed
- **Blessed comments not rendering** - Fixed `polis render` failing to find blessed comments when URLs had mismatched extensions
  - Normalized `update_blessed_comments_json` to always store post URLs with `.md` extension
  - Now checks both `.md` and `.html` extensions when looking up posts

### Changed
- **README restructure** - Reorganized for better new user experience
  - New "Try it now" section featuring polis-tui as recommended path
  - Updated tagline: "Your content, free from platform control"
  - Command-line mode section for users who prefer CLI workflows

## [0.20.0] - 2026-01-10

### Fixed
- **Embedded source cleanup** - Removed YAML `---` delimiters from embedded frontmatter in HTML comments
  - Eliminates `--` escaping that produced `&#45;&#45;` artifacts
  - Cleaner output with just the frontmatter fields

## [0.19.0] - 2026-01-09

### Changed
- **Embedded source optimization** - `polis render` now embeds only frontmatter in HTML comments instead of full content
  - Avoids duplication since body content is already rendered in HTML
  - Source reference uses canonical URL instead of local file path
- **README documentation** - Added "Render to a deployable website" section
  - Documents `polis render` workflow and output
  - Explains template customization with `--init-templates`
  - Highlights verifiable HTML with embedded signed source

## [0.18.0] - 2026-01-09

### Added
- **Embedded source in rendered HTML** - Each rendered HTML file now includes the original markdown source and frontmatter in an HTML comment at the end
  - Enables verification that HTML matches the signed source
  - Allows extraction of original markdown from rendered files
  - Useful for debugging template rendering issues

### Fixed
- **Template variable substitution bug** - Fixed variables containing special characters (like `&middot;` in `{{blessed_comments}}`) not being substituted correctly
  - Root cause: Bash pattern replacement treats `&` as a special character
  - Added `escape_for_replacement()` helper to properly escape `&` and `\` in template values
- **Render command exiting early** - Fixed `polis render` terminating before completion when processing posts with blessed comments
  - Root cause: Functions returning non-zero exit codes triggered `set -e` behavior
  - Changed post-increment `((var++))` to pre-increment `((++var))` throughout render code
- **Index generation failure** - Fixed `index.html` not being created due to early exit issues described above

### Changed
- Enhanced `polis render` diagnostics
  - Added working directory output for troubleshooting
  - Added error check for empty index templates
  - Improved progress messages
- Updated USAGE.md with comprehensive `polis render` documentation
  - Added pandoc to prerequisites section
  - Added template variables reference table
  - Added "Embedded Source" documentation
  - Updated directory structure examples

## [0.17.0] - 2026-01-08

### Added
- **Templating system** (`polis render`) - Render markdown posts and comments to HTML
  - Generates `.html` files alongside `.md` files for blog-like interfaces
  - Renders blessed comments inline on post pages
  - Generates `index.html` listing all posts and comments
  - Supports custom templates in `.polis/templates/`
  - Incremental rendering (skips unchanged files based on timestamps)
  - `--force` flag to re-render all files regardless of timestamps
  - `--init-templates` flag to export default templates for customization
  - Template variables: `{{title}}`, `{{content}}`, `{{published}}`, `{{blessed_comments}}`, etc.
  - Requires pandoc for markdown to HTML conversion

### Changed
- Updated dependencies documentation to include pandoc (optional, for render command)

## [0.16.0] - 2026-01-08

### Changed
- **Rebuild command refactor**: `polis rebuild` now requires explicit target flags for better control
  - `--content` - Rebuild public.jsonl from posts and comments
  - `--comments` - Full rebuild of blessed-comments.json from discovery service
  - `--all` - Rebuild all indexes
  - Flags are combinable (e.g., `--content --comments`)
  - Replaced `--diff` and `--full` flags (removed incremental sync feature)
  - JSON output now includes counts: `posts_indexed`, `comments_indexed`, `blessed_comments`
  - Updated help text and skills documentation

### Removed
- **Incremental blessed comments sync**: Removed `--diff` flag from `polis rebuild`
  - Incremental sync proved unreliable for maintaining consistency
  - Full rebuild (`--comments`) is now the only supported method

## [0.15.0] - 2026-01-08

### Added
- **Key rotation command** (`polis rotate-key`) - Generate new keypair and re-sign all content
  - Addresses critical security gap: previously no way to recover from key compromise without domain migration
  - Generates new Ed25519 keypair and re-signs all posts and comments
  - Updates `.well-known/polis` with new public key
  - Archives old key to `.polis/keys/id_ed25519.old` (or deletes with `--delete-old-key` flag)
  - Provides recovery mechanism if rotation is interrupted

### Security
- **Key rotation support** - Users can now rotate compromised keys without migrating domains
  - Immediate mitigation for suspected key exposure
  - Enables routine security hygiene practices
  - See SECURITY-MODEL.md for rotation procedures and best practices

## [0.14.0] - 2026-01-08

### Added
- **Automatic .env creation**: `polis init` now automatically copies `.env.example` to `.env` if it doesn't exist
  - Displays helpful warning reminding users to update configuration values
  - JSON mode includes `env_created` boolean flag in response data

### Changed
- **Environment variable naming**: Updated `.env.example` to use `DISCOVERY_SERVICE_KEY` instead of `SUPABASE_ANON_KEY` for consistency
- **Test framework refactor**: Tests now run in isolated temporary directories instead of tracked test-data/ directory for better isolation
- **Test fixtures**: Enhanced to support `POSTS_DIR` and `COMMENTS_DIR` environment variables

### Fixed
- **Blessing workflow tests**: Updated to use content hash-based blessing operations instead of database IDs

## [0.13.0] - 2026-01-08

### Breaking Changes
- **Blessing status**: Renamed `rejected` status to `denied` for consistency with CLI terminology
  - Database migration required: `ALTER TYPE public.blessing_status_enum RENAME VALUE 'rejected' TO 'denied';`
- **Blessing operations**: Now use content hash instead of database ID
  - `polis blessing grant <hash>` (was `<id>`)
  - `polis blessing deny <hash>` (was `<id>`)
  - Hash can be short form (e.g., `f4bac5-350fd2`) or full `sha256:...` prefix
  - CLI displays short hash in blessing requests table
  - Signed payload field changed from `comment_id` to `comment_version`

### Fixed
- **Nested comments**: Added missing `warn` and `warn_human` helper functions that caused silent failures when commenting on comments
- **Follow message**: Corrected misleading message "manually blessed" → "automatically blessed" when following an author with no pending comments
- **Comment URL argument**: `polis comment <file> <url>` now correctly uses the URL argument instead of always prompting interactively
- **Blessing preview**: Fixed "Redirecting" appearing in preview by following HTTP redirects when fetching comment content

## [0.12.0] - 2026-01-08

### Fixed
- **Sync bug**: `blessing_status` query parameter was ignored by discovery service, causing pending comments to be incorrectly added to `blessed-comments.json` during sync
- **URL validation**: `polis blessing grant` now validates comment URL exists before blessing (blocks on 404)
- `polis blessing deny` now warns if comment URL returns 404

### Changed
- `polis blessing grant` and `polis blessing deny` now fetch request details in all modes (not just human mode)

## [0.11.0] - 2026-01-07

### Added
- `polis rebuild --diff` - Sync missing blessed comments from discovery service (incremental)
- `polis rebuild --full` - Rebuild blessed-comments.json entirely from discovery service

### Changed
- Removed automatic `git add` staging from all commands (users manage git manually)

## [0.10.0] - 2026-01-07

### Removed
- `polis reset` command - Removed to simplify the CLI. See USAGE.md "Starting Fresh" section for manual reset instructions if needed.

## [0.9.0] - 2026-01-07

### Added
- **Domain migration tracking** - Migrations are now recorded in the discovery service for discoverability
  - Commenters can discover when authors they interact with have migrated
  - Enables updating local references when followed authors change domains
- **Notifications command** (`polis notifications`) - View pending actions requiring attention
  - Pending blessing requests for your posts
  - Domain migrations for authors you follow or interact with
- **Migrations apply command** (`polis migrations apply`) - Update local references to migrated domains
  - Interactive confirmation before modifying files
  - **Key continuity verification** - Ensures new domain is controlled by same owner
  - Updates following.json, blessed-comments.json, and comment frontmatter
  - Warns and skips if public key mismatch detected (hijacking protection)

### Changed
- **BREAKING**: Replaced `/comments-migrate` endpoint with RESTful `/migrations` endpoint
  - POST /migrations: Record a migration
  - GET /migrations: Query migration history for specified domains
- Updated Claude Code skill with notifications workflow

## [0.8.0] - 2026-01-07

### Security
- **BREAKING**: Grant/deny blessing endpoints now require Ed25519 signature verification
  - Previous implementation used self-reported email for authorization (spoofable)
  - Now uses same cryptographic verification pattern as beseech endpoint
  - CLI signs `{action, comment_id, timestamp}` payload with author's private key
  - Server verifies signature using public key from post author's `.well-known/polis`

### Changed
- `polis blessing grant` now signs requests with Ed25519
- `polis blessing deny` now signs requests with Ed25519

## [0.7.0] - 2026-01-07

### Added
- **Domain migration command** (`polis migrate <new-domain>`) - Migrate all content to a new domain
  - Auto-detects current domain from published files
  - Updates all local files (posts, comments, metadata)
  - Re-signs all content with new URLs (required for comments where URL is in signed payload)
  - Updates discovery service database (preserves blessing status)
  - New edge function `comments-migrate` for authenticated database updates
  - New SQL function `migrate_domain()` for bulk URL updates

### Changed
- **BREAKING**: Renamed `SUPABASE_ANON_KEY` environment variable to `DISCOVERY_SERVICE_KEY`

## [0.6.0] - 2026-01-07

### Fixed
- Documentation: Corrected `polis comment` argument order (`<file> <url>`, not `<url> [file]`)
- Documentation: Fixed directory format references (`YYYYMMDD`, not `YYYY/MM`)

## [0.5.0] - 2026-01-06

### Added
- **Claude Code skill** - AI-powered workflows for publishing, discovering, commenting, and managing blessings
- `cli/skills/polis/` - Skill definition with command reference and JSON response schemas
- `CLAUDE.md` - Project context file for Claude Code integration
- Skill installation instructions in CLI README

## [0.4.0] - 2026-01-06

### Fixed
- **macOS compatibility** - `extract_domain_from_url()` now uses portable bash parameter expansion instead of GNU sed-specific `\?` regex that fails on BSD sed
- **Signature verification for nested comments** - Discovery service now includes `root_post` in signed payload verification to match CLI behavior

## [0.3.0] - 2026-01-06

### Added
- **Nested comment threads** - Comments can now reply to other comments, not just posts
- **Thread-specific auto-blessing** - Authors who have been blessed once on a post are auto-blessed for future comments on that same post
- `root-post` frontmatter field - Tracks the original post for nested comments
- `is_comment_url()` helper for detecting comment vs post URLs
- `fetch_root_post_for_comment()` helper for querying root post from discovery service
- `isAuthorBlessedOnThread()` server-side function for thread trust queries
- Database migration for `root_post` column (`migrations/002_nested_comments.sql`)

### Changed
- **BREAKING**: `in_reply_to` now means "immediate parent" (can be post OR comment URL)
- **BREAKING**: New required field `root_post` in beseech payload (always the original post URL)
- **BREAKING**: Old CLI versions will fail validation (missing `root_post` field)
- Blessing requests query now uses `root_post` domain (not `in_reply_to`) to properly support nested threads
- Updated documentation for nested threads and auto-blessing

### Migration
See `discovery-service/migrations/002_nested_comments.sql` for database migration.
Existing comments are automatically backfilled with `root_post = in_reply_to`.

## [0.2.0] - 2026-01-05

### Added
- `polis init` - Initialize a new Polis site with Ed25519 keypair
- `polis publish` - Publish markdown posts with cryptographic signatures
- `polis comment` - Create signed comments on others' posts
- `polis republish` - Update existing posts with version history
- `polis preview` - Preview and verify remote content before blessing
- `polis blessing requests` - View pending blessing requests
- `polis blessing grant` - Approve a comment for amplification
- `polis blessing deny` - Reject a blessing request
- `polis follow` - Follow an author and auto-bless their comments
- `polis unfollow` - Stop following an author
- `polis index` - Generate public.jsonl index of all content
- `polis rebuild` - Rebuild metadata files from source
- `polis version` - Display CLI version
- Interactive tutorial (`polis-tutorial`)
- JSON mode (`--json`) for all commands
- SHA256 checksum verification for script integrity
- Comprehensive documentation (USAGE.md)

### Security
- Ed25519 signatures for all published content
- SHA256 content hashes for integrity verification
- SSH-based signature verification for remote content

## [0.1.0] - 2026-01-01

### Added
- Initial proof of concept
- Basic publish and comment functionality
