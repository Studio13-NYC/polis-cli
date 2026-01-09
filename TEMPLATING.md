# Polis Templating System

The polis templating system renders markdown posts and comments to HTML, enabling blog-like interfaces for your polis site.

## Purpose

Polis stores content as signed markdown files with YAML frontmatter. While this format is ideal for:
- Cryptographic verification
- AI/LLM consumption
- Programmatic access via `public.jsonl`

It's not ideal for human readers browsing your site. The templating system bridges this gap by generating HTML files that:
- Display beautifully in browsers
- Include blessed comments inline on post pages
- Provide an index page listing all content
- Preserve the original markdown files for verification

## Quick Start

```bash
# Initialize templates (optional - uses built-in defaults otherwise)
polis render --init-templates

# Render all posts and comments to HTML
polis render

# Force re-render everything (ignore timestamps)
polis render --force
```

## How It Works

### Rendering Process

1. **Scan content directories** - Finds all `.md` files in `posts/` and `comments/`
2. **Check timestamps** - Skips files where `.html` is newer than `.md` (unless `--force`)
3. **Extract frontmatter** - Reads title, published date, signature, etc.
4. **Convert to HTML** - Uses pandoc to render markdown body
5. **Apply template** - Substitutes variables into HTML template
6. **Write output** - Creates `.html` file alongside `.md` file
7. **Generate index** - Creates `index.html` from `public.jsonl`

### File Relationships

```
posts/
└── 2026/
    └── 01/
        ├── my-post.md           # Source (signed markdown)
        └── my-post.html         # Generated (rendered HTML)

comments/
└── 2026/
    └── 01/
        ├── reply.md             # Source (signed markdown)
        └── reply.html           # Generated (rendered HTML)

index.html                       # Generated (listing page)
```

The `.md` files remain the source of truth. HTML files are regenerated from them.

## Important Files and Paths

| Path | Purpose |
|------|---------|
| `posts/**/*.md` | Source markdown posts |
| `posts/**/*.html` | Generated HTML posts |
| `comments/**/*.md` | Source markdown comments |
| `comments/**/*.html` | Generated HTML comments |
| `index.html` | Generated listing page |
| `.polis/templates/` | Custom templates (optional) |
| `metadata/public.jsonl` | Content index (used for index.html) |
| `metadata/blessed-comments.json` | Blessed comments (rendered inline) |
| `.well-known/polis` | Site config (title, author name) |

## Configuration

### Site Information

The templating system reads site information from `.well-known/polis`:

```json
{
  "name": "Your Name",
  "email": "you@example.com",
  "public_key": "...",
  "config": {
    "site_title": "My Polis Site"
  }
}
```

If `site_title` is not set, it defaults to the author's name.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `POLIS_BASE_URL` | Your site's public URL (used for canonical links) |

## Template Variables

### Available in All Templates

| Variable | Description | Example |
|----------|-------------|---------|
| `{{site_url}}` | Base URL from `POLIS_BASE_URL` | `https://example.com` |
| `{{site_title}}` | From `.well-known/polis` config or author name | `My Polis Site` |
| `{{year}}` | Current year (for copyright) | `2026` |

### Post and Comment Templates

| Variable | Description | Example |
|----------|-------------|---------|
| `{{title}}` | Post/comment title | `Why I Left Substack` |
| `{{content}}` | HTML-rendered markdown body | `<p>The story begins...</p>` |
| `{{published}}` | Publication date (ISO 8601) | `2026-01-08T12:00:00Z` |
| `{{published_human}}` | Human-readable date | `January 8, 2026` |
| `{{url}}` | Canonical URL | `https://example.com/posts/2026/01/post.md` |
| `{{version}}` | Content hash | `sha256:abc123...` |
| `{{author_name}}` | From `.well-known/polis` | `Alice Smith` |
| `{{author_url}}` | Site base URL | `https://example.com` |
| `{{signature_short}}` | Truncated signature (16 chars) | `AAAAC3NzaC1lZD...` |

### Post-Specific Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{blessed_comments}}` | Rendered HTML of blessed comments | `<article class="comment">...</article>` |
| `{{blessed_count}}` | Number of blessed comments | `3` |

### Comment-Specific Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{in_reply_to_url}}` | Parent post/comment URL | `https://bob.com/posts/original.md` |

