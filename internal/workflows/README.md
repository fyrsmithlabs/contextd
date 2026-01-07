# Temporal Workflows

This package contains Temporal workflow definitions for contextd automation tasks.

## Plugin Update Validation Workflow

Automated workflow that validates Claude plugin updates in pull requests.

### Overview

The `PluginUpdateValidationWorkflow` monitors pull requests for code changes that require corresponding plugin updates. When code is modified that affects MCP tools, services, or configuration, the workflow:

1. Detects the changes
2. Checks if the `.claude-plugin/` directory was updated
3. Validates plugin schemas if modified
4. Posts reminder or success comments to the PR

### Architecture

```
GitHub PR Event
      ↓
GitHub Webhook Server (cmd/github-webhook)
      ↓
Temporal Workflow Engine
      ↓
Plugin Validation Workflow
      ├── FetchPRFilesActivity (GitHub API)
      ├── CategorizeFilesActivity (regex patterns)
      ├── ValidatePluginSchemasActivity (JSON validation)
      └── PostCommentActivity (GitHub API)
```

### Components

| Component | Purpose | Location |
|-----------|---------|----------|
| Workflow | Orchestrates validation steps | `plugin_validation.go` |
| Activities | GitHub API interactions, validation | `plugin_validation_activities.go` |
| Worker | Executes workflows and activities | `cmd/plugin-validator/main.go` |
| Webhook Server | Receives GitHub events | `cmd/github-webhook/main.go` |

### Running the Stack

#### Prerequisites

- Docker and Docker Compose
- GitHub personal access token (for API access)
- GitHub webhook secret (for webhook validation)

#### Local Development

```bash
# Set environment variables
export GITHUB_TOKEN=ghp_your_token_here
export GITHUB_WEBHOOK_SECRET=your_secret_here

# Start the full stack
docker-compose -f deploy/docker-compose.temporal.yml up

# Access Temporal Web UI
open http://localhost:8080
```

The stack includes:
- PostgreSQL (Temporal state store) on port 5432
- Temporal Server (gRPC) on port 7233
- Temporal Web UI on port 8080
- GitHub Webhook Server on port 3000
- Plugin Validator Worker (background)

#### GitHub Webhook Configuration

1. Go to your repository Settings > Webhooks
2. Add webhook:
   - Payload URL: `https://your-domain.com/webhook`
   - Content type: `application/json`
   - Secret: Your webhook secret
   - Events: Pull requests (opened, synchronize, reopened)

### File Categorization

The workflow categorizes changed files into two categories:

#### Code Files (require plugin update)

- `internal/mcp/tools.go` - MCP tool definitions
- `internal/mcp/handlers/*.go` - MCP tool handlers
- `internal/*/service.go` - Service implementations
- `internal/config/{types,config}.go` - Configuration types

#### Plugin Files

- `.claude-plugin/**/*` - All plugin files

### Validation Behavior

| Scenario | Action |
|----------|--------|
| Code changed, plugin NOT updated | Post reminder comment |
| Code changed, plugin updated, schemas valid | Post success comment |
| Code changed, plugin updated, schemas invalid | Post error details |
| Only docs/tests changed | No action |

### Testing

```bash
# Run workflow tests
go test ./internal/workflows/... -v

# Test specific workflow
go test ./internal/workflows/... -run TestPluginUpdateValidationWorkflow -v

# Test activity categorization
go test ./internal/workflows/... -run TestCategorizeFilesActivity -v
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Yes | GitHub personal access token for API access |
| `GITHUB_WEBHOOK_SECRET` | Yes | Secret for validating webhook signatures |
| `TEMPORAL_HOST` | No | Temporal server address (default: localhost:7233) |
| `PORT` | No | Webhook server port (default: 3000) |

### Monitoring

- **Temporal Web UI**: http://localhost:8080 - View workflow executions, activity status, and errors
- **Workflow Logs**: Check worker container logs for detailed execution traces
- **GitHub Comments**: Workflow posts comments to PRs with validation results

### Extending

To add new file patterns to detect:

1. Edit `CategorizeFilesActivity` in `plugin_validation_activities.go`
2. Add regex pattern to `codePatterns` slice
3. Update tests in `plugin_validation_test.go`

To customize comments:

1. Edit `buildReminderComment()` or `buildSuccessComment()` functions
2. Update comment templates with your messaging

### Related

- Issue #56: Claude plugin update validation automation
- CLAUDE.md: Priority #3 - Update Claude Plugin on Changes
- PR Template: `.github/pull_request_template.md`

## Version Validation Workflow

Automated workflow that ensures `VERSION` file consistency across the codebase.

### Overview

The `VersionValidationWorkflow` validates that the `VERSION` file matches the version in `.claude-plugin/plugin.json` for all pull requests. This prevents version mismatches that could cause confusion about which version is deployed.

When versions don't match, the workflow:
1. Detects the mismatch
2. Posts a detailed comment explaining the issue
3. Provides exact commands to fix the problem
4. Auto-updates the comment if versions are later synced

### Architecture

```
GitHub PR Event (VERSION or plugin.json changed)
      ↓
GitHub Webhook Server
      ↓
Temporal Workflow Engine
      ↓
Version Validation Workflow
      ├── FetchFileContentActivity (VERSION)
      ├── FetchFileContentActivity (plugin.json)
      ├── Compare versions
      └── PostVersionMismatchCommentActivity (if needed)
```

### Components

| Component | Purpose | Location |
|-----------|---------|----------|
| Workflow | Orchestrates version validation | `version_validation.go` |
| Activities | Fetches files, posts comments | `version_validation_activities.go` |
| Tests | Comprehensive test coverage | `version_validation_test.go` |

### Validation Logic

**Version matching:**
- Fetches `VERSION` file content from PR HEAD
- Fetches `.claude-plugin/plugin.json` from PR HEAD
- Parses JSON to extract version field
- Compares versions (after trimming whitespace)
- Posts comment only if versions don't match

**Supported version formats:**
- Standard semantic versions: `1.2.3`
- Pre-release versions: `1.0.0-rc.1`
- Build metadata: `1.0.0+build.123`
- Complex versions: `2.0.0-beta.1+exp.sha.5114f85`

### Comment Behavior

| Scenario | Action |
|----------|--------|
| Versions match | No comment posted |
| Versions don't match (first time) | Post new mismatch comment |
| Versions don't match (updated PR) | Update existing mismatch comment |
| Versions fixed after comment | Comment remains (manual removal or future enhancement) |

### Testing

```bash
# Run workflow tests
go test ./internal/workflows/... -run TestVersionValidationWorkflow -v

# Run all version validation tests
go test ./internal/workflows/... -run Version -v

# Test the sync script itself
./scripts/sync-version_test.sh
```

### Configuration

The workflow is triggered automatically by the webhook server when:
- Pull request is opened
- Pull request is synchronized (new commits pushed)
- Pull request is reopened

No manual configuration required beyond standard webhook setup.

### Related

- `scripts/sync-version.sh` - Script to sync versions across files
- `scripts/sync-version_test.sh` - Test suite for sync script
- `docs/VERSIONING.md` - Complete version management documentation
- `VERSION` - Single source of truth for version
