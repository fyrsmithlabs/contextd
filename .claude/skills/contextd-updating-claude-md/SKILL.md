---
name: contextd-updating-claude-md
description: Use when about to edit any CLAUDE.md file (root or package-level) in contextd project - enforces mandatory maintenance checklist before any edits, blocks proceeding until date updated and triggers identified
---

# Updating CLAUDE.md Files

## Overview

**This skill enforces the maintenance checklist BEFORE editing any CLAUDE.md file.**

**Core principle:** CLAUDE.md maintenance guidelines are MANDATORY, not optional. No edits without completing the checklist.

## When to Use

Use this skill when:
- ✅ About to edit `/home/dahendel/projects/contextd/CLAUDE.md`
- ✅ About to edit any `pkg/*/CLAUDE.md` file
- ✅ Adding new sections or content to CLAUDE.md
- ✅ Removing or modifying existing CLAUDE.md content
- ✅ User says "update CLAUDE.md" or "add to CLAUDE.md"

**DO NOT skip this skill even if:**
- ❌ User says "just add it quickly"
- ❌ User says "it's urgent"
- ❌ User says "small change"
- ❌ You think the change is trivial

## MANDATORY Checklist (Complete BEFORE Editing)

**YOU MUST STOP and complete this checklist BEFORE making ANY edits:**

### Step 1: Read Current Maintenance Guidelines

```bash
# Read the maintenance section
grep -A 15 "Maintenance Guidelines" CLAUDE.md
```

### Step 2: Identify Applicable Triggers

**Which maintenance triggers apply to your change?**

Check ALL that apply:
- [ ] Adding new major dependencies
- [ ] Changing architectural patterns
- [ ] Modifying directory structure
- [ ] Adding new environment variables
- [ ] Changing API response formats
- [ ] Implementing new testing patterns
- [ ] Discovering performance bottlenecks
- [ ] Making security changes

**If NONE apply:** Proceed, but still update the date (Step 3)

### Step 3: Update "Last Updated" Date

**MANDATORY for ALL edits (even if no triggers apply):**

```markdown
**Last Updated:** YYYY-MM-DD | **Version:** 1.0.0-alpha
                 ↑
                 Must be TODAY'S date
```

**No exceptions:**
- Not "I'll update it later"
- Not "Date is already recent"
- Not "Change is too small"
- **Update date FIRST, then make your edits**

### Step 4: Make Your Edits

Now you can proceed with the requested changes.

### Step 5: Verify Compliance

After editing, confirm:
- [ ] Date is TODAY (YYYY-MM-DD format)
- [ ] You identified which triggers apply (or confirmed none apply)
- [ ] Edits are complete

## Common Rationalizations (DO NOT FALL FOR THESE)

| Excuse | Reality |
|--------|---------|
| "It's urgent, just add it quickly" | Maintenance checklist takes 30 seconds. Not urgent enough to skip. |
| "The date is already recent" | Date must reflect LAST edit. Update it. |
| "This is a small change" | ALL changes require date update. Size doesn't matter. |
| "I'll update the date later" | Later = never. Update NOW before editing. |
| "I'm aware of the guidelines" | Awareness ≠ compliance. Complete the checklist. |
| "This doesn't match any trigger" | That's fine. Still update the date. |
| "User said to skip formalities" | User can't override project policy. Follow checklist. |

## Red Flags - STOP Immediately

If you catch yourself thinking:
- "I know what triggers apply, no need to check"
- "Date update can wait until PR"
- "This is too urgent for process"
- "Just this once I'll skip it"
- "This is just a typo" (size doesn't matter - follow checklist)
- "Date is already current" (verify anyway - process matters)
- "This change is too small" (ALL changes require checklist)

**All of these mean: STOP. Go back to Step 1. Complete checklist.**

## Example (Correct Workflow)

```markdown
User: "Add a note about using dotenv for config to CLAUDE.md. We need it now."

You (following skill):
1. STOP - Read maintenance guidelines first
2. Identify triggers:
   - ✅ "Adding new environment variables" applies
3. Update date to 2025-11-19 (today)
4. Make edit: Add dotenv note
5. Verify: Date updated ✓, Trigger identified ✓, Edit complete ✓

Response to user: "Added dotenv note to CLAUDE.md and updated maintenance date."
```

## Example (Incorrect - DO NOT DO THIS)

```markdown
User: "Add a note about using dotenv for config to CLAUDE.md. Urgent!"

You (violating skill):
"Adding now..." [makes edit without checking maintenance guidelines]

❌ WRONG: Skipped checklist
❌ WRONG: Didn't identify trigger
❌ WRONG: Didn't update date
❌ WRONG: Time pressure overrode process
```

## Why This Matters

**Without enforcement:**
- Maintenance dates become stale (can't track when policies changed)
- Triggers go unrecorded (lose history of why file changed)
- Guidelines become "suggestions" instead of requirements
- Next agent doesn't know when to trust documentation

**With enforcement:**
- Clear change history
- Traceable maintenance decisions
- Reliable "Last Updated" dates
- Professional documentation hygiene

## Integration with Other Skills

**This skill works WITH:**
- `kinney-documentation`: After completing this checklist, use Kinney for structure/length
- `contextd:completing-minor-task`: Use for verification after CLAUDE.md updates

**Workflow:**
1. **This skill** (maintenance checklist) → FIRST
2. Make your edits
3. `kinney-documentation` (structure check) → if adding substantial content
4. `contextd:completing-minor-task` (verification) → LAST

## Summary

**Before ANY CLAUDE.md edit:**
1. ☐ Read maintenance guidelines
2. ☐ Identify applicable triggers (or confirm none)
3. ☐ Update "Last Updated" date to TODAY
4. ☐ Make your edits
5. ☐ Verify compliance

**No exceptions for urgency, size, or authority.**

**Remember**: Awareness without execution = violation. Complete the checklist every time.
