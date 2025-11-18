# REFACTOR Phase: Edge Cases and Additional Loopholes

## Potential New Loopholes Identified

### Loophole 6: "Partial Verification is Sufficient"
**Rationalization**: "I added the completing-major-task subtask, that's enough"
**Missing**: Security checks, CHANGELOG, build/test subtasks
**Risk**: Incomplete verification, missing critical checks
**Status in Skill**: ⚠️ PARTIALLY ADDRESSED
- Templates show all required subtasks
- But no explicit counter to "partial verification OK"

**Recommended Addition**: Add to rationalization table:
- "I added verification subtask, that's sufficient" → "Partial verification = incomplete verification. ALL required subtasks mandatory."

### Loophole 7: "Read-Only Tasks Don't Need Verification"
**Rationalization**: "Just reading/analyzing code, no verification needed"
**Missing**: Skill says "DO NOT use for reading/analyzing" but doesn't explain what to do instead
**Risk**: Agent creates TodoWrite for read-only tasks without verification
**Status in Skill**: ⚠️ PARTIALLY ADDRESSED
- "When to Use" section excludes read-only tasks
- But doesn't explain alternative (no TodoWrite needed for pure reading)

**Recommended Addition**: Clarify in "When to Use":
- "Pure reading/analysis tasks: Don't create TodoWrite (no deliverables to verify)"

### Loophole 8: "Spirit vs Letter"
**Rationalization**: "I'm following the spirit by adding some verification, even if not exact template"
**Missing**: No explicit counter to "spirit of the rule" rationalization
**Risk**: Agent adds minimal verification, claims compliance
**Status in Skill**: ❌ NOT ADDRESSED

**Recommended Addition**: Add early in skill:
- "**Following the spirit without the letter IS a violation.** Use exact templates, all required subtasks, no shortcuts."

### Loophole 9: "This is Research, Not Implementation"
**Rationalization**: "Researching SDK options isn't implementation, no verification needed"
**Missing**: Research tasks that produce decisions need verification
**Risk**: Research findings not verified (incomplete evaluation, missing options)
**Status in Skill**: ⚠️ PARTIALLY ADDRESSED
- "Pure research tasks" excluded from skill
- But research that produces specifications or decisions SHOULD have verification

**Recommended Addition**: Clarify research distinction:
- "Research with deliverables (specs, ADRs, decisions): Add verification subtask for deliverable"
- "Pure exploration (no deliverable): No TodoWrite needed"

### Loophole 10: "Documentation Doesn't Need Full Verification"
**Rationalization**: "Just updating docs, only need completing-minor-task"
**Missing**: Complex documentation (guides, specs) should use completing-major-task
**Risk**: Complex documentation treated as minor task, insufficient verification
**Status in Skill**: ⚠️ PARTIALLY ADDRESSED
- Detection rules mention "documentation for complex features" as major
- But documentation examples are all minor (typo fix)

**Recommended Addition**: Add documentation example showing major vs minor:
- Major: "Write DEVELOPMENT-WORKFLOW.md guide" → completing-major-task
- Minor: "Fix typo in README" → completing-minor-task

## Edge Cases to Address

### Edge Case 1: Multiple Related Tasks
**Scenario**: "Create 5 todos for feature with shared verification"
**Current Skill**: Implies per-task verification
**Question**: Do we need verification subtask after EACH implementation task?
**Answer**: YES (per Rule 5: "Verification is per-task")

**Verification**: Skill already addresses this in Rule 5. No change needed.

### Edge Case 2: Security-Sensitive Detection
**Scenario**: Task doesn't mention "auth" or "token" but affects security
**Current Skill**: Detection rules are keyword-based
**Risk**: Security-sensitive tasks missed by keyword detection
**Recommendation**: Add guidance for manual override:
- "If task affects security (even without keywords): Manually add security verification subtask"

### Edge Case 3: CHANGELOG Exemptions
**Scenario**: Documentation-only changes, internal refactoring
**Current Skill**: "All tasks MUST update CHANGELOG (unless pure docs)"
**Clarification**: Does "pure docs" mean all documentation or only certain types?
**Recommendation**: Clarify CHANGELOG exemptions:
- Exempt: Internal docs (CLAUDE.md, code comments), typo fixes
- Required: User-facing docs (README, API docs), guides, specs

### Edge Case 4: Emergency Hotfixes
**Scenario**: Production down, need to deploy fix in 5 minutes
**Current Skill**: No emergency bypass provision
**Question**: Should there be an emergency exception?
**Answer**: NO - even emergencies need verification (prevents worse breakage)

**Verification**: Skill already addresses in rationalization table ("urgent" excuse). Example 1 shows urgent bug WITH verification. No change needed.

### Edge Case 5: Dependency Between Verification Subtasks
**Scenario**: "Verify X" depends on "Run tests" completing first
**Current Skill**: Doesn't specify dependency ordering
**Risk**: Agent tries to verify before running tests
**Recommendation**: Add ordering guidance:
- "Verification subtasks run AFTER implementation and test subtasks complete"
- Order: Implementation → Tests → Build/Test Run → Security Checks → Verification (completing-*-task) → CHANGELOG

## Skill Updates Needed

### Critical Updates (MUST add):
1. ✅ Add Loophole 8 counter: "Spirit vs Letter" violation
2. ✅ Add Loophole 6 counter: "Partial verification" excuse
3. ⚠️ Clarify Edge Case 5: Verification subtask ordering

### Important Updates (SHOULD add):
4. ⚠️ Add documentation example (major vs minor)
5. ⚠️ Clarify CHANGELOG exemptions
6. ⚠️ Add manual security override guidance
7. ⚠️ Clarify read-only vs research with deliverables

### Optional Updates (COULD add):
8. ⚠️ Add more examples for different task types
9. ⚠️ Add flowchart for task classification (major vs minor)

## Refactoring Plan

1. **Add "Spirit vs Letter" statement** early in skill (after Overview)
2. **Expand rationalization table** with Loopholes 6, 7, 9
3. **Add verification subtask ordering** in "Integration with Completion Skills"
4. **Add documentation example** in "Examples in Context"
5. **Clarify CHANGELOG exemptions** in Rule 4

## Current Skill Strengths (Keep)

✅ Clear before/after TodoWrite examples
✅ Comprehensive rationalization table
✅ Red flags section for self-checking
✅ Enforcement rules make verification non-negotiable
✅ Quick reference table for task types
✅ Integration with completion skills clearly explained
✅ Multiple real-world examples

## Conclusion

**Additional loopholes found**: 5 (Loopholes 6-10)
**Edge cases identified**: 5
**Critical updates needed**: 3
**Skill quality**: GOOD, needs refinement
**Ready for updates**: YES
