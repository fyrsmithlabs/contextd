# Stateful Checkpoint Specification

**Epic**: 2.3 - Intelligent Checkpoint Orchestration
**Feature**: Stateful checkpoint snapshots (avoid re-indexing on resume)
**Date**: 2025-01-10
**Status**: Specification
**Priority**: CRITICAL (blocks efficient context management)

---

## Problem Statement

**Current Checkpoint System** (Summary-Only):
```json
{
  "summary": "Implemented auth middleware",
  "description": "Added JWT authentication with bcrypt...",
  "tags": ["auth", "security"],
  "created_at": "2025-01-10T12:00:00Z"
}
```

**Resume Flow** (Inefficient):
1. Load checkpoint summary
2. Claude reads summary
3. Claude asks: "What files were you working on?"
4. Claude uses Glob/Grep to find files
5. Claude uses Read to read files
6. Claude re-analyzes code structure
7. Claude figures out where you left off
8. **Result**: 50K+ tokens wasted, 2-5 minutes of "catching up"

**What You Want** (Stateful Snapshot):
```json
{
  "summary": "Implemented auth middleware",
  "state": {
    "files": {
      "pkg/auth/middleware.go": "<full content>",
      "pkg/auth/middleware_test.go": "<full content>",
      "cmd/contextd/main.go": "<relevant section>"
    },
    "analysis": {
      "current_task": "Implementing JWT validation",
      "next_steps": ["Add token refresh", "Update docs"],
      "decisions_made": ["Use HS256 for now, RS256 later"],
      "blockers": []
    },
    "context": {
      "relevant_packages": ["pkg/auth", "pkg/mcp"],
      "recent_commands": ["go test ./pkg/auth/", "go build ./cmd/contextd/"],
      "working_directory": "/home/dahendel/projects/contextd"
    }
  }
}
```

**Resume Flow** (Instant):
1. Load stateful checkpoint
2. Claude sees all file contents directly
3. Claude sees analysis context (decisions, next steps)
4. Claude continues immediately
5. **Result**: <5K tokens, <10 seconds

---

## User Stories

### Story 1: Stateful Checkpoint Capture
**As a** Claude Code user
**I want** checkpoints to capture all relevant file contents and analysis
**So that** I don't waste time re-indexing on resume

**Acceptance Criteria**:
- ✅ Checkpoint captures full file contents (modified files only)
- ✅ Checkpoint captures code analysis (structure, decisions)
- ✅ Checkpoint captures next steps and task state
- ✅ Checkpoint size optimized (only relevant context)
- ✅ Can be configured: `CONTEXTD_CHECKPOINT_MODE=summary|stateful`

### Story 2: Instant Resume
**As a** Claude Code user
**I want** to resume instantly without re-reading files
**So that** I can continue working immediately

**Acceptance Criteria**:
- ✅ Resume loads full file contents from checkpoint
- ✅ Resume injects analysis context directly
- ✅ No Glob/Grep/Read needed after resume
- ✅ Resume completes in <10 seconds
- ✅ Context usage <5K tokens for resume

### Story 3: Smart File Capture
**As a** Claude Code user
**I want** checkpoints to only capture relevant files
**So that** checkpoint size stays manageable

**Acceptance Criteria**:
- ✅ Only captures files modified in current session
- ✅ Only captures files explicitly read/edited
- ✅ Excludes build artifacts, node_modules, etc.
- ✅ Configurable max file size (default: 10MB total)
- ✅ Warns if checkpoint too large

---

## Architecture

### Component 1: Stateful Checkpoint Service

**File**: `pkg/checkpoint/stateful.go` (new)

**Core Types**:
```go
type StatefulCheckpoint struct {
    // Standard fields
    ID          string    `json:"id"`
    Summary     string    `json:"summary"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`

    // Stateful fields (new)
    State *CheckpointState `json:"state"`
}

type CheckpointState struct {
    // File snapshots
    Files map[string]FileSnapshot `json:"files"`

    // Code analysis
    Analysis AnalysisContext `json:"analysis"`

    // Session context
    Context SessionContext `json:"context"`

    // Size tracking
    TotalSize int `json:"total_size_bytes"`
}

type FileSnapshot struct {
    Path         string `json:"path"`
    Content      string `json:"content"`
    Hash         string `json:"hash"`       // SHA256 for dedup
    Size         int    `json:"size_bytes"`
    LastModified time.Time `json:"last_modified"`
}

type AnalysisContext struct {
    CurrentTask    string   `json:"current_task"`
    NextSteps      []string `json:"next_steps"`
    DecisionsMade  []string `json:"decisions_made"`
    Blockers       []string `json:"blockers"`
    RelevantDocs   []string `json:"relevant_docs"`
}

type SessionContext struct {
    WorkingDirectory string   `json:"working_directory"`
    RelevantPackages []string `json:"relevant_packages"`
    RecentCommands   []string `json:"recent_commands"`
    GitBranch        string   `json:"git_branch"`
    GitCommit        string   `json:"git_commit"`
}
```

