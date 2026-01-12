# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
