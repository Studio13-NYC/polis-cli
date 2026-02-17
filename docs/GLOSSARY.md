# Polis Glossary

Quick reference for polis-specific terminology.

---

### beseech

Request blessing from a post author via the discovery service. When you publish a comment, it automatically beseeches the original author for their approval.

**Related**: blessing, comment, discovery service

---

### blessed-comments.json

Local index file (`metadata/blessed-comments.json`) storing approved comments from other authors. Populated by `polis blessing grant`, `polis follow`, or `polis blessing sync`. Used during render to embed blessed comments in post HTML.

**Related**: blessing, following.json, render

---

### blessing

An author's approval of a comment, making it visible and amplified to their audience. The blessing model is polis's anti-spam mechanism: comments exist regardless, but only blessed ones are promoted. Statuses: pending, blessed, denied.

**Related**: beseech, comment, follow

---

### canonical_url

The authoritative HTTPS URL where a post or comment is published (e.g., `https://alice.com/posts/2026/01/hello.md`). Set automatically based on `POLIS_BASE_URL` and used for signature verification.

**Related**: frontmatter, signature, version

---

### comment

A reply to a post or another comment, published on the commenter's own domain. Comments are signed, include `in_reply_to` metadata pointing to the parent, and automatically request blessing from the original author.

**Related**: blessing, post, beseech

---

### discovery service

A centralized registry (Supabase Edge Functions) that coordinates blessing requests between authors. It receives beseech requests, verifies signatures, stores blessing status, and allows querying blessed comments on any post.

**Related**: beseech, blessing, signature

---

### follow / unfollow

**Follow**: Add an author to your trust list, auto-blessing all their future comments on any of your posts. Stored in `following.json`.

**Unfollow**: Remove an author from your trust list and remove all their previously blessed comments (destructive operation).

**Related**: blessing, following.json

---

### following.json

Local index file (`metadata/following.json`) listing authors you trust. Comments from followed authors are automatically blessed without manual review.

**Related**: follow, blessed-comments.json

---

### frontmatter

YAML metadata section at the top of markdown posts and comments, enclosed by `---` markers. Contains `canonical_url`, `version`, `author`, `published`, `signature`, and for comments, `in_reply_to`.

**Related**: canonical_url, signature, version

---

### manifest

Site metadata file (`metadata/manifest.json`) storing configuration like `active_theme`, `post_count`, and `comment_count`. Read by the render command to determine theming. Note: `site_title` is stored in `.well-known/polis`.

**Related**: theme, render

---

### post

Original content published by an author to their own domain. Posts are markdown files with signed frontmatter, stored in `posts/YYYY/MM/`, indexed in `public.jsonl`, and rendered to HTML using themes.

**Related**: comment, signature, render

---

### public key

The Ed25519 public key used to verify signatures on your posts and comments. Published at `.well-known/polis` so anyone (including the discovery service) can verify your content's authenticity.

**Related**: signature, .well-known/polis

---

### public.jsonl

Line-delimited JSON index (`metadata/public.jsonl`) containing metadata for all published posts and comments. Each line is a JSON object with URL, type, title, published date, version, and author. Used to generate `index.html`.

**Related**: post, comment, JSONL

---

### render

Convert markdown posts and comments to styled HTML using the active theme's templates. The `polis render` command processes all content, applies mustache templating, embeds blessed comments, and generates `index.html`.

**Related**: theme, snippet, template

---

### signature

An Ed25519 cryptographic signature embedded in post/comment frontmatter, proving the content was authored by the claimed author and hasn't been tampered with. Verified by the discovery service before accepting blessing requests.

**Related**: public key, Ed25519, frontmatter

---

### snippet

A reusable template fragment (HTML or markdown) included in rendered pages via `{{> snippet-name}}` syntax. Theme snippets live in `.polis/themes/{theme}/snippets/`; global snippets in `snippets/` override theme defaults.

**Related**: theme, template, render

---

### theme

A complete styling package containing HTML templates (`index.html`, `post.html`, `comment.html`, `comment-inline.html`), CSS, and snippets. Polis ships with three themes: turbo, zane, and sols. Set in `manifest.json`.

**Related**: snippet, render, manifest

---

### TUI

*(Deprecated — use the [webapp](WEBAPP-USER-MANUAL.md) instead.)*

Terminal User Interface (`polis-tui`): a menu-driven, interactive dashboard for polis operations. Was replaced by the webapp in v0.46.0.

**Related**: CLI, webapp

---

### version

SHA-256 hash of content used for content-addressing and change detection. Updated by `polis republish` when content changes. Version history stored in `.versions/` directories alongside content.

**Related**: canonical_url, signature, frontmatter

---

### .well-known/polis

Public metadata file at your domain root announcing author details, email, and Ed25519 public key. Used by the discovery service and other readers to verify your content's signatures.

**Related**: public key, signature, canonical_url
