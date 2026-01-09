# Polis CLI Command Reference

Quick reference for polis CLI commands with `--json` mode.

## Publishing Commands

### `polis publish <file>`
Sign and publish a new post or comment.

```bash
./cli/bin/polis --json publish posts/my-draft.md
```

### `polis publish -` (stdin)
Publish content piped from stdin.

```bash
echo "# My Post" | ./cli/bin/polis --json publish - --filename my-post.md --title "My Post"
```

Options:
- `--filename <name>` - Output filename (default: stdin-TIMESTAMP.md)
- `--title <title>` - Override title extraction

### `polis republish <file>`
Update an already-published file (creates new version).

```bash
./cli/bin/polis --json republish posts/20260106/my-post.md
```

## Comment Commands

### `polis comment <file> [url]`
Create a comment in reply to a post or another comment.

```bash
# From file
./cli/bin/polis --json comment my-reply.md https://alice.com/posts/hello.md

# From stdin
echo "Great post!" | ./cli/bin/polis --json comment - https://alice.com/posts/hello.md --filename reply.md
```

## Preview Command

### `polis preview <url>`
Preview content at a URL with signature verification.

```bash
./cli/bin/polis --json preview https://alice.com/posts/hello.md
```

## Blessing Commands

### `polis blessing sync`
Sync auto-blessed comments from discovery service.

```bash
./cli/bin/polis --json blessing sync
```

### `polis blessing requests`
List pending blessing requests for your posts.

```bash
./cli/bin/polis --json blessing requests
```

### `polis blessing grant <hash>`
Approve a pending blessing request. Hash can be short form (e.g., `f4bac5-350fd2`) or full hash.

```bash
./cli/bin/polis --json blessing grant f4bac5-350fd2
```

### `polis blessing deny <hash>`
Deny a pending blessing request. Hash can be short form or full hash.

```bash
./cli/bin/polis --json blessing deny f4bac5-350fd2
```

### `polis blessing beseech <id>`
Re-request blessing for a comment (retry). Looks up request by database ID.

```bash
./cli/bin/polis --json blessing beseech 42
```

## Social Commands

### `polis follow <author-url>`
Follow an author (auto-bless their future comments).

```bash
./cli/bin/polis --json follow https://alice.com
```

### `polis unfollow <author-url>`
Stop following an author and hide their comments.

```bash
./cli/bin/polis --json unfollow https://alice.com
```

## Index Commands

### `polis index`
View the content index.

```bash
./cli/bin/polis --json index
```

### `polis rebuild`
Rebuild local indexes. Requires at least one target flag.

```bash
# Rebuild content index (public.jsonl)
./cli/bin/polis --json rebuild --content

# Full rebuild of blessed comments from discovery service
./cli/bin/polis --json rebuild --comments

# Rebuild all indexes
./cli/bin/polis --json rebuild --all

# Flags are combinable
./cli/bin/polis --json rebuild --content --comments
```

Options:
- `--content` - Rebuild public.jsonl from posts and comments
- `--comments` - Full rebuild of blessed-comments.json from discovery service
- `--all` - Rebuild all indexes (equivalent to `--content --comments`)

## Render Commands

### `polis render`
Render markdown posts and comments to HTML. Generates `.html` files alongside `.md` files.

```bash
# Render all posts and comments, generate index.html
./cli/bin/polis --json render

# Force re-render all files (ignore timestamps)
./cli/bin/polis --json render --force

# Export default templates for customization
./cli/bin/polis render --init-templates
```

Options:
- `--force` - Re-render all files regardless of timestamps
- `--init-templates` - Create `.polis/templates/` with default templates

Requires: `pandoc` (for markdown to HTML conversion)

**Template Variables:**

Available in all templates:
- `{{site_url}}` - Base URL from config
- `{{site_title}}` - From .well-known/polis
- `{{year}}` - Current year (for copyright)

Post/Comment templates:
- `{{title}}` - Post/comment title
- `{{content}}` - HTML-rendered markdown body
- `{{published}}` - Publication date (ISO format)
- `{{published_human}}` - Human-readable date
- `{{url}}` - Canonical URL
- `{{version}}` - Current version hash
- `{{author_name}}` - From .well-known/polis
- `{{author_url}}` - Site base URL
- `{{signature_short}}` - Truncated signature for display

Comment-specific:
- `{{in_reply_to_url}}` - Parent post/comment URL

Post-specific:
- `{{blessed_comments}}` - HTML-rendered list of blessed comments
- `{{blessed_count}}` - Number of blessed comments

Index template:
- `{{posts_list}}` - Generated HTML list of posts
- `{{comments_list}}` - Generated HTML list of comments
- `{{post_count}}` - Number of posts
- `{{comment_count}}` - Number of comments

**Custom Templates:**

Run `polis render --init-templates` to create `.polis/templates/` with:
- `post.html` - Single post template
- `comment.html` - Single comment template
- `comment-inline.html` - Blessed comment rendering
- `index.html` - Listing page template

Modify these files to customize the HTML output.

## Utility Commands

### `polis init`
Initialize a new polis directory.

```bash
./cli/bin/polis --json init
```

### `polis version`
Print CLI version.

```bash
./cli/bin/polis version
```

### `polis rotate-key`
Generate new keypair and re-sign all content.

```bash
./cli/bin/polis --json rotate-key
```

Use when: key compromise, routine security hygiene, or before transferring device access.

Options:
- `--delete-old-key` - Delete old keypair instead of archiving it

Old keypair is archived at `.polis/keys/id_ed25519.old` unless `--delete-old-key` is specified.

### `polis migrate <new-domain>`
Migrate all content to a new domain (re-signs files, updates database).

```bash
./cli/bin/polis --json migrate newdomain.com
```

Auto-detects current domain from published files. Updates all URLs, re-signs content, and updates discovery service database (preserves blessing status).

### `polis notifications`
Show pending actions: blessing requests, domain migrations.

```bash
./cli/bin/polis --json notifications
```

Returns pending blessings for your posts and domain migrations for authors you interact with.

### `polis migrations apply`
Interactively apply discovered domain migrations to local files.

```bash
./cli/bin/polis migrations apply
```

For each migration:
- Verifies public key continuity (same owner controls new domain)
- Shows affected local files
- Prompts for confirmation
- Updates following.json, blessed-comments.json, and comment frontmatter

### `polis get-version <file> <hash>`
Reconstruct a specific version from history.

```bash
./cli/bin/polis get-version posts/20260106/my-post.md sha256:abc123...
```
