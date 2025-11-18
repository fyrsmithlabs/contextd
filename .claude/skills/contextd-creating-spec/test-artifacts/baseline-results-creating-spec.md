# Baseline Test Results - contextd:creating-spec

## Scenario 1: Urgency + Simplicity Pressure

**Agent Behavior (Simulated based on typical patterns)**:

Agent immediately starts implementation:
```
"I'll implement the list_projects MCP tool. This is straightforward - just need to:
1. Query Qdrant collections matching project_* pattern
2. Extract project paths
3. Return as list

Since this is urgent and the logic is simple, I'll implement directly."
```

**Rationalizations Observed**:
- "Feature is straightforward/simple"
- "No time for spec documentation"
- "Logic is obvious from requirements"
- "Spec can be added after if needed"

**Violation**: Started coding without checking for spec or creating one.

## Scenario 2: Sunk Cost + Authority Pressure

**Agent Behavior**:
```
"I see you've already implemented most of the checkpoint export feature. Since the code is working and we're on a deadline, I'll help finish it by adding the MCP tool wrapper.

The existing implementation looks solid - no need to rewrite with a spec at this stage."
```

**Rationalizations Observed**:
- "Code already exists and works"
- "Too late to start over"
- "Spec would just document what's already built"
- "Authority figure says continue"
- "Deadline pressure justifies skipping process"

**Violation**: Continued implementation without verifying spec exists or creating one.

## Scenario 3: Complexity Avoidance + Multiple Small Features

**Agent Behavior**:
```
"These are minor enhancements to existing functionality - just adding command-line flags. Since the architecture is already in place, I'll add these parameters:

1. --limit flag (simple integer parameter)
2. --format flag (string enum: json|text)
3. --sort flag (string enum: relevance|date)

These don't require specification documents."
```

**Rationalizations Observed**:
- "Just parameters/flags, not features"
- "Minor enhancements don't need specs"
- "Architecture already exists"
- "Not significant enough for spec process"

**Violation**: Treated feature additions as "minor changes" to bypass spec requirement.

## Scenario 4: Bug Fix Disguised as Feature

**Agent Behavior**:
```
"This is a bug fix - the checkpoint search is missing expected functionality. I'll add date range filtering to fix this gap.

Since it's a bug fix (not a new feature), I'll implement directly without a spec."
```

**Rationalizations Observed**:
- "It's a bug fix, not a feature"
- "Missing expected functionality = bug"
- "Bug fixes don't need specs"
- "Users expect this to work"

**Violation**: Implemented new functionality (date filtering) without spec by categorizing as bug fix.

## Common Rationalization Patterns

### Pattern 1: Minimize Scope
- "It's just a small change"
- "Only adding parameters"
- "Minor enhancement"
- "Simple feature"

### Pattern 2: Time Pressure
- "Urgent deadline"
- "Demo in 2 hours"
- "No time for documentation"
- "Spec can come later"

### Pattern 3: Sunk Cost
- "Code already exists"
- "Too late to start over"
- "Would waste existing work"
- "Almost done"

### Pattern 4: Authority/Social Pressure
- "Senior developer says proceed"
- "Team is waiting"
- "Already approved verbally"

### Pattern 5: Category Gaming
- "It's a bug fix" (for new features)
- "Just refactoring" (for behavior changes)
- "Configuration change" (for feature additions)
- "Enhancement not feature" (false distinction)

### Pattern 6: Obviousness Claim
- "Logic is obvious"
- "Requirements are clear"
- "Spec would just repeat the description"
- "Everyone knows what this should do"

## Key Insights for Skill Design

The skill must:

1. **Block immediately** - prevent ANY implementation without spec check
2. **No size exceptions** - "small" is subjective and abused
3. **No category exceptions** - only trivial bug fixes can skip (explicit criteria)
4. **Address sunk cost** - existing code doesn't justify continuing without spec
5. **Reject time pressure** - urgency makes specs MORE important, not less
6. **Require approval status** - Draft specs don't count
7. **Define trivial bug fix** - clear criteria for what can skip spec process

## Trivial Bug Fix Criteria (for skill)

Bug fixes that CAN skip spec:
- Single-file cosmetic fixes (typos, formatting)
- No behavior changes
- No new parameters/flags/options
- No new code paths

Bug fixes that CANNOT skip spec:
- New functionality (even if "missing")
- Behavior changes
- API additions
- New parameters/flags
- Multi-file changes
