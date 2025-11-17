# Start Task

**Command**: `/start-task <issue-number> [--tdd]`

**Description**: Initialize development environment for a GitHub issue with automatic TDD setup.

**Usage**:
```
/start-task 45              # Start work on issue #45
/start-task 45 --tdd        # Start with TDD template (default)
/start-task 45 --no-branch  # Don't create branch (work on current)
```

## Purpose

Streamlines the most common workflow start by:
- Fetching issue details from GitHub
- Creating feature branch with proper naming
- Updating issue status to "In Progress"
- Creating test file template if `--tdd` flag used (default)
- Assigning issue to bot account
- Adding initial progress comment

## ⚠️ CRITICAL: Go Code Delegation

**After running this command, ALL Go implementation work MUST be delegated to golang-pro:**

```
Use the golang-pro skill to implement [issue description/requirements]
```

**NEVER write Go code directly.** The golang-pro agent:
- Enforces TDD (writes tests first)
- Ensures ≥70% test coverage
- Validates all tests pass
- Updates CHANGELOG.md
- Creates proper conventional commits

## Agent Workflow

When this command is invoked, execute the following workflow:

```bash
# 1. Fetch issue details
ISSUE_NUMBER=$1
ISSUE_DATA=$(gh issue view $ISSUE_NUMBER --json title,body,labels)

# 2. Create branch
BRANCH_NAME=$(echo "feature/issue-${ISSUE_NUMBER}" | tr '[:upper:]' '[:lower:]')
git checkout -b $BRANCH_NAME

# 3. Create test template if --tdd (default)
if [[ "$2" != "--no-tdd" ]]; then
    # Determine package from issue labels or title
    # Create test file template
fi

# 4. Update issue status
gh issue comment $ISSUE_NUMBER --body "🤖 Started working on this issue"
gh issue edit $ISSUE_NUMBER --add-assignee @me

# 5. Output summary
echo "✅ Task started: Issue #$ISSUE_NUMBER"
echo "Branch: $BRANCH_NAME"
echo "Test template: [path if created]"
```

## Implementation Script

Location: `.scripts/start-task.sh`

