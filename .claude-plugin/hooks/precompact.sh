#!/bin/bash
# Auto-checkpoint before context compaction
# Called by Claude Code PreCompact hook

set -e

# Derive project ID from git remote
PROJECT_ID=$(git remote get-url origin 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' | tr '/' '_' || echo "unknown")

# Session ID from env or generate
SESSION_ID=${CLAUDE_SESSION_ID:-$(date +%s)}

# Context percentage (passed as argument or default)
PERCENT=${1:-70}

# contextd HTTP endpoint
CONTEXTD_URL=${CONTEXTD_URL:-"http://localhost:9090"}

echo "[contextd] Auto-checkpoint at ${PERCENT}% context for project ${PROJECT_ID}"

# Primary: HTTP call
if curl -sf -X POST "${CONTEXTD_URL}/api/v1/threshold" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"${PROJECT_ID}\",\"session_id\":\"${SESSION_ID}\",\"percent\":${PERCENT}}" \
  --max-time 5; then
    echo "[contextd] Checkpoint created successfully"
    exit 0
fi

# Fallback: Print instruction for Claude to call MCP tool
echo "[contextd] HTTP failed. Call context_threshold tool:"
echo "  project_id: ${PROJECT_ID}"
echo "  session_id: ${SESSION_ID}"
echo "  percent: ${PERCENT}"
exit 0