**Implementation**:
```go
// CaptureStateful creates stateful checkpoint from current session
func (s *Service) CaptureStateful(ctx context.Context, req *StatefulCheckpointRequest) (*StatefulCheckpoint, error) {
    cp := &StatefulCheckpoint{
        ID:          uuid.New().String(),
        Summary:     req.Summary,
        Description: req.Description,
        CreatedAt:   time.Now(),
        State:       &CheckpointState{},
    }

    // 1. Capture modified files (from git status)
    files, err := s.captureModifiedFiles(ctx, req.ProjectPath)
    if err != nil {
        return nil, fmt.Errorf("failed to capture files: %w", err)
    }
    cp.State.Files = files

    // 2. Extract analysis context from conversation
    cp.State.Analysis = s.extractAnalysisContext(ctx, req.ConversationHistory)

    // 3. Capture session context
    cp.State.Context = s.captureSessionContext(ctx, req.ProjectPath)

    // 4. Calculate total size
    cp.State.TotalSize = s.calculateCheckpointSize(cp.State)

    // 5. Warn if too large
    if cp.State.TotalSize > req.MaxSize {
        return nil, fmt.Errorf("checkpoint too large: %d bytes (max: %d)", cp.State.TotalSize, req.MaxSize)
    }

    return cp, nil
}

// ResumeFromStateful loads stateful checkpoint and injects context
func (s *Service) ResumeFromStateful(ctx context.Context, checkpointID string) (*ResumeContext, error) {
    // 1. Load stateful checkpoint
    cp, err := s.GetStateful(ctx, checkpointID)
    if err != nil {
        return nil, fmt.Errorf("failed to load checkpoint: %w", err)
    }

    // 2. Build resume context (ready to inject into Claude)
    resumeCtx := &ResumeContext{
        Summary:     cp.Summary,
        Files:       cp.State.Files,
        Analysis:    cp.State.Analysis,
        Context:     cp.State.Context,
        ResumeReady: true,
    }

    return resumeCtx, nil
}
```

### Component 2: File Capture Strategy

**Implementation**:
```go
// captureModifiedFiles captures files from current session
func (s *Service) captureModifiedFiles(ctx context.Context, projectPath string) (map[string]FileSnapshot, error) {
    files := make(map[string]FileSnapshot)

    // 1. Get modified files from git
    cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
    cmd.Dir = projectPath
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    // 2. Parse git status output
    modifiedFiles := parseGitStatus(string(output))

    // 3. Capture each file
    for _, path := range modifiedFiles {
        fullPath := filepath.Join(projectPath, path)

        // Skip large files
        info, err := os.Stat(fullPath)
        if err != nil || info.Size() > 1*1024*1024 { // 1MB limit per file
            continue
        }

        content, err := os.ReadFile(fullPath)
        if err != nil {
            continue
        }

        files[path] = FileSnapshot{
            Path:         path,
            Content:      string(content),
            Hash:         sha256Hash(content),
            Size:         int(info.Size()),
            LastModified: info.ModTime(),
        }
    }

    return files, nil
}
```

### Component 3: Analysis Context Extraction

**Implementation**:
```go
// extractAnalysisContext extracts structured analysis from conversation
func (s *Service) extractAnalysisContext(ctx context.Context, conversationHistory []Message) AnalysisContext {
    // Use LLM to extract structured context from conversation
    prompt := `Extract the following from the conversation:
1. Current task being worked on
2. Next steps identified
3. Key decisions made
4. Any blockers mentioned

Conversation:
%s

Return JSON:
{
  "current_task": "...",
  "next_steps": ["..."],
  "decisions_made": ["..."],
  "blockers": ["..."]
}
`

    // Call LLM to extract
    response := s.llm.Complete(ctx, fmt.Sprintf(prompt, formatConversation(conversationHistory)))

    // Parse JSON response
    var analysis AnalysisContext
    json.Unmarshal([]byte(response), &analysis)

    return analysis
}
```

### Component 4: Resume Context Injection

**File**: `pkg/mcp/resume.go` (new)

**Implementation**:
```go
// InjectResumeContext formats stateful checkpoint for Claude
func (s *Server) InjectResumeContext(ctx context.Context, resumeCtx *checkpoint.ResumeContext) string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("Resumed from checkpoint: %s\n\n", resumeCtx.Summary))

    // 1. Current Task
    sb.WriteString(fmt.Sprintf("**Current Task**: %s\n\n", resumeCtx.Analysis.CurrentTask))

    // 2. Next Steps
    sb.WriteString("**Next Steps**:\n")
    for i, step := range resumeCtx.Analysis.NextSteps {
        sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
    }
    sb.WriteString("\n")

    // 3. Decisions Made
    sb.WriteString("**Decisions Made**:\n")
    for _, decision := range resumeCtx.Analysis.DecisionsMade {
        sb.WriteString(fmt.Sprintf("- %s\n", decision))
    }
    sb.WriteString("\n")

    // 4. File Contents (injected directly - no Read needed!)
    sb.WriteString("**Working Files**:\n")
    for path, snapshot := range resumeCtx.Files {
        sb.WriteString(fmt.Sprintf("\n### %s\n", path))
        sb.WriteString("```go\n")
        sb.WriteString(snapshot.Content)
        sb.WriteString("\n```\n")
    }

    // 5. Session Context
    sb.WriteString(fmt.Sprintf("\n**Working Directory**: %s\n", resumeCtx.Context.WorkingDirectory))
    sb.WriteString(fmt.Sprintf("**Git Branch**: %s\n", resumeCtx.Context.GitBranch))

    return sb.String()
}
```

---

## User Experience

### Example: Stateful Checkpoint + Resume

**Session 1 (Before /clear)**:
```
Claude: Working on JWT authentication...
Claude: [Edits pkg/auth/middleware.go]
Claude: [Edits pkg/auth/middleware_test.go]
Claude: Tests passing! Next: Add token refresh

