# Changelog

All notable changes to contextd-marketplace will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `repository_search` tool schema (input/output definitions)
- `branch` field to `repository_index_input` schema
- `branch` and `collection_name` fields to `repository_index_output` schema
- `/init` command - Initialize contextd for new/existing projects with Kinney CLAUDE.md framework
- Git commit re-index protocol in `session-lifecycle` skill

### Changed
- **BREAKING**: Skills now prioritize `repository_search` over Read/Grep/Glob
  - `using-contextd`: Added "Code Search Priority" section establishing search order
  - `cross-session-memory`: Updated learning loop to search code first, then memories
  - `session-lifecycle`: Added re-index triggers after git commit and session end
- Updated `using-contextd` skill to include `repository_search` tool
- Session end protocol now includes `repository_index` before `memory_record`

## [0.1.0] - 2025-12-04

### Added
- Initial release
- MCP tool schemas for all contextd tools:
  - `checkpoint_save`, `checkpoint_list`, `checkpoint_resume`
  - `memory_search`, `memory_record`, `memory_feedback`
  - `remediation_search`, `remediation_record`
  - `repository_index`
  - `troubleshoot_diagnose`
- Skills:
  - `using-contextd` - Overview of contextd tools
  - `session-lifecycle` - Session start/end protocols
  - `cross-session-memory` - Learning loop workflow
  - `checkpoint-workflow` - Context preservation at 70%
  - `error-remediation` - Error resolution flow
- Slash commands:
  - `/search` - Search memories and remediations
  - `/remember` - Record a learning
  - `/checkpoint` - Save checkpoint
  - `/status` - Show contextd status
  - `/diagnose` - AI-powered error diagnosis
  - `/resume` - Resume from checkpoint
  - `/troubleshoot` - Error diagnosis
  - `/auto-checkpoint` - Auto-save before clear
  - `/context-check` - Check context usage
- Skill file format schema (`skill_frontmatter`, `skill_file`)
