# Version Management Automation

Automate version synchronization between plugin.json, CHANGELOG.md, and binary version strings. Currently these can drift causing confusion about what version is actually running.

## Rationale
Version inconsistency between plugin.json, CHANGELOG, and binary creates user confusion and support burden (noted in technical debt). Automated version management prevents drift and simplifies releases. Addresses technical debt item.

## User Stories
- As a user, I want to know what version I'm running so that I can report issues accurately
- As a maintainer, I want automated version updates so that releases don't have drift
- As a support person, I want consistent versions so that I can troubleshoot effectively

## Acceptance Criteria
- [ ] Single source of truth for version (e.g., VERSION file or git tag)
- [ ] Build process injects version into binary
- [ ] Release workflow updates CHANGELOG and plugin.json
- [ ] CI fails if versions are out of sync
- [ ] Version is visible in MCP server status response
