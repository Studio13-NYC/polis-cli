#!/bin/bash
# test_framework.sh - Core test framework utilities for Polis CLI tests
#
# Provides:
#   - Test environment setup/teardown (isolated temp directories)
#   - Test execution and tracking
#   - Output formatting (human-readable or JSON)

# Test state
TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_DIR=""
ORIGINAL_DIR=""
TEST_RESULTS=()

# Output mode (set by run_tests.sh)
: "${JSON_OUTPUT:=false}"
: "${SKIP_NETWORK:=false}"

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

# Initialize test environment
# Creates isolated temp directory with git repo
setup_test_env() {
    local test_name="$1"

    ORIGINAL_DIR="$(pwd)"
    TEST_DIR=$(mktemp -d -t "polis-test-${test_name}-XXXXXX")
    cd "$TEST_DIR" || exit 1

    # Initialize git repo for tests (polis uses git for staging)
    git init --quiet
    git config user.email "test@polis-cli.test"
    git config user.name "Polis Test Runner"

    # Ensure no remote is configured (prevent accidental pushes)
    git remote remove origin 2>/dev/null || true

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo "  [SETUP] Test directory: $TEST_DIR"
    fi
}

# Clean up test environment
teardown_test_env() {
    cd "$ORIGINAL_DIR" || cd /
    if [[ -n "$TEST_DIR" && -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR"
        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo "  [TEARDOWN] Cleaned up: $TEST_DIR"
        fi
    fi
    TEST_DIR=""
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