```bash
#!/bin/bash
# Script: start-task.sh
# Purpose: Initialize development environment for GitHub issue
# Usage: ./start-task.sh <issue-number> [--tdd|--no-tdd] [--no-branch]

set -e

ISSUE_NUMBER=$1
TDD_MODE="yes"
CREATE_BRANCH="yes"

if [[ -z "$ISSUE_NUMBER" ]]; then
    echo "❌ Usage: ./start-task.sh <issue-number> [--tdd|--no-tdd] [--no-branch]"
    exit 1
fi

# Parse flags
shift
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-tdd)
            TDD_MODE="no"
            shift
            ;;
        --tdd)
            TDD_MODE="yes"
            shift
            ;;
        --no-branch)
            CREATE_BRANCH="no"
            shift
            ;;
        *)
            echo "❌ Unknown flag: $1"
            exit 1
            ;;
    esac
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Starting Task: Issue #$ISSUE_NUMBER"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Fetch issue details
echo "📋 Fetching issue details..."
if ! gh issue view $ISSUE_NUMBER &> /dev/null; then
    echo "❌ Issue #$ISSUE_NUMBER not found"
    exit 1
fi

ISSUE_TITLE=$(gh issue view $ISSUE_NUMBER --json title -q .title)
ISSUE_LABELS=$(gh issue view $ISSUE_NUMBER --json labels -q '.labels[].name' | tr '\n' ',' | sed 's/,$//')

echo "Title: $ISSUE_TITLE"
echo "Labels: $ISSUE_LABELS"
echo ""

# Create feature branch
if [[ "$CREATE_BRANCH" == "yes" ]]; then
    # Create branch name from issue number
    BRANCH_NAME="feature/issue-${ISSUE_NUMBER}"

    echo "🌿 Creating branch: $BRANCH_NAME"

    # Check if branch exists
    if git show-ref --verify --quiet refs/heads/$BRANCH_NAME; then
        echo "⚠️  Branch already exists, checking out..."
        git checkout $BRANCH_NAME
    else
        git checkout -b $BRANCH_NAME
        echo "✅ Created and checked out branch: $BRANCH_NAME"
    fi
    echo ""
fi

# Create test template if TDD mode
if [[ "$TDD_MODE" == "yes" ]]; then
    echo "🧪 TDD Mode: Creating test template..."

    # Try to determine package from labels or title
    PACKAGE_NAME=""

    # Check for package label (e.g., "pkg:auth", "package:cache")
    if echo "$ISSUE_LABELS" | grep -q "pkg:"; then
        PACKAGE_NAME=$(echo "$ISSUE_LABELS" | grep -o "pkg:[^,]*" | cut -d: -f2)
    elif echo "$ISSUE_LABELS" | grep -q "package:"; then
        PACKAGE_NAME=$(echo "$ISSUE_LABELS" | grep -o "package:[^,]*" | cut -d: -f2)
    fi

    # If no package found in labels, ask user or use generic location
    if [[ -z "$PACKAGE_NAME" ]]; then
        echo "⚠️  No package label found (use 'pkg:name' or 'package:name')"
        echo "Using generic test location: tests/"
        PACKAGE_NAME="generic"
        TEST_DIR="tests"
        mkdir -p $TEST_DIR
    else
        TEST_DIR="pkg/$PACKAGE_NAME"
        if [[ ! -d "$TEST_DIR" ]]; then
            echo "Creating package directory: $TEST_DIR"
            mkdir -p $TEST_DIR
        fi
    fi

    # Create test file name from issue title
    SANITIZED_TITLE=$(echo "$ISSUE_TITLE" | tr '[:upper:]' '[:lower:]' | tr ' ' '_' | sed 's/[^a-z0-9_]//g')
    TEST_FILE="${TEST_DIR}/${SANITIZED_TITLE}_test.go"

    # Only create if doesn't exist
    if [[ ! -f "$TEST_FILE" ]]; then
        cat > "$TEST_FILE" <<EOF
package ${PACKAGE_NAME}

import (
	"testing"
)

// TestSuite for Issue #${ISSUE_NUMBER}: ${ISSUE_TITLE}
// TODO: Implement test cases following TDD approach

// Test_TODO_ReplaceWithActualTestName is a template test
// Replace with actual test cases based on acceptance criteria
func Test_TODO_ReplaceWithActualTestName(t *testing.T) {
	t.Skip("TODO: Implement test for issue #${ISSUE_NUMBER}")

	// Arrange
	// TODO: Set up test data and dependencies

	// Act
	// TODO: Execute the function/method being tested

	// Assert
	// TODO: Verify expected outcomes
}

// Add more test cases as needed:
// - Happy path tests
// - Error cases
// - Edge cases
// - Boundary conditions
EOF
        echo "✅ Created test template: $TEST_FILE"
    else
        echo "⚠️  Test file already exists: $TEST_FILE"
    fi
    echo ""
fi

# Update issue status
echo "📝 Updating issue status..."

# Add comment
gh issue comment $ISSUE_NUMBER --body "🤖 Started working on this issue

**Branch**: \`${BRANCH_NAME:-current}\`
**TDD Mode**: ${TDD_MODE}
${TEST_FILE:+**Test Template**: \`$TEST_FILE\`}

Development workflow:
1. Write failing tests (TDD)
2. Implement code to pass tests
3. Run quality gates: \`/run-quality-gates quick\`
4. Create PR when ready" 2>/dev/null || echo "⚠️  Could not add comment (may need authentication)"

# Try to assign issue (may fail if not authenticated)
gh issue edit $ISSUE_NUMBER --add-assignee @me 2>/dev/null || echo "⚠️  Could not assign issue (may need authentication)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Task Started Successfully!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Issue #${ISSUE_NUMBER}: ${ISSUE_TITLE}"
if [[ "$CREATE_BRANCH" == "yes" ]]; then
    echo "Branch: ${BRANCH_NAME}"
fi
if [[ "$TDD_MODE" == "yes" && -n "$TEST_FILE" ]]; then
    echo "Test Template: ${TEST_FILE}"
fi
echo ""
echo "Next steps:"
echo "  1. Review issue acceptance criteria"
echo "  2. ⚠️  DELEGATE to golang-pro: Use the golang-pro skill to implement..."
if [[ "$TDD_MODE" == "yes" ]]; then
    echo "     (golang-pro will write tests in ${TEST_FILE} first, then implement)"
    echo "  3. Run: /run-quality-gates quick"
else
    echo "     (golang-pro will implement and write tests)"
    echo "  3. Run: /run-quality-gates full"
fi
echo "  4. Create PR: /create-pr ${ISSUE_NUMBER}"
echo ""
echo "⚠️  IMPORTANT: Do NOT write Go code directly - always delegate to golang-pro"
echo ""
