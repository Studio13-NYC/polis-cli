#!/bin/bash
# run_tests.sh - Main test runner for Polis CLI
#
# Usage:
#   ./tests/run_tests.sh                    # Run all tests, human output
#   ./tests/run_tests.sh --json             # Run all tests, JSON output
#   ./tests/run_tests.sh --category unit    # Run only unit tests
#   ./tests/run_tests.sh --push             # Prompt to push after tests pass
#   ./tests/run_tests.sh --skip-network     # Skip network API calls
#
# Categories: unit, integration, e2e, all (default)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(dirname "$CLI_DIR")"

# Export polis binary path
export POLIS_BIN="$CLI_DIR/bin/polis"

# Parse arguments
export JSON_OUTPUT=false
export SKIP_NETWORK=false
PUSH_AFTER_TEST=false
TEST_CATEGORY="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --push)
            PUSH_AFTER_TEST=true
            shift
            ;;
        --skip-network)
            SKIP_NETWORK=true
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
            echo "  --skip-network    Skip actual API calls, assume success"
            echo "  --push            Prompt to push changes after successful tests"
            echo "  --help            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

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
    echo ""
fi

# Source test libraries
source "$SCRIPT_DIR/lib/test_framework.sh"
source "$SCRIPT_DIR/lib/assertions.sh"
source "$SCRIPT_DIR/lib/fixtures.sh"

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

# Handle push workflow (human mode only, after successful tests)
if [[ "$PUSH_AFTER_TEST" == true && "$JSON_OUTPUT" != "true" ]]; then
    if [[ $TEST_RESULT -eq 0 ]]; then
        echo ""
        echo "=============================================="
        echo "All tests passed!"
        echo "=============================================="

        # Check if we're in a git repo with changes
        cd "$REPO_ROOT"
        if git rev-parse --git-dir > /dev/null 2>&1; then
            # Check for staged or unstaged changes
            if ! git diff --quiet || ! git diff --cached --quiet; then
                echo ""
                echo "The following files have changes:"
                git status --short
                echo ""
                read -p "Do you want to push these changes to remote? (y/N): " -n 1 -r
                echo ""

                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    # Stage test files if modified
                    git add "$SCRIPT_DIR"

                    echo ""
                    echo "Files staged. Please commit and push when ready."
                    echo "Example: git commit -m 'Add/update CLI tests' && git push"
                else
                    echo "Push cancelled. Changes remain locally."
                fi
            else
                echo "No changes to push."
            fi
        fi
    else
        echo ""
        echo "=============================================="
        echo "Tests failed - not pushing."
        echo "=============================================="
        echo "Fix the failing tests before pushing."
    fi
fi

exit $TEST_RESULT
