# Polis CLI

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](../LICENSE)
[![Platform: Linux | macOS](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey.svg)]()

**Decentralized social networking for the AI era.**

Your content. Your domain. Your network.

---

## Remember the open web?

Before Twitter became a fiefdom. Before LinkedIn owned your professional network. Before Substack took 10% and controlled your subscriber list. There was a time when you published to *your* domain, people subscribed via RSS, and discovery was joyful.

**We can have that again—but better.**

Polis is federated social networking where:

- **Your content lives on your domain** — Publish to GitHub Pages, Vercel, Netlify, or any static host
- **No platform algorithms** — Follow who you choose, no engagement optimization
- **Cryptographically signed** — Ed25519 signatures prove authorship, SHA-256 hashes ensure integrity
- **AI handles the hard parts** — Your AI controls the algorithm.  Publishing, discovery, summaries, trends.  Bring your own model.
- **Standards-based** — Just HTTPS, DNS, and cryptography. No blockchain, no tokens, no lock-in

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
        • On Alice's "LLM Reasoning" post - disagreeing with her benchmark methodology
        • On your "Polis Architecture" post - asking about signature verification
        • On Carol's "Distributed Systems pt 2" - sharing a related paper
        • On David's "First Post" - welcoming him to the network

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

## See it in action

### Initialize and publish a post

```bash
$ polis init
[OK] Generated Ed25519 keypair in .polis/keys/
[OK] Created .well-known/polis (public metadata)
[OK] Ready to publish

$ polis publish my-thoughts.md
[i] Content hash: sha256:a3b5c7d9e1f2...
[i] Signing with Ed25519 key...
[OK] Published: posts/2026/01/my-thoughts.md
```

### Comment on someone's post

```bash
$ polis comment https://alice.example.com/posts/hello.md
# Editor opens - write your reply

[OK] Created: comments/2026/01/reply-to-alice.md
[OK] Blessing requested from alice.example.com
```

### Curate comments on your posts

```bash
$ polis blessing requests
ID    Author              Post
42    bob@example.com     /posts/hello.md
73    carol@example.com   /posts/hello.md

$ polis blessing grant 42
[OK] Comment blessed - now visible to your audience
```

---

## Try the interactive tutorial

New to Polis? Learn by doing with our interactive tutorial:

```bash
./bin/polis-tutorial
```

The tutorial walks you through the complete workflow with simulated commands - no real changes are made to your system. Type the commands shown or press Enter to advance.

---

## Quick start

```bash
# Clone and add to PATH
git clone https://github.com/vdibart/polis-cli.git
export PATH="$PATH:$(pwd)/polis-cli/bin"

# Initialize your site
mkdir my-blog && cd my-blog
polis init

# Set your domain
export POLIS_BASE_URL="https://yourdomain.com"

# Write and publish
echo "# Hello World" > hello.md
polis publish hello.md

# Deploy to any static host
git init && git add . && git commit -m "First post"
git push  # to GitHub Pages, Netlify, etc.
```

### Prerequisites

- **OpenSSH 8.0+** — Ed25519 signing
- **jq** — JSON processing
- **curl** — API communication
- **git** — Version control (optional)

```bash
# macOS
brew install openssh jq curl git

# Ubuntu/Debian
sudo apt-get install openssh-client jq curl git
```

### Verifying Your Download

After cloning, verify the scripts haven't been altered:

```bash
cd cli/bin
sha256sum -c polis.sha256
sha256sum -c polis-tutorial.sha256
```

You should see `polis: OK` and `polis-tutorial: OK`

---

## How it works

### System Architecture

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

### 1. You own your content

Posts are markdown files with cryptographic signatures. Host them anywhere—GitHub Pages, Vercel, Netlify, your own server. Move anytime. No lock-in. No export needed.

### 2. The blessing model

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

### 3. Following and trust

- **Follow an author** — Auto-bless all their future comments
- **Bless a comment** — Auto-bless their future comments on that post
- **Unfollow** — Hide all their comments

```bash
$ polis follow https://alice.example.com
[OK] Following alice.example.com
[OK] 3 existing comments auto-blessed
```

---

## Scripting & Automation

All commands support `--json` for machine-readable output:

```bash
# Get content hash after publishing
hash=$(polis --json publish draft.md | jq -r '.data.content_hash')

# Pipe content directly (no temp files)
echo "# Quick thought" | polis publish - --filename thought.md

# Auto-grant all pending blessings
polis --json blessing requests | jq -r '.data.requests[].id' | \
  xargs -I{} polis blessing grant {}
```

See [USAGE.md](USAGE.md#json-mode) for complete JSON mode documentation.

---

## Documentation

- **[USAGE.md](USAGE.md)** — Complete command reference
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — Development guidelines
- **[SECURITY.md](SECURITY.md)** — Security policy

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

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

Questions or issues? [Open a GitHub issue](https://github.com/vdibart/polis-cli/issues)

## License

**AGPL-3.0** — See [LICENSE](LICENSE)

---

*Polis: Self-governed social networking*
