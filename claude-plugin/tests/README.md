# Skill Testing

Testing framework for contextd-marketplace skills using TDD principles applied to process documentation.

## Two Testing Approaches

| Approach | When to Use | Method |
|----------|-------------|--------|
| **Pressure Testing** | Discipline-enforcing skills | Subagent scenarios with 3+ combined pressures |
| **Evaluations** | Reference skills | JSON test cases with expected behaviors |

## Skill Classification

| Skill | Type | Test Method |
|-------|------|-------------|
| `using-contextd` | Reference | Evaluations |
| `cross-session-memory` | Discipline | Pressure tests |
| `checkpoint-workflow` | Discipline | Pressure tests |
| `error-remediation` | Discipline | Pressure tests |
| `session-lifecycle` | Discipline | Pressure tests |

## Running Pressure Tests

### Method 1: Task Tool with Subagent

```
Use the Task tool with subagent_type="general-purpose":

prompt: |
  [Read the contents of tests/pressure-scenarios/<skill>/<scenario>.md]

  You have access to the <skill> skill. Answer the scenario.
```

### Method 2: Slash Command

```
/contextd:test-skill cross-session-memory scenario-1
```

### Interpreting Results

**Pass:** Agent chooses correct option AND cites skill sections

**Fail:** Agent rationalizes wrong choice. Document rationalization verbatim.

**Meta-test:** After failure, ask: "How could the skill be clearer?"

## Running Evaluations

Evaluations in `tests/evaluations/` define expected behaviors:

```json
{
  "skills": ["contextd:cross-session-memory"],
  "query": "Implement a caching layer",
  "expected_behavior": [
    "Calls memory_search before starting",
    "Reviews returned memories",
    "Provides memory_feedback"
  ]
}
```

Run by giving query to fresh subagent with skill loaded. Check all behaviors occur.

## TDD Cycle for Skills

| Phase | Action | Artifact |
|-------|--------|----------|
| **RED** | Run scenario WITHOUT skill | Baseline rationalizations |
| **GREEN** | Run WITH skill, verify compliance | Passing test |
| **REFACTOR** | Find new rationalizations, plug holes | Updated skill |

## Directory Structure

```
tests/
├── README.md                    # This file
├── pressure-scenarios/
│   ├── template.md              # Scenario template
│   ├── cross-session-memory/
│   │   ├── scenario-1.md        # Time + Confidence
│   │   ├── scenario-2.md        # Sunk cost + Exhaustion
│   │   └── scenario-3.md        # Authority + Speed
│   ├── checkpoint-workflow/
│   ├── error-remediation/
│   └── session-lifecycle/
└── evaluations/
    ├── using-contextd.json
    ├── cross-session-memory.json
    └── ...
```

## Creating New Pressure Scenarios

See `pressure-scenarios/template.md` for required elements:
- 3+ combined pressures
- Concrete A/B/C options
- Realistic constraints (times, file paths)
- Expected correct answer
- Rationalizations to watch for
