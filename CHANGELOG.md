# Changelog

All notable changes to contextd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.4] - 2026-01-14

### Added
- **Histogram Metrics** - Added performance histograms for key operations
  - `contextd.memory.search_duration_seconds` - Memory search latency distribution
  - `contextd.memory.confidence` - Confidence score distribution of retrieved memories
  - `contextd.checkpoint.size_bytes` - Checkpoint size distribution
- **Compression Service** - Wired compression service into main server startup

### Changed
- **Release Pipeline** - Windows-only binary builds via goreleaser
  - Mac/Linux users should use Homebrew exclusively: `brew install fyrsmithlabs/contextd/contextd`
  - Windows users download zip from GitHub releases

### Fixed
- **Telemetry Version** - Fixed `service_version` label reporting build-time version instead of hardcoded "0.1.0"

## [0.3.3] - 2026-01-14

### Fixed
- **Telemetry Version** - Set `ServiceVersion` in telemetry config from build-time version variable

## [0.3.2] - 2026-01-14

### Fixed
- **Grafana Metrics** - Initial hotfix for missing metrics instrumentation

## [0.3.1] - 2026-01-12

### Added
- **Real LLM API Integration** (Spec 003) - Replace stub implementations with production-ready API clients
  - Anthropic/Claude API: Uses Messages API at `/v1/messages` with proper headers
  - OpenAI API: Uses Chat Completions API at `/v1/chat/completions` with bearer auth
  - Rate limiting: Token bucket limiter at ~50 req/min with burst of 5
  - Exponential backoff: Up to 3 retries with 1s, 2s, 4s backoff for 429/5xx errors
  - Secret scrubbing: All content scrubbed before sending to prevent API key leaks
  - Test coverage: 88.4% with mocked HTTP servers
- **Version Management Automation** (Spec 005) - Tooling to keep VERSION, CHANGELOG.md, and plugin.json in sync
  - `scripts/check-version-sync.sh`: Validates version consistency across files
  - `scripts/sync-version.sh`: Syncs VERSION to plugin.json
  - Makefile targets: `version-check`, `version-check-strict`, `version-sync`
  - GitHub Actions workflow: `.github/workflows/version-check.yml` for CI validation
  - HTTP status endpoint now includes version field

### Changed
- **Interface Migration Cleanup** (Spec 004) - Removed dead adapter/interface code
  - Deleted `internal/embeddings/adapter.go` (interfaces now in `embeddings/provider.go`)
  - Deleted `internal/qdrant/adapter.go` (unused legacy Qdrant adapter)
  - Deleted `internal/remediation/interfaces.go` (duplicate interfaces from before migration)

## [0.3.0] - 2026-01-06

