# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-01-06

### Added
- `polis blessing sync` - Pull auto-blessed comments from discovery service
- Automatic blessing sync integrated into `polis blessing requests`
- `sync_blessed_comments()` helper function for retrieving server-blessed comments
- `is_comment_blessed()` helper function to check blessing status

### Changed
- Enhanced `polis beseech` to handle server auto-blessed responses
- Updated `polis unfollow` to query server for unsynced auto-blessed comments
- Improved blessing workflow to automatically track server-side auto-blessings

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
