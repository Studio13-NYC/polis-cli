#!/bin/bash
# test_version.sh - Unit tests for 'polis version' command
#
# Tests covered:
#   - Basic version outputs version string
#   - JSON mode outputs structured version data

# Test: Basic version works correctly
test_version_basic() {
    setup_test_env "version_basic"
    trap teardown_test_env EXIT

    local result
    result=$("$POLIS_BIN" version 2>&1)
    local exit_code=$?

    assert_exit_code 0 "$exit_code" || return 1

    # Should output "polis X.Y.Z" format
    if echo "$result" | grep -q "^polis [0-9]"; then
        log "  [OK] Version string format correct: $result"
    else
        log_error "Expected 'polis X.Y.Z' format, got: $result"
        return 1
    fi

    return 0
}

# Test: JSON mode version
test_version_json() {
    setup_test_env "version_json"
    trap teardown_test_env EXIT

    local result
    result=$("$POLIS_BIN" --json version 2>&1)
    local exit_code=$?

    assert_exit_code 0 "$exit_code" || return 1
    assert_valid_json "$result" || return 1
    assert_json_success "$result" "version" || return 1
    assert_json_has_field "$result" ".data.version" || return 1

    # Verify version format
    local version
    version=$(echo "$result" | jq -r '.data.version')
    if echo "$version" | grep -q "^[0-9]"; then
        log "  [OK] JSON version: $version"
    else
        log_error "Invalid version format in JSON: $version"
        return 1
    fi

    return 0
}

# Run tests
run_test "Version Basic" test_version_basic
run_test "Version JSON" test_version_json
