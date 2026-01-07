# Polis CLI Test Suite

Automated testing framework for the Polis CLI using JSON mode output.

## Quick Start

```bash
# Run all tests
./tests/run_tests.sh

# Run specific category
./tests/run_tests.sh --category unit
./tests/run_tests.sh --category integration
./tests/run_tests.sh --category e2e

# Skip network calls (assume success)
./tests/run_tests.sh --skip-network

# JSON output for automation
./tests/run_tests.sh --json

# Prompt to push after tests pass
./tests/run_tests.sh --push
```

## Test Categories

### Unit Tests (`--category unit`)
Test individual commands in isolation:
- `test_init.sh` - polis init command
- `test_publish.sh` - polis publish command
- `test_republish.sh` - polis republish command
- `test_version.sh` - polis version command

### Integration Tests (`--category integration`)
Test command sequences and workflows:
- `test_publish_workflow.sh` - Full init → publish → republish cycle

### E2E Tests (`--category e2e`)
End-to-end tests with real API calls:
- `test_blessing_workflow.sh` - Comment and blessing workflow

**Prerequisites for E2E tests:**
- `POLIS_BASE_URL` environment variable set
- `DISCOVERY_SERVICE_KEY` environment variable set
- Discovery service deployed and reachable

## Options

| Option | Description |
|--------|-------------|
| `--json` | Output results in JSON format |
| `--category TYPE` | Run only tests of TYPE (unit, integration, e2e, all) |
| `--skip-network` | Skip actual API calls, assume success |
| `--push` | Prompt to push changes after successful tests |
| `--help` | Show help message |

## Directory Structure

```
tests/
├── run_tests.sh              # Main test runner
├── lib/
│   ├── test_framework.sh     # Core utilities
│   ├── assertions.sh         # JSON/file assertions
│   └── fixtures.sh           # Test data generators
├── unit/                     # Unit tests
├── integration/              # Integration tests
└── e2e/                      # End-to-end tests
```

## Writing New Tests

1. Create a test file in the appropriate directory (unit/, integration/, e2e/)
2. Source the test libraries
3. Define test functions
4. Call `run_test` for each test

Example:

```bash
#!/bin/bash
# test_my_feature.sh

test_my_feature() {
    setup_test_env "my_feature"
    trap teardown_test_env EXIT

    # Test logic
    "$POLIS_BIN" --json init > /dev/null 2>&1 || return 1

    local result
    result=$("$POLIS_BIN" --json my-command 2>&1)

    assert_exit_code 0 $? || return 1
    assert_json_success "$result" "my-command" || return 1

    return 0
}

run_test "My Feature" test_my_feature
```

## Test Isolation

Each test runs in an isolated temporary directory with its own git repository:
- Directories are created in `/tmp/polis-test-*`
- Cleaned up automatically after each test
- No interference between tests

## Git Workflow

Tests use git staging but never push automatically:
- Files are staged after polis commands
- Use `--push` to prompt for manual push after tests pass
- User must confirm before any push occurs
