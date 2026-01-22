# Polis Notifications

This document describes the Polis notification system, including all notification types, their data sources, query logic, and implementation status.

---

## Table of Contents

1. [Overview](#overview)
2. [Notification Types](#notification-types)
3. [Local Storage](#local-storage)
4. [CLI Commands](#cli-commands)
5. [Discovery Service Integration](#discovery-service-integration)
6. [Privacy Considerations](#privacy-considerations)

---

## Overview

The Polis notification system follows these design principles:

1. **Discovery service stays dumb** - No triggers, no derived state, no per-user tracking
2. **CLI does aggregation** - The CLI knows your context (who you follow, what you commented on) and generates notifications locally
3. **Edge function does filtering** - Fetches your `following.json` to scope queries, reducing data transfer
4. **Signed requests** - Only you can query your notifications (Ed25519 signature verification)

---

## Notification Types

### Implemented

| Type | Source | Description |
|------|--------|-------------|
| `version_available` | Discovery (`polis_versions`) | A new CLI/TUI version is available (per-component tracking) |
| `version_pending` | Local | A new version was downloaded but metadata files haven't been updated |

### Planned (Phase 2)

| Type | Source | Description |
|------|--------|-------------|
| `new_follower` | Discovery (`follow_metadata`) | Someone you don't follow started following you |
| `followed_author_followed` | Discovery + author's `following.json` | Someone you follow followed/unfollowed another author |

### Planned (Phase 3)

| Type | Source | Description |
|------|--------|-------------|
| `new_post` | Discovery (`posts_metadata`) | An author you follow published a new post |
| `content_updated` | Discovery (`posts_metadata`, `comment_metadata`) | A post or comment you replied to was updated |

### Planned (Phase 4)

| Type | Source | Description |
|------|--------|-------------|
| `blessing_changed` | Discovery (`comment_metadata`) | A comment you made was blessed or unblessed |
| `key_changed` | Author's `.well-known/polis` | A followed author's public key changed |

### Future (Not in Scope)

| Type | Notes |
|------|-------|
| `thread_reply` | Someone blessed replied in a thread you commented on. Requires tracking thread participants. |
| `blessing_response` | Author blessed a comment in response to your comment. Requires tracking comment chains. |
| `author_migrated` | Author moved/relocated posts or comments. Could use existing `domain_migrations` table. |

---

## Notification Type Details

### `version_available`

**Status:** Implemented (Phase 1)

**Data sources:**
- Discovery service: `polis_versions` table (with `component` column for CLI/TUI/upgrade differentiation)
- Local: Current CLI version from `VERSION` constant, TUI from `TUI_VERSION`

**Query logic:**
1. CLI calls `GET /polis-version?current=<version>&component=cli`
2. TUI calls `GET /polis-version?current=<version>&component=tui`
3. Discovery service returns latest version info for that component
4. If `latest > current`, generate notification with upgrade details

**Component tracking:**
The `polis_versions` table tracks versions independently per component (`cli`, `tui`, `upgrade`).
Each component has its own `is_latest` flag and version history. The `polis-upgrade` script
handles migrations and binary updates across version jumps.

**Payload example:**
```json
{
  "type": "version_available",
  "payload": {
    "current_version": "0.35.0",
    "latest_version": "0.36.0",
    "released_at": "2026-01-15T00:00:00Z",
    "download_url": "https://github.com/...",
    "release_notes": "- Feature A\n- Bug fix B"
  }
}
```

---

### `version_pending`

**Status:** Planned (Phase 1)

**Data sources:**
- Local: CLI version from `polis --version`
- Local: Version in `metadata/manifest.json`

**Query logic:**
1. Compare CLI version to version recorded in `metadata/manifest.json`
2. If CLI version > manifest version, the user upgraded but hasn't republished
3. Generate notification suggesting `polis rebuild` or `polis publish`

**Payload example:**
```json
{
  "type": "version_pending",
  "payload": {
    "cli_version": "0.36.0",
    "metadata_version": "0.35.0",
    "suggestion": "Run 'polis rebuild' to update metadata files"
  }
}
```

---

### `new_follower`

**Status:** Planned (Phase 2)

**Data sources:**
- Discovery service: `follow_metadata` table

**Query logic:**
1. CLI calls `GET /notifications?domain=<your_domain>&since=<last_requested_ts>`
2. Edge function queries `follow_metadata WHERE followed_domain = your_domain AND created_at > since`
3. For each result, CLI checks if `follower_domain` is in local `following.json`
4. If not in following list, generate `new_follower` notification

**Payload example:**
```json
{
  "type": "new_follower",
  "payload": {
    "follower_domain": "carol.com",
    "action": "follow",
    "announced_at": "2026-01-20T12:00:00Z"
  }
}
```

**Privacy note:** This notification only works if the follower used `--announce` when following. Follows are opt-in broadcasts.

---

### `followed_author_followed`

**Status:** Planned (Phase 2)

**Data sources:**
- Local: `metadata/following.json` (who you follow)
- Remote: Each followed author's `metadata/following.json`

**Query logic:**
1. For each author in your `following.json`:
   - Fetch their `metadata/following.json`
   - Compare to cached version in `.polis/following-cache/<domain>.json`
   - If entries added/removed, generate notifications
2. Update cache with current version

**Payload example:**
```json
{
  "type": "followed_author_followed",
  "payload": {
    "author_domain": "alice.com",
    "action": "follow",
    "target_domain": "bob.com"
  }
}
```

**Privacy note:** Only detects changes in public `following.json` files. Authors who don't publish this file won't generate these notifications.

---

### `new_post`

**Status:** Planned (Phase 3)

**Data sources:**
- Discovery service: `posts_metadata` table
- Local: `metadata/following.json`

**Query logic:**
1. CLI calls `GET /notifications` with signed request
2. Edge function fetches your `following.json` to get followed domains
3. Queries `posts_metadata WHERE author_domain IN (followed_domains) AND updated_at > since`
4. CLI compares returned posts against local cache
5. New posts (not in cache) generate notifications

**Payload example:**
```json
{
  "type": "new_post",
  "payload": {
    "post_url": "https://alice.com/posts/2026/01/new-article.md",
    "title": "New Article Title",
    "author": "alice@example.com",
    "published_at": "2026-01-20T12:00:00Z"
  }
}
```

---

### `content_updated`

**Status:** Planned (Phase 3)

**Data sources:**
- Discovery service: `posts_metadata` table
- Local: `metadata/public.jsonl` (your comments with `in_reply_to` references)

**Query logic:**
1. CLI reads local `public.jsonl` to find posts you've commented on
2. Extracts list of `in_reply_to` URLs
3. Edge function queries `posts_metadata WHERE post_url IN (commented_urls) AND updated_at > since`
4. For each returned post, compare `current_version` to version you originally replied to
5. If version changed, generate notification

**Payload example:**
```json
{
  "type": "content_updated",
  "payload": {
    "post_url": "https://bob.com/posts/2026/01/original.md",
    "old_version": "sha256:abc123...",
    "new_version": "sha256:def456...",
    "your_comment_url": "https://you.com/comments/2026/01/reply.md"
  }
}
```

---

### `blessing_changed`

**Status:** Planned (Phase 4)

**Data sources:**
- Discovery service: `comment_metadata` table

**Query logic:**
1. Edge function queries `comment_metadata WHERE author_domain = your_domain AND updated_at > since`
2. CLI compares returned blessing statuses against local cache
3. If `blessing_status` changed (pending→blessed, pending→denied, blessed→denied), generate notification

**Payload example:**
```json
{
  "type": "blessing_changed",
  "payload": {
    "comment_url": "https://you.com/comments/2026/01/reply.md",
    "post_url": "https://bob.com/posts/2026/01/hello.md",
    "old_status": "pending",
    "new_status": "blessed",
    "blessed_at": "2026-01-20T12:00:00Z"
  }
}
```

---

### `key_changed`

**Status:** Planned (Phase 4)

**Data sources:**
- Remote: Each followed author's `.well-known/polis`
- Local: Cached public keys in `.polis/key-cache/<domain>.pub`

**Query logic:**
1. For each author in your `following.json`:
   - Fetch their `.well-known/polis`
   - Compare `public_key` to cached version
   - If changed, generate notification (this could indicate key rotation or compromise)
2. Update cache with current key

**Payload example:**
```json
{
  "type": "key_changed",
  "payload": {
    "author_domain": "alice.com",
    "old_key_fingerprint": "SHA256:abc123...",
    "new_key_fingerprint": "SHA256:def456...",
    "warning": "Public key changed. Verify this is intentional."
  }
}
```

**Security note:** Key changes could indicate legitimate rotation or domain compromise. Users should verify out-of-band if unexpected.

---

## Local Storage

Notification state is stored in `.polis/` (private, not published).

### `.polis/notifications.jsonl`

Append-only log of notifications. One JSON object per line. Can be tailed for real-time updates.

**Schema:**
```json
{
  "id": "notif_1737388800_abc123",
  "type": "new_follower",
  "created_at": "2026-01-20T12:00:00Z",
  "source": "discovery",
  "payload": { ... },
  "dedup_key": "new_follower:carol.com"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier: `notif_{timestamp}_{random}` |
| `type` | string | Notification type (see types above) |
| `created_at` | ISO8601 | When notification was created locally |
| `source` | string | `local`, `discovery`, or `author_site` |
| `payload` | object | Type-specific data |
| `dedup_key` | string | Prevents duplicate notifications |

**Lifecycle:**
- New notification → append line to JSONL
- Mark as read → remove line from JSONL
- Dismiss → remove line from JSONL

### `.polis/notifications-manifest.json`

State file for operational metadata.

```json
{
  "version": "0.1.0",
  "last_requested_ts": "2026-01-20T12:00:00Z",
  "preferences": {
    "poll_interval_minutes": 60,
    "enabled_types": ["new_follower", "new_post", "version_available"],
    "muted_domains": []
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Schema version |
| `last_requested_ts` | ISO8601 | Watermark for discovery queries |
| `preferences.poll_interval_minutes` | number | Minimum time between syncs (floor: 15) |
| `preferences.enabled_types` | string[] | Which notification types to generate |
| `preferences.muted_domains` | string[] | Domains to ignore |

---

## CLI Commands

### List Notifications

```bash
# Show unread notifications (default)
polis notifications

# Show all notifications
polis notifications --all

# Filter by type
polis notifications --type new_follower,new_post

# JSON output
polis notifications --json
```

### Manage Notifications

```bash
# Mark as read (removes from list)
polis notifications read <id>
polis notifications read --all

# Dismiss (removes from list)
polis notifications dismiss <id>
polis notifications dismiss --older-than 30d
```

### Sync from Discovery

```bash
# Fetch new notifications
polis notifications sync

# Reset watermark and do full re-sync
polis notifications sync --reset
```

### Configure Preferences

```bash
# Show current config
polis notifications config

# Set poll interval
polis notifications config --poll-interval 30m

# Enable/disable notification types
polis notifications config --enable new_post
polis notifications config --disable following_change

# Mute notifications from specific domain
polis notifications config --mute spam.com
polis notifications config --unmute spam.com
```

### Follow with Announcement

```bash
# Follow and announce to discovery service (opt-in)
polis follow https://alice.com --announce

# Unfollow and announce (opt-in)
polis unfollow https://alice.com --announce
```

---

## Discovery Service Integration

### `GET /notifications` Endpoint

**Request:**
```
GET /notifications?domain=mysite.com&since=2026-01-20T10:00:00Z

Headers:
  X-Polis-Signature: <Ed25519 signature>
  X-Polis-Timestamp: 2026-01-20T12:05:00Z
```

**Signature payload:**
```
GET /notifications domain=mysite.com since=2026-01-20T10:00:00Z timestamp=2026-01-20T12:05:00Z
```

**Edge function behavior:**
1. Verify signature against domain's `.well-known/polis` public key
2. Reject if signature invalid (403) or timestamp > 5 min old (401)
3. Fetch `https://{domain}/metadata/following.json` (cache 10 min TTL)
4. Run scoped queries using followed domains
5. Return filtered results

**Response:**
```json
{
  "posts": [...],
  "comments": [...],
  "followers": [...],
  "queried_at": "2026-01-20T12:05:00Z"
}
```

**Rate limit:** 60 requests per hour per domain

### `POST /follow-announce` Endpoint

**Request:**
```json
{
  "action": "follow",
  "follower_domain": "mysite.com",
  "followed_domain": "alice.com",
  "announced_at": "2026-01-20T12:00:00Z",
  "signature": "..."
}
```

**Rate limit:** 100 requests per day per domain

### `GET /polis-version` Endpoint

**Request:**
```
GET /polis-version?current=0.35.0
```

**Response:**
```json
{
  "latest": "0.36.0",
  "released_at": "2026-01-15T00:00:00Z",
  "download_url": "...",
  "upgrade_available": true
}
```

**Rate limit:** 100 requests per hour (global, no auth required)

---

## Privacy Considerations

### What the Discovery Service Learns

| Data | Visibility | Notes |
|------|------------|-------|
| Your domain | Full | Required for authentication |
| When you poll | Full | Request timestamps |
| Who you follow | Temporary | Fetched on-demand, not stored |
| Your comments | Partial | Only metadata, not content |

### What Remains Private

- **Notification preferences** - Stored locally only
- **Which notifications you've read** - Stored locally only
- **Your full social graph** - Discovery fetches `following.json` on-demand but doesn't store it

### Opt-in Behaviors

- **Follow announcements** - Require explicit `--announce` flag
- **Notification sync** - User initiates with `polis notifications sync`

### Privacy-Preserving Design Choices

1. **Edge function fetches `following.json`** - Scopes queries without storing your social graph
2. **Signed requests** - Prevents enumeration of other users' notifications
3. **Local aggregation** - CLI determines what's relevant, not the server

---

## Document History

- 2026-01-21: Initial version
