# Changelog

All notable changes to contextd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
