# Changelog

All notable changes to the Go CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.48.0]

### Added

### Changed

### Fixed

### Security

- **[H3] `rotate-key` now updates `.well-known/polis`**: Previously, `rotate-key` read `.well-known/polis` but never wrote back the new public key, leaving the site broken after rotation. Now the command parses the JSON, replaces `public_key`, and writes it back while preserving all other fields.
- **[M6] Temp files use private directory**: `computeUnifiedDiff()` now creates a private temp directory (`0700`) instead of using the system `/tmp` directly. Files created during diff computation are no longer world-readable.
- **[Webapp] [H1] Error detail redaction**: All HTTP error responses that previously included `err.Error()` (leaking file paths, OS error strings) now return generic messages. Internal error details are logged server-side via `s.LogError()`.
- **[Webapp] [M1] Draft ID whitelist sanitization**: Draft IDs are now sanitized with a whitelist regex (`[^a-zA-Z0-9_-]` replaced with `-`) instead of the previous blacklist approach that only stripped `/` and `\`.
- **[Webapp] [M2] Path traversal canonicalization**: `validatePostPath()` and `validateContentPath()` now apply `filepath.Clean()` before checking for `..`, preventing encoded traversal sequences from bypassing the check.

### Tests

- **[Webapp]** Added `TestErrorResponsesRedacted` to verify error responses don't contain OS error strings or file paths.
- **[Webapp]** Added `TestDraftIDSanitization` to verify special characters, path traversal, null bytes, and unicode are stripped from draft IDs.
- **[Webapp]** Added `TestValidatePostPath_Canonicalization` and `TestValidateContentPath_Canonicalization` to verify `filepath.Clean` inputs are handled correctly.

## [0.47.0]

### Added

- **Shell completions for `serve` and `validate`**: Both bash and zsh completions now include `serve` (with `--data-dir`/`-d` flags) and `validate` (with `--json` flag)
- **`polis serve --help`**: Serve command now prints flag documentation when invoked with `--help`/`-h`
- **`{{target_author}}` and `{{preview}}` in comment loops**: Template engine now wires `target_author` and `preview` variables through `{{#comments}}` sections and partial includes
- **`polis init` creates `webapp-config.json`**: Init now creates `.polis/webapp-config.json` with webapp-specific defaults (`setup_at`, `view_mode`, `show_frontmatter`). Discovery credentials are not included — they belong in `.env` and are loaded at runtime.
- **Hooks auto-discovery**: `RunHook()` and `GetHookPath()` now check `.polis/hooks/{event}.sh` when no explicit path is configured. Placing a script in the conventional location just works without registering it in `webapp-config.json`.
- **[Webapp] CLI version propagation**: Server accepts `CLIVersion` via `RunOptions` and propagates it to `publish`, `comment`, and `metadata` packages so all generated metadata uses the correct CLI version

### Fixed

- **`polis post` now moves the original file**: `polis post <file>` removes the original file after publishing into `posts/`, matching the bash CLI's move behavior. Non-fatal on failure (warns only). Does not apply to `republish`.

- **Comment count mismatch after blessing**: `MoveComment()` now calls `publish.UpdateManifest()` after blessing a comment, so `manifest.json` `comment_count` stays in sync with actual blessed comments
- **Hardcoded version strings in metadata files**: `.well-known/polis`, `manifest.json`, `following.json`, and `blessed-comments.json` now use the CLI version from `version.txt` instead of hardcoded `"0.1.0"`, `"1.0"`, or `"0.42.0"`
- **Hardcoded generator tags in frontmatter**: Post and comment generator fields now use `"polis-cli-go/<version>"` computed from the CLI version, replacing hardcoded `"polis-cli-go/0.1.0"` and `"polis-webapp/0.1.0"`

### Previously added

- **Filename collision prevention**: Posts and comments auto-append `-2`, `-3`, etc. when a filename already exists across posts, drafts, and all comment status directories
- **Random slug for untitled posts**: Publishing without a title now generates `untitled-<random hex>` instead of bare `untitled`, preventing silent overwrites
- **Blessed comments in rendered posts**: Renderer loads local comment content (strips frontmatter, renders markdown to HTML) for blessed comments instead of leaving content empty
- **Rebuild fetches blessed comments**: `rebuild --comments` now queries the discovery service for blessed comments and populates `blessed-comments.json` (falls back to empty file when discovery is not configured)
- **Webhook safety regression tests**: Tests verify hooks only fire after successful operations, never on error paths
- **`polis init` flag parity**: Added 10 missing flags (`--site-title`, `--register`, `--keys-dir`, `--posts-dir`, `--comments-dir`, `--snippets-dir`, `--themes-dir`, `--versions-dir`, `--public-index`, `--blessed-comments`, `--following-index`); renamed `--title` to `--site-title`; removed Go-only flags `--author`, `--email`, `--base-url` (author/email sourced from git config)

### Fixed

- **Random theme selection when no active theme set**: Replaced hardcoded `"turbo"` fallback with random selection from available themes, matching the bash CLI's `select_theme()` behavior; also fixes the empty `active_theme` bug where `GetActiveTheme()` returned `("", nil)` causing `theme.Load()` to fail with "theme name is required"

### Changed

- **public.jsonl deduplication**: `AppendToPublicIndex()` now checks for existing entries by path and updates in place instead of always appending; `publish.AppendToIndex()` delegates to `metadata.AppendPostToIndex()` for unified dedup
- **MoveComment populates blessed-comments.json**: Moving a comment to blessed status now adds it to both `public.jsonl` and `blessed-comments.json`
- **Flexible blessed comment path matching**: `GetBlessedCommentsForPost()` matches across `.md`/`.html` extensions and full URL vs relative path variants
- **Renderer skips .versions directories**: `RenderAll()` Walk callbacks now skip `.versions` directories, matching `index/rebuild.go` behavior
- **Drafts directory renamed**: `.polis/drafts` → `.polis/posts/drafts`; old path still accepted in content validation for backwards compatibility
- **[Webapp] Hooks fire without explicit config**: Publish, republish, and beseech handlers no longer guard hook execution behind `s.Config.Hooks != nil`. `RunHook()` now handles nil config gracefully with auto-discovery from `.polis/hooks/`.
- **[Webapp] Automations list shows auto-discovered hooks**: `getAutomations()` uses `GetHookPathWithDiscovery()` so the settings UI displays hooks found at conventional paths, not just those registered in `webapp-config.json`.
- **[Webapp] Native confirm() replaced**: All 5 browser `confirm()` calls replaced with styled `showConfirmModal()` dialogs with appropriate danger/default types
- **[Webapp] Subdomain removed from webapp-config.json**: `SaveConfig()` strips the deprecated `Subdomain` field; `LoadEnv()` no longer derives subdomain from `POLIS_BASE_URL`; all runtime usage goes through `GetSubdomain()` which derives from `BaseURL`
- **[Webapp] Beseech auto-bless renders site**: The auto-blessed branch of the beseech handler now calls `RenderSite()` before running hooks, ensuring HTML is generated before deployment
- **[Webapp] Drafts directory migration on startup**: Automatic migration from `.polis/drafts` to `.polis/posts/drafts` on webapp startup
- **[Webapp] Init handler compatibility**: Removed deleted `Author`/`Email`/`BaseURL` fields from `InitOptions` construction to match updated `cli-go/pkg/site` API

## [0.46.0] - 2026-02-05

### Deprecated

- **polis-tui**: Terminal UI deprecated in favor of webapp (`polis-full serve`)

## [0.45.0] - 2026-02-05

### Summary

The Go CLI reaches implementation parity with the Bash CLI. This version is
**untested in production** but implements all 27 commands with matching output
formats and error codes. The Go CLI will be the active implementation going
forward.

### Added

- Full command parity with Bash CLI (27 commands)
- Packages: remote, verify, version, following, notification, index, migrate, clone
- Commands: post, republish, comment, preview, extract, index, about
- Commands: follow, unfollow, discover, clone
- Commands: blessing (requests, grant, deny, beseech, sync)
- Commands: notifications (list, read, dismiss, sync, config)
- Commands: rebuild, migrate, migrations apply, rotate-key
- Commands: init, render, register, unregister, version, serve (stub)
- JSON output mode (`--json` flag)
- Data directory override (`--data-dir` flag)

### Notes

- The `serve` command is a stub in the CLI-only binary; requires the bundled
  binary (`polis-full`) for actual web server functionality
- This release has not been tested against production discovery services
- Report issues at: https://github.com/vdibart/polis-cli/issues