### Index Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{posts_list}}` | Generated HTML list of posts | `<li><a href="...">...</a></li>` |
| `{{comments_list}}` | Generated HTML list of comments | `<li><a href="...">...</a></li>` |
| `{{post_count}}` | Number of posts | `12` |
| `{{comment_count}}` | Number of comments | `5` |

## Custom Templates

### Exporting Default Templates

To customize templates, first export the defaults:

```bash
polis render --init-templates
```

This creates `.polis/templates/` with:

| File | Purpose |
|------|---------|
| `post.html` | Single post page |
| `comment.html` | Single comment page |
| `comment-inline.html` | Blessed comment (rendered inside posts) |
| `index.html` | Listing page |

### Template Structure

Templates are plain HTML with `{{variable}}` placeholders:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{title}} - {{site_title}}</title>
</head>
<body>
    <h1>{{title}}</h1>
    <time datetime="{{published}}">{{published_human}}</time>
    <div class="content">{{content}}</div>

    <section class="comments">
        <h2>Comments ({{blessed_count}})</h2>
        {{blessed_comments}}
    </section>
</body>
</html>
```

### Template Loading Priority

1. **Custom templates** - `.polis/templates/*.html` (if exists)
2. **Built-in defaults** - Embedded in the CLI script

If `.polis/templates/` exists but is missing a specific template file, the built-in default is used for that template.

## Blessed Comments

When rendering a post, the system:

1. Looks up the post URL in `metadata/blessed-comments.json`
2. For each blessed comment:
   - Fetches the comment markdown (local file or remote URL)
   - Extracts frontmatter and body
   - Renders body to HTML with pandoc
   - Applies `comment-inline.html` template
3. Concatenates all rendered comments
4. Substitutes into `{{blessed_comments}}` in post template

This creates a complete blog experience with comments displayed inline.

## Troubleshooting

### "pandoc is required for rendering"

Install pandoc:

```bash
# Linux
sudo apt install pandoc

# macOS
brew install pandoc
```

### "Polis not initialized"

Run `polis init` first to create the required directory structure.

### HTML files not updating

The render command skips files where the `.html` is newer than the `.md`. Use `--force` to re-render:

```bash
polis render --force
```

### Template variables not substituting

Check that:
1. Variables use double curly braces: `{{variable}}`
2. Variable names are spelled correctly (case-sensitive)
3. Required data exists (e.g., `POLIS_BASE_URL` for `{{site_url}}`)

### Blessed comments not appearing

Verify that:
1. `metadata/blessed-comments.json` exists and contains entries
2. The post URL in the JSON matches the post being rendered
3. Comment files are accessible (local or via HTTP)

Run `polis blessing sync` to update blessed comments from the discovery service.

### Styling issues

The default templates include minimal inline CSS. For custom styling:

1. Export templates: `polis render --init-templates`
2. Edit `.polis/templates/*.html`
3. Add your own CSS (inline or via `<link>`)
4. Re-render: `polis render --force`

### Index page empty

The index is generated from `metadata/public.jsonl`. Ensure:

1. Posts are published (have frontmatter with `version` field)
2. Run `polis rebuild --content` to regenerate the index

## Examples

### Basic Workflow

```bash
# Write a post
cat > draft.md << 'EOF'
# My First Post

This is my first polis post!
EOF

# Publish it
polis publish draft.md

# Render to HTML
polis render

# View the result
open posts/2026/01/my-first-post.html
```

### CI/CD Integration

```bash
# In your deploy script
polis render --force
rsync -av --include='*.html' --include='*/' --exclude='*' . server:/var/www/
```

### Custom Blog Theme

```bash
# Export and customize templates
polis render --init-templates

# Edit templates with your design
vim .polis/templates/post.html

# Render with custom templates
polis render --force
```

## Dependencies

| Tool | Required | Purpose |
|------|----------|---------|
| pandoc | Yes | Markdown to HTML conversion |
| jq | Yes | JSON parsing |
| curl | For remote comments | Fetching blessed comments |

## Future Enhancements

Planned features for future releases:

- RSS/Atom feed generation
- Pagination for index page
- Tag/category pages
- Partial re-render (specific files only)
- Custom CSS file support
- Dark mode toggle in default templates
