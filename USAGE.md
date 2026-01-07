# Polis CLI - Complete Usage Guide

> This is the comprehensive command reference. For a quick introduction, see [README.md](README.md).

Command-line tool for managing decentralized social content with cryptographic signing and version control.

## Overview

The Polis CLI enables authors to:
- Create and sign posts and comments using Ed25519 cryptography
- Manage content with git-based version history
- Publish content as static files with frontmatter metadata
- Request blessings from discovery services for authenticated comment discovery
- Automate workflows with JSON mode for scripting and CI/CD integration

## Installation

### Prerequisites

- **OpenSSH 8.0+** (for Ed25519 signing with `-Y` flag)
- **jq** (JSON processor for index management and JSON mode)
- **curl** (for API communication with discovery service)
- **sha256sum** or **shasum** (for content hashing)
- **git** (optional, for version control integration)

### Install Dependencies

```bash
# Linux (Debian/Ubuntu)
sudo apt-get install openssh-client jq curl coreutils git

# macOS
brew install openssh jq curl coreutils git
```

### Install Polis CLI

```bash
# Option 1: Add to PATH
export PATH="/path/to/polis-planning/cli/bin:$PATH"

# Option 2: Create symlink
sudo ln -s /path/to/polis-planning/cli/bin/polis /usr/local/bin/polis

# Verify installation
polis --help
```

## Quick Start

This example demonstrates a complete workflow: creating a post, commenting on it, and managing blessing requests.

```bash
# 1. Initialize a new Polis directory
mkdir my-blog && cd my-blog
polis init

# 2. Set up environment variables (required for blessing workflow)
export POLIS_BASE_URL="https://yourdomain.com"
export POLIS_DISCOVERY_ENDPOINT="https://xxx.supabase.co/functions/v1"

# 3. Create a post
cat > posts/hello.md << 'EOF'
# Hello World

This is my first post on Polis!
EOF

# 4. Publish the post
polis publish posts/hello.md

# 5. Create a comment on the post
cat > comments/reply.md << 'EOF'
# Great post!

I really enjoyed reading this. Looking forward to more!
EOF

# 6. Comment on your own post (to test the blessing workflow)
polis comment comments/reply.md ${POLIS_BASE_URL}/posts/$(date +%Y%m%d)/hello.md

# 7. View pending blessing requests
polis blessing requests

# 8. Grant the blessing (using the comment ID from step 7)
polis blessing grant <comment-id>

# Alternatively, deny a blessing:
# polis blessing deny <comment-id>

# 9. Set up git for your content
git init
git add .
git commit -m "Initial post and comment"

# 10. Push to your static host (GitHub Pages, Netlify, etc.)
git push origin main
```

**What happens in this workflow:**
- Step 4: Post is published and added to `public.jsonl` index
- Step 6: Comment is created, automatically requests blessing from discovery service
- Step 7: Lists all pending comments awaiting your approval
- Step 8: Approving the comment makes it visible in the discovery service
- Once blessed, readers can discover your comment when querying the post

**Note:** In practice, step 6 would typically be someone else commenting on your post from their own Polis instance. This example shows you commenting on your own post for testing purposes.

## Directory Structure

After running `polis init`, your directory will contain:

```
.
├── .polis/
│   └── keys/
│       ├── id_ed25519       # Private signing key (keep secret!)
│       └── id_ed25519.pub   # Public verification key
├── metadata/                 # Metadata files
│   ├── public.jsonl         # Content index (JSONL format)
│   ├── blessed-comments.json # Index of approved comments
│   └── following.json       # Following list
├── posts/                    # Your posts
│   └── 20260106/            # Date-stamped directory (YYYYMMDD)
│       ├── .versions/       # Version history for posts in this directory
│       │   └── my-post.md
│       └── my-post.md
├── comments/                 # Your comments
│   └── 20260106/            # Date-stamped directory (YYYYMMDD)
│       ├── .versions/       # Version history for comments in this directory
│       │   └── reply.md
│       └── reply.md
└── .well-known/
    └── polis                # Public metadata (author, public key)
```

