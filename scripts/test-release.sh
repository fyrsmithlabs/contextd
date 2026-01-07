#!/bin/bash
#
# Comprehensive Release Testing Script
# Tests all scenarios from brand new user to veteran usage
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
TOTAL=0

# Test results array
declare -a TEST_RESULTS

log_test() {
    ((TOTAL++))
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        ((PASSED++))
        TEST_RESULTS+=("PASS: $2")
    else
        echo -e "${RED}✗${NC} $2"
        ((FAILED++))
        TEST_RESULTS+=("FAIL: $2")
        if [ -n "$3" ]; then
            echo -e "  ${RED}Error: $3${NC}"
        fi
    fi
}

section() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo ""
}

subsection() {
    echo ""
    echo -e "${YELLOW}▶ $1${NC}"
    echo ""
}

section "BUILD VALIDATION"
subsection "Building binaries"

CGO_ENABLED=1 go build -o ./contextd-test ./cmd/contextd 2>/dev/null && log_test 0 "contextd builds" || log_test 1 "contextd builds"
go build -o ./ctxd-test ./cmd/ctxd 2>/dev/null && log_test 0 "ctxd builds" || log_test 1 "ctxd builds"
chmod +x ./contextd-test ./ctxd-test 2>/dev/null || true

section "SCENARIO 1: BRAND NEW USER"
subsection "Plugin validation"

[ -f ".claude-plugin/plugin.json" ] && log_test 0 "plugin.json exists" || log_test 1 "plugin.json exists"
python3 -c "import json; json.load(open('.claude-plugin/plugin.json'))" 2>/dev/null && log_test 0 "plugin.json valid JSON" || log_test 1 "plugin.json valid JSON"

AGENT_COUNT=$(python3 -c "import json; print(len(json.load(open('.claude-plugin/plugin.json'))['agents']))" 2>/dev/null)
[ "$AGENT_COUNT" = "5" ] && log_test 0 "5 agents registered" || log_test 1 "5 agents registered"

section "SCENARIO 2: MCP COMMANDS"
./ctxd-test mcp --help 2>&1 | grep -q "install" && log_test 0 "mcp has install" || log_test 1 "mcp has install"
./ctxd-test mcp --help 2>&1 | grep -q "status" && log_test 0 "mcp has status" || log_test 1 "mcp has status"
./ctxd-test mcp --help 2>&1 | grep -q "uninstall" && log_test 0 "mcp has uninstall" || log_test 1 "mcp has uninstall"

section "SCENARIO 3: STATUSLINE"
./ctxd-test statusline --help 2>&1 | grep -q "install" && log_test 0 "statusline has install" || log_test 1 "statusline has install"
grep -q '"type".*"command"' cmd/ctxd/statusline.go && log_test 0 "statusline uses correct format" || log_test 1 "statusline format"

section "SCENARIO 4: DOCUMENTATION"
for doc in README.md QUICKSTART.md ONBOARDING.md; do
    [ -f "$doc" ] && log_test 0 "$doc exists" || log_test 1 "$doc exists"
done
grep -qi "ctxd mcp install" README.md && log_test 0 "README documents automation" || log_test 1 "README automation"

section "SCENARIO 5: SKILLS"
grep -qi "MANDATORY.*semantic_search" CLAUDE.md && log_test 0 "CLAUDE.md has enforcement" || log_test 1 "CLAUDE.md enforcement"

section "SCENARIO 6: AGENTS"
for agent in .claude-plugin/agents/*.md; do
    grep -q "^name:" "$agent" && log_test 0 "$(basename $agent) has name" || log_test 1 "$(basename $agent) name"
done

section "TEST SUMMARY"
echo ""
echo "Results: Passed=$PASSED Failed=$FAILED Total=$TOTAL"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    exit 0
else
    echo -e "${RED}✗ TESTS FAILED${NC}"
    exit 1
fi
