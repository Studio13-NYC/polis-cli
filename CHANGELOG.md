# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
