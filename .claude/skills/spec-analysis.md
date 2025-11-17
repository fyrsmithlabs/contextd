# Spec Analysis Skill

**Type**: Reusable Skill
**Category**: Documentation Analysis
**Version**: 1.0.0

## Purpose

Analyzes specifications for completeness, consistency, and quality. Can be used by any agent that needs to validate or review specification documents.

## Capabilities

- Validate spec structure and required sections
- Check for missing acceptance criteria
- Identify incomplete requirements
- Verify links and references
- Detect inconsistencies
- Assess token efficiency

## Input Format

```json
{
  "spec_path": "docs/specs/feature-name.md",
  "check_types": ["structure", "completeness", "consistency", "token_efficiency"],
  "strict_mode": true
}
```

## Output Format

```json
{
  "valid": true|false,
  "score": 85,
  "issues": [
    {
      "severity": "error|warning|info",
      "type": "missing_section|incomplete|inconsistency|efficiency",
      "message": "Description of issue",
      "location": "Section name or line number",
      "suggestion": "How to fix"
    }
  ],
  "summary": {
    "structure_score": 90,
    "completeness_score": 80,
    "consistency_score": 85,
    "token_efficiency_score": 85
  }
}
```

## Validation Checklist

### Structure (Required Sections)

- [ ] Title and overview
- [ ] Purpose/Objective
- [ ] Scope (In/Out)
- [ ] Architecture/Design
- [ ] Implementation tasks
- [ ] Acceptance criteria
- [ ] Related specifications

### Completeness

- [ ] All acceptance criteria are testable
- [ ] All dependencies listed
- [ ] All assumptions documented
- [ ] All risks identified
- [ ] All open questions resolved

### Consistency

- [ ] Terminology used consistently
- [ ] References are valid (no broken links)
- [ ] Architecture aligns with docs/standards/architecture.md
- [ ] Coding standards referenced correctly

### Token Efficiency

- [ ] Uses tables for structured data
- [ ] References standards instead of duplicating
- [ ] Uses progressive disclosure (<details> tags)
- [ ] Avoids redundant examples
- [ ] Links to external docs instead of embedding

## Usage Examples

### Example 1: Basic Validation

```
Input:
  spec_path: "docs/specs/authentication.md"
  check_types: ["structure", "completeness"]
  strict_mode: true

Output:
  valid: false
  score: 75
  issues:
    - severity: "error"
      type: "missing_section"
      message: "Acceptance Criteria section is missing"
      location: "End of document"
      suggestion: "Add ## Acceptance Criteria section with testable criteria"
    - severity: "warning"
      type: "incomplete"
      message: "Only 2 of 5 implementation tasks have acceptance criteria"
      location: "## Implementation Tasks"
      suggestion: "Add acceptance criteria for remaining 3 tasks"
```

### Example 2: Token Efficiency Check

```
Input:
  spec_path: "docs/specs/user-service.md"
  check_types: ["token_efficiency"]
  strict_mode: false

Output:
  valid: true
  score: 60
  issues:
    - severity: "info"
      type: "efficiency"
      message: "Duplicates content from coding-standards.md (850 tokens)"
      location: "## Coding Standards section"
      suggestion: "Replace with: 'See docs/standards/coding-standards.md'"
    - severity: "info"
      type: "efficiency"
      message: "Could use table instead of list (40% reduction)"
      location: "## Configuration Options"
      suggestion: "Convert list to table format"
```

## Integration

### With spec-writer Agent

```
1. spec-writer creates initial specification
2. spec-writer uses spec-analysis skill to validate
3. spec-analysis returns issues
4. spec-writer fixes issues
5. spec-writer re-validates until score ≥ 85
```

### With orchestrator Agent

```
1. orchestrator receives completed spec
2. orchestrator uses spec-analysis skill for final validation
3. If score < 80, escalate to spec-writer for fixes
4. If score ≥ 80, approve for implementation
```

### With code-reviewer Agent

```
1. code-reviewer checks if spec exists for feature
2. code-reviewer uses spec-analysis to verify spec quality
3. If spec invalid/incomplete, block implementation
4. Request spec fixes before code review continues
```

## Performance Metrics

| Metric | Target | Notes |
|--------|--------|-------|
| **Validation time** | <2s | For specs up to 10KB |
| **Accuracy** | >95% | False positive rate |
| **Coverage** | 100% | All check types |

## Version History

- **1.0.0** (2025-10-25): Initial version with structure, completeness, consistency, and token efficiency checks
