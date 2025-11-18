# Repository Indexing Requirements

**Parent**: [../SPEC.md](../SPEC.md)

## Motivation

### Problem Statement

Developers often need to:
- Find code examples across large repositories
- Search documentation and markdown files
- Locate configuration files and patterns
- Discover where specific functionality is implemented
- Navigate unfamiliar codebases quickly

Traditional text-based search (grep, find) requires:
- Exact keyword matching
- Knowledge of specific terms
- Multiple iterations to find relevant code
- Manual filtering of irrelevant results

### Solution

Repository indexing enables:
- **Semantic search**: Find code by describing what it does, not just keywords
- **Natural language queries**: "authentication middleware" finds auth code
- **Cross-project search**: Search across multiple indexed repositories
- **Context preservation**: Search results include file paths and surrounding context
- **One-time indexing**: Index once, search repeatedly without re-scanning

### Use Cases

1. **Onboarding**: New developers quickly find relevant code examples
2. **Documentation search**: Find docs by topic without knowing exact filenames
3. **Code discovery**: Locate similar patterns across projects
4. **Architecture analysis**: Understand codebase structure through semantic exploration
5. **Knowledge base**: Index documentation, READMEs, and guides

---

## Functional Requirements

### FR1: File Tree Traversal
- **FR1.1**: Recursively walk directory structure from root path
- **FR1.2**: Follow symlinks with cycle detection
- **FR1.3**: Skip directories that match exclude patterns
- **FR1.4**: Handle permission errors gracefully

### FR2: Pattern Matching
- **FR2.1**: Support glob-style include patterns (e.g., `*.go`, `**/*.md`)
- **FR2.2**: Support glob-style exclude patterns (e.g., `node_modules/**`, `*.log`)
- **FR2.3**: Include patterns: whitelist files to index (empty = all files)
- **FR2.4**: Exclude patterns: blacklist files to skip (e.g., binaries, logs)
- **FR2.5**: Pattern matching on both basename and full path

### FR3: File Size Filtering
- **FR3.1**: Skip files exceeding max file size limit
- **FR3.2**: Default max file size: 1MB
- **FR3.3**: Configurable max file size: 0 to 10MB
- **FR3.4**: Report skipped files in debug logs

### FR4: File Content Reading
- **FR4.1**: Read text file contents as UTF-8
- **FR4.2**: Handle binary files gracefully (skip or error)
- **FR4.3**: Respect context cancellation during read operations
- **FR4.4**: Validate file paths against repository root (prevent traversal)

### FR5: Checkpoint Creation
- **FR5.1**: Create one checkpoint per indexed file
- **FR5.2**: Checkpoint summary: `"Indexed file: <relative-path>"`
- **FR5.3**: Checkpoint description: Full file contents
- **FR5.4**: Checkpoint project_path: Repository root path
- **FR5.5**: Checkpoint context: `{"indexed_file": "<relative-path>"}`
- **FR5.6**: Checkpoint tags: `["indexed", "repository", "<file-extension>"]`
- **FR5.7**: Generate vector embeddings automatically via checkpoint service

### FR6: Indexing Results
- **FR6.1**: Return total count of files indexed
- **FR6.2**: Return include patterns used
- **FR6.3**: Return exclude patterns used
- **FR6.4**: Return max file size applied
- **FR6.5**: Return timestamp when indexing completed

---

## Non-Functional Requirements

### NFR1: Performance
- **NFR1.1**: Index 1000 files in < 5 minutes (assuming 1KB average file size)
- **NFR1.2**: Respect 5-minute timeout for MCP tool operations
- **NFR1.3**: Batch embedding generation where possible
- **NFR1.4**: Minimal memory footprint (stream file contents, don't load all at once)

### NFR2: Scalability
- **NFR2.1**: Support repositories up to 10,000 files
- **NFR2.2**: Support file sizes up to 10MB
- **NFR2.3**: Handle deeply nested directory structures (>100 levels)

### NFR3: Reliability
- **NFR3.1**: Continue indexing if individual file fails
- **NFR3.2**: Log errors for skipped files
- **NFR3.3**: Return count of successfully indexed files
- **NFR3.4**: Idempotent: re-indexing creates duplicate checkpoints (by design)

### NFR4: Security
- **NFR4.1**: Validate repository path exists and is accessible
- **NFR4.2**: Prevent path traversal attacks (validate all file paths)
- **NFR4.3**: Respect file system permissions
- **NFR4.4**: Don't index sensitive files (*.env, credentials.json, etc.)

### NFR5: Observability
- **NFR5.1**: OpenTelemetry tracing for indexing operations
- **NFR5.2**: Metrics: files indexed, time taken, errors encountered
- **NFR5.3**: Log progress at regular intervals (every 100 files)
- **NFR5.4**: Structured logging with repository path and file count
