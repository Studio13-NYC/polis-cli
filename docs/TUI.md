# Terminal User Interface (polis-tui)

For users who prefer a menu-based interface, polis includes a Terminal UI that wraps the CLI.

## Features

- **Dashboard** with status overview (posts, following, pending blessings)
- **Inline selection** with arrow keys or number keys
- **$EDITOR integration** for writing posts and comments
- **Git integration** with automatic commit messages and push workflow
- **HTML regeneration** prompts after content changes

## Usage

```bash
# Start the TUI
polis-tui

# Show help
polis-tui --help

# Show version
polis-tui --version
```

## Navigation

| Keys | Action |
|------|--------|
| `↑/↓` or `j/k` | Move selection up/down |
| `1-9` | Jump directly to numbered option |
| `Enter` | Confirm selection |
| `q` or `Esc` | Go back / Cancel |
| `Ctrl+C` | Exit immediately |

## Workflow Example

1. Start `polis-tui` in your initialized polis directory
2. Select "Publish new post" (arrow keys or press `1`)
3. Write your content in the editor, save and close
4. Enter a filename when prompted
5. Choose to regenerate HTML or commit to git
6. Edit the suggested commit message if needed
7. Push to remote when prompted

## Requirements

- Same dependencies as polis CLI (jq, curl, etc.)
- Polis must be initialized (`polis init`)
- `$EDITOR` environment variable (falls back to nano/vi)
