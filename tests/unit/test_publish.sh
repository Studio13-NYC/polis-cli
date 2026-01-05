#!/bin/bash
# test_publish.sh - Unit tests for 'polis publish' command
#
# Tests covered:
#   - Basic publish creates canonical file with frontmatter
#   - Publish missing file returns error
#   - Publish updates index

# Test: Basic publish works correctly
test_publish_basic() {
    setup_test_env "publish_basic"
    trap teardown_test_env EXIT

    # Initialize first
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    # Create sample post
    create_sample_post "my-post.md" "Test Post"

    local result
    result=$("$POLIS_BIN" --json publish my-post.md 2>&1)
    local exit_code=$?

    assert_exit_code 0 "$exit_code" || return 1
    assert_valid_json "$result" || return 1
    assert_json_success "$result" "publish" || return 1

    # Verify data fields exist
    assert_json_has_field "$result" ".data.file_path" || return 1
    assert_json_has_field "$result" ".data.content_hash" || return 1
    assert_json_has_field "$result" ".data.timestamp" || return 1
    assert_json_has_field "$result" ".data.signature" || return 1

    # Get canonical path from response
    local canonical_path
    canonical_path=$(echo "$result" | jq -r '.data.file_path')

    # Verify canonical file was created
    assert_file_exists "$canonical_path" || return 1

    # Verify original file was moved (no longer exists at original location)
    assert_file_not_exists "my-post.md" || return 1

    # Verify frontmatter was added
    assert_file_contains "$canonical_path" "^---" || return 1
    assert_file_contains "$canonical_path" "title:" || return 1
    assert_file_contains "$canonical_path" "current-version:" || return 1
    assert_file_contains "$canonical_path" "signature:" || return 1

    # Verify index was updated
    assert_file_contains "metadata/public.jsonl" "Test Post" || return 1

    # Verify hash format
    local content_hash
    content_hash=$(echo "$result" | jq -r '.data.content_hash')
    assert_hash_format "$content_hash" || return 1

    return 0
}

# Test: Publish missing file returns error
test_publish_missing_file() {
    setup_test_env "publish_missing_file"
    trap teardown_test_env EXIT

    # Initialize first
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    local result
    result=$("$POLIS_BIN" --json publish nonexistent.md 2>&1)
    local exit_code=$?

    assert_exit_code 1 "$exit_code" || return 1
    assert_valid_json "$result" || return 1
    assert_json_field "$result" ".status" "error" || return 1

    # Check for appropriate error code
    local error_code
    error_code=$(echo "$result" | jq -r '.error.code')
    log "  Error code: $error_code"

    # Should be FILE_NOT_FOUND or similar
    [[ -n "$error_code" && "$error_code" != "null" ]] || return 1

    return 0
}

# Test: Publish without init returns error
test_publish_without_init() {
    setup_test_env "publish_without_init"
    trap teardown_test_env EXIT

    # Do NOT initialize - try to publish directly
    create_sample_post "my-post.md" "Test Post"

    local result
    result=$("$POLIS_BIN" --json publish my-post.md 2>&1)
    local exit_code=$?

    assert_exit_code 1 "$exit_code" || return 1
    assert_valid_json "$result" || return 1
    assert_json_field "$result" ".status" "error" || return 1

    return 0
}

# Test: Publish stages files in git
test_publish_git_staging() {
    setup_test_env "publish_git_staging"
    trap teardown_test_env EXIT

    # Initialize
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    # Commit init files
    git add -A && git commit -m "init" --quiet

    # Create and publish post
    create_sample_post "my-post.md" "Test Post"

    local result
    result=$("$POLIS_BIN" --json publish my-post.md 2>&1)

    # Get canonical path
    local canonical_path
    canonical_path=$(echo "$result" | jq -r '.data.file_path')

    # Check that files are staged
    local staged_files
    staged_files=$(git diff --cached --name-only)

    # Should have the published file staged
    echo "$staged_files" | grep -q "$(basename "$canonical_path")" || {
        log_error "Published file not staged"
        return 1
    }

    log "  [OK] Published file is staged in git"
    return 0
}

# Run tests
run_test "Publish Basic" test_publish_basic
run_test "Publish Missing File" test_publish_missing_file
run_test "Publish Without Init" test_publish_without_init
run_test "Publish Git Staging" test_publish_git_staging
