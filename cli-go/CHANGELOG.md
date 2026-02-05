# Changelog

All notable changes to the Go CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
