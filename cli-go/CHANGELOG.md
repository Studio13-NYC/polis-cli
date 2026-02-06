# Changelog

All notable changes to the Go CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.47.0]

### Added

- **Shell completions for `serve` and `validate`**: Both bash and zsh completions now include `serve` (with `--data-dir`/`-d` flags) and `validate` (with `--json` flag)
- **`polis serve --help`**: Serve command now prints flag documentation when invoked with `--help`/`-h`
- **`{{target_author}}` and `{{preview}}` in comment loops**: Template engine now wires `target_author` and `preview` variables through `{{#comments}}` sections and partial includes
- **`polis init` creates `webapp-config.json`**: Init now creates `.polis/webapp-config.json` with webapp-specific defaults (`setup_at`, `view_mode`, `show_frontmatter`). Discovery credentials are not included — they belong in `.env` and are loaded at runtime.
- **Hooks auto-discovery**: `RunHook()` and `GetHookPath()` now check `.polis/hooks/{event}.sh` when no explicit path is configured. Placing a script in the conventional location just works without registering it in `webapp-config.json`.

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