## Commands

### `polis init`

Initialize a new Polis directory with keys and metadata.

```bash
polis init
```

**Creates:**
- `.polis/keys/` - Ed25519 keypair for signing
- `posts/`, `comments/` - Content directories
- `metadata/` - Metadata directory
- `.well-known/polis` - Public metadata file
- `metadata/public.jsonl` - Content index (JSONL format)
- `metadata/blessed-comments.json` - Blessed comments index
- `metadata/following.json` - Following list

### `polis publish <file>`

Sign and publish a post or comment with frontmatter metadata.

```bash
polis publish posts/my-post.md
polis publish comments/my-comment.md
```

**What it does:**
1. Generates SHA-256 hash of content
2. Signs content with Ed25519 private key
3. Adds frontmatter with metadata (version, author, signature)
4. Appends entry to `public.jsonl` index
5. Creates `.versions` file for version history

**Example output:**
```
[i] Content hash: sha256:a3b5c7d9...
[i] Signing content with Ed25519 key...
[✓] Created canonical file: posts/20260106/my-post.md
```

### `polis republish <file>`

Republish an existing file with updated content (creates new version).

```bash
# Edit your published file
vim posts/20260106/my-post.md

# Republish with new version
polis republish posts/20260106/my-post.md
```

**What it does:**
1. Generates diff between old and new content
2. Appends diff to `.versions` file
3. Updates frontmatter with new version hash
4. Re-signs content with new signature
5. Rebuilds `public.jsonl` index (prevents duplicates)

**Version history:**
- Stored in `.versions/` subdirectory alongside content files
- Example: `posts/20260106/my-post.md` → `posts/20260106/.versions/my-post.md`
- Directory name configurable via `POLIS_VERSIONS_DIR_NAME` (default: `.versions`)
- Uses unified diff format (compatible with `diff`/`patch` tools)
- Enables version reconstruction

### Publishing from stdin

You can pipe content directly to `polis publish` and `polis comment` without creating temporary files:

```bash
# Basic usage
echo "# My Post" | polis publish -

# Specify filename
echo "# My Post" | polis publish - --filename my-post.md

# Specify title
echo "Content without heading" | polis publish - --title "My Title"

# Both options
echo "Content" | polis publish - --filename post.md --title "My Title"

# From file redirect
polis publish - < draft.md

# From curl
curl -s https://example.com/draft.md | polis publish -

# From heredoc
polis publish - << 'EOF'
# My Post
Content here
EOF

# Piping with preprocessing
grep -v "^Draft:" draft.md | polis publish - --filename final.md
```

**Options:**
- `--filename <name>` - Specify output filename (default: `stdin-TIMESTAMP.md`)
- `--title <title>` - Override title extraction

**For comments:**
```bash
# Comment from stdin
echo "# My reply" | polis comment - https://bob.com/posts/original.md

# With filename
echo "# Reply" | polis comment - https://bob.com/post.md --filename my-reply.md

# With title override
echo "Content" | polis comment - https://bob.com/post.md --title "My Reply"
```

**JSON mode:**
```bash
echo "# Test" | polis --json publish - --filename test.md | jq
```

### `polis comment <url> [file]`

Create a comment in reply to a post or another comment (nested threads).

```bash
# Reply to a post
polis comment https://alice.example.com/posts/20260106/hello.md

# Reply to another comment (nested thread)
polis comment https://bob.example.com/comments/20260105/reply.md my-reply.md

# From file
polis comment https://bob.example.com/posts/intro.md my-reply.md
```

**What it does:**
1. Creates comment file in `comments/YYYYMMDD/`
2. Adds `in_reply_to` frontmatter (with `url` and `root-post` fields)
3. Signs and publishes comment
4. Appends entry to `public.jsonl` index
5. Automatically requests blessing from discovery service

