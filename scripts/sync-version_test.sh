#!/usr/bin/env bash
# sync-version_test.sh - Tests for sync-version.sh script
#
# Usage: ./scripts/sync-version_test.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_SCRIPT="$SCRIPT_DIR/sync-version.sh"

# Create temporary test directory
TEST_DIR=$(mktemp -d)
trap "rm -rf '$TEST_DIR'" EXIT

echo "=== Version Sync Script Tests ==="
echo "Test directory: $TEST_DIR"
echo ""

# Helper functions
run_test() {
    local test_name="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Test $TESTS_RUN: $test_name... "
}

pass_test() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}PASS${NC}"
}

fail_test() {
    local reason="$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}FAIL${NC}"
    echo "  Reason: $reason"
}

setup_test_repo() {
    # Create minimal test repository structure
    mkdir -p "$TEST_DIR/.claude-plugin"

    # Create VERSION file
    echo "1.2.3" > "$TEST_DIR/VERSION"

    # Create plugin.json
    cat > "$TEST_DIR/.claude-plugin/plugin.json" <<'EOF'
{
  "name": "contextd",
  "version": "0.0.0",
  "description": "Test plugin"
}
EOF

    # Initialize git repo (needed for tag checks)
    cd "$TEST_DIR"
    git init -q
    git config user.email "test@example.com"
    git config user.name "Test User"
}

# Test 1: Script exists and is executable
run_test "Script exists and is executable"
if [[ -x "$SYNC_SCRIPT" ]]; then
    pass_test
else
    fail_test "Script not found or not executable: $SYNC_SCRIPT"
fi

# Test 2: Valid version format syncs correctly
run_test "Valid semantic version syncs to plugin.json"
setup_test_repo
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    SYNCED_VERSION=$(jq -r '.version' .claude-plugin/plugin.json 2>/dev/null || grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)
    if [[ "$SYNCED_VERSION" == "1.2.3" ]]; then
        pass_test
    else
        fail_test "Expected version 1.2.3, got $SYNCED_VERSION"
    fi
else
    fail_test "Script failed to execute"
fi

# Test 3: Invalid version format is rejected
run_test "Invalid version format is rejected"
setup_test_repo
echo "invalid.version.format.extra" > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    fail_test "Script should have rejected invalid version format"
else
    pass_test
fi

# Test 4: Empty VERSION file is rejected
run_test "Empty VERSION file is rejected"
setup_test_repo
echo "" > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    fail_test "Script should have rejected empty VERSION file"
else
    pass_test
fi

# Test 5: Missing VERSION file is handled
run_test "Missing VERSION file is handled gracefully"
setup_test_repo
rm "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    fail_test "Script should have failed with missing VERSION file"
else
    pass_test
fi

# Test 6: Pre-release versions are accepted
run_test "Pre-release version (rc) is accepted"
setup_test_repo
echo "1.0.0-rc.1" > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    SYNCED_VERSION=$(jq -r '.version' .claude-plugin/plugin.json 2>/dev/null || grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)
    if [[ "$SYNCED_VERSION" == "1.0.0-rc.1" ]]; then
        pass_test
    else
        fail_test "Expected version 1.0.0-rc.1, got $SYNCED_VERSION"
    fi
else
    fail_test "Script failed to accept pre-release version"
fi

# Test 7: Build metadata is accepted
run_test "Build metadata version is accepted"
setup_test_repo
echo "1.0.0+build.123" > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    SYNCED_VERSION=$(jq -r '.version' .claude-plugin/plugin.json 2>/dev/null || grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)
    if [[ "$SYNCED_VERSION" == "1.0.0+build.123" ]]; then
        pass_test
    else
        fail_test "Expected version 1.0.0+build.123, got $SYNCED_VERSION"
    fi
else
    fail_test "Script failed to accept build metadata version"
fi

# Test 8: Missing plugin.json is handled gracefully
run_test "Missing plugin.json is handled gracefully"
setup_test_repo
rm "$TEST_DIR/.claude-plugin/plugin.json"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    pass_test  # Should still succeed, just skip that file
else
    fail_test "Script should handle missing plugin.json gracefully"
fi

# Test 9: Whitespace in VERSION file is trimmed
run_test "Whitespace in VERSION file is trimmed"
setup_test_repo
echo "  1.2.3  " > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    SYNCED_VERSION=$(jq -r '.version' .claude-plugin/plugin.json 2>/dev/null || grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)
    if [[ "$SYNCED_VERSION" == "1.2.3" ]]; then
        pass_test
    else
        fail_test "Expected version 1.2.3, got $SYNCED_VERSION"
    fi
else
    fail_test "Script failed to handle whitespace"
fi

# Test 10: Complex pre-release + build metadata
run_test "Complex version with pre-release and build metadata"
setup_test_repo
echo "2.0.0-beta.1+exp.sha.5114f85" > "$TEST_DIR/VERSION"
cd "$TEST_DIR"
if bash "$SYNC_SCRIPT" > /dev/null 2>&1; then
    SYNCED_VERSION=$(jq -r '.version' .claude-plugin/plugin.json 2>/dev/null || grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)
    if [[ "$SYNCED_VERSION" == "2.0.0-beta.1+exp.sha.5114f85" ]]; then
        pass_test
    else
        fail_test "Expected version 2.0.0-beta.1+exp.sha.5114f85, got $SYNCED_VERSION"
    fi
else
    fail_test "Script failed to accept complex version"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "Total tests: $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
