# Polis CLI JSON Response Schemas

Reference for parsing JSON responses from polis CLI commands.

## Standard Response Format

### Success Response
```json
{
  "status": "success",
  "command": "command-name",
  "data": {
    // Command-specific fields
  }
}
```

### Error Response
```json
{
  "status": "error",
  "command": "command-name",
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

---

## Command Responses

### `polis init`
```json
{
  "status": "success",
  "command": "init",
  "data": {
    "directories_created": [".polis/keys", "posts", "comments", "metadata"],
    "files_created": [
      ".well-known/polis",
      "metadata/public.json",
      "metadata/blessed-comments.json",
      "metadata/following.json"
    ],
    "key_paths": {
      "private": ".polis/keys/id_ed25519",
      "public": ".polis/keys/id_ed25519.pub"
    }
  }
}
```

### `polis publish`
```json
{
  "status": "success",
  "command": "publish",
  "data": {
    "file_path": "posts/20260106/my-post.md",
    "content_hash": "sha256:abc123...",
    "timestamp": "2026-01-06T12:00:00Z",
    "signature": "-----BEGIN SSH SIGNATURE-----...",
    "canonical_url": "https://example.com/posts/20260106/my-post.md"
  }
}
```

### `polis comment`
```json
{
  "status": "success",
  "command": "comment",
  "data": {
    "file_path": "comments/20260106/reply.md",
    "content_hash": "sha256:def456...",
    "in_reply_to": "https://bob.com/posts/original.md",
    "timestamp": "2026-01-06T12:30:00Z",
    "beseech_status": "pending"
  }
}
```

### `polis preview`
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

### `polis blessing requests`
```json
{
  "status": "success",
  "command": "blessing-requests",
  "data": {
    "count": 3,
    "requests": [
      {
        "id": 1,
        "comment_url": "https://alice.com/comments/reply.md",
        "author": "alice@example.com",
        "timestamp": "2026-01-05T12:00:00Z"
      }
    ]
  }
}
```

### `polis blessing grant`
```json
{
  "status": "success",
  "command": "blessing-grant",
  "data": {
    "comment_id": 123,
    "comment_url": "https://alice.com/comments/reply.md",
    "blessed_at": "2026-01-06T13:00:00Z",
    "blessed_by": "bob@example.com"
  }
}
```

### `polis blessing deny`
```json
{
  "status": "success",
  "command": "blessing-deny",
  "data": {
    "comment_id": 123,
    "comment_url": "https://alice.com/comments/reply.md",
    "denied_at": "2026-01-06T13:00:00Z",
    "denied_by": "bob@example.com"
  }
}
```

### `polis follow`
```json
{
  "status": "success",
  "command": "follow",
  "data": {
    "author_url": "https://alice.com",
    "author_email": "alice@example.com",
    "comments_found": 5,
    "comments_blessed": 5,
    "added_to_following": true
  }
}
```

### `polis unfollow`
```json
{
  "status": "success",
  "command": "unfollow",
  "data": {
    "author_url": "https://alice.com",
    "removed_from_following": true,
    "comments_denied": 3
  }
}
```

### `polis index`
Returns grouped JSON with posts and comments:
```json
{
  "status": "success",
  "command": "index",
  "data": {
    "posts": [
      {
        "title": "My Post",
        "canonical_url": "https://example.com/posts/20260106/my-post.md",
        "published": "2026-01-06T12:00:00Z",
        "content_hash": "sha256:..."
      }
    ],
    "comments": [
      {
        "title": "Re: Hello",
        "canonical_url": "https://example.com/comments/20260106/reply.md",
        "in_reply_to": "https://alice.com/posts/hello.md",
        "published": "2026-01-06T12:30:00Z"
      }
    ]
  }
}
```

### `polis rebuild`
```json
{
  "status": "success",
  "command": "rebuild",
  "data": {
    "posts_indexed": 12,
    "comments_indexed": 34,
    "index_path": "metadata/public.json"
  }
}
```

---

## Error Codes

| Code | Description | Recovery Action |
|------|-------------|-----------------|
| `FILE_NOT_FOUND` | Required file doesn't exist | Check file path, help user locate correct file |
| `INVALID_INPUT` | User input validation failed | Explain required format |
| `API_ERROR` | Remote API call failed | Check network, verify discovery service status |
| `SIGNATURE_ERROR` | Signature verification failed | Report verification failure, check key |
| `MISSING_DEPENDENCY` | Required tool not found | Guide user to install (jq, ssh-keygen, etc.) |
| `PERMISSION_ERROR` | File/directory permission denied | Check permissions |
| `INVALID_STATE` | Operation not valid in current state | Guide user (e.g., run `polis init` first) |

### Error Example
```json
{
  "status": "error",
  "command": "publish",
  "error": {
    "code": "FILE_NOT_FOUND",
    "message": "File not found: article.md",
    "details": {}
  }
}
```

---

## Parsing Tips

### Extract specific fields with jq
```bash
# Get content hash from publish
./cli/bin/polis --json publish post.md | jq -r '.data.content_hash'

# Get pending request count
./cli/bin/polis --json blessing requests | jq -r '.data.count'

# List all request IDs
./cli/bin/polis --json blessing requests | jq -r '.data.requests[].id'

# Check for error
result=$(./cli/bin/polis --json publish test.md 2>&1)
if echo "$result" | jq -e '.status == "error"' > /dev/null; then
  echo "Error: $(echo "$result" | jq -r '.error.message')"
fi
```

### Handle missing fields gracefully
```bash
# Use // for fallback values
hash=$(echo "$result" | jq -r '.data.content_hash // "unknown"')
```
