# Institutional Knowledge Specification

**Feature**: Institutional Knowledge (Layer 3)
**Status**: Draft
**Created**: 2025-11-22

## Overview

Institutional Knowledge consolidates proven patterns from multiple projects into organization-wide knowledge. It enables onboarding, consistency, and collective intelligence across teams.

## User Scenarios

### P1: Developer Onboarding

**Story**: As a new developer joining a project, I want automatic context about org/team standards, so that I follow established patterns from day one.

**Acceptance Criteria**:
```gherkin
Given a new developer starting on project "contextd"
When they begin their first agent session
Then they receive a briefing containing:
  - Organization-wide patterns (confidence > 0.9)
  - Team-specific standards for "platform" team
  - Project-specific conventions for "contextd"
And briefing is <1000 tokens
And patterns are prioritized by relevance and confidence
```

**Edge Cases**:
- New org with no institutional knowledge
- Developer switching between projects
- Conflicting patterns at different scopes

### P2: Pattern Promotion

**Story**: As a team lead, I want successful patterns from one project to be available to other projects, so that we don't reinvent solutions.

**Acceptance Criteria**:
```gherkin
Given a memory "JWT refresh handling" exists in project A with confidence 0.95
And similar pattern exists in project B with confidence 0.92
When consolidation runs
Then a generalized pattern is promoted to team level
And project-specific details are stripped
And original memories link to promoted pattern
```

### P3: Cross-Team Learning

**Story**: As an org admin, I want proven patterns from multiple teams to become org-wide standards, so that the whole org benefits.

**Acceptance Criteria**:
```gherkin
Given pattern X exists in team "platform" with high confidence
And similar pattern X' exists in team "frontend" with high confidence
When patterns are present in 2+ teams
Then the pattern is promoted to org level
And a generalized description is created
And all teams can access it
```

### P2: Standards Enforcement

**Story**: As a developer, I want violations of established patterns flagged, so that I maintain consistency.

**Acceptance Criteria**:
```gherkin
Given org-wide anti-pattern "raw SQL in handlers" exists
When an agent session involves similar code
Then a warning is injected into context
And the anti-pattern explanation is provided
And the recommended alternative is shown
```

## Functional Requirements

### FR-001: Hierarchical Scoping
Knowledge MUST flow through hierarchy: Project → Team → Organization.

### FR-002: Retrieval Cascade
Searches MUST cascade from specific (project) to general (org), deduplicating results.

### FR-003: Promotion Pipeline
The system MUST detect cross-project patterns and promote them to higher scopes.

### FR-004: Generalization
Promoted patterns MUST strip project-specific details to remain applicable across contexts.

### FR-005: Briefing Generation
The system MUST generate onboarding briefings combining knowledge from all relevant scopes.

### FR-006: Manual Promotion
Administrators MUST be able to manually promote knowledge to higher scopes.

### FR-007: Scope Isolation
Lower scopes MUST NOT automatically access higher-scope knowledge without explicit retrieval.

### FR-008: Conflict Resolution
When patterns conflict across scopes, more specific scope MUST take precedence.

## Success Criteria

### SC-001: Onboarding Efficiency
New developers should reach productivity 30% faster with institutional knowledge briefings.

### SC-002: Pattern Reuse
>40% of high-confidence project patterns should be applicable to other projects.

### SC-003: Consistency
Teams using shared institutional knowledge should have >80% consistency in approach to common problems.

### SC-004: Knowledge Growth
Org-level knowledge base should grow by >10 patterns per month from promotion pipeline.
