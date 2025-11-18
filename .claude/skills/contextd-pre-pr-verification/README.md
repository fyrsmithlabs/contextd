# contextd:pre-pr-verification Skill

## Purpose

Comprehensive pre-PR verification skill that catches issues BEFORE code review, saving review cycles and CI time.

## When to Use

- Before creating pull request
- Before invoking `contextd:code-review` skill
- Before requesting human code review
- After completing implementation work

## What It Does

Runs mandatory 7-section verification checklist:
0. Pre-commit hooks (security critical)
1. Build & test (all tests pass, ≥80% coverage)
2. Code quality (format, lint, vet, staticcheck, gosec)
3. Documentation (CHANGELOG, godoc, specs)
4. Standards compliance (naming, errors, context, security)
5. Verification evidence (completion skill usage)
6. Git hygiene (commit format, branch status)

## Key Features

- **Iron Rule**: NEVER create PR without running ALL checks
- **No Exceptions**: Applies regardless of authority, size, time pressure
- **Rationalization Table**: Counters 9 common excuses for skipping verification
- **Red Flags**: Self-check list for recognizing rationalization
- **Anti-Patterns**: Examples of what NOT to do
- **Quick Script**: Automated verification helper (bash)
- **Structured Output**: Template for reporting results

## Development Methodology

**Created using TDD for skills** (RED-GREEN-REFACTOR):

### RED Phase
- Created 5 pressure scenarios (time + confidence + authority + sunk cost + exhaustion)
- Predicted 7 rationalization categories based on established patterns
- Documented baseline failures in `test-scenarios.md` and `baseline-analysis.md`

### GREEN Phase
- Wrote skill addressing all predicted rationalizations
- Validated coverage against all 5 scenarios
- Documented validation in `green-phase-validation.md`

### REFACTOR Phase
- Identified 2 additional loopholes during validation
- Added explicit counters (red flags + anti-pattern)
- Final loophole count: 9 closed

## Testing Results

**Pressure Scenarios**: 5/5 countered
**Rationalization Categories**: 7/7 addressed
**Specific Excuses**: 9 mapped to reality
**Loopholes Closed**: 9 total
**Enforcement Mechanisms**: 6 deployed

See `quality-checks.md` for complete validation.

## Files

- `SKILL.md` - Main skill file (1914 words)
- `README.md` - This file
- `test-scenarios.md` - RED phase pressure scenarios
- `baseline-analysis.md` - RED phase rationalization predictions
- `green-phase-validation.md` - GREEN phase coverage analysis
- `quality-checks.md` - Final quality validation and deployment checklist

## Integration

**Workflow sequence**:
1. Complete implementation
2. Use `contextd:completing-major-task` or `contextd:completing-minor-task`
3. **Use this skill** ← YOU ARE HERE
4. If READY: Use `contextd:code-review`
5. If APPROVED: Create pull request

## Success Criteria

Skill is effective when agents:
- Always run verification before PR creation
- Never rationalize skipping checks
- Provide structured output with actual command results
- Block PR creation when verification fails
- Fix issues immediately rather than deferring

## Maintenance

Update this skill when:
- New verification tools added to project
- New rationalization patterns discovered
- Standards change (update Section 4)
- New anti-patterns identified

## License

Part of contextd project. See project root for license.
