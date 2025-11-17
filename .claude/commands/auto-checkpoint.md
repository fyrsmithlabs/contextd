---
name: auto-checkpoint
description: Auto-save checkpoint when context approaching limits
---

Automatically save current session to contextd checkpoint.

This command:
1. Saves current context to checkpoint with auto-generated summary
2. Reports checkpoint ID for easy resume
3. Recommends /clear if context > 90%
4. Provides resume command for after /clear

Use this command when:
- Context warning appears (>70%)
- Before ending session
- Before /clear command
- When switching major tasks
