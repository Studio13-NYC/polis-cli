# Polis CLI

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](https://github.com/vdibart/polis-cli?tab=AGPL-3.0-1-ov-file)
[![Platform: Linux | macOS](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey.svg)]()

**Decentralized social networking for the AI era.**

Your content, free from platform control.

---

## Remember the open web?

Before Twitter became a fiefdom. Before LinkedIn owned your professional network. Before Substack took 10% and controlled your subscriber list. There was a time when you published to *your* domain, people subscribed via RSS, and discovery was joyful.

**We can have that again—but better.**

Polis is federated social networking where:

- **Your content lives on your domain** — Publish to GitHub Pages, Vercel, Netlify, or any static host
- **No platform algorithms** — Follow who you choose, no engagement optimization
- **Cryptographically signed** — Ed25519 signatures prove authorship, SHA-256 hashes ensure integrity
- **AI handles the hard parts** — Your AI controls the algorithm. Publishing, discovery, summaries, trends. Bring your own model.
- **Standards-based** — Just HTTPS, DNS, and cryptography. No blockchain, no tokens, no lock-in

Read the [full manifesto](docs/MANIFESTO.md) for more on our vision.

---

## Try it now

Two paths based on how you like to work:

### Interactive mode (recommended for new users)

```bash
$ ./bin/polis-tui
```

Menu-driven dashboard with keyboard navigation:
- **Publish** — Write posts in your editor, auto-suggested filenames
- **Comment** — Preview target post, compose reply
- **Blessings** — Review requests one-by-one with grant/deny/skip
- **Discover** — Browse posts from authors you follow
- **Preview** — View any polis content with signature verification

No commands to memorize. Git integration built in.

### Command line mode

```bash
$ polis init
[OK] Generated Ed25519 keypair in .polis/keys/
[OK] Created .well-known/polis (public metadata)
[OK] Ready to publish

$ polis post my-thoughts.md
[i] Content hash: sha256:a3b5c7d9e1f2...
[i] Signing with Ed25519 key...
[OK] Published: posts/20260106/my-thoughts.md
```

Full scriptability, JSON output for automation, composable with other tools.

### Learn by doing

New to Polis? Try the interactive tutorial:

```bash
$ ./bin/polis-tutorial
```

Walks you through the complete workflow with simulated commands—no real changes to your system.

---

## The Vision: AI as Your Social Layer

Today, Polis is a CLI. Tomorrow, your AI handles everything:

```
You: "Summarize what people in my network are talking about this week"

Claude: Based on 47 posts from 12 authors you follow:
        - Alice and Bob are debating LLM reasoning capabilities (23 comments)
        - Carol published a 3-part series on distributed systems
        - New author recommendation: David (3 people you follow blessed his comments)

You: "Show me everything Bob commented on yesterday"

Claude: Bob left 4 comments yesterday:
        - On Alice's "LLM Reasoning" post - disagreeing with her benchmark methodology
        - On your "Polis Architecture" post - asking about signature verification
        - On Carol's "Distributed Systems pt 2" - sharing a related paper
        - On David's "First Post" - welcoming him to the network

You: "Who's been most active in discussions about distributed systems?"

Claude: This month, 8 authors discussed distributed systems:
        1. Carol (12 posts, 34 blessed comments received)
        2. Bob (3 posts, 18 comments given)
        3. Alice (2 posts, sparked the most debate)
        You might enjoy Carol's series - 4 people you follow blessed it.

You: "Draft a reply to Bob's question on my post"
```

The structured data—signed markdown, blessed comments, following graphs—is designed for AI to orchestrate. Not to trap you in another algorithm, but to give you a natural language interface to your own network.

---

## Quick start

### Prerequisites

- **OpenSSH 8.0+** — Ed25519 signing
- **jq** — JSON processing
- **curl** — API communication
- **pandoc** — Markdown to HTML (for `polis render`)
- **git** — Version control (optional)

```bash
# macOS
brew install openssh jq curl pandoc git

# Ubuntu/Debian
sudo apt-get install openssh-client jq curl pandoc git
```

### Install

```bash
git clone https://github.com/vdibart/polis-cli.git
export PATH="$PATH:$(pwd)/polis-cli/bin"
```

### Initialize your site

```bash
mkdir my-blog && cd my-blog
polis init
export POLIS_BASE_URL="https://yourdomain.com"
```

### Publish your first post

**Interactive:**
```bash
polis-tui  # Select "Publish", write in your editor
```

**Command line:**
```bash
echo "# Hello World" > hello.md
polis post hello.md
```

### Deploy

```bash
polis render                    # Generate HTML
git init && git add . && git commit -m "First post"
git push                        # To GitHub Pages, Netlify, etc.
```

### Verifying your download

After cloning, verify the scripts haven't been altered:

```bash
cd cli/bin
sha256sum -c polis.sha256
sha256sum -c polis-tutorial.sha256
```

You should see `polis: OK` and `polis-tutorial: OK`

---

## How it works

### System architecture

Polis is a three-part system:

```
┌─────────────────┐      ┌───────────────────┐      ┌─────────────────┐
│   Polis CLI     │─────▶│ Discovery Service │◀─────│  Static Host    │
│   (this repo)   │      │    (Supabase)     │      │ (GitHub Pages,  │
│                 │      │                   │      │  Vercel, etc.)  │
└─────────────────┘      └───────────────────┘      └─────────────────┘
        │                         │                         │
   Signs content            Indexes comments           Serves your
   Manages blessings        Routes blessing requests   signed content
   Tracks following         Verifies signatures        to the world
```

- **Polis CLI** — Local tool that signs your content and manages your social graph
- **Discovery Service** — Federated index that helps authors find comments on their posts
- **Static Hosting** — Any HTTPS host serves your content; you control the files

No single point of failure. Move hosts anytime. Run your own discovery service if you want.

### You own your content

Posts are markdown files with cryptographic signatures. Host them anywhere—GitHub Pages, Vercel, Netlify, your own server. Move anytime. No lock-in. No export needed.

### The blessing model

When someone comments on your post:
1. They publish the comment on *their* domain
2. They request your "blessing" (via a discovery service)
3. You review and approve (or ignore)
4. Blessed comments get amplified to your audience

```
   Your Post                        Their Comment
       |                                 |
       |<------ blessing request --------|
       |                                 |
       v                                 v
  Your audience                    Their audience
       |                                 |
       +---------(if blessed)------------+
                     |
              Combined reach
```

**Anyone can respond. You curate what gets amplified.** Anti-spam without censorship.

### Following and trust

- **Follow an author** — Auto-bless all their future comments
- **Bless a comment** — Auto-bless their future comments on that post
- **Unfollow** — Hide all their comments

```bash
$ polis follow https://alice.example.com
[OK] Following alice.example.com
[OK] 3 existing comments auto-blessed
```

---

## Render to a deployable website

Your posts and comments are markdown files. The `render` command turns them into a complete static website:

```bash
$ polis render
[i] Rendering posts...
[✓] posts/20260106/hello.md → posts/20260106/hello.html
[✓] posts/20260108/followup.md → posts/20260108/followup.html
[i] Rendering comments...
[✓] comments/20260107/reply.md → comments/20260107/reply.html
[i] Generating index...
[✓] index.html (2 posts)
[✓] Render complete: 3 files updated
```

The result is ready to deploy to any static host—GitHub Pages, Vercel, Netlify, your own server.

### What render does

- **Converts markdown to HTML** using pandoc with customizable templates
- **Embeds blessed comments** directly in post pages
- **Generates an index page** listing all your posts
- **Skips unchanged files** for fast incremental builds
- **Embeds signed frontmatter** in every HTML file (see below)

### Customize your templates

```bash
$ polis render --init-templates
[✓] Created .polis/templates/post.html
[✓] Created .polis/templates/comment.html
[✓] Created .polis/templates/comment-inline.html
[✓] Created .polis/templates/index.html
```

Edit these files to change your site's look. Templates use `{{variable}}` substitution for title, content, dates, signatures, and more. See [TEMPLATING.md](docs/TEMPLATING.md) for the full variable reference.

### Verifiable HTML

Every rendered HTML file includes the signed frontmatter as an embedded comment:

```html
<!--
=== POLIS SOURCE ===
Source: https://yourdomain.com/posts/20260106/hello.md
title: Hello World
published: 2026-01-06T12:00:00Z
signature: AAAAB3NzaC1lZDI1NTE5...
=== END POLIS SOURCE ===
-->
```

This means anyone can verify the HTML matches its cryptographic signature—the signature and metadata travel with the rendered page.

---

## Going deeper

### Scripting & automation

All commands support `--json` for machine-readable output:

```bash
# Get content hash after publishing
hash=$(polis --json post draft.md | jq -r '.data.content_hash')

# Pipe content directly (no temp files)
echo "# Quick thought" | polis post - --filename thought.md

# Auto-grant all pending blessings
polis --json blessing requests | jq -r '.data.requests[].id' | \
  xargs -I{} polis blessing grant {}
```

See [USAGE.md](docs/USAGE.md) for complete command reference and JSON mode documentation.

### Claude Code integration

Polis includes a [Claude Code](https://claude.ai/claude-code) skill for AI-powered workflows. Instead of memorizing commands, just describe what you want:

```
You: "publish my draft about distributed systems"
You: "check if I have any pending blessing requests"
You: "comment on Alice's latest post agreeing with her point about caching"
```

#### Installing the skill

```bash
# Create skills directory if it doesn't exist
mkdir -p ~/.claude/skills

# Symlink the polis skill
ln -s "$(pwd)/cli/skills/polis" ~/.claude/skills/polis
```

#### What the skill does

- **Publish** — Signs and publishes posts, suggests titles, offers git commits
- **Discover** — Searches your network for relevant content
- **Comment** — Drafts replies matching your writing tone
- **Manage Blessings** — Reviews pending requests with recommendations
- **Status** — Shows dashboard of your polis activity

The skill uses `--json` mode for reliable parsing and handles errors gracefully.

#### Skill documentation

- **[skills/polis/SKILL.md](skills/polis/SKILL.md)** — Skill overview and workflows
- **[skills/polis/references/commands.md](skills/polis/references/commands.md)** — CLI command reference
- **[skills/polis/references/json-responses.md](skills/polis/references/json-responses.md)** — JSON response schemas

---

## Why Polis?

> "AI can lead us back to the open web we lost instead of down another rabbit hole."

We're not fighting Twitter's network effects. We're serving the people platforms are losing.

**For people who want conversation, not consumption:**
- Substack authors tired of platform fees and comment spam
- Bloggers who remember RSS and Google Reader
- Anyone who values ownership over convenience

**Technical principles:**
- **Standards-based** — HTTPS, DNS, Ed25519. No blockchain, no tokens
- **File-based** — Markdown + frontmatter, version controlled, portable
- **AI-agnostic** — Structured data works with any AI. No lock-in

---

## Documentation

- **[USAGE.md](docs/USAGE.md)** — Complete command reference
- **[JSON-MODE.md](docs/JSON-MODE.md)** — JSON output format for scripting
- **[TEMPLATING.md](docs/TEMPLATING.md)** — HTML template customization
- **[SECURITY-MODEL.md](docs/SECURITY-MODEL.md)** — Cryptographic security deep-dive
- **[MANIFESTO.md](docs/MANIFESTO.md)** — Vision and philosophy
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — Development guidelines
- **[SECURITY.md](SECURITY.md)** — Security policy

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

Questions or issues? [Open a GitHub issue](https://github.com/vdibart/polis-cli/issues)

## License

**AGPL-3.0** — See [LICENSE](https://github.com/vdibart/polis-cli?tab=AGPL-3.0-1-ov-file)

---

*Polis: Your content, free from platform control.*
