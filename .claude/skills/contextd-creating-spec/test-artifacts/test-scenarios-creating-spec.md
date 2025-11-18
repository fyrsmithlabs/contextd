# Test Scenarios for contextd:creating-spec Skill

## Scenario 1: Urgency + Simplicity Pressure

**Setup**: Agent is asked to implement a simple feature with time pressure

**Prompt to Subagent**:
```
I need you to implement a new MCP tool called "list_projects" that returns all indexed projects from Qdrant. This is urgent - a demo is in 2 hours and we need this feature working.

The feature is straightforward:
1. Query Qdrant for all project databases (pattern: project_*)
2. Return list of project paths

Please implement this now.
```

**Expected Baseline Behavior (WITHOUT skill)**:
- Starts implementation immediately
- Rationalizations: "Feature is simple/obvious", "No time for spec", "Spec can be added after"

## Scenario 2: Sunk Cost + Authority Pressure

**Setup**: Agent has already started coding, receives instruction from "senior developer"

**Prompt to Subagent**:
```
I'm the senior developer on this project. I started implementing the checkpoint export feature yesterday (already have 150 lines of code working). We need to finish this today for the release.

The feature exports checkpoints to JSON files. I've already implemented:
- JSON serialization
- File writing logic
- Basic error handling

Can you help me finish the implementation by adding the MCP tool wrapper?
```

**Expected Baseline Behavior (WITHOUT skill)**:
- Continues with existing code
- Rationalizations: "Code already exists", "Spec would just document what's built", "Too late to start over"

## Scenario 3: Complexity Avoidance + Multiple Small Features

**Setup**: Agent asked to implement several "minor" features

**Prompt to Subagent**:
```
I need you to add three small enhancements to the remediation system:

1. Add --limit flag to remediation search (default 10)
2. Add --format flag for JSON vs text output
3. Add --sort flag for relevance vs date sorting

These are just minor parameter additions, not major features. The architecture is already in place, just need to add these flags to the existing code.

Please implement these enhancements.
```

**Expected Baseline Behavior (WITHOUT skill)**:
- Treats as "minor changes" not requiring specs
- Rationalizations: "Just parameters", "Not new features", "Enhancements don't need specs"

## Scenario 4: Bug Fix Disguised as Feature

**Setup**: Agent asked to "fix" something that's actually a new feature

**Prompt to Subagent**:
```
There's a bug in the checkpoint system - it doesn't support filtering by date range. Users expect to be able to search for checkpoints from the last week, but there's no way to do this.

Can you fix this bug by adding date filtering to the checkpoint search?
```

**Expected Baseline Behavior (WITHOUT skill)**:
- Treats as bug fix, skips spec
- Rationalizations: "It's a bug fix", "Users expect this functionality", "Missing feature = bug"

## Testing Protocol

For each scenario:

1. **Run WITHOUT skill** - deploy subagent, capture exact rationalizations used
2. **Document failures** - what did they do wrong, exact quotes
3. **Identify patterns** - common themes in rationalizations
4. **Write skill** - address those specific rationalizations
5. **Run WITH skill** - verify compliance
6. **Find loopholes** - identify new rationalizations
7. **Refactor skill** - close loopholes
8. **Re-test** - repeat until bulletproof
