# GREEN Phase Validation - Skill Counters Rationalizations

## Validation Against Pressure Scenarios

### Scenario 1: Time + Confidence + CI Dependency
**Rationalizations**:
- "User is waiting for PR"
- "Already manually tested"
- "CI will catch issues anyway"

**Skill Counters**:
✅ **Iron Rule**: "NEVER create PR without running ALL verification checks. NO EXCEPTIONS."
✅ **Rationalization Table**: "CI will catch any issues" → "Catch locally, save CI cycles"
✅ **Rationalization Table**: "I already manually tested it" → "Manual testing ≠ comprehensive verification"
✅ **Red Flags**: "Just need to create the PR quickly" listed
✅ **Time Comparison**: "Verification takes 5 minutes. Fixing broken PR takes 2 hours."

**Verdict**: ✅ COUNTERED

### Scenario 2: Sunk Cost + Small Change Rationalization
**Rationalizations**:
- "It's just a typo"
- "Obviously correct"
- "Full verification is overkill"

**Skill Counters**:
✅ **Iron Rule**: "This applies regardless of: How small the change is"
✅ **Rationalization Table**: "It's just a typo/small change" → "Even small changes need verification"
✅ **Red Flags**: "Too small to need full verification" listed
✅ **Anti-Pattern**: "Trusting Manual Testing" example

**Verdict**: ✅ COUNTERED

### Scenario 3: Authority + Exhaustion + Deadline
**Rationalizations**:
- "Senior dev said it's OK"
- "Long session, tired"
- "Need to ship today"

**Skill Counters**:
✅ **Iron Rule**: "This applies regardless of: Who approved proceeding"
✅ **Rationalization Table**: "Senior dev said skip verification" → "Standards apply regardless of authority"
✅ **Red Flags**: "User/lead approved skipping checks" listed
✅ **Time Pressure Counter**: Built into rationalization table

**Verdict**: ✅ COUNTERED

### Scenario 4: Pre-commit Hook Slowness
**Rationalizations**:
- "Hooks are too slow"
- "Using --no-verify for speed"
- "This change is safe to skip hooks"

**Skill Counters**:
✅ **Section 0**: Pre-commit hooks FIRST (before all other checks)
✅ **Section 0**: "If hooks fail: Fix issues, do NOT bypass with --no-verify"
✅ **Rationalization Table**: "Pre-commit hooks are slow" → "Hooks are mandatory security layer. NEVER use --no-verify"
✅ **Red Flags**: "Using --no-verify to save time" listed
✅ **Anti-Pattern**: "Skipping Pre-commit Hooks" example

**Verdict**: ✅ COUNTERED

### Scenario 5: Flaky Tests
**Rationalizations**:
- "Test is broken, not my code"
- "Will fix test later"
- "Can merge with flaky test"

**Skill Counters**:
✅ **Section 1**: "All tests pass (no failures, no skips)"
✅ **Rationalization Table**: "Tests are flaky, not my code" → "Fix flaky tests before PR"
✅ **Rationalization Table**: "Will fix issues in follow-up PR" → "Fix before THIS PR"
✅ **Red Flags**: "Tests are flaky, merging anyway" listed
✅ **Anti-Pattern**: "Deferring Issues to Later" example

**Verdict**: ✅ COUNTERED

## Coverage Analysis

### Rationalization Categories Covered

✅ **CI Dependency** - Multiple counters
✅ **Confidence-Based Skipping** - Rationalization table + anti-pattern
✅ **Triviality Rationalization** - Iron Rule + examples
✅ **Authority Deference** - Explicit counter in Iron Rule
✅ **Tool Workarounds** - Section 0 emphasis + table entry
✅ **Deferred Action** - Multiple entries in table
✅ **Time Pressure** - Time comparison + red flags

### Enforcement Mechanisms Used

✅ **"MUST" language** - Used throughout checklist
✅ **No exceptions clause** - Iron Rule section
✅ **Blocking language** - "Cannot skip any section"
✅ **Authority override prevention** - Explicit in Iron Rule
✅ **Tool workaround prevention** - Pre-commit section
✅ **Defer prevention** - Rationalization table

## Structural Quality Checks

### Frontmatter
✅ Name: `contextd-pre-pr-verification` (letters, numbers, hyphens only)
✅ Description: Starts with "Use when..."
✅ Description: Includes specific triggers (before PR, before code review)
✅ Description: Written in third person
✅ Description: Under 500 characters

### Content Structure
✅ Overview with core principle
✅ "When to Use This Skill" section with triggers
✅ Clear Iron Rule section
✅ Mandatory checklist (7 sections, cannot skip)
✅ Output template (structured format)
✅ Quick script for automation
✅ Rationalization table
✅ Red flags list
✅ Anti-patterns with examples
✅ Integration with other skills

### CSO (Claude Search Optimization)
✅ Keywords: "PR", "code review", "verification", "build", "test", "coverage", "security"
✅ Symptoms: "before creating PR", "before requesting review"
✅ Tools mentioned: pre-commit, go build, go test, gosec, staticcheck
✅ Error patterns: Skipping verification, bypassing hooks

## REFACTOR Phase Notes

### Potential Loopholes to Monitor

1. **"Partial verification is enough"**
   - Current counter: "Cannot skip any section"
   - Strengthen: Add to red flags list

2. **"Verification template without running commands"**
   - Current counter: "Record output: Paste actual command output"
   - Strengthen: Add anti-pattern

3. **"Fixed issues but didn't re-verify"**
   - Current counter: "If changes needed: Apply them, commit, then re-run verification"
   - Sufficient as-is

### Enhancements for REFACTOR

**Loophole 1**: Add to red flags:
- "Running some checks but not all"
- "Filling template without actual commands"

**Loophole 2**: Add anti-pattern:
```markdown
### ❌ Template Without Verification
Bad: Fills output template with "✅ PASS" without running commands
Missing: Actual command execution and output
Result: False confidence, issues slip through
```

## Validation Summary

**Skill Coverage**: ✅ COMPLETE
- All 5 pressure scenarios countered
- All 7 rationalization categories addressed
- All 6 enforcement mechanisms present

**Structural Quality**: ✅ COMPLETE
- Frontmatter compliant
- CSO optimized
- Content well-organized

**Potential Improvements**: 2 loopholes identified for REFACTOR phase

**Ready for GREEN → REFACTOR transition**: YES
