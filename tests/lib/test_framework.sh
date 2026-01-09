#!/bin/bash
# test_framework.sh - Core test framework utilities for Polis CLI tests
#
# Tests run in the current working directory using the existing git repo.
# Test artifacts are created in test-data/, committed, then git rm'd at cleanup.
#
# Provides:
#   - Test environment setup/teardown (test-data/ subdirectory)
#   - Git-tracked artifact management
#   - Test execution and tracking
#   - Output formatting (human-readable or JSON)

# Test state
TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_RESULTS=()

# Configuration (set by run_tests.sh or environment)
: "${JSON_OUTPUT:=false}"
: "${SKIP_NETWORK:=false}"
: "${AUTO_PUSH:=false}"
TEST_DATA_DIR="test-data"

# Color codes (disabled in JSON mode)
if [[ "$JSON_OUTPUT" != "true" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Check if test-data directory exists
has_test_data() {
    [[ -d "$TEST_DATA_DIR" ]]
}

# Emergency cleanup for failed tests or manual recovery
emergency_cleanup() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo "  Running emergency cleanup..."
    fi

    if [[ -d "$TEST_DATA_DIR" ]]; then
        # Try git rm first
        git rm -rf --quiet "$TEST_DATA_DIR" 2>/dev/null || true

        # Force remove if still exists
        rm -rf "$TEST_DATA_DIR" 2>/dev/null || true

        # Commit cleanup if there are staged changes
        if ! git diff --cached --quiet 2>/dev/null; then
            git commit -m "test-emergency-cleanup: $(date +%Y%m%d-%H%M%S)" --quiet 2>/dev/null || true
        fi
    fi

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo "  Cleanup complete"
    fi
}

# Initialize test run (called once at start of test suite)
init_test_run() {
    # Check for leftover test data from previous failed run
    if has_test_data; then
        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo "[WARN] Found orphaned test-data/ directory. Cleaning up..."
        fi
        emergency_cleanup
    fi
}

# Initialize test environment
# Creates test-data/ subdirectory with posts, comments, metadata
setup_test_env() {
    local test_name="$1"

    # Verify we're in a git repo
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository. Tests require an existing git repo."
        exit 1
    fi

    # Create test-data directory structure
    mkdir -p "$TEST_DATA_DIR/posts"
    mkdir -p "$TEST_DATA_DIR/comments"
    mkdir -p "$TEST_DATA_DIR/metadata"

    # Set environment variables for fixtures
    export POSTS_DIR="$TEST_DATA_DIR/posts"
    export COMMENTS_DIR="$TEST_DATA_DIR/comments"
    export METADATA_DIR="$TEST_DATA_DIR/metadata"

    # Initialize test metadata files
    echo '{"comments":[],"last_updated":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' > "$TEST_DATA_DIR/metadata/blessed-comments.json"
    echo '{"following":[]}' > "$TEST_DATA_DIR/metadata/following.json"
    touch "$TEST_DATA_DIR/metadata/public.jsonl"

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo "  [SETUP] Test directory: $TEST_DATA_DIR (test: $test_name)"
    fi
}

# Clean up test environment
# Commits test artifacts, then removes them via git rm
teardown_test_env() {
    if [[ -d "$TEST_DATA_DIR" ]]; then
        # Stage test artifacts
        git add "$TEST_DATA_DIR" 2>/dev/null || true

        # Commit test artifacts if anything was staged
        if ! git diff --cached --quiet 2>/dev/null; then
            git commit -m "test-artifacts: $(date +%Y%m%d-%H%M%S)" --quiet 2>/dev/null || true
        fi

        # Remove test-data via git rm
        git rm -rf --quiet "$TEST_DATA_DIR" 2>/dev/null || rm -rf "$TEST_DATA_DIR"

        # Commit the removal
        if ! git diff --cached --quiet 2>/dev/null; then
            git commit -m "test-cleanup: removed test-data" --quiet 2>/dev/null || true
        fi

        # Auto-push if enabled
        if [[ "$AUTO_PUSH" == "true" ]]; then
            if git remote get-url origin &>/dev/null; then
                git push --quiet 2>/dev/null || {
                    if [[ "$JSON_OUTPUT" != "true" ]]; then
                        echo "  [WARN] Auto-push failed. Run 'git push' manually."
                    fi
                }
            fi
        fi

        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo "  [TEARDOWN] Cleaned up: $TEST_DATA_DIR"
        fi
    fi

    # Clear environment variables
    unset POSTS_DIR COMMENTS_DIR METADATA_DIR
}

# Run a single test
# Usage: run_test "Test Name" test_function
run_test() {
    local test_name="$1"
    local test_func="$2"
    local start_time end_time duration

    TEST_COUNT=$((TEST_COUNT + 1))
    start_time=$(date +%s%N)

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo ""
        echo -e "${BLUE}=== TEST: $test_name ===${NC}"
    fi

    # Run the test function and capture result
    local result=0
    if $test_func; then
        result=0
    else
        result=1
    fi

    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))  # milliseconds

    if [[ $result -eq 0 ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo -e "${GREEN}[PASS]${NC} $test_name (${duration}ms)"
        fi
        TEST_RESULTS+=("{\"name\":\"$test_name\",\"status\":\"pass\",\"duration_ms\":$duration}")
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo -e "${RED}[FAIL]${NC} $test_name (${duration}ms)"
        fi
        TEST_RESULTS+=("{\"name\":\"$test_name\",\"status\":\"fail\",\"duration_ms\":$duration}")
    fi

    return $result
}

# Skip a test (for conditional skipping)
skip_test() {
    local test_name="$1"
    local reason="$2"

    TEST_COUNT=$((TEST_COUNT + 1))
    SKIP_COUNT=$((SKIP_COUNT + 1))

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo ""
        echo -e "${YELLOW}[SKIP]${NC} $test_name: $reason"
    fi
    TEST_RESULTS+=("{\"name\":\"$test_name\",\"status\":\"skip\",\"reason\":\"$reason\"}")
}

# Print test summary
print_summary() {
    if [[ "$JSON_OUTPUT" == "true" ]]; then
        # JSON output
        local results_json
        results_json=$(printf '%s\n' "${TEST_RESULTS[@]}" | paste -sd ',' -)
        cat <<EOF
{
  "summary": {
    "total": $TEST_COUNT,
    "passed": $PASS_COUNT,
    "failed": $FAIL_COUNT,
    "skipped": $SKIP_COUNT
  },
  "results": [$results_json],
  "success": $([ $FAIL_COUNT -eq 0 ] && echo "true" || echo "false")
}
EOF
    else
        # Human-readable output
        echo ""
        echo "=========================================="
        echo "TEST SUMMARY"
        echo "=========================================="
        echo "Total:   $TEST_COUNT"
        echo -e "Passed:  ${GREEN}$PASS_COUNT${NC}"
        echo -e "Failed:  ${RED}$FAIL_COUNT${NC}"
        echo -e "Skipped: ${YELLOW}$SKIP_COUNT${NC}"
        echo ""

        if [[ $FAIL_COUNT -eq 0 ]]; then
            echo -e "${GREEN}All tests passed!${NC}"
        else
            echo -e "${RED}Some tests failed!${NC}"
        fi
    fi

    [[ $FAIL_COUNT -eq 0 ]]
}

# Check if network tests should be skipped
should_skip_network() {
    [[ "$SKIP_NETWORK" == "true" ]]
}

# Log message (only in human mode)
log() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo "  $*"
    fi
}

# Log error (only in human mode)
log_error() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo -e "  ${RED}ERROR:${NC} $*" >&2
    fi
}
