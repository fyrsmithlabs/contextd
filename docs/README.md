# Documentation Index

## Getting Started

| Document | Description |
|----------|-------------|
| [Main README](../README.md) | Installation, quick start, and feature overview |
| [Onboarding Guide](ONBOARDING.md) | Guided tutorial for new users |
| [Docker Setup](DOCKER.md) | Running contextd in Docker containers |

## Configuration

| Document | Description |
|----------|-------------|
| [Configuration Reference](configuration.md) | Environment variables, embedding models, data persistence |
| [Hook Setup](HOOKS.md) | Claude Code lifecycle hook integration |

## Architecture

| Document | Description |
|----------|-------------|
| [Architecture Overview](architecture.md) | Component design and data flow |
| [Qdrant Implementation](QDRANT_IMPLEMENTATION.md) | Qdrant vector database integration details |
| [Temporal Workflows](TEMPORAL_WORKFLOWS.md) | Automation workflows for plugin validation |

## API

| Document | Description |
|----------|-------------|
| [MCP Tools](api/mcp-tools.md) | MCP tool definitions and usage |
| [Error Codes](api/error-codes.md) | Error code reference |

## Guides

| Document | Description |
|----------|-------------|
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Multi-Agent Code Review](workflows/multi-agent-code-review.md) | Consensus review workflow |
| [Versioning](VERSIONING.md) | Version scheme and compatibility |
| [Releasing](RELEASING.md) | Release process and checklist |

## Features

| Document | Description |
|----------|-------------|
| [Fallback Storage](features/fallback-storage.md) | Local fallback when remote vectorstore is unavailable |
| [Secret Scrubbing Limitation](issues/secret-scrubbing-limitation.md) | Known limitation in secret detection |

## Operations

| Document | Description |
|----------|-------------|
| [Alerting](operations/ALERTING.md) | Alert configuration and thresholds |
| [Metadata Health Monitoring](operations/METADATA_HEALTH_MONITORING.md) | Vectorstore metadata health checks |
| [Metadata Recovery](operations/METADATA_RECOVERY.md) | Recovery procedures for corrupt metadata |

## Testing (Contributor)

| Document | Description |
|----------|-------------|
| [Testing Overview](testing/README.md) | Test strategy and structure |
| [Test Architecture](testing/ARCHITECTURE.md) | Test framework design |
| [Running Tests](testing/RUNNING_TESTS.md) | How to run the test suites |
| [Test Suites](testing/TEST_SUITES.md) | Available test suites and coverage |
| [Chromem Testing](testing/CHROMEM_TESTING.md) | Chromem-specific test patterns |
| [Semantic Similarity](testing/semantic-similarity.md) | Semantic similarity test methodology |

## Migration (Contributor)

| Document | Description |
|----------|-------------|
| [Payload Filtering Migration](migration/payload-filtering.md) | Migrating to payload-based tenant isolation |

## Specifications (Contributor)

Detailed technical specifications live in [`spec/`](spec/). Key areas:

- [Collection Architecture](spec/collection-architecture/SPEC.md) - Vector collection design
- [Config](spec/config/SPEC.md) - Configuration system specification
- [Context-Folding](spec/context-folding/SPEC.md) - Branch/return isolation
- [Conversation Indexing](spec/conversation-indexing/SPEC.md) - Conversation search
- [Installation](spec/installation/SPEC.md) - Install flow specification
- [Logging](spec/logging/SPEC.md) - Structured logging specification
- [Observability](spec/observability/SPEC.md) - Metrics, tracing, alerts
- [ONNX Auto-Download](spec/onnx-auto-download/SPEC.md) - Runtime download flow
- [Reasoning Bank](spec/reasoning-bank/SPEC.md) - Memory system specification
- [Vector Storage](spec/vector-storage/SPEC.md) - Storage layer specification

## Internal (Contributor)

| Document | Description |
|----------|-------------|
| [Agents Guide](AGENTS.md) | Multi-agent orchestration patterns |
| [1.0 Release Gaps](1.0-RELEASE-GAPS.md) | Tracked gaps for 1.0 release |
| [Tier-0 Injection](TIER-0-INJECTION.md) | Tier-0 context injection design |
| [Issue 15 Completion](ISSUE_15_COMPLETION.md) | Issue 15 tracking document |

## Design Plans (Contributor)

Historical design documents live in [`plans/`](plans/). These capture the reasoning behind architectural decisions.

## Archive (Contributor)

Superseded documents live in [`archive/`](archive/).
