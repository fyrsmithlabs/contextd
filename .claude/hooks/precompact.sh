#!/bin/bash
# .claude/hooks/precompact.sh
# Triggers auto-checkpoint before context compaction
# Called by Claude Code PreCompact hook
#
# IMPORTANT: This hook cannot access conversation context directly.
# It instructs Claude to call checkpoint_save with a proper summary.

set -e

# Derive tenant ID from git remote
TENANT_ID=$(git remote get-url origin 2>/dev/null | sed 's/.*github.com[:/]\([^/]*\).*/\1/' | tr '[:upper:]' '[:lower:]' || echo "local")

# Session ID from env or generate
SESSION_ID=${CLAUDE_SESSION_ID:-$(date +%s)}

# Project path
PROJECT_PATH=$(pwd)

# Context percentage (passed as argument or default)
PERCENT=${1:-70}

# Output instruction for Claude to create a meaningful checkpoint
# Claude will see this output and should act on it
cat << EOF
[contextd] Context at ${PERCENT}% - checkpoint required before compaction.

IMPORTANT: Create a checkpoint NOW with a meaningful summary.

Call mcp__contextd__checkpoint_save with:
- session_id: "${SESSION_ID}"
- tenant_id: "${TENANT_ID}"
- project_path: "${PROJECT_PATH}"
- name: "<brief descriptive name of current work>"
- description: "Auto-checkpoint at ${PERCENT}% context"
- summary: "<what you accomplished, what's in progress, what's next>"
- context: "<key decisions, blockers, important context for resumption>"
- full_state: ""
- token_count: 0
- threshold: $(echo "scale=2; ${PERCENT}/100" | bc)
- auto_created: true

The summary should include:
1. What was accomplished this session
2. What's currently in progress
3. What should be done next
4. Any key decisions or blockers

DO NOT use generic text like "Context at 70% threshold" - that's useless for resumption.
EOF

exit 0
