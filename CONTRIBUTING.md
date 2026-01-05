# Contributing to Polis CLI

Thank you for your interest in contributing to the Polis CLI! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- **Bash 4.0+** (the CLI is a bash script)
- **OpenSSH 8.0+** (for Ed25519 signing)
- **jq** (JSON processor)
- **curl** (for API communication)
- **sha256sum** or **shasum** (for content hashing)
- **ShellCheck** (for linting - optional but recommended)

### Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/vdibart/polis-cli.git
   cd polis-cli
   ```

2. Make the CLI executable:
   ```bash
   chmod +x bin/polis
   ```

3. Add to your PATH for testing:
   ```bash
   export PATH="$(pwd)/bin:$PATH"
   ```

## How to Contribute

### Reporting Bugs

Before submitting a bug report:

1. Check existing issues to avoid duplicates
2. Use the latest version of the CLI
3. Include in your report:
   - OS and version
   - Bash version (`bash --version`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant error messages

### Suggesting Features

Feature requests are welcome! Please:

1. Check existing issues for similar requests
2. Describe the use case clearly
3. Explain why this would benefit other users

### Submitting Changes

1. **Fork** the repository
2. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** following the code style guidelines below
4. **Test** your changes thoroughly
5. **Commit** with clear, descriptive messages
6. **Push** to your fork
7. **Open a Pull Request** with:
   - Clear description of the changes
   - Reference to any related issues
   - Screenshots/output examples if applicable

## Code Style Guidelines

### Bash Best Practices

- Use `shellcheck` to lint your code before submitting
- Quote variables: `"$variable"` not `$variable`
- Use `[[ ]]` for conditionals instead of `[ ]`
- Prefer `$(command)` over backticks for command substitution
- Use meaningful function and variable names
- Add comments for complex logic

### Naming Conventions

- Functions: `snake_case` (e.g., `publish_content`, `verify_signature`)
- Local variables: `snake_case`
- Constants/Environment variables: `UPPER_SNAKE_CASE`

### Code Organization

- Keep functions focused and single-purpose
- Group related functions together
- Document public functions with comments explaining:
  - Purpose
  - Parameters
  - Return values/exit codes

### Example

```bash
# Publishes content to the specified path
# Arguments:
#   $1 - Source file path
#   $2 - Destination directory (optional)
# Returns:
#   0 on success, 1 on failure
publish_content() {
    local source_file="$1"
    local dest_dir="${2:-posts}"

    if [[ ! -f "$source_file" ]]; then
        log_error "File not found: $source_file"
        return 1
    fi

    # ... implementation
}
```

## Testing

### Manual Testing

Before submitting, test your changes with:

```bash
# Initialize a test directory
mkdir /tmp/polis-test && cd /tmp/polis-test
polis init

# Test affected commands
polis publish test-post.md
polis --json publish test-post.md  # JSON mode

# Clean up
rm -rf /tmp/polis-test
```

### JSON Mode

If your changes affect command output, ensure JSON mode still works:

```bash
polis --json <command> | jq .  # Should produce valid JSON
```

## Pull Request Process

1. Ensure your code passes `shellcheck bin/polis`
2. Update documentation if you've changed functionality
3. Add entries to CHANGELOG.md for notable changes
4. PRs require at least one approving review before merge
5. Keep PRs focused - one feature/fix per PR

## Community Guidelines

- Be respectful and constructive in discussions
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md)
- Help others when you can

## Questions?

If you have questions about contributing, feel free to:

- Open a discussion issue
- Ask in an existing related issue

Thank you for contributing to Polis CLI!
