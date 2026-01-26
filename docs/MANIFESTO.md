# Polis: The Social Network That Isn't

---

## The Thesis

Every attempt to "fix" social media has made the same mistake: competing with platforms on their terms. Build a better Twitter. A more decentralized Facebook. A less toxic Instagram.

Polis asks a different question: **What if social networking didn't require a network at all?**

The open web already exists. Billions of pages, accessible to anyone with a browser. The infrastructure is there. What's missing is the social layer—the discovery, the conversation, the sense of community that platforms captured and enclosed.

Polis doesn't build a new network. It adds a social layer to the web that already exists. Your content stays on your domain. Your identity is your URL. The "platform" is just a coordination point that helps people find each other.  All of this made to feel either like a CLI or a creator platform depending on how you want it to work.

The architecture scales from blogging tool to Instagram clone and everything in between.

---

## Kingdoms Need Walls

Platform economics have a gravity that pulls everything toward extraction:

- **Substack** takes 10% and owns the subscriber list—you're a tenant, not an owner
- **Medium** shows competitor content below your articles—your readers are their product
- **Twitter/X** can vaporize your account and audience with no recourse
- **LinkedIn** charges you to reach followers you already have

The "alternatives" accept the premise that you need *someone's* infrastructure. Mastodon requires Rails + PostgreSQL + Redis. Nostr scatters your content across relay servers you don't control. Bluesky promises decentralization while running a centralized service.

**They're all building better cages.** Polis asks why you need a cage at all.

The infrastructure for publishing is already free and permanent: static file hosting. GitHub Pages. Vercel. Cloudflare. Your own server. Content that exists as files on your domain cannot be deplatformed, rate-limited, or algorithmically suppressed. It's just... there. On the web. Forever.

---

## What Polis Actually Is

On the surface: a tool for publishing signed markdown files to your domain.

Underneath: **a protocol for making the open web social.**

Every Polis site exposes content indexes and metadata in standard formats. This transforms static websites into nodes in a social graph that humans *or* AI can traverse, query, and reason about.

**The primitives are deliberately simple:**

| What | How | Why It Matters |
|------|-----|----------------|
| **Content** | Markdown + cryptographic signatures | Files, not database records. Impossible to lock in. |
| **Identity** | Your domain | `alice.com`, not `npub1x7fq...` or `@alice@mastodon.social` |
| **Hosting** | Static files | GitHub Pages, Vercel, your own nginx. Already free. |
| **Discovery** | Coordination service + AI | A thin layer, not a platform |

The magic isn't in any single piece—it's in what emerges when you combine them.

**This is social networking without a social network.**

Want full CLI control and self-hosting?  No problem.  Prefer if Polis feels like Twitter?  Same.

---

## "But Wait—How Does the Social Part Work?"

The immediate question: if everyone publishes to their own domain, how do comments, sharing, and discovery happen?

Polis' Discovery Service is a federated layer of metadata management.  It does not store content, it stores the minimal information about the content to enable connection, discovery, and analysis of content that is shared across the network.

Most users will rely on the canonical Discovery Service.  Some communities will opt to run their own Discovery Service.  Some users may decide to use no Discovery Service at all.  Nothing breaks.  The Discovery Service improves the experience but the experience doesn't rely on it.

**What about private or premium content?** You control your domain, so you control access. The spectrum is natural: don't list something in your public metadata and it's unlisted. Use `.htaccess` or server rules to restrict access. Put an application layer in front for authentication. Encrypt content for specific recipients. These aren't Polis features—they're standard web capabilities that work because your content is on infrastructure you control.

Polis isn't trying to re-invent anything.  It's trying to restore what has already been invented to make the open web fun again.

---

## Where This Goes

Today, Polis looks like a blogging tool. But the primitives peer into the future: conversations across domains, reputation as cryptographic history you own, AI agents navigating the social graph on your behalf, community moderation through shared blocklists and curated discovery.

What if the extractive, centralized internet isn't inevitable—just the result of constraints that no longer apply?

**If the primitives work, the implications compound:**

- Publishing becomes signing files to your domain. No platform needed.
- Following becomes a list of URLs. No algorithm decides what you see.
- Conversation becomes linked posts across domains. No one owns the thread.
- Reputation becomes your cryptographic history. No platform can grant or revoke it.

Taken to its logical end, this inverts the structure of the internet. Instead of platforms owning your content and identity, you own them—and platforms (if they exist at all) compete to provide services on top of what you control.

**That's the long-term vision. Not a better social network, but the end of social networks as a category.**

---

## The End of Spam: The Blessing Model

Comments on platforms face a trilemma: allow spam, require heavy-handed censorship, or disable comments entirely. Polis sidesteps this with **blessing**—anyone can comment by publishing on their domain, but you choose which comments to amplify to your audience. Unblessed comments still exist publicly; they're just not promoted.

**Comments aren't censored. They're curated.**

---

## Seven Experience Levels, One Ownership Model

Polis defines [Seven Experience Levels](EXPERIENCE-PRINCIPLES.md) from CLI power user to casual content creator.  At each layer Polis presents a natural, thoughtful interface with the underlying guarantee that moving between the layers (e.g. from managed hosting to self-hosting) or within layers (e.g. move from one managed hosting service to another) is completely frictionless.

**This is not a policy, this is the design.**

---

## Why Now

The conditions that made platforms necessary are disappearing:

**AI eliminates the discovery problem.** Platforms existed because browsing distributed content was tedious. AI changes this—it can traverse, filter, and summarize content across domains as easily as querying a database. The convenience moat is gone.

**Hosting became free and permanent.** GitHub Pages. Vercel. Cloudflare Pages. Static file hosting costs nothing, requires no maintenance, and will outlive any VC-funded startup.

**Platform trust is collapsing.** Musk's Twitter. Meta's pivots. TikTok bans. LinkedIn's pay-to-reach model. People now understand viscerally that platforms are not neutral infrastructure—they're businesses that will eventually optimize against user interests.

---

## Competitive Position

Others got pieces right—Nostr's cryptography, Mastodon's federation, Bluesky's protocol ambition—but all still require specialized infrastructure, clients, or trust in specific services. Polis uses the web itself: your content is HTML accessible to any browser, your identity is a domain name meaningful to humans, and self-hosting means uploading files, not running servers.

The goal isn't to win users from Twitter or Mastodon. It's to make the open web social again—and let people realize they don't need platforms at all.

---

## Business Model: Open Core

**The line is clear:** Everything below the ownership bar is free and open. Above that bar, paid services provide convenience.

**Always free:**
- CLI tool, protocol spec, data format
- Desktop and simple web interfaces
- Ability to export, self-host, switch providers

**Potentially paid:**
- Managed hosting and domain provisioning
- Real-time data access
- Advanced analytics and scheduling
- Priority support and onboarding
- Specialized discovery services

**The test:** Can a technical user do this for free? If yes, the paid version is convenience, not lock-in.

---

## The Ask

Polis is built lean by design—one person and AI, minimal infrastructure, no external capital. It doesn't need investment to survive.

What it needs is people who care about this problem:

- **Early adopters** who'll tolerate CLI rough edges because they understand what ownership means
- **Builders** who see the primitives and want to create clients, tools, or discovery services
- **Critics** who can stress-test the model and find where it breaks

The goal has never been to compete with Twitter on scale. It's to prove that social networking can work without platforms—and to build the infrastructure so that when people are ready to leave, there's somewhere real to go.
