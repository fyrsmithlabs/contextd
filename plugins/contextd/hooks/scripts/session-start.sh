#!/usr/bin/env bash
# contextd plugin — SessionStart hook.
#
# Reminds Claude to use contextd's cross-session memory and checkpoint tools at
# the start of a session. Defensive by design: if the contextd binaries are not
# installed, it exits 0 silently so it never blocks or breaks a session.
set -euo pipefail

# No contextd binary present → nothing to do.
if ! command -v contextd >/dev/null 2>&1 && ! command -v ctxd >/dev/null 2>&1; then
  exit 0
fi

# SessionStart hooks may emit additionalContext as JSON on stdout.
cat <<'JSON'
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "contextd is available. Before exploring code, run semantic_search and memory_search. Record durable learnings with memory_record and save context with checkpoint_save when nearing context limits."
  }
}
JSON
