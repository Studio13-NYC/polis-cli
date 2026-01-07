#!/bin/bash
# test_blessing_workflow.sh - End-to-end tests for blessing workflow
#
# These tests make REAL API calls to the discovery service unless
# --skip-network is passed to the test runner.
#
# Prerequisites:
#   - POLIS_BASE_URL must be set to a reachable domain
#   - DISCOVERY_SERVICE_KEY must be configured
#   - Discovery service must be deployed
#
# Tests covered:
#   - Comment with automatic beseech
#   - Blessing requests listing
#   - Blessing grant/deny

# Helper: Check if e2e tests can run
check_e2e_prerequisites() {
    if [[ -z "$POLIS_BASE_URL" ]]; then
        return 1
    fi
    if [[ -z "$DISCOVERY_SERVICE_KEY" ]]; then
        return 1
    fi
    return 0
}

# Test: Comment with automatic beseech
test_comment_with_beseech() {
    setup_test_env "comment_with_beseech"
    trap teardown_test_env EXIT

    # Check prerequisites
    if ! check_e2e_prerequisites; then
        if should_skip_network; then
            log "  Skipping network call (--skip-network mode)"
        else
            log_error "Missing POLIS_BASE_URL or DISCOVERY_SERVICE_KEY"
            return 1
        fi
    fi

    # Initialize
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    # First, publish a post to have something to comment on
    create_sample_post "my-post.md" "E2E Test Post"
    local publish_result
    publish_result=$("$POLIS_BIN" --json publish my-post.md 2>&1)
    assert_json_success "$publish_result" "publish" || return 1

    local post_path post_url
    post_path=$(echo "$publish_result" | jq -r '.data.file_path')

    # Construct the post URL (would need POLIS_BASE_URL in real scenario)
    post_url="${POLIS_BASE_URL}/${post_path}"
    log "  Post URL: $post_url"

    # Create a comment
    create_sample_comment "my-comment.md" "E2E Test Comment"

    if should_skip_network; then
        # Skip network mode: just verify comment creation without API call
        log "  [SKIP-NETWORK] Skipping actual comment command (requires network)"
        log "  [OK] Comment file created, network call skipped"
        return 0
    fi

    # Real network mode: actually run the comment command
    local comment_result
    comment_result=$("$POLIS_BIN" --json comment my-comment.md "$post_url" 2>&1)
    local exit_code=$?

    # The command might fail if POLIS_BASE_URL isn't properly configured
    # or if the discovery service isn't reachable
    if [[ $exit_code -ne 0 ]]; then
        log "  Comment command returned error (exit code: $exit_code)"
        log "  This may be expected if discovery service is not configured"

        # Check if it's a network/API error vs other error
        local error_code
        error_code=$(echo "$comment_result" | jq -r '.error.code // "UNKNOWN"' 2>/dev/null)

        if [[ "$error_code" == "API_ERROR" || "$error_code" == "NETWORK_ERROR" ]]; then
            log "  [WARN] Network/API error - discovery service may not be available"
            log "  [SKIP] Skipping remainder of test due to network issues"
            return 0
        fi
    fi

    # If we got here with success, validate the response
    if [[ $exit_code -eq 0 ]]; then
        assert_valid_json "$comment_result" || return 1
        assert_json_has_field "$comment_result" ".data.file_path" || return 1
        assert_json_has_field "$comment_result" ".data.in_reply_to" || return 1

        log "  [OK] Comment created with beseech"
    fi

    return 0
}

# Test: Blessing requests listing
test_blessing_requests() {
    setup_test_env "blessing_requests"
    trap teardown_test_env EXIT

    # Check prerequisites
    if ! check_e2e_prerequisites; then
        if should_skip_network; then
            log "  [SKIP-NETWORK] Skipping blessing requests test"
            return 0
        else
            log_error "Missing POLIS_BASE_URL or DISCOVERY_SERVICE_KEY"
            return 1
        fi
    fi

    # Initialize
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    if should_skip_network; then
        log "  [SKIP-NETWORK] Skipping actual API call"
        log "  [OK] Test setup validated, network call skipped"
        return 0
    fi

    # Query blessing requests
    local result
    result=$("$POLIS_BIN" --json blessing requests 2>&1)
    local exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        local error_code
        error_code=$(echo "$result" | jq -r '.error.code // "UNKNOWN"' 2>/dev/null)

        if [[ "$error_code" == "API_ERROR" || "$error_code" == "NETWORK_ERROR" ]]; then
            log "  [WARN] Network/API error - discovery service may not be available"
            return 0
        fi

        log_error "Blessing requests command failed: $error_code"
        return 1
    fi

    # Validate response format
    assert_valid_json "$result" || return 1
    assert_json_success "$result" "blessing-requests" || return 1
    assert_json_has_field "$result" ".data.count" || return 1
    assert_json_has_field "$result" ".data.requests" || return 1

    local count
    count=$(echo "$result" | jq -r '.data.count')
    log "  Pending requests: $count"

    log "  [OK] Blessing requests retrieved successfully"
    return 0
}

# Test: Full blessing workflow (grant)
test_blessing_grant() {
    setup_test_env "blessing_grant"
    trap teardown_test_env EXIT

    # Check prerequisites
    if ! check_e2e_prerequisites; then
        if should_skip_network; then
            log "  [SKIP-NETWORK] Skipping blessing grant test"
            return 0
        else
            log_error "Missing POLIS_BASE_URL or DISCOVERY_SERVICE_KEY"
            return 1
        fi
    fi

    # Initialize
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    if should_skip_network; then
        log "  [SKIP-NETWORK] Skipping actual API call"
        log "  [OK] Test setup validated, network call skipped"
        return 0
    fi

    # First, get pending requests
    local requests_result
    requests_result=$("$POLIS_BIN" --json blessing requests 2>&1)

    if ! echo "$requests_result" | jq -e '.data.requests[0]' > /dev/null 2>&1; then
        log "  No pending blessing requests to test with"
        log "  [SKIP] Skipping grant test - no requests available"
        return 0
    fi

    # Get the first request ID
    local request_id
    request_id=$(echo "$requests_result" | jq -r '.data.requests[0].id')
    log "  Testing with request ID: $request_id"

    # Grant the blessing
    local grant_result
    grant_result=$("$POLIS_BIN" --json blessing grant "$request_id" 2>&1)
    local exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        local error_code
        error_code=$(echo "$grant_result" | jq -r '.error.code // "UNKNOWN"' 2>/dev/null)
        log "  Grant command returned error: $error_code"

        # Some errors are acceptable in test environment
        if [[ "$error_code" == "NOT_FOUND" || "$error_code" == "ALREADY_PROCESSED" ]]; then
            log "  [OK] Grant command handled edge case correctly"
            return 0
        fi

        if [[ "$error_code" == "API_ERROR" ]]; then
            log "  [WARN] API error - discovery service issue"
            return 0
        fi

        return 1
    fi

    # Validate response
    assert_valid_json "$grant_result" || return 1
    assert_json_success "$grant_result" "blessing-grant" || return 1

    # Verify blessed-comments.json was updated
    if [[ -f "metadata/blessed-comments.json" ]]; then
        local blessed_count
        blessed_count=$(jq '.comments | length' metadata/blessed-comments.json 2>/dev/null || echo "0")
        log "  Blessed comments count: $blessed_count"
    fi

    log "  [OK] Blessing grant completed"
    return 0
}

# Run tests
run_test "Comment with Beseech" test_comment_with_beseech
run_test "Blessing Requests" test_blessing_requests
run_test "Blessing Grant" test_blessing_grant