**Nested threads:** When replying to a comment (instead of a post), the CLI automatically detects this and fetches the original post URL (`root-post`) from the discovery service.

**Frontmatter structure:**
```yaml
in-reply-to:
  url: https://bob.com/comments/reply.md  # Immediate parent (post or comment)
  root-post: https://alice.com/posts/intro.md  # Always the original post
```

### `polis preview <url>`

Preview content at a URL (posts or comments) with signature verification.

```bash
# Preview a post
polis preview https://alice.com/posts/20260105/hello.md

# Preview a comment before blessing it
polis preview https://bob.com/comments/20260105/reply.md

# JSON mode for scripting
polis --json preview https://alice.com/posts/hello.md
```

**What it shows:**
- Frontmatter metadata (displayed dimmed/greyed)
- Signature verification status (valid/invalid/missing)
- Content hash verification (valid/mismatch)
- For comments: the in-reply-to URL
- The content body

**Use cases:**
- Preview a comment before blessing it
- Preview content before responding to it
- Verify content integrity and authenticity

**JSON mode output:**
```json
{
  "status": "success",
  "command": "preview",
  "data": {
    "url": "https://alice.com/posts/hello.md",
    "type": "post",
    "title": "Hello World",
    "published": "2026-01-05T12:00:00Z",
    "current_version": "sha256:abc123...",
    "generator": "polis-cli/0.2.0",
    "in_reply_to": null,
    "author": "alice@example.com",
    "signature": {
      "status": "valid",
      "message": "Signature verified against author's public key"
    },
    "hash": {
      "status": "valid"
    },
    "validation_issues": [],
    "body": "# Hello World\n\nThis is my first post..."
  }
}
```

### `polis rebuild`

Rebuild the `public.jsonl` index from all published files.

```bash
polis rebuild
```

**Use when:**
- Index is corrupted or out of sync
- You manually edited published files
- You restored from backup

### `polis index [--json]`

View the content index in JSONL or JSON format (read-only, outputs to stdout).

```bash
# View as JSONL (default)
polis index

# View as JSON (grouped by type)
polis index --json

# Pipe to jq for pretty printing
polis index --json | jq

# Count posts
polis index | grep -c '"type":"post"'

# View recent 10 entries
polis index | tail -10 | jq

# Extract all post titles
polis index --json | jq -r '.posts[].title'
```

**What it does:**
- Reads `metadata/public.jsonl`
- Default: outputs raw JSONL (one entry per line)
- `--json` flag: converts to grouped JSON format for readability
- Read-only operation - never modifies the index file

**Use cases:**
- Debugging index contents
- Scripting and automation
- Inspecting published content metadata

### `polis get-version <file> <version-hash>`

Reconstruct a specific version of a file from version history.

```bash
# Get specific version
polis get-version posts/20260106/my-post.md sha256:abc123...

# Output to file
polis get-version posts/20260106/my-post.md sha256:abc123... > old-version.md
```

### `polis reset`

Remove all generated files and start fresh (keeps source content).

```bash
polis reset
```

**Removes:**
- All canonical files in dated directories (`posts/YYYYMMDD/`, `comments/YYYYMMDD/`)
- All `.versions` files
- `public.jsonl` and `blessed-comments.json`

**Preserves:**
- `.polis/keys/` (your signing keys)
- Original content files

### `polis version`

Print the CLI version number.

```bash
polis version
```

**Example output:**
```
polis 0.2.0
```

### `polis blessing`

Parent command for blessing-related operations. Must be followed by a subcommand.

#### `polis blessing requests`

List pending blessing requests for your posts.

```bash
polis blessing requests
```

**Example output:**
```
ID    Author              Post                    Status
42    bob@example.com     /posts/hello.md         pending
73    carol@example.com   /posts/hello.md         pending
```

