# Backup System Specification

## Overview


**Version**: 1.0.0
**Status**: Implemented
**Last Updated**: 2025-11-04

## Purpose

The backup system addresses several critical needs:

1. **Data Protection**: Prevent data loss from system failures, corruption, or user errors
2. **Disaster Recovery**: Enable complete restoration of contextd state
3. **Migration Support**: Facilitate moving contextd between systems
4. **Experimentation Safety**: Allow safe testing with rollback capability
5. **Compliance**: Meet data retention and backup requirements

## Features and Capabilities

### Core Features

1. **Automated Backups**
   - Scheduled periodic backups (configurable interval)
   - Manual on-demand backups
   - Automatic cleanup of old backups based on retention policy

2. **Backup Types**
   - **Collection-Specific**: Future support for individual collections

3. **Compression**
   - gzip compression (tar.gz format)
   - Configurable compression levels (1-9, default: 6)
   - Achieves 60-80% size reduction typical

4. **Metadata Tracking**
   - Backup ID generation with timestamp
   - Collection inventory
   - File count and size tracking
   - Backup type classification

5. **Validation**
   - Integrity verification (gzip header check)
   - Size validation against metadata
   - Archive format validation

6. **Search and Discovery**
   - List all backups (sorted by timestamp)
   - Search by pattern (ID or collection name)
   - Get detailed backup information

## Architecture and Design

### System Components

```
┌─────────────────────────────────────────────────────┐
│                 Backup System                        │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │   Service    │  │  Scheduler   │  │ Compress  │ │
│  │   (backup.   │──│  (periodic)  │  │ (tar.gz)  │ │
│  │   Service)   │  │              │  │           │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
│         │                  │                │       │
│         │                  │                │       │
│  ┌──────▼──────────────────▼────────────────▼─────┐ │
│  │         Filesystem (Backup Directory)          │ │
│  │  ~/.local/share/contextd/backups/              │ │
│  │    ├── backup_20250104_120000.tar.gz           │ │
│  │    ├── backup_20250104_120000.json             │ │
│  │    └── ...                                     │ │
│  └────────────────────────────────────────────────┘ │
│                                                      │
└─────────────────────────────────────────────────────┘
         │                                    │
         │ Read/Write                         │ Restore
         ▼                                    ▼
┌──────────────────┐              ┌──────────────────┐
│  Directory       │              │  Directory       │
│  (Source)        │              │  (Destination)   │
└──────────────────┘              └──────────────────┘
```

### Package Structure

```
pkg/backup/
├── backup.go       # Service implementation, core logic
├── config.go       # Configuration loading and defaults
├── compress.go     # Compression/decompression utilities
├── backup_test.go  # Service tests
├── config_test.go  # Configuration tests
└── compress_test.go # Compression tests

internal/handlers/
└── backup.go       # HTTP API handlers

cmd/ctxd/
└── backup.go       # CLI backup commands

examples/backup/
└── main.go         # Usage examples
```

### Data Flow

#### Backup Creation Flow

```
1. CreateBackup() called
   ├── Generate backup ID (backup_YYYYMMDD_HHMMSS)
   ├── Create backup file path
   │
2. CompressDirectory()
   ├── Create tar.gz archive
   │   ├── Open gzip writer (compression level)
   │   ├── Open tar writer
   │   ├── For each file:
   │   │   ├── Create tar header
   │   │   ├── Write header
   │   │   └── Copy file contents
   │   └── Close writers
   └── Return file count, total size
   │
3. Create Metadata
   ├── Backup ID
   ├── Timestamp
   ├── Collections list
   ├── File count
   ├── Archive size
   └── Backup type
   │
4. Save Metadata JSON
   ├── Marshal to JSON
   └── Write to backup_ID.json (0600 permissions)
   │
5. Record Metrics
   ├── Increment backup counter
   ├── Record backup size
   └── Record backup duration
```

#### Restore Flow

```
1. RestoreBackup(backupID) called
   ├── Verify backup exists
   └── Load metadata
   │
2. Create temporary directory
   ├── Create restore_TIMESTAMP directory
   └── Set 0700 permissions
   │
3. DecompressArchive()
   ├── Open backup file
   ├── Create gzip reader
   ├── Create tar reader
   ├── For each file:
   │   ├── Validate path (no traversal)
   │   ├── Check size limits (100MB per file, 500MB total)
   │   ├── Create parent directories (0750)
   │   └── Extract file (preserve mode)
   └── Return file count
   │
   │
5. Cleanup
   └── Remove temporary directory (if error occurred)
```

### Design Patterns

1. **Service Pattern**: Centralized service with dependency injection
2. **Configuration as Code**: Environment-driven configuration with defaults
3. **Fail-Safe Operations**: Temporary directories, atomic operations
4. **Comprehensive Instrumentation**: OpenTelemetry traces and metrics throughout

## Backup Scope

### Data Backed Up

- All collection data files
- Vector index files
- Metadata files
- WAL (Write-Ahead Log) files

**Collections Included**:
- `checkpoints` - Session checkpoints
- `remediations` - Error solutions and patterns
- `skills` - Reusable workflow templates
- `documents` - Indexed repository files
- `troubleshooting` - Troubleshooting patterns