### Added
- **Agent Policies** (Issue #46) - STRICT guardrails for agent behavior
  - New `policies` skill - defines policy schema, storage pattern, and management workflow
  - New `/policies` command - list, add, remove, stats, init subcommands (like `/plugin`)
  - Policies stored as memories with `type:policy` tags
  - Built-in recommended policies: no-secrets-in-context, test-before-fix, contextd-first, etc.
  - Policy compliance evaluation integrated into `/reflect` command
- **Conversation Indexing** (Issue #46) - Extract learnings from past Claude Code sessions
  - New `conversation-indexing` skill - guides JSONL parsing and extraction
  - Extracts remediations (error→fix), memories (learnings), and policies (corrections)
  - Secret scrubbing before processing (gitleaks patterns)
  - Deduplication against existing entries
  - Context cost warnings with batch mode option
- **Updated `/onboard` command** with `--conversations` flag
  - Index past Claude Code conversations for current project
  - `--batch` flag for offline processing (no context cost)
  - `--file={uuid}` for indexing specific conversations
- **Updated `/reflect` command** with policy compliance checking
  - `--policies` flag for policy-focused compliance report
  - Policy violation tracking with evidence
  - Policy stats update (violations/successes) after evaluation
- **Context-Folding Design** (Issue #17) - branch()/return() MCP tools for context isolation
- **Production Mode Fail-Fast** (Issue #39) - `CONTEXTD_PRODUCTION_MODE=1` environment variable
  - Provider fails to start without explicit auth acknowledgment in production mode
  - Prevents accidental deployment without security review
  - Override available via `LocalModeAcknowledged=true` or `CONTEXTD_LOCAL_MODE=1`


### Security
- **GitHub Webhook Hardening** - Production-ready security improvements for plugin validation workflow
  - Input validation: validatePREvent() prevents injection attacks on webhook data
  - XSS prevention: Markdown sanitization in PR comments (11 test cases)
  - Rate limiting: 60 requests/min per IP with token bucket algorithm
  - DoS prevention: Regex compilation moved to package level
  - Thread safety: Refactored global gitHubToken to parameter-based approach
  - Proper timeout handling: 5-minute timeout for AI agent validation vs 2-minute for API calls
  - Deleted files handling: Filter removed files before schema validation to prevent 404 errors
  - Fixed X-Forwarded-For parsing for proper IP extraction from proxies
- **Installation Security Hardening** (from persona testing)
  - Fixed ONNX library directory permissions: Changed from 0755 to 0700 for user-only access
  - Config directory already secure: Uses 0700 permissions (owner read/write/execute only)
  - Vectorstore provider validation: Clear error messages for invalid provider values

### Changed
- **Enhanced Pressure Testing** (Issue #39) - Updated reflect.md with manual pressure testing process
  - v2 roadmap note for automated testing
  - Step-by-step scenario generation guide
  - Pass/fail criteria for instruction validation

### User Experience
- **Improved Error Messages** (from persona testing with 4 developer personas)
  - Enhanced ctxd health error messages with actionable hints
  - Better guidance when HTTP server is not running
  - Clear distinction between HTTP mode and MCP mode operations
  - Persona testing results: 75% approval rate (3 of 4 personas approved)
  - Detailed testing report: PERSONA_TEST_RESULTS.md

### Documentation
- **Multi-Tenancy Documentation** (Issue #45) - Added comprehensive documentation for unified payload filtering
  - Updated `CLAUDE.md` with multi-tenancy architecture section
  - Updated `docs/architecture.md` with security model and tenant context flow
  - Updated `docs/spec/vector-storage/security.md` with TenantFromContext pattern, fail-closed behavior, and defense-in-depth layers
  - Created `docs/migration/payload-filtering.md` migration guide
  - Created `internal/vectorstore/CLAUDE.md` package documentation
  - Created `internal/reasoningbank/CLAUDE.md` package documentation
  - Updated `internal/checkpoint/CLAUDE.md` with tenant isolation section
  - Updated `internal/repository/CLAUDE.md` with tenant isolation section
  - Added multi-tenancy section to `README.md`

- **SEC-004: Session Authorization** for context-folding
  - `SessionValidator` interface with `PermissiveSessionValidator` (default, single-user) and `StrictSessionValidator` (multi-tenant)
  - `CallerID` field added to `BranchRequest` and `ReturnRequest`
  - `ErrSessionUnauthorized` error code (FOLD022) and `IsAuthorizationError()` helper
  - Authorization enforced in `BranchManager.Create()` and `BranchManager.Return()`
  - Comprehensive test coverage for all validation scenarios
  - Research document with 2025 state-of-the-art (AgentFold, ACON, Claude Agent SDK)
  - Consensus design review with 4 specialized agents (Security, Correctness, Performance, Architecture)
  - TDD implementation plan with 10 phases
  - Updated SPEC.md with security requirements (SEC-001 through SEC-005)
  - Updated ARCH.md with architectural decisions (isolation model, event pattern)
  - FR-009/FR-010: Child branch cleanup and session end cleanup requirements
- `repository-search` skill in claude-plugin
- **`collection_name` parameter for `repository_search`** - allows passing collection name directly from `repository_index` output, avoiding tenant_id derivation issues
- **CountFromCollections helper** - Extracted duplicated collection counting logic into `internal/http/counts.go`
- **OTEL gauge metrics for resource counts** (#17)
  - `contextd.checkpoint.count` - Observable gauge for checkpoint count via collection enumeration
  - `contextd.memory.count` - Observable gauge for memory count via collection enumeration
- **Prometheus `/metrics` endpoint** - Exposes OTEL metrics in Prometheus format at HTTP server
- **VectorStore added to services registry** - Enables HTTP server to count resources directly

### Fixed
- **`repository_search` collection not found** - when `repository_index` used explicit `tenant_id` but `repository_search` derived tenant_id differently (e.g., from git config), search would fail with "collection not found"
  - Added `collection_name` parameter to `repository_search` (preferred over tenant_id + project_path)
  - `repository_index` output includes `collection_name` - use this value for subsequent searches
  - Existing tenant_id + project_path derivation still works as fallback
- **Duplicate type declarations** - Extracted StatusResponse, StatusCounts, ContextStatus, CompressionStatus, MemoryStatus to `internal/http/types.go`; statusline.go now uses type aliases
- **Duplicate collection counting** - Both server.go and statusline.go now use shared `CountFromCollections()` helper
- **Nil service access** - Added nil checks before accessing Scrubber/Checkpoint/Hooks services in HTTP handlers
- **Embedder resource leak** - Added `defer embedder.Close()` in statusline direct mode
- **Error output to stderr** - Statusline errors now logged to stderr instead of stdout
- **Magic numbers replaced with constants** - Added CheckpointNameMaxLength, MaxSummaryLength, etc.
- **Statusline showing 0 counts** (#17)
  - Root cause: chromem compression mismatch (config defaulted to `compress: true` but data was uncompressed)
  - Changed chromem default to `compress: false` to match existing data
  - Statusline now correctly shows checkpoint and memory counts
- **Statusline direct mode collection counting** - Now uses VectorStore.ListCollections() instead of service queries

### Security
- **Command injection prevention** - Shell-escape paths in statusline install to prevent injection attacks
- **Shell metacharacter validation** - Added `containsShellMetacharacters()` and `isValidScriptPath()` to reject unsafe statusline concatenation
- **Path traversal prevention** - Validate settings path stays within home directory
- **Path traversal validation fix** - Check for `..` BEFORE `filepath.Clean()` (Clean removes `..` sequences, making post-clean check ineffective)
- **Secure file permissions** - Changed from 0755/0644 to 0700/0600 for settings files
- **Input validation on /threshold endpoint** - Added percent range (1-100), length limits, and path sanitization
- **UserHomeDir error handling** - Fixed unchecked `os.UserHomeDir()` errors in statusline install/uninstall
- **SEC-005: Block insecure telemetry to remote endpoints** (#17)
  - Added `isLocalEndpoint()` validation in telemetry config
  - Insecure gRPC connections now only allowed for localhost/127.0.0.1/::1
  - Remote endpoints require `insecure: false` for TLS

### Added
- **Comprehensive statusline tests** - Added `statusline_test.go` with 49 test cases covering:
  - `formatStatusline()` formatting (8 tests)
  - `getHealthIcon()` status icons (3 tests)
  - `shellEscape()` escaping (5 tests)
  - `containsShellMetacharacters()` detection (15 tests)
  - `isValidScriptPath()` validation (5 tests)
  - `fetchStatusHTTP()` HTTP fetching (4 tests)
  - Path validation and settings path security (9 tests)
- **/threshold endpoint validation tests** - Added tests for path traversal, percent range, length limits, and service unavailability

### Fixed
- **Embedding model cache path now uses `~/.config/contextd/models`** (fixes model not found when running from different directories)
  - Default cache changed from relative `./local_cache` to absolute `~/.config/contextd/models`
  - `--download-models` flag now defaults to user config directory instead of `/data/models`
  - Models downloaded via `ctxd init` or `contextd --download-models` are now found regardless of working directory
- **Telemetry health state consistency** - `healthy` now correctly set to `false` when enabled but no providers initialized
- **Telemetry degradation logging** - `setDegraded()` now logs warnings via slog instead of silently discarding errors
- **TestTelemetry atomic boolean initialization** - atomic booleans now properly initialized in test harness

### Added (continued)
- **PreCompact hook for auto-checkpoint** - saves checkpoint before context compaction
- `/contextd:install` command for guided MCP server installation (homebrew, binary, docker)

### Changed
- **Plugin restructured for marketplace distribution**
  - marketplace.json uses object format for GitHub source: `{"source": "github", "repo": "fyrsmithlabs/contextd"}`
  - All paths in plugin.json prefixed with `.claude-plugin/` for external source resolution
  - hooks.json paths use `${CLAUDE_PLUGIN_ROOT}/.claude-plugin/hooks/` format
  - MCP server installation separated from plugin (manual via `/contextd:install` or homebrew)

### Fixed
- Path resolution for external GitHub source plugins (paths resolve from repo root, not .claude-plugin/)

### Added (continued)
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
- **Plugin simplified to native binary only** (Docker variant removed)
  - Plugin uses `--mcp --no-http` flags by default for multi-session support
  - Docker documentation moved to `docs/DOCKER.md` for manual setup
- **Added `--no-http` flag** to disable HTTP server
  - Allows multiple Claude Code sessions to run contextd simultaneously
  - Resolves "address already in use" port 9090 conflicts
  - HTTP server disabled by default in plugin configuration
- **Plugin shell wrapper for auto-download**
  - `bin/contextd-wrapper.sh` downloads binary on first run from GitHub releases
  - Installs to `~/.local/bin/contextd` (or uses existing if in PATH)
  - Detects platform (darwin/linux, amd64/arm64) automatically
  - Checks if `--no-http` flag supported before using (backwards compatible)
  - **UX improvements** (consensus review findings):
    - Progress bar during download
    - Network timeouts (30s API, 60s download) prevent infinite hangs
    - Empty version validation with actionable error message
    - Consistent error messages with brew (works on macOS and Linux) + GitHub fallback
    - PATH warning when `~/.local/bin` not in PATH
    - Extracted binary validation before move
    - Signal trap cleanup (INT, TERM, EXIT)

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
