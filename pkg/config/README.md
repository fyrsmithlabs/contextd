# Config Package

The `config` package provides application configuration management for contextd, including safe JSON config merging for Claude Code MCP server integration.

## Features

- **Safe Config Merging**: Atomic file operations with validation
- **MCP Server Management**: Add, update, remove, and retrieve MCP server configurations
- **JSON Validation**: Comprehensive validation before and after merge operations
- **Conflict Detection**: Detect and handle existing server configurations
- **File Permission Management**: Maintains secure 0600 permissions
- **Atomic Operations**: Uses temporary files and atomic replacement to prevent partial writes

## Core Components

### Configuration Loading (`config.go`)

Loads application configuration from environment variables with sensible defaults:

```go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

### Types (`types.go`)

Defines core data structures for Claude Code configuration:

- `ClaudeConfig`: Represents ~/.claude/config.json structure
- `MCPServer`: MCP server configuration (command, args, env)
- `MergeOptions`: Controls merge behavior

### Validator (`validator.go`)

Provides validation functions:

- `ValidateJSON(path string)`: Validates JSON file syntax
- `ValidateMCPServer(server MCPServer)`: Validates MCP server configuration
- `ValidateClaudeConfig(config ClaudeConfig)`: Validates complete config structure

### Merger (`merger.go`)

Provides safe config merging operations:

- `MergeMCPServer(opts MergeOptions)`: Merge/add MCP server configuration
- `RemoveMCPServer(configPath, serverName string)`: Remove MCP server
- `GetMCPServer(configPath, serverName string)`: Retrieve MCP server config

## Usage Examples

### Basic Merge

```go
import "github.com/axyzlabs/contextd/pkg/config"

opts := config.MergeOptions{
    ConfigPath: "~/.claude/config.json",
    ServerName: "contextd",
    Server: config.MCPServer{
        Command: "/usr/local/bin/contextd",
        Args:    []string{"--mcp"},
        Env: map[string]string{
            "EMBEDDING_BASE_URL": "http://localhost:8080/v1",
            "EMBEDDING_MODEL":    "BAAI/bge-small-en-v1.5",
        },
    },
    CreateIfMissing: true,
}

if err := config.MergeMCPServer(opts); err != nil {
    log.Fatal(err)
}
```

### Overwrite Existing Server

```go
opts := config.MergeOptions{
    ConfigPath: "~/.claude/config.json",
    ServerName: "contextd",
    Server: config.MCPServer{
        Command: "/usr/local/bin/contextd",
        Args:    []string{"--mcp", "--verbose"},
    },
    Overwrite: true, // Allow overwriting existing server
}

if err := config.MergeMCPServer(opts); err != nil {
    log.Fatal(err)
}
```

### Retrieve Server Configuration

```go
server, err := config.GetMCPServer("~/.claude/config.json", "contextd")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Command: %s\n", server.Command)
fmt.Printf("Args: %v\n", server.Args)
```

### Remove Server Configuration

```go
if err := config.RemoveMCPServer("~/.claude/config.json", "contextd"); err != nil {
    log.Fatal(err)
}
```

## Integration with Detector

The merger works seamlessly with the `pkg/detector` package to generate environment-specific configurations:

```go
import (
    "context"
    "github.com/axyzlabs/contextd/pkg/config"
    "github.com/axyzlabs/contextd/pkg/detector"
)

ctx := context.Background()

// Detect environment
env, err := detector.DetectAll(ctx)
if err != nil {
    log.Fatal(err)
}

// Generate MCP server config based on detected environment
mcpServer := config.MCPServer{
    Command: "/usr/local/bin/contextd",
    Args:    []string{"--mcp"},
    Env:     make(map[string]string),
}

// Configure embedding backend
if env.Embedding.Backend == "tei" {
    mcpServer.Env["EMBEDDING_BASE_URL"] = env.Embedding.BaseURL
    mcpServer.Env["EMBEDDING_MODEL"] = env.Embedding.Model
} else {
    // OpenAI API key should be set separately
    mcpServer.Env["OPENAI_API_KEY"] = "YOUR_API_KEY"
}

// Configure vector database
if env.VectorDB.Type == "qdrant" {
    mcpServer.Env["QDRANT_URL"] = env.VectorDB.URL
} else {
}

// Configure telemetry
if env.Telemetry.Enabled {
    mcpServer.Env["OTEL_EXPORTER_OTLP_ENDPOINT"] = env.Telemetry.Endpoint
    mcpServer.Env["OTEL_SERVICE_NAME"] = "contextd"
    mcpServer.Env["OTEL_ENVIRONMENT"] = "local"
}

// Merge into Claude Code config
opts := config.MergeOptions{
    ConfigPath:      "~/.claude/config.json",
    ServerName:      "contextd",
    Server:          mcpServer,
    CreateIfMissing: true,
}

if err := config.MergeMCPServer(opts); err != nil {
    log.Fatal(err)
}
```

## Safety Features

### Atomic File Operations

All file operations use a temporary file + atomic rename pattern:

1. Write to `config.json.tmp`
2. Validate the temporary file
3. Atomically rename `config.json.tmp` → `config.json`
4. Clean up temporary file on any error

This ensures the config file is never left in a partial or invalid state.

### JSON Validation

JSON is validated at multiple points:

- Before reading existing config
- After merging new configuration
- Before writing temporary file
- After writing temporary file

### Permission Management

All config files maintain 0600 permissions (owner read/write only) for security.

### Error Handling

All functions return descriptive errors with context:

```go
if err := config.MergeMCPServer(opts); err != nil {
    // Errors include context about what failed and why
    log.Printf("Failed to merge config: %v", err)
}
```

## Testing

The package includes comprehensive tests:

- **Validator Tests**: JSON validation, server validation, config validation
- **Merger Tests**: Merge scenarios, conflict handling, atomic operations
- **Example Tests**: Real-world usage examples
- **Coverage**: >80% for new code (merger.go, validator.go, types.go)

Run tests:

```bash
go test -v ./pkg/config/
```

Run with coverage:

```bash
go test -v -cover ./pkg/config/
```

## Design Decisions

### Why Atomic File Operations?

Atomic operations prevent:
- Partial writes on failure
- Invalid JSON states
- Race conditions
- Data corruption

### Why Separate Validator?

Separating validation logic:
- Makes it reusable
- Easier to test
- Clear separation of concerns
- Better error messages

### Why MergeOptions Struct?

Using a struct for options:
- Makes API more flexible
- Allows future additions without breaking changes
- Clear, self-documenting code
- Easy to set defaults

## Related Packages

- `pkg/detector`: Environment detection for generating MCP configs
- `pkg/backup`: Backup and restore config files
- `cmd/ctxd`: CLI tool that uses this package for `setup-claude` command

## References

- [PR #99: Installation Safety](https://github.com/axyzlabs/contextd/pull/99)
- [PR #100: MCP Installation](https://github.com/axyzlabs/contextd/pull/100)
- [Research-First TDD Workflow](../../docs/RESEARCH-FIRST-TDD-WORKFLOW.md)
