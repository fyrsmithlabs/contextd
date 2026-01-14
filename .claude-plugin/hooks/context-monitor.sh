#!/bin/bash
# Context Monitor: Three-tier context window monitoring
# Tier 1: context_window from statusline JSON (most accurate)
# Tier 2: JSONL transcript parsing + fudge factor (fallback)
# Tier 3: PreCompact hook (safety net - separate file)

set -euo pipefail

# Configuration
CONTEXT_WINDOW_SIZE=${CONTEXT_WINDOW_SIZE:-200000}
USABLE_CONTEXT=$((CONTEXT_WINDOW_SIZE * 80 / 100))  # 160k usable before auto-compact
FUDGE_FACTOR=130  # 30% increase to account for hidden context (system prompt, MCP tools, etc.)

# Thresholds (percentage of usable context)
THRESHOLD_LOW=50
THRESHOLD_MEDIUM=75
THRESHOLD_HIGH=90

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# ============================================================================
# Tier 1: Try to read context_window from statusline JSON (stdin)
# ============================================================================
try_statusline_context() {
    local stdin_data="$1"

    if [ -z "$stdin_data" ]; then
        return 1
    fi

    # Check if context_window exists in the JSON
    local total_input
    total_input=$(echo "$stdin_data" | jq -r '.context_window.total_input_tokens // empty' 2>/dev/null)

    if [ -n "$total_input" ] && [ "$total_input" != "null" ]; then
        local window_size
        window_size=$(echo "$stdin_data" | jq -r '.context_window.context_window_size // 200000' 2>/dev/null)
        echo "$total_input $window_size"
        return 0
    fi

    return 1
}

# ============================================================================
# Tier 2: Parse JSONL transcript files
# ============================================================================
parse_jsonl_transcript() {
    local claude_projects="$HOME/.claude/projects"

    if [ ! -d "$claude_projects" ]; then
        return 1
    fi

    # Find most recent JSONL file modified in last 60 minutes
    local latest_session
    latest_session=$(find "$claude_projects" -name "*.jsonl" -type f -mmin -60 2>/dev/null | \
        xargs -I {} stat --format='%Y %n' {} 2>/dev/null | \
        sort -rn | head -1 | cut -d' ' -f2-)

    if [ -z "$latest_session" ] || [ ! -f "$latest_session" ]; then
        return 1
    fi

    # Find the most recent assistant message with usage data
    # Key insight: Anthropic API returns CUMULATIVE tokens - we only need the last entry
    local usage
    usage=$(tac "$latest_session" 2>/dev/null | while IFS= read -r line; do
        # Look for assistant messages with usage data
        local has_usage
        has_usage=$(echo "$line" | jq -r 'select(.message.usage != null) | .message.usage' 2>/dev/null)
        if [ -n "$has_usage" ] && [ "$has_usage" != "null" ]; then
            echo "$has_usage"
            break
        fi
    done)

    if [ -z "$usage" ]; then
        return 1
    fi

    # Sum the three token types that count toward context
    local input_tokens cache_read cache_create total
    input_tokens=$(echo "$usage" | jq -r '.input_tokens // 0')
    cache_read=$(echo "$usage" | jq -r '.cache_read_input_tokens // 0')
    cache_create=$(echo "$usage" | jq -r '.cache_creation_input_tokens // 0')

    total=$((input_tokens + cache_read + cache_create))

    # Apply fudge factor for hidden context (system prompt, MCP definitions, CLAUDE.md)
    local adjusted
    adjusted=$((total * FUDGE_FACTOR / 100))

    echo "$adjusted $CONTEXT_WINDOW_SIZE"
    return 0
}

# ============================================================================
# Calculate percentage and generate output
# ============================================================================
calculate_and_output() {
    local tokens="$1"
    local window_size="$2"
    local source="$3"

    # Calculate percentage of usable context (80% of window)
    local usable=$((window_size * 80 / 100))
    local percent=$((tokens * 100 / usable))

    # Cap at 100%
    if [ "$percent" -gt 100 ]; then
        percent=100
    fi

    # Determine action based on threshold
    if [ "$percent" -ge "$THRESHOLD_HIGH" ]; then
        cat << EOF
## Context at ${percent}% - CHECKPOINT NOW

**Source:** $source
**Tokens:** ~${tokens} / ${usable} usable (${window_size} total window)

You MUST run \`checkpoint_save()\` immediately. Auto-compact is imminent.

\`\`\`
mcp__contextd__checkpoint_save(
  session_id: "\${SESSION_ID}",
  project_path: "$(pwd)",
  name: "context-${percent}pct-$(date +%H%M)",
  description: "Checkpoint at ${percent}% context usage",
  summary: "[FILL: Current task state]",
  context: "[FILL: Key context to preserve]",
  full_state: "[FILL: Complete state]",
  token_count: ${tokens},
  threshold: 0.${percent},
  auto_created: false
)
\`\`\`
EOF
    elif [ "$percent" -ge "$THRESHOLD_MEDIUM" ]; then
        cat << EOF
## Context at ${percent}% - Checkpoint Recommended

**Source:** $source | **Tokens:** ~${tokens} / ${usable} usable

Consider running \`checkpoint_save()\` soon. You're approaching auto-compact territory.
EOF
    elif [ "$percent" -ge "$THRESHOLD_LOW" ]; then
        cat << EOF
## Context at ${percent}%

**Source:** $source | **Tokens:** ~${tokens} / ${usable} usable

Good time for an optional checkpoint if this is a good stopping point.
EOF
    fi
    # Below 50%: No output (don't clutter context)
}

# ============================================================================
# Main
# ============================================================================
main() {
    local stdin_data=""
    local tokens=""
    local window_size=""
    local source=""

    # Read stdin if available (for statusline JSON)
    if [ ! -t 0 ]; then
        stdin_data=$(cat)
    fi

    # Tier 1: Try statusline context_window
    if result=$(try_statusline_context "$stdin_data"); then
        tokens=$(echo "$result" | cut -d' ' -f1)
        window_size=$(echo "$result" | cut -d' ' -f2)
        source="context_window (accurate)"
    # Tier 2: Fall back to JSONL parsing
    elif result=$(parse_jsonl_transcript); then
        tokens=$(echo "$result" | cut -d' ' -f1)
        window_size=$(echo "$result" | cut -d' ' -f2)
        source="JSONL transcript (+30% estimate)"
    else
        # No data available - silent exit
        exit 0
    fi

    # Generate output based on thresholds
    calculate_and_output "$tokens" "$window_size" "$source"
}

main "$@"