**Total Data Size**: Typically 10MB - 1GB depending on usage
**Compressed Size**: 2MB - 200MB (60-80% compression)

### Data Excluded

- API tokens (`~/.config/contextd/token`)
- OpenAI API keys (`~/.config/contextd/openai_api_key`)
- Configuration files (`~/.claude/config.json`)
- Service logs
- Temporary files

**Note**: For complete system backup including credentials, use the installer backup system (`pkg/installer/backup.go`) which provides separate "config" and "full" backup types.

## Backup Schedule

### Automated Scheduling

**Default Schedule**: Every 24 hours (configurable)

**Configuration**:
```bash
# Environment variable
export CONTEXTD_BACKUP_SCHEDULE="24h"  # Valid time.Duration string

# Programmatic
config := &backup.Config{
    ScheduleInterval: 24 * time.Hour,
}
```

**Scheduling Options**:
- `0` or `""` - Scheduling disabled (manual only)
- `1h` - Hourly backups
- `6h` - Every 6 hours
- `24h` - Daily (default)
- `168h` - Weekly

**Scheduler Behavior**:
- Runs in background goroutine
- First backup occurs immediately on service start
- Subsequent backups at configured interval
- Automatic cleanup after each backup
- Graceful shutdown with context cancellation

### Manual Backups

Users can trigger backups manually via:

1. **CLI**: `ctxd backup create` (future implementation)
2. **API**: `POST /api/v1/backups`
3. **MCP Tool**: `backup_create` (future implementation)
4. **Programmatic**: `service.CreateBackup(ctx)`

## Retention Policies

### Retention Configuration

**Default Retention**: Keep 10 most recent backups

**Configuration**:
```bash
# Environment variable
export CONTEXTD_BACKUP_RETENTION=10

# Programmatic
config := &backup.Config{
    RetentionCount: 10,
}
```

**Retention Options**:
- `0` - Keep all backups (no cleanup)
- `1-N` - Keep N most recent backups
- Recommended: `10` (2 weeks of daily backups)

### Cleanup Behavior

**Automatic Cleanup**:
- Runs after each scheduled backup
- Deletes oldest backups beyond retention count
- Sorts by timestamp (newest first)
- Logs cleanup operations

**Cleanup Algorithm**:
```
1. List all backups (sorted newest → oldest)
2. If total ≤ retention_count: exit
3. Calculate excess: total - retention_count
4. Delete excess oldest backups
5. Record metrics (backups_retained)
```

**Cleanup Failures**:
- Non-fatal: Logs error and continues
- Does not prevent new backups
- Retries on next scheduled backup

### Manual Cleanup

Users can manually delete backups:
- CLI: `ctxd backup delete <backup-id>` (future)
- API: `DELETE /api/v1/backups/:name`
- Programmatic: `service.DeleteBackup(ctx, backupID)`

## Backup Storage Location

### Default Directories

**Linux**:
- Backup Directory: `~/.local/share/contextd/backups/`

**macOS**:
- Backup Directory: `~/Library/Application Support/contextd/backups/`

### Directory Structure

```
~/.local/share/contextd/backups/
├── backup_20250104_120000.tar.gz       # Compressed backup archive
├── backup_20250104_120000.json         # Backup metadata
├── backup_20250104_180000.tar.gz
├── backup_20250104_180000.json
└── ...

Metadata JSON Format:
{
  "id": "backup_20250104_120000",
  "timestamp": "2025-01-04T12:00:00Z",
  "collections": ["checkpoints", "remediations", "skills"],
  "file_count": 127,
  "size_bytes": 15728640,
  "type": "lite"
}
```

### File Permissions

**Security**:
- Backup directory: `0700` (owner only)
- Archive files: `0600` (owner read/write only)
- Metadata files: `0600` (owner read/write only)
- No network exposure (local filesystem only)

### Storage Requirements

**Disk Space Planning**:
```
Required Space = (Average Backup Size × Retention Count) × 1.5

Example (100MB average, 10 retention):
Required = (100MB × 10) × 1.5 = 1.5GB
```

**Size Estimates**:
- Light usage (1-2 weeks): 10-50MB per backup
- Moderate usage (1-2 months): 50-200MB per backup
- Heavy usage (6+ months): 200MB-1GB per backup

### Custom Backup Location

Override via environment variable:
```bash
export CONTEXTD_BACKUP_DIR="/custom/backup/path"
```

## Recovery Procedures

### Standard Recovery

**Prerequisites**:
1. Stop contextd service
2. Verify backup integrity
3. Create safety backup of current state

**Steps**:

```bash
# 1. Stop contextd
systemctl --user stop contextd  # Linux
launchctl stop com.axyzlabs.contextd  # macOS

# 2. List available backups
ctxd backup list

# 3. Validate backup (optional but recommended)
ctxd backup validate backup_20250104_120000

# 4. Restore from backup
ctxd backup restore backup_20250104_120000

# 5. Start contextd
systemctl --user start contextd  # Linux
launchctl start com.axyzlabs.contextd  # macOS

# 6. Verify service
ctxd health
```

### Emergency Recovery

