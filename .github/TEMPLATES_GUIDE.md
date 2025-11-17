# GitHub Templates Guide

This document provides guidance on using the GitHub templates for the contextd repository.

## Overview

The contextd repository uses comprehensive GitHub templates to ensure high-quality contributions and maintain consistency across issues and pull requests.

## Available Templates

### Pull Request Template

**When to use**: Automatically loaded for all pull requests

**Key sections**:
- **Summary**: Brief description of changes
- **Type of Change**: Select appropriate categories
- **Research & Design**: Link research documents for features
- **Testing**: Comprehensive test coverage documentation
- **Code Quality**: Self-review and style compliance
- **Documentation**: Keep docs in sync with code
- **Breaking Changes**: Document migrations if applicable

**Best practices**:
- Fill out all applicable sections
- Link related issues using keywords (`Closes #123`)
- Provide test evidence (paste test output)
- Include screenshots for UI changes
- Document any breaking changes thoroughly
- Check all items in the pre-review checklist

### Bug Report Template

**When to use**: Reporting bugs, crashes, or unexpected behavior

**Key information to include**:
- Clear bug description
- Complete environment details (OS, version, installation method)
- Exact reproduction steps
- Expected vs actual behavior
- Logs and error messages (use collapsible sections)
- Debugging output from `ctxd debug`

**Best practices**:
- Test on latest version first
- Search for existing bug reports
- Provide complete reproduction steps
- Sanitize sensitive information (API keys, tokens)
- Include relevant logs (not entire log files)
- Assess impact and severity accurately

### Feature Request Template

**When to use**: Proposing new features or enhancements

**Key sections**:
- **Feature Description**: What, why, and who
- **Problem Statement**: User pain point
- **Proposed Solution**: Detailed implementation approach
- **Research Requirements**: SDK evaluation checklist
- **Alternatives Considered**: Other approaches and trade-offs
- **Technical Considerations**: Complexity, security, performance

**Best practices**:
- Follow research-first workflow (see [RESEARCH-FIRST-TDD-WORKFLOW.md](../docs/RESEARCH-FIRST-TDD-WORKFLOW.md))
- Evaluate existing SDKs before proposing custom code
- Document in `docs/research/` directory
- Provide clear acceptance criteria
- Estimate complexity and timeline
- Consider breaking changes and migration paths

### Question/Help Template

**When to use**: Asking questions or requesting help

**Before creating**:
- Check [README.md](../README.md)
- Check [Getting Started](../GETTING-STARTED.md)
- Search [existing issues](https://github.com/axyzlabs/contextd/issues)
- Review [Troubleshooting Guide](../docs/TROUBLESHOOTING.md)
- Check [Discussions](https://github.com/axyzlabs/contextd/discussions)

**Best practices**:
- Describe what you're trying to accomplish
- Show what you've already tried
- Provide environment details if relevant
- Be specific with your question

### Documentation Improvement Template

**When to use**: Suggesting documentation improvements

**Types of improvements**:
- Missing information
- Incorrect/outdated content
- Unclear explanations
- Missing examples
- Poor organization
- Typos/grammar

**Best practices**:
- Reference specific documents/pages
- Describe the problem clearly
- Suggest specific improvements
- Offer to contribute the fix
- Consider who would benefit

### Security Vulnerability Template

**When to use**: Reporting security issues

**⚠️ IMPORTANT**: For sensitive vulnerabilities, use GitHub's [private security reporting](https://github.com/axyzlabs/contextd/security/advisories) instead of public issues.

**Use public template for**:
- Security hardening suggestions
- Non-exploitable improvements
- Public discussions after fixes

**Best practices**:
- Assess severity accurately (Critical, High, Medium, Low)
- Provide detailed analysis
- Include proof of concept (non-destructive)
- Suggest mitigation strategies
- Follow responsible disclosure practices

## Template Customization

### For Repository Maintainers

Templates can be customized by editing files in `.github/`:
- `PULL_REQUEST_TEMPLATE.md` - PR template
- `ISSUE_TEMPLATE/*.md` - Issue templates
- `ISSUE_TEMPLATE/config.yml` - Template configuration

### For Contributors

When templates don't fit your use case:
1. Use the closest matching template
2. Adapt sections as needed
3. Remove inapplicable sections
4. Add custom sections if necessary
5. Explain why standard template doesn't fit

## Template Workflow Integration

### Research-First Development

All feature requests must follow the research-first workflow:
1. SDK/library research
2. Documentation in `docs/research/`
3. Architecture decision (ADR if needed)
4. Review and approval
5. Implementation

See [RESEARCH-FIRST-TDD-WORKFLOW.md](../docs/RESEARCH-FIRST-TDD-WORKFLOW.md)

### Test-Driven Development

All code changes must follow TDD:
1. Write tests (red phase)
2. Implement feature (green phase)
3. Refactor code
4. Maintain ≥80% coverage

See [TDD-ENFORCEMENT-POLICY.md](../docs/TDD-ENFORCEMENT-POLICY.md)

### Persona Agent Testing

New features should be tested by persona agents:
- `@qa-engineer` - Comprehensive testing
- `@developer-user` - Workflow testing
- `@security-tester` - Security audit
- `@performance-tester` - Performance benchmarks

## Quick Reference

### Creating Issues

```bash
# Bug report
gh issue create --template bug_report.md

# Feature request
gh issue create --template feature_request.md

# Question
gh issue create --template question.md

# Documentation
gh issue create --template documentation.md

# Security (public)
gh issue create --template security.md
```

### Creating Pull Requests

```bash
# PR template automatically loaded
gh pr create --title "feat: your feature" --body ""

# Draft PR (recommended for research phase)
gh pr create --draft --title "feat: your feature"
```

## Template Maintenance

Templates are maintained by repository maintainers and updated based on:
- Community feedback
- Workflow improvements
- New project requirements
- Best practice evolution

To suggest template improvements:
1. Create an issue using the Documentation template
2. Describe the improvement
3. Provide rationale
4. Offer to contribute changes

## Support

If you have questions about templates:
- Ask in [GitHub Discussions](https://github.com/axyzlabs/contextd/discussions)
- Create a question issue
- Reference this guide

## References

- [Contributing Guide](../CONTRIBUTING.md)
- [Research-First Workflow](../docs/RESEARCH-FIRST-TDD-WORKFLOW.md)
- [TDD Policy](../docs/TDD-ENFORCEMENT-POLICY.md)
- [Bug Tracking](../CLAUDE.md#bug-tracking)
- [Testing Strategy](../CLAUDE.md#testing-strategy)
