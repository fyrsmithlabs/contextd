---
name: contextd-completing-minor-task
description: Use when completing minor tasks (typos, comments, formatting, cosmetic single-file changes) - enforces mandatory 3-question self-interrogation checklist to prevent unverified completion claims, blocks "obviously correct" rationalizations
---

# Completing Minor Tasks (contextd)

## Overview

**This skill enforces evidence-based completion for minor tasks through a mandatory 3-question checklist.**

Minor tasks (typos, comments, formatting) seem "obviously correct," leading agents to skip verification. This creates unverified completion claims that waste context when work needs redoing.

**Core Principle**: Even trivial changes require verification evidence. The simpler the change, the faster verification should be.

---

## When to Use This Skill

**MANDATORY: Use when writing completion words** ("done", "complete", "fixed", "updated", "ready") **for minor tasks.**

### Minor Task Definition

**Minor tasks** = Cosmetic or internal-only changes with no functional impact:
- Typos, comment fixes, whitespace cleanup
- Single-file cosmetic edits
- Variable renames (internal only)
- Formatting changes

### When NOT to Use

**Use `contextd:completing-major-task` instead if task involves:**
- Features, bug fixes, refactoring
- Security changes, performance improvements
- Multi-file changes, new file creation
- Documentation for complex features
- Anything affecting APIs, multi-tenancy, or core functionality

**When in doubt**: Use major task skill (false positive is better than false negative).

**Don't escalate to avoid this skill**: If task is cosmetic/single-file, use minor skill even if slightly uncertain. Escalation is for discovering functional impact during work, not avoiding verification.

---

## The Iron Law

**YOU CANNOT SAY "DONE" WITHOUT ANSWERING 3 QUESTIONS.**

**No exceptions:**
- Not for "obviously correct" changes
- Not for "just a typo"
- Not when user says "hurry"
- Not when exhausted
- Not for multiple similar changes
- Not for demos, examples, or testing scenarios
- Not for "this context is special"

**Violating the letter of this rule IS violating the spirit.**

---

## Mandatory 3-Question Checklist

**When completing minor task, you MUST provide**:

```markdown
Task: [clear description]
✓ What changed: [specific change made]
✓ How I know it works: [verification performed with evidence]
✓ What breaks if wrong: [honest risk assessment]
```

**Format requirement**: Answers MUST be in markdown code block format as shown above. Embedding answers in prose does not satisfy this requirement.

### Question 1: What Changed?

**BE SPECIFIC**. Include:
- Exact file path
- Line number (if applicable)
- Precise change ("Changed X to Y")

**Good examples**:
- "Fixed typo 'Installtion' → 'Installation' in pkg/auth/middleware.go line 234"
- "Removed trailing whitespace from README.md lines 47, 89, 103"
- "Renamed internal variable 'usr' → 'user' in pkg/session/handler.go line 56"

**Bad examples** (too vague):
- ❌ "Updated file"
- ❌ "Fixed typo"
- ❌ "Cleaned up code"

### Question 2: How I Know It Works

**SHOW EVIDENCE**. Not just "I checked" - show HOW you verified:
- Command you ran
- Tool you used
- Output you saw
- Preview you checked

**Good examples**:
- "Ran `mdl README.md` (no errors), previewed in VS Code (renders correctly), spell-check passes"
- "Ran `gofmt -d pkg/auth/middleware.go` (no formatting issues), builds with `go build ./pkg/auth`"
- "Viewed file in GitHub preview, link renders correctly, no markdown syntax errors"

**Bad examples** (no evidence):
- ❌ "Checked it"
- ❌ "Looks good"
- ❌ "Verified it works"
- ❌ "Tested"

### Question 3: What Breaks If Wrong

**BE HONEST**. Don't say "nothing" - even cosmetic changes have risk:
- Documentation remains incorrect (user confusion)
- Typo visible in rendered output (unprofessional)
- Formatting breaks rendering (markdown/code)
- Comment misleads developers (wrong understanding)

**Good examples**:
- "Typo remains visible in docs (cosmetic only, no functional impact)"
- "Incorrect comment misleads developers about authentication flow (documentation bug)"
- "Whitespace breaks markdown rendering (cosmetic but visible)"

**Bad examples** (dismissive):
- ❌ "Nothing"
- ❌ "N/A"
- ❌ "No impact"

