# ctxd CLI

`ctxd` is a command-line interface for interacting with the contextd HTTP server. It provides commands for scrubbing secrets from files or stdin and checking server health.

## Installation

### Build from source

```bash
# Build just ctxd
make build-ctxd

# Build both contextd and ctxd
make build-all

# Install to GOPATH/bin
make go-install
```

### Using go install

```bash
go install github.com/fyrsmithlabs/contextd/cmd/ctxd@latest
```

## Commands

### Initialize (ONNX Runtime Setup)

Download and install the ONNX runtime library required for local embeddings.

```bash
# Initialize contextd (downloads ONNX runtime)
ctxd init

# Force re-download even if already installed
ctxd init --force
```

**Output:**
```
Downloading ONNX Runtime v1.23.0...
✓ ONNX Runtime installed to ~/.config/contextd/lib/libonnxruntime.so
```

The ONNX runtime is downloaded from GitHub releases and installed to `~/.config/contextd/lib/`. This is required for FastEmbed local embeddings.

**Skip if:**
- `ONNX_PATH` environment variable is set (uses your existing installation)
- Runtime is already installed (use `--force` to re-download)

### Scrub Secrets

Scrub secrets from files or stdin using the contextd secret scrubber.

```bash
# Scrub a file
ctxd scrub .env

# Scrub from stdin
cat output.log | ctxd scrub -

# Use a different server
ctxd scrub --server http://localhost:8080 .env

# Scrub and redirect output
ctxd scrub secrets.txt > clean.txt
```

**Output:**
- Scrubbed content is written to stdout
- If secrets were found, a summary is written to stderr: `[ctxd] Scrubbed N secret(s)`

### Health Check

Check the health status of the contextd HTTP server.

```bash
# Check health (default server)
ctxd health

# Check health on a different server
ctxd health --server http://localhost:8080
```

**Output:**
```
Server Status: ok
Server URL: http://localhost:9090
```

### Checkpoint Management

Manage session checkpoints for saving and resuming context state. Checkpoints allow you to preserve session context across interruptions or for later recovery.

#### Save a Checkpoint

Save the current session state as a checkpoint.

```bash
# Save a checkpoint with a name (minimum required)
ctxd checkpoint save --tenant-id dahendel --name "Before refactoring"

# Save with description and summary
ctxd checkpoint save \
  --tenant-id dahendel \
  --name "Feature X complete" \
  --description "Completed user authentication feature" \
  --summary "Implemented OAuth2, JWT tokens, and middleware"

# Save with context content
ctxd checkpoint save \
  --tenant-id dahendel \
  --name "Context save" \
  --context "$(cat context.txt)"

# Save with session ID
ctxd checkpoint save \
  --tenant-id dahendel \
  --session-id sess_abc123 \
  --name "Mid-session checkpoint"

# Output as JSON
ctxd checkpoint save --tenant-id dahendel --name "My checkpoint" --json
```

**Required flags:**
- `--tenant-id`: Tenant identifier (required for all checkpoint operations)
- `--name`: Checkpoint name (required)

**Optional flags:**
- `--session-id`: Session identifier
- `--description`: Detailed description of the checkpoint
- `--summary`: Brief summary of what was accomplished
- `--context`: Context content to save
- `--project-path`: Project path (defaults to current directory)
- `--project-id`: Project identifier (defaults to project path basename)
- `--team-id`: Team identifier (defaults to tenant-id)
- `--json`: Output results as JSON

**Output:**
```
Checkpoint saved successfully
ID: ckpt_8x9y0z
Name: Before refactoring
Created: 2026-01-01 10:30:45
Description: Completed user authentication feature
Summary: Implemented OAuth2, JWT tokens, and middleware
```

#### List Checkpoints

List available checkpoints for a project or session.

```bash
# List all checkpoints for current project
ctxd checkpoint list --tenant-id dahendel

# List checkpoints for a specific project path
ctxd checkpoint list --tenant-id dahendel --project-path /home/user/myproject

# List checkpoints for a specific session
ctxd checkpoint list --tenant-id dahendel --session-id sess_abc123

# List only auto-created checkpoints
ctxd checkpoint list --tenant-id dahendel --auto-only

# Limit results
ctxd checkpoint list --tenant-id dahendel --limit 10

# Output as JSON
ctxd checkpoint list --tenant-id dahendel --json
```

**Required flags:**
- `--tenant-id`: Tenant identifier

**Optional flags:**
- `--session-id`: Filter by session ID
- `--project-path`: Project path (defaults to current directory)
- `--project-id`: Project identifier (defaults to project path basename)
- `--team-id`: Team identifier (defaults to tenant-id)
- `--auto-only`: Only show auto-created checkpoints
- `--limit`: Maximum number of checkpoints to return (default: 20)
- `--json`: Output results as JSON

**Output:**
```
ID            NAME                           SESSION       CREATED           AUTO  TOKENS
ckpt_8x9y0z   Before refactoring            sess_abc123   2026-01-01 10:30        1523
ckpt_7w8x9y   Feature X complete            sess_abc123   2026-01-01 09:15        2341
ckpt_6v7w8x   Mid-session checkpoint        sess_def456   2026-01-01 08:00  yes   1876
```

