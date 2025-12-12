# Changelog

All notable changes to contextd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `repository-search` skill in claude-plugin
  - Documents semantic code search that finds code by meaning, not keywords
  - Covers `repository_index` and `repository_search` tool usage
  - Includes query writing tips and common mistakes
- `/contextd:help` command listing all skills and commands
- Shared error handling pattern (`_error-handling.md`) for DRY command definitions
- Conversation indexing specification (SPEC.md, DESIGN.md, SCHEMA.md, CONFIG.md)
  - Index past Claude Code sessions for semantic search
  - Extract decisions with heuristic patterns and optional LLM refinement
  - Cross-reference conversations with files and commits
  - Support for langchain-go providers (Anthropic, OpenAI/Ollama)
  - Templated configuration with Go template functions
- `secret-scrubbing` skill in claude-plugin
  - Documents PostToolUse hook integration with `ctxd scrub`
  - Explains HTTP server requirement (port 9090)
  - Provides settings.json hook configuration for Read, Bash, Grep, WebFetch tools
  - Includes troubleshooting for "connection refused" and other common issues

### Changed
- Updated `using-contextd` skill to reference new `secret-scrubbing` skill
- Added HTTP server key concept note to `using-contextd` skill
- Commands now reference shared `@_error-handling.md` for consistent error handling
- Fixed `@kinney-guide.md` import to use explicit relative path `@./kinney-guide.md`
- **Auto-checkpoint now supports meaningful summaries**
  - `POST /api/v1/threshold` accepts `summary`, `context`, and `project_path` fields
  - PreCompact hook now instructs Claude to call `checkpoint_save` with proper context
  - Checkpoint name derived from summary (first 50 chars) instead of generic "Auto-checkpoint at 70%"
  - `checkpoint-workflow` skill updated with auto-checkpoint guidance and examples

### Improved
- UX/Documentation improvements across claude-plugin skills:
  - `using-contextd`: Explicit tenant ID derivation explanation with verification command
  - `using-contextd`: Added example memory_search response with confidence scores
  - `secret-scrubbing`: Moved critical prerequisite (HTTP server) to top of skill
  - `cross-session-memory`: Added example memory_record response
  - `cross-session-memory`: Added "Understanding Confidence Scores" section
  - `checkpoint-workflow`: Added example checkpoint_save response
  - `checkpoint-workflow`: Replaced vague resume levels with concrete token counts and content descriptions
  - `session-lifecycle`: Added re-indexing efficiency clarification (idempotent, incremental, fast)

## [0.2.0-rc7] - 2025-12-09

### Added
- ONNX runtime auto-download feature
  - Automatic download of ONNX runtime v1.23.0 on first FastEmbed use
  - `ctxd init` command for explicit ONNX runtime setup
  - Platform support: linux/darwin, amd64/arm64
  - Installs to `~/.config/contextd/lib/`
  - Respects `ONNX_PATH` environment variable override
  - `--force` flag for re-download

### Fixed
- `repository_search` now uses consistent tenant ID with `repository_index` (fixes #19)
  - Both tools now use `tenant.GetTenantIDForPath()` when no tenant_id provided
  - Added regression tests for tenant ID consistency

## [0.2.0-rc4] - 2025-12-06

### Fixed
- Docker entrypoint now respects `CONTEXTD_VECTORSTORE_PROVIDER` (chromem default, qdrant optional)
- Fixed ONNX library path in Docker: `onnxruntime.so` -> `libonnxruntime.so`
- Fixed chromem `GetCollection` to always pass embedding function (prevents 401 Unauthorized errors)
- Docker container no longer starts Qdrant unnecessarily when using chromem

### Added
- Complete Phase 6 documentation: architecture.md, CONTEXTD.md, HOOKS.md

## [0.2.0-rc3] - 2025-12-05

### Fixed
- Telemetry disabled by default (no OTEL collector required)
- Config directory created if not exists

## [0.2.0-rc2] - 2025-12-05

### Changed
- Simplified release workflow: Linux amd64, macOS amd64/arm64 only
- Docker image now linux/amd64 only (removed arm64 emulation)
- All binaries built with CGO_ENABLED=1 for FastEmbed/ONNX support
- Homebrew formula now builds from source with CGO
- Auto-update Homebrew formula on release

### Removed
- Windows builds
- Linux ARM64 builds (temporarily)

## [0.2.0-alpha] - 2025-12-05

### Added
- `repository_search` MCP tool for semantic code search over indexed repositories
- chromem vectorstore provider (embedded, pure Go, zero-config)
- Factory pattern for vectorstore provider selection (`vectorstore.NewStore`)
- `CreateCollection` now accepts 0 to use store's configured default dimension
- Regression test `TestChromemStore_CreateCollection_ZeroUsesDefault`

### Changed
- **BREAKING**: Vectorstore is now chromem by default (was Qdrant)
- Checkpoint service uses `vectorstore.Store` interface instead of direct Qdrant client
- Remediation service uses `vectorstore.Store` interface instead of direct Qdrant client
- Reasoningbank service uses `vectorstore.Store` interface
- Services pass `0` to `CreateCollection` to use store's configured dimension (no more hardcoded 384)
- `cmd/contextd/main.go` simplified - uses factory, removed direct Qdrant initialization
- `repository_index` output now includes `branch` and `collection_name` fields

### Removed
- Direct Qdrant client dependency in checkpoint/remediation services
- `--qdrant-host` and `--qdrant-port` CLI flags (use config file instead)
- Hardcoded 384 dimension in service CreateCollection calls

### Fixed
- "vector size X does not match configured size Y" error when using non-384 dimension embeddings
- Services now respect embedder's actual dimension instead of assuming 384

## [0.1.5] - 2025-12-04

### Added
- HTTP server with `/api/v1/scrub`, `/api/v1/threshold`, `/api/v1/status` endpoints
- `ctxd` CLI binary for manual operations
- Lifecycle hooks for session management and auto-checkpoint

## [0.1.4] - 2025-12-03

### Added
- ReasoningBank memory package with MCP tools
- `memory_search`, `memory_record`, `memory_feedback` tools
- Confidence scoring with feedback adjustment

## [0.1.3] - 2025-12-02

### Added
- MCP server integration with simplified architecture
- Tool handlers with secret scrubbing
- gitleaks SDK integration for credential detection

## [0.1.2] - 2025-12-01

### Added
- Core services: vectorstore, embeddings, checkpoint, remediation, repository, troubleshoot
- FastEmbed local ONNX embeddings
- Qdrant gRPC client

## [0.1.1] - 2025-11-30

### Added
- Foundation: config (Koanf), logging (Zap), telemetry (OpenTelemetry)
- Entry point with stdio MCP transport

### Changed
- Migrated from `contextd-v2` repository
- Simplified architecture (removed gRPC complexity)

## [0.1.0] - 2025-11-25

### Added
- Initial project structure
- Basic MCP server skeleton
