# Work PRs (Tabbed Session Standard)

**STANDARD**: Single tmux session with multiple tabs (one per task).

## Usage

```bash
/work-prs-tabbed --task-ids 29,30,31,32
```

## What It Does

Creates a **single tmux session** with multiple tabs (windows), one tab per task:
- Each tab in its own worktree directory
- Each tab on its own feature branch
- Each tab displays task information
- Easy switching with `Ctrl+b [0-3]` or `Ctrl+b n/p`

## Session Structure

```
Session: contextd-p0-epics
├─ Tab 0 (Task-29-Epic1): ~/projects/contextd-epic-1-ollama-provider
├─ Tab 1 (Task-30-Epic2): ~/projects/contextd-task-30-epic-2-tokenizer
├─ Tab 2 (Task-31-Epic3): ~/projects/contextd-task-31-epic-3-config
└─ Tab 3 (Task-32-Epic4): ~/projects/contextd-task-32-epic-4-testing
```

## Workflow

1. **Attach to session**: `tmux attach -t contextd-p0-epics`
2. **Switch tabs**: `Ctrl+b [0-3]` or `Ctrl+b n/p`
3. **In each tab**: Start Claude Code and use `golang-pro` skill
4. **Work in parallel**: Switch between tabs as needed
5. **Detach**: `Ctrl+b d` (session keeps running)

## Navigation

- `Ctrl+b 0` - Task 29 (Epic 1)
- `Ctrl+b 1` - Task 30 (Epic 2)
- `Ctrl+b 2` - Task 31 (Epic 3)
- `Ctrl+b 3` - Task 32 (Epic 4)
- `Ctrl+b n` - Next tab
- `Ctrl+b p` - Previous tab
- `Ctrl+b w` - List all windows
- `Ctrl+b d` - Detach (keeps running)

## Stop Session

```bash
tmux kill-session -t contextd-p0-epics
```

## Implementation

```bash
#!/bin/bash
SESSION_NAME="contextd-p0-epics"
BASE_DIR="$HOME/projects"

# Kill existing session
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Create session with first tab
tmux new-session -d -s "$SESSION_NAME" -n "Task-29-Epic1" -c "$BASE_DIR/contextd-epic-1-ollama-provider"
tmux send-keys -t "$SESSION_NAME:0" "task-master show 29" C-m

# Add more tabs
tmux new-window -t "$SESSION_NAME:1" -n "Task-30-Epic2" -c "$BASE_DIR/contextd-task-30-epic-2-tokenizer"
tmux send-keys -t "$SESSION_NAME:1" "task-master show 30" C-m

# ... repeat for tasks 31, 32

# Select first tab
tmux select-window -t "$SESSION_NAME:0"
```

## Why This Is Standard

✅ **Single session** - Easier to manage than 4 separate sessions
✅ **Tab navigation** - Fast switching with `Ctrl+b [0-3]`
✅ **Persistent** - Detach/reattach without losing state
✅ **Clear organization** - Each tab = one epic/task
✅ **Parallel work** - Switch between tasks seamlessly

## Notes

- Session persists after detaching (`Ctrl+b d`)
- Survives terminal close (reattach with `tmux attach`)
- Each tab is independent workspace
- All tabs share same tmux session
- Compatible with Claude Code in each tab