**Scenario**: Complete data loss, service won't start

```bash
# 1. Stop any running instances
killall contextd

# 2. Remove corrupted data

# 3. Restore from latest backup
cd ~/.local/share/contextd/backups/

# 4. Fix permissions

# 5. Restart service
systemctl --user restart contextd
```

### Partial Recovery

**Scenario**: Recover specific collections only

```bash
# 1. Extract to temporary location
mkdir /tmp/backup-extract
tar -xzf backup_20250104_120000.tar.gz -C /tmp/backup-extract

# 2. Copy specific collection

# 3. Restart contextd
systemctl --user restart contextd

# 4. Cleanup
rm -rf /tmp/backup-extract
```

### Migration Between Systems

**Scenario**: Move contextd to new machine

```bash
# Source Machine:
# 1. Create backup
ctxd backup create --type full

# 2. Copy backup to new machine
scp ~/.local/share/contextd/backups/backup_*.tar.gz user@newmachine:~/

# Destination Machine:
# 1. Install contextd
curl -sSL https://contextd.dev/install.sh | sh

# 2. Restore backup
mkdir -p ~/.local/share/contextd/backups
mv ~/backup_*.tar.gz ~/.local/share/contextd/backups/
ctxd backup restore backup_20250104_120000

# 3. Start service
ctxd start
```

### Recovery Verification

After any recovery:

```bash
# 1. Check service health
ctxd health

# 2. Verify collections exist
ctxd checkpoint list
ctxd remediation list

# 3. Test operations
ctxd checkpoint save "Test checkpoint after recovery"
ctxd checkpoint search "test"

# 4. Check logs for errors
journalctl --user -u contextd -n 100
```

### Recovery Time Objectives

**RTO (Recovery Time Objective)**:
- Small backup (< 100MB): 30 seconds
- Medium backup (100-500MB): 1-2 minutes
- Large backup (> 500MB): 2-5 minutes

**RPO (Recovery Point Objective)**:
- With daily backups: Maximum 24 hours data loss
- With hourly backups: Maximum 1 hour data loss

## API Specifications

### HTTP API Endpoints

**Base URL**: `http://localhost` (Unix socket)
**Authentication**: Bearer token required

#### Create Backup

```http
POST /api/v1/backups
Authorization: Bearer <token>
```

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": "backup_20250104_120000",
    "timestamp": "2025-01-04T12:00:00Z",
    "collections": ["checkpoints", "remediations", "skills"],
    "file_count": 127,
    "size_bytes": 15728640,
    "type": "lite"
  }
}
```

#### List Backups

```http
GET /api/v1/backups
Authorization: Bearer <token>
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "backups": [
      {
        "id": "backup_20250104_180000",
        "timestamp": "2025-01-04T18:00:00Z",
        "collections": ["checkpoints", "remediations"],
        "file_count": 134,
        "size_bytes": 16777216,
        "type": "lite"
      },
      {
        "id": "backup_20250104_120000",
        "timestamp": "2025-01-04T12:00:00Z",
        "collections": ["checkpoints", "remediations"],
        "file_count": 127,
        "size_bytes": 15728640,
        "type": "lite"
      }
    ],
    "total": 2
  }
}
```

#### Get Backup Info

```http
GET /api/v1/backups/:name
Authorization: Bearer <token>
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "id": "backup_20250104_120000",
    "timestamp": "2025-01-04T12:00:00Z",
    "collections": ["checkpoints", "remediations"],
    "file_count": 127,
    "size_bytes": 15728640,
    "type": "lite"
  }
}
```

#### Validate Backup

```http
POST /api/v1/backups/:name/validate
Authorization: Bearer <token>
```

**Response** (200 OK - Valid):
```json
{
  "success": true,
  "data": {
    "message": "Backup is valid",
    "backup_id": "backup_20250104_120000",
    "valid": true
  }
}
```

**Response** (200 OK - Invalid):
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Backup validation failed",
    "details": "size mismatch: expected 15728640, got 15000000"
  }
}
```

#### Restore Backup

```http
POST /api/v1/backups/:name/restore
Authorization: Bearer <token>
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "Backup restored successfully",
    "backup_id": "backup_20250104_120000"
  }
}
```

#### Delete Backup

```http
DELETE /api/v1/backups/:name
Authorization: Bearer <token>
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "Backup deleted successfully",
    "backup_id": "backup_20250104_120000"
  }
}
```

### Error Responses

**400 Bad Request**:
```json
{
  "success": false,
  "error": {
    "code": "MISSING_PARAMETER",
    "message": "Backup name is required"
  }
}
```

**404 Not Found**:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Backup not found",
    "details": "backup not found: backup_20250104_120000"
  }
}
```

**500 Internal Server Error**:
```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Failed to create backup",
    "details": "failed to compress directory: disk full"
  }
}
```

### CLI Commands (Future)

```bash
# Create backup
ctxd backup create
ctxd backup create --type full

# List backups
ctxd backup list

# Get backup info
ctxd backup info backup_20250104_120000

# Validate backup
ctxd backup validate backup_20250104_120000

# Restore backup
ctxd backup restore backup_20250104_120000

