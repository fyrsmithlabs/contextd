---
name: context-check
description: Check current context usage and recommend actions
---

Check current context usage and recommend checkpoint if needed.

Steps:
1. Report current token usage from system context
2. Calculate percentage used (out of 200K budget)
3. If > 70%: Recommend /auto-checkpoint and /clear
4. If > 90%: URGENT - save checkpoint immediately
5. Provide checkpoint search command for easy resume

Context thresholds:
- 0-70% (0-140K): Safe
- 70-90% (140K-180K): Warning - checkpoint recommended
- 90%+ (180K+): Critical - checkpoint and clear NOW
