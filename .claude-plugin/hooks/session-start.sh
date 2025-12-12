#!/usr/bin/env bash
# SessionStart hook for contextd plugin
# Injects checkpoint resume protocol into session context

set -euo pipefail

# Get project path from environment or current directory
PROJECT_PATH="${CLAUDE_PROJECT_PATH:-$(pwd)}"
TENANT_ID="${CLAUDE_TENANT_ID:-$(git config user.name 2>/dev/null || echo 'default')}"

# Escape for JSON
escape_json() {
    echo "$1" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}'
}

# Build the context injection
read -r -d '' CONTEXT << 'EOF' || true
## Session Start Protocol (CONTEXTD)

**MANDATORY: Before your first substantive response, you MUST:**

1. Run `mcp__contextd__checkpoint_list(tenant_id, project_path)` to check for existing checkpoints
2. If checkpoints exist, present the most recent relevant checkpoint to the user:
   - Show checkpoint name, summary, and when it was created
   - Ask: "Would you like to resume from this checkpoint?"
3. If user says yes, run `mcp__contextd__checkpoint_resume(checkpoint_id, tenant_id, level)` with level="context"
4. Run `mcp__contextd__memory_search(project_id, "current task context")` to retrieve relevant memories

**This protocol ensures continuity across sessions. Do not skip these steps.**
EOF

CONTEXT_ESCAPED=$(escape_json "$CONTEXT")

# Output JSON for hook system
cat <<JSONEOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "${CONTEXT_ESCAPED}"
  }
}
JSONEOF

exit 0