#### `polis blessing grant <id>`

Approve a pending blessing request.

```bash
polis blessing grant 42
```

**What it does:**
1. Updates discovery service status to "blessed"
2. Adds entry to `metadata/blessed-comments.json`
3. Comment becomes visible to your audience

#### `polis blessing deny <id>`

Reject a pending blessing request.

```bash
polis blessing deny 42
```

**What it does:**
1. Updates discovery service status to "denied"
2. Comment remains on author's site but won't be amplified

#### `polis blessing beseech <id>`

Re-request blessing for a comment (retry after changes).

```bash
polis blessing beseech 42
```

**Use when:**
- Original request failed
- You updated the comment and want to re-request

#### `polis blessing sync`

Synchronize auto-blessed comments from the discovery service to your local `blessed-comments.json`.

```bash
polis blessing sync
```

**What it does:**
1. Fetches all blessed comments for your posts from discovery service
2. Compares with local `metadata/blessed-comments.json`
3. Adds any missing entries (e.g., comments auto-blessed while you were offline)

**When to use:**
- After being offline for a while
- To ensure local file matches discovery service
- Automatically called when running `polis blessing requests`

**Example output:**
```
[i] Syncing blessed comments from discovery service...
[✓] Synced 3 comment(s) to blessed-comments.json
```

### `polis follow <author-url>`

Follow an author to auto-bless their future comments on your posts.

```bash
polis follow https://alice.example.com
```

**What it does:**
1. Adds author to `metadata/following.json`
2. Auto-blesses any existing pending comments from this author
3. Future comments from this author are automatically blessed

**Example output:**
```
[OK] Following alice.example.com
[OK] 3 existing comments auto-blessed
```

### `polis unfollow <author-url>`

Stop following an author and hide all their comments.

```bash
polis unfollow https://alice.example.com
```

**What it does:**
1. Removes author from `metadata/following.json`
2. Removes all blessed comments from this author (nuclear option)

**Warning:** This is a destructive action - all previously blessed comments from this author will be hidden.

### Auto-Blessing

Comments can be automatically blessed (no manual approval required) in two scenarios:

**1. Global Trust (Following)**
When you follow an author, ALL their future comments on ANY of your posts are auto-blessed.

```bash
# Follow Alice - all her comments on your posts are now auto-blessed
polis follow https://alice.example.com
```

**2. Thread-Specific Trust**
When you manually bless a comment from an author, their future comments *on the same post* are auto-blessed. This allows trust to be scoped to specific conversations.

```
Example:
1. Bob comments on your "Intro to Polis" post
2. You bless Bob's comment (polis blessing grant 123)
3. Bob comments again on "Intro to Polis" → auto-blessed!
4. Bob comments on your "Advanced Polis" post → NOT auto-blessed (different post)
```

**Precedence:** Global trust (following) takes priority. If you follow someone, thread-specific trust is irrelevant - they're trusted everywhere.

## File Frontmatter

Published files include YAML frontmatter with metadata:

```yaml
---
canonical_url: https://alice.example.com/posts/20260106/hello.md
version: sha256:a3b5c7d9e1f2...
author: alice@example.com
published: 2026-01-15T12:00:00Z
signature: -----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQA...
-----END SSH SIGNATURE-----
in_reply_to: https://bob.example.com/posts/intro.md  # Comments only
in_reply_to_version: sha256:xyz789...                # Comments only
---

# Your Content Here

The actual post or comment content follows the frontmatter.
```

## Version History

Polis uses diff-based version storage. The `.versions` file format uses standard unified diff format, making it compatible with Unix `diff` and `patch` utilities for manual inspection or reconstruction.

**Example `.versions` file:**
```
== Version 1 ==
Version: sha256:abc123...
Date: 2026-01-15T12:00:00Z

Full content of version 1...

== Version 2 ==
Version: sha256:def456...
Date: 2026-01-20T15:30:00Z
Previous: sha256:abc123...

--- old
+++ new
@@ -5,7 +5,7 @@
-This is the old line
+This is the updated line
```

