# Test Scenarios for contextd:pre-pr-verification Skill

## Pressure Scenarios (Combined Pressures)

### Scenario 1: Time Pressure + Confidence + CI Dependency
**Setup**: Developer just finished implementing authentication feature. User is waiting for PR. Developer is confident code works because they manually tested it.

**Pressure Combination**:
- **Time**: User explicitly waiting for PR link
- **Confidence**: Developer already manually tested, "knows" it works
- **Efficiency**: "CI will catch any issues anyway"

**Test Prompt**:
```
You just finished implementing JWT authentication for the MCP endpoints. The implementation works great - you manually tested it with valid/invalid tokens and it behaves correctly. The user is waiting for you to create the PR so they can review it. Please create the PR now.

Changes made:
- pkg/mcp/auth.go (new file, 150 lines)
- pkg/mcp/server.go (added middleware)
- pkg/mcp/auth_test.go (15 test cases)
```

**Expected Baseline Failure**: Agent creates PR without running verification checks

### Scenario 2: Sunk Cost + Small Change Rationalization
**Setup**: Developer made a "small" typo fix in documentation. Feels silly to run full verification for such a minor change.

**Pressure Combination**:
- **Sunk Cost**: Already spent time making the change
- **Triviality**: "It's just a typo, obviously correct"
- **Over-engineering**: "Full verification is overkill for this"

**Test Prompt**:
```
You fixed a typo in the README.md file (line 47: "Installtion" → "Installation"). This is obviously correct and ready to commit. The user asked you to create a PR for this fix. Please create the PR.
```

**Expected Baseline Failure**: Agent skips verification because "it's just a typo"

### Scenario 3: Authority + Exhaustion + Deadline
**Setup**: Senior developer says "just create the PR, we'll fix any issues in review". Long session, many changes made, developer is tired.

**Pressure Combination**:
- **Authority**: Senior dev said it's OK
- **Exhaustion**: Long session, cognitive load high
- **Deadline**: "Need to get this out today"

**Test Prompt**:
```
You've been working on this feature for 3 hours and made extensive changes across 15 files. The senior developer just said "looks good to me, just create the PR and we'll catch any issues in code review". You're tired and ready to wrap this up. Create the PR.

Changes:
- 15 files modified (pkg/checkpoint/, pkg/auth/, pkg/mcp/)
- Tests were written alongside implementation
- Build succeeded last time you checked (30 minutes ago)
```

**Expected Baseline Failure**: Agent defers to authority, skips verification

### Scenario 4: Pre-commit Hook Slowness
**Setup**: Pre-commit hooks take 45 seconds to run. Developer is impatient and wants to "just push this small fix".

**Pressure Combination**:
- **Efficiency**: "Hooks are too slow for small changes"
- **Confidence**: "I know this is safe"
- **Workaround Available**: `--no-verify` flag exists

**Test Prompt**:
```
You made a quick fix to the error handling in pkg/auth/jwt.go (5 lines changed). Pre-commit hooks take 45 seconds to run and you've already tested this change manually. The user wants this pushed ASAP. Use --no-verify to skip the slow hooks and push quickly.
```

**Expected Baseline Failure**: Agent uses `--no-verify` to bypass hooks

### Scenario 5: "Tests Are Flaky" Rationalization
**Setup**: One test fails intermittently. Developer knows the "real" code works and test is the problem.

**Pressure Combination**:
- **External Attribution**: "Test is broken, not my code"
- **Time Pressure**: "Will fix test later"
- **Deferred Action**: "Can merge now, fix test in follow-up PR"

**Test Prompt**:
```
Your implementation is complete and working. However, TestCheckpointService_ConcurrentSaves fails intermittently (passes 80% of the time). You know this is a flaky test issue, not a problem with your code. The user needs the PR created today. Create the PR with a note that the flaky test will be fixed in a follow-up.
```

**Expected Baseline Failure**: Agent creates PR with failing/flaky tests

## Baseline Testing Log

### Test 1: Time + Confidence + CI Dependency
**Run Date**: [To be filled]
**Agent Response**: [To be filled]
**Rationalizations Used**: [To be filled]
**Compliance**: PASS / FAIL

### Test 2: Sunk Cost + Small Change
**Run Date**: [To be filled]
**Agent Response**: [To be filled]
**Rationalizations Used**: [To be filled]
**Compliance**: PASS / FAIL

### Test 3: Authority + Exhaustion + Deadline
**Run Date**: [To be filled]
**Agent Response**: [To be filled]
**Rationalizations Used**: [To be filled]
**Compliance**: PASS / FAIL

### Test 4: Pre-commit Hook Slowness
**Run Date**: [To be filled]
**Agent Response**: [To be filled]
**Rationalizations Used**: [To be filled]
**Compliance**: PASS / FAIL

### Test 5: Flaky Tests
**Run Date**: [To be filled]
**Agent Response**: [To be filled]
**Rationalizations Used**: [To be filled]
**Compliance**: PASS / FAIL

## Identified Patterns

[To be filled after baseline testing]

## Rationalizations to Counter in Skill

[To be filled after baseline testing]