User: /clear

Claude: Saving stateful checkpoint...
Claude: Captured:
  - 2 files (pkg/auth/middleware.go, pkg/auth/middleware_test.go)
  - Analysis: Current task, 2 next steps, 3 decisions
  - Size: 24KB

Claude: ✅ Checkpoint saved
Claude: Clearing context...
```

**Session 2 (After resume)**:
```
[New session starts]

Claude: Resumed from checkpoint: "Implemented JWT middleware" (saved 1 hour ago)

**Current Task**: Implementing JWT validation

**Next Steps**:
1. Add token refresh logic
2. Update documentation

**Decisions Made**:
- Use HS256 for now (RS256 later)
- 1-hour token expiration
- Store secret in env var

**Working Files**:

### pkg/auth/middleware.go
```go
[FULL FILE CONTENT HERE - NO READ NEEDED]
```

### pkg/auth/middleware_test.go
```go
[FULL FILE CONTENT HERE - NO READ NEEDED]
```

**Working Directory**: /home/dahendel/projects/contextd
**Git Branch**: feature/auth-middleware

Claude: Ready to continue with token refresh. Shall I implement it now?
```

**Key Difference**: No Glob, Grep, or Read calls needed! Claude has everything instantly.

---

## Configuration

**File**: `~/.config/contextd/config.json`

```json
{
  "checkpoint": {
    "mode": "stateful",              // "summary" | "stateful"
    "capture_files": true,
    "capture_analysis": true,
    "max_checkpoint_size": 10485760, // 10MB
    "max_file_size": 1048576,        // 1MB per file
    "exclude_patterns": [
      "node_modules/**",
      "*.log",
      "*.bin"
    ]
  }
}
```

**Environment Override**:
```bash
CONTEXTD_CHECKPOINT_MODE=stateful
CONTEXTD_CHECKPOINT_MAX_SIZE=10485760
```

---

## Implementation Plan

### Phase 1: Stateful Types & Capture (Day 1)
- Create `pkg/checkpoint/stateful.go`
- Implement StatefulCheckpoint types
- Implement captureModifiedFiles()
- Tests (≥80% coverage)

### Phase 2: Analysis Extraction (Day 2)
- Implement extractAnalysisContext()
- LLM-based extraction from conversation
- Tests for extraction accuracy

### Phase 3: Resume Context Injection (Day 3)
- Create `pkg/mcp/resume.go`
- Implement InjectResumeContext()
- Format stateful context for Claude
- Tests

### Phase 4: Integration & Testing (Day 4)
- Integrate with SessionStart hook
- End-to-end testing (checkpoint → clear → resume)
- Measure context savings
- Documentation

**Total Estimate**: 4 days (can run parallel with Task #8 Phase 1-2)

---

## Success Criteria

- ✅ Stateful checkpoints capture file contents
- ✅ Stateful checkpoints capture analysis context
- ✅ Resume injects all context directly (no Read/Grep/Glob)
- ✅ Resume completes in <10 seconds
- ✅ Context usage for resume <5K tokens (vs 50K+ before)
- ✅ **90% context reduction on resume**
- ✅ Test coverage ≥80%

---

## Metrics

**Before (Summary-Only Checkpoints)**:
- Resume time: 2-5 minutes
- Context used: 50K-100K tokens
- Tool calls: 20-50 (Glob, Grep, Read)

**After (Stateful Checkpoints)**:
- Resume time: <10 seconds
- Context used: <5K tokens
- Tool calls: 0 (everything injected)

**Improvement**: 10-20x faster, 90-95% context reduction

---

## Related Documents

- **AUTO-CHECKPOINT-SYSTEM.md** - User guide
- **SPEC.md** - Main checkpoint automation spec
- **Task #8** - Intelligent Checkpoint Orchestration
- **Epic 2.3** - Parent task

---

## Next Steps

1. **Validate this spec** - Ensure it addresses your concern
2. **Create sub-task** - Add to Task #8 as "Phase 5: Stateful Snapshots"
3. **Prioritize** - High priority (critical for autonomous workflows)
4. **Implement** - After Task #8 Phase 1-4 complete

---

**Status**: Specification complete, ready for implementation

**Priority**: CRITICAL - blocks efficient context management
