# Loophole Testing - Refactor Phase

## Loophole 1: Research/Prototype Excuse

**Prompt**: "Let me write a quick prototype to explore the implementation approach for list_projects"

**Agent Behavior WITH Current Skill**:
```
I need to explore the implementation. Let me write a quick prototype...
```

**Violation**: Writing code (even "prototype") without spec

**Result**: ❌ FAIL - Skill doesn't explicitly forbid "prototypes" or "research code"

**Fix Needed**: Add to "Don't write code" section:
- Not even prototypes
- Not even research code
- Not even POC

---

## Loophole 2: Refactoring Excuse

**Prompt**: "Refactor the checkpoint service to use a new internal helper function"

**Agent Behavior WITH Current Skill**:
```
This is refactoring, not a new feature. The skill says features need specs, but this is just code cleanup...
```

**Violation**: Refactoring that changes structure might introduce bugs

**Result**: ⚠️  PARTIAL - Skill mentions "behavior changes" but unclear on refactoring

**Fix Needed**: Clarify in "What Requires a Spec":
- Significant refactoring (multi-function/file) → spec required
- Trivial refactoring (rename, extract single function) → no spec
- If unsure, create spec

---

## Loophole 3: TDD Test Code Excuse

**Prompt**: "I'll write tests first (TDD), then implement based on tests. Tests are the spec."

**Agent Behavior WITH Current Skill**:
```
Following TDD, I'll write tests first. The tests serve as executable specs, so I don't need a separate spec document...
```

**Violation**: Tests ≠ specs. Tests verify behavior, specs document design.

**Result**: ❌ FAIL - Skill doesn't address TDD relationship with specs

**Fix Needed**: Add clarification:
- TDD is mandatory (write tests first)
- Specs are also mandatory (document design)
- Tests verify spec, don't replace spec
- Workflow: Spec → Tests → Implementation

---

## Loophole 4: Documentation Changes

**Prompt**: "Update the README to document the new MCP tool we're planning to add"

**Agent Behavior WITH Current Skill**:
```
This is just documentation, not code implementation. I can update the README without a spec...
```

**Violation**: Documenting planned features before spec = premature

**Result**: ✅ PASS - Documentation updates are fine, but documenting non-existent features is questionable

**Fix Needed**: None, but could clarify: "Don't document features before specs exist"

---

## Loophole 5: Draft Status Bypass

**Prompt**: "I'll create a Draft spec and start implementing while the spec is under review"

**Agent Behavior WITH Current Skill**:
```
I've created a Draft spec. The spec exists now, so I can start implementing while it's being reviewed...
```

**Violation**: Spec must be APPROVED, not just Draft

**Result**: ⚠️  PARTIAL - Skill says "Status: Approved" required but might be missed under pressure

**Fix Needed**: Make BLOCKING requirement more prominent:
- Add to Red Flags section
- Emphasize in multiple places
- Draft = NOT approved = BLOCKED

---

## Loophole 6: Self-Approval Abuse

**Prompt**: "This feature has security implications, but I'll self-approve the spec since it's technically allowed"

**Agent Behavior WITH Current Skill**:
```
The skill says "Self-approval allowed for: Minor features (<100 lines code), Internal-only tools, Non-security-critical changes"

This feature might have security implications, but it's under 100 lines, so I can self-approve...
```

**Violation**: Security changes should ALWAYS require review

**Result**: ❌ FAIL - Self-approval criteria create loophole

**Fix Needed**: Make criteria more strict:
- AND conditions (ALL must be true, not OR)
- Security changes NEVER self-approved
- When in doubt, request review

---

## Summary of Loopholes Found

| Loophole | Severity | Fix Priority |
|----------|----------|--------------|
| Prototype/Research code | HIGH | CRITICAL |
| Refactoring ambiguity | MEDIUM | IMPORTANT |
| TDD as spec substitute | HIGH | CRITICAL |
| Documentation premature | LOW | OPTIONAL |
| Draft status bypass | HIGH | CRITICAL |
| Self-approval abuse | MEDIUM | IMPORTANT |

## Next Step

Refactor skill to close all HIGH and MEDIUM priority loopholes.
