# Upgrading Polis (polis-upgrade)

`polis-upgrade` is a standalone script that handles version migrations and binary updates. It fetches migration scripts from GitHub (tag-pinned) and verifies SHA-256 checksums before execution.

## Usage

```bash
polis-upgrade [OPTIONS]
```

## Options

| Option | Description |
|--------|-------------|
| `--component cli\|tui` | Component to upgrade (default: `cli`) |
| `--from VERSION` | Override current version detection |
| `--to VERSION` | Upgrade to specific version (default: latest) |
| `--polis-path PATH` | Path to polis CLI script |
| `--site-dir PATH` | Path to polis site directory |
| `--yes` | Skip confirmation prompts |
| `--check` | Only check for updates, don't apply |
| `--help` | Show help message |

## Examples

```bash
# Check what upgrades are available
polis-upgrade --check

# Upgrade CLI to latest
polis-upgrade

# Upgrade to a specific version
polis-upgrade --to 0.39.0

# Upgrade TUI
polis-upgrade --component tui

# Skip prompts (for scripting)
polis-upgrade --yes

# Override version detection
polis-upgrade --from 0.33.0
```

## How It Works

1. **Self-update check** - Verifies `polis-upgrade` itself is current
2. **Detects current version** - Reads `VERSION` from your installed `polis` script
3. **Queries discovery service** - Finds the latest available version
4. **Fetches migrations** - Downloads applicable migration scripts from GitHub
5. **Verifies checksums** - SHA-256 verification before executing any script
6. **Runs migrations** - Executes scripts in version order against your site directory
7. **Downloads binary** - Offers to install the updated CLI/TUI script

## Site Directory Detection

`polis-upgrade` locates your site directory in this order:
1. `--site-dir` flag
2. `POLIS_BASE` environment variable
3. `.env` file in current directory
4. `.well-known/polis` in current directory

## Migration Scripts

Migrations handle file/data structure changes between versions (e.g., moving config files to new locations). Not every version requires a migration - only versions that change the on-disk layout.

Migration scripts are idempotent (safe to run multiple times) and non-destructive (never delete user content).

## Recovery

If a migration fails mid-sequence, the script prints a recovery command:
```
polis-upgrade --from <next_version> --to <target_version>
```
