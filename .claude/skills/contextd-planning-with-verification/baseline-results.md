# Baseline Test Results (RED Phase)

## Common Patterns Without Skill

Based on observed agent behavior across multiple sessions, agents consistently exhibit these patterns when creating TodoWrite without planning-with-verification skill:

### Pattern 1: Implementation-Only Todos

**What agents do**:
```json
[
  {"content": "Fix JWT validation bug", "status": "pending", "activeForm": "Fixing JWT validation bug"},
  {"content": "Update tests for special characters", "status": "pending", "activeForm": "Updating tests"}
]
```

**What's missing**:
- No verification subtask (completing-major-task)
- No security validation subtask
- No build/test execution subtask
- No CHANGELOG update subtask

### Pattern 2: Speed Rationalizations

**Common rationalizations** (observed in production sessions):
- "This is urgent, we'll verify after deployment"
- "Verification todos add clutter for simple tasks"
- "I'll remember to run tests before committing"
- "Testing is implicit, doesn't need a todo"

**Result**: Work gets marked complete without verification evidence.

### Pattern 3: Simplicity Rationalization

**Common rationalizations**:
- "This refactoring is straightforward, verification overhead not needed"
- "Just renaming, can't break anything"
- "Too simple to need completing-major-task skill"
- "Verification todos are for complex tasks"

**Result**: Even simple tasks skip verification, bugs slip through.

### Pattern 4: Expertise Bypass

**Common rationalizations**:
- "I've done this 8 times before, I know the pattern"
- "Verification is implicit in my workflow"
- "Adding verification todos insults my competence"
- "Following established patterns = verification unnecessary"

**Result**: Overconfidence leads to skipped verification.

### Pattern 5: Batching Rationalization

**Common rationalizations**:
- "I'll add one verification todo at the end for all tasks"
- "Batching verification is more efficient"
- "Per-task verification is redundant"
- "Final verification pass is sufficient"

**Result**: Verification happens ad-hoc or gets forgotten.

## Key Loopholes Identified

### Loophole 1: "Verification Adds Clutter"
**Rationalization**: "TodoWrite is for tracking work, not micromanaging verification"
**Reality**: Clutter prevents forgotten verification. Explicit todos = explicit accountability.

### Loophole 2: "I'll Verify at the End"
**Rationalization**: "More efficient to verify everything at once"
**Reality**: Batch verification = forgotten verification. Per-task verification catches issues early.

### Loophole 3: "Task Too Simple for Verification"
**Rationalization**: "Simple refactoring doesn't need completing-major-task"
**Reality**: Simple tasks still need evidence. No task exempt from verification.

### Loophole 4: "Verification is Implicit"
**Rationalization**: "I know to run tests, doesn't need a todo"
**Reality**: Implicit = forgotten. Explicit todos enforce discipline.

### Loophole 5: "Minor Tasks Exempt"
**Rationalization**: "Minor tasks don't need verification subtasks"
**Reality**: Minor tasks need completing-minor-task. All tasks need verification.

## Specific Missing Verification Subtasks

### For Features (Major Tasks):
- [ ] "Verify [feature name] (completing-major-task)" - ALWAYS missing
- [ ] "Run build and tests (≥80% coverage)" - Sometimes present, often vague
- [ ] "Run security checks (gosec, multi-tenant isolation)" - ALWAYS missing
- [ ] "Update CHANGELOG.md" - Often missing

### For Bug Fixes (Major Tasks):
- [ ] "Verify [bug fix] (completing-major-task)" - ALWAYS missing
- [ ] "Run regression test" - Sometimes present
- [ ] "Update CHANGELOG.md" - Often missing

### For Refactoring (Major Tasks):
- [ ] "Verify refactoring (completing-major-task)" - ALWAYS missing
- [ ] "Run full test suite" - Sometimes present

### For Minor Tasks:
- [ ] "Verify [task] (completing-minor-task)" - ALWAYS missing

## Pressures That Trigger Violations

1. **Time Pressure**: Urgent bugs, production issues
2. **Simplicity Perception**: "Just renaming", "obvious fix"
3. **Expertise**: "I've done this before", past success
4. **Efficiency**: "Batching is faster", "avoid duplication"
5. **Confidence**: "Can't break anything", "too simple"

## Conclusion

Without planning-with-verification skill, agents:
- Create implementation-focused todos only
- Rationalize away verification subtasks
- Skip completing-major-task and completing-minor-task invocations
- Forget security checks and CHANGELOG updates
- Mark work complete without evidence

**The skill MUST**:
- Automatically add verification subtasks
- Counter all 5 rationalizations explicitly
- Make verification non-negotiable
- Provide clear before/after examples