# Delete backup
ctxd backup delete backup_20250104_120000

# Cleanup old backups manually
ctxd backup cleanup --keep 5
```

## Data Models and Schemas

### Configuration Schema

```go
type Config struct {
    // BackupDir is the directory where backups are stored
    BackupDir string

    // RetentionCount is the number of backups to keep (0 = keep all)
    RetentionCount int

    // ScheduleInterval is the interval between automatic backups (0 = disabled)
    ScheduleInterval time.Duration


    // CompressionLevel is the gzip compression level (1-9, 0 = default)
    CompressionLevel int

    // Collections is the list of collection names to back up
    Collections []string
}
```

**Default Values**:
```go
BackupDir:        "~/.local/share/contextd/backups"
RetentionCount:   10
ScheduleInterval: 24 * time.Hour
CompressionLevel: 6
Collections:      ["checkpoints", "remediations"]
```

### Backup Metadata Schema

```go
type BackupMetadata struct {
    // ID is the unique backup identifier (backup_YYYYMMDD_HHMMSS)
    ID string `json:"id"`

    // Timestamp is when the backup was created
    Timestamp time.Time `json:"timestamp"`

    Collections []string `json:"collections"`

    // FileCount is the number of files in the backup
    FileCount int64 `json:"file_count"`

    // SizeBytes is the size of the backup archive in bytes
    SizeBytes int64 `json:"size_bytes"`

    // Type is the backup type ("lite" or "cluster")
    Type string `json:"type"`
}
```

### Service Schema

```go
type Service struct {
    config *Config

    // OpenTelemetry instrumentation
    tracer trace.Tracer
    meter  metric.Meter

    // Metrics
    backupDuration  metric.Float64Histogram
    backupSize      metric.Int64Histogram
    backupCount     metric.Int64Counter
    restoreDuration metric.Float64Histogram
    cleanupDuration metric.Float64Histogram
    backupsRetained metric.Int64Gauge

    // Scheduler control
    stopScheduler chan struct{}
}
```

### JSON Metadata File Format

**File**: `backup_20250104_120000.json`

```json
{
  "id": "backup_20250104_120000",
  "timestamp": "2025-01-04T12:00:00Z",
  "collections": [
    "checkpoints",
    "remediations",
    "skills",
    "documents"
  ],
  "file_count": 127,
  "size_bytes": 15728640,
  "type": "lite"
}
```

**Validation Rules**:
- `id`: Must match filename without extension
- `timestamp`: ISO 8601 format (RFC3339)
- `collections`: Array of strings, may be empty
- `file_count`: Non-negative integer
- `size_bytes`: Positive integer matching actual file size
- `type`: "lite" or "cluster"

## Performance Characteristics

### Backup Performance

**Compression Speed** (Compression Level 6):
- Small dataset (10MB): 100-200ms
- Medium dataset (100MB): 1-2 seconds
- Large dataset (1GB): 10-20 seconds

**Compression Ratios**:
- Vector data: 70-80% reduction
- Metadata: 85-90% reduction
- Mixed workload: 60-70% reduction

**CPU Usage**:
- Compression: 1 CPU core at 80-100%
- Duration: Proportional to data size
- Impact: Low (background operation)

**Memory Usage**:
- Baseline: 10-20MB
- Peak (1GB backup): 50-100MB
- Streaming: No full data loading

**Disk I/O**:
- Write: Sequential to backup file
- Minimal random I/O

### Restore Performance

**Decompression Speed**:
- Small backup (10MB): 50-100ms
- Medium backup (100MB): 500ms-1s
- Large backup (1GB): 5-10 seconds

**Validation Overhead**:
- Gzip header check: < 1ms
- Size verification: < 1ms
- Full validation: Same as decompression

**Service Downtime**:
- Duration: Decompression time + 2-5 seconds restart

### Scalability Limits

**File Count**:
- Maximum: 1,000,000 files per backup
- Typical: 100-10,000 files
- Impact: Linear increase in compression time

**Backup Size**:
- Maximum: 10GB (practical limit)
- Recommended: < 1GB per backup
- Large backups: Consider collection-specific backups

**Concurrent Operations**:
- Backups: Single-threaded (sequential)
- Restores: Single-threaded (sequential)
- No parallel backup/restore support

### Optimization Recommendations

1. **Compression Level**:
   - Level 6 (default): Best balance
   - Level 1-3: Faster, larger files
   - Level 7-9: Slower, smaller files

2. **Scheduling**:
   - Run during low-activity periods
   - Avoid overlapping with heavy queries
   - Consider weekly schedules for large datasets

3. **Retention**:
   - Balance storage vs. history needs
   - Keep 7-14 days typical
   - Archive old backups to cold storage

4. **Storage**:
   - Use fast local disk for backups
   - Network storage adds latency
   - SSD preferred over HDD

## Error Handling

### Error Categories

#### Backup Creation Errors

**Disk Space Errors**:
```go
Error: "failed to create archive file: no space left on device"
Recovery: Free disk space or change backup location
Prevention: Monitor disk usage, set alerts at 80%
```

**Permission Errors**:
```go
Error: "failed to create backup directory: permission denied"
Recovery: Fix directory permissions (chmod 0700)
Prevention: Run installer as correct user
```

**Compression Errors**:
```go
Error: "failed to compress directory: source directory not found"
Prevention: Validate configuration on startup
```

#### Restore Errors

**Backup Not Found**:
```go
Error: "backup not found: backup_20250104_120000"
Recovery: List available backups, verify backup ID
Prevention: Validate backup exists before restore
```

**Corruption Errors**:
```go
Error: "failed to decompress backup: unexpected EOF"
Recovery: Try older backup, re-download if transferred
Prevention: Validate backups after creation
```

**Path Traversal Detection**:
```go
Error: "archive contains path traversal: ../../etc/passwd"
Recovery: Backup is malicious or corrupted, discard
Prevention: Only restore from trusted sources
```

**Decompression Bomb**:
```go
Error: "file exceeds size limit: large_file (150MB > 100MB)"
Recovery: Backup may be malicious, verify source
Prevention: Enforce size limits during decompression
```

#### Scheduler Errors

**Schedule Failure**:
```go
Error: "scheduled backup failed: [error details]"
Recovery: Automatic retry on next interval
Prevention: Monitor backup logs, set up alerts
```

**Cleanup Failure**:
```go
Error: "backup cleanup failed: failed to delete old backup"
Recovery: Manual cleanup via delete API
Prevention: Check disk permissions, monitor logs
```

### Error Handling Strategy

1. **Fail-Safe Operations**:
   - Use temporary directories for restore
   - Atomic operations where possible
   - Rollback on error

2. **Error Context**:
   - Wrap errors with context
   - Include operation details
   - Log full stack traces

3. **User Communication**:
   - Clear error messages
   - Actionable recovery steps
   - Avoid technical jargon in API responses

4. **Monitoring**:
   - Log all errors
   - Emit error metrics
   - Alert on critical failures

### Error Codes

| Code | Meaning | HTTP Status |
|------|---------|-------------|
| `MISSING_PARAMETER` | Required parameter missing | 400 |
| `INVALID_PARAMETER` | Parameter format invalid | 400 |
| `NOT_FOUND` | Backup not found | 404 |
| `VALIDATION_FAILED` | Backup validation failed | 200* |
| `INTERNAL_ERROR` | Internal server error | 500 |
| `DISK_FULL` | Insufficient disk space | 500 |
| `PERMISSION_DENIED` | Permission error | 500 |

*Note: Validation endpoint returns 200 with success=false for invalid backups

## Security Considerations

### File System Security

**Directory Permissions**:
```bash
~/.local/share/contextd/backups/  # 0700 (rwx------)
backup_20250104_120000.tar.gz     # 0600 (rw-------)
backup_20250104_120000.json       # 0600 (rw-------)
```

**Rationale**:
- Only owner can read/write/execute
- No group or other access
- Prevents unauthorized backup access
- Protects backup metadata

### Path Traversal Protection

**Multiple Defense Layers**:

1. **Path Cleaning**:
   ```go
   cleanPath := filepath.Clean(path)
   ```

2. **Absolute Path Detection**:
   ```go
   if filepath.IsAbs(cleanPath) {
       return error // Reject absolute paths
   }
   ```

3. **Parent Directory Detection**:
   ```go
   if strings.Contains(cleanPath, "..") {
       return error // Reject path traversal
   }
   ```

4. **Base Directory Verification**:
   ```go
   if !isPathWithinBase(resolvedPath, baseDir) {
       return error // Path escapes base
   }
   ```

**Attack Prevention**:
- Blocks `../../etc/passwd`
- Blocks `/etc/passwd`
- Blocks `./././../../../etc/passwd`
- Blocks symlink attacks

### Symlink Attack Prevention

**Detection Methods**:

1. **Lstat Instead of Stat**:
   ```go
   info, err := os.Lstat(path)  // Detects symlink
   ```

2. **Mode Bit Check**:
   ```go
   if info.Mode()&os.ModeSymlink != 0 {
       return error // Reject symlink
   }
   ```

3. **Path Component Scanning**:
   ```go
   for each component in path {
       if isSymlink(component) {
           return error
       }
   }
   ```

**Attack Prevention**:
- Blocks symlink to `/etc/passwd`
- Blocks symlink in directory tree
- Prevents TOCTOU (Time-of-Check-Time-of-Use) attacks

### Decompression Bomb Protection

**Size Limits**:
```go
const maxFileSize = 100 * 1024 * 1024    // 100MB per file
const maxTotalSize = 500 * 1024 * 1024   // 500MB total extraction
```

**Enforcement**:
```go
if header.Size > maxFileSize {
    return error // File too large
}

