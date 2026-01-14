#!/bin/bash
# PreCompact Hook: Auto-checkpoint before context compaction
# Outputs instructions for Claude to save state before summarization

set -euo pipefail

# Derive project context
PROJECT_PATH=$(pwd)
PROJECT_ID=$(git remote get-url origin 2>/dev/null | sed 's/.*[:/]\([^/]*\/[^/]*\)\.git.*/\1/' | tr '/' '_' || basename "$PROJECT_PATH")
BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
SESSION_ID="${CLAUDE_SESSION_ID:-session-$(date +%Y%m%d-%H%M%S)}"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Get recent git activity for context
RECENT_FILES=$(git diff --name-only HEAD~5 2>/dev/null | head -10 | tr '\n' ', ' || echo "none")
UNCOMMITTED=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')

# Estimate token usage from Claude session files (if available)
CLAUDE_PROJECTS="$HOME/.claude/projects"
TOKEN_ESTIMATE="unknown"
if [ -d "$CLAUDE_PROJECTS" ]; then
  LATEST_SESSION=$(find "$CLAUDE_PROJECTS" -name "*.jsonl" -type f -mmin -60 2>/dev/null | head -1)
  if [ -n "$LATEST_SESSION" ] && command -v jq &>/dev/null; then
    TOKEN_ESTIMATE=$(jq -s '[.[].total_tokens // 0] | add' "$LATEST_SESSION" 2>/dev/null || echo "unknown")
  fi
fi

# Output checkpoint instructions for Claude
cat << EOF
## ⚠️ PreCompact: Context About to Be Summarized

**You MUST save a checkpoint NOW before context is lost.**

### Checkpoint Command
\`\`\`
mcp__contextd__checkpoint_save(
  session_id: "${SESSION_ID}",
  project_path: "${PROJECT_PATH}",
  name: "precompact-${TIMESTAMP}",
  description: "Auto-checkpoint before context compaction",
  summary: "[FILL: 1-2 sentence summary of current task state]",
  context: "[FILL: Key context that should survive compaction]",
  full_state: "[FILL: Complete task state, decisions made, next steps]",
  token_count: ${TOKEN_ESTIMATE:-0},
  threshold: 0.9,
  auto_created: true
)
\`\`\`

### Current Session Context
| Field | Value |
|-------|-------|
| Project | ${PROJECT_ID} |
| Branch | ${BRANCH} |
| Session | ${SESSION_ID} |
| Uncommitted files | ${UNCOMMITTED} |
| Recent changes | ${RECENT_FILES:-none} |
| Est. tokens used | ${TOKEN_ESTIMATE} |

### What to Capture
1. **Current task:** What are you working on?
2. **Progress:** What's done, what's remaining?
3. **Decisions:** Key choices made and why
4. **Blockers:** Any issues encountered
5. **Next steps:** What to do after resuming

**DO NOT skip this checkpoint.** Context compaction will summarize away details you may need.
EOF