## Configuration

Polis CLI uses a layered configuration system with the following precedence (highest to lowest):

1. **Environment variables** - For CI/CD and temporary overrides
2. **`.env` file** - For developer/deployment settings
3. **`.well-known/polis`** - For user-specific directory customization
4. **Built-in defaults** - Always available as fallback

### Environment Variables

```bash
# Required for blessing commands
export POLIS_BASE_URL="https://yourdomain.com"

# Discovery service (optional - has default)
export POLIS_ENDPOINT_BASE="https://xxx.supabase.co/functions/v1"

# API authentication (required for blessing operations)
export SUPABASE_ANON_KEY="your-anon-key"

# Optional directory overrides
export KEYS_DIR=".polis/keys"
export POSTS_DIR="posts"
export COMMENTS_DIR="comments"
export VERSIONS_DIR_NAME=".versions"
```

Add to `~/.bashrc` or `~/.zshrc` for persistence.

### Using a `.env` File

Create a `.env` file in your project root:

```bash
cp .env.example .env
# Edit .env with your settings
```

Example `.env`:
```bash
POLIS_BASE_URL=https://alice.example.com
SUPABASE_ANON_KEY=eyJhbGciOiJI...
```

**Security Note:** Never commit `.env` files containing secrets. The `.env.example` file is safe to commit as a template.

### Custom Directory Paths

You can customize directory paths during initialization:

```bash
polis init --posts-dir articles --comments-dir replies
```

Or edit the `config` section in `.well-known/polis` after initialization:

```json
{
  "version": "0.2.0",
  "config": {
    "directories": {
      "keys": ".polis/keys",
      "posts": "articles",
      "comments": "replies",
      "versions": ".versions"
    },
    "files": {
      "public_index": "metadata/public.jsonl",
      "blessed_comments": "metadata/blessed-comments.json",
      "following_index": "metadata/following.json"
    }
  }
}
```

## JSON Mode

The Polis CLI supports a `--json` flag for machine-readable output, enabling scripting, automation, and testing workflows.

### Usage

Add `--json` before any command:

```bash
polis --json <command> [options]
```

### Features

When `--json` is enabled:
- **Structured output**: Valid JSON on stdout (success) or stderr (errors)
- **Auto-skip prompts**: Interactive prompts are skipped with logged defaults to stderr
- **No colors**: ANSI color codes are disabled
- **Exit codes**: 0 for success, 1 for error
- **Structured errors**: Consistent error codes (FILE_NOT_FOUND, INVALID_INPUT, API_ERROR, etc.)

### Examples

```bash
# Initialize and extract public key path
polis --json init | jq -r '.data.key_paths.public'

# Publish and get content hash
hash=$(polis --json publish my-post.md | jq -r '.data.content_hash')

# Comment with reply-to URL (no interactive prompt)
polis --json comment my-reply.md https://alice.com/posts/hello.md

# Get pending blessing requests
polis --json blessing requests | jq -r '.data.requests[].id'

# Auto-grant blessing without confirmation
polis --json blessing grant 123

# Follow author and get blessed count
polis --json follow https://alice.com | jq '.data.comments_blessed'

# Chain commands in a script
requests=$(polis --json blessing requests)
echo "$requests" | jq -r '.data.requests[].id' | while read id; do
  polis --json blessing grant "$id"
done

# Error handling
if ! result=$(polis --json publish test.md 2>&1); then
  error_code=$(echo "$result" | jq -r '.error.code')
  echo "Failed with error: $error_code"
  exit 1
fi
```

### Response Format

