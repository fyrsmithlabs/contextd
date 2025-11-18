# contextd-planning-with-verification

## Overview

Skill that enforces systematic verification by automatically adding verification subtasks when creating TodoWrite for major work.

## Purpose

Prevents common failure mode: agents create todos for implementation but forget verification steps, leading to unverified completion claims and missing evidence.

## TDD Development Process

This skill was developed following Test-Driven Development methodology for process documentation:

### RED Phase (Baseline Testing)
- Created 3 pressure scenarios (speed, simplicity, expertise)
- Documented 5 common baseline patterns from production sessions
- Identified 5 key loopholes (clutter, batching, simplicity, implicit, exemption)

**Baseline failures documented in**: `baseline-results.md`

### GREEN Phase (Minimal Skill)
- Wrote skill addressing all 5 baseline failures
- Provided before/after TodoWrite examples
- Added rationalization table with 8 counters
- Created enforcement rules (5 rules)
- Added 3 real-world examples

**Skill verification checklist**: `verification-checklist.md`

### REFACTOR Phase (Close Loopholes)
- Identified 5 additional loopholes (partial verification, read-only tasks, spirit vs letter, research distinction, documentation classification)
- Identified 5 edge cases (multiple tasks, security detection, CHANGELOG exemptions, emergency hotfixes, subtask ordering)
- Updated skill with 3 critical refinements:
  - "Spirit vs Letter" violation counter
  - Verification subtask ordering
  - CHANGELOG exemption clarification
  - Complex documentation example
- Added 3 rationalization table entries

**Edge cases analysis**: `edge-cases-and-loopholes.md`

## CSO (Claude Search Optimization)

**Discoverability**: HIGH
- Description starts with "Use when creating TodoWrite for major work"
- Keywords: TodoWrite, verification, features, bugs, refactoring, security
- Symptom-based triggers

**Token efficiency**: 994 words (acceptable for project-specific skill with comprehensive examples)

**CSO compliance checklist**: `cso-checklist.md`

## Skill Structure

- **When to Use**: Clear triggers for major work (features, bugs, refactoring, security, multi-file)
- **Task Classification**: Major vs minor task detection rules
- **Mandatory Patterns**: 4 before/after TodoWrite examples (feature, bug fix, refactoring, minor task, complex docs)
- **Verification Templates**: JSON templates for all required subtasks
- **Detection Rules**: Keyword-based detection for major/minor/security-sensitive
- **Integration**: Clear workflow with completing-major-task, completing-minor-task, code-review skills
- **Rationalization Table**: 11 excuse/reality pairs
- **Red Flags**: 5 self-check triggers
- **Enforcement Rules**: 5 non-negotiable rules
- **Quick Reference**: Task type → verification subtasks mapping

## Testing Results

**Baseline failures addressed**: 5/5 ✅
**Loopholes closed**: 10 (5 baseline + 5 REFACTOR) ✅
**Pressure resistance**: HIGH ✅
**CSO compliance**: PASS ✅

## Files

- `SKILL.md` - Main skill (994 words)
- `README.md` - This file
- `test-scenarios.md` - RED phase pressure scenarios
- `baseline-results.md` - RED phase documented failures
- `verification-checklist.md` - GREEN phase verification
- `edge-cases-and-loopholes.md` - REFACTOR phase analysis
- `cso-checklist.md` - CSO validation

## Usage

**Invoke when**: Creating TodoWrite for any major work (features, bugs, refactoring, security changes, multi-file changes, complex documentation)

**Result**: TodoWrite will include all required verification subtasks:
- Verify [task] (completing-major-task or completing-minor-task)
- Run build and tests (≥80% coverage)
- Run security checks (if applicable)
- Update CHANGELOG.md (if applicable)

## Integration

Works with:
- `contextd:completing-major-task` - Invoked by verification subtasks for major work
- `contextd:completing-minor-task` - Invoked by verification subtasks for minor work
- `contextd:code-review` - Validates all verification evidence before PR

## Maintenance

**Update when**:
- New rationalization patterns observed in production
- New edge cases discovered
- Integration with new verification skills
- Task classification rules need refinement

## Version

**Created**: 2025-11-18
**Status**: Production-ready (TDD validated)
**Test methodology**: RED-GREEN-REFACTOR with pressure scenarios
