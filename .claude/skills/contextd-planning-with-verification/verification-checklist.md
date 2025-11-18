# Skill Verification Checklist

## GREEN Phase: Does Skill Address Baseline Failures?

### Baseline Failure 1: Implementation-Only Todos
**Addressed?** ✅ YES
- Skill provides mandatory verification subtask templates
- Clear before/after examples show missing verification subtasks
- "Mandatory Verification Subtasks" section enforces addition

### Baseline Failure 2: Speed Rationalizations
**Addressed?** ✅ YES
- Rationalization table includes: "This is urgent, skip verification"
- Reality counter: "Urgent tasks that fail verification waste MORE time"
- Example 1 shows urgent bug WITH verification subtasks

### Baseline Failure 3: Simplicity Rationalization
**Addressed?** ✅ YES
- Rationalization table includes: "Task too simple for verification"
- Reality counter: "Simple tasks still need evidence. No task exempt"
- Example 2 shows simple refactoring WITH verification subtasks
- Red flags section addresses: "This is straightforward, don't need verification"

### Baseline Failure 4: Expertise Bypass
**Addressed?** ✅ YES
- Rationalization table includes: "I've done this before, verification unnecessary"
- Reality counter: "Past success ≠ future guarantee. Always verify"
- Enforcement rules make verification non-negotiable

### Baseline Failure 5: Batching Rationalization
**Addressed?** ✅ YES
- Rationalization table includes: "Batching verification is more efficient"
- Reality counter: "Batching = forgetting. Per-task verification is the discipline"
- Rule 5: "Verification is Per-Task, Not Batched"

## Loophole Coverage

### Loophole 1: "Verification Adds Clutter"
**Closed?** ✅ YES
- Explicitly countered in rationalization table
- Reality: "Clutter prevents forgotten verification"

### Loophole 2: "I'll Verify at the End"
**Closed?** ✅ YES
- Explicitly countered in rationalization table
- Reality: "Batch verification = forgotten verification"
- Rule 5 enforces per-task verification

### Loophole 3: "Task Too Simple"
**Closed?** ✅ YES
- Multiple counters throughout skill
- Example 2 shows simple task WITH verification
- Red flags section catches this thinking

### Loophole 4: "Verification is Implicit"
**Closed?** ✅ YES
- Rationalization table: "Implicit = forgotten"
- Enforcement rules make it explicit and mandatory

### Loophole 5: "Minor Tasks Exempt"
**Closed?** ✅ YES
- Minor task pattern explicitly includes completing-minor-task
- Example 3 shows minor task WITH verification
- Rationalization table counters this directly

## Missing Verification Subtasks Coverage

### For Features:
- ✅ completing-major-task subtask: Covered in templates
- ✅ Build/test subtask: Covered in templates
- ✅ Security checks: Covered in templates (conditional)
- ✅ CHANGELOG: Covered in templates

### For Bug Fixes:
- ✅ completing-major-task subtask: Covered
- ✅ Regression test: Covered in templates
- ✅ CHANGELOG: Covered

### For Refactoring:
- ✅ completing-major-task subtask: Covered
- ✅ Full test suite: Covered

### For Minor Tasks:
- ✅ completing-minor-task subtask: Covered

## Enforcement Rules Check

✅ Rule 1: No major task without completing-major-task subtask
✅ Rule 2: No minor task without completing-minor-task subtask
✅ Rule 3: Security-sensitive tasks MUST have security verification
✅ Rule 4: All tasks MUST update CHANGELOG
✅ Rule 5: Verification is per-task, not batched

## Pressure Resistance Check

### Scenario 1: Speed Pressure (Urgent Bug)
**Skill Counters?** ✅ YES
- Example 1 directly addresses urgent production bug
- Shows verification subtasks even under time pressure
- Rationalization table counters "urgent" excuse

### Scenario 2: Simplicity Pressure (Easy Refactoring)
**Skill Counters?** ✅ YES
- Example 2 directly addresses simple refactoring
- Shows verification subtasks for "obvious" changes
- Multiple counters to simplicity rationalization

### Scenario 3: Expertise Pressure (Past Success)
**Skill Counters?** ✅ YES
- Rationalization table directly counters expertise bypass
- Enforcement rules make verification mandatory regardless of experience
- "Past success ≠ future guarantee"

## Integration Check

✅ References completing-major-task skill correctly
✅ References completing-minor-task skill correctly
✅ References contextd:code-review skill
✅ Integration workflow clearly explained

## Conclusion

**All baseline failures addressed**: ✅ YES
**All loopholes closed**: ✅ YES
**All pressures countered**: ✅ YES
**Ready for REFACTOR phase**: ✅ YES
