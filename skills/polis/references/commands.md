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

### `polis comment <url> [file]`
Create a comment in reply to a post or another comment.

```bash
# From file
./cli/bin/polis --json comment https://alice.com/posts/hello.md my-reply.md

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

### `polis blessing grant <id>`
Approve a pending blessing request.

```bash
./cli/bin/polis --json blessing grant 42
```

### `polis blessing deny <id>`
Reject a pending blessing request.

```bash
./cli/bin/polis --json blessing deny 42
```

### `polis blessing beseech <id>`
Re-request blessing for a comment (retry).

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
Rebuild the index from published files.

```bash
./cli/bin/polis --json rebuild
```

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

### `polis reset`
Archive current state and start fresh.

```bash
./cli/bin/polis --json reset
```

### `polis get-version <file> <hash>`
Reconstruct a specific version from history.

```bash
./cli/bin/polis get-version posts/20260106/my-post.md sha256:abc123...
```
