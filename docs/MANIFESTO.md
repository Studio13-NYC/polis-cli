# Polis: A Social Layer for the Open Web

## The Consensus Position

Social media platforms are walled gardens. Your content, your audience, your identity—all held at the pleasure of a company that can change the rules, alter the algorithm, or disappear entirely.

- **Substack** takes 10% and owns the subscriber list—you're a tenant, not an owner
- **Medium** shows competitor content below your articles—your readers are their product
- **Twitter/X** can vaporize your account and audience with no recourse
- **LinkedIn** charges you to reach followers you already have

We need a better meeting place.

---

## Where Previous Attempts Went Wrong

Most decentralization efforts started from the question: "How do we build Twitter/Facebook, but decentralized?"

This framing begs for complex architectures based on federation protocols and instance selection.

The result: systems that are difficult to explain, challenging to configure and deploy, confusing to join, and not useful until you've already convinced your friends to switch.  It's just a different kind of lock-in.

Polis asks a different question.

---

## A Different Starting Question

What if social didn't require a platform?

Not "how do I claim my social space" but **"how do we gather without a third party dictating the terms?"**

This framing changes the possibilities:

- A private group of five people sharing links and commentary? That works.
- A solo writer publishing essays and fielding responses? Same primitives.
- A community that wants shared ownership, no single authority? Also possible.
- A commons that nobody owns? The architecture supports it.

The key shift: Polis' primitives don't have an agenda. They can be assembled into any shape. The system is agnostic to governance—it just provides the building blocks for gathering and conversation. What you do with them is up to the people involved, not the platform.

---

## The Architecture That Follows

**Self-hostable by default.** No instance to choose, no federation to configure. The software runs wherever you want it to run—from Vercel to your own server to a managed server or whitelabeled app. Content that exists as files on your domain cannot be deplatformed, rate-limited, or algorithmically suppressed. It's just there. On the web.

**Composable primitives.** Posts, comments, identity, discovery—each is a separable concern. Use what you need. The same pieces work for someone running a private group and someone building a public forum.

**Plain text and well-known structures.** Markdown files, predictable paths, signed content. No proprietary formats, no API required to read your own data.

| Primitive | Implementation | Why It Matters |
|-----------|----------------|----------------|
| **Content** | Markdown + cryptographic signatures | Files, not database records. Impossible to lock in. |
| **Identity** | Your domain + private key | `alice.com`, not `npub1x7fq...` or `@alice@mastodon.social` |
| **Hosting** | Static files | GitHub Pages, Vercel, your own nginx. Already free. |
| **Discovery** | Coordination service + AI | A thin layer, not a platform |

The magic isn't in any single piece—it's in what emerges when you combine them.

---

## How the Social Part Works

"But wait, am I just going to be screaming into the void?  How do people find me?"  Polis' **Discovery Service** is a layer of metadata management. The Discovery Service doesn't store content.  It stores metadata about the content on the network to enable connection and discovery.

Most users will rely on the canonical Discovery Service. Some communities will run their own. Some users may use none at all. Nothing breaks. The Discovery Service improves the experience but the experience doesn't depend on it.

"Ugh.  But public content == endless spam."  Nope.  With Polis' **blessing model** your comments publish to your server.  The original post's author can amplify them (or not) but the comment lives on.

Conversations aren't censored. They're curated.

"What if I don't want all of my content to be public?"  You control your infrastructure, so you control access. Don't list something in your public metadata and it's unlisted. Use server rules to restrict access. Put an authentication layer in front. Encrypt content for specific recipients.

The internet already solved these problems.  Polis isn't trying to re-invent solutions.

---

## Seven Experience Levels, One Model

Polis defines [seven experience levels](https://github.com/vdibart/polis-cli/blob/main/docs/EXPERIENCE-PRINCIPLES.md), from CLI power user to casual content creator. At each layer, Polis presents a natural system interface with an underlying guarantee: moving between layers (e.g. from managed hosting to self-hosting) or within layers (e.g. from one managed host to another) is frictionless.

This is not a policy or a promise. This is the design.

---

## Where This Goes

Today, Polis looks like a publishing tool. But the primitives point somewhere:

- **Publishing** becomes signing files to your domain. No platform needed.
- **Following** becomes a list of URLs. No algorithm decides what you see.
- **Conversation** becomes linked posts across domains. No one owns the thread.
- **Reputation** becomes your cryptographic history. No platform can grant or revoke it.

Taken to its logical end, this inverts the structure of the internet. Instead of platforms owning your content and identity, you own them—and platforms (if they exist at all) compete to provide services on top of what you control.

That's the long-term vision. Not a better social network, but the end of social networks as a category.

---

## An Emergent Benefit

It turns out that the same properties that make content portable and governance-agnostic—structured data, plain text, predictable locations—also make it friendly for AI agents.

Most platforms are actively hostile to automated access. Polis content is readable by anything that can fetch a URL and parse markdown.

This isn't a bet on any particular AI future. It's a consequence of building for simplicity and openness.

The conditions that made platforms necessary are also disappearing: hosting became free and permanent, platform trust is collapsing, and discovery no longer requires a centralized feed. The convenience moat is eroding.

---

## Business Model: Open Core

The line is clear: everything below the ownership bar is free and open. Above that bar, paid services provide convenience.

**Always free:**
- access to your content
- baseline tools (CLI / webapp), protocol spec, data format
- Ability to crawl content, switch providers

**Potentially paid:**
- Managed hosting and domain provisioning
- Real-time data firehose, advanced analytics
- Specialized discovery services

**The test:** Can a technical user do this for free? If yes, the paid version is convenience, not lock-in.

---

## The Ask

Polis is built lean by design—one person and AI, minimal infrastructure, no external capital. It doesn't need investment to survive.

What it needs is people who care about this problem:

- **Early adopters** who'll tolerate rough edges because they understand what's at stake
- **Builders** who see the primitives and want to create clients, tools, or discovery services
- **Critics** who can stress-test the model and find where it breaks

The goal has never been to compete with Twitter on scale. It's to prove that social can work without platforms—and to build the infrastructure so that when people are ready to leave, there's somewhere real to go.