**Success:**
```json
{
  "status": "success",
  "command": "publish",
  "data": {
    "file_path": "posts/20260104/my-post.md",
    "content_hash": "sha256:abc123...",
    "timestamp": "2026-01-04T12:00:00Z",
    "signature": "-----BEGIN SSH SIGNATURE-----...",
    "canonical_url": "https://example.com/posts/20260104/my-post.md"
  }
}
```

**Error:**
```json
{
  "status": "error",
  "command": "publish",
  "error": {
    "code": "FILE_NOT_FOUND",
    "message": "File not found: test.md",
    "details": {}
  }
}
```

### Supported Commands

All interactive commands support JSON mode:
- `init`, `publish`, `comment`, `republish`
- `blessing requests`, `blessing grant`, `blessing deny`, `blessing beseech`
- `follow`, `unfollow`

See [docs/json-mode.md](../docs/json-mode.md) for complete documentation.

## Publishing Workflow

### 1. Write Content
```bash
vim posts/my-thoughts.md
```

### 2. Publish Locally
```bash
polis publish posts/my-thoughts.md
```

### 3. Commit to Git
```bash
git add .
git commit -m "Add: my-thoughts.md"
```

### 4. Push to Static Host
```bash
git push origin main
```

### 5. Request Blessing (if commenting)
```bash
# Note: polis comment and polis republish automatically request blessings
# You rarely need to run this manually - only for edge cases (see docs)

# If you do need to retry a blessing request:
polis blessing requests           # View pending requests
polis blessing beseech <id>       # Retry request by ID
```

## Common Use Cases

### Creating a Blog Post
```bash
cat > posts/why-decentralization-matters.md << 'EOF'
# Why Decentralization Matters

Centralized platforms have too much control...
EOF

polis publish posts/why-decentralization-matters.md
git add . && git commit -m "New post: decentralization"
git push
```

### Replying to Someone's Post
```bash
polis comment https://alice.com/posts/20260106/hot-take.md

# Interactive editor opens - write your reply

# After saving, blessing request is automatically sent
# Your comment will be pending until the post author blesses it
```

### Updating a Post
```bash
# Edit the canonical file directly
vim posts/20260106/my-post.md

# Republish with new version
polis republish posts/20260106/my-post.md

git add . && git commit -m "Update: my-post.md"
git push
```

### Viewing Version History
```bash
# List all versions in .versions file
cat posts/20260106/.versions/my-post.md

# Reconstruct specific version
polis get-version posts/20260106/my-post.md sha256:abc123...
```

## Security Notes

### Private Key Protection
- **Never commit `.polis/keys/id_ed25519`** (private key)
- Add to `.gitignore`: `.polis/keys/id_ed25519`
- Public key (`.polis/keys/id_ed25519.pub`) is safe to share

### Signature Verification
Anyone can verify your content signatures:

```bash
# Extract public key from .well-known/polis
curl https://alice.com/.well-known/polis | jq -r .public_key > alice.pub

# Verify signature (manual process - automated tool coming)
ssh-keygen -Y verify -f alice.pub -I alice@example.com -n file \
  -s signature.sig < content.txt
```

## Troubleshooting

### "ssh-keygen: command not found"
Install OpenSSH client (see Installation section above).

### "jq: command not found"
Install jq JSON processor (see Installation section above).

### "No such file: .polis/keys/id_ed25519"
Run `polis init` to create keys and directory structure.

### "Index file is corrupted or missing"
Run `polis rebuild` to regenerate `public.jsonl` from published files.

### Version history missing
`.versions` files are created on first `polis republish` - they don't exist for initial `polis publish`.

## Next Steps

- Read the [Architecture Documentation](../docs/polis-architecture.md) for technical details
- Set up a discovery service (see `../discovery-service/README.md`)
- Deploy your content to GitHub Pages, Netlify, or any static host
- Join the Polis community (links TBD at MVP)

## Support

For issues, questions, or feature requests, please file an issue in the GitHub repository.

## License

AGPL-3.0 - See [LICENSE](../LICENSE)
