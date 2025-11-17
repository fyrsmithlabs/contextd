#!/bin/bash
# Standard: Spawn parallel worktrees in single tmux session with tabs
# Usage: ./spawn-parallel-worktrees.sh SESSION_NAME TASK_ID1,TASK_ID2,TASK_ID3

set -e

SESSION_NAME="${1:-contextd-parallel-tasks}"
TASK_IDS="${2:-29,30,31,32}"
BASE_DIR="$HOME/projects"
PROJECT_NAME="contextd"

# Parse task IDs
IFS=',' read -ra TASKS <<< "$TASK_IDS"

echo "🚀 Creating single tmux session: $SESSION_NAME"
echo "   Tasks: ${TASKS[*]}"
echo ""

# Kill existing session if exists
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Track if first window created
FIRST_WINDOW=true

for TASK_ID in "${TASKS[@]}"; do
  # Get task details from TaskMaster
  TASK_JSON=$(task-master show "$TASK_ID" 2>/dev/null || echo "{}")
  TASK_TITLE=$(echo "$TASK_JSON" | jq -r '.title // "Task-'$TASK_ID'"' 2>/dev/null || echo "Task-$TASK_ID")
  TASK_SLUG=$(echo "$TASK_TITLE" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | tr -cd '[:alnum:]-' | cut -c1-40)

  # Determine worktree path
  if [ "$TASK_ID" = "29" ]; then
    WORKTREE_PATH="$BASE_DIR/$PROJECT_NAME-epic-1-ollama-provider"
  else
    WORKTREE_PATH="$BASE_DIR/$PROJECT_NAME-task-$TASK_ID-$TASK_SLUG"
  fi

  # Ensure worktree exists
  if [ ! -d "$WORKTREE_PATH" ]; then
    echo "⚠️  Worktree not found: $WORKTREE_PATH"
    echo "   Creating worktree..."
    BRANCH="feature/task-$TASK_ID-$TASK_SLUG"
    cd "$BASE_DIR/$PROJECT_NAME"
    git worktree add "$WORKTREE_PATH" -b "$BRANCH" 2>/dev/null || git worktree add "$WORKTREE_PATH" "$BRANCH"
  fi

  # Create tmux window
  WINDOW_NAME="Task-$TASK_ID"

  if [ "$FIRST_WINDOW" = true ]; then
    # Create new session with first window
    tmux new-session -d -s "$SESSION_NAME" -n "$WINDOW_NAME" -c "$WORKTREE_PATH"
    WINDOW_INDEX=0
    FIRST_WINDOW=false
  else
    # Add new window to existing session
    tmux new-window -t "$SESSION_NAME" -n "$WINDOW_NAME" -c "$WORKTREE_PATH"
    WINDOW_INDEX=$(($(tmux list-windows -t "$SESSION_NAME" | wc -l) - 1))
  fi

  # Create TaskMaster symlink if needed
  if [ ! -L "$WORKTREE_PATH/.taskmaster" ]; then
    ln -sf "$BASE_DIR/$PROJECT_NAME/.taskmaster" "$WORKTREE_PATH/.taskmaster" 2>/dev/null || true
  fi

  # Calculate GitHub issue number (Task 29 = Issue 206, etc.)
  GITHUB_ISSUE=$((TASK_ID + 177))

  # Send commands to window - show GitHub issue details
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "clear" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "gh issue view $GITHUB_ISSUE | head -50" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo ''" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━'" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo '🚀 Ready to work on Task $TASK_ID (Issue #$GITHUB_ISSUE)!'" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo '   Worktree: $WORKTREE_PATH'" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo '   Start: claude (then use golang-pro skill)'" C-m
  tmux send-keys -t "$SESSION_NAME:$WINDOW_INDEX" "echo '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━'" C-m

  echo "✅ Tab $WINDOW_INDEX: Task $TASK_ID - $TASK_TITLE"
done

# Select first window
tmux select-window -t "$SESSION_NAME:0"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "⚙️  Session Created: $SESSION_NAME"
echo ""
echo "🚀 Attach to session:"
echo "   tmux attach -t $SESSION_NAME"
echo ""
echo "📊 Navigate tabs:"
echo "   Ctrl+b [0-9]  # Jump to tab N"
echo "   Ctrl+b n      # Next tab"
echo "   Ctrl+b p      # Previous tab"
echo "   Ctrl+b w      # List all tabs"
echo "   Ctrl+b d      # Detach (keeps running)"
echo ""
echo "🛑 Kill session:"
echo "   tmux kill-session -t $SESSION_NAME"
echo ""
