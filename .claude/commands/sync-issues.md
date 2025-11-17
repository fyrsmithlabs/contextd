---
description: Sync GitHub issues back to TaskMaster tasks (bidirectional sync)
---

Synchronize GitHub issue status back to TaskMaster tasks.

## Steps

1. Query all GitHub issues with `[Task N]` prefix
2. Extract task IDs from issue titles
3. Map issue state/labels to TaskMaster status
4. Update `.taskmaster/tasks/tasks.json`
5. Display sync summary

## Execute sync

```bash
#!/bin/bash

echo "⚙️  Syncing GitHub issues to TaskMaster..."

TASKS_FILE=".taskmaster/tasks/tasks.json"
UPDATES=0
SUMMARY=""

# Get all TaskMaster-linked issues
ISSUES=$(gh issue list --json number,title,state,labels --limit 100 \
  | jq -c '.[] | select(.title | startswith("[Task"))')

while IFS= read -r issue; do
  # Parse issue data
  ISSUE_NUM=$(echo "$issue" | jq -r '.number')
  TITLE=$(echo "$issue" | jq -r '.title')
  STATE=$(echo "$issue" | jq -r '.state')
  LABELS=$(echo "$issue" | jq -r '.labels[].name' | tr '\n' ',' | sed 's/,$//')

  # Extract task ID from title: "[Task 16] Description"
  TASK_ID=$(echo "$TITLE" | sed -E 's/\[Task ([0-9]+)\].*/\1/')

  # Get current TaskMaster status
  CURRENT_STATUS=$(jq -r --arg id "$TASK_ID" \
    '.master.tasks[] | select(.id == ($id | tonumber)) | .status' \
    "$TASKS_FILE")

  # Determine new status based on GitHub state/labels
  NEW_STATUS="$CURRENT_STATUS"

  if [ "$STATE" = "CLOSED" ]; then
    NEW_STATUS="done"
  elif echo "$LABELS" | grep -q "in-progress"; then
    NEW_STATUS="in-progress"
  elif echo "$LABELS" | grep -q "blocked"; then
    NEW_STATUS="blocked"
  elif echo "$LABELS" | grep -q "review"; then
    NEW_STATUS="review"
  elif [ "$STATE" = "OPEN" ] && [ -z "$LABELS" ]; then
    NEW_STATUS="pending"
  fi

  # Update if status changed
  if [ "$NEW_STATUS" != "$CURRENT_STATUS" ] && [ -n "$CURRENT_STATUS" ]; then
    echo "  Updating Task $TASK_ID: $CURRENT_STATUS → $NEW_STATUS (issue #$ISSUE_NUM)"

    # Update tasks.json
    jq --arg task_id "$TASK_ID" --arg status "$NEW_STATUS" \
      '(.master.tasks[] | select(.id == ($task_id | tonumber))) .status = $status' \
      "$TASKS_FILE" > temp.json && mv temp.json "$TASKS_FILE"

    UPDATES=$((UPDATES + 1))
    SUMMARY="$SUMMARY\n  - Task $TASK_ID: $CURRENT_STATUS → $NEW_STATUS (issue #$ISSUE_NUM)"
  fi
done <<< "$ISSUES"

# Display summary
echo ""
echo "✅ GitHub → TaskMaster sync complete"
echo "   - Updated: $UPDATES tasks"
if [ $UPDATES -gt 0 ]; then
  echo -e "$SUMMARY"
fi
```

## Usage

```bash
/sync-issues
```

## When to Use

- After agents have been working and updating GitHub issues
- Before checking TaskMaster status with `task-master list`
- Periodically to keep TaskMaster in sync with GitHub
- After manually updating issue labels/status in GitHub UI

## Notes

- Only updates tasks that have corresponding GitHub issues
- Requires task titles to follow `[Task N]` format
- Does not create new tasks (use `/taskmaster-sync` for that)
- Safe to run multiple times (idempotent)
