#!/usr/bin/env bash
# Integration test for sync-version.sh to verify it works in container/install scenarios

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Sync Version Script Installation Tests ==="
echo "Testing scenarios that occur during fresh installs/deployments"
echo ""

PASS_COUNT=0
FAIL_COUNT=0

pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo -e "${RED}FAIL${NC}: $1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

# Test 1: Fresh git clone scenario (simulates CI/CD)
test_fresh_clone() {
    echo "Test 1: Fresh git clone scenario..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    cd "$TEST_DIR"
    git clone "$REPO_ROOT" test-repo 2>&1 > /dev/null
    cd test-repo

    # Run sync script from repo root
    if ./scripts/sync-version.sh > /dev/null 2>&1; then
        # Verify it synced the actual repo files
        CURRENT_VERSION=$(cat VERSION | tr -d '[:space:]')
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$CURRENT_VERSION" == "$PLUGIN_VERSION" ]]; then
            pass "Fresh clone: script syncs correctly from repo root"
        else
            fail "Fresh clone: version mismatch (VERSION=$CURRENT_VERSION, plugin=$PLUGIN_VERSION)"
        fi
    else
        fail "Fresh clone: script failed to execute"
    fi
}

# Test 2: Running from scripts directory
test_from_scripts_dir() {
    echo "Test 2: Running from scripts directory..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    cd "$TEST_DIR"
    git clone "$REPO_ROOT" test-repo 2>&1 > /dev/null
    cd test-repo/scripts

    # Run from scripts dir using ./sync-version.sh
    if ./sync-version.sh > /dev/null 2>&1; then
        CURRENT_VERSION=$(cat ../VERSION | tr -d '[:space:]')
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' ../.claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$CURRENT_VERSION" == "$PLUGIN_VERSION" ]]; then
            pass "Scripts dir: script works when run from scripts/"
        else
            fail "Scripts dir: version mismatch"
        fi
    else
        fail "Scripts dir: script failed to execute"
    fi
}

# Test 3: Running with bash scripts/sync-version.sh from repo root
test_explicit_bash() {
    echo "Test 3: Explicit bash invocation..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    cd "$TEST_DIR"
    git clone "$REPO_ROOT" test-repo 2>&1 > /dev/null
    cd test-repo

    if bash scripts/sync-version.sh > /dev/null 2>&1; then
        CURRENT_VERSION=$(cat VERSION | tr -d '[:space:]')
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$CURRENT_VERSION" == "$PLUGIN_VERSION" ]]; then
            pass "Explicit bash: works with 'bash scripts/sync-version.sh'"
        else
            fail "Explicit bash: version mismatch"
        fi
    else
        fail "Explicit bash: script failed"
    fi
}

# Test 4: Non-git directory (copied files without .git)
test_non_git_directory() {
    echo "Test 4: Non-git directory (extracted tarball scenario)..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    # Copy repo without .git (simulates tar.gz extraction)
    cp -r "$REPO_ROOT" "$TEST_DIR/contextd"
    rm -rf "$TEST_DIR/contextd/.git"

    cd "$TEST_DIR/contextd"

    # Change version to test sync
    echo "9.9.9" > VERSION

    if ./scripts/sync-version.sh > /dev/null 2>&1; then
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$PLUGIN_VERSION" == "9.9.9" ]]; then
            pass "Non-git: script works in non-git directory"
        else
            fail "Non-git: version not synced (got $PLUGIN_VERSION, expected 9.9.9)"
        fi
    else
        fail "Non-git: script failed"
    fi
}

# Test 5: Subdirectory of git repo
test_git_subdirectory() {
    echo "Test 5: Git subdirectory scenario..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    cd "$TEST_DIR"
    git clone "$REPO_ROOT" test-repo 2>&1 > /dev/null
    cd test-repo
    mkdir -p subdir
    cd subdir

    # Run from subdirectory - should use git root
    if bash ../scripts/sync-version.sh > /dev/null 2>&1; then
        CURRENT_VERSION=$(cat ../VERSION | tr -d '[:space:]')
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' ../.claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$CURRENT_VERSION" == "$PLUGIN_VERSION" ]]; then
            pass "Subdirectory: finds git root correctly"
        else
            fail "Subdirectory: version mismatch"
        fi
    else
        fail "Subdirectory: script failed"
    fi
}

# Test 6: Verify jq fallback to sed works
test_sed_fallback() {
    echo "Test 6: sed fallback (no jq available)..."

    TEST_DIR=$(mktemp -d)
    trap "rm -rf '$TEST_DIR'" RETURN

    cd "$TEST_DIR"
    git clone "$REPO_ROOT" test-repo 2>&1 > /dev/null
    cd test-repo

    # Run with jq disabled (use sed fallback)
    if PATH="/usr/bin:/bin" ./scripts/sync-version.sh > /dev/null 2>&1; then
        CURRENT_VERSION=$(cat VERSION | tr -d '[:space:]')
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' .claude-plugin/plugin.json | cut -d'"' -f4)

        if [[ "$CURRENT_VERSION" == "$PLUGIN_VERSION" ]]; then
            pass "sed fallback: works without jq"
        else
            fail "sed fallback: version mismatch"
        fi
    else
        fail "sed fallback: script failed"
    fi
}

# Test 7: Permission check (executable bit)
test_executable_permission() {
    echo "Test 7: Executable permission..."

    if [[ -x "$REPO_ROOT/scripts/sync-version.sh" ]]; then
        pass "Permissions: script is executable"
    else
        fail "Permissions: script is not executable"
    fi
}

# Test 8: Shebang line check
test_shebang() {
    echo "Test 8: Shebang line..."

    FIRST_LINE=$(head -n 1 "$REPO_ROOT/scripts/sync-version.sh")
    if [[ "$FIRST_LINE" == "#!/usr/bin/env bash" ]]; then
        pass "Shebang: correct (#!/usr/bin/env bash)"
    else
        fail "Shebang: incorrect or missing"
    fi
}

# Run all tests
test_executable_permission
test_shebang
test_fresh_clone
test_from_scripts_dir
test_explicit_bash
test_non_git_directory
test_git_subdirectory
test_sed_fallback

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "Total tests: $((PASS_COUNT + FAIL_COUNT))"
echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
if [[ $FAIL_COUNT -gt 0 ]]; then
    echo -e "${RED}Failed: $FAIL_COUNT${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
