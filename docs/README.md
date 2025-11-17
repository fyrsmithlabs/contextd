# contextd Documentation

Complete documentation for contextd - a context-optimized API service for Claude Code.

## Quick Navigation

### 🚀 Getting Started
- [Getting Started Guide](guides/GETTING-STARTED.md) - Initial setup and configuration
- [MCP Quickstart](guides/QUICKSTART-MCP.md) - Quick MCP integration guide
- [Environment Variables](guides/ENV-VARIABLES.md) - Shell configuration (Fish/Bash/Zsh)

### 🧪 Testing
- [Testing Guide](testing/TESTING.md) - Testing workflows and commands
- [Test Suite Overview](testing/TEST-SUITE-OVERVIEW.md) - Comprehensive test documentation
- [Status Line Setup](testing/STATUS-LINE.md) - Real-time status monitoring

### 🏗️ Architecture
- [Implementation Summary](architecture/IMPLEMENTATION-SUMMARY.md) - Recent implementation details
- [MCP Implementation Complete](architecture/MCP-IMPLEMENTATION-COMPLETE.md) - MCP integration overview
- [MCP Code Review](architecture/MCP-CODE-REVIEW.md) - Code review findings
- [MCP Fixes Summary](architecture/MCP-FIXES-SUMMARY.md) - Bug fixes and improvements
- [Monitoring Integration](architecture/MONITORING-INTEGRATION-SUMMARY.md) - OpenTelemetry setup

### ✨ Features
- [Profile Management](features/PROFILE-MANAGEMENT.md) - Symlink-based profile switching
- [Security & Redaction](features/SECURITY.md) - Secret redaction and security policy

### 📚 Research
- [Research Documents Index](research/RESEARCH_DOCUMENTS_INDEX.md) - AI research document catalog
- [Research Indexing Quickstart](research/RESEARCH_INDEXING_QUICKSTART.md) - How to index documents

### 📁 Additional Resources

**In `docs/` (existing):**
- [Architecture Recommendations](ARCHITECTURE-RECOMMENDATIONS.md) - Design patterns
- [OpenTelemetry Implementation](OPENTELEMETRY-IMPLEMENTATION.md) - Observability details
- [Claude Code Integration](CLAUDE-CODE-INTEGRATION.md) - Integration guide
- [Jaeger Tracing](JAEGER-TRACING.md) - Distributed tracing setup
- [Monitoring Setup](MONITORING-SETUP.md) - Full monitoring stack
- [Research Document Indexing](RESEARCH_DOCUMENT_INDEXING.md) - Document processing
- [AI Troubleshooting Research](ai-troubleshooting-research.md) - Troubleshooting patterns
- [Security Research Report](security-research-report.md) - Security analysis

**Archive:**
- [archive/CREATE-CHECKPOINT.md](archive/CREATE-CHECKPOINT.md) - Development notes

## Documentation Structure

```
docs/
├── README.md                    # This file
├── guides/                      # Getting started guides
│   ├── GETTING-STARTED.md
│   └── QUICKSTART-MCP.md
├── testing/                     # Testing documentation
│   ├── TESTING.md
│   ├── TEST-SUITE-OVERVIEW.md
│   └── STATUS-LINE.md
├── architecture/                # Architecture & implementation
│   ├── IMPLEMENTATION-SUMMARY.md
│   ├── MCP-IMPLEMENTATION-COMPLETE.md
│   ├── MCP-CODE-REVIEW.md
│   ├── MCP-FIXES-SUMMARY.md
│   └── MONITORING-INTEGRATION-SUMMARY.md
├── features/                    # Feature documentation
│   ├── PROFILE-MANAGEMENT.md
│   └── SECURITY.md
├── research/                    # Research & AI documents
│   ├── RESEARCH_DOCUMENTS_INDEX.md
│   ├── RESEARCH_INDEXING_QUICKSTART.md
│   └── RESEARCH_SCHEMA_SUMMARY.md
└── archive/                     # Archived/historical docs
    └── CREATE-CHECKPOINT.md
```

## Key Concepts

### Context Optimization
contextd is built with **context efficiency as the primary goal**. Every design decision prioritizes minimizing token usage while maximizing functionality.

### Local-First Architecture
- Background sync to hosted cluster (when configured)
- Unix socket communication (no network exposure)
- Bearer token authentication

### MCP Integration
contextd provides 7 MCP tools for Claude Code:
1. `checkpoint_save` - Save session checkpoints
2. `checkpoint_search` - Semantic search across past work
3. `checkpoint_list` - List recent checkpoints
4. `remediation_save` - Store error solutions
5. `remediation_search` - Find similar error fixes
6. `troubleshoot` - AI-powered error diagnosis
7. `list_patterns` - Browse troubleshooting knowledge base

### Security
All data sent to OpenAI API is automatically sanitized via `pkg/security` redaction. Secrets are replaced with `[REDACTED]` before processing.

## Quick Links

- [Main README](../README.md) - Project overview
- [CLAUDE.md](../CLAUDE.md) - Claude Code instructions
- [Contributing](../CONTRIBUTING.md) - How to contribute
- [Makefile](../Makefile) - Build and test commands

## Need Help?

1. Start with [Getting Started Guide](guides/GETTING-STARTED.md)
2. For testing: [Testing Guide](testing/TESTING.md)
3. For profile management: [Profile Management](features/PROFILE-MANAGEMENT.md)
4. For security: [Security Policy](features/SECURITY.md)

## Reporting Issues

Found a bug or have a question?
- Open an issue: https://github.com/axyzlabs/contextd/issues
- Check existing docs first (use search in this README)