#### Resume from Checkpoint

Resume session context from a saved checkpoint at different detail levels.

```bash
# Resume with context level (recommended)
ctxd checkpoint resume ckpt_8x9y0z --tenant-id dahendel --level context

# Resume with summary level (minimal tokens)
ctxd checkpoint resume ckpt_8x9y0z --tenant-id dahendel --level summary

# Resume with full state (complete context)
ctxd checkpoint resume ckpt_8x9y0z --tenant-id dahendel --level full

# Output as JSON
ctxd checkpoint resume ckpt_8x9y0z --tenant-id dahendel --json
```

**Resume levels:**
- `summary`: Only the brief summary (~20 tokens, minimal context)
- `context`: Summary + relevant context (~200 tokens, recommended)
- `full`: Complete checkpoint state (lazy load, full context)

**Required flags:**
- `<checkpoint-id>`: The checkpoint ID to resume (positional argument)
- `--tenant-id`: Tenant identifier

**Optional flags:**
- `--level`: Resume level (default: "context")
- `--project-path`: Project path (defaults to current directory)
- `--project-id`: Project identifier (defaults to project path basename)
- `--team-id`: Team identifier (defaults to tenant-id)
- `--json`: Output results as JSON

**Output:**
```
Checkpoint: Before refactoring
Description: Completed user authentication feature
Created: 2026-01-01 10:30:45
Session: sess_abc123
Token Count: 1523

--- Content (context level) ---

Implemented OAuth2 authentication flow with JWT token generation.
Added middleware for protected routes. Configured session storage.
Ready to begin refactoring user management module.
```

#### Checkpoint Workflow Example

```bash
# Start working on a feature
ctxd checkpoint save \
  --tenant-id dahendel \
  --name "Starting feature X" \
  --summary "Initial state before feature implementation"

# Work on the feature...
# Save progress checkpoint
ctxd checkpoint save \
  --tenant-id dahendel \
  --name "Feature X - authentication done" \
  --summary "OAuth2 and JWT implementation complete"

# List checkpoints to see progress
ctxd checkpoint list --tenant-id dahendel --limit 5

# Later, resume from a checkpoint
ctxd checkpoint resume ckpt_8x9y0z --tenant-id dahendel --level context
```

## Global Flags

- `--server string`: contextd server URL (default: `http://localhost:9090`)
- `--help, -h`: Help for any command
- `--version, -v`: Show version information

## Configuration

### Server URL

You can specify the server URL in three ways (in order of precedence):

1. Command-line flag: `--server http://localhost:8080`
2. Default value: `http://localhost:9090`

### Environment Variables

Currently, ctxd does not support environment variables for configuration. Use the `--server` flag instead.

## Exit Codes

- `0`: Success
- `1`: Error (connection failed, invalid input, server error, etc.)

## API Endpoints

ctxd communicates with the following contextd HTTP API endpoints:

- `POST /api/v1/scrub`: Scrub secrets from content
  - Request: `{"content": "..."}`
  - Response: `{"content": "...", "findings_count": 0}`
- `GET /health`: Check server health
  - Response: `{"status": "ok"}`

**Note**: HTTP checkpoint endpoints (`/checkpoint/save`, `/checkpoint/list`, `/checkpoint/resume`) were removed for security reasons (CVE-2025-CONTEXTD-001). Use MCP tools for checkpoint operations:
- `checkpoint_save` - Save a checkpoint
- `checkpoint_list` - List available checkpoints
- `checkpoint_resume` - Resume from a checkpoint

## Examples

### Basic Workflow

```bash
# Start the contextd HTTP server (in another terminal)
contextd --http

# Scrub a configuration file
ctxd scrub config.yaml > config.clean.yaml

# Check if the server is running
ctxd health

# Scrub command output
docker inspect container_id | ctxd scrub -
```

### Pipeline Usage

```bash
# Scrub logs before sharing
kubectl logs pod-name | ctxd scrub - > safe-logs.txt

# Scrub environment variables
env | ctxd scrub - | grep CONTEXT

# Clean up multiple files
for f in *.log; do
  ctxd scrub "$f" > "clean/$f"
done
```

## Development

### Building

```bash
go build -o ctxd ./cmd/ctxd
```

### Testing

The ctxd CLI can be tested manually:

```bash
# Test help output
./ctxd --help
./ctxd scrub --help
./ctxd health --help

# Test version
./ctxd --version

# Test with running server
echo "secret: sk-1234" | ./ctxd scrub -
```

## Troubleshooting

### Connection Refused

```
Error: failed to send request to http://localhost:9090/api/v1/scrub:
dial tcp 127.0.0.1:9090: connect: connection refused
```

**Solution**: Ensure the contextd HTTP server is running:
```bash
contextd --http
```

### Server returned status 400

```
Error: Server returned status 400: invalid request body
```

**Solution**: Ensure you're providing content to scrub:
```bash
# Wrong - empty input
echo "" | ctxd scrub -

# Right
echo "content" | ctxd scrub -
```

### No content to scrub

```
Error: no content to scrub
```

**Solution**: Provide input via file or stdin:
```bash
# Provide file
ctxd scrub .env

# Provide stdin
cat .env | ctxd scrub -
```

## License

Same as contextd project.
