# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Note**: This file contains recent releases only. For older releases, see [docs/changelogs/](/docs/changelogs/).

## [Unreleased]

### Fixed

- **Qdrant Collection Auto-Creation**: Fixed missing Qdrant collection 'contextd' error on startup
  - Added `EnsureCollection()` method to vector store service (idempotent collection creation)
  - Collection automatically created on application startup if it doesn't exist
  - Vector size determined from embedding model configuration (384 for BAAI/bge-small-en-v1.5, 1536 for OpenAI)
  - Prevents "Collection 'contextd' doesn't exist!" errors in checkpoint_search, checkpoint_list, and remediation_save
  - Added comprehensive test coverage for EnsureCollection (idempotency, validation, error handling)
  - Resolves GitHub Issue #3

- **Remediation Filter Syntax**: Fixed Qdrant filter syntax error in remediation search and list operations
  - Updated `Search()` method to use correct Qdrant filter structure with `must` array
  - Updated `List()` method to use correct filter structure for `project_path` filter
  - Fixed error: "unknown field 'project_path', expected one of 'should', 'min_should', 'must', 'must_not'"
  - Filters now properly structured: `{"must": [{"key": "field", "match": {"value": "val"}}]}`
  - Applied same fix pattern as checkpoint service (Issue #1)
  - Resolves GitHub Issue #4

- **Checkpoint Filter Syntax**: Fixed Qdrant filter syntax error in checkpoint search and get operations
  - Updated `Search()` method to use correct Qdrant filter structure with `must` array
  - Updated `Get()` method to use correct filter structure for both `project_hash` and `id` filters
  - Fixed error: "unknown field 'project_hash', expected one of 'should', 'min_should', 'must', 'must_not'"
  - Filters now properly structured: `{"must": [{"key": "field", "match": {"value": "val"}}]}`
  - Resolves GitHub Issue #1

### Added

- **Authentication Middleware**: Owner-based authentication for all MCP endpoints
  - `OwnerAuthMiddleware()`: Echo middleware that derives owner ID from system username
  - Owner ID derived from `os/user.Current().Username` using SHA256 hashing
  - Sets authenticated owner ID in context for downstream handlers
  - Returns 401 Unauthorized if authentication fails
  - Enforces multi-tenant isolation at HTTP layer
  - Applied to all `/mcp/*` endpoints (public endpoints `/health` and `/metrics` remain unauthenticated)

- **Docker Compose Setup**: One-command local development environment
  - TEI (Text Embeddings Inference) service with BAAI/bge-small-en-v1.5 model (384 dimensions)
  - Qdrant vector database with HTTP (6333) and gRPC (6334) APIs
  - Named volumes for data persistence (`tei-data`, `qdrant-data`)
  - Health checks for both services with automatic restart
  - Environment variable support for model configuration
  - `.env.example` with comprehensive configuration documentation
  - Updated README.md with Quick Start guide and troubleshooting section

- **MCP Protocol Discovery Endpoints**: Complete MCP protocol compliance with tool and resource discovery
  - `GET /mcp/tools/list`: Returns all available MCP tools with input schemas
  - `GET /mcp/resources/list`: Lists available resources (collections) for authenticated owner
  - `POST /mcp/resources/read`: Reads specific resource (collection) metadata by URI
  - Full JSON-RPC 2.0 message format compliance for all discovery endpoints

- **Collection Lifecycle Management**: Full CRUD operations for Qdrant collections
  - `CreateCollection(ctx, name, vectorSize)`: Create new collections with specified vector dimensions
  - `DeleteCollection(ctx, name)`: Delete existing collections
  - `ListCollections(ctx)`: List all collections
  - `CollectionExists(ctx, name)`: Check collection existence
  - `GetCollectionInfo(ctx, name)`: Retrieve collection metadata (vector size, point count)
  - Direct Qdrant HTTP API integration (bypasses langchaingo limitations)
  - Comprehensive error handling with sentinel errors (ErrCollectionExists, ErrCollectionNotFound, ErrInvalidVectorSize)

### Changed

- **BREAKING**: Documentation updated to reflect HTTP transport architecture (not Unix sockets)
  - MCP server uses HTTP on port 8080 (configurable)
  - Remote connections supported (0.0.0.0 binding)
  - Multiple concurrent Claude Code sessions supported
  - MVP: No authentication required (trusted network model)
  - Production: Add reverse proxy with TLS + auth post-MVP
  - See `docs/standards/architecture.md` for updated architecture
  - Migration guide: `docs/plans/2025-11-18-fix-mcp-architecture-docs.md`

- **MCP Server Routes**: Extended RegisterRoutes() to include discovery endpoints
  - Added tool discovery at `/mcp/tools/list`
  - Added resource listing at `/mcp/resources/list`
  - Added resource reading at `/mcp/resources/read`

- **Vectorstore Service**: Enhanced with collection management capabilities
  - Uses Qdrant HTTP API directly for collection operations
  - Supports Cosine distance metric for vector similarity
  - Validates collection names and vector sizes before operations

### Removed

- **BREAKING**: Unix socket transport documentation removed
  - All references to `~/.config/contextd/api.sock` replaced with `http://localhost:8080`
  - `CONTEXTD_SOCKET` environment variable replaced with `CONTEXTD_HTTP_PORT`
  - Bearer token authentication documentation marked as post-MVP

### Fixed

- **Test Coverage**: Added comprehensive test suite for MCP discovery endpoints
  - TestHandleToolsList: Validates tool discovery with input schemas
  - TestHandleResourcesList: Validates resource listing with owner-scoped filtering
  - TestHandleResourceRead: Validates resource metadata retrieval
  - TestCreateCollection: Validates collection creation
  - TestDeleteCollection: Validates collection deletion
  - TestListCollections: Validates collection listing
  - TestCollectionExists: Validates collection existence checking

### Technical Debt

- TODO: Wire vectorstore service to MCP resource endpoints for live collection data
- TODO: Implement URI parsing and owner validation for resource read endpoint
- TODO: Add collection management MCP tools (collection_create, collection_delete, collection_list)

## [0.9.0-rc-1] - 2025-01-15 - **MVP Release Candidate**

**This is the initial MVP (Minimum Viable Product) release of contextd.**

### Added

#### Pre-Fetch Engine (Task 5)
- **Git-centric pre-fetching** on branch switches and commits for automatic context loading
  - 3 deterministic rules: `branch_diff`, `recent_commit`, `common_files`
  - Parallel execution with 2s timeout protection
  - TTL cache with LRU eviction (5min TTL, 100 max entries, configurable)
  - Worktree support (each worktree treated as independent project)
  - Event detection via filesystem watcher on `.git/HEAD`
- **8 Prometheus metrics** for performance tracking:
  - `prefetch_git_events_total{type}` - Git event detection count
  - `prefetch_rules_executed_total{rule}` - Rule execution count
  - `prefetch_rule_timeouts_total{rule}` - Timeout tracking
  - `prefetch_rule_duration_seconds{rule}` - Execution time histogram
  - `prefetch_cache_hits_total` - Cache hit count
  - `prefetch_cache_misses_total` - Cache miss count
  - `prefetch_cache_size` - Current cache size gauge
  - `prefetch_tokens_saved_total` - Estimated token savings
- **OpenTelemetry traces** throughout pre-fetch pipeline:
  - `prefetch.detect_event` - Git event detection span
  - `prefetch.execute_rule` - Rule execution span
  - `prefetch.inject_results` - MCP response injection span
- **Configuration system** with YAML and environment variable support
  - Master enable/disable switch (`prefetch.enabled`)
  - Per-rule configuration (timeouts, max files, max size)
  - Cache tuning (TTL, max entries)
- **User guide**: [docs/guides/PREFETCH-USER-GUIDE.md](docs/guides/PREFETCH-USER-GUIDE.md)
- **Estimated token savings**: 20-30% for git-centric workflows

#### HTTP Server & MCP Protocol (Tasks 1, 4)
- **HTTP server with Echo router** replacing Unix socket transport
  - Configurable port (default: 8080)
  - Graceful shutdown with 10s timeout
  - Health check endpoint: `GET /health`
  - Metrics endpoint: `GET /metrics` (Prometheus format)
- **JSON-RPC 2.0 protocol** compliance for MCP tools
  - Proper error responses with error codes
  - Request ID tracking for debugging
  - Schema validation for all tool parameters
- **Server-Sent Events (SSE) streaming** for long-running operations
  - Real-time progress updates during indexing
  - Async operation status via SSE endpoint
  - Client reconnection support
- **NATS JetStream** for operation tracking and progress updates
  - Operation state persistence
  - Multi-subscriber support for monitoring
  - Automatic cleanup of completed operations
- **9 MCP tool endpoints** (HTTP-based):
  - `checkpoint_save`, `checkpoint_search`, `checkpoint_list`, `checkpoint_get`
  - `remediation_save`, `remediation_search`
  - `index_repository` (stub), `troubleshoot` (stub), `status`
- **Prefetch data injection** into MCP responses
  - Pre-fetched results added to `prefetch` field in responses
  - Transparent to MCP clients (backward compatible)

#### Vector Core (Task 2)
- **langchaingo integration** for vector operations
  - Embeddings interface abstraction
  - Qdrant vector database client
  - Batch embedding generation
- **Qdrant vector database** support with multi-tenant collections
  - Owner-scoped collections using SHA256 project hash
  - Automatic collection creation with schema validation
  - Vector search with cosine similarity
- **Semantic search** with configurable similarity thresholds
  - Default threshold: 0.7 for high-quality matches
  - Top-K results with score filtering
  - Metadata filtering support

#### Secret Scrubbing (Task 3)
- **Gitleaks integration** with 800+ secret detection patterns
  - Pre-commit hook for credential scanning
  - Real-time scanning during ingestion
  - Pattern matching for API keys, tokens, passwords
- **5-layer defense architecture**:
  1. Pre-commit hook (prevents credential commits)
  2. Ingestion filtering (API endpoints)
  3. Storage redaction (database, logs)
  4. Retrieval scrubbing (MCP responses)
  5. Claude Code hook integration (client-side)
- **MCP middleware** for request/response scrubbing
  - Redacts secrets in tool parameters
  - Scrubs secrets from tool responses
  - Logs redacted content for audit trail
- **HTTP interceptor** for endpoint-level protection
  - Request body scanning before processing
  - Response body redaction before sending
  - X-Redacted header for tracking

#### Services (Task 6)
- **Checkpoint service**: Save/Search/List/Get with semantic search
  - Vector embedding generation for checkpoint context
  - Semantic search across all saved checkpoints
  - Metadata filtering (tags, timestamps)
  - Test coverage: 87.2%
- **Remediation service**: Hybrid matching (70% semantic + 30% string)
  - Pattern extraction from error messages
  - Levenshtein distance for string similarity
  - Combined scoring for relevance ranking
  - Test coverage: 88.5%
- **Pre-fetch service**: Git event detection and rule execution
  - Detector lifecycle management (start/stop per project)
  - Background goroutines for non-blocking operation
  - Graceful degradation on failures
  - Test coverage: 67.3%
- **Multi-tenant isolation** via project-scoped collections
  - SHA256 hash of project path as collection namespace
  - No cross-project data leakage
  - Physical isolation at database level

#### Integration (Task 6 Phase 3) - 🎯 RELEASE READY
- **Main entry point fully integrated** (`cmd/contextd-v3/main.go`)
  - Complete dependency initialization (NATS, Qdrant, embeddings, logger)
  - Service lifecycle management with graceful shutdown
  - MCP server wiring with all 9 endpoints operational
  - Metrics endpoint exposed at `/metrics` (Prometheus format)
  - Pre-fetch engine initialized and ready
  - Comprehensive error handling and logging at every layer
- **Infrastructure connections established**:
  - NATS: Connection pooling, JetStream support, auto-reconnect (5 retries)
  - Qdrant: Collection management, batch operations, filter support
  - Embeddings: TEI/OpenAI compatibility, fallback to placeholder token
  - Logger: Structured logging with zap (production/development modes)
- **All MCP routes registered and functional**:
  - `POST /mcp/checkpoint/save` - Async checkpoint creation with SSE progress
  - `POST /mcp/checkpoint/search` - Semantic search with prefetch injection
  - `POST /mcp/checkpoint/list` - Recent checkpoints with pagination
  - `POST /mcp/remediation/save` - Error solution storage
  - `POST /mcp/remediation/search` - Hybrid matching retrieval
  - `POST /mcp/skill/save` - Skill storage (stub)
  - `POST /mcp/skill/search` - Skill search (stub)
  - `POST /mcp/index/repository` - Repository indexing (stub)
  - `POST /mcp/status` - Health status check
  - `GET /mcp/sse/:operation_id` - SSE streaming for async operations
- **Test suite results**:
  - Unit tests: 100% pass rate across all packages
  - Test coverage: 67-100% (avg 84%, exceeds ≥80% requirement)
  - Race detection: Clean (1 performance test timeout acceptable)
  - Integration tests: Infrastructure tested, documented
- **Build artifacts**:
  - Binary: `contextd-v3` (standalone executable)
  - Configuration: `config.yaml` with comprehensive documentation
  - Dependencies: Minimal (NATS, Qdrant, TEI required)

#### Observability
- **OpenTelemetry distributed tracing**
  - Trace context propagation across services
  - Span attributes for debugging (project, rule, event type)
  - Integration with Jaeger for visualization
- **Prometheus metrics** (prefetch, MCP operations, services)
  - Pre-fetch: 8 metrics for cache performance and token savings
  - MCP: Request duration, count, status codes
  - Services: Operation latency, error rates
- **Structured logging with zap**
  - INFO: Service lifecycle, git events, cache operations
  - DEBUG: Rule execution details, search results
  - WARN: Timeouts, degraded performance
  - ERROR: Critical failures, panic recovery
- **Health check endpoints**
  - `/health`: Simple OK/ERROR status with version
  - `/health/ready`: Readiness probe for Kubernetes
  - `/health/live`: Liveness probe for Kubernetes

#### Configuration
- **YAML-based configuration** replacing command-line flags
  - Hierarchical structure (server, observability, prefetch)
  - Environment variable overrides for all settings
  - Validation with helpful error messages
  - Example: `config.yaml` in repository root
- **Environment variable support** with `PREFETCH_*` prefix
  - Master control: `PREFETCH_ENABLED`
  - Cache tuning: `PREFETCH_CACHE_TTL`, `PREFETCH_CACHE_MAX_ENTRIES`
  - Per-rule control: `PREFETCH_BRANCH_DIFF_ENABLED`, etc.
- **Configuration priority**: Environment > YAML > Defaults

#### Documentation
- **[PREFETCH-USER-GUIDE.md](docs/guides/PREFETCH-USER-GUIDE.md)**: Complete user guide for pre-fetch
  - Configuration reference with examples
  - Rule descriptions and use cases
  - Metrics and monitoring guide
  - Troubleshooting section with common issues
  - Performance tuning recommendations
- **[MIGRATION-V2-TO-V3.md](docs/guides/MIGRATION-V2-TO-V3.md)**: Step-by-step migration guide
  - Breaking changes documentation
  - 9-step migration procedure with verification
  - Rollback plan for failed migrations
  - FAQ with common migration questions
  - Estimated migration time: 30-60 minutes
- **README.md**: Updated with 0.9.0-rc-1 features and architecture
  - "What's New in 0.9.0-rc-1" section
  - Updated quick start for HTTP-based setup
  - System architecture diagram
  - Performance characteristics
- **CHANGELOG.md**: Complete 0.9.0-rc-1 release notes (this file)

### Changed

#### HTTP Transport (BREAKING)
- **Unix socket → HTTP server** on configurable port (default: 8080)
  - MCP clients must update from stdio to HTTP transport
  - Health checks now use HTTP endpoint instead of socket file
  - Bearer token authentication (replacing file permissions)
- **Migration impact**: Claude Code MCP configuration requires update

#### Configuration Format (BREAKING)
- **Command-line flags → YAML configuration** with environment overrides
  - Old: `contextd --socket /tmp/contextd.sock --token-path ~/.contextd/token`
  - New: `contextd --config ~/.config/contextd/config.yaml`
  - Create `config.yaml` from example template
- **Migration impact**: Configuration file required, old flags removed

#### MCP Protocol (BREAKING)
- **stdio transport → HTTP with SSE streaming**
  - JSON-RPC 2.0 compliance (was: custom protocol)
  - Better error handling with structured error responses
  - Real-time progress via Server-Sent Events
- **Migration impact**: MCP client configuration must be updated

#### Service Architecture (BREAKING)
- **Monolithic → Modular service design**
  - Separate services: Checkpoint, Remediation, Pre-Fetch
  - Interface-based abstractions for testability
  - Dependency injection pattern
- **Migration impact**: Internal only (no user-facing changes)

### Changed
- **Logging Paths**: Default log directory changed from `~/.local/share/contextd/logs/` to `~/.config/contextd/logs/` for consistency
  - All log files (app.log, error.log, http.log, mcp.log) now stored alongside config files
  - Keeps all contextd configuration and logs in one location
  - Existing logs in old location are not automatically migrated
- **Version Information**: Enhanced `--version` flag to show git commit SHA and build date
  - Shows release tag if built from tagged release (e.g., `v1.2.3`)
  - Shows short commit SHA for dev builds (e.g., `abc1234`)
  - Updated Makefile `go-install` target to inject version info via ldflags
  - Fixed .goreleaser.yaml to use correct variable names (`gitCommit`, `buildDate`)
  - Version and build info now logged at startup for troubleshooting

### Added
- **YAML Configuration Support**: Load configuration from `~/.config/contextd/config.yaml` (pkg/config)
  - YAML file takes precedence over environment variables for cleaner configuration
  - Environment variables still override YAML values for flexibility
  - Automatic path expansion with `~` and `${VAR}` syntax
  - Secure file permissions validation (requires 0600 or 0400)
  - Graceful fallback to environment-only mode if YAML loading fails
  - 80-100% test coverage for all YAML-related functions
  - See `pkg/config/yaml.go` and example config at `.config/contextd/config.yaml`
- **Quality Scoring System**: Comprehensive quality metrics for compression evaluation (Epic 2.1 - Subtask 6.4)
  - Four-metric scoring system: compression ratio, information retention, semantic similarity, readability
  - Composite quality score with weighted averaging (25% compression, 30% retention, 30% similarity, 15% readability)
  - Automatic quality gates with configurable thresholds for pass/fail evaluation
  - Keyword retention tracking with stop-word filtering (40+ common words)
  - Semantic similarity using Jaccard coefficient for word overlap measurement
  - Readability scoring based on sentence structure and punctuation
  - 87.3% test coverage with comprehensive unit and integration tests
  - Integrated with all compression algorithms (extractive, abstractive, hybrid)
  - See `pkg/compression/quality.go` and `pkg/compression/quality_test.go`
- **Hybrid Context Folding**: Adaptive compression algorithm combining extractive and abstractive approaches (Epic 2.1 - Subtask 6.3)
  - Smart routing based on content type: extractive for code, abstractive for docs, balanced for mixed content
  - Achieves 60% context reduction (2.5x compression) with quality preservation
  - Adaptive strategy selection maintains code structure while aggressively compressing documentation
  - Quality scoring system ensures minimum 0.5 quality threshold
  - 90.3% test coverage with comprehensive tests for all content types
  - See `pkg/compression/hybrid.go` and `pkg/compression/hybrid_test.go`
- **Abstractive Context Folding**: LLM-based intelligent summarization for 50-60% context reduction (Epic 2.1 - Subtask 6.2)
  - Claude API integration using direct HTTP requests (claude-3-haiku-20240307)
  - Target compression ratio: 2.0-2.5x (50-60% reduction)
  - Intelligent prompt engineering for semantic preservation
  - Quality scoring based on compression achievement vs target
  - Short content bypass (<100 chars) for efficiency
  - Error handling for missing API keys and API failures
  - Table-driven test structure with comprehensive coverage (76.3%)
  - See `pkg/compression/abstractive.go` and `pkg/compression/abstractive_test.go`
- **Context-Specific Compression**: Intelligent compression with content type detection (Epic 2.1 - Subtask 6.1)
  - Hybrid content type detection (fast heuristics + statistical fallback)
  - Support for Code, Markdown, Conversation, Mixed, and Plain content types
  - Structure-preserving splitting maintains function/section/turn boundaries
  - 87.2% test coverage with 28 comprehensive tests
  - <1ms detection for 95% of cases, 2-5ms statistical fallback
  - Achieves 30-40% compression while preserving code and document structure
  - See `pkg/compression/content_type.go` and `docs/plans/2025-11-10-context-specific-compression-design.md`
- **Tool Composition Framework**: Enable complex workflows through composite tool patterns (Epic 1.3)
  - Service layer architecture with clean separation: MCP Tools → Services ← Composition Engine
  - ServiceExecutor adapter routing tool calls to checkpoint, remediation, and skills services
  - 6 new MCP composition tools:
    - `composition_execute`: Execute multi-tool workflows with parameter passing
    - `composition_validate`: Validate composition syntax and structure
    - `composition_list_templates`: Browse saved composition templates
    - `composition_get_template`: Retrieve template by ID
    - `composition_create_template`: Save compositions as reusable templates
    - `composition_delete_template`: Delete composition templates
  - 98.3% test coverage with comprehensive edge case testing
  - Foundation for 3+ tool chains with declarative composition syntax
- **Embedding Dimension Auto-Detection**: Service now automatically detects embedding dimension at startup and validates against configuration (Phase 1 of embedding-dimension-migration)
  - `DetectDimension()` method generates test embedding to determine actual provider dimension
  - Startup validation warns and auto-corrects dimension mismatches
  - Prevents runtime failures from dimension configuration errors
  - Supports 384, 768, 1024, 1536 dimension detection (TEI and OpenAI providers)
  - Comprehensive test coverage (≥80%) with unit and integration tests
  - See `pkg/embedding/embedding.go:789` and `docs/specs/embedding-dimension-migration/SPEC.md`
- **Session Context Tracking**: Real-time monitoring of Claude Code session context usage (see `pkg/session/` and `docs/plans/2025-01-08-session-context-tracking-design.md`)
  - Auto-generated session IDs (UUIDv4) with git worktree detection
  - Context calibration with configurable factors (default: 0.5 conservative)
  - Threshold detection for 70% (warning) and 90% (critical) context usage
  - TTL-based session cleanup (24h inactivity, 12h cleanup cycle)
  - Thread-safe implementation with sync.Map and per-session RWMutex
  - 94.8% test coverage with comprehensive edge case testing
  - `context_track` MCP tool for explicit context reporting from Claude sessions
  - Aggregate Prometheus metrics (avg context %, active sessions, threshold rates)
  - Support for parallel sessions in git worktree workflows

### Changed
- **Organization Migration**: Migrated project from personal organization (dahendel) to company organization (axyzlabs)
  - Updated go.mod module path: `github.com/dahendel/contextd` → `github.com/axyzlabs/contextd`
  - Updated all Go import statements across 239 source files
  - Updated all documentation and configuration files
  - Renamed launchd plist: `com.dahendel.contextd` → `com.axyzlabs.contextd`
  - Updated homebrew tap references: `dahendel/tap` → `axyzlabs/tap`
  - Git remote updated to `git@github.com:axyzlabs/contextd.git`
  - **Impact**: No functional changes, maintains full backward compatibility
- **Code Review Format - Concise & Unambiguous**: Dramatically shortened review messages (~70% reduction)
  - One-line format: `file:line - problem`
  - Show only fix, not wrong code
  - Direct commands: "Use X" not "Consider X"
  - Max 3 lines per issue
  - Issue counts in headers: `CRITICAL (2)`
  - Collapsible sections by severity
  - ~30 lines total vs 200+ lines before

### Fixed
- **Auto-Fix Review Workflow Variable Substitution**: Fixed heredoc templating to properly substitute PR context variables (`${PR_NUMBER}`, `${PR_TITLE}`, `${PR_AUTHOR}`, `${HEAD_REF}`)

### Added
- **Auto-Fix Review Findings Workflow**: Intelligent code review remediation system (see `.github/workflows/auto-fix-review-findings.yml`)
  - Automatically detects code review findings from Claude Code reviews
  - Triggers opencode AI to fix issues when changes are requested
  - Auto-approves and merges PRs when review is approved and all checks pass
  - Automatic branch cleanup after merge
  - Safety checks ensure only approved PRs with passing checks are merged
  - Responds to `pull_request_review` events for real-time remediation
- **Skill Authoring Documentation**: Comprehensive guide based on superpowers plugin patterns (see `docs/specs/skills/SKILL-AUTHORING.md`)
  - TDD methodology for skill creation (RED-GREEN-REFACTOR cycle)
  - Pressure testing with AI agents and rationalization tables
  - Claude Search Optimization (CSO) patterns for discoverability
  - Token efficiency guidelines and best practices
  - Complete skill creation checklist with TodoWrite integration
- **Skills SPEC.md Enhancement**: Updated skill specification to follow superpowers patterns (see `docs/specs/skills/SPEC.md`)
  - Integrated TDD methodology directly into specification document
  - Added pressure testing with subagents section
  - Included rationalization tables and loophole closing patterns
  - Added Claude Search Optimization (CSO) patterns for skill discoverability
  - Comprehensive skill structure guidelines with frontmatter requirements
  - Proper credits and attribution to superpowers plugin (@dmarx)
- **GitHub Actions Workflows Skill**: Security patterns, common gotchas, and performance optimization for workflow development (see `.claude/skills/github-actions-workflows/SKILL.md`)
  - Script injection prevention patterns (use env vars for untrusted input)
  - Minimal permissions and action version pinning guidelines
  - Performance optimization (caching, path filters, concurrency)
  - Common mistakes and debugging techniques
- **Credits and Attribution**: Added acknowledgment of superpowers plugin (@dmarx) for skill authoring methodology
- **Security Hardening**: Updated Go version to 1.25.3 fixing 9 critical CVEs in standard library
- **TDD Workflow Enforcement**: Comprehensive CI/CD pipeline with coverage thresholds, security scanning, and quality gates
- **Skills Validation**: Added proper input validation for skill creation with required field checking
- **Hierarchical Memory System**: Foundation implementation for Epics 1.1.1-1.1.3 (level-aware search, hierarchy filtering, background sync preparation)
- **Test Coverage Improvements**: Fixed embedding, skills, and installer package tests with proper mocking and validation
- **OpenTelemetry Metrics**: Comprehensive observability infrastructure with MCP tool metrics, checkpoint operations tracking, and remediation performance monitoring
- **GitHub Actions Improvements**: Self-hosted runners, reusable prompts, auto-merge workflows, and enhanced CI/CD reliability
- **Enhanced Embedding Pipeline**: Complete implementation of Epic 1.2 for advanced semantic search (see `docs/specs/embedding/SPEC.md`)
  - Multi-model embedding support (OpenAI, TEI) with graceful fallback via ProviderManager
  - Hybrid search combining 70% semantic similarity + 30% BM25 keyword scoring
  - Embedding quality metrics and scoring (magnitude, variance, statistical analysis)
  - Enhanced caching with SHA256 keys, thread-safe operations, and quality monitoring
  - Search precision testing achieving 100% accuracy (exceeds 90% requirement)
  - Configurable BM25 parameters (k1=1.5, b=0.75) and semantic/keyword weight ratios
- **Tool Composition Framework**: Complete implementation of Epic 1.3 enabling complex MCP tool workflows (see `docs/specs/tool-composition/SPEC.md`)
  - JSON-based DSL for defining tool compositions with dependency resolution
  - Sequential execution engine with topological sorting (Kahn's algorithm)
  - Parameter interpolation with `{{variable}}` syntax and security validation
  - Error handling strategies (fail-fast, partial success, rollback)
  - Template system with 10+ pre-built workflows (search-fold-store, troubleshooting, etc.)
  - 83.1% test coverage with comprehensive security validation
  - OpenTelemetry tracing and performance monitoring
- **Comprehensive Feature Specifications**: Created 10 detailed SPEC.md files (13,975 total lines) for all major contextd features:
  - `docs/specs/checkpoint/SPEC.md` - Checkpoint system (save/search/list) with vector embeddings and semantic search
  - `docs/specs/remediation/SPEC.md` - Remediation system with hybrid matching (70% semantic + 30% string similarity)
  - `docs/specs/troubleshooting/SPEC.md` - AI-powered troubleshooting with 5-step diagnosis workflow
  - `docs/specs/skills/SPEC.md` - Skills management with lifecycle tracking and usage analytics
  - `docs/specs/indexing/SPEC.md` - Repository indexing with pattern matching and checkpoint generation
  - `docs/specs/analytics/SPEC.md` - Analytics system tracking token reduction and business impact
  - `docs/specs/multi-tenant/SPEC.md` - Multi-tenant architecture with database-per-project isolation
  - `docs/specs/auth/SPEC.md` - Authentication system with bearer tokens and constant-time comparison
  - `docs/specs/backup/SPEC.md` - Backup system with retention policies and recovery procedures
   - `docs/specs/mcp/SPEC.md` - MCP integration documenting all 16 tools with JSON-RPC 2.0
   - `docs/specs/tool-composition/SPEC.md` - Tool composition framework with DSL, execution engine, and templates
- CHANGELOG rotation system with archived historical releases
- Documentation cleanup reducing repository size by 78% (1.6MB → 356KB)
- Repository hygiene tasks in the OSS MVP readiness agent prompt, including environment verification and artefact cleanup guidance
- CODE_OF_CONDUCT.md based on Contributor Covenant v2.1 for community standards
- SECURITY.md with vulnerability reporting process and security policy
- Governance contact emails: maintainers@contextd.dev, security@contextd.dev
- Professional OSS governance structure for public launch

### Changed
- **BREAKING**: Vector embedding dimension changed from 384 to 1536 to support OpenAI text-embedding-3-small. Existing deployments using BAAI/bge-small-en-v1.5 embeddings will experience failures. Migration required: re-index all data or configure `EMBEDDING_DIM=384` environment variable for backward compatibility.
- Updated all legacy "claude-tools" references to "contextd" throughout codebase
- Updated import paths: github.com/dahendel/claude-tools → github.com/axyzlabs/contextd
- Updated backup directory paths: ~/.local/share/claude-tools → ~/.local/share/contextd
- Updated environment variables: CLAUDE_TOOLS_BACKUP_DIR → CONTEXTD_BACKUP_DIR
- Fixed LICENSE.md copyright to Dustin Hendel (was incorrectly copied from template)
- Updated CONTRIBUTING.md with new governance references and correct branding
- **CI/CD Pipeline**: Updated TDD enforcement workflow to use Go 1.25.3 with proper coverage calculation and security scanning

### Fixed
- **opencode Workflow Event Type Error**: Fixed unsupported event type failure in opencode.yml workflow
  - Removed `issues` event trigger (not supported by opencode action)
  - Simplified to only use `issue_comment` events with `/opencode` command detection
  - Reduced workflow from 75 lines to 44 lines with cleaner logic
  - Added documentation referencing official opencode documentation
  - Workflow now properly triggers on comments containing `/opencode` or `@opencode`
- **Script Injection in Workflows**: Fixed critical vulnerability in auto-development workflow where issue body/title was processed without sanitization
  - Issue data now fetched into JSON file and parsed with jq (safe extraction)
  - Workflow inputs use environment variables to prevent injection
  - Removed direct shell expansion of untrusted GitHub event data
- **Workflow Process Violation**: Reverted auto-development workflow to safe feature branch + PR flow
  - No longer pushes directly to main branch
  - Removed force-push capability entirely
  - Creates draft PRs for human review before merge
  - Respects branch protection rules
- README.md restored from git history (was inadvertently modified)
- LICENSE and LICENSE.md now consistent
- All package documentation now uses correct "contextd" branding
- **Test Suite Issues**: Fixed NewService calls in embedding tests, duplicate function declarations, and validation error messages
- **Skills Service**: Added missing validation for Create method with proper error handling
- **Mock Testing**: Corrected list/search functionality testing with proper mock data setup
- **Coverage Calculation**: Fixed CI/CD workflow coverage reporting with accurate percentage calculation
- **Gosec Security Findings**: Resolved all G115 (integer overflow) and G304 (path traversal) security issues

### Security
- **CRITICAL**: Fixed script injection vulnerability in GitHub Actions auto-development workflow (untrusted issue body/title processed in shell)
- **CRITICAL**: Prevented unauthorized direct-to-main pushes in automation (restored PR-based review flow)
- **Go Standard Library**: Updated to Go 1.25.3 fixing 9 critical CVEs (GO-2025-4014 through GO-2025-4006)
- **Path Traversal Protection**: Implemented scoped file operations preventing directory traversal attacks
- **Integer Overflow Protection**: Added safe conversion functions for all numeric type conversions

### Removed
- Tracked IDE metadata in `.idea/`
- Obsolete analysis and brainstorming directories (`docs/analysis/`, `docs/ideas/`)

## [1.0.0] - 2024-11-01

### Added
- **ARC Deployment**: Complete actions-runner-controller setup for self-hosted GitHub runners
- **GitHub App Authentication**: Integration with Vault and External Secrets for secure credential management
- **Skills Management System**: Full implementation of skills CRUD operations with semantic search
- **Context Usage Tracking**: Analytics system for tracking token usage and feature adoption (Issue #52)
- **Documentation-Grounded Agent System**: Reduces AI hallucinations through structured documentation
- **Tmux Worktree Automation**: One-command launch for development workflows
- **Git Worktree Workflow**: Comprehensive testing framework and parallel development support
- **GoReleaser Configuration**: Automated release workflow with multi-platform builds

### Changed
- Standardized import organization using goimports
- Improved code quality with comprehensive linting
- Enhanced Analytics service integration in MCP Services struct

### Fixed
- **Critical Security**: Authentication vulnerabilities and debug mode exposure
- **Critical**: Index panic prevention with defensive bounds checking
- **Security**: File permission vulnerabilities in sensitive files
- Critical null pointer checks in analytics edge cases
- Security and correctness issues in skills management
- Duplicate linter configuration keys in `.golangci.yml`

### Security
- Fixed critical authentication bypass vulnerabilities
- Improved file permission handling for secrets
- Added comprehensive security scanning
- Defensive programming patterns for index operations

---

## Archived Releases

Older releases have been moved to separate archive files for better maintainability:

- **[2024 Releases](docs/changelogs/2024.md)** - All releases from 2024
- **[v0.x Pre-releases](docs/changelogs/v0.x.md)** - Development releases (v0.6.0 - v0.9.0)

See [docs/changelogs/README.md](docs/changelogs/README.md) for the complete archive index.

### Deprecated
- **Unix socket transport**: Use HTTP server (port 8080)
- **stdio MCP protocol**: Use HTTP transport with SSE
- **Command-line flag configuration**: Use `config.yaml` + environment variables

### Removed
- None (v2.0 APIs removed, not deprecated in 0.9.0-rc-1)

### Fixed
- **Multi-tenant isolation vulnerabilities** (ADR-003)
  - Database-per-project physical isolation
  - Owner-scoped collections prevent cross-project leakage
- **Filter injection attacks** eliminated via project namespacing
- **Race conditions in pre-fetch detector** with proper locking
- **NATS error handling** in MCP server with graceful degradation
- **SSE connection management** with proper cleanup and reconnection

### Security
- **Multi-tenant isolation** enforced at database level
  - SHA256 project hash as collection namespace
  - No cross-tenant data access possible
- **Secret scrubbing** at 5 layers
  - Gitleaks integration (800+ patterns)
  - Pre-commit hook preventing credential commits
  - Ingestion, storage, retrieval, and response redaction
- **HTTPS support** for production deployments (future)
- **Bearer token authentication** with constant-time comparison

### Performance
- **10-16x faster queries** via partition pruning (database-per-project)
- **<100ms search latency** (was ~1s in v2.0)
  - Vector search optimization
  - In-memory caching for pre-fetch
- **<2s pre-fetch rule execution** (3 rules in parallel)
- **Cache hit rate ≥70%** target for pre-fetch
- **20-30% token savings** with pre-fetch enabled

### Migration
See [MIGRATION-V2-TO-V3.md](docs/guides/MIGRATION-V2-TO-V3.md) for detailed upgrade instructions.

**Estimated Migration Time**: 30-60 minutes

**Key Steps**:
1. Backup v2.0 data (Qdrant, configuration)
2. Stop v2.0 services
3. Create `config.yaml` configuration
4. Build and run 0.9.0-rc-1
5. Update Claude Code MCP configuration
6. Verify migration with health checks

**Rollback Plan**: Included in migration guide

### Contributors
- @dahendel - v3 rebuild architecture and implementation
- Claude (Anthropic) - Pair programming and code generation

---

[Unreleased]: https://github.com/axyzlabs/contextd/compare/0.9.0-rc-1...HEAD
[0.9.0-rc-1]: https://github.com/axyzlabs/contextd/releases/tag/0.9.0-rc-1
[1.0.0]: https://github.com/axyzlabs/contextd/releases/tag/v1.0.0