totalExtracted += header.Size
if totalExtracted > maxTotalSize {
    return error // Total size exceeded
}
```

**Attack Prevention**:
- Prevents infinite expansion attacks
- Limits memory usage
- Protects against malicious archives

### API Security

**Authentication**:
- Bearer token required for all endpoints
- Token stored in `~/.config/contextd/token` (0600)
- Constant-time token comparison

**Authorization**:
- Single-user system (local only)
- No multi-tenancy considerations
- All authenticated users have full access

**Transport Security**:
- Unix domain socket (no network exposure)
- No TLS required (local IPC)
- Socket permissions: 0600

### Backup Encryption

**Current State**: Not implemented

**Future Considerations**:
- GPG encryption for backups
- Password-protected archives
- Key management for automation

**Implementation Example**:
```go
// Encrypt backup after creation
func (s *Service) EncryptBackup(backupPath, key string) error {
    // Use age or GPG for encryption
}
```

### Audit Logging

**Security Events Logged**:
- Backup creation (who, when, size)
- Backup restoration (who, which backup, when)
- Backup deletion (who, which backup, when)
- Validation failures (which backup, failure reason)
- Path traversal attempts (blocked path, source)
- Symlink attack attempts (symlink path, source)

**Log Format**:
```json
{
  "timestamp": "2025-01-04T12:00:00Z",
  "event": "backup_restored",
  "backup_id": "backup_20250104_120000",
  "user": "username",
  "duration_ms": 1234,
  "status": "success"
}
```

## Testing Requirements

### Unit Tests

**Coverage Targets**:
- Overall: ≥80%
- Core functions: 100%
- Error paths: ≥90%

**Test Categories**:

1. **Configuration Tests** (`config_test.go`):
   - Default configuration
   - Environment variable overrides
   - Directory creation
   - Invalid paths

2. **Compression Tests** (`compress_test.go`):
   - Directory compression
   - Archive decompression
   - File count accuracy
   - Size calculation
   - Path traversal rejection
   - Symlink rejection
   - Decompression bomb protection

3. **Service Tests** (`backup_test.go`):
   - Backup creation
   - Backup listing and sorting
   - Backup restoration
   - Backup deletion
   - Backup validation
   - Search functionality
   - Metadata handling
   - Retention cleanup
   - Scheduler operation

### Integration Tests

**Test Scenarios**:

1. **Full Backup/Restore Cycle**:
   ```go
   func TestFullBackupRestoreCycle(t *testing.T) {
       // 2. Create backup
       // 3. Modify/delete original data
       // 4. Restore from backup
       // 5. Verify data matches original
   }
   ```

2. **Scheduled Backup**:
   ```go
   func TestScheduledBackup(t *testing.T) {
       // 1. Start scheduler with 1-second interval
       // 2. Wait for multiple backups
       // 3. Verify backups created
       // 4. Verify retention enforced
       // 5. Stop scheduler
   }
   ```

3. **Concurrent Operations**:
   ```go
   func TestConcurrentBackups(t *testing.T) {
       // 1. Trigger multiple backups simultaneously
       // 2. Verify sequential execution
       // 3. Verify no data corruption
   }
   ```

### Security Tests

**Attack Scenarios**:

1. **Path Traversal**:
   ```go
   func TestPathTraversalRejection(t *testing.T) {
       maliciousPaths := []string{
           "../../etc/passwd",
           "/etc/passwd",
           "./././../../../etc/passwd",
       }
       for _, path := range maliciousPaths {
           err := restore(path)
           assert.Error(t, err)
           assert.Contains(t, err.Error(), "invalid file path")
       }
   }
   ```

2. **Symlink Attack**:
   ```go
   func TestSymlinkRejection(t *testing.T) {
       // 1. Create symlink in test directory
       // 2. Attempt to backup directory
       // 3. Verify symlink rejected
   }
   ```

3. **Decompression Bomb**:
   ```go
   func TestDecompressionBombProtection(t *testing.T) {
       // 1. Create archive with huge file
       // 2. Attempt to restore
       // 3. Verify size limit enforced
   }
   ```

### Performance Tests

**Benchmarks**:

```go
func BenchmarkBackupCreation(b *testing.B) {
    // Measure backup creation time for various sizes
}

