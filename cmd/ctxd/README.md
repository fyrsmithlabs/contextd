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
