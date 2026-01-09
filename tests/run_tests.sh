#!/bin/bash
# run_tests.sh - Main test runner for Polis CLI
#
# Tests run in the current working directory using the existing git repo.
# Test artifacts are created in test-data/, committed, then git rm'd at cleanup.
#
# Usage:
#   ./tests/run_tests.sh                    # Run all tests
#   ./tests/run_tests.sh --json             # JSON output
#   ./tests/run_tests.sh --category unit    # Run only unit tests
#   ./tests/run_tests.sh --skip-network     # Skip network API calls
#   ./tests/run_tests.sh --auto-push        # Auto-push commits
#   ./tests/run_tests.sh --cleanup          # Cleanup orphaned test data
#
# Categories: unit, integration, e2e, all (default)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(dirname "$CLI_DIR")"

# Export polis binary path (check ./bin/polis first, then ./polis)
if [[ -x "$CLI_DIR/bin/polis" ]]; then
    export POLIS_BIN="$CLI_DIR/bin/polis"
elif [[ -x "$CLI_DIR/polis" ]]; then
    export POLIS_BIN="$CLI_DIR/polis"
else
    export POLIS_BIN="$CLI_DIR/bin/polis"  # Default for error message
fi

# Parse arguments
export JSON_OUTPUT=false
export SKIP_NETWORK=false
export AUTO_PUSH=false
RUN_CLEANUP_ONLY=false
TEST_CATEGORY="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --skip-network)
            SKIP_NETWORK=true
            shift
            ;;
        --auto-push)
            AUTO_PUSH=true
            shift
            ;;
        --cleanup)
            RUN_CLEANUP_ONLY=true
            shift
            ;;
        --category)
            TEST_CATEGORY="$2"
            shift 2
            ;;
        --help|-h)
            echo "Polis CLI Test Runner"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --json            Output results in JSON format"
            echo "  --category TYPE   Run only tests of TYPE (unit, integration, e2e, all)"
            echo "  --skip-network    Skip actual API calls to discovery service"
            echo "  --auto-push       Auto-push commits after test cleanup"
            echo "  --cleanup         Cleanup orphaned test data (from failed tests)"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Run all tests"
            echo "  $0 --skip-network           # Run without API calls"
            echo "  $0 --auto-push              # Run and push to trigger deploy"
            echo "  $0 --cleanup                # Clean up after failed test"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Source test framework first (needed for cleanup)
source "$SCRIPT_DIR/lib/test_framework.sh"

# Handle cleanup-only mode
if [[ "$RUN_CLEANUP_ONLY" == "true" ]]; then
    if [[ "$JSON_OUTPUT" == "true" ]]; then
        if has_test_data; then
            emergency_cleanup
            echo '{"status": "success", "command": "cleanup", "data": {"cleaned": true}}'
        else
            echo '{"status": "success", "command": "cleanup", "data": {"cleaned": false, "message": "No test data found"}}'
        fi
    else
        if has_test_data; then
            emergency_cleanup
            echo "Cleanup complete."
        else
            echo "No test data to clean up."
        fi
    fi
    exit 0
fi

# Source .env file if it exists (for e2e tests)
if [[ -f "$CLI_DIR/.env" ]]; then
    # shellcheck disable=SC1091
    source "$CLI_DIR/.env"
fi

# Verify polis binary exists
if [[ ! -x "$POLIS_BIN" ]]; then
    if [[ "$JSON_OUTPUT" == "true" ]]; then
        echo '{"error": "polis binary not found", "path": "'"$POLIS_BIN"'"}'
    else
        echo "Error: polis binary not found at $POLIS_BIN"
    fi
    exit 1
fi

# Verify we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    if [[ "$JSON_OUTPUT" == "true" ]]; then
        echo '{"error": "Not in a git repository. Tests require an existing git repo."}'
    else
        echo "Error: Not in a git repository. Tests require an existing git repo."
    fi
    exit 1
fi

# Verify dependencies
for cmd in jq git; do
    if ! command -v "$cmd" &> /dev/null; then
        if [[ "$JSON_OUTPUT" == "true" ]]; then
            echo '{"error": "Missing dependency", "command": "'"$cmd"'"}'
        else
            echo "Error: Required dependency '$cmd' not found"
        fi
        exit 1
    fi
done

# Print header (human mode only)
if [[ "$JSON_OUTPUT" != "true" ]]; then
    echo "=============================================="
    echo "Polis CLI Test Suite"
    echo "=============================================="
    echo "Polis Binary: $POLIS_BIN"
    echo "Test Category: $TEST_CATEGORY"
    echo "Skip Network: $SKIP_NETWORK"
    echo "Auto Push: $AUTO_PUSH"
    echo ""
fi

# Source remaining test libraries
source "$SCRIPT_DIR/lib/assertions.sh"
source "$SCRIPT_DIR/lib/fixtures.sh"

# Initialize test run (cleans up orphaned test data)
init_test_run

# Run tests in a category
run_test_category() {
    local category="$1"
    local test_dir="$SCRIPT_DIR/$category"

    if [[ ! -d "$test_dir" ]]; then
        if [[ "$JSON_OUTPUT" != "true" ]]; then
            echo "No tests found for category: $category"
        fi
        return
    fi

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo ""
        echo ">>> Running $category tests..."
        echo ""
    fi

    for test_file in "$test_dir"/test_*.sh; do
        if [[ -f "$test_file" ]]; then
            if [[ "$JSON_OUTPUT" != "true" ]]; then
                echo "--- $(basename "$test_file") ---"
            fi
            # Source the test file (runs tests defined in it)
            # shellcheck disable=SC1090
            source "$test_file"
        fi
    done
}

# Run tests based on category
case $TEST_CATEGORY in
    unit)
        run_test_category "unit"
        ;;
    integration)
        run_test_category "integration"
        ;;
    e2e)
        run_test_category "e2e"
        ;;
    all)
        run_test_category "unit"
        run_test_category "integration"
        run_test_category "e2e"
        ;;
    *)
        if [[ "$JSON_OUTPUT" == "true" ]]; then
            echo '{"error": "Unknown category", "category": "'"$TEST_CATEGORY"'"}'
        else
            echo "Unknown category: $TEST_CATEGORY"
            echo "Valid options: unit, integration, e2e, all"
        fi
        exit 1
        ;;
esac

# Print summary
print_summary
TEST_RESULT=$?

exit $TEST_RESULT