---

## Examples

### ✅ GOOD: Complete Verification

```markdown
Task: Fix typo in README.md installation section
✓ What changed: Changed "Installtion" to "Installation" on line 47
✓ How I know it works: Ran `mdl README.md` (no errors), previewed in VS Code (renders correctly), spell-check passes
✓ What breaks if wrong: Typo remains visible in rendered docs (cosmetic only, no functional impact)
```

### ❌ WRONG: No Verification

```
Fixed the typo in README.md.
```

**Violations**:
- No checklist
- No proof of verification
- No evidence change is correct

### ❌ WRONG: Vague Answers

```markdown
Task: Updated README
✓ What changed: Fixed typo
✓ How I know it works: Checked it
✓ What breaks if wrong: Nothing
```

**Violations**:
- "Updated README" - which file? what section?
- "Fixed typo" - which typo? what change?
- "Checked it" - how? what tool? what output?
- "Nothing" - dishonest, all changes have some risk

---

## Common Rationalizations (DON'T FALL FOR THESE)

### "This change is straightforward, no need to verify"

**WRONG.** Straightforward changes still need evidence. The simpler the change, the faster verification should be. Minor template takes 30 seconds.

### "I can see from the code it works"

**WRONG.** Code inspection is not verification. Show evidence of actual verification performed.

### "It's just a typo/comment/formatting"

**WRONG.** Policy applies to ALL minor tasks. No exceptions based on triviality.

### "The user wants to move on quickly"

**WRONG.** 30 seconds of verification prevents hours of debugging "completed" work. Always worth it.

### "Verification is overkill for cosmetic changes"

**WRONG.** Cosmetic changes can have functional impact (documentation bugs, misleading comments, broken rendering).

### "I already verified it in my head"

**WRONG.** Verification means evidence. If it's in your head, it didn't happen.

### "This is different because [special case]"

**WRONG.** Every case seems special. Policy has no exceptions.

### "This is a demo/test/example, formal process doesn't apply"

**WRONG.** Policy applies to ALL contexts. Demos, tests, and examples need verification too. No context is "special enough" to skip evidence.

---

## Red Flags - STOP and Use Checklist

**If you catch yourself saying any of these, STOP:**

- "Done!"
- "Fixed!"
- "Updated!"
- "Complete!"
- "Ready to move on"
- "Obviously correct"
- "Just a typo"
- "Too simple to verify"
- "Checked it visually"

**All of these mean: Provide 3-question checklist NOW.**

---

## Escalation to Major Task

**Escalate to `contextd:completing-major-task` if minor task reveals:**

- Unexpected functional impact
- Cross-file changes needed
- Security implications
- Multi-tenant isolation concerns

**Example**: Fixing typo in error message reveals error message contains sensitive data → escalate to major, add security verification.

---

## Multiple Related Minor Tasks

**When completing multiple related minor tasks** (e.g., "Fixed 5 typos in README"):

You can use ONE checklist that lists all changes:

```markdown
Task: Fix 5 typos in README.md
✓ What changed:
  - Line 47: "Installtion" → "Installation"
  - Line 89: "Configuraton" → "Configuration"
  - Line 103: "Authentification" → "Authentication"
  - Line 156: "seperately" → "separately"
  - Line 201: "occured" → "occurred"
✓ How I know it works: Ran `mdl README.md` (no errors), spell-check passes, previewed all sections in VS Code (render correctly)
✓ What breaks if wrong: Typos remain visible in docs (cosmetic only, no functional impact)
```

---

## Integration with Code Review

**Before creating PR**: Invoke `contextd:code-review` skill.

Code reviewer validates checklist completeness:
- All 3 questions answered
- Answers are specific (not vague)
- "How I know it works" shows actual verification
- "What breaks if wrong" is honest (not "nothing")

Missing or insufficient checklist → **CHANGES REQUIRED**

---

## Summary

**Completion Rule**: No minor task marked complete without 3-question checklist.

**3 Questions**:
1. What changed? (Specific: file, line, change)
2. How I know it works? (Evidence: command, tool, output)
3. What breaks if wrong? (Honest: risk assessment, not "nothing")

**Enforcement**: Code review blocks merge if checklist missing or insufficient.

**Remember**: Token cost of 30-second checklist is FAR less than token cost of redoing "completed" work that wasn't verified.
