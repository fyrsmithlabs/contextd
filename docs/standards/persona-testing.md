# Persona-Driven Testing Pattern

**Status**: Mandatory Pattern
**Purpose**: Enforce edge case discovery through realistic user personas
**When**: Before implementing any user-facing feature or flow

---

## Core Principle

**When we find an edge case, create a persona we can use to evaluate the viability of the application.**

Every feature MUST be validated against realistic personas representing different:
- Experience levels (beginner, intermediate, expert)
- Workflows (new project, existing project, enterprise, hobby)
- Technical contexts (different stacks, infrastructures, domains)
- Organizational contexts (solo, team, enterprise, client work)

---

## The Pattern

### 1. Identify Edge Cases During Design

When designing a feature, ask:
- Who will use this?
- What are their edge cases?
- What can go wrong?
- What assumptions are we making?

### 2. Create Personas for Each Edge Case

For each identified edge case, create a persona that represents that scenario.

**Persona Template**:
```markdown
## Persona: [Name]

**Role**: [Job title/role]
**Experience**: [Beginner/Intermediate/Expert]
**Context**: [Work/Personal/Client/Mixed]

### Scenario
[Describe their typical workflow]

### Pain Points
- [What frustrates them]
- [What blocks them]

### Edge Cases
- [Specific edge case this persona exposes]

### Success Criteria
- [What success looks like for this persona]
```

### 3. Test Design Against Personas

For each persona, walk through the feature:
1. Does it work for this persona?
2. Does it fail gracefully?
3. Does it create confusion?
4. Does it require manual intervention?
5. Does it violate their expectations?

### 4. Document Persona Test Results

Create test scenarios showing how the feature behaves for each persona.

---

## Persona Library

Personas are stored in `docs/personas/` and categorized by:
- **User Type**: Developer, DevOps Engineer, Team Lead, etc.
- **Context**: Personal, Enterprise, Client Work, Mixed
- **Tech Stack**: Go, Java, Python, Multi-language
- **Experience**: Beginner, Intermediate, Expert

---

## Example: Multi-Tenancy Personas

Based on our current brainstorming session, here are the personas that expose edge cases:

### Persona 1: DevOps Engineer (You)
- Works on personal projects AND work projects
- Multiple tech stacks (Go, Java, Terraform)
- Same machine for different contexts
- Edge Case: Context confusion between projects

### Persona 2: Enterprise Developer
- Work laptop (MDM-managed)
- Personal laptop (unmanaged)
- Same person, different physical systems
- Edge Case: Cross-system access control required

### Persona 3: Freelance Developer
- Multiple clients (Client A, Client B)
- NDA/confidentiality requirements
- Same tech stack across clients
- Edge Case: Tag collision, data leakage risk

### Persona 4: Hobbyist Experimenting
- Adding contextd to existing project mid-development
- No .contextd.yaml exists
- Doesn't know contextd conventions
- Edge Case: Initialization/inference required

### Persona 5: Team Lead
- Wants to share patterns with team
- Different team members have different experience levels
- Some patterns are "approved", some are experimental
- Edge Case: Pattern approval workflow, RBAC

---

## Enforcement Checklist

Before implementing any user-facing feature:

- [ ] Identified at least 3 personas affected by this feature
- [ ] Created persona documents in `docs/personas/`
- [ ] Walked through feature with each persona
- [ ] Documented test scenarios for each persona
- [ ] Identified failure modes for each persona
- [ ] Designed graceful degradation for edge cases
- [ ] Updated persona library if new edge cases discovered

---

## Integration with Development Workflow

### Brainstorming Phase
When designing a feature (using superpowers:brainstorming):
1. Identify personas during design discussion
2. Ask: "Who will use this? What are their edge cases?"
3. Create persona documents before writing code

### Implementation Phase
When implementing (using superpowers:test-driven-development):
1. Write tests for each persona scenario
2. Tests MUST cover persona edge cases
3. Implementation MUST pass all persona tests

### Review Phase
When reviewing (using superpowers:requesting-code-review):
1. Code reviewer validates against persona scenarios
2. Missing persona tests = failed review
3. New edge cases discovered = create new persona

---

## Persona Test Format

```markdown
# Persona Test: [Persona Name] - [Feature Name]

## Setup
- Persona: [Link to persona doc]
- Feature: [Feature being tested]
- Scenario: [Specific scenario]

## Test Steps
1. [Step 1]
2. [Step 2]
...

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Edge Cases Exposed
- [Edge case 1]
- [Edge case 2]

## Pass/Fail
[✓/✗] Persona test result

## Issues Found
- [Issue 1]
- [Issue 2]

## Recommendations
- [Fix 1]
- [Fix 2]
```

---

## Example Persona Test

### Persona Test: Hobbyist - Project Initialization

**Setup**:
- Persona: Hobbyist Experimenting (adding contextd mid-project)
- Feature: `.contextd.yaml` initialization
- Scenario: User tries contextd on existing 6-month-old Minecraft project

**Test Steps**:
1. User opens Minecraft project in Claude Code
2. User asks: "Help me fix this server crash"
3. Claude calls `remediation_search` MCP tool
4. MCP server checks for `.contextd.yaml`
5. File does NOT exist

**Expected Behavior**:
- MCP returns error: "not_initialized" with auto-detected config
- Claude sees error, prompts user: "Set up contextd for this project?"
- Shows detected config (Java, Spigot, k8s)
- User confirms → creates `.contextd.yaml`
- Retries original operation

**Actual Behavior** (MVP design):
- ✓ MCP detects missing config
- ✓ Returns auto-detected values
- ✓ Claude prompts user
- ✗ User doesn't understand what "tech_stack" means
- ✗ "domain" field is confusing

**Edge Cases Exposed**:
- User unfamiliar with contextd terminology
- Auto-detection might be wrong (user has to fix)
- No validation of user's changes

**Recommendations**:
- Add tooltips/examples in prompts
- Provide "common domains" suggestions
- Validate tech_stack against known values
- Allow "skip" option (use defaults)

---

## Persona Evolution

Personas are **living documents**:
- Update when new edge cases discovered
- Add new personas when new user types identified
- Archive personas if edge case is resolved
- Reference personas in ADRs and design docs

---

## Success Metrics

A feature is **persona-validated** when:
- All identified personas have test scenarios
- All persona tests pass
- All edge cases have graceful degradation
- No new edge cases discovered during review

**Failure criteria**:
- Edge case found without corresponding persona
- Persona test fails
- Feature confuses/blocks a persona

---

## References

- **Superpowers Skills**: `test-driven-development`, `brainstorming`, `requesting-code-review`
- **ADR Template**: All ADRs MUST reference affected personas
- **Feature Specs**: All specs MUST include persona scenarios

---

## Next Steps

1. Create initial persona library in `docs/personas/`
2. Add persona validation to code review checklist
3. Integrate persona tests into CI/CD (future)
4. Train all contributors on persona pattern