func BenchmarkCompression(b *testing.B) {
    // Measure compression performance by level
}

func BenchmarkRestore(b *testing.B) {
    // Measure restore performance
}
```

**Load Tests**:
- 1000 small backups
- 100 medium backups
- 10 large backups
- Retention cleanup with 1000 backups

### Test Fixtures

**Test Data**:
```
testdata/
├── corrupt.tar.gz  # Invalid archive for error testing
└── malicious.tar.gz # Path traversal archive
```

### Test Execution

```bash
# Run all tests
go test ./pkg/backup/...

# Run with coverage
go test -coverprofile=coverage.out ./pkg/backup/...
go tool cover -html=coverage.out

# Run integration tests only
go test -tags=integration ./pkg/backup/...

# Run security tests only
go test -run "TestSecurity" ./pkg/backup/...

# Run benchmarks
go test -bench=. ./pkg/backup/...
```

## Usage Examples

### Example 1: Basic Backup and Restore

```go
package main

import (
    "context"
    "log"

    "github.com/axyzlabs/contextd/pkg/backup"
)

func main() {
    ctx := context.Background()

    // Create backup service with defaults
    service, err := backup.NewService(nil)
    if err != nil {
        log.Fatal(err)
    }

    // Create backup
    metadata, err := service.CreateBackup(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Backup created: %s (%d bytes)",
        metadata.ID, metadata.SizeBytes)

    // List backups
    backups, err := service.ListBackups(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d backups", len(backups))

    // Restore latest backup
    if err := service.RestoreBackup(ctx, metadata.ID); err != nil {
        log.Fatal(err)
    }

    log.Println("Backup restored successfully")
}
```

### Example 2: Custom Configuration

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/axyzlabs/contextd/pkg/backup"
)

func main() {
    ctx := context.Background()

    // Custom configuration
    config := &backup.Config{
        BackupDir:        "/mnt/backup/contextd",
        RetentionCount:   30,  // Keep 30 backups
        ScheduleInterval: 6 * time.Hour,  // Every 6 hours
        CompressionLevel: 9,  // Maximum compression
        Collections:      []string{"checkpoints", "remediations", "skills"},
    }

    service, err := backup.NewService(config)
    if err != nil {
        log.Fatal(err)
    }

    // Start scheduler in background
    go service.StartScheduler(ctx)
    defer service.StopScheduler()

    // Manual backup
    metadata, err := service.CreateBackup(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Backup created: %s", metadata.ID)

    // Keep running for scheduler
    select {
    case <-ctx.Done():
        log.Println("Shutting down...")
    }
}
```

### Example 3: Backup Validation

```go
package main

import (
    "context"
    "log"

    "github.com/axyzlabs/contextd/pkg/backup"
)

func main() {
    ctx := context.Background()

    service, err := backup.NewService(nil)
    if err != nil {
        log.Fatal(err)
    }

    // List all backups
    backups, err := service.ListBackups(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Validate each backup
    for _, b := range backups {
        err := service.ValidateBackup(ctx, b.ID)
        if err != nil {
            log.Printf("Backup %s is INVALID: %v", b.ID, err)
        } else {
            log.Printf("Backup %s is valid", b.ID)
        }
    }
}
```

### Example 4: Search and Cleanup

```go
package main

import (
    "context"
    "log"

    "github.com/axyzlabs/contextd/pkg/backup"
)

func main() {
    ctx := context.Background()

    service, err := backup.NewService(nil)
    if err != nil {
        log.Fatal(err)
    }

    // Search for backups from January
    results, err := service.SearchBackups(ctx, "202501")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d backups from January", len(results))

    // Manual cleanup
    if err := service.CleanupOldBackups(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Cleanup completed")
}
```

### Example 5: HTTP API Usage

```bash
# Get authentication token
TOKEN=$(cat ~/.config/contextd/token)

# Create backup
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  -X POST \
  http://localhost/api/v1/backups

# List backups
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost/api/v1/backups

# Validate backup
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  -X POST \
  http://localhost/api/v1/backups/backup_20250104_120000/validate

# Restore backup
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  -X POST \
  http://localhost/api/v1/backups/backup_20250104_120000/restore

# Delete backup
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  -X DELETE \
  http://localhost/api/v1/backups/backup_20250104_120000
```

## Monitoring and Observability

### OpenTelemetry Metrics

**Metrics Exported**:

1. **backup.duration** (histogram, ms)
   - Labels: `operation` (create, restore, cleanup)
   - Tracks backup operation duration

2. **backup.size** (histogram, bytes)
   - Labels: `backup_id`
   - Tracks backup archive size

3. **backup.count** (counter)
   - Labels: `type` (lite, cluster)
   - Total backups created

4. **backup.restore.duration** (histogram, ms)
   - Labels: `backup_id`
   - Tracks restore operation duration

5. **backup.cleanup.duration** (histogram, ms)
   - Tracks cleanup operation duration

6. **backup.retained** (gauge)
   - Current number of backups retained

**Grafana Dashboard Queries**:

```promql
# Average backup duration
rate(backup_duration_sum[5m]) / rate(backup_duration_count[5m])

# Backup size over time
backup_size{backup_id=~".*"}

# Backups created per hour
rate(backup_count[1h])

# Current backups retained
backup_retained
```

### OpenTelemetry Traces

**Spans Created**:

- `backup.create` - Full backup creation
  - `backup.compress_directory` - Compression operation

- `backup.list` - List backups operation

- `backup.restore` - Full restore operation
  - `backup.decompress_archive` - Decompression operation

- `backup.delete` - Delete backup operation

- `backup.cleanup` - Cleanup operation

- `backup.validate` - Validation operation

- `backup.start_scheduler` - Scheduler start event

**Trace Attributes**:
- `backup_id` - Backup identifier
- `backup_type` - Backup type (lite/cluster)
- `file_count` - Number of files
- `total_size` - Total size in bytes
- `archive_size` - Compressed size
- `compression_level` - gzip level used

### Logging

**Log Events**:

```json
{
  "level": "info",
  "timestamp": "2025-01-04T12:00:00Z",
  "message": "backup created",
  "backup_id": "backup_20250104_120000",
  "size_bytes": 15728640,
  "file_count": 127,
  "duration_ms": 1234
}

{
  "level": "info",
  "timestamp": "2025-01-04T12:05:00Z",
  "message": "backup restored",
  "backup_id": "backup_20250104_120000",
  "duration_ms": 567
}

{
  "level": "warning",
  "timestamp": "2025-01-04T12:10:00Z",
  "message": "backup cleanup skipped",
  "reason": "retention_count is 0"
}

{
  "level": "error",
  "timestamp": "2025-01-04T12:15:00Z",
  "message": "backup creation failed",
  "error": "no space left on device",
  "backup_dir": "/var/backups/contextd"
}
```

### Health Checks

**Backup Health Indicators**:

1. **Last Backup Age**:
   - Green: < 24 hours
   - Yellow: 24-48 hours
   - Red: > 48 hours

2. **Backup Success Rate**:
   - Green: ≥ 95%
   - Yellow: 90-95%
   - Red: < 90%

3. **Disk Space Available**:
   - Green: > 20% free
   - Yellow: 10-20% free
   - Red: < 10% free

**Health Check Endpoint** (Future):
```http
GET /health/backup

{
  "status": "healthy",
  "last_backup": "2025-01-04T12:00:00Z",
  "last_backup_age_hours": 1.5,
  "total_backups": 10,
  "disk_space_free_percent": 45.2,
  "success_rate_24h": 100.0
}
```

### Alerting Rules

**Recommended Alerts**:

1. **No Recent Backup**:
   ```yaml
   alert: NoRecentBackup
   expr: time() - backup_last_success_timestamp > 172800  # 48 hours
   severity: critical
   ```

2. **Backup Failure**:
   ```yaml
   alert: BackupFailure
   expr: rate(backup_errors_total[1h]) > 0
   severity: warning
   ```

3. **Low Disk Space**:
   ```yaml
   alert: BackupDiskSpaceLow
   expr: backup_disk_free_bytes / backup_disk_total_bytes < 0.1
   severity: warning
   ```

4. **High Backup Duration**:
   ```yaml
   alert: SlowBackup
   expr: backup_duration_seconds > 300  # 5 minutes
   severity: warning
   ```

## Implementation Notes

### Current Implementation Status

**Completed**:
- ✅ Core backup service
- ✅ Compression/decompression
- ✅ Metadata management
- ✅ Retention policies
- ✅ Scheduled backups
- ✅ HTTP API handlers
- ✅ OpenTelemetry instrumentation
- ✅ Security protections (path traversal, symlinks, decompression bombs)
- ✅ Unit tests
- ✅ Integration tests
- ✅ Examples

**In Progress**:
- 🔄 CLI commands (partially implemented in installer)
- 🔄 MCP tool integration

**Future Enhancements**:
- ⏳ Incremental backups
- ⏳ Backup encryption
- ⏳ Cloud storage integration (S3, GCS)
- ⏳ Backup verification with checksums
- ⏳ Differential backups
- ⏳ Collection-specific backups
- ⏳ Backup compression strategies
- ⏳ Parallel compression
- ⏳ Backup health dashboard

### Known Limitations

1. **Single-Threaded**: Backup/restore operations are sequential
2. **No Encryption**: Backups stored unencrypted on disk
3. **Local Only**: No cloud storage support
4. **Full Backups Only**: No incremental or differential backups
5. **No Checksums**: Validation relies on gzip header and size only
6. **Service Downtime**: Restore requires stopping contextd

### Future Roadmap

**Phase 1 - CLI Integration** (v2.1.0):
- Complete CLI backup commands
- Interactive backup management
- Progress bars for long operations

**Phase 2 - Enhanced Validation** (v2.2.0):
- SHA256 checksums
- Integrity verification
- Corruption detection

**Phase 3 - Incremental Backups** (v2.3.0):
- Incremental backup support
- Delta compression
- Faster backup cycles

**Phase 4 - Cloud Integration** (v2.4.0):
- S3-compatible storage
- Google Cloud Storage
- Azure Blob Storage
- Automatic cloud sync

**Phase 5 - Advanced Features** (v2.5.0):
- Backup encryption (GPG/age)
- Parallel compression
- Collection-specific backups
- Point-in-time recovery

## Related Documentation

- [GETTING-STARTED.md](../../guides/GETTING-STARTED.md) - Setup and installation
- [MONITORING-SETUP.md](../../guides/MONITORING-SETUP.md) - Observability configuration
- [../../CLAUDE.md](../../CLAUDE.md) - Project overview
- [../standards/architecture.md](../standards/architecture.md) - Architecture patterns
- [../standards/testing-standards.md](../standards/testing-standards.md) - Testing requirements

## References

### External Documentation

- [Go tar Package](https://pkg.go.dev/archive/tar)
- [Go gzip Package](https://pkg.go.dev/compress/gzip)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

### Security Best Practices

- [OWASP Path Traversal](https://owasp.org/www-community/attacks/Path_Traversal)
- [CWE-22: Improper Limitation of Pathname](https://cwe.mitre.org/data/definitions/22.html)
- [CWE-59: Improper Link Resolution](https://cwe.mitre.org/data/definitions/59.html)
- [Zip Slip Vulnerability](https://snyk.io/research/zip-slip-vulnerability)

### Performance Resources

- [Go Performance Tuning](https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html)
- [Compression Benchmarks](https://github.com/klauspost/compress)

---

**Document Ownership**: Contextd Development Team
**Review Cycle**: Quarterly
**Next Review**: 2025-04-04
