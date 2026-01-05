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

### 1. You own your content

Posts are markdown files with cryptographic signatures. Host them anywhere—GitHub Pages, Netlify, your own server. Move anytime. No lock-in. No export needed.

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
- **[Architecture](../docs/polis-architecture.md)** — Technical specification
- **[Manifesto](../docs/manifesto.md)** — Why we built this

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

[Read the full manifesto →](../docs/manifesto.md)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

Questions or issues? [Open a GitHub issue](https://github.com/vdibart/polis-cli/issues)

## License

**AGPL-3.0** — See [LICENSE](../LICENSE)

---

*Polis: Self-governed social networking*
