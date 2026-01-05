#!/bin/bash
# fixtures.sh - Test fixture generation utilities for Polis CLI tests
#
# Provides:
#   - Sample content generators for posts and comments
#   - Helper functions for creating test data

# Create a sample post file
# Usage: create_sample_post "filename.md" "Title"
# Returns: path to created file
create_sample_post() {
    local filename="${1:-sample-post.md}"
    local title="${2:-Sample Post}"

    cat > "$filename" << EOF
# $title

This is sample content for testing purposes.

## Section 1

Some paragraphs with content to make this a realistic test file.

## Section 2

More content here for good measure.
EOF
    echo "$filename"
}

# Create a sample comment file
# Usage: create_sample_comment "filename.md" "Title"
# Returns: path to created file
create_sample_comment() {
    local filename="${1:-sample-comment.md}"
    local title="${2:-Sample Comment}"

    cat > "$filename" << EOF
# $title

This is a sample comment for testing purposes.

I found this post very interesting and wanted to share my thoughts.
EOF
    echo "$filename"
}

# Create a longer sample post (for version testing)
# Usage: create_long_post "filename.md" "Title"
create_long_post() {
    local filename="${1:-long-post.md}"
    local title="${2:-Long Post}"

    cat > "$filename" << EOF
# $title

This is a longer post for testing version history and republishing.

## Introduction

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua.

## Main Content

Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi
ut aliquip ex ea commodo consequat.

### Subsection A

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
dolore eu fugiat nulla pariatur.

### Subsection B

Excepteur sint occaecat cupidatat non proident, sunt in culpa qui
officia deserunt mollit anim id est laborum.

## Conclusion

This concludes our sample post content.
EOF
    echo "$filename"
}

# Append content to a file (for republish testing)
# Usage: append_to_file "filename.md" "Additional content"
append_to_file() {
    local filename="$1"
    local content="$2"

    echo "" >> "$filename"
    echo "$content" >> "$filename"
}

# Get a unique test identifier
# Usage: unique_id
# Returns: timestamp-based unique string
unique_id() {
    echo "test-$(date +%s%N | sha256sum | head -c 8)"
}

# Create sample .env content for e2e tests
# Usage: create_test_env "filename"
create_test_env() {
    local filename="${1:-.env}"

    cat > "$filename" << 'EOF'
# Test environment configuration
POLIS_BASE_URL=https://test.example.com
SUPABASE_ANON_KEY=test-anon-key-for-testing
EOF
    echo "$filename"
}
