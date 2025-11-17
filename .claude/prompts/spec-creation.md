# Spec Creation Prompt

Create a technical specification for issue #{{ issue_number }} in {{ repository }}.

## Process
1. **Read Issue**: `gh issue view {{ issue_number }} --repo {{ repository }}`
2. **Create Structure**: `docs/specs/{{ feature_name }}/` with SPEC.md, research/, decisions/, resolutions/
3. **Write SPEC.md** with: Overview, Requirements, Architecture, Implementation Plan, Acceptance Criteria
4. **Research**: Document SDK evaluations and technology decisions
5. **Create Issues**: Break down into implementation tasks with GitHub MCP
6. **Update Parent**: Add `status:spec-ready` label

## Requirements
- Follow `docs/standards/architecture.md` patterns
- Research SDKs per `docs/RESEARCH-FIRST-POLICY.md`
- ≥80% test coverage requirement
- Use existing codebase patterns

## Output
Complete specification and implementation-ready issues.
